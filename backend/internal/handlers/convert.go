package handlers

import (
	"context"
	"encoding/json"
	"time"

	"quizapp/internal/api"
	"quizapp/internal/models"
	"quizapp/internal/services"
	contextutils "quizapp/internal/utils"

	openapi_types "github.com/oapi-codegen/runtime/types"
)

// Helper functions for pointer conversion
func stringPtr(s string) *string {
	return &s
}

func boolPtr(b bool) *bool {
	return &b
}

func int64Ptr(i int) *int64 {
	i64 := int64(i)
	return &i64
}

func float32Ptr(f float32) *float32 {
	return &f
}

func intPtr(i int) *int {
	return &i
}

func int64FromUint(u uint) *int64 {
	i64 := int64(u)
	return &i64
}

func timePtr(t time.Time) *time.Time {
	return &t
}

// formatTimePtr formats a time.Time into an RFC3339 string pointer
func formatTimePtr(t time.Time) *string {
	s := t.In(time.UTC).Format(time.RFC3339)
	return &s
}

// formatTimePointer converts a *time.Time to *string (RFC3339) or nil
func formatTimePointer(tp *time.Time) *string {
	if tp == nil {
		return nil
	}
	s := tp.In(time.UTC).Format(time.RFC3339)
	return &s
}

// formatTime formats a time.Time into an RFC3339 string
func formatTime(t time.Time) string {
	return t.In(time.UTC).Format(time.RFC3339)
}

// Convert models.AuthAPIKey to api.APIKeySummary
func convertAuthAPIKeyToAPI(key *models.AuthAPIKey) api.APIKeySummary {
	apiKey := api.APIKeySummary{}

	// Scalars
	if key.ID != 0 {
		apiKey.Id = intPtr(key.ID)
	}
	if key.KeyName != "" {
		apiKey.KeyName = stringPtr(key.KeyName)
	}
	if key.KeyPrefix != "" {
		apiKey.KeyPrefix = stringPtr(key.KeyPrefix)
	}
	if key.PermissionLevel != "" {
		pl := api.APIKeySummaryPermissionLevel(key.PermissionLevel)
		apiKey.PermissionLevel = &pl
	}

	// Times
	if !key.CreatedAt.IsZero() {
		t := key.CreatedAt
		apiKey.CreatedAt = &t
	}
	if !key.UpdatedAt.IsZero() {
		t := key.UpdatedAt
		apiKey.UpdatedAt = &t
	}
	if key.LastUsedAt.Valid {
		t := key.LastUsedAt.Time
		apiKey.LastUsedAt = &t
	} else {
		// Leave nil to represent null
		apiKey.LastUsedAt = nil
	}

	return apiKey
}

// Convert slice of models.AuthAPIKey to []api.APIKeySummary
func convertAuthAPIKeysToAPI(keys []models.AuthAPIKey) []api.APIKeySummary {
	if len(keys) == 0 {
		return []api.APIKeySummary{}
	}
	out := make([]api.APIKeySummary, 0, len(keys))
	for i := range keys {
		out = append(out, convertAuthAPIKeyToAPI(&keys[i]))
	}
	return out
}

