package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"quizapp/internal/config"
	"quizapp/internal/observability"
	contextutils "quizapp/internal/utils"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// uuidRegex matches standard UUID format (8-4-4-4-12 hex digits)
var uuidRegex = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

// Linear API constants
const (
	// LinearAPIEndpoint is the base URL for Linear's GraphQL API
	LinearAPIEndpoint = "https://api.linear.app/graphql"
	// LinearHTTPTimeout is the timeout for Linear API requests
	LinearHTTPTimeout = 30 * time.Second
)

// LinearService handles Linear API integration
type LinearService struct {
	config     *config.Config
	httpClient *http.Client
	logger     *observability.Logger
	apiURL     string // Allow overriding API endpoint for testing
}

// LinearIssueResponse represents the response from Linear API
type LinearIssueResponse struct {
	Data struct {
		IssueCreate struct {
			Success bool `json:"success"`
			Issue   struct {
				ID    string `json:"id"`
				Title string `json:"title"`
				URL   string `json:"url"`
			} `json:"issue"`
		} `json:"issueCreate"`
	} `json:"data"`
	Errors []struct {
		Message    string                 `json:"message"`
		Extensions map[string]interface{} `json:"extensions,omitempty"`
		Path       []interface{}          `json:"path,omitempty"`
	} `json:"errors,omitempty"`
}

// LinearIssueResult represents the result of creating a Linear issue
type LinearIssueResult struct {
	IssueID  string `json:"issue_id"`
	IssueURL string `json:"issue_url"`
	Title    string `json:"title"`
}

// NewLinearService creates a new Linear service instance
func NewLinearService(cfg *config.Config, logger *observability.Logger) *LinearService {
	return &LinearService{
		config: cfg,
		httpClient: &http.Client{
			Timeout: LinearHTTPTimeout,
			Transport: otelhttp.NewTransport(http.DefaultTransport,
				otelhttp.WithSpanOptions(trace.WithSpanKind(trace.SpanKindClient)),
			),
		},
		logger: logger,
		apiURL: LinearAPIEndpoint,
	}
}

// NewLinearServiceWithURL creates a new LinearService instance with a custom API URL (for testing)
func NewLinearServiceWithURL(cfg *config.Config, logger *observability.Logger, apiURL string) *LinearService {
	return &LinearService{
		config: cfg,
		httpClient: &http.Client{
			Timeout: LinearHTTPTimeout,
			Transport: otelhttp.NewTransport(http.DefaultTransport,
				otelhttp.WithSpanOptions(trace.WithSpanKind(trace.SpanKindClient)),
			),
		},
		logger: logger,
		apiURL: apiURL,
	}
}

// getTeamIDByName looks up a team ID by name, or returns the ID if it's already a UUID
func (s *LinearService) getTeamIDByName(ctx context.Context, teamIdentifier string) (string, error) {
	// If it looks like a UUID, return it as-is (case-insensitive check)
	if uuidRegex.MatchString(strings.ToLower(teamIdentifier)) {
		return teamIdentifier, nil
	}

	// Otherwise, query Linear for teams
	query := `
		query Teams {
			teams {
				nodes {
					id
					name
				}
			}
		}
	`

	requestBody := map[string]interface{}{
		"query": query,
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return "", contextutils.WrapError(err, "failed to marshal team lookup request")
	}

	apiURL := s.apiURL
	if apiURL == "" {
		apiURL = LinearAPIEndpoint
	}
	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", contextutils.WrapError(err, "failed to create team lookup request")
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", s.config.Linear.APIKey)
	req.Header.Set("User-Agent", "quizapp/1.0")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", contextutils.WrapErrorf(err, "failed to query Linear teams")
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			s.logger.Warn(ctx, "Failed to close response body", map[string]interface{}{"error": closeErr.Error()})
		}
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", contextutils.WrapError(err, "failed to read team lookup response")
	}

	if resp.StatusCode != http.StatusOK {
		return "", contextutils.NewAppError(
			contextutils.ErrorCodeServiceUnavailable,
			contextutils.SeverityError,
			fmt.Sprintf("Linear API returned status %d when looking up teams: %s", resp.StatusCode, string(body)),
			"",
		)
	}

	var teamResponse struct {
		Data struct {
			Teams struct {
				Nodes []struct {
					ID   string `json:"id"`
					Name string `json:"name"`
				} `json:"nodes"`
			} `json:"teams"`
		} `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors,omitempty"`
	}

	if err := json.Unmarshal(body, &teamResponse); err != nil {
		return "", contextutils.WrapError(err, "failed to unmarshal team lookup response")
	}

	if len(teamResponse.Errors) > 0 {
		return "", contextutils.NewAppError(
			contextutils.ErrorCodeServiceUnavailable,
			contextutils.SeverityError,
			fmt.Sprintf("Linear API error when looking up teams: %s", teamResponse.Errors[0].Message),
			"",
		)
	}

	// Find team by name (case-insensitive)
	for _, team := range teamResponse.Data.Teams.Nodes {
		if strings.EqualFold(team.Name, teamIdentifier) {
			return team.ID, nil
		}
	}

	return "", contextutils.NewAppError(
		contextutils.ErrorCodeInvalidInput,
		contextutils.SeverityError,
		fmt.Sprintf("Team '%s' not found in Linear", teamIdentifier),
		"",
	)
}

