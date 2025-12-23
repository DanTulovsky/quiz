// Package services provides business logic services for the quiz application.
package services

import (
	"context"
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"

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

// parseAPNSKeyFromBytes parses an APNS key from bytes (PKCS#8 format)
func parseAPNSKeyFromBytes(keyBytes []byte) (*ecdsa.PrivateKey, error) {
	// Try to parse as PEM first
	block, _ := pem.Decode(keyBytes)
	if block != nil {
		keyBytes = block.Bytes
	}

	// Parse PKCS#8 private key
	privateKey, err := x509.ParsePKCS8PrivateKey(keyBytes)
	if err != nil {
		return nil, contextutils.WrapErrorf(err, "failed to parse PKCS#8 private key")
	}

	// Convert to ECDSA private key
	ecdsaKey, ok := privateKey.(*ecdsa.PrivateKey)
	if !ok {
		return nil, contextutils.ErrorWithContextf("key is not an ECDSA private key")
	}

	return ecdsaKey, nil
}

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
	if cfg.APNS.KeyPath == "" && cfg.APNS.Key == "" {
		return nil, contextutils.ErrorWithContextf("APNS key_path or key is required when APNS is enabled")
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

	// Load APNS key from environment variable (preferred) or file path
	var authKey *ecdsa.PrivateKey
	var err error
	if cfg.APNS.Key != "" {
		// Try to load from key content (base64 encoded or raw)
		keyBytes := []byte(cfg.APNS.Key)
		// Try base64 decoding first
		if decoded, decodeErr := base64.StdEncoding.DecodeString(cfg.APNS.Key); decodeErr == nil {
			keyBytes = decoded
		}
		authKey, err = parseAPNSKeyFromBytes(keyBytes)
		if err != nil {
			return nil, contextutils.WrapErrorf(err, "failed to load APNS key from key content")
		}
	} else {
		// Fall back to file path
		authKey, err = token.AuthKeyFromFile(cfg.APNS.KeyPath)
		if err != nil {
			return nil, contextutils.WrapErrorf(err, "failed to load APNS key from file: %s", cfg.APNS.KeyPath)
		}
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

	// Handle payload structure - can be either:
	// 1. Direct structure: { "alert": {...}, "sound": "...", ... }
	// 2. Nested structure: { "aps": { "alert": {...}, "sound": "..." }, ... }
	var apsData map[string]interface{}
	if aps, ok := notificationPayload["aps"].(map[string]interface{}); ok {
		// Nested structure - extract aps data
		apsData = aps
		// Extract non-aps custom data for later
		for key, value := range notificationPayload {
			if key != "aps" {
				p.Custom(key, value)
			}
		}
	} else {
		// Direct structure - use the payload directly
		apsData = notificationPayload
	}

	// Handle alert (can be string or map)
	if alert, ok := apsData["alert"].(string); ok {
		p.Alert(alert)
	} else if alertMap, ok := apsData["alert"].(map[string]interface{}); ok {
		if title, ok := alertMap["title"].(string); ok {
			p.AlertTitle(title)
		}
		if body, ok := alertMap["body"].(string); ok {
			p.AlertBody(body)
		}
	}

	// Handle sound
	if sound, ok := apsData["sound"].(string); ok {
		p.Sound(sound)
	} else {
		p.Sound("default")
	}

	// Handle badge
	if badge, ok := apsData["badge"].(int); ok {
		p.Badge(badge)
	}

	// For direct structure, add remaining custom data (everything except aps fields)
	if _, ok := notificationPayload["aps"]; !ok {
		for key, value := range notificationPayload {
			if key != "alert" && key != "sound" && key != "badge" {
				p.Custom(key, value)
			}
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
			"apns_env":     map[bool]string{true: "production", false: "sandbox"}[a.cfg.APNS.Production],
			"token_length": len(deviceToken),
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