// Convert models.User to api.User
func convertUserToAPI(user *models.User) api.User {
	apiUser := api.User{
		Id:       int64Ptr(user.ID),
		Username: stringPtr(user.Username),
	}

	if !user.CreatedAt.IsZero() {
		apiUser.CreatedAt = formatTimePtr(user.CreatedAt)
	}

	if user.LastActive.Valid {
		apiUser.LastActive = formatTimePointer(&user.LastActive.Time)
	}

	if user.Email.Valid {
		s := user.Email.String
		apiUser.Email = &s
	}

	if user.Timezone.Valid {
		s := user.Timezone.String
		apiUser.Timezone = &s
	}

	if user.PreferredLanguage.Valid {
		s := user.PreferredLanguage.String
		apiUser.PreferredLanguage = &s
	}

	if user.CurrentLevel.Valid {
		s := user.CurrentLevel.String
		apiUser.CurrentLevel = &s
	}

	if user.AIProvider.Valid {
		s := user.AIProvider.String
		apiUser.AiProvider = &s
	}

	if user.AIModel.Valid {
		s := user.AIModel.String
		apiUser.AiModel = &s
	}

	// Always set ai_enabled as a boolean (never null)
	aiEnabled := user.AIEnabled.Valid && user.AIEnabled.Bool
	apiUser.AiEnabled = &aiEnabled

	// For backwards compatibility, we'll set has_api_key to false here
	// The proper check should be done using convertUserToAPIWithService
	hasAPIKey := false
	apiUser.HasApiKey = &hasAPIKey

	// Include user roles if they exist
	if len(user.Roles) > 0 {
		apiRoles := make([]api.Role, len(user.Roles))
		for i, role := range user.Roles {
			apiRoles[i] = api.Role{
				Id:          int64(role.ID),
				Name:        role.Name,
				Description: role.Description,
				CreatedAt:   formatTime(role.CreatedAt),
				UpdatedAt:   formatTime(role.UpdatedAt),
			}
		}
		apiUser.Roles = &apiRoles
	}

	return apiUser
}

// convertUserToAPIWithService converts a models.User to api.User with proper API key checking
func convertUserToAPIWithService(ctx context.Context, user *models.User, userService services.UserServiceInterface) api.User {
	apiUser := convertUserToAPI(user)

	// Check if user has a valid API key for their current provider using the new table
	hasAPIKey := false
	if user.AIProvider.Valid && user.AIProvider.String != "" {
		// Use the new per-provider API key system instead of the old user.AIAPIKey field
		if userService != nil {
			savedKey, err := userService.GetUserAPIKey(ctx, user.ID, user.AIProvider.String)
			if err == nil && savedKey != "" {
				// API key is available but not exposed in the API response for security
				hasAPIKey = true
			}
		}
	}
	// If user doesn't have an AI provider set, hasAPIKey remains false (default)
	apiUser.HasApiKey = &hasAPIKey

	return apiUser
}

// Convert models.Question to api.Question
func convertQuestionToAPI(question *models.Question) api.Question {
	apiQuestion := api.Question{
		Id:              int64Ptr(question.ID),
		DifficultyScore: float32Ptr(float32(question.DifficultyScore)),
		CorrectAnswer:   intPtr(question.CorrectAnswer),
		// UsageCount removed; use total_responses instead
	}

	if !question.CreatedAt.IsZero() {
		v := formatTime(question.CreatedAt)
		apiQuestion.CreatedAt = &v
	}

	if question.Type != "" {
		qType := api.QuestionType(question.Type)
		apiQuestion.Type = &qType
	}

	if question.Language != "" {
		lang := api.Language(question.Language)
		apiQuestion.Language = &lang
	}

	if question.Level != "" {
		level := api.Level(question.Level)
		apiQuestion.Level = &level
	}

	if question.Explanation != "" {
		apiQuestion.Explanation = &question.Explanation
	}

	if question.Status != "" {
		status := api.QuestionStatus(question.Status)
		apiQuestion.Status = &status
	}

	// Convert content map to api.QuestionContent
	if question.Content != nil {
		content := &api.QuestionContent{}

		if q, ok := question.Content["question"].(string); ok {
			content.Question = q
		}
		if hint, ok := question.Content["hint"].(string); ok {
			content.Hint = &hint
		}
		if passage, ok := question.Content["passage"].(string); ok {
			content.Passage = &passage
		}
		if sentence, ok := question.Content["sentence"].(string); ok {
			content.Sentence = &sentence
		}
		if opts, ok := question.Content["options"].([]interface{}); ok {
			var options []string
			for _, opt := range opts {
				if o, ok := opt.(string); ok {
					options = append(options, o)
				}
			}
			if len(options) > 0 {
				content.Options = options
			}
		}
		apiQuestion.Content = content
	}

	// Add variety elements to the API response
	if question.TopicCategory != "" {
		apiQuestion.TopicCategory = &question.TopicCategory
	}
	if question.GrammarFocus != "" {
		apiQuestion.GrammarFocus = &question.GrammarFocus
	}
	if question.VocabularyDomain != "" {
		apiQuestion.VocabularyDomain = &question.VocabularyDomain
	}
	if question.Scenario != "" {
		apiQuestion.Scenario = &question.Scenario
	}
	if question.StyleModifier != "" {
		apiQuestion.StyleModifier = &question.StyleModifier
	}
	if question.DifficultyModifier != "" {
		apiQuestion.DifficultyModifier = &question.DifficultyModifier
	}
	if question.TimeContext != "" {
		apiQuestion.TimeContext = &question.TimeContext
	}

	return apiQuestion
}

