package services

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"quizapp/internal/config"
	"quizapp/internal/observability"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newLinearTestService(t *testing.T, handler http.HandlerFunc) (*LinearService, func()) {
	t.Helper()
	server := httptest.NewServer(handler)
	cfg := &config.Config{}
	cfg.Linear.Enabled = true
	cfg.Linear.APIKey = "linear-token"
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	service := NewLinearServiceWithURL(cfg, logger, server.URL)

	cleanup := func() {
		server.Close()
	}
	return service, cleanup
}

func TestLinearService_CreateIssue_Success(t *testing.T) {
	var receivedBodies [][]byte
	handler := func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		receivedBodies = append(receivedBodies, body)

		resp := LinearIssueResponse{
			Data: struct {
				IssueCreate struct {
					Success bool `json:"success"`
					Issue   struct {
						ID    string `json:"id"`
						Title string `json:"title"`
						URL   string `json:"url"`
					} `json:"issue"`
				} `json:"issueCreate"`
			}{
				IssueCreate: struct {
					Success bool `json:"success"`
					Issue   struct {
						ID    string `json:"id"`
						Title string `json:"title"`
						URL   string `json:"url"`
					} `json:"issue"`
				}{
					Success: true,
					Issue: struct {
						ID    string `json:"id"`
						Title string `json:"title"`
						URL   string `json:"url"`
					}{
						ID:    "ISSUE-123",
						Title: "Created title",
						URL:   "https://linear.app/wetsnow/issue/ISSUE-123",
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode(resp))
	}

	service, cleanup := newLinearTestService(t, handler)
	defer cleanup()

	result, err := service.CreateIssue(context.Background(),
		"Test title",
		"Test description",
		"3fa85f64-5717-4562-b3fc-2c963f66afa6",
		"",
		nil,
		"",
	)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "ISSUE-123", result.IssueID)
	assert.Equal(t, "https://linear.app/wetsnow/issue/ISSUE-123", result.IssueURL)

	require.Len(t, receivedBodies, 1)
	assert.Contains(t, string(receivedBodies[0]), "mutation IssueCreate")
	assert.Contains(t, string(receivedBodies[0]), "Test title")
	assert.Contains(t, string(receivedBodies[0]), "Test description")
}

func TestLinearService_CreateIssue_Disabled(t *testing.T) {
	cfg := &config.Config{}
	cfg.Linear.Enabled = false
	cfg.Linear.APIKey = "ignored"
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	service := NewLinearService(cfg, logger)

	result, err := service.CreateIssue(context.Background(), "t", "d", "", "", nil, "")
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "Linear integration is disabled")
}

