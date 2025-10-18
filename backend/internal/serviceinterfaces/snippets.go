// Package serviceinterfaces defines service interfaces for dependency injection and testing.
package serviceinterfaces

import (
	"context"

	"quizapp/internal/api"
	"quizapp/internal/models"
)

// SnippetsService defines the interface for snippets services
type SnippetsService interface {
	// CreateSnippet creates a new vocabulary snippet
	CreateSnippet(ctx context.Context, userID int64, req api.CreateSnippetRequest) (*models.Snippet, error)

	// GetSnippets retrieves snippets for a user with optional filtering
	GetSnippets(ctx context.Context, userID int64, params api.GetV1SnippetsParams) (*api.SnippetList, error)

	// GetSnippet retrieves a specific snippet by ID
	GetSnippet(ctx context.Context, userID, snippetID int64) (*models.Snippet, error)

	// UpdateSnippet updates a snippet's context
	UpdateSnippet(ctx context.Context, userID, snippetID int64, req api.UpdateSnippetRequest) (*models.Snippet, error)

	// DeleteSnippet deletes a snippet
	DeleteSnippet(ctx context.Context, userID, snippetID int64) error
}