// Convert services.QuestionWithStats to a JSON-compatible map using generated
// api.Question for fields, and include any additional fields the frontend
// expects (e.g., report_reasons) that are not present on the generated type.
func convertQuestionWithStatsToAPIMap(q *services.QuestionWithStats) map[string]interface{} {
	apiQ := api.Question{}
	if q != nil && q.Question != nil {
		apiQ = convertQuestionToAPI(q.Question)
	}

	// Attach stats
	if q != nil {
		apiQ.CorrectCount = intPtr(q.CorrectCount)
		apiQ.IncorrectCount = intPtr(q.IncorrectCount)
		apiQ.TotalResponses = intPtr(q.TotalResponses)
		apiQ.UserCount = intPtr(q.UserCount)
		if q.Reporters != "" {
			apiQ.Reporters = &q.Reporters
		}
		// ConfidenceLevel is optional
		if q.ConfidenceLevel != nil {
			apiQ.ConfidenceLevel = q.ConfidenceLevel
		}
	}

	// Marshal to generic map so we can add fields not present in api.Question
	m := map[string]interface{}{}
	if b, err := json.Marshal(apiQ); err == nil {
		_ = json.Unmarshal(b, &m)
	}

	// Add report_reasons if available on the service struct
	if q != nil && q.ReportReasons != "" {
		m["report_reasons"] = q.ReportReasons
	}

	return m
}

// Convert models.UserProgress to api.UserProgress
func convertUserProgressToAPI(ctx context.Context, progress *models.UserProgress, userID int, userLookup func(context.Context, int) (*models.User, error)) api.UserProgress {
	apiProgress := api.UserProgress{
		TotalQuestions: intPtr(progress.TotalQuestions),
		CorrectAnswers: intPtr(progress.CorrectAnswers),
		AccuracyRate:   float32Ptr(float32(progress.AccuracyRate / 100.0)),
	}

	if progress.CurrentLevel != "" {
		level := api.Level(progress.CurrentLevel)
		apiProgress.CurrentLevel = &level
	}

	if progress.SuggestedLevel != "" {
		level := api.Level(progress.SuggestedLevel)
		apiProgress.SuggestedLevel = &level
	}

	if progress.WeakAreas != nil {
		apiProgress.WeakAreas = &progress.WeakAreas
	}

	// Convert performance metrics
	if progress.PerformanceByTopic != nil {
		perfMap := make(map[string]api.PerformanceMetrics)
		for topic, metrics := range progress.PerformanceByTopic {
			if metrics != nil {
				perfMap[topic] = api.PerformanceMetrics{
					TotalAttempts:         intPtr(metrics.TotalAttempts),
					CorrectAttempts:       intPtr(metrics.CorrectAttempts),
					AverageResponseTimeMs: float32Ptr(float32(metrics.AverageResponseTimeMs)),
					LastUpdated: func() *string {
						if metrics.LastUpdated.IsZero() {
							return nil
						}
						s, _, err := contextutils.FormatTimeInUserTimezone(ctx, userID, metrics.LastUpdated, time.RFC3339, userLookup)
						if err != nil || s == "" {
							tmp := metrics.LastUpdated.In(time.UTC).Format(time.RFC3339)
							return &tmp
						}
						return &s
					}(),
				}
			}
		}
		apiProgress.PerformanceByTopic = &perfMap
	}

	// Convert recent activity
	if progress.RecentActivity != nil {
		var recentActivity []api.UserResponse
		for _, activity := range progress.RecentActivity {
			apiActivity := api.UserResponse{
				QuestionId: int64Ptr(activity.QuestionID),
				IsCorrect:  &activity.IsCorrect,
			}
			if !activity.CreatedAt.IsZero() {
				s, _, err := contextutils.FormatTimeInUserTimezone(ctx, userID, activity.CreatedAt, time.RFC3339, userLookup)
				if err != nil || s == "" {
					tmp := activity.CreatedAt.In(time.UTC).Format(time.RFC3339)
					apiActivity.CreatedAt = &tmp
				} else {
					apiActivity.CreatedAt = &s
				}
			}
			recentActivity = append(recentActivity, apiActivity)
		}
		apiProgress.RecentActivity = &recentActivity
	}

	return apiProgress
}