func TestLinearService_getTeamIDByName_ByName(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		assert.Contains(t, string(body), "query Teams")

		resp := map[string]any{
			"data": map[string]any{
				"teams": map[string]any{
					"nodes": []map[string]string{
						{"id": "team-123", "name": "Alpha"},
						{"id": "team-456", "name": "Target Team"},
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode(resp))
	}

	service, cleanup := newLinearTestService(t, handler)
	defer cleanup()

	id, err := service.getTeamIDByName(context.Background(), "Target Team")
	require.NoError(t, err)
	assert.Equal(t, "team-456", id)
}

func TestLinearService_getTeamIDByName_NotFound(t *testing.T) {
	handler := func(w http.ResponseWriter, _ *http.Request) {
		resp := map[string]any{
			"data": map[string]any{
				"teams": map[string]any{
					"nodes": []map[string]string{
						{"id": "team-123", "name": "Another"},
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode(resp))
	}

	service, cleanup := newLinearTestService(t, handler)
	defer cleanup()

	id, err := service.getTeamIDByName(context.Background(), "Missing Team")
	assert.Empty(t, id)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Team 'Missing Team' not found")
}

func TestLinearService_getProjectIDByName_ByName(t *testing.T) {
	var seen bool
	handler := func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		assert.Contains(t, string(body), "query Projects")
		seen = true

		resp := map[string]any{
			"data": map[string]any{
				"team": map[string]any{
					"projects": map[string]any{
						"nodes": []map[string]string{
							{"id": "proj-1", "name": "Inbox"},
							{"id": "proj-42", "name": "Target Project"},
						},
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode(resp))
	}

	service, cleanup := newLinearTestService(t, handler)
	defer cleanup()

	id, err := service.getProjectIDByName(context.Background(), "Target Project", "team-456")
	require.NoError(t, err)
	assert.Equal(t, "proj-42", id)
	assert.True(t, seen)
}

func TestLinearService_getLabelIDByName_TeamFallback(t *testing.T) {
	var requestBodies [][]byte
	handler := func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		requestBodies = append(requestBodies, body)

		switch len(requestBodies) {
		case 1:
			// Organization labels call, return no match
			resp := map[string]any{
				"data": map[string]any{
					"organization": map[string]any{
						"labels": map[string]any{
							"nodes": []map[string]string{
								{"id": "org-1", "name": "Other Label"},
							},
						},
					},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			require.NoError(t, json.NewEncoder(w).Encode(resp))
		case 2:
			// Team labels fallback
			resp := map[string]any{
				"data": map[string]any{
					"team": map[string]any{
						"labels": map[string]any{
							"nodes": []map[string]string{
								{"id": "team-label", "name": "Target Label"},
							},
						},
					},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			require.NoError(t, json.NewEncoder(w).Encode(resp))
		case 3:
			// Final issue creation mutation
			resp := LinearIssueResponse{
				Data: struct {
					IssueCreate struct {
						Success bool `json:"success"`
						Issue   struct {
							ID    string `json:"id"`
							Title string `json:"title"`
							URL   string `json:"url"`
						} `json:"issue"`
					} `json:"issueCreate"`
				}{
					IssueCreate: struct {
						Success bool `json:"success"`
						Issue   struct {
							ID    string `json:"id"`
							Title string `json:"title"`
							URL   string `json:"url"`
						} `json:"issue"`
					}{
						Success: true,
						Issue: struct {
							ID    string `json:"id"`
							Title string `json:"title"`
							URL   string `json:"url"`
						}{
							ID:    "ISSUE-999",
							Title: "title",
							URL:   "https://linear.app/issue/ISSUE-999",
						},
					},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			require.NoError(t, json.NewEncoder(w).Encode(resp))
		default:
			w.WriteHeader(http.StatusInternalServerError)
		}
	}

	service, cleanup := newLinearTestService(t, handler)
	defer cleanup()

	service.config.Linear.DefaultLabels = []string{"Target Label"}
	service.config.Linear.TeamID = "3fa85f64-5717-4562-b3fc-2c963f66afa6"

	_, err := service.CreateIssue(context.Background(), "title", "desc", "3fa85f64-5717-4562-b3fc-2c963f66afa6", "", nil, "")
	require.NoError(t, err)

	// Ensure both organization and team label requests were made.
	require.Len(t, requestBodies, 3)
	assert.True(t, strings.Contains(string(requestBodies[0]), "query Labels"))
	assert.True(t, strings.Contains(string(requestBodies[1]), "query TeamLabels"))
	assert.True(t, strings.Contains(string(requestBodies[2]), "mutation IssueCreate"))
}

type mockResponse struct {
	expectedSubstring string
	status            int
	body              any
}

func TestLinearService_getTeamIDByName_UUID(t *testing.T) {
	service, cleanup := newLinearTestService(t, func(_ http.ResponseWriter, _ *http.Request) {
		t.Fatalf("unexpected HTTP request for UUID input")
	})
	defer cleanup()

	id, err := service.getTeamIDByName(context.Background(), "3fa85f64-5717-4562-b3fc-2c963f66afa6")
	require.NoError(t, err)
	assert.Equal(t, "3fa85f64-5717-4562-b3fc-2c963f66afa6", id)
}

func TestLinearService_getProjectIDByName_UUID(t *testing.T) {
	service, cleanup := newLinearTestService(t, func(_ http.ResponseWriter, _ *http.Request) {
		t.Fatalf("unexpected HTTP request for UUID input")
	})
	defer cleanup()

	id, err := service.getProjectIDByName(context.Background(), "3fa85f64-5717-4562-b3fc-2c963f66afa6", "team-123")
	require.NoError(t, err)
	assert.Equal(t, "3fa85f64-5717-4562-b3fc-2c963f66afa6", id)
}

func TestLinearService_getLabelIDByName_UUID(t *testing.T) {
	service, cleanup := newLinearTestService(t, func(_ http.ResponseWriter, _ *http.Request) {
		t.Fatalf("unexpected HTTP request for UUID input")
	})
	defer cleanup()

	id, err := service.getLabelIDByName(context.Background(), "3fa85f64-5717-4562-b3fc-2c963f66afa6")
	require.NoError(t, err)
	assert.Equal(t, "3fa85f64-5717-4562-b3fc-2c963f66afa6", id)
}

func TestLinearService_CreateIssue_WithLookupsAndProjectLabelFallback(t *testing.T) {
	responses := []mockResponse{
		{
			expectedSubstring: "query Teams",
			status:            http.StatusOK,
			body: map[string]any{
				"data": map[string]any{
					"teams": map[string]any{
						"nodes": []map[string]string{
							{"id": "team-xyz", "name": "Team Name"},
						},
					},
				},
			},
		},
		{
			expectedSubstring: "query Projects",
			status:            http.StatusOK,
			body: map[string]any{
				"data": map[string]any{
					"team": map[string]any{
						"projects": map[string]any{
							"nodes": []map[string]string{
								{"id": "proj-123", "name": "My Project"},
							},
						},
					},
				},
			},
		},
		{
			expectedSubstring: "query Labels",
			status:            http.StatusOK,
			body: map[string]any{
				"data": map[string]any{
					"organization": map[string]any{
						"labels": map[string]any{
							"nodes": []map[string]string{
								{"id": "org-1", "name": "Other Label"},
							},
						},
					},
				},
			},
		},
		{
			expectedSubstring: "query TeamLabels",
			status:            http.StatusOK,
			body: map[string]any{
				"data": map[string]any{
					"team": map[string]any{
						"labels": map[string]any{
							"nodes": []map[string]string{
								{"id": "team-1", "name": "Different Label"},
							},
						},
					},
				},
			},
		},
		{
			expectedSubstring: "query ProjectLabels",
			status:            http.StatusOK,
			body: map[string]any{
				"data": map[string]any{
					"project": map[string]any{
						"labels": map[string]any{
							"nodes": []map[string]string{
								{"id": "proj-label-1", "name": "Project Label"},
							},
						},
					},
				},
			},
		},
		{
			expectedSubstring: "mutation IssueCreate",
			status:            http.StatusOK,
			body: LinearIssueResponse{
				Data: struct {
					IssueCreate struct {
						Success bool `json:"success"`
						Issue   struct {
							ID    string `json:"id"`
							Title string `json:"title"`
							URL   string `json:"url"`
						} `json:"issue"`
					} `json:"issueCreate"`
				}{
					IssueCreate: struct {
						Success bool `json:"success"`
						Issue   struct {
							ID    string `json:"id"`
							Title string `json:"title"`
							URL   string `json:"url"`
						} `json:"issue"`
					}{
						Success: true,
						Issue: struct {
							ID    string `json:"id"`
							Title string `json:"title"`
							URL   string `json:"url"`
						}{
							ID:    "ISSUE-456",
							Title: "Complex title",
							URL:   "https://linear.app/issue/ISSUE-456",
						},
					},
				},
			},
		},
	}

	handler := func(w http.ResponseWriter, r *http.Request) {
		if len(responses) == 0 {
			t.Fatalf("unexpected request: %s", r.URL.Path)
		}
		next := responses[0]
		responses = responses[1:]

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		if next.expectedSubstring != "" {
			assert.Contains(t, string(body), next.expectedSubstring)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(next.status)
		require.NoError(t, json.NewEncoder(w).Encode(next.body))
	}

	service, cleanup := newLinearTestService(t, handler)
	defer cleanup()

	service.config.Linear.TeamID = "Team Name"
	service.config.Linear.ProjectID = "My Project"
	service.config.Linear.DefaultLabels = []string{"Project Label"}

	result, err := service.CreateIssue(context.Background(), "Complex title", "Full description", "", "", nil, "")
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "ISSUE-456", result.IssueID)
	assert.Equal(t, 0, len(responses), "not all mock responses were consumed")
}

func TestLinearService_CreateIssue_TeamLookupFailure(t *testing.T) {
	handler := func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, err := fmt.Fprint(w, `{"errors":[{"message":"boom"}]}`)
		require.NoError(t, err)
	}

	service, cleanup := newLinearTestService(t, handler)
	defer cleanup()

	service.config.Linear.TeamID = "Named Team"

	result, err := service.CreateIssue(context.Background(), "title", "desc", "", "", nil, "")
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "Linear API returned status 500")
}

func TestLinearService_getProjectLabelIDByName_Success(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		assert.Contains(t, string(body), "query ProjectLabels")

		resp := map[string]any{
			"data": map[string]any{
				"project": map[string]any{
					"labels": map[string]any{
						"nodes": []map[string]string{
							{"id": "proj-label-2", "name": "Desired Label"},
						},
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode(resp))
	}

	service, cleanup := newLinearTestService(t, handler)
	defer cleanup()

	id, err := service.getProjectLabelIDByName(context.Background(), "proj-123", "Desired Label")
	require.NoError(t, err)
	assert.Equal(t, "proj-label-2", id)
}

func TestLinearService_getProjectLabelIDByName_HTTPError(t *testing.T) {
	handler := func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, err := fmt.Fprint(w, `{"message":"failed"}`)
		require.NoError(t, err)
	}

	service, cleanup := newLinearTestService(t, handler)
	defer cleanup()

	id, err := service.getProjectLabelIDByName(context.Background(), "proj-123", "Label")
	assert.Empty(t, id)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Linear API returned status 502")
}

func TestLinearService_getTeamIDByName_HTTPError(t *testing.T) {
	handler := func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, err := fmt.Fprint(w, "bad gateway")
		require.NoError(t, err)
	}

	service, cleanup := newLinearTestService(t, handler)
	defer cleanup()

	id, err := service.getTeamIDByName(context.Background(), "Team Name")
	assert.Empty(t, id)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "status 502")
}

func TestLinearService_getProjectIDByName_GraphQLError(t *testing.T) {
	handler := func(w http.ResponseWriter, _ *http.Request) {
		resp := map[string]any{
			"errors": []map[string]string{
				{"message": "project error"},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode(resp))
	}

	service, cleanup := newLinearTestService(t, handler)
	defer cleanup()

	id, err := service.getProjectIDByName(context.Background(), "Project Name", "team-123")
	assert.Empty(t, id)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Linear API error when looking up projects")
}

func TestLinearService_getLabelIDByName_GraphQLError(t *testing.T) {
	handler := func(w http.ResponseWriter, _ *http.Request) {
		resp := map[string]any{
			"errors": []map[string]string{
				{"message": "label error"},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode(resp))
	}

	service, cleanup := newLinearTestService(t, handler)
	defer cleanup()

	id, err := service.getLabelIDByName(context.Background(), "Label")
	assert.Empty(t, id)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Linear API error when looking up labels")
}

func TestLinearService_getTeamLabelIDByName_NotFound(t *testing.T) {
	handler := func(w http.ResponseWriter, _ *http.Request) {
		resp := map[string]any{
			"data": map[string]any{
				"team": map[string]any{
					"labels": map[string]any{
						"nodes": []map[string]string{
							{"id": "team-label", "name": "Different"},
						},
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode(resp))
	}

	service, cleanup := newLinearTestService(t, handler)
	defer cleanup()

	id, err := service.getTeamLabelIDByName(context.Background(), "team-123", "Missing Label")
	assert.Empty(t, id)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Label 'Missing Label' not found in Linear team")
}

func TestLinearService_getProjectLabelIDByName_NotFound(t *testing.T) {
	handler := func(w http.ResponseWriter, _ *http.Request) {
		resp := map[string]any{
			"data": map[string]any{
				"project": map[string]any{
					"labels": map[string]any{
						"nodes": []map[string]string{
							{"id": "other", "name": "Different"},
						},
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode(resp))
	}

	service, cleanup := newLinearTestService(t, handler)
	defer cleanup()

	id, err := service.getProjectLabelIDByName(context.Background(), "proj-123", "Missing")
	assert.Empty(t, id)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Label 'Missing' not found in Linear project")
}

func TestLinearService_CreateIssue_NoAPIKey(t *testing.T) {
	cfg := &config.Config{}
	cfg.Linear.Enabled = true
	cfg.Linear.APIKey = ""
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	service := NewLinearService(cfg, logger)

	result, err := service.CreateIssue(context.Background(), "t", "d", "team", "", nil, "")
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "Linear API key is not configured")
}

func TestLinearService_CreateIssue_GraphQLError(t *testing.T) {
	handler := func(w http.ResponseWriter, _ *http.Request) {
		resp := map[string]any{
			"data": map[string]any{},
			"errors": []map[string]string{
				{"message": "mutation failed"},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode(resp))
	}

	service, cleanup := newLinearTestService(t, handler)
	defer cleanup()

	result, err := service.CreateIssue(context.Background(), "title", "desc", "3fa85f64-5717-4562-b3fc-2c963f66afa6", "", nil, "")
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "mutation failed")
}

func TestLinearService_CreateIssue_LabelNotFound(t *testing.T) {
	responses := []mockResponse{
		{
			expectedSubstring: "query Projects",
			status:            http.StatusOK,
			body: map[string]any{
				"data": map[string]any{
					"team": map[string]any{
						"projects": map[string]any{
							"nodes": []map[string]string{
								{"id": "proj-123", "name": "Project Name"},
							},
						},
					},
				},
			},
		},
		{
			expectedSubstring: "query Labels",
			status:            http.StatusOK,
			body: map[string]any{
				"data": map[string]any{
					"organization": map[string]any{
						"labels": map[string]any{
							"nodes": []map[string]string{
								{"id": "org-1", "name": "Other"},
							},
						},
					},
				},
			},
		},
		{
			expectedSubstring: "query TeamLabels",
			status:            http.StatusOK,
			body: map[string]any{
				"data": map[string]any{
					"team": map[string]any{
						"labels": map[string]any{
							"nodes": []map[string]string{
								{"id": "team-1", "name": "Different"},
							},
						},
					},
				},
			},
		},
		{
			expectedSubstring: "query ProjectLabels",
			status:            http.StatusOK,
			body: map[string]any{
				"data": map[string]any{
					"project": map[string]any{
						"labels": map[string]any{
							"nodes": []map[string]string{
								{"id": "proj-1", "name": "Other"},
							},
						},
					},
				},
			},
		},
		{
			expectedSubstring: "mutation IssueCreate",
			status:            http.StatusOK,
			body: LinearIssueResponse{
				Data: struct {
					IssueCreate struct {
						Success bool `json:"success"`
						Issue   struct {
							ID    string `json:"id"`
							Title string `json:"title"`
							URL   string `json:"url"`
						} `json:"issue"`
					} `json:"issueCreate"`
				}{
					IssueCreate: struct {
						Success bool `json:"success"`
						Issue   struct {
							ID    string `json:"id"`
							Title string `json:"title"`
							URL   string `json:"url"`
						} `json:"issue"`
					}{
						Success: true,
						Issue: struct {
							ID    string `json:"id"`
							Title string `json:"title"`
							URL   string `json:"url"`
						}{
							ID:    "ISSUE-777",
							Title: "title",
							URL:   "https://linear.app/issue/ISSUE-777",
						},
					},
				},
			},
		},
	}

	handler := func(w http.ResponseWriter, r *http.Request) {
		if len(responses) == 0 {
			t.Fatalf("unexpected request: %s", r.URL.Path)
		}
		next := responses[0]
		responses = responses[1:]

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		if next.expectedSubstring != "" {
			assert.Contains(t, string(body), next.expectedSubstring)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(next.status)
		require.NoError(t, json.NewEncoder(w).Encode(next.body))
	}

	service, cleanup := newLinearTestService(t, handler)
	defer cleanup()

	service.config.Linear.ProjectID = "Project Name"

	_, err := service.CreateIssue(context.Background(), "title", "desc",
		"3fa85f64-5717-4562-b3fc-2c963f66afa6",
		"Project Name",
		[]string{"Unknown Label"},
		"",
	)
	require.NoError(t, err)
	assert.Equal(t, 0, len(responses), "not all requests were consumed")
}

func TestLinearService_getLabelIDByName_HTTPError(t *testing.T) {
	handler := func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, err := fmt.Fprint(w, "oops")
		require.NoError(t, err)
	}

	service, cleanup := newLinearTestService(t, handler)
	defer cleanup()

	id, err := service.getLabelIDByName(context.Background(), "Label")
	assert.Empty(t, id)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "status 500")
}

func TestLinearService_getTeamLabelIDByName_HTTPError(t *testing.T) {
	handler := func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		_, err := fmt.Fprint(w, "fail")
		require.NoError(t, err)
	}

	service, cleanup := newLinearTestService(t, handler)
	defer cleanup()

	id, err := service.getTeamLabelIDByName(context.Background(), "team-123", "Label")
	assert.Empty(t, id)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "status 503")
}

func TestLinearService_getTeamLabelIDByName_GraphQLError(t *testing.T) {
	handler := func(w http.ResponseWriter, _ *http.Request) {
		resp := map[string]any{
			"errors": []map[string]string{
				{"message": "team label error"},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode(resp))
	}

	service, cleanup := newLinearTestService(t, handler)
	defer cleanup()

	id, err := service.getTeamLabelIDByName(context.Background(), "team-123", "Label")
	assert.Empty(t, id)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "team label error")
}

func TestLinearService_getProjectLabelIDByName_GraphQLError(t *testing.T) {
	handler := func(w http.ResponseWriter, _ *http.Request) {
		resp := map[string]any{
			"errors": []map[string]string{
				{"message": "project label error"},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode(resp))
	}

	service, cleanup := newLinearTestService(t, handler)
	defer cleanup()

	id, err := service.getProjectLabelIDByName(context.Background(), "proj-123", "Label")
	assert.Empty(t, id)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "project label error")
}

func TestLinearService_CreateIssue_ProjectLookupFails(t *testing.T) {
	responses := []mockResponse{
		{
			expectedSubstring: "query Projects",
			status:            http.StatusInternalServerError,
			body:              map[string]any{"message": "boom"},
		},
		{
			expectedSubstring: "mutation IssueCreate",
			status:            http.StatusOK,
			body: LinearIssueResponse{
				Data: struct {
					IssueCreate struct {
						Success bool `json:"success"`
						Issue   struct {
							ID    string `json:"id"`
							Title string `json:"title"`
							URL   string `json:"url"`
						} `json:"issue"`
					} `json:"issueCreate"`
				}{
					IssueCreate: struct {
						Success bool `json:"success"`
						Issue   struct {
							ID    string `json:"id"`
							Title string `json:"title"`
							URL   string `json:"url"`
						} `json:"issue"`
					}{
						Success: true,
						Issue: struct {
							ID    string `json:"id"`
							Title string `json:"title"`
							URL   string `json:"url"`
						}{
							ID:    "ISSUE-888",
							Title: "title",
							URL:   "https://linear.app/issue/ISSUE-888",
						},
					},
				},
			},
		},
	}

	handler := func(w http.ResponseWriter, r *http.Request) {
		if len(responses) == 0 {
			t.Fatalf("unexpected request: %s", r.URL.Path)
		}
		next := responses[0]
		responses = responses[1:]

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		if next.expectedSubstring != "" {
			assert.Contains(t, string(body), next.expectedSubstring)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(next.status)
		require.NoError(t, json.NewEncoder(w).Encode(next.body))
	}

	service, cleanup := newLinearTestService(t, handler)
	defer cleanup()
	service.config.Linear.ProjectID = "Project Name"

	result, err := service.CreateIssue(context.Background(), "title", "desc", "3fa85f64-5717-4562-b3fc-2c963f66afa6", "", nil, "")
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, 0, len(responses))
}

func TestLinearService_CreateIssue_HTTPStatusError(t *testing.T) {
	handler := func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, err := fmt.Fprint(w, "bad response")
		require.NoError(t, err)
	}

	service, cleanup := newLinearTestService(t, handler)
	defer cleanup()

	result, err := service.CreateIssue(context.Background(), "title", "desc", "3fa85f64-5717-4562-b3fc-2c963f66afa6", "", nil, "")
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "returned status 502")
}

func TestLinearService_CreateIssue_SuccessNoURL(t *testing.T) {
	handler := func(w http.ResponseWriter, _ *http.Request) {
		resp := LinearIssueResponse{
			Data: struct {
				IssueCreate struct {
					Success bool `json:"success"`
					Issue   struct {
						ID    string `json:"id"`
						Title string `json:"title"`
						URL   string `json:"url"`
					} `json:"issue"`
				} `json:"issueCreate"`
			}{
				IssueCreate: struct {
					Success bool `json:"success"`
					Issue   struct {
						ID    string `json:"id"`
						Title string `json:"title"`
						URL   string `json:"url"`
					} `json:"issue"`
				}{
					Success: true,
					Issue: struct {
						ID    string `json:"id"`
						Title string `json:"title"`
						URL   string `json:"url"`
					}{
						ID:    "ISSUE-999",
						Title: "created",
						URL:   "",
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode(resp))
	}

	service, cleanup := newLinearTestService(t, handler)
	defer cleanup()

	result, err := service.CreateIssue(context.Background(), "title", "desc", "3fa85f64-5717-4562-b3fc-2c963f66afa6", "", nil, "")
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "https://linear.app/issue/ISSUE-999", result.IssueURL)
}

func TestLinearService_CreateIssue_Unsuccessful(t *testing.T) {
	handler := func(w http.ResponseWriter, _ *http.Request) {
		resp := LinearIssueResponse{
			Data: struct {
				IssueCreate struct {
					Success bool `json:"success"`
					Issue   struct {
						ID    string `json:"id"`
						Title string `json:"title"`
						URL   string `json:"url"`
					} `json:"issue"`
				} `json:"issueCreate"`
			}{
				IssueCreate: struct {
					Success bool `json:"success"`
					Issue   struct {
						ID    string `json:"id"`
						Title string `json:"title"`
						URL   string `json:"url"`
					} `json:"issue"`
				}{
					Success: false,
					Issue: struct {
						ID    string `json:"id"`
						Title string `json:"title"`
						URL   string `json:"url"`
					}{
						ID:    "",
						Title: "",
						URL:   "",
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode(resp))
	}

	service, cleanup := newLinearTestService(t, handler)
	defer cleanup()

	result, err := service.CreateIssue(context.Background(), "title", "desc", "3fa85f64-5717-4562-b3fc-2c963f66afa6", "", nil, "")
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "Linear issue creation was not successful")
}

func TestLinearService_getTeamIDByName_UnmarshalError(t *testing.T) {
	handler := func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, err := fmt.Fprint(w, "{invalid")
		require.NoError(t, err)
	}

	service, cleanup := newLinearTestService(t, handler)
	defer cleanup()

	id, err := service.getTeamIDByName(context.Background(), "Example")
	assert.Empty(t, id)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to unmarshal team lookup response")
}

func TestLinearService_getProjectIDByName_UnmarshalError(t *testing.T) {
	handler := func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, err := fmt.Fprint(w, "{invalid")
		require.NoError(t, err)
	}

	service, cleanup := newLinearTestService(t, handler)
	defer cleanup()

	id, err := service.getProjectIDByName(context.Background(), "Project", "team-123")
	assert.Empty(t, id)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to unmarshal project lookup response")
}

func TestLinearService_getLabelIDByName_UnmarshalError(t *testing.T) {
	handler := func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, err := fmt.Fprint(w, "{invalid")
		require.NoError(t, err)
	}

	service, cleanup := newLinearTestService(t, handler)
	defer cleanup()

	id, err := service.getLabelIDByName(context.Background(), "Label")
	assert.Empty(t, id)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to unmarshal label lookup response")
}

func TestLinearService_getTeamLabelIDByName_UnmarshalError(t *testing.T) {
	handler := func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, err := fmt.Fprint(w, "{invalid")
		require.NoError(t, err)
	}

	service, cleanup := newLinearTestService(t, handler)
	defer cleanup()

	id, err := service.getTeamLabelIDByName(context.Background(), "team-123", "Label")
	assert.Empty(t, id)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to unmarshal team label lookup response")
}

func TestLinearService_getProjectLabelIDByName_UnmarshalError(t *testing.T) {
	handler := func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, err := fmt.Fprint(w, "{invalid")
		require.NoError(t, err)
	}

	service, cleanup := newLinearTestService(t, handler)
	defer cleanup()

	id, err := service.getProjectLabelIDByName(context.Background(), "proj-123", "Label")
	assert.Empty(t, id)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to unmarshal project label lookup response")
}

func TestLinearService_CreateIssue_MissingTeam(t *testing.T) {
	handler := func(http.ResponseWriter, *http.Request) {
		t.Fatalf("no requests expected")
	}

	service, cleanup := newLinearTestService(t, handler)
	defer cleanup()
	service.config.Linear.TeamID = ""

	result, err := service.CreateIssue(context.Background(), "title", "desc", "", "", nil, "")
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "Linear team ID or name is required")
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestLinearService_CreateIssue_HTTPClientError(t *testing.T) {
	service, cleanup := newLinearTestService(t, func(http.ResponseWriter, *http.Request) {})
	defer cleanup()
	service.httpClient = &http.Client{
		Transport: roundTripFunc(func(_ *http.Request) (*http.Response, error) {
			return nil, fmt.Errorf("network down")
		}),
	}

	result, err := service.CreateIssue(context.Background(), "title", "desc", "3fa85f64-5717-4562-b3fc-2c963f66afa6", "", nil, "")
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "network down")
}
