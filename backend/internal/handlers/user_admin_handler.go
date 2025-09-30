package handlers

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"strings"
	"time"

	"quizapp/internal/api"
	"quizapp/internal/config"
	"quizapp/internal/models"
	"quizapp/internal/observability"
	"quizapp/internal/services"
	contextutils "quizapp/internal/utils"

	"github.com/gin-gonic/gin"
)

// UserAdminHandler handles user management operations
type UserAdminHandler struct {
	userService services.UserServiceInterface
	cfg         *config.Config
	templates   *template.Template
	logger      *observability.Logger
}

// NewUserAdminHandler creates a new UserAdminHandler instance
func NewUserAdminHandler(userService services.UserServiceInterface, cfg *config.Config, logger *observability.Logger) *UserAdminHandler {
	return &UserAdminHandler{
		userService: userService,
		cfg:         cfg,
		templates:   nil,
		logger:      logger,
	}
}

// UserCreateRequest represents a request to create a new user
// Using the generated type from api package for automatic validation
type UserCreateRequest = api.UserCreateRequest

// UserUpdateRequest represents a request to update user profile
// Using the generated type from api package for automatic validation
type UserUpdateRequest = api.UserUpdateRequest

// PasswordResetRequest represents a request to reset user password
// Using the generated type from api package for automatic validation
type PasswordResetRequest = api.PasswordResetRequest

// ProfileResponse represents user profile data
type ProfileResponse struct {
	ID                int           `json:"id"`
	Username          string        `json:"username"`
	Email             *string       `json:"email"`
	Timezone          *string       `json:"timezone"`
	LastActive        *time.Time    `json:"last_active"`
	PreferredLanguage *string       `json:"preferred_language"`
	CurrentLevel      *string       `json:"current_level"`
	CreatedAt         time.Time     `json:"created_at"`
	UpdatedAt         time.Time     `json:"updated_at"`
	AIEnabled         bool          `json:"ai_enabled"`
	AIProvider        *string       `json:"ai_provider"`
	AIModel           *string       `json:"ai_model"`
	Roles             []models.Role `json:"roles,omitempty"`
	IsPaused          bool          `json:"is_paused"`
}

// GetAllUsers handles GET /userz - list all users (admin only) - JSON API
func (h *UserAdminHandler) GetAllUsers(c *gin.Context) {
	users, err := h.userService.GetAllUsers(c.Request.Context())
	if err != nil {
		h.logger.Error(c.Request.Context(), "Error retrieving users", err, nil)
		HandleAppError(c, contextutils.WrapError(err, "failed to retrieve users"))
		return
	}

	// Convert to response format
	var userResponses []ProfileResponse
	for _, user := range users {
		userResponses = append(userResponses, h.convertUserToProfileResponse(c.Request.Context(), &user))
	}

	c.JSON(http.StatusOK, gin.H{"users": userResponses})
}

// GetUsersPaginated handles GET /userz/paginated - list users with pagination (admin only)
func (h *UserAdminHandler) GetUsersPaginated(c *gin.Context) {
	// Parse pagination parameters
	page, pageSize := h.parsePagination(c)

	// Parse filters
	search := c.Query("search")
	language := c.Query("language")
	level := c.Query("level")
	aiProvider := c.Query("ai_provider")
	aiModel := c.Query("ai_model")
	aiEnabled := c.Query("ai_enabled")
	active := c.Query("active")

	// Get paginated users from service
	var users []models.User
	var total int
	var err error
	users, total, err = h.userService.GetUsersPaginated(
		c.Request.Context(),
		page,
		pageSize,
		search,
		language,
		level,
		aiProvider,
		aiModel,
		aiEnabled,
		active,
	)
	if err != nil {
		h.logger.Error(c.Request.Context(), "Error retrieving paginated users", err, map[string]interface{}{
			"page":      page,
			"page_size": pageSize,
			"search":    search,
		})
		HandleAppError(c, contextutils.WrapError(err, "failed to retrieve users"))
		return
	}

	// Convert to response format
	var userResponses []ProfileResponse
	for _, user := range users {
		userResponses = append(userResponses, h.convertUserToProfileResponse(c.Request.Context(), &user))
	}

	// Calculate pagination info
	totalPages := (total + pageSize - 1) / pageSize

	c.JSON(http.StatusOK, gin.H{
		"users": userResponses,
		"pagination": gin.H{
			"page":        page,
			"page_size":   pageSize,
			"total":       total,
			"total_pages": totalPages,
		},
	})
}