// Convert models.DailyQuestionAssignmentWithQuestion to api.DailyQuestionWithDetails
func convertDailyAssignmentToAPI(ctx context.Context, assignment *models.DailyQuestionAssignmentWithQuestion, userID int, userLookup func(context.Context, int) (*models.User, error)) api.DailyQuestionWithDetails {
	var completedAt *string
	if assignment.CompletedAt.Valid {
		if s, _, err := contextutils.FormatTimeInUserTimezone(ctx, userID, assignment.CompletedAt.Time, time.RFC3339, userLookup); err == nil && s != "" {
			completedAt = &s
		} else {
			tmp := assignment.CompletedAt.Time.In(time.UTC).Format(time.RFC3339)
			completedAt = &tmp
		}
	}

	apiQuestion := api.Question{}
	if assignment.Question != nil {
		apiQuestion = convertQuestionToAPI(assignment.Question)
		// Override total_responses so UI 'Shown' reflects Daily-only impressions
		if assignment.DailyShownCount > 0 {
			apiQuestion.TotalResponses = &assignment.DailyShownCount
		}
	}

	// AssignmentDate: produce date-only value (YYYY-MM-DD) using openapi_types.Date
	ad := assignment.AssignmentDate
	assignDate := openapi_types.Date{Time: ad}

	// CreatedAt in user's timezone (with error-checked fallback)
	var createdStr string
	if s, _, err := contextutils.FormatTimeInUserTimezone(ctx, userID, assignment.CreatedAt, time.RFC3339, userLookup); err == nil && s != "" {
		createdStr = s
	} else {
		createdStr = assignment.CreatedAt.In(time.UTC).Format(time.RFC3339)
	}

	var submittedAt *string
	if assignment.SubmittedAt != nil {
		if s, _, err := contextutils.FormatTimeInUserTimezone(ctx, userID, *assignment.SubmittedAt, time.RFC3339, userLookup); err == nil && s != "" {
			submittedAt = &s
		} else {
			tmp := assignment.SubmittedAt.In(time.UTC).Format(time.RFC3339)
			submittedAt = &tmp
		}
	}

	result := api.DailyQuestionWithDetails{
		Id:              int64(assignment.ID),
		UserId:          int64(assignment.UserID),
		QuestionId:      int64(assignment.QuestionID),
		AssignmentDate:  assignDate,
		IsCompleted:     assignment.IsCompleted,
		CompletedAt:     completedAt,
		CreatedAt:       createdStr,
		UserAnswerIndex: assignment.UserAnswerIndex,
		SubmittedAt:     submittedAt,
		Question:        apiQuestion,
	}

	// Attach per-user stats when available
	if assignment.DailyShownCount >= 0 {
		shown := int64(assignment.DailyShownCount)
		result.UserShownCount = &shown
	}
	if assignment.UserTotalResponses >= 0 {
		total := int64(assignment.UserTotalResponses)
		result.UserTotalResponses = &total
	}
	if assignment.UserCorrectCount >= 0 {
		cc := int64(assignment.UserCorrectCount)
		result.UserCorrectCount = &cc
	}
	if assignment.UserIncorrectCount >= 0 {
		ic := int64(assignment.UserIncorrectCount)
		result.UserIncorrectCount = &ic
	}

	return result
}

