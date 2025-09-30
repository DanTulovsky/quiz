package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"quizapp/internal/config"
	"quizapp/internal/models"
	"quizapp/internal/observability"
	contextutils "quizapp/internal/utils"
)

// ErrSignupsDisabled is returned when user registration is disabled by config
var ErrSignupsDisabled = errors.New("user registration is currently disabled")

// OAuth sentinel errors
var (
	ErrOAuthCodeAlreadyUsed  = errors.New("authorization code has already been used")
	ErrOAuthClientConfig     = errors.New("OAuth client configuration error")
	ErrOAuthInvalidRequest   = errors.New("invalid OAuth request")
	ErrOAuthUnauthorized     = errors.New("OAuth client is not authorized")
	ErrOAuthUnsupportedGrant = errors.New("unsupported OAuth grant type")
)

// OAuthService handles OAuth authentication flows
type OAuthService struct {
	config           *config.Config
	TokenEndpoint    string // for testing/mocking
	UserInfoEndpoint string // for testing/mocking
	logger           *observability.Logger
}

// NewOAuthServiceWithLogger creates a new OAuth service with logger
func NewOAuthServiceWithLogger(cfg *config.Config, logger *observability.Logger) *OAuthService {
	return &OAuthService{
		config:           cfg,
		TokenEndpoint:    "https://oauth2.googleapis.com/token",
		UserInfoEndpoint: "https://www.googleapis.com/oauth2/v2/userinfo",
		logger:           logger,
	}
}

// GoogleUserInfo represents the user information returned by Google OAuth
type GoogleUserInfo struct {
	ID            string `json:"id"`
	Email         string `json:"email"`
	Name          string `json:"name"`
	GivenName     string `json:"given_name"`
	FamilyName    string `json:"family_name"`
	Picture       string `json:"picture"`
	VerifiedEmail bool   `json:"verified_email"`
}

// GoogleTokenResponse represents the token response from Google OAuth
type GoogleTokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token,omitempty"`
	IDToken      string `json:"id_token,omitempty"`
}

// GetGoogleAuthURL generates the Google OAuth authorization URL
func (s *OAuthService) GetGoogleAuthURL(ctx context.Context, state string) string {
	_, span := observability.TraceOAuthFunction(ctx, "get_google_auth_url",
		attribute.String("oauth.state", state),
		attribute.String("oauth.client_id", s.config.GoogleOAuthClientID),
		attribute.String("oauth.redirect_url", s.config.GoogleOAuthRedirectURL),
	)
	defer span.End()

	// Debug logging
	if s.config.GoogleOAuthClientID == "" {
		if s.logger != nil {
			s.logger.Warn(ctx, "Google OAuth client ID is not set", map[string]interface{}{"env_var": "GOOGLE_OAUTH_CLIENT_ID"})
		}
	}
	if s.config.GoogleOAuthRedirectURL == "" {
		if s.logger != nil {
			s.logger.Warn(ctx, "Google OAuth redirect URL is not set", map[string]interface{}{"env_var": "GOOGLE_OAUTH_REDIRECT_URL"})
		}
	}

	params := url.Values{}
	params.Set("client_id", s.config.GoogleOAuthClientID)
	params.Set("redirect_uri", s.config.GoogleOAuthRedirectURL)
	params.Set("response_type", "code")
	params.Set("scope", "openid email profile")
	params.Set("state", state)
	params.Set("access_type", "offline")
	params.Set("prompt", "consent")

	return fmt.Sprintf("https://accounts.google.com/o/oauth2/v2/auth?%s", params.Encode())
}