// getProjectIDByName looks up a project ID by name within a team, or returns the ID if it's already a UUID
func (s *LinearService) getProjectIDByName(ctx context.Context, projectIdentifier, teamID string) (string, error) {
	// If it looks like a UUID, return it as-is (case-insensitive check)
	if uuidRegex.MatchString(strings.ToLower(projectIdentifier)) {
		return projectIdentifier, nil
	}

	// Otherwise, query Linear for projects in the team
	query := `
		query Projects($teamId: String!) {
			team(id: $teamId) {
				projects {
					nodes {
						id
						name
					}
				}
			}
		}
	`

	variables := map[string]interface{}{
		"teamId": teamID,
	}

	requestBody := map[string]interface{}{
		"query":     query,
		"variables": variables,
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return "", contextutils.WrapError(err, "failed to marshal project lookup request")
	}

	apiURL := s.apiURL
	if apiURL == "" {
		apiURL = LinearAPIEndpoint
	}
	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", contextutils.WrapError(err, "failed to create project lookup request")
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", s.config.Linear.APIKey)
	req.Header.Set("User-Agent", "quizapp/1.0")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", contextutils.WrapErrorf(err, "failed to query Linear projects")
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			s.logger.Warn(ctx, "Failed to close response body", map[string]interface{}{"error": closeErr.Error()})
		}
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", contextutils.WrapError(err, "failed to read project lookup response")
	}

	if resp.StatusCode != http.StatusOK {
		return "", contextutils.NewAppError(
			contextutils.ErrorCodeServiceUnavailable,
			contextutils.SeverityError,
			fmt.Sprintf("Linear API returned status %d when looking up projects: %s", resp.StatusCode, string(body)),
			"",
		)
	}

	var projectResponse struct {
		Data struct {
			Team struct {
				Projects struct {
					Nodes []struct {
						ID   string `json:"id"`
						Name string `json:"name"`
					} `json:"nodes"`
				} `json:"projects"`
			} `json:"team"`
		} `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors,omitempty"`
	}

	if err := json.Unmarshal(body, &projectResponse); err != nil {
		return "", contextutils.WrapError(err, "failed to unmarshal project lookup response")
	}

	if len(projectResponse.Errors) > 0 {
		return "", contextutils.NewAppError(
			contextutils.ErrorCodeServiceUnavailable,
			contextutils.SeverityError,
			fmt.Sprintf("Linear API error when looking up projects: %s", projectResponse.Errors[0].Message),
			"",
		)
	}

	// Find project by name (case-insensitive)
	for _, project := range projectResponse.Data.Team.Projects.Nodes {
		if strings.EqualFold(project.Name, projectIdentifier) {
			return project.ID, nil
		}
	}

	return "", contextutils.NewAppError(
		contextutils.ErrorCodeInvalidInput,
		contextutils.SeverityError,
		fmt.Sprintf("Project '%s' not found in team", projectIdentifier),
		"",
	)
}

