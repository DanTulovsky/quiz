// Package serviceinterfaces defines service interfaces for dependency injection and testing.
package serviceinterfaces

import (
	"context"
)

// APNSService defines the interface for iOS push notification functionality
type APNSService interface {
	// SendNotification sends a push notification to a device token
	SendNotification(ctx context.Context, deviceToken string, payload map[string]interface{}) error

	// IsEnabled returns whether APNS functionality is enabled
	IsEnabled() bool
}