// parsePagination parses pagination parameters from the request
func (h *UserAdminHandler) parsePagination(c *gin.Context) (page, pageSize int) {
	page = 1
	pageSize = 20

	if pageStr := c.Query("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	if pageSizeStr := c.Query("page_size"); pageSizeStr != "" {
		if ps, err := strconv.Atoi(pageSizeStr); err == nil && ps > 0 && ps <= 100 {
			pageSize = ps
		}
	}

	return page, pageSize
}

// CreateUser handles POST /userz - create new user (admin only)
func (h *UserAdminHandler) CreateUser(c *gin.Context) {
	var req UserCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		HandleAppError(c, contextutils.NewAppErrorWithCause(
			contextutils.ErrorCodeInvalidInput,
			contextutils.SeverityWarn,
			"Invalid request data",
			"",
			err,
		))
		return
	}

	// Validate required fields
	if req.Username == "" {
		HandleAppError(c, contextutils.ErrMissingRequired)
		return
	}
	if req.Password == "" {
		HandleAppError(c, contextutils.ErrMissingRequired)
		return
	}

	// Extract values from generated types
	timezone := "UTC"
	if req.Timezone != nil && *req.Timezone != "" {
		timezone = *req.Timezone
		// Validate timezone if provided
		if !h.isValidTimezone(timezone) {
			HandleAppError(c, contextutils.ErrInvalidFormat)
			return
		}
	}

	preferredLanguage := "italian"
	if req.PreferredLanguage != nil && *req.PreferredLanguage != "" {
		preferredLanguage = *req.PreferredLanguage
	}

	currentLevel := "A1"
	if req.CurrentLevel != nil && *req.CurrentLevel != "" {
		currentLevel = *req.CurrentLevel
	}

	email := ""
	if req.Email != nil {
		email = string(*req.Email)
	}

	// Check if username already exists
	existingUser, err := h.userService.GetUserByUsername(c.Request.Context(), req.Username)
	if err != nil {
		h.logger.Error(c.Request.Context(), "Error checking existing username", err, nil)
		HandleAppError(c, contextutils.WrapError(err, "failed to check existing username"))
		return
	}
	if existingUser != nil {
		HandleAppError(c, contextutils.ErrRecordExists)
		return
	}

	// Check if email already exists (if provided)
	if email != "" {
		existingUser, err := h.userService.GetUserByEmail(c.Request.Context(), email)
		if err != nil {
			h.logger.Error(c.Request.Context(), "Error checking existing email", err, nil)
			HandleAppError(c, contextutils.WrapError(err, "failed to check email uniqueness"))
			return
		}
		if existingUser != nil {
			HandleAppError(c, contextutils.ErrRecordExists)
			return
		}
	}

	// Create user
	user, err := h.userService.CreateUserWithEmailAndTimezone(
		c.Request.Context(),
		req.Username,
		email,
		timezone,
		preferredLanguage,
		currentLevel,
	)
	if err != nil {
		h.logger.Error(c.Request.Context(), "Error creating user", err, nil)
		HandleAppError(c, contextutils.WrapError(err, "failed to create user"))
		return
	}

	// Set password
	err = h.userService.UpdateUserPassword(c.Request.Context(), user.ID, req.Password)
	if err != nil {
		h.logger.Error(c.Request.Context(), "Error setting user password", err, nil)
		// Try to clean up the created user
		_ = h.userService.DeleteUser(c.Request.Context(), user.ID)
		HandleAppError(c, contextutils.WrapError(err, "failed to set user password"))
		return
	}

	// Return the created user profile
	c.JSON(http.StatusCreated, gin.H{
		"message": "User created successfully",
		"user":    h.convertUserToProfileResponse(c.Request.Context(), user),
	})
}