// ExchangeCodeForToken exchanges the authorization code for an access token
func (s *OAuthService) ExchangeCodeForToken(ctx context.Context, code string) (result0 *GoogleTokenResponse, err error) {
	ctx, span := observability.TraceOAuthFunction(ctx, "exchange_code_for_token",
		attribute.String("oauth.code", code),
		attribute.String("oauth.token_endpoint", s.TokenEndpoint),
	)
	defer observability.FinishSpan(span, &err)

	data := url.Values{}
	data.Set("client_id", s.config.GoogleOAuthClientID)
	data.Set("client_secret", s.config.GoogleOAuthClientSecret)
	data.Set("code", code)
	data.Set("grant_type", "authorization_code")
	data.Set("redirect_uri", s.config.GoogleOAuthRedirectURL)

	tokenURL := s.TokenEndpoint
	if tokenURL == "" {
		tokenURL = "https://oauth2.googleapis.com/token"
	}

	req, err := http.NewRequest("POST", tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		span.SetAttributes(attribute.String("error", err.Error()))
		return nil, contextutils.WrapError(err, "failed to create token request")
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// Use instrumented HTTP client for automatic tracing with explicit span options
	client := &http.Client{
		Timeout: config.OAuthHTTPTimeout,
		Transport: otelhttp.NewTransport(http.DefaultTransport,
			otelhttp.WithSpanOptions(trace.WithSpanKind(trace.SpanKindClient)),
		),
	}
	resp, err := client.Do(req.WithContext(ctx))
	if err != nil {
		span.SetAttributes(attribute.String("error", err.Error()))
		return nil, contextutils.WrapError(err, "failed to exchange code for token")
	}
	defer func() {
		cerr := resp.Body.Close()
		if cerr != nil {
			s.logger.Warn(ctx, "Failed to close response body", map[string]interface{}{"error": cerr.Error()})
		}
	}()

	span.SetAttributes(attribute.Int("http.status_code", resp.StatusCode))

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)

		// Try to parse the error response for better error messages
		var errorResp struct {
			Error            string `json:"error"`
			ErrorDescription string `json:"error_description"`
		}

		if json.Unmarshal(body, &errorResp) == nil {
			span.SetAttributes(
				attribute.String("oauth.error", errorResp.Error),
				attribute.String("oauth.error_description", errorResp.ErrorDescription),
			)
			switch errorResp.Error {
			case "invalid_grant":
				return nil, contextutils.WrapErrorf(ErrOAuthCodeAlreadyUsed, "please try signing in again")
			case "invalid_client":
				return nil, contextutils.WrapError(ErrOAuthClientConfig, "")
			case "invalid_request":
				return nil, contextutils.WrapError(ErrOAuthInvalidRequest, "")
			case "unauthorized_client":
				return nil, contextutils.WrapError(ErrOAuthUnauthorized, "")
			case "unsupported_grant_type":
				return nil, contextutils.WrapError(ErrOAuthUnsupportedGrant, "")
			default:
				return nil, contextutils.WrapErrorf(contextutils.ErrOAuthProviderError, "OAuth error: %s - %s", errorResp.Error, errorResp.ErrorDescription)
			}
		}

		return nil, contextutils.WrapErrorf(contextutils.ErrOAuthProviderError, "token exchange failed with status %d: %s", resp.StatusCode, string(body))
	}

	var tokenResp GoogleTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		span.SetAttributes(attribute.String("error", err.Error()))
		return nil, contextutils.WrapError(err, "failed to decode token response")
	}

	span.SetAttributes(
		attribute.String("oauth.token_type", tokenResp.TokenType),
		attribute.Int("oauth.expires_in", tokenResp.ExpiresIn),
	)

	return &tokenResp, nil
}

// GetGoogleUserInfo retrieves user information from Google using the access token
func (s *OAuthService) GetGoogleUserInfo(ctx context.Context, accessToken string) (result0 *GoogleUserInfo, err error) {
	ctx, span := observability.TraceOAuthFunction(ctx, "get_google_user_info",
		attribute.String("oauth.userinfo_endpoint", s.UserInfoEndpoint),
	)
	defer observability.FinishSpan(span, &err)

	userinfoURL := s.UserInfoEndpoint
	if userinfoURL == "" {
		userinfoURL = "https://www.googleapis.com/oauth2/v2/userinfo"
	}

	req, err := http.NewRequest("GET", userinfoURL, nil)
	if err != nil {
		span.SetAttributes(attribute.String("error", err.Error()))
		return nil, contextutils.WrapError(err, "failed to create userinfo request")
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)

	// Use instrumented HTTP client for automatic tracing with explicit span options
	client := &http.Client{
		Timeout: config.OAuthHTTPTimeout,
		Transport: otelhttp.NewTransport(http.DefaultTransport,
			otelhttp.WithSpanOptions(trace.WithSpanKind(trace.SpanKindClient)),
		),
	}
	resp, err := client.Do(req.WithContext(ctx))
	if err != nil {
		span.SetAttributes(attribute.String("error", err.Error()))
		return nil, contextutils.WrapError(err, "failed to get user info")
	}
	defer func() {
		cerr := resp.Body.Close()
		if cerr != nil {
			s.logger.Warn(ctx, "Failed to close response body", map[string]interface{}{"error": cerr.Error()})
		}
	}()

	span.SetAttributes(attribute.Int("http.status_code", resp.StatusCode))

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		span.SetAttributes(attribute.String("error", fmt.Sprintf("userinfo request failed with status %d: %s", resp.StatusCode, string(body))))
		return nil, contextutils.WrapErrorf(contextutils.ErrOAuthProviderError, "userinfo request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var userInfo GoogleUserInfo
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		span.SetAttributes(attribute.String("error", err.Error()))
		return nil, contextutils.WrapError(err, "failed to decode user info")
	}

	span.SetAttributes(
		attribute.String("user.email", userInfo.Email),
		attribute.String("user.id", userInfo.ID),
		attribute.Bool("user.verified_email", userInfo.VerifiedEmail),
	)

	return &userInfo, nil
}

