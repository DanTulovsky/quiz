package handlers

import (
	"net/http"

	"quizapp/internal/middleware"
	"quizapp/internal/observability"
	"quizapp/internal/services"
	contextutils "quizapp/internal/utils"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

// IOSHandler handles iOS-specific HTTP requests
type IOSHandler struct {
	userService services.UserServiceInterface
	logger      *observability.Logger
}

// NewIOSHandler creates a new IOSHandler instance
func NewIOSHandler(userService services.UserServiceInterface, logger *observability.Logger) *IOSHandler {
	return &IOSHandler{
		userService: userService,
		logger:      logger,
	}
}

// RegisterDeviceTokenRequest represents the request body for device token registration
type RegisterDeviceTokenRequest struct {
	DeviceToken string `json:"device_token" binding:"required"`
}

// RegisterDeviceToken registers or updates a device token for the authenticated user
func (h *IOSHandler) RegisterDeviceToken(c *gin.Context) {
	ctx, span := observability.TraceHandlerFunction(c.Request.Context(), "register_device_token")
	defer observability.FinishSpan(span, nil)

	session := sessions.Default(c)
	userID, ok := session.Get(middleware.UserIDKey).(int)
	if !ok {
		HandleAppError(c, contextutils.ErrUnauthorized)
		return
	}

	var body RegisterDeviceTokenRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		HandleAppError(c, contextutils.NewAppErrorWithCause(
			contextutils.ErrorCodeInvalidInput,
			contextutils.SeverityWarn,
			"Invalid request body",
			"",
			err,
		))
		return
	}

	if err := h.userService.RegisterDeviceToken(ctx, userID, body.DeviceToken); err != nil {
		HandleAppError(c, contextutils.WrapError(err, "failed to register device token"))
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// RemoveDeviceToken removes a device token for the authenticated user
func (h *IOSHandler) RemoveDeviceToken(c *gin.Context) {
	ctx, span := observability.TraceHandlerFunction(c.Request.Context(), "remove_device_token")
	defer observability.FinishSpan(span, nil)

	session := sessions.Default(c)
	userID, ok := session.Get(middleware.UserIDKey).(int)
	if !ok {
		HandleAppError(c, contextutils.ErrUnauthorized)
		return
	}

	var body RegisterDeviceTokenRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		HandleAppError(c, contextutils.NewAppErrorWithCause(
			contextutils.ErrorCodeInvalidInput,
			contextutils.SeverityWarn,
			"Invalid request body",
			"",
			err,
		))
		return
	}

	if err := h.userService.RemoveDeviceToken(ctx, userID, body.DeviceToken); err != nil {
		if contextutils.IsError(err, contextutils.ErrRecordNotFound) {
			HandleAppError(c, err)
			return
		}
		HandleAppError(c, contextutils.WrapError(err, "failed to remove device token"))
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}