// UpdateUser handles PUT /userz/:id - update user details (admin or self)
func (h *UserAdminHandler) UpdateUser(c *gin.Context) {
	userIDStr := c.Param("id")
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		HandleAppError(c, contextutils.ErrInvalidFormat)
		return
	}

	// Check if user exists
	user, err := h.userService.GetUserByID(c.Request.Context(), userID)
	if err != nil {
		h.logger.Error(c.Request.Context(), "Error retrieving user", err, nil)
		HandleAppError(c, contextutils.WrapError(err, "database error"))
		return
	}
	if user == nil {
		HandleAppError(c, contextutils.ErrRecordNotFound)
		return
	}

	// Check authorization (admin or self) - skip for direct routes (testing)
	if currentUserID, err := GetCurrentUserID(c); err == nil {
		if err := RequireSelfOrAdmin(c.Request.Context(), h.userService, currentUserID, userID); err != nil {
			if contextutils.IsError(err, contextutils.ErrForbidden) {
				HandleAppError(c, contextutils.ErrForbidden)
				return
			}
			h.logger.Error(c.Request.Context(), "Error checking authorization", err, nil)
			HandleAppError(c, contextutils.WrapError(err, "failed to check authorization"))
			return
		}
	}

	var req UserUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		HandleAppError(c, contextutils.NewAppErrorWithCause(
			contextutils.ErrorCodeInvalidInput,
			contextutils.SeverityWarn,
			"Invalid request data",
			"",
			err,
		))
		return
	}

	// Validate timezone if provided
	if req.Timezone != nil && *req.Timezone != "" && !h.isValidTimezone(*req.Timezone) {
		HandleAppError(c, contextutils.ErrInvalidFormat)
		return
	}

	// Use existing values if not provided in request
	username := user.Username
	if req.Username != nil && *req.Username != "" {
		username = *req.Username
	}

	email := ""
	if user.Email.Valid {
		email = user.Email.String
	}
	if req.Email != nil {
		email = string(*req.Email)
	}

	timezone := ""
	if user.Timezone.Valid {
		timezone = user.Timezone.String
	}
	if req.Timezone != nil && *req.Timezone != "" {
		timezone = *req.Timezone
	}

	preferredLanguage := ""
	if user.PreferredLanguage.Valid {
		preferredLanguage = user.PreferredLanguage.String
	}
	if req.PreferredLanguage != nil && *req.PreferredLanguage != "" {
		preferredLanguage = *req.PreferredLanguage
	}

	currentLevel := ""
	if user.CurrentLevel.Valid {
		currentLevel = user.CurrentLevel.String
	}
	if req.CurrentLevel != nil && *req.CurrentLevel != "" {
		currentLevel = *req.CurrentLevel
	}

	// Check if new username already exists (if changed)
	if username != user.Username {
		existingUser, err := h.userService.GetUserByUsername(c.Request.Context(), username)
		if err != nil {
			h.logger.Error(c.Request.Context(), "Error checking existing username", err, nil)
			HandleAppError(c, contextutils.WrapError(err, "failed to check username uniqueness"))
			return
		}
		if existingUser != nil {
			HandleAppError(c, contextutils.ErrRecordExists)
			return
		}
	}

	// Check if new email already exists (if changed)
	if email != "" && user.Email.Valid && email != user.Email.String {
		existingUser, err := h.userService.GetUserByEmail(c.Request.Context(), email)
		if err != nil {
			h.logger.Error(c.Request.Context(), "Error checking existing email", err, nil)
			HandleAppError(c, contextutils.WrapError(err, "failed to check email uniqueness"))
			return
		}
		if existingUser != nil {
			HandleAppError(c, contextutils.ErrRecordExists)
			return
		}
	}

	// Update user profile
	err = h.userService.UpdateUserProfile(c.Request.Context(), userID, username, email, timezone)
	if err != nil {
		h.logger.Error(c.Request.Context(), "Error updating user profile", err, nil)

		// Check if the error is due to user not found
		if errors.Is(err, contextutils.ErrRecordNotFound) {
			HandleAppError(c, contextutils.ErrRecordNotFound)
			return
		}

		HandleAppError(c, contextutils.WrapError(err, "failed to update user profile"))
		return
	}

	// Handle AI settings update if provided
	needsAIUpdate := req.AiEnabled != nil || (req.AiProvider != nil && *req.AiProvider != "") || (req.AiModel != nil && *req.AiModel != "") || (req.ApiKey != nil && *req.ApiKey != "")
	if needsAIUpdate {
		// Prepare AI settings
		aiSettings := &models.UserSettings{
			Language:  preferredLanguage,
			Level:     currentLevel,
			AIEnabled: req.AiEnabled != nil && *req.AiEnabled,
		}

		// Set AI provider and model
		if req.AiProvider != nil && *req.AiProvider != "" {
			aiSettings.AIProvider = *req.AiProvider
		} else if user.AIProvider.Valid {
			aiSettings.AIProvider = user.AIProvider.String
		}

		if req.AiModel != nil && *req.AiModel != "" {
			aiSettings.AIModel = *req.AiModel
		} else if user.AIModel.Valid {
			aiSettings.AIModel = user.AIModel.String
		}

		// Set API key if provided
		if req.ApiKey != nil && *req.ApiKey != "" {
			aiSettings.AIAPIKey = *req.ApiKey
		}

		// Update AI settings
		err = h.userService.UpdateUserSettings(c.Request.Context(), userID, aiSettings)
		if err != nil {
			h.logger.Error(c.Request.Context(), "Error updating user AI settings", err, nil)

			// Check if the error is due to user not found
			if errors.Is(err, contextutils.ErrRecordNotFound) {
				HandleAppError(c, contextutils.ErrRecordNotFound)
				return
			}

			HandleAppError(c, contextutils.WrapError(err, "failed to update AI settings"))
			return
		}
	}

	// Handle role updates if provided
	if req.SelectedRoles != nil {
		// Get current user roles
		currentRoles, err := h.userService.GetUserRoles(c.Request.Context(), userID)
		if err != nil {
			h.logger.Error(c.Request.Context(), "Error getting current user roles", err, nil)
			HandleAppError(c, contextutils.WrapError(err, "failed to get current user roles"))
			return
		}

		// Get all available roles
		allRoles, err := h.userService.GetAllRoles(c.Request.Context())
		if err != nil {
			h.logger.Error(c.Request.Context(), "Error getting all roles", err, nil)
			HandleAppError(c, contextutils.WrapError(err, "failed to get available roles"))
			return
		}

		// Create maps for efficient lookup
		currentRoleNames := make(map[string]bool)
		for _, role := range currentRoles {
			currentRoleNames[role.Name] = true
		}

		requestedRoleNames := make(map[string]bool)
		for _, roleName := range *req.SelectedRoles {
			requestedRoleNames[roleName] = true
		}

		// Find roles to add and remove
		for _, roleName := range *req.SelectedRoles {
			if !currentRoleNames[roleName] {
				// Find role by name
				var roleToAdd *models.Role
				for _, role := range allRoles {
					if role.Name == roleName {
						roleToAdd = &role
						break
					}
				}
				if roleToAdd != nil {
					err = h.userService.AssignRole(c.Request.Context(), userID, roleToAdd.ID)
					if err != nil {
						h.logger.Error(c.Request.Context(), "Error assigning role to user", err, map[string]interface{}{
							"user_id":   userID,
							"role_id":   roleToAdd.ID,
							"role_name": roleName,
						})
						HandleAppError(c, contextutils.WrapError(err, "failed to assign role"))
						return
					}
				}
			}
		}

		// Remove roles that are no longer selected
		for _, role := range currentRoles {
			if !requestedRoleNames[role.Name] {
				err = h.userService.RemoveRole(c.Request.Context(), userID, role.ID)
				if err != nil {
					h.logger.Error(c.Request.Context(), "Error removing role from user", err, map[string]interface{}{
						"user_id":   userID,
						"role_id":   role.ID,
						"role_name": role.Name,
					})
					HandleAppError(c, contextutils.WrapError(err, "failed to remove role"))
					return
				}
			}
		}
	}

	// Get updated user
	updatedUser, err := h.userService.GetUserByID(c.Request.Context(), userID)
	if err != nil {
		h.logger.Error(c.Request.Context(), "Error retrieving updated user", err, nil)
		HandleAppError(c, contextutils.WrapError(err, "failed to retrieve updated user"))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "User updated successfully",
		"user":    h.convertUserToProfileResponse(c.Request.Context(), updatedUser),
	})
}