// getLabelIDByName looks up a label ID by name, or returns the ID if it's already a UUID
func (s *LinearService) getLabelIDByName(ctx context.Context, labelIdentifier string) (string, error) {
	// If it looks like a UUID, return it as-is
	if len(labelIdentifier) == 36 && strings.Contains(labelIdentifier, "-") {
		return labelIdentifier, nil
	}

	// Query Linear for both organization and team labels
	// First try organization-level labels (workspace-wide)
	query := `
		query Labels {
			organization {
				labels {
					nodes {
						id
						name
					}
				}
			}
		}
	`

	requestBody := map[string]interface{}{
		"query": query,
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return "", contextutils.WrapError(err, "failed to marshal label lookup request")
	}

	apiURL := s.apiURL
	if apiURL == "" {
		apiURL = LinearAPIEndpoint
	}
	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", contextutils.WrapError(err, "failed to create label lookup request")
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", s.config.Linear.APIKey)
	req.Header.Set("User-Agent", "quizapp/1.0")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", contextutils.WrapErrorf(err, "failed to query Linear labels")
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			s.logger.Warn(ctx, "Failed to close response body", map[string]interface{}{"error": closeErr.Error()})
		}
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", contextutils.WrapError(err, "failed to read label lookup response")
	}

	if resp.StatusCode != http.StatusOK {
		return "", contextutils.NewAppError(
			contextutils.ErrorCodeServiceUnavailable,
			contextutils.SeverityError,
			fmt.Sprintf("Linear API returned status %d when looking up labels: %s", resp.StatusCode, string(body)),
			"",
		)
	}

	var labelResponse struct {
		Data struct {
			Organization struct {
				Labels struct {
					Nodes []struct {
						ID   string `json:"id"`
						Name string `json:"name"`
					} `json:"nodes"`
				} `json:"labels"`
			} `json:"organization"`
		} `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors,omitempty"`
	}

	if err := json.Unmarshal(body, &labelResponse); err != nil {
		return "", contextutils.WrapError(err, "failed to unmarshal label lookup response")
	}

	if len(labelResponse.Errors) > 0 {
		return "", contextutils.NewAppError(
			contextutils.ErrorCodeServiceUnavailable,
			contextutils.SeverityError,
			fmt.Sprintf("Linear API error when looking up labels: %s", labelResponse.Errors[0].Message),
			"",
		)
	}

	// Find label by name (case-insensitive) in organization labels
	for _, label := range labelResponse.Data.Organization.Labels.Nodes {
		if strings.EqualFold(label.Name, labelIdentifier) {
			return label.ID, nil
		}
	}

	// If not found in organization labels, try team-specific labels
	// Note: We need the team ID to query team labels, but we don't have it here
	// For now, we'll return an error. In the future, we could pass teamID to this function
	// or query team labels separately in CreateIssue after we have the team ID

	return "", contextutils.NewAppError(
		contextutils.ErrorCodeInvalidInput,
		contextutils.SeverityError,
		fmt.Sprintf("Label '%s' not found in Linear workspace. Make sure the label exists at the workspace level (Settings > Workspace > Labels)", labelIdentifier),
		"",
	)
}

