package handlers

import (
	"context"
	"fmt"
	"html/template"
	"net/http"
	"strings"
	"time"

	"quizapp/internal/config"
	"quizapp/internal/models"
	"quizapp/internal/observability"
	"quizapp/internal/services"
	contextutils "quizapp/internal/utils"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel/attribute"
)

// WordOfTheDayHandler handles word of the day HTTP requests
type WordOfTheDayHandler struct {
	userService         services.UserServiceInterface
	wordOfTheDayService services.WordOfTheDayServiceInterface
	cfg                 *config.Config
	logger              *observability.Logger
}

// NewWordOfTheDayHandler creates a new WordOfTheDayHandler
func NewWordOfTheDayHandler(
	userService services.UserServiceInterface,
	wordOfTheDayService services.WordOfTheDayServiceInterface,
	cfg *config.Config,
	logger *observability.Logger,
) *WordOfTheDayHandler {
	return &WordOfTheDayHandler{
		userService:         userService,
		wordOfTheDayService: wordOfTheDayService,
		cfg:                 cfg,
		logger:              logger,
	}
}

// ParseDateInUserTimezone parses a date string in the user's timezone
func (h *WordOfTheDayHandler) ParseDateInUserTimezone(ctx context.Context, userID int, dateStr string) (time.Time, string, error) {
	// Delegate to shared util with injected user lookup
	return contextutils.ParseDateInUserTimezone(ctx, userID, dateStr, h.userService.GetUserByID)
}

// GetWordOfTheDay handles GET /v1/word-of-day/:date
func (h *WordOfTheDayHandler) GetWordOfTheDay(c *gin.Context) {
	ctx, span := observability.TraceHandlerFunction(c.Request.Context(), "get_word_of_the_day")
	defer observability.FinishSpan(span, nil)

	userID, exists := GetUserIDFromSession(c)
	if !exists {
		HandleAppError(c, contextutils.ErrUnauthorized)
		return
	}

	// Parse date parameter
	dateStr := c.Param("date")
	if dateStr == "" {
		HandleAppError(c, contextutils.ErrMissingRequired)
		return
	}

	// Parse date in user's timezone
	date, timezone, err := h.ParseDateInUserTimezone(ctx, userID, dateStr)
	if err != nil {
		if strings.Contains(err.Error(), "invalid date format") {
			HandleAppError(c, contextutils.ErrInvalidFormat)
			return
		}
		HandleAppError(c, contextutils.WrapError(err, "failed to get user information"))
		return
	}

	span.SetAttributes(
		observability.AttributeUserID(userID),
		attribute.String("date", dateStr),
		attribute.String("timezone", timezone),
	)

	// Get word of the day
	word, err := h.wordOfTheDayService.GetWordOfTheDay(ctx, userID, date)
	if err != nil {
		h.logger.Error(ctx, "Failed to get word of the day", err, map[string]interface{}{
			"user_id": userID,
			"date":    dateStr,
		})
		HandleAppError(c, contextutils.WrapError(err, "failed to get word of the day"))
		return
	}

	c.JSON(http.StatusOK, word)
}

// GetWordOfTheDayToday handles GET /v1/word-of-day
// It resolves "today" in the user's timezone and returns that day's word
func (h *WordOfTheDayHandler) GetWordOfTheDayToday(c *gin.Context) {
	ctx, span := observability.TraceHandlerFunction(c.Request.Context(), "get_word_of_the_day_today")
	defer observability.FinishSpan(span, nil)

	userID, exists := GetUserIDFromSession(c)
	if !exists {
		HandleAppError(c, contextutils.ErrUnauthorized)
		return
	}

	// Determine today's date string and parse it in user's timezone
	todayStr := time.Now().Format("2006-01-02")
	date, timezone, err := h.ParseDateInUserTimezone(ctx, userID, todayStr)
	if err != nil {
		HandleAppError(c, contextutils.WrapError(err, "failed to resolve today's date"))
		return
	}

	span.SetAttributes(
		observability.AttributeUserID(userID),
		attribute.String("date", todayStr),
		attribute.String("timezone", timezone),
	)

	// Get word of the day
	word, err := h.wordOfTheDayService.GetWordOfTheDay(ctx, userID, date)
	if err != nil {
		h.logger.Error(ctx, "Failed to get today's word of the day", err, map[string]interface{}{
			"user_id": userID,
			"date":    todayStr,
		})
		HandleAppError(c, contextutils.WrapError(err, "failed to get word of the day"))
		return
	}

	c.JSON(http.StatusOK, word)
}

