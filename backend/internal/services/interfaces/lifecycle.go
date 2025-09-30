// Package serviceinterfaces defines service interfaces for dependency injection and testing.
package serviceinterfaces

import (
	"context"
)

// Lifecycle defines the interface for services that need lifecycle management
type Lifecycle interface {
	// Startup is called when the service should initialize
	Startup(ctx context.Context) error

	// Shutdown is called when the service should cleanup
	Shutdown(ctx context.Context) error

	// IsReady returns whether the service is ready to handle requests
	IsReady() bool
}