// DeleteUser handles DELETE /userz/:id - delete user (admin only)
func (h *UserAdminHandler) DeleteUser(c *gin.Context) {
	userIDStr := c.Param("id")
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		HandleAppError(c, contextutils.ErrInvalidFormat)
		return
	}

	// Check if user exists
	user, err := h.userService.GetUserByID(c.Request.Context(), userID)
	if err != nil {
		h.logger.Error(c.Request.Context(), "Error retrieving user", err, nil)
		HandleAppError(c, contextutils.WrapError(err, "database error"))
		return
	}
	if user == nil {
		HandleAppError(c, contextutils.ErrRecordNotFound)
		return
	}

	// Delete user
	err = h.userService.DeleteUser(c.Request.Context(), userID)
	if err != nil {
		h.logger.Error(c.Request.Context(), "Error deleting user", err, nil)
		HandleAppError(c, contextutils.WrapError(err, "failed to delete user"))
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User deleted successfully"})
}

// ResetUserPassword handles POST /userz/:id/reset-password - reset user password (admin only)
func (h *UserAdminHandler) ResetUserPassword(c *gin.Context) {
	userIDStr := c.Param("id")
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		HandleAppError(c, contextutils.ErrInvalidFormat)
		return
	}

	// Check if user exists
	user, err := h.userService.GetUserByID(c.Request.Context(), userID)
	if err != nil {
		h.logger.Error(c.Request.Context(), "Error retrieving user", err, map[string]interface{}{"user_id": userID})
		HandleAppError(c, contextutils.WrapError(err, "database error"))
		return
	}
	if user == nil {
		h.logger.Warn(c.Request.Context(), "User not found for password reset", map[string]interface{}{"user_id": userID})
		HandleAppError(c, contextutils.ErrRecordNotFound)
		return
	}

	var req PasswordResetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error(c.Request.Context(), "Invalid request data for password reset", err, map[string]interface{}{"user_id": userID})
		HandleAppError(c, contextutils.NewAppErrorWithCause(
			contextutils.ErrorCodeInvalidInput,
			contextutils.SeverityWarn,
			"Invalid request data",
			"",
			err,
		))
		return
	}

	// Validate password
	if req.NewPassword == "" {
		HandleAppError(c, contextutils.ErrMissingRequired)
		return
	}

	// Update password
	err = h.userService.UpdateUserPassword(c.Request.Context(), userID, req.NewPassword)
	if err != nil {
		h.logger.Error(c.Request.Context(), "Error updating user password", err, map[string]interface{}{"user_id": userID})
		HandleAppError(c, contextutils.WrapError(err, "failed to update password"))
		return
	}

	h.logger.Info(c.Request.Context(), "Password reset successful", map[string]interface{}{"user_id": userID, "username": user.Username})
	c.JSON(http.StatusOK, gin.H{"message": "Password reset successfully"})
}