// GetWordOfTheDayEmbed handles GET /v1/word-of-day/:date/embed
// This endpoint returns HTML for embedding in an iframe. Requires an authenticated session.
func (h *WordOfTheDayHandler) GetWordOfTheDayEmbed(c *gin.Context) {
	ctx, span := observability.TraceHandlerFunction(c.Request.Context(), "get_word_of_the_day_embed")
	defer observability.FinishSpan(span, nil)

	// Determine user via session; no query parameters are supported
	userID, exists := GetUserIDFromSession(c)
	if !exists {
		c.Data(http.StatusUnauthorized, "text/html; charset=utf-8", []byte("Unauthorized"))
		return
	}

	// Resolve date parameter from path, query, or default to today's date
	dateStr := c.Param("date")
	if dateStr == "" {
		dateStr = c.Query("date")
	}
	if dateStr == "" {
		dateStr = time.Now().Format("2006-01-02")
	}

	// Parse date in user's timezone
	date, timezone, err := h.ParseDateInUserTimezone(ctx, userID, dateStr)
	if err != nil {
		c.Data(http.StatusBadRequest, "text/html; charset=utf-8", []byte("Invalid date format"))
		return
	}

	span.SetAttributes(
		observability.AttributeUserID(userID),
		attribute.String("date", dateStr),
		attribute.String("timezone", timezone),
	)

	// Get word of the day
	word, err := h.wordOfTheDayService.GetWordOfTheDay(ctx, userID, date)
	if err != nil {
		h.logger.Error(ctx, "Failed to get word of the day for embed", err, map[string]interface{}{
			"user_id": userID,
			"date":    dateStr,
		})
		c.Data(http.StatusInternalServerError, "text/html; charset=utf-8", []byte("Failed to load word of the day"))
		return
	}

	// Render HTML template
	html := h.renderEmbedHTML(word)
	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(html))
}

// GetWordOfTheDayHistory handles GET /v1/word-of-day/history
func (h *WordOfTheDayHandler) GetWordOfTheDayHistory(c *gin.Context) {
	ctx, span := observability.TraceHandlerFunction(c.Request.Context(), "get_word_of_the_day_history")
	defer observability.FinishSpan(span, nil)

	userID, exists := GetUserIDFromSession(c)
	if !exists {
		HandleAppError(c, contextutils.ErrUnauthorized)
		return
	}

	// Parse date range parameters
	startDateStr := c.Query("start_date")
	endDateStr := c.Query("end_date")

	if startDateStr == "" || endDateStr == "" {
		HandleAppError(c, contextutils.ErrMissingRequired)
		return
	}

	// Parse dates in user's timezone
	startDate, _, err := h.ParseDateInUserTimezone(ctx, userID, startDateStr)
	if err != nil {
		HandleAppError(c, contextutils.WrapError(err, "invalid start_date"))
		return
	}

	endDate, _, err := h.ParseDateInUserTimezone(ctx, userID, endDateStr)
	if err != nil {
		HandleAppError(c, contextutils.WrapError(err, "invalid end_date"))
		return
	}

	span.SetAttributes(
		observability.AttributeUserID(userID),
		attribute.String("start_date", startDateStr),
		attribute.String("end_date", endDateStr),
	)

	// Get word history
	words, err := h.wordOfTheDayService.GetWordHistory(ctx, userID, startDate, endDate)
	if err != nil {
		h.logger.Error(ctx, "Failed to get word of the day history", err, map[string]interface{}{
			"user_id":    userID,
			"start_date": startDateStr,
			"end_date":   endDateStr,
		})
		HandleAppError(c, contextutils.WrapError(err, "failed to get word history"))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"words": words,
		"count": len(words),
	})
}