// Convert slice of assignments
func convertDailyAssignmentsToAPI(ctx context.Context, assignments []*models.DailyQuestionAssignmentWithQuestion, userID int, userLookup func(context.Context, int) (*models.User, error)) []api.DailyQuestionWithDetails {
	if len(assignments) == 0 {
		return []api.DailyQuestionWithDetails{}
	}
	apiAssignments := make([]api.DailyQuestionWithDetails, len(assignments))
	for i, a := range assignments {
		apiAssignments[i] = convertDailyAssignmentToAPI(ctx, a, userID, userLookup)
	}
	return apiAssignments
}

// Convert models.DailyProgress to api.DailyProgress
func convertDailyProgressToAPI(progress *models.DailyProgress) api.DailyProgress {
	return api.DailyProgress{
		Date:      openapi_types.Date{Time: progress.Date},
		Completed: progress.Completed,
		Total:     progress.Total,
	}
}

// Convert models.Story to api.Story
func convertStoryToAPI(story *models.Story) api.Story {
	apiStory := api.Story{
		Id:       int64FromUint(story.ID),
		UserId:   int64FromUint(story.UserID),
		Title:    stringPtr(story.Title),
		Language: stringPtr(story.Language),
		Status:   (*api.StoryStatus)(stringPtr(string(story.Status))),
	}

	if story.Subject != nil {
		apiStory.Subject = story.Subject
	}
	if story.AuthorStyle != nil {
		apiStory.AuthorStyle = story.AuthorStyle
	}
	if story.TimePeriod != nil {
		apiStory.TimePeriod = story.TimePeriod
	}
	if story.Genre != nil {
		apiStory.Genre = story.Genre
	}
	if story.Tone != nil {
		apiStory.Tone = story.Tone
	}
	if story.CharacterNames != nil {
		apiStory.CharacterNames = story.CharacterNames
	}
	if story.CustomInstructions != nil {
		apiStory.CustomInstructions = story.CustomInstructions
	}
	// Handle enum field - only set if not nil (will be omitted from JSON due to omitempty)
	if story.SectionLengthOverride != nil {
		lengthOverride := api.StorySectionLengthOverride(*story.SectionLengthOverride)
		apiStory.SectionLengthOverride = &lengthOverride
	}

	if !story.CreatedAt.IsZero() {
		apiStory.CreatedAt = timePtr(story.CreatedAt)
	}
	if !story.UpdatedAt.IsZero() {
		apiStory.UpdatedAt = timePtr(story.UpdatedAt)
	}
	if story.LastSectionGeneratedAt != nil {
		apiStory.LastSectionGeneratedAt = timePtr(*story.LastSectionGeneratedAt)
	}

	return apiStory
}

// Convert models.StorySection to api.StorySection
func convertStorySectionToAPI(section *models.StorySection) api.StorySection {
	apiSection := api.StorySection{
		Id:            int64FromUint(section.ID),
		StoryId:       int64FromUint(section.StoryID),
		SectionNumber: intPtr(section.SectionNumber),
		Content:       stringPtr(section.Content),
		LanguageLevel: stringPtr(section.LanguageLevel),
		WordCount:     intPtr(section.WordCount),
	}

	if !section.GeneratedAt.IsZero() {
		apiSection.GeneratedAt = timePtr(section.GeneratedAt)
	}

	// Convert time.Time to openapi_types.Date for generation_date
	if !section.GenerationDate.IsZero() {
		generationDate := openapi_types.Date{Time: section.GenerationDate}
		apiSection.GenerationDate = &generationDate
	}

	return apiSection
}

