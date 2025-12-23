// Package services provides business logic services for the quiz application.
package services

import (
	"context"

	"quizapp/internal/config"
	"quizapp/internal/observability"
	serviceinterfaces "quizapp/internal/serviceinterfaces"
	contextutils "quizapp/internal/utils"

	"github.com/sideshow/apns2"
	"github.com/sideshow/apns2/payload"
	"github.com/sideshow/apns2/token"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// APNSService implements the interfaces.APNSService interface using apns2
type APNSService struct {
	cfg     *config.Config
	logger  *observability.Logger
	client  *apns2.Client
	enabled bool
}

// APNSServiceInterface defines the interface for APNS functionality
type APNSServiceInterface = serviceinterfaces.APNSService

// Ensure APNSService implements the APNSServiceInterface
var _ serviceinterfaces.APNSService = (*APNSService)(nil)

// NewAPNSService creates a new APNSService instance
func NewAPNSService(cfg *config.Config, logger *observability.Logger) (*APNSService, error) {
	service := &APNSService{
		cfg:     cfg,
		logger:  logger,
		enabled: cfg.APNS.Enabled,
	}

	if !cfg.APNS.Enabled {
		logger.Info(context.Background(), "APNS disabled in configuration", nil)
		return service, nil
	}

	// Validate required configuration
	if cfg.APNS.KeyPath == "" {
		return nil, contextutils.ErrorWithContextf("APNS key_path is required when APNS is enabled")
	}
	if cfg.APNS.KeyID == "" {
		return nil, contextutils.ErrorWithContextf("APNS key_id is required when APNS is enabled")
	}
	if cfg.APNS.TeamID == "" {
		return nil, contextutils.ErrorWithContextf("APNS team_id is required when APNS is enabled")
	}
	if cfg.APNS.BundleID == "" {
		return nil, contextutils.ErrorWithContextf("APNS bundle_id is required when APNS is enabled")
	}

	// Load APNS key
	authKey, err := token.AuthKeyFromFile(cfg.APNS.KeyPath)
	if err != nil {
		return nil, contextutils.WrapErrorf(err, "failed to load APNS key from file: %s", cfg.APNS.KeyPath)
	}

	// Create token
	apnsToken := &token.Token{
		AuthKey: authKey,
		KeyID:   cfg.APNS.KeyID,
		TeamID:  cfg.APNS.TeamID,
	}

	// Create APNS client
	client := apns2.NewTokenClient(apnsToken)
	if cfg.APNS.Production {
		client.Production()
	} else {
		client.Development()
	}

	service.client = client

	logger.Info(context.Background(), "APNS service initialized", map[string]interface{}{
		"bundle_id":  cfg.APNS.BundleID,
		"production": cfg.APNS.Production,
		"key_id":     cfg.APNS.KeyID,
		"team_id":    cfg.APNS.TeamID,
	})

	return service, nil
}

// IsEnabled returns whether APNS functionality is enabled
func (a *APNSService) IsEnabled() bool {
	return a.enabled && a.client != nil
}

// SendNotification sends a push notification to a device token
func (a *APNSService) SendNotification(ctx context.Context, deviceToken string, notificationPayload map[string]interface{}) (err error) {
	ctx, span := otel.Tracer("apns-service").Start(ctx, "SendNotification",
		trace.WithAttributes(
			attribute.String("device_token", deviceToken[:20]+"..."), // Only log first 20 chars for security
		),
	)
	defer observability.FinishSpan(span, &err)

	if !a.IsEnabled() {
		a.logger.Info(ctx, "APNS disabled, skipping notification", map[string]interface{}{
			"device_token": deviceToken[:20] + "...",
		})
		return nil
	}

	// Build notification payload
	notification := &apns2.Notification{}
	notification.DeviceToken = deviceToken
	notification.Topic = a.cfg.APNS.BundleID

	// Build APS payload
	p := payload.NewPayload()

	// Handle alert (can be string or map)
	if alert, ok := notificationPayload["alert"].(string); ok {
		p.Alert(alert)
	} else if alertMap, ok := notificationPayload["alert"].(map[string]interface{}); ok {
		if title, ok := alertMap["title"].(string); ok {
			p.AlertTitle(title)
		}
		if body, ok := alertMap["body"].(string); ok {
			p.AlertBody(body)
		}
	}

	// Handle sound
	if sound, ok := notificationPayload["sound"].(string); ok {
		p.Sound(sound)
	} else {
		p.Sound("default")
	}

	// Handle badge
	if badge, ok := notificationPayload["badge"].(int); ok {
		p.Badge(badge)
	}

	// Add custom data (everything except aps)
	for key, value := range notificationPayload {
		if key != "aps" && key != "alert" && key != "sound" && key != "badge" {
			p.Custom(key, value)
		}
	}

	notification.Payload = p

	// Send notification
	res, err := a.client.Push(notification)
	if err != nil {
		span.RecordError(err, trace.WithStackTrace(true))
		a.logger.Error(ctx, "Failed to send APNS notification", err, map[string]interface{}{
			"device_token": deviceToken[:20] + "...",
		})
		return contextutils.WrapError(err, "failed to send APNS notification")
	}

	if !res.Sent() {
		err := contextutils.ErrorWithContextf("APNS notification failed: %s (status: %d)", res.Reason, res.StatusCode)
		span.RecordError(err, trace.WithStackTrace(true))
		a.logger.Error(ctx, "APNS notification rejected", err, map[string]interface{}{
			"device_token": deviceToken[:20] + "...",
			"status_code":  res.StatusCode,
			"reason":       res.Reason,
		})
		return err
	}

	span.SetAttributes(
		attribute.Int("apns.status_code", res.StatusCode),
		attribute.String("apns.apns_id", res.ApnsID),
	)

	a.logger.Info(ctx, "APNS notification sent successfully", map[string]interface{}{
		"device_token": deviceToken[:20] + "...",
		"apns_id":      res.ApnsID,
	})

	return nil
}
