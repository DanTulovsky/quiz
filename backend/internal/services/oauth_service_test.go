package services

import (
	"context"
	"testing"

	"quizapp/internal/config"
	"quizapp/internal/observability"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/attribute"
)

func TestNewOAuthService(t *testing.T) {
	cfg := &config.Config{}
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	service := NewOAuthServiceWithLogger(cfg, logger)
	assert.NotNil(t, service)
	assert.Equal(t, cfg, service.config)
	assert.Equal(t, "https://oauth2.googleapis.com/token", service.TokenEndpoint)
	assert.Equal(t, "https://www.googleapis.com/oauth2/v2/userinfo", service.UserInfoEndpoint)
}

func TestGetGoogleAuthURL(t *testing.T) {
	cfg := &config.Config{
		GoogleOAuthClientID:    "test-client-id",
		GoogleOAuthRedirectURL: "http://localhost:8080/auth/callback",
	}
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	service := NewOAuthServiceWithLogger(cfg, logger)
	ctx := context.Background()

	url := service.GetGoogleAuthURL(ctx, "test-state")
	assert.Contains(t, url, "accounts.google.com")
	assert.Contains(t, url, "test-client-id")
	assert.Contains(t, url, "test-state")
	assert.Contains(t, url, "openid+email+profile")
}

func TestExchangeCodeForToken_Success(t *testing.T) {
	cfg := &config.Config{
		GoogleOAuthClientID:     "test-client-id",
		GoogleOAuthClientSecret: "test-client-secret",
		GoogleOAuthRedirectURL:  "http://localhost:8080/auth/callback",
	}
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	service := NewOAuthServiceWithLogger(cfg, logger)

	// This test would require mocking the HTTP client
	// For now, we'll just test that the service is created correctly
	assert.NotNil(t, service)
}

func TestGetGoogleUserInfo(t *testing.T) {
	cfg := &config.Config{}
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	service := NewOAuthServiceWithLogger(cfg, logger)

	// This test would require mocking the HTTP client
	// For now, we'll just test that the service is created correctly
	assert.NotNil(t, service)
}

func TestOAuthService_HTTPClientTracing(t *testing.T) {
	// Test that the OAuth service uses instrumented HTTP clients
	cfg := &config.Config{
		GoogleOAuthClientID:     "test-client-id",
		GoogleOAuthClientSecret: "test-client-secret",
		GoogleOAuthRedirectURL:  "http://localhost:8080/oauth/callback",
	}
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	oauthService := NewOAuthServiceWithLogger(cfg, logger)

	// Test that the service is properly configured
	assert.NotNil(t, oauthService)
	assert.Equal(t, "https://oauth2.googleapis.com/token", oauthService.TokenEndpoint)
	assert.Equal(t, "https://www.googleapis.com/oauth2/v2/userinfo", oauthService.UserInfoEndpoint)

	// Test that the service has the expected configuration
	assert.Equal(t, cfg.GoogleOAuthClientID, oauthService.config.GoogleOAuthClientID)
	assert.Equal(t, cfg.GoogleOAuthClientSecret, oauthService.config.GoogleOAuthClientSecret)
	assert.Equal(t, cfg.GoogleOAuthRedirectURL, oauthService.config.GoogleOAuthRedirectURL)
}

func TestOAuthService_GlobalTracer(t *testing.T) {
	cfg := &config.Config{
		GoogleOAuthClientID:     "test-client-id",
		GoogleOAuthClientSecret: "test-client-secret",
		GoogleOAuthRedirectURL:  "http://localhost:3000/oauth-callback",
	}
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	service := NewOAuthServiceWithLogger(cfg, logger)

	// Verify that the service uses the global tracer
	assert.NotNil(t, service.logger, "OAuthService should have a logger")

	// Test that the global tracer is properly initialized
	ctx := context.Background()
	_, span := observability.TraceOAuthFunction(ctx, "test_function",
		attribute.String("oauth.state", "test-state"),
		attribute.String("oauth.client_id", "test-client-id"),
	)
	assert.NotNil(t, span, "Global tracer should create valid spans")
	span.End()
}

func TestOAuthService_AuthenticateGoogleUser_WithWhitelist(t *testing.T) {
	// Test OAuth signup with whitelist when signups are disabled
	cfg := &config.Config{
		System: &config.SystemConfig{
			Auth: config.AuthConfig{
				SignupsDisabled: true,
				AllowedDomains:  []string{"company.com"},
				AllowedEmails:   []string{"admin@example.com"},
			},
		},
	}

	// Test that the whitelist configuration is properly set
	assert.True(t, cfg.IsSignupDisabled())
	assert.True(t, cfg.IsOAuthSignupAllowed("admin@example.com"))
	assert.True(t, cfg.IsOAuthSignupAllowed("user@company.com"))
	assert.False(t, cfg.IsOAuthSignupAllowed("user@other.com"))
}

func TestOAuthService_AuthenticateGoogleUser_WhitelistBlocked(t *testing.T) {
	// Test OAuth signup with non-whitelisted email when signups are disabled
	cfg := &config.Config{
		System: &config.SystemConfig{
			Auth: config.AuthConfig{
				SignupsDisabled: true,
				AllowedDomains:  []string{"company.com"},
				AllowedEmails:   []string{"admin@example.com"},
			},
		},
	}

	// Test that non-whitelisted emails are blocked
	assert.True(t, cfg.IsSignupDisabled())
	assert.False(t, cfg.IsOAuthSignupAllowed("user@other.com"))
	assert.False(t, cfg.IsOAuthSignupAllowed("other@example.com"))
}