// UpdateCurrentUserProfile handles PUT /userz/profile - update current user profile
func (h *UserAdminHandler) UpdateCurrentUserProfile(c *gin.Context) {
	// Get user ID from context/session
	userID, err := GetCurrentUserID(c)
	if err != nil {
		HandleAppError(c, contextutils.ErrUnauthorized)
		return
	}

	var req UserUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		HandleAppError(c, contextutils.NewAppErrorWithCause(
			contextutils.ErrorCodeInvalidInput,
			contextutils.SeverityWarn,
			"Invalid request data",
			"",
			err,
		))
		return
	}

	// Validate timezone if provided
	if req.Timezone != nil && *req.Timezone != "" && !h.isValidTimezone(*req.Timezone) {
		HandleAppError(c, contextutils.ErrInvalidFormat)
		return
	}

	// Email validation is handled automatically by openapi_types.Email

	// Get current user
	user, err := h.userService.GetUserByID(c.Request.Context(), userID)
	if err != nil {
		h.logger.Error(c.Request.Context(), "Error retrieving user", err, nil)
		HandleAppError(c, contextutils.WrapError(err, "database error"))
		return
	}
	if user == nil {
		HandleAppError(c, contextutils.ErrRecordNotFound)
		return
	}

	// Check authorization (self-only for this endpoint)
	if err := RequireSelfOrAdmin(c.Request.Context(), h.userService, userID, userID); err != nil {
		if contextutils.IsError(err, contextutils.ErrForbidden) {
			HandleAppError(c, contextutils.ErrForbidden)
			return
		}
		h.logger.Error(c.Request.Context(), "Error checking authorization", err, nil)
		HandleAppError(c, contextutils.WrapError(err, "failed to check authorization"))
		return
	}

	// Use existing values if not provided in request
	username := user.Username
	if req.Username != nil && *req.Username != "" {
		username = *req.Username
	}

	email := ""
	if user.Email.Valid {
		email = user.Email.String
	}
	if req.Email != nil {
		email = string(*req.Email)
	}

	timezone := ""
	if user.Timezone.Valid {
		timezone = user.Timezone.String
	}
	if req.Timezone != nil && *req.Timezone != "" {
		timezone = *req.Timezone
	}

	// Check if new username already exists (if changed)
	if username != user.Username {
		existingUser, err := h.userService.GetUserByUsername(c.Request.Context(), username)
		if err != nil {
			h.logger.Error(c.Request.Context(), "Error checking existing username", err, nil)
			HandleAppError(c, contextutils.WrapError(err, "failed to check username uniqueness"))
			return
		}
		if existingUser != nil {
			HandleAppError(c, contextutils.ErrRecordExists)
			return
		}
	}

	// Check if new email already exists (if changed)
	if email != "" && user.Email.Valid && email != user.Email.String {
		existingUser, err := h.userService.GetUserByEmail(c.Request.Context(), email)
		if err != nil {
			h.logger.Error(c.Request.Context(), "Error checking existing email", err, nil)
			HandleAppError(c, contextutils.WrapError(err, "failed to check email uniqueness"))
			return
		}
		if existingUser != nil {
			HandleAppError(c, contextutils.ErrRecordExists)
			return
		}
	}

	// Use existing AI values if not provided in request
	preferredLanguage := ""
	if user.PreferredLanguage.Valid {
		preferredLanguage = user.PreferredLanguage.String
	}
	if req.PreferredLanguage != nil && *req.PreferredLanguage != "" {
		preferredLanguage = *req.PreferredLanguage
	}

	currentLevel := ""
	if user.CurrentLevel.Valid {
		currentLevel = user.CurrentLevel.String
	}
	if req.CurrentLevel != nil && *req.CurrentLevel != "" {
		currentLevel = *req.CurrentLevel
	}

	// Update user profile
	err = h.userService.UpdateUserProfile(c.Request.Context(), userID, username, email, timezone)
	if err != nil {
		h.logger.Error(c.Request.Context(), "Error updating user profile", err, nil)
		HandleAppError(c, contextutils.WrapError(err, "failed to update user profile"))
		return
	}

	// Handle AI settings update if provided
	needsAIUpdate := req.AiEnabled != nil || (req.AiProvider != nil && *req.AiProvider != "") || (req.AiModel != nil && *req.AiModel != "") || (req.PreferredLanguage != nil && *req.PreferredLanguage != "") || (req.CurrentLevel != nil && *req.CurrentLevel != "") || (req.ApiKey != nil && *req.ApiKey != "")

	if needsAIUpdate {
		aiSettings := &models.UserSettings{
			Language:  preferredLanguage,
			Level:     currentLevel,
			AIEnabled: req.AiEnabled != nil && *req.AiEnabled,
		}

		if req.AiProvider != nil && *req.AiProvider != "" {
			aiSettings.AIProvider = *req.AiProvider
		} else if user.AIProvider.Valid {
			aiSettings.AIProvider = user.AIProvider.String
		}

		if req.AiModel != nil && *req.AiModel != "" {
			aiSettings.AIModel = *req.AiModel
		} else if user.AIModel.Valid {
			aiSettings.AIModel = user.AIModel.String
		}

		if req.ApiKey != nil && *req.ApiKey != "" {
			aiSettings.AIAPIKey = *req.ApiKey
		}

		err = h.userService.UpdateUserSettings(c.Request.Context(), userID, aiSettings)
		if err != nil {
			h.logger.Error(c.Request.Context(), "Error updating user AI settings", err, nil)
			HandleAppError(c, contextutils.WrapError(err, "failed to update AI settings"))
			return
		}
	}

	// Get updated user
	updatedUser, err := h.userService.GetUserByID(c.Request.Context(), userID)
	if err != nil {
		h.logger.Error(c.Request.Context(), "Error retrieving updated user", err, nil)
		HandleAppError(c, contextutils.WrapError(err, "failed to retrieve updated profile"))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Profile updated successfully",
		"user":    h.convertUserToProfileResponse(c.Request.Context(), updatedUser),
	})
}