// getTeamLabelIDByName looks up a team-specific label ID by name
func (s *LinearService) getTeamLabelIDByName(ctx context.Context, teamID, labelIdentifier string) (string, error) {
	// Query Linear for team-specific labels
	query := `
		query TeamLabels($teamId: String!) {
			team(id: $teamId) {
				labels {
					nodes {
						id
						name
					}
				}
			}
		}
	`

	requestBody := map[string]interface{}{
		"query": query,
		"variables": map[string]interface{}{
			"teamId": teamID,
		},
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return "", contextutils.WrapError(err, "failed to marshal team label lookup request")
	}

	apiURL := s.apiURL
	if apiURL == "" {
		apiURL = LinearAPIEndpoint
	}
	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", contextutils.WrapError(err, "failed to create team label lookup request")
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", s.config.Linear.APIKey)
	req.Header.Set("User-Agent", "quizapp/1.0")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", contextutils.WrapErrorf(err, "failed to query Linear team labels")
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			s.logger.Warn(ctx, "Failed to close response body", map[string]interface{}{"error": closeErr.Error()})
		}
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", contextutils.WrapError(err, "failed to read team label lookup response")
	}

	if resp.StatusCode != http.StatusOK {
		return "", contextutils.NewAppError(
			contextutils.ErrorCodeServiceUnavailable,
			contextutils.SeverityError,
			fmt.Sprintf("Linear API returned status %d when looking up team labels: %s", resp.StatusCode, string(body)),
			"",
		)
	}

	var labelResponse struct {
		Data struct {
			Team struct {
				Labels struct {
					Nodes []struct {
						ID   string `json:"id"`
						Name string `json:"name"`
					} `json:"nodes"`
				} `json:"labels"`
			} `json:"team"`
		} `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors,omitempty"`
	}

	if err := json.Unmarshal(body, &labelResponse); err != nil {
		return "", contextutils.WrapError(err, "failed to unmarshal team label lookup response")
	}

	if len(labelResponse.Errors) > 0 {
		return "", contextutils.NewAppError(
			contextutils.ErrorCodeServiceUnavailable,
			contextutils.SeverityError,
			fmt.Sprintf("Linear API error when looking up team labels: %s", labelResponse.Errors[0].Message),
			"",
		)
	}

	// Find label by name (case-insensitive)
	for _, label := range labelResponse.Data.Team.Labels.Nodes {
		if strings.EqualFold(label.Name, labelIdentifier) {
			return label.ID, nil
		}
	}

	return "", contextutils.NewAppError(
		contextutils.ErrorCodeInvalidInput,
		contextutils.SeverityError,
		fmt.Sprintf("Label '%s' not found in Linear team", labelIdentifier),
		"",
	)
}

// getProjectLabelIDByName looks up a project-specific label ID by name
func (s *LinearService) getProjectLabelIDByName(ctx context.Context, projectID, labelIdentifier string) (string, error) {
	// Query Linear for project-specific labels
	query := `
		query ProjectLabels($projectId: String!) {
			project(id: $projectId) {
				labels {
					nodes {
						id
						name
					}
				}
			}
		}
	`

	requestBody := map[string]interface{}{
		"query": query,
		"variables": map[string]interface{}{
			"projectId": projectID,
		},
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return "", contextutils.WrapError(err, "failed to marshal project label lookup request")
	}

	apiURL := s.apiURL
	if apiURL == "" {
		apiURL = LinearAPIEndpoint
	}
	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", contextutils.WrapError(err, "failed to create project label lookup request")
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", s.config.Linear.APIKey)
	req.Header.Set("User-Agent", "quizapp/1.0")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", contextutils.WrapErrorf(err, "failed to query Linear project labels")
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			s.logger.Warn(ctx, "Failed to close response body", map[string]interface{}{"error": closeErr.Error()})
		}
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", contextutils.WrapError(err, "failed to read project label lookup response")
	}

	if resp.StatusCode != http.StatusOK {
		return "", contextutils.NewAppError(
			contextutils.ErrorCodeServiceUnavailable,
			contextutils.SeverityError,
			fmt.Sprintf("Linear API returned status %d when looking up project labels: %s", resp.StatusCode, string(body)),
			"",
		)
	}

	var labelResponse struct {
		Data struct {
			Project struct {
				Labels struct {
					Nodes []struct {
						ID   string `json:"id"`
						Name string `json:"name"`
					} `json:"nodes"`
				} `json:"labels"`
			} `json:"project"`
		} `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors,omitempty"`
	}

	if err := json.Unmarshal(body, &labelResponse); err != nil {
		return "", contextutils.WrapError(err, "failed to unmarshal project label lookup response")
	}

	if len(labelResponse.Errors) > 0 {
		return "", contextutils.NewAppError(
			contextutils.ErrorCodeServiceUnavailable,
			contextutils.SeverityError,
			fmt.Sprintf("Linear API error when looking up project labels: %s", labelResponse.Errors[0].Message),
			"",
		)
	}

	// Find label by name (case-insensitive)
	for _, label := range labelResponse.Data.Project.Labels.Nodes {
		if strings.EqualFold(label.Name, labelIdentifier) {
			return label.ID, nil
		}
	}

	return "", contextutils.NewAppError(
		contextutils.ErrorCodeInvalidInput,
		contextutils.SeverityError,
		fmt.Sprintf("Label '%s' not found in Linear project", labelIdentifier),
		"",
	)
}

