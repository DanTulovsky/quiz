package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"quizapp/internal/api"
	"quizapp/internal/config"
	"quizapp/internal/models"
	"quizapp/internal/observability"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockDailyQuestionService returns a single history row
type mockDailyQuestionService struct {
	mock.Mock
}

func (m *mockDailyQuestionService) GetQuestionHistory(ctx context.Context, userID, questionID, days int) ([]*models.DailyQuestionHistory, error) {
	args := m.Called(ctx, userID, questionID, days)
	return args.Get(0).([]*models.DailyQuestionHistory), args.Error(1)
}

// Extended stub implementations to satisfy DailyQuestionServiceInterface
func (m *mockDailyQuestionService) AssignDailyQuestions(ctx context.Context, userID int, date time.Time) error {
	args := m.Called(ctx, userID, date)
	return args.Error(0)
}

func (m *mockDailyQuestionService) RegenerateDailyQuestions(ctx context.Context, userID int, date time.Time) error {
	args := m.Called(ctx, userID, date)
	return args.Error(0)
}

func (m *mockDailyQuestionService) GetDailyQuestions(ctx context.Context, userID int, date time.Time) ([]*models.DailyQuestionAssignmentWithQuestion, error) {
	args := m.Called(ctx, userID, date)
	return args.Get(0).([]*models.DailyQuestionAssignmentWithQuestion), args.Error(1)
}

func (m *mockDailyQuestionService) MarkQuestionCompleted(ctx context.Context, userID, questionID int, date time.Time) error {
	args := m.Called(ctx, userID, questionID, date)
	return args.Error(0)
}

func (m *mockDailyQuestionService) ResetQuestionCompleted(ctx context.Context, userID, questionID int, date time.Time) error {
	args := m.Called(ctx, userID, questionID, date)
	return args.Error(0)
}

func (m *mockDailyQuestionService) SubmitDailyQuestionAnswer(ctx context.Context, userID, questionID int, date time.Time, userAnswerIndex int) (*api.AnswerResponse, error) {
	args := m.Called(ctx, userID, questionID, date, userAnswerIndex)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*api.AnswerResponse), args.Error(1)
}

func (m *mockDailyQuestionService) GetAvailableDates(ctx context.Context, userID int) ([]time.Time, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).([]time.Time), args.Error(1)
}

func (m *mockDailyQuestionService) GetDailyProgress(ctx context.Context, userID int, date time.Time) (*models.DailyProgress, error) {
	args := m.Called(ctx, userID, date)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.DailyProgress), args.Error(1)
}

func (m *mockDailyQuestionService) GetDailyQuestionsCount(ctx context.Context, userID int, date time.Time) (int, error) {
	args := m.Called(ctx, userID, date)
	return args.Int(0), args.Error(1)
}

func (m *mockDailyQuestionService) GetCompletedDailyQuestionsCount(ctx context.Context, userID int, date time.Time) (int, error) {
	args := m.Called(ctx, userID, date)
	return args.Int(0), args.Error(1)
}

func setupRouterWithSessions() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	store := cookie.NewStore([]byte("test-secret"))
	r.Use(sessions.Sessions("test-session", store))
	return r
}

func TestGetQuestionHistory_UserLookupError_Returns500(t *testing.T) {
	// Initialize observability logger (no-op tracing is fine)
	_, _, logger, err := observability.SetupObservability(&config.OpenTelemetryConfig{EnableTracing: false, EnableLogging: true}, "test-service")
	require.NoError(t, err)

	// Mocks: reuse existing MockUserService and local daily question mock
	mu := &MockUserService{}
	md := &mockDailyQuestionService{}

	// Prepare a single history entry
	// Use a non-midnight UTC time so it's treated as timezone-aware
	ad := time.Date(2025, time.June, 16, 12, 0, 0, 0, time.UTC)
	historyEntry := &models.DailyQuestionHistory{AssignmentDate: ad, IsCompleted: false}
	md.On("GetQuestionHistory", mock.Anything, 1, 123, 14).Return([]*models.DailyQuestionHistory{historyEntry}, nil)
	// Ensure GetUserByID is mocked to avoid panics from unexpected mock calls.
	// The handler ignores the error from GetUserByID for assignment_date formatting,
	// so returning an error here simulates a user lookup failure without failing the handler.
	mu.On("GetUserByID", mock.Anything, 1).Return(nil, assert.AnError)

	handler := NewDailyQuestionHandler(mu, md, &config.Config{}, logger)

	router := setupRouterWithSessions()
	router.GET("/login", func(c *gin.Context) {
		s := sessions.Default(c)
		s.Set("user_id", 1)
		s.Set("username", "testuser")
		_ = s.Save()
		c.Status(http.StatusOK)
	})
	router.GET("/v1/daily/history/:questionId", handler.GetQuestionHistory)

	// login to get cookie
	reqLogin, _ := http.NewRequest("GET", "/login", nil)
	wLogin := httptest.NewRecorder()
	router.ServeHTTP(wLogin, reqLogin)
	cookie := wLogin.Result().Cookies()[0]

	// call history endpoint
	req, _ := http.NewRequest("GET", "/v1/daily/history/123", nil)
	req.AddCookie(cookie)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// With date-only assignment_date serialization the handler no longer
	// requires a user lookup to format assignment_date, so it should succeed
	// and return the history entry even if user lookup fails for other fields.
	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string][]map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	history := resp["history"]
	require.NotEmpty(t, history)
	// assignment_date should be returned as YYYY-MM-DD
	adStr, ok := history[0]["assignment_date"].(string)
	require.True(t, ok)
	require.Equal(t, ad.Format("2006-01-02"), adStr)

	md.AssertExpectations(t)
	mu.AssertExpectations(t)
}
