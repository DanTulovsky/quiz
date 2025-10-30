package services

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"testing"

	"quizapp/internal/config"
	"quizapp/internal/models"

	"quizapp/internal/observability"

	"github.com/stretchr/testify/assert"
)

type mockUserService struct {
	usersByEmail    map[string]*models.User
	usersByUsername map[string]*models.User
	createdUsers    []*models.User
}

func (m *mockUserService) GetUserByEmail(_ context.Context, email string) (result0 *models.User, err error) {
	return m.usersByEmail[email], nil
}

func (m *mockUserService) GetUserByUsername(_ context.Context, username string) (result0 *models.User, err error) {
	return m.usersByUsername[username], nil
}

func (m *mockUserService) CreateUserWithEmailAndTimezone(_ context.Context, username, email, _, _, _ string) (result0 *models.User, err error) {
	user := &models.User{
		ID:       len(m.createdUsers) + 1,
		Username: username,
		Email:    sql.NullString{String: email, Valid: true},
	}
	m.usersByEmail[email] = user
	m.usersByUsername[username] = user
	m.createdUsers = append(m.createdUsers, user)
	return user, nil
}

// The rest are not needed for this test
func (m *mockUserService) CreateUser(_ context.Context, _, _, _ string) (result0 *models.User, err error) {
	return nil, nil
}

func (m *mockUserService) CreateUserWithPassword(_ context.Context, _, _, _, _ string) (result0 *models.User, err error) {
	return nil, nil
}

func (m *mockUserService) AuthenticateUser(_ context.Context, _, _ string) (result0 *models.User, err error) {
	return nil, nil
}

func (m *mockUserService) UpdateUserSettings(_ context.Context, _ int, _ *models.UserSettings) error {
	return nil
}

func (m *mockUserService) UpdateUserProfile(_ context.Context, _ int, _, _, _ string) error {
	return nil
}
func (m *mockUserService) UpdateUserPassword(_ context.Context, _ int, _ string) error { return nil }
func (m *mockUserService) UpdateLastActive(_ context.Context, _ int) error             { return nil }
func (m *mockUserService) GetAllUsers(_ context.Context) ([]models.User, error)        { return nil, nil }
func (m *mockUserService) DeleteUser(_ context.Context, _ int) error                   { return nil }
func (m *mockUserService) DeleteAllUsers(_ context.Context) error                      { return nil }
func (m *mockUserService) EnsureAdminUserExists(_ context.Context, _, _ string) error {
	return nil
}
func (m *mockUserService) ResetDatabase(_ context.Context) error               { return nil }
func (m *mockUserService) ClearUserData(_ context.Context) error               { return nil }
func (m *mockUserService) ClearUserDataForUser(_ context.Context, _ int) error { return nil }
func (m *mockUserService) GetUserAPIKey(_ context.Context, _ int, _ string) (result0 string, err error) {
	return "", nil
}

func (m *mockUserService) GetUserAPIKeyWithID(_ context.Context, _ int, _ string) (string, *int, error) {
	return "", nil, nil
}
func (m *mockUserService) SetUserAPIKey(_ context.Context, _ int, _, _ string) error { return nil }
func (m *mockUserService) HasUserAPIKey(_ context.Context, _ int, _ string) (result0 bool, err error) {
	return false, nil
}

// Role management methods
func (m *mockUserService) GetUserRoles(_ context.Context, _ int) (result0 []models.Role, err error) {
	return []models.Role{}, nil
}

func (m *mockUserService) AssignRole(_ context.Context, _, _ int) error {
	return nil
}

func (m *mockUserService) AssignRoleByName(_ context.Context, _ int, _ string) error {
	return nil
}

func (m *mockUserService) RemoveRole(_ context.Context, _, _ int) error {
	return nil
}

func (m *mockUserService) HasRole(_ context.Context, _ int, _ string) (result0 bool, err error) {
	return false, nil
}

func (m *mockUserService) IsAdmin(_ context.Context, _ int) (result0 bool, err error) {
	return false, nil
}

func (m *mockUserService) GetDB() *sql.DB {
	return nil
}

func (m *mockUserService) GetAllRoles(_ context.Context) (result0 []models.Role, err error) {
	return []models.Role{}, nil
}

func (m *mockUserService) GetUsersPaginated(_ context.Context, _, _ int, _, _, _, _, _, _, _ string) (result0 []models.User, result1 int, err error) {
	return nil, 0, nil
}

func (m *mockUserService) GetUserByID(_ context.Context, id int) (result0 *models.User, err error) {
	for _, user := range m.usersByEmail {
		if user.ID == id {
			return user, nil
		}
	}
	return nil, nil
}

func (m *mockUserService) GetUsersWithEmail(_ context.Context) (result0 []models.User, err error) {
	var users []models.User
	for _, user := range m.usersByEmail {
		users = append(users, *user)
	}
	return users, nil
}

func (m *mockUserService) UpdateWordOfDayEmailEnabled(_ context.Context, _ int, _ bool) error {
	return nil
}

func TestAuthenticateGoogleUser_MockedEndpoints(t *testing.T) {
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if _, err := w.Write([]byte(`{"access_token":"fake-access-token","token_type":"Bearer","expires_in":3600}`)); err != nil {
			t.Fatalf("failed to write response: %v", err)
		}
	}))
	defer tokenServer.Close()

	userinfoServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if _, err := w.Write([]byte(`{"id":"123","email":"test@example.com","verified_email":true}`)); err != nil {
			t.Fatalf("failed to write response: %v", err)
		}
	}))
	defer userinfoServer.Close()

	cfg := &config.Config{
		GoogleOAuthClientID:     "test-client-id",
		GoogleOAuthClientSecret: "test-client-secret",
		GoogleOAuthRedirectURL:  "http://localhost:3000/oauth-callback",
	}
	// Create OAuth service
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	oauthService := NewOAuthServiceWithLogger(cfg, logger)
	oauthService.TokenEndpoint = tokenServer.URL
	oauthService.UserInfoEndpoint = userinfoServer.URL

	mockUsers := &mockUserService{
		usersByEmail:    make(map[string]*models.User),
		usersByUsername: make(map[string]*models.User),
	}

	user, err := oauthService.AuthenticateGoogleUser(context.Background(), "fake-code", mockUsers)
	assert.NoError(t, err)
	assert.NotNil(t, user)
	assert.Equal(t, "test@example.com", user.Email.String)
	assert.Equal(t, "test@example.com", user.Username)

	// Second call should return the same user (simulate login for existing user)
	existing, err := oauthService.AuthenticateGoogleUser(context.Background(), "fake-code", mockUsers)
	assert.NoError(t, err)
	assert.Equal(t, user, existing)
}