// CreateIssue creates a new issue in Linear
func (s *LinearService) CreateIssue(ctx context.Context, title, description, teamID, projectID string, labels []string, state string) (result *LinearIssueResult, err error) {
	ctx, span := observability.TraceFunction(ctx, "linear", "create_issue",
		attribute.String("linear.title", title),
		attribute.String("linear.team_id", teamID),
		attribute.String("linear.project_id", projectID),
	)
	defer observability.FinishSpan(span, &err)

	if !s.config.Linear.Enabled {
		err = contextutils.NewAppError(
			contextutils.ErrorCodeServiceUnavailable,
			contextutils.SeverityError,
			"Linear integration is disabled",
			"",
		)
		return nil, err
	}

	if s.config.Linear.APIKey == "" {
		err = contextutils.NewAppError(
			contextutils.ErrorCodeServiceUnavailable,
			contextutils.SeverityError,
			"Linear API key is not configured",
			"",
		)
		return nil, err
	}

	if teamID == "" {
		teamID = s.config.Linear.TeamID
		if teamID == "" {
			err = contextutils.NewAppError(
				contextutils.ErrorCodeInvalidInput,
				contextutils.SeverityError,
				"Linear team ID or name is required",
				"",
			)
			return nil, err
		}
	}

	// Look up team ID by name if it's not a UUID
	actualTeamID, err := s.getTeamIDByName(ctx, teamID)
	if err != nil {
		return nil, err
	}
	teamID = actualTeamID

	// Use default project ID if none provided and resolve it
	actualProjectID := projectID
	if actualProjectID == "" {
		actualProjectID = s.config.Linear.ProjectID
	}

	// Look up project ID by name if provided and not a UUID (needed for project label lookup)
	if actualProjectID != "" {
		resolvedProjectID, err := s.getProjectIDByName(ctx, actualProjectID, teamID)
		if err != nil {
			// If project lookup fails, log warning but continue without project
			s.logger.Warn(ctx, "Failed to look up Linear project, continuing without project", map[string]interface{}{
				"project_identifier": actualProjectID,
				"error":              err.Error(),
			})
			actualProjectID = "" // Don't include project if lookup failed
		} else {
			actualProjectID = resolvedProjectID
		}
	}

	// Look up label IDs by name if provided
	// Try organization labels first, then team labels, then project labels
	var labelIDs []string
	if len(labels) > 0 {
		for _, labelName := range labels {
			labelID, err := s.getLabelIDByName(ctx, labelName)
			if err != nil {
				// Try team-specific labels as fallback
				labelID, err = s.getTeamLabelIDByName(ctx, teamID, labelName)
				if err != nil {
					// Try project-specific labels if project ID is available
					if actualProjectID != "" {
						labelID, err = s.getProjectLabelIDByName(ctx, actualProjectID, labelName)
						if err != nil {
							// Log warning but continue without this label
							s.logger.Warn(ctx, "Failed to look up Linear label (tried organization, team, and project labels), continuing without it", map[string]interface{}{
								"label_name": labelName,
								"team_id":    teamID,
								"project_id": actualProjectID,
								"error":      err.Error(),
							})
							continue
						}
					} else {
						// Log warning but continue without this label
						s.logger.Warn(ctx, "Failed to look up Linear label (tried organization and team labels), continuing without it", map[string]interface{}{
							"label_name": labelName,
							"team_id":    teamID,
							"error":      err.Error(),
						})
						continue
					}
				}
			}
			labelIDs = append(labelIDs, labelID)
		}
	} else if len(s.config.Linear.DefaultLabels) > 0 {
		// Use default labels if none provided
		for _, labelName := range s.config.Linear.DefaultLabels {
			labelID, err := s.getLabelIDByName(ctx, labelName)
			if err != nil {
				// Try team-specific labels as fallback
				labelID, err = s.getTeamLabelIDByName(ctx, teamID, labelName)
				if err != nil {
					// Try project-specific labels if project ID is available
					if actualProjectID != "" {
						labelID, err = s.getProjectLabelIDByName(ctx, actualProjectID, labelName)
						if err != nil {
							// Log warning but continue without this label
							s.logger.Warn(ctx, "Failed to look up default Linear label (tried organization, team, and project labels), continuing without it", map[string]interface{}{
								"label_name": labelName,
								"team_id":    teamID,
								"project_id": actualProjectID,
								"error":      err.Error(),
							})
							continue
						}
					} else {
						// Log warning but continue without this label
						s.logger.Warn(ctx, "Failed to look up default Linear label (tried organization and team labels), continuing without it", map[string]interface{}{
							"label_name": labelName,
							"team_id":    teamID,
							"error":      err.Error(),
						})
						continue
					}
				}
			}
			labelIDs = append(labelIDs, labelID)
		}
	}

	// Use default state if none provided
	// Note: State is not yet implemented (requires fetching state ID from Linear)
	if state == "" {
		_ = s.config.Linear.DefaultState // Will be used when state ID lookup is implemented
	}

	projectID = actualProjectID

	// Build GraphQL mutation
	// Required fields: teamId, title
	// Optional fields: description, projectId, assigneeId, labelIds (array of IDs), stateId (ID, not name)
	mutation := `
		mutation IssueCreate($input: IssueCreateInput!) {
			issueCreate(input: $input) {
				success
				issue {
					id
					title
					url
				}
			}
		}
	`

	input := map[string]interface{}{
		"title":  title,
		"teamId": teamID,
	}

	// Only add description if it's not empty (Linear may reject empty strings)
	if description != "" {
		input["description"] = description
	}

	// Add project ID if provided (Linear accepts projectId as UUID or name)
	// Note: Linear expects projectId to be a valid UUID or identifier
	if projectID != "" {
		input["projectId"] = projectID
	}

	// Add label IDs if any were resolved
	if len(labelIDs) > 0 {
		input["labelIds"] = labelIDs
	}

	variables := map[string]interface{}{
		"input": input,
	}

	requestBody := map[string]interface{}{
		"query":     mutation,
		"variables": variables,
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		span.SetAttributes(attribute.String("error", err.Error()))
		return nil, contextutils.WrapError(err, "failed to marshal GraphQL request")
	}

	apiURL := s.apiURL
	if apiURL == "" {
		apiURL = LinearAPIEndpoint
	}
	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		span.SetAttributes(attribute.String("error", err.Error()))
		return nil, contextutils.WrapError(err, "failed to create HTTP request")
	}

	req.Header.Set("Content-Type", "application/json")
	// Personal API keys should NOT use "Bearer" prefix per Linear docs
	// OAuth2 tokens use "Bearer" prefix, but personal API keys use the key directly
	req.Header.Set("Authorization", s.config.Linear.APIKey)
	req.Header.Set("User-Agent", "quizapp/1.0")

	startTime := time.Now()
	resp, err := s.httpClient.Do(req)
	duration := time.Since(startTime)

	if err != nil {
		s.logger.Error(ctx, "Linear HTTP request failed", err, map[string]interface{}{
			"duration": duration.String(),
		})
		span.SetAttributes(
			attribute.String("error", err.Error()),
			attribute.String("duration", duration.String()),
		)
		return nil, contextutils.WrapErrorf(err, "Linear HTTP request failed after %v", duration)
	}
	defer func() {
		if cerr := resp.Body.Close(); cerr != nil {
			s.logger.Warn(ctx, "Failed to close response body", map[string]interface{}{
				"error": cerr.Error(),
			})
		}
	}()

	span.SetAttributes(
		attribute.Int("http.status_code", resp.StatusCode),
		attribute.String("duration", duration.String()),
	)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		span.SetAttributes(attribute.String("error", err.Error()))
		return nil, contextutils.WrapError(err, "failed to read response body")
	}

	if resp.StatusCode != http.StatusOK {
		s.logger.Error(ctx, "Linear API returned non-200 status", nil, map[string]interface{}{
			"status_code": resp.StatusCode,
			"body":        string(body),
		})
		span.SetAttributes(
			attribute.String("error", fmt.Sprintf("Linear API returned status %d", resp.StatusCode)),
			attribute.String("response_body", string(body)),
		)
		return nil, contextutils.NewAppError(
			contextutils.ErrorCodeServiceUnavailable,
			contextutils.SeverityError,
			fmt.Sprintf("Linear API returned status %d: %s", resp.StatusCode, string(body)),
			"",
		)
	}

	var linearResp LinearIssueResponse
	if err := json.Unmarshal(body, &linearResp); err != nil {
		span.SetAttributes(attribute.String("error", err.Error()))
		return nil, contextutils.WrapError(err, "failed to unmarshal Linear response")
	}

	// Check for GraphQL errors
	if len(linearResp.Errors) > 0 {
		errorMsg := linearResp.Errors[0].Message
		// Log full error details including extensions which may contain validation details
		errorDetails := make([]map[string]interface{}, len(linearResp.Errors))
		for i, err := range linearResp.Errors {
			errorDetails[i] = map[string]interface{}{
				"message": err.Message,
			}
			if len(err.Extensions) > 0 {
				errorDetails[i]["extensions"] = err.Extensions
			}
			if len(err.Path) > 0 {
				errorDetails[i]["path"] = err.Path
			}
		}

		// Build detailed error message with all error information
		var detailedErrorMsg strings.Builder
		detailedErrorMsg.WriteString(errorMsg)
		if len(linearResp.Errors[0].Extensions) > 0 {
			detailedErrorMsg.WriteString("\nExtensions: ")
			extJSON, _ := json.Marshal(linearResp.Errors[0].Extensions)
			detailedErrorMsg.WriteString(string(extJSON))
		}
		if len(linearResp.Errors[0].Path) > 0 {
			detailedErrorMsg.WriteString("\nPath: ")
			pathJSON, _ := json.Marshal(linearResp.Errors[0].Path)
			detailedErrorMsg.WriteString(string(pathJSON))
		}

		s.logger.Error(ctx, "Linear GraphQL error", nil, map[string]interface{}{
			"errors":        errorDetails,
			"request_body":  string(jsonData), // Log the request for debugging
			"full_response": string(body),     // Log full response for debugging
		})
		span.SetAttributes(attribute.String("error", detailedErrorMsg.String()))
		return nil, contextutils.NewAppError(
			contextutils.ErrorCodeServiceUnavailable,
			contextutils.SeverityError,
			detailedErrorMsg.String(),
			"",
		)
	}

	if !linearResp.Data.IssueCreate.Success {
		s.logger.Error(ctx, "Linear issue creation failed", nil, map[string]interface{}{})
		span.SetAttributes(attribute.String("error", "Linear issue creation was not successful"))
		return nil, contextutils.NewAppError(
			contextutils.ErrorCodeServiceUnavailable,
			contextutils.SeverityError,
			"Linear issue creation was not successful",
			"",
		)
	}

	issue := linearResp.Data.IssueCreate.Issue

	// Construct the URL if not provided (Linear sometimes doesn't return it)
	issueURL := issue.URL
	if issueURL == "" {
		issueURL = fmt.Sprintf("https://linear.app/issue/%s", issue.ID)
	}

	result = &LinearIssueResult{
		IssueID:  issue.ID,
		IssueURL: issueURL,
		Title:    issue.Title,
	}

	s.logger.Info(ctx, "Linear issue created successfully", map[string]interface{}{
		"issue_id":  issue.ID,
		"issue_url": issueURL,
		"duration":  duration.String(),
	})

	span.SetAttributes(
		attribute.String("linear.issue_id", issue.ID),
		attribute.String("linear.issue_url", issueURL),
	)

	return result, nil
}