// Convert models.StoryWithSections to api.StoryWithSections
func convertStoryWithSectionsToAPI(story *models.StoryWithSections) api.StoryWithSections {
	apiStory := api.StoryWithSections{
		Id:                   int64FromUint(story.ID),
		UserId:               int64FromUint(story.UserID),
		Title:                stringPtr(story.Title),
		Language:             stringPtr(story.Language),
		Status:               (*api.StoryWithSectionsStatus)(stringPtr(string(story.Status))),
		AutoGenerationPaused: boolPtr(story.AutoGenerationPaused),
	}

	if story.Subject != nil {
		apiStory.Subject = story.Subject
	}
	if story.AuthorStyle != nil {
		apiStory.AuthorStyle = story.AuthorStyle
	}
	if story.TimePeriod != nil {
		apiStory.TimePeriod = story.TimePeriod
	}
	if story.Genre != nil {
		apiStory.Genre = story.Genre
	}
	if story.Tone != nil {
		apiStory.Tone = story.Tone
	}
	if story.CharacterNames != nil {
		apiStory.CharacterNames = story.CharacterNames
	}
	if story.CustomInstructions != nil {
		apiStory.CustomInstructions = story.CustomInstructions
	}
	// Handle enum field - only set if not nil (will be omitted from JSON due to omitempty)
	if story.SectionLengthOverride != nil {
		lengthOverride := api.StoryWithSectionsSectionLengthOverride(*story.SectionLengthOverride)
		apiStory.SectionLengthOverride = &lengthOverride
	}

	if !story.CreatedAt.IsZero() {
		apiStory.CreatedAt = timePtr(story.CreatedAt)
	}
	if !story.UpdatedAt.IsZero() {
		apiStory.UpdatedAt = timePtr(story.UpdatedAt)
	}
	if story.LastSectionGeneratedAt != nil {
		apiStory.LastSectionGeneratedAt = timePtr(*story.LastSectionGeneratedAt)
	}

	// Convert sections using the section conversion function
	if len(story.Sections) > 0 {
		apiSections := make([]api.StorySection, len(story.Sections))
		for i, section := range story.Sections {
			apiSections[i] = convertStorySectionToAPI(&section)
		}
		apiStory.Sections = &apiSections
	}

	return apiStory
}

// Convert models.StorySectionWithQuestions to api.StorySectionWithQuestions
func convertStorySectionWithQuestionsToAPI(sectionWithQuestions *models.StorySectionWithQuestions) api.StorySectionWithQuestions {
	apiSectionWithQuestions := api.StorySectionWithQuestions{
		Id:            int64FromUint(sectionWithQuestions.ID),
		StoryId:       int64FromUint(sectionWithQuestions.StoryID),
		SectionNumber: intPtr(sectionWithQuestions.SectionNumber),
		Content:       stringPtr(sectionWithQuestions.Content),
		LanguageLevel: stringPtr(sectionWithQuestions.LanguageLevel),
		WordCount:     intPtr(sectionWithQuestions.WordCount),
	}

	if !sectionWithQuestions.GeneratedAt.IsZero() {
		apiSectionWithQuestions.GeneratedAt = timePtr(sectionWithQuestions.GeneratedAt)
	}

	// Convert time.Time to openapi_types.Date for generation_date
	if !sectionWithQuestions.GenerationDate.IsZero() {
		generationDate := openapi_types.Date{Time: sectionWithQuestions.GenerationDate}
		apiSectionWithQuestions.GenerationDate = &generationDate
	}

	// Convert questions
	if len(sectionWithQuestions.Questions) > 0 {
		apiQuestions := make([]api.StorySectionQuestion, len(sectionWithQuestions.Questions))
		for i, question := range sectionWithQuestions.Questions {
			apiQuestions[i] = api.StorySectionQuestion{
				Id:                 int64FromUint(question.ID),
				SectionId:          int64FromUint(question.SectionID),
				QuestionText:       stringPtr(question.QuestionText),
				Options:            &question.Options,
				CorrectAnswerIndex: intPtr(question.CorrectAnswerIndex),
				CreatedAt:          timePtr(question.CreatedAt),
			}
			if question.Explanation != nil {
				apiQuestions[i].Explanation = question.Explanation
			}
		}
		apiSectionWithQuestions.Questions = &apiQuestions
	}

	return apiSectionWithQuestions
}