// isUserPaused checks if a user is paused by checking the worker_settings table
func (h *UserAdminHandler) isUserPaused(ctx context.Context, userID int) bool {
	query := `SELECT setting_value FROM worker_settings WHERE setting_key = $1`
	var value string
	settingKey := fmt.Sprintf("user_pause_%d", userID)

	err := h.userService.GetDB().QueryRowContext(ctx, query, settingKey).Scan(&value)
	if err != nil {
		// If no setting exists, user is not paused
		if errors.Is(err, sql.ErrNoRows) {
			return false
		}
		// Log error but don't fail - default to not paused
		h.logger.Warn(ctx, "Failed to check user pause status", map[string]interface{}{
			"user_id": userID,
			"error":   err.Error(),
		})
		return false
	}

	return value == "true"
}

// Helper functions

// convertUserToProfileResponse converts a User model to ProfileResponse
func (h *UserAdminHandler) convertUserToProfileResponse(ctx context.Context, user *models.User) ProfileResponse {
	// Get user roles
	roles, err := h.userService.GetUserRoles(ctx, user.ID)
	if err != nil {
		// Log error but don't fail the response
		h.logger.Warn(ctx, "Failed to get user roles", map[string]interface{}{
			"user_id": user.ID,
			"error":   err.Error(),
		})
		roles = []models.Role{}
	}

	return ProfileResponse{
		ID:                user.ID,
		Username:          user.Username,
		Email:             nullStringToPointer(user.Email),
		Timezone:          nullStringToPointer(user.Timezone),
		LastActive:        nullTimeToPointer(user.LastActive),
		PreferredLanguage: nullStringToPointer(user.PreferredLanguage),
		CurrentLevel:      nullStringToPointer(user.CurrentLevel),
		CreatedAt:         user.CreatedAt,
		UpdatedAt:         user.UpdatedAt,
		AIEnabled:         user.AIEnabled.Valid && user.AIEnabled.Bool,
		AIProvider:        nullStringToPointer(user.AIProvider),
		AIModel:           nullStringToPointer(user.AIModel),
		Roles:             roles,
		IsPaused:          h.isUserPaused(ctx, user.ID),
	}
}

// isValidTimezone checks if a timezone string is valid
func (h *UserAdminHandler) isValidTimezone(tz string) bool {
	// Common timezone validation - check if it can be loaded
	_, err := time.LoadLocation(tz)
	if err != nil {
		// Also allow UTC as fallback
		return strings.ToUpper(tz) == "UTC"
	}
	return true
}

// Helper function to convert sql.NullString to *string (if not already available)
func nullStringToPointer(ns sql.NullString) *string {
	if ns.Valid {
		return &ns.String
	}
	return nil
}

// Helper function to convert sql.NullTime to *time.Time (if not already available)
func nullTimeToPointer(nt sql.NullTime) *time.Time {
	if nt.Valid {
		return &nt.Time
	}
	return nil
}