// AuthenticateGoogleUser handles the complete Google OAuth flow
func (s *OAuthService) AuthenticateGoogleUser(ctx context.Context, code string, userService UserServiceInterface) (result0 *models.User, err error) {
	ctx, span := observability.TraceOAuthFunction(ctx, "authenticate_google_user",
		attribute.String("oauth.code", code),
	)
	defer observability.FinishSpan(span, &err)

	// Exchange code for token
	tokenResp, err := s.ExchangeCodeForToken(ctx, code)
	if err != nil {
		span.SetAttributes(attribute.String("error", err.Error()))
		return nil, contextutils.WrapError(err, "failed to exchange code for token")
	}

	// Get user info from Google
	userInfo, err := s.GetGoogleUserInfo(ctx, tokenResp.AccessToken)
	if err != nil {
		span.SetAttributes(attribute.String("error", err.Error()))
		return nil, contextutils.WrapError(err, "failed to get user info")
	}

	span.SetAttributes(
		attribute.String("user.email", userInfo.Email),
		attribute.String("user.id", userInfo.ID),
	)

	// Check if user exists by email
	existingUser, err := userService.GetUserByEmail(ctx, userInfo.Email)
	if err != nil {
		span.SetAttributes(attribute.String("error", err.Error()))
		return nil, contextutils.WrapError(err, "failed to check existing user")
	}

	if existingUser != nil {
		// User exists, return the user
		span.SetAttributes(
			attribute.Int("user.id", existingUser.ID),
			attribute.String("auth.result", "existing_user"),
		)
		return existingUser, nil
	}

	// Check if signups are disabled before creating new user
	if s.config != nil && s.config.IsSignupDisabled() {
		// Check if OAuth signup is allowed via whitelist
		if !s.config.IsOAuthSignupAllowed(userInfo.Email) {
			span.SetAttributes(
				attribute.String("auth.result", "oauth_signup_blocked"),
				attribute.String("user.email", userInfo.Email),
			)
			return nil, ErrSignupsDisabled
		}
		// Allow OAuth signup for whitelisted email/domain
		span.SetAttributes(
			attribute.String("auth.result", "oauth_signup_allowed"),
			attribute.String("user.email", userInfo.Email),
		)
	}

	// User doesn't exist, create new user
	// Use email as username (we'll handle conflicts)
	username := userInfo.Email
	email := userInfo.Email

	// Check if username already exists, if so, append a number
	counter := 1
	for {
		existingUser, err := userService.GetUserByUsername(ctx, username)
		if err != nil {
			span.SetAttributes(attribute.String("error", err.Error()))
			return nil, contextutils.WrapError(err, "failed to check username availability")
		}
		if existingUser == nil {
			break
		}
		username = fmt.Sprintf("%s_%d", userInfo.Email, counter)
		counter++
	}

	span.SetAttributes(
		attribute.String("user.username", username),
		attribute.String("user.email", email),
		attribute.String("auth.result", "new_user"),
	)

	// Create user with default settings
	// Use email as username (we'll handle conflicts)
	user, err := userService.CreateUserWithEmailAndTimezone(ctx, username, email, "UTC", "italian", "beginner")
	if err != nil {
		span.SetAttributes(attribute.String("error", err.Error()))
		return nil, contextutils.WrapError(err, "failed to create user")
	}

	span.SetAttributes(attribute.Int("user.id", user.ID))

	return user, nil
}