// renderEmbedHTML renders the embed HTML template
func (h *WordOfTheDayHandler) renderEmbedHTML(word *models.WordOfTheDayDisplay) string {
	if word == nil {
		// Gracefully handle missing word to avoid panics in tests/environments with no data
		return "<html><head><meta charset=\"UTF-8\"></head><body>Word of the Day is unavailable.</body></html>"
	}
	const embedTemplate = `
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Word of the Day</title>
    <style>
        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
        }
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            color: #333;
            padding: 20px;
            min-height: 100vh;
            display: flex;
            align-items: center;
            justify-content: center;
        }
        .card {
            background: white;
            border-radius: 16px;
            box-shadow: 0 10px 30px rgba(0, 0, 0, 0.2);
            padding: 30px;
            max-width: 500px;
            width: 100%;
        }
        .date {
            color: #667eea;
            font-size: 14px;
            font-weight: 600;
            text-transform: uppercase;
            letter-spacing: 1px;
            margin-bottom: 10px;
        }
        .word {
            font-size: 48px;
            font-weight: bold;
            color: #1a1a1a;
            margin-bottom: 10px;
            line-height: 1.2;
        }
        .translation {
            font-size: 24px;
            color: #667eea;
            margin-bottom: 20px;
            font-style: italic;
        }
        .sentence {
            font-size: 18px;
            line-height: 1.6;
            color: #555;
            background: #f7f7f7;
            padding: 20px;
            border-radius: 8px;
            border-left: 4px solid #667eea;
            margin-bottom: 15px;
        }
        .meta {
            display: flex;
            gap: 10px;
            flex-wrap: wrap;
            margin-top: 20px;
        }
        .badge {
            background: #e0e7ff;
            color: #667eea;
            padding: 6px 12px;
            border-radius: 20px;
            font-size: 12px;
            font-weight: 600;
        }
        .explanation {
            font-size: 14px;
            color: #666;
            margin-top: 15px;
            padding: 15px;
            background: #fafafa;
            border-radius: 8px;
            border-left: 3px solid #764ba2;
        }
    </style>
</head>
<body>
    <div class="card">
        <div class="date">{{.FormattedDate}}</div>
        <div class="word">{{.Word}}</div>
        <div class="translation">{{.Translation}}</div>
        {{if .Sentence}}
        <div class="sentence">{{.Sentence}}</div>
        {{end}}
        <div class="meta">
            {{if .Language}}<span class="badge">{{.Language}}</span>{{end}}
            {{if .Level}}<span class="badge">{{.Level}}</span>{{end}}
            {{if .TopicCategory}}<span class="badge">{{.TopicCategory}}</span>{{end}}
        </div>
        {{if .Explanation}}
        <div class="explanation">{{.Explanation}}</div>
        {{end}}
    </div>
</body>
</html>
`

	tmpl, err := template.New("embed").Parse(embedTemplate)
	if err != nil {
		return fmt.Sprintf("<html><body>Error rendering template: %v</body></html>", err)
	}

	data := struct {
		FormattedDate string
		Word          string
		Translation   string
		Sentence      string
		Language      string
		Level         string
		TopicCategory string
		Explanation   string
	}{
		FormattedDate: word.Date.Format("January 2, 2006"),
		Word:          word.Word,
		Translation:   word.Translation,
		Sentence:      word.Sentence,
		Language:      word.Language,
		Level:         word.Level,
		TopicCategory: word.TopicCategory,
		Explanation:   word.Explanation,
	}

	var buf strings.Builder
	if err := tmpl.Execute(&buf, data); err != nil {
		return fmt.Sprintf("<html><body>Error executing template: %v</body></html>", err)
	}

	return buf.String()
}
