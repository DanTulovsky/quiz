// Package worker contains the background worker responsible for generating
// and maintaining daily question assignments, scheduling generation jobs,
// and reporting worker health. The worker runs independently of HTTP
// request handling and interacts with the database, AI providers, and
// other internal services to keep question queues primed for users.
package worker

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"quizapp/internal/config"
	"quizapp/internal/models"
	"quizapp/internal/observability"
	"quizapp/internal/services"
	"quizapp/internal/services/mailer"
	contextutils "quizapp/internal/utils"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

const (
	// NoActionPrefix is used to identify actions that should not be processed
	NoActionPrefix        = config.NoActionPrefix
	triggerThrottleWindow = config.WorkerTriggerThrottle // Prevent multiple triggers for same user within this window
)

// Status represents the current state of the worker
type Status struct {
	IsRunning       bool      `json:"is_running"`
	IsPaused        bool      `json:"is_paused"`
	CurrentActivity string    `json:"current_activity,omitempty"`
	LastRunStart    time.Time `json:"last_run_start"`
	LastRunFinish   time.Time `json:"last_run_finish"`
	LastRunError    string    `json:"last_run_error,omitempty"`
	NextRun         time.Time `json:"next_run"`
}

// RunRecord tracks individual worker runs
type RunRecord struct {
	StartTime time.Time     `json:"start_time"`
	EndTime   time.Time     `json:"end_time"`
	Duration  time.Duration `json:"duration"`
	Status    string        `json:"status"` // Success, Failure
	Details   string        `json:"details"`
}

// ActivityLog represents a single activity log entry
type ActivityLog struct {
	Timestamp time.Time `json:"timestamp"`
	Level     string    `json:"level"` // INFO, WARN, ERROR
	Message   string    `json:"message"`
	UserID    *int      `json:"user_id,omitempty"`
	Username  *string   `json:"username,omitempty"`
}

// UserFailureInfo tracks failure information for exponential backoff
type UserFailureInfo struct {
	ConsecutiveFailures int
	LastFailureTime     time.Time
	NextRetryTime       time.Time
}

// Config holds worker-specific configuration
type Config struct {
	StartWorkerPaused bool
	DailyHorizonDays  int
}

// Worker manages AI question generation in the background
type Worker struct {
	userService          services.UserServiceInterface
	questionService      services.QuestionServiceInterface
	aiService            services.AIServiceInterface
	learningService      services.LearningServiceInterface
	workerService        services.WorkerServiceInterface
	dailyQuestionService services.DailyQuestionServiceInterface
	emailService         mailer.Mailer
	hintService          services.GenerationHintServiceInterface
	instance             string
	status               Status
	history              []RunRecord
	activityLogs         []ActivityLog // Circular buffer for recent activity logs
	mu                   sync.RWMutex
	manualTrigger        chan bool
	cfg                  *config.Config
	workerCfg            Config
	logger               *observability.Logger

	// Track failures for exponential backoff
	userFailures map[int]*UserFailureInfo // userID -> failure info
	failureMu    sync.RWMutex             // mutex for failure tracking

	// Time function for testing - defaults to time.Now
	timeNow func() time.Time
	cancel  context.CancelFunc // Added for cleanup
}

// checkForDailyReminders checks if any users need daily reminder emails
func (w *Worker) checkForDailyReminders(ctx context.Context) error {
	ctx, span := otel.Tracer("worker").Start(ctx, "checkForDailyReminders",
		trace.WithAttributes(
			attribute.String("worker.instance", w.instance),
			attribute.Bool("email.daily_reminder.enabled", w.cfg.Email.DailyReminder.Enabled),
			attribute.Int("email.daily_reminder.hour", w.cfg.Email.DailyReminder.Hour),
			attribute.Bool("email.enabled", w.cfg.Email.Enabled),
		),
	)
	defer span.End()

	if !w.cfg.Email.DailyReminder.Enabled {
		w.logger.Info(ctx, "Daily reminders disabled, skipping", nil)
		return nil
	}

	// Get current time in UTC
	now := w.timeNow().UTC()
	currentHour := now.Hour()

	// Check if it's time to send reminders (default: 9 AM)
	reminderHour := w.cfg.Email.DailyReminder.Hour
	if currentHour != reminderHour {
		span.SetAttributes(
			attribute.Int("check.current_hour", currentHour),
			attribute.Int("check.reminder_hour", reminderHour),
			attribute.Bool("check.should_send", false),
			attribute.String("check.reason", "wrong_hour"),
		)
		return nil
	}

	span.SetAttributes(
		attribute.Int("check.current_hour", currentHour),
		attribute.Int("check.reminder_hour", reminderHour),
		attribute.Bool("check.should_send", true),
	)

	w.logger.Info(ctx, "Checking for users needing daily reminders", map[string]interface{}{
		"reminder_hour": reminderHour,
	})

	// Get users who need daily reminders
	users, err := w.getUsersNeedingDailyReminders(ctx)
	if err != nil {
		span.RecordError(err)
		span.SetAttributes(
			attribute.Int("users.total", 0),
			attribute.Int("users.eligible", 0),
			attribute.Int("reminders.sent", 0),
		)
		w.logger.Error(ctx, "Failed to get users needing daily reminders", err, nil)
		return contextutils.WrapError(err, "failed to get users needing daily reminders")
	}

	span.SetAttributes(
		attribute.Int("users.total", len(users)),
	)

	remindersSent := 0
	failedReminders := 0

	for _, user := range users {
		// Record the sent notification
		subject := "Time for your daily quiz! 🧠"
		status := "sent"
		errorMsg := ""

		if err := w.emailService.SendDailyReminder(ctx, &user); err != nil {
			failedReminders++
			status = "failed"
			errorMsg = err.Error()
			w.logger.Error(ctx, "Failed to send daily reminder", err, map[string]interface{}{
				"user_id": user.ID,
				"email":   user.Email.String,
			})
		} else {
			remindersSent++
		}

		// Record the sent notification in the database
		if err := w.emailService.RecordSentNotification(ctx, user.ID, "daily_reminder", subject, "daily_reminder", status, errorMsg); err != nil {
			w.logger.Error(ctx, "Failed to record sent notification", err, map[string]interface{}{
				"user_id": user.ID,
			})
		}

		// Update the last reminder sent timestamp for this user
		if err := w.learningService.UpdateLastDailyReminderSent(ctx, user.ID); err != nil {
			w.logger.Error(ctx, "Failed to update last daily reminder sent timestamp", err, map[string]interface{}{
				"user_id": user.ID,
			})
			// Don't count this as a failed reminder since the email was sent successfully
		}
	}

	span.SetAttributes(
		attribute.Int("users.eligible", len(users)),
		attribute.Int("reminders.sent", remindersSent),
		attribute.Int("reminders.failed", failedReminders),
		attribute.Float64("reminders.success_rate", float64(remindersSent)/float64(len(users))),
	)

	w.logger.Info(ctx, "Daily reminders processed", map[string]interface{}{
		"total_users":    len(users),
		"reminders_sent": remindersSent,
		"reminder_hour":  reminderHour,
	})

	return nil
}

// getUsersNeedingDailyReminders returns users who should receive daily reminders
func (w *Worker) getUsersNeedingDailyReminders(ctx context.Context) ([]models.User, error) {
	ctx, span := otel.Tracer("worker").Start(ctx, "getUsersNeedingDailyReminders")
	defer span.End()

	// Get all users and filter for those with email addresses and daily reminders enabled
	users, err := w.userService.GetAllUsers(ctx)
	if err != nil {
		span.RecordError(err)
		return nil, contextutils.WrapError(err, "failed to get users")
	}

	var eligibleUsers []models.User
	today := w.timeNow().UTC().Format("2006-01-02")

	for _, user := range users {
		// Check if user has email address
		if !user.Email.Valid || user.Email.String == "" {
			continue
		}

		// Get user's learning preferences to check daily reminder setting
		prefs, err := w.learningService.GetUserLearningPreferences(ctx, user.ID)
		if err != nil {
			w.logger.Warn(ctx, "Failed to get user learning preferences for daily reminder check", map[string]interface{}{
				"user_id":  user.ID,
				"username": user.Username,
				"error":    err.Error(),
			})
			continue
		}

		// Check if daily reminders are enabled for this user
		if prefs == nil || !prefs.DailyReminderEnabled {
			continue
		}

		// Check if we've already sent a reminder today
		if prefs.LastDailyReminderSent != nil {
			lastReminderDate := prefs.LastDailyReminderSent.Format("2006-01-02")
			if lastReminderDate == today {
				continue
			}
		}

		eligibleUsers = append(eligibleUsers, user)
	}

	w.logger.Info(ctx, "Found users eligible for daily reminders", map[string]interface{}{
		"total_users":    len(users),
		"eligible_users": len(eligibleUsers),
	})

	return eligibleUsers, nil
}

// checkForDailyQuestionAssignments assigns daily questions to all eligible users
// This runs independently of email reminders to ensure users get daily questions
// even if they have email reminders disabled
func (w *Worker) checkForDailyQuestionAssignments(ctx context.Context) error {
	ctx, span := observability.TraceWorkerFunction(ctx, "check_for_daily_question_assignments",
		attribute.String("worker.instance", w.instance),
	)
	defer observability.FinishSpan(span, nil)

	w.logger.Info(ctx, "Checking for daily question assignments", map[string]interface{}{
		"instance": w.instance,
	})

	// Get users who are eligible for daily questions
	users, err := w.getUsersEligibleForDailyQuestions(ctx)
	if err != nil {
		span.RecordError(err)
		w.logger.Error(ctx, "Failed to get users eligible for daily questions", err, nil)
		return contextutils.WrapError(err, "failed to get users eligible for daily questions")
	}

	if len(users) == 0 {
		w.logger.Info(ctx, "No users eligible for daily question assignments", map[string]interface{}{
			"instance": w.instance,
		})
		return nil
	}

	span.SetAttributes(
		attribute.Int("users.total", len(users)),
	)

	successfulAssignments := 0
	failedAssignments := 0

	for _, user := range users {
		// Get user's timezone, default to UTC if not set
		timezone := "UTC"
		if user.Timezone.Valid && user.Timezone.String != "" {
			timezone = user.Timezone.String
		}

		// Get today's date in the user's timezone
		loc, err := time.LoadLocation(timezone)
		if err != nil {
			w.logger.Warn(ctx, "Invalid timezone for user, using UTC", map[string]interface{}{
				"user_id":  user.ID,
				"username": user.Username,
				"timezone": timezone,
				"error":    err.Error(),
			})
			loc = time.UTC
		}

		// Get today's date in the user's timezone
		now := w.timeNow().In(loc)
		today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)

		// Assign daily questions for dates in [today .. today+N]
		horizon := w.workerCfg.DailyHorizonDays
		if horizon <= 0 {
			// default to 2 days ahead when misconfigured or not set
			horizon = 2
		}

		// Ensure the worker horizon covers the configured avoid window so
		// that when future assignments are removed (e.g., after a correct
		// submission) the worker run will top up missing slots. Use server
		// config as the source of truth for the avoid window.
		avoidDays := 7
		if w.cfg != nil && w.cfg.Server.DailyRepeatAvoidDays > 0 {
			avoidDays = w.cfg.Server.DailyRepeatAvoidDays
		}
		if horizon < avoidDays {
			w.logger.Info(ctx, "Extending worker daily horizon to cover daily repeat avoid window", map[string]interface{}{
				"old_horizon": horizon,
				"new_horizon": avoidDays,
				"user_id":     user.ID,
			})
			horizon = avoidDays
		}
		for d := 0; d <= horizon; d++ {
			target := today.AddDate(0, 0, d)
			// Assign daily questions for target date in user's timezone
			if err := w.dailyQuestionService.AssignDailyQuestions(ctx, user.ID, target); err != nil {
				failedAssignments++
				w.logger.Error(ctx, "Failed to assign daily questions", err, map[string]interface{}{
					"user_id":  user.ID,
					"username": user.Username,
					"timezone": timezone,
					"date":     target.Format("2006-01-02"),
				})
			} else {
				successfulAssignments++
				w.logger.Info(ctx, "Successfully assigned daily questions", map[string]interface{}{
					"user_id":  user.ID,
					"username": user.Username,
					"timezone": timezone,
					"date":     target.Format("2006-01-02"),
				})
			}
		}
	}

	span.SetAttributes(
		attribute.Int("assignments.successful", successfulAssignments),
		attribute.Int("assignments.failed", failedAssignments),
	)

	w.logger.Info(ctx, "Completed daily question assignment check", map[string]interface{}{
		"instance":               w.instance,
		"eligible_users":         len(users),
		"successful_assignments": successfulAssignments,
		"failed_assignments":     failedAssignments,
	})

	return nil
}

// getUsersEligibleForDailyQuestions returns users who should receive daily questions
// This is independent of email reminder preferences
func (w *Worker) getUsersEligibleForDailyQuestions(ctx context.Context) ([]models.User, error) {
	ctx, span := otel.Tracer("worker").Start(ctx, "getUsersEligibleForDailyQuestions")
	defer span.End()

	// Get all users
	users, err := w.userService.GetAllUsers(ctx)
	if err != nil {
		span.RecordError(err)
		return nil, contextutils.WrapError(err, "failed to get users")
	}

	var eligibleUsers []models.User

	for _, user := range users {
		// Check if user has language and level preferences set
		if !user.PreferredLanguage.Valid || user.PreferredLanguage.String == "" {
			w.logger.Debug(ctx, "User missing preferred language, skipping daily question assignment", map[string]interface{}{
				"user_id":  user.ID,
				"username": user.Username,
			})
			continue
		}

		if !user.CurrentLevel.Valid || user.CurrentLevel.String == "" {
			w.logger.Debug(ctx, "User missing current level, skipping daily question assignment", map[string]interface{}{
				"user_id":  user.ID,
				"username": user.Username,
			})
			continue
		}

		eligibleUsers = append(eligibleUsers, user)
	}

	w.logger.Info(ctx, "Found users eligible for daily questions", map[string]interface{}{
		"total_users":    len(users),
		"eligible_users": len(eligibleUsers),
	})

	return eligibleUsers, nil
}

// NewWorker creates a new Worker instance
func NewWorker(userService services.UserServiceInterface, questionService services.QuestionServiceInterface, aiService services.AIServiceInterface, learningService services.LearningServiceInterface, workerService services.WorkerServiceInterface, dailyQuestionService services.DailyQuestionServiceInterface, emailService mailer.Mailer, hintService services.GenerationHintServiceInterface, instance string, cfg *config.Config, logger *observability.Logger) *Worker {
	if instance == "" {
		instance = "default"
	}

	ctx, cancel := context.WithCancel(context.Background())

	// Prefer value from config file when set (>0). If not set, default to 1.
	dailyHorizon := cfg.Server.DailyHorizonDays
	if dailyHorizon <= 0 {
		dailyHorizon = 1
	}

	w := &Worker{
		userService:          userService,
		questionService:      questionService,
		aiService:            aiService,
		learningService:      learningService,
		workerService:        workerService,
		dailyQuestionService: dailyQuestionService,
		emailService:         emailService,
		hintService:          hintService,
		instance:             instance,
		status:               Status{IsRunning: false, CurrentActivity: "Initialized"},
		history:              make([]RunRecord, 0, cfg.Server.MaxHistory),
		activityLogs:         make([]ActivityLog, 0, cfg.Server.MaxActivityLogs),
		manualTrigger:        make(chan bool, 1),
		cfg:                  cfg,
		workerCfg:            Config{StartWorkerPaused: getEnvBool("WORKER_START_PAUSED", false), DailyHorizonDays: dailyHorizon},
		logger:               logger,
		userFailures:         make(map[int]*UserFailureInfo),
		timeNow:              time.Now, // Default to real time
	}

	// Handle startup pause if configured
	if w.workerCfg.StartWorkerPaused {
		w.handleStartupPause(ctx)
	}

	// Store cancel function for cleanup
	w.cancel = cancel

	return w
}

// getEnvBool is a helper function to get boolean environment variables
func getEnvBool(key string, defaultValue bool) bool {
	valStr := os.Getenv(key)
	if valStr == "" {
		return defaultValue
	}
	val, err := strconv.ParseBool(valStr)
	if err != nil {
		return defaultValue
	}
	return val
}

// Start begins the worker's background processing loop
func (w *Worker) Start(ctx context.Context) {
	w.status.IsRunning = true
	w.updateDatabaseStatus(ctx)

	w.handleStartupPause(ctx)

	// Start heartbeat goroutine
	go w.heartbeatLoop(ctx)

	// Main worker loop
	ticker := time.NewTicker(config.WorkerHeartbeatInterval) // Check every 30 seconds
	defer ticker.Stop()

	initialStatus := w.getInitialWorkerStatus(ctx)

	w.logger.Info(ctx, "Worker started", map[string]interface{}{
		"instance": w.instance,
		"status":   initialStatus,
	})
	w.logActivity(ctx, "INFO", fmt.Sprintf("Worker %s started (%s)", w.instance, initialStatus), nil, nil)

	for {
		select {
		case <-ctx.Done():
			w.logger.Info(ctx, "Worker shutting down", map[string]interface{}{
				"instance": w.instance,
			})
			w.logActivity(ctx, "INFO", fmt.Sprintf("Worker %s shutting down", w.instance), nil, nil)
			w.status.IsRunning = false
			w.updateDatabaseStatus(ctx)
			return

		case <-ticker.C:
			w.run()

		case <-w.manualTrigger:
			w.logger.Info(ctx, "Worker triggered manually", map[string]interface{}{
				"instance": w.instance,
			})
			w.logActivity(ctx, "INFO", fmt.Sprintf("Worker %s triggered manually", w.instance), nil, nil)
			w.run()
		}
	}
}

// handleStartupPause sets global pause if configured
func (w *Worker) handleStartupPause(ctx context.Context) {
	if w.workerCfg.StartWorkerPaused {
		w.logger.Info(ctx, "Worker configured to start paused - setting global pause", map[string]interface{}{
			"instance": w.instance,
		})
		if err := w.workerService.SetGlobalPause(ctx, true); err != nil {
			w.logger.Error(ctx, "Failed to set global pause on startup", err, map[string]interface{}{
				"instance": w.instance,
			})
		} else {
			w.logger.Info(ctx, "Global pause set on startup as configured", map[string]interface{}{
				"instance": w.instance,
			})
		}
	}
}

// getInitialWorkerStatus determines the initial status string
func (w *Worker) getInitialWorkerStatus(ctx context.Context) string {
	initialStatus := "running"
	globalPaused, err := w.workerService.IsGlobalPaused(ctx)
	if err != nil {
		w.logger.Error(ctx, "Failed to check global pause status on startup", err, map[string]interface{}{
			"instance": w.instance,
		})
	} else if globalPaused {
		initialStatus = "paused (globally)"
	} else {
		status, err := w.workerService.GetWorkerStatus(ctx, w.instance)
		if err != nil {
			// Worker status not found is expected on first startup - this is normal
			w.logger.Debug(ctx, "Worker status not found on startup (expected for new worker)", map[string]interface{}{
				"instance": w.instance,
			})
		} else if status != nil && status.IsPaused {
			initialStatus = "paused (instance)"
		}
	}
	return initialStatus
}

func (w *Worker) heartbeatLoop(ctx context.Context) {
	ticker := time.NewTicker(config.WorkerHeartbeatInterval) // Heartbeat every 30 seconds
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.updateHeartbeat(ctx)
		}
	}
}

// updateHeartbeat updates the heartbeat in the database
func (w *Worker) updateHeartbeat(ctx context.Context) {
	if err := w.workerService.UpdateHeartbeat(ctx, w.instance); err != nil {
		w.logger.Error(ctx, "Failed to update heartbeat for worker", err, map[string]interface{}{
			"instance": w.instance,
		})
	}
}

// run executes a single worker cycle
func (w *Worker) run() {
	ctx, span := observability.TraceWorkerFunction(context.Background(), "run",
		attribute.String("worker.instance", w.instance),
	)
	defer observability.FinishSpan(span, nil)

	// Ensure worker status is up to date before checking pause status
	w.updateDatabaseStatus(ctx)

	paused, reason := w.checkPauseStatus(ctx)
	if paused {
		span.SetAttributes(attribute.String("pause_reason", reason))
		w.updateActivity(reason)
		return
	}

	w.status.LastRunStart = time.Now()
	w.updateDatabaseStatus(ctx)
	details, err := w.generateNeededQuestions(ctx)

	// Assign daily questions to all eligible users (independent of email reminders)
	if err := w.checkForDailyQuestionAssignments(ctx); err != nil {
		w.logger.Error(ctx, "Failed to check daily question assignments", err, map[string]interface{}{
			"instance": w.instance,
		})
	}

	// Check for daily email reminders
	if err := w.checkForDailyReminders(ctx); err != nil {
		w.logger.Error(ctx, "Failed to check daily reminders", err, map[string]interface{}{
			"instance": w.instance,
		})
	}

	w.status.LastRunFinish = time.Now()
	if err != nil {
		w.status.LastRunError = err.Error()
		w.logger.Error(ctx, "Worker run failed", err, map[string]interface{}{
			"instance": w.instance,
		})
	} else {
		w.status.LastRunError = ""
	}

	w.recordRunHistory(details, err)
	w.updateDatabaseStatus(ctx)
}

// checkPauseStatus checks global and instance pause
func (w *Worker) checkPauseStatus(ctx context.Context) (bool, string) {
	globalPaused, err := w.workerService.IsGlobalPaused(ctx)
	if err != nil {
		w.logger.Error(ctx, "Failed to check global pause status", err, map[string]interface{}{
			"instance": w.instance,
		})
		return true, "Error checking global pause status"
	}
	if globalPaused {
		return true, "Globally paused"
	}
	status, err := w.workerService.GetWorkerStatus(ctx, w.instance)
	if err != nil {
		// Worker status not found might happen during startup - assume not paused
		w.logger.Debug(ctx, "Worker status not found during pause check (assuming not paused)", map[string]interface{}{
			"instance": w.instance,
		})
		return false, ""
	} else if status != nil && status.IsPaused {
		return true, "Worker instance paused"
	}
	return false, ""
}

// recordRunHistory records the run in history and trims the slice
func (w *Worker) recordRunHistory(details string, err error) {
	record := RunRecord{
		StartTime: w.status.LastRunStart,
		EndTime:   w.status.LastRunFinish,
		Duration:  w.status.LastRunFinish.Sub(w.status.LastRunStart),
		Details:   details,
	}
	if err != nil {
		record.Status = "Failure"
	} else {
		record.Status = "Success"
	}
	w.mu.Lock()
	w.history = append(w.history, record)
	if len(w.history) > w.cfg.Server.MaxHistory {
		w.history = w.history[len(w.history)-w.cfg.Server.MaxHistory:]
	}
	w.mu.Unlock()
}

// GetStatus returns the current worker status
func (w *Worker) GetStatus() Status {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.status
}

// GetHistory returns the worker's run history
func (w *Worker) GetHistory() []RunRecord {
	w.mu.RLock()
	defer w.mu.RUnlock()
	// Return a copy to avoid race conditions
	history := make([]RunRecord, len(w.history))
	copy(history, w.history)
	return history
}

// GetActivityLogs returns recent activity logs
func (w *Worker) GetActivityLogs() []ActivityLog {
	w.mu.RLock()
	defer w.mu.RUnlock()

	// Return a copy to avoid concurrent access issues
	logs := make([]ActivityLog, len(w.activityLogs))
	copy(logs, w.activityLogs)
	return logs
}

// GetInstance returns the worker instance name
func (w *Worker) GetInstance() string {
	return w.instance
}

// GetEmailService returns the email service
func (w *Worker) GetEmailService() mailer.Mailer {
	return w.emailService
}

// TriggerManualRun triggers a manual worker run
func (w *Worker) TriggerManualRun() {
	ctx := context.Background()
	select {
	case w.manualTrigger <- true:
		w.logger.Info(ctx, "Manual trigger sent to worker", map[string]interface{}{
			"instance": w.instance,
		})
	default:
		w.logger.Info(ctx, "Manual trigger already pending for worker", map[string]interface{}{
			"instance": w.instance,
		})
	}
}

// Pause pauses the worker
func (w *Worker) Pause(ctx context.Context) {
	if err := w.workerService.PauseWorker(ctx, w.instance); err != nil {
		w.logger.Warn(ctx, "Failed to pause worker in service", map[string]interface{}{
			"instance": w.instance,
			"error":    err.Error(),
		})
	}
	w.logger.Info(ctx, "Worker paused", map[string]interface{}{
		"instance": w.instance,
	})
	w.logActivity(ctx, "INFO", fmt.Sprintf("Worker %s paused", w.instance), nil, nil)
	w.status.IsPaused = true
	w.updateDatabaseStatus(ctx)
}

// Resume resumes the worker
func (w *Worker) Resume(ctx context.Context) {
	if err := w.workerService.ResumeWorker(ctx, w.instance); err != nil {
		w.logger.Warn(ctx, "Failed to resume worker in service", map[string]interface{}{
			"instance": w.instance,
			"error":    err.Error(),
		})
		// Do not unpause if resume failed
		w.updateDatabaseStatus(ctx)
		return
	}
	w.logger.Info(ctx, "Worker resumed", map[string]interface{}{
		"instance": w.instance,
	})
	w.logActivity(ctx, "INFO", fmt.Sprintf("Worker %s resumed", w.instance), nil, nil)
	w.status.IsPaused = false
	w.updateDatabaseStatus(ctx)
}

// Shutdown gracefully shuts down the worker and cleans up resources
func (w *Worker) Shutdown(ctx context.Context) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.logger.Info(ctx, "Worker starting shutdown", map[string]interface{}{
		"instance": w.instance,
	})

	// Cancel the shutdown context to signal shutdown
	if w.cancel != nil {
		w.cancel()
	}

	// Wait for any active operations to complete
	// This is a simple implementation - in a more complex system,
	// you might want to track active operations more precisely
	time.Sleep(config.WorkerSleepDuration)

	// Clean up user failures map
	w.failureMu.Lock()
	w.userFailures = make(map[int]*UserFailureInfo)
	w.failureMu.Unlock()

	// Clear activity logs
	w.activityLogs = make([]ActivityLog, 0)

	w.logger.Info(ctx, "Worker shutdown completed", map[string]interface{}{
		"instance": w.instance,
	})
	return nil
}

// updateDatabaseStatus updates the worker status in the database
func (w *Worker) updateDatabaseStatus(ctx context.Context) {
	dbStatus := &models.WorkerStatus{
		WorkerInstance:          w.instance,
		IsRunning:               w.status.IsRunning,
		IsPaused:                w.status.IsPaused,
		CurrentActivity:         sql.NullString{String: w.status.CurrentActivity, Valid: w.status.CurrentActivity != ""},
		LastHeartbeat:           sql.NullTime{Time: time.Now(), Valid: true},
		LastRunStart:            sql.NullTime{Time: w.status.LastRunStart, Valid: !w.status.LastRunStart.IsZero()},
		LastRunFinish:           sql.NullTime{Time: w.status.LastRunFinish, Valid: !w.status.LastRunFinish.IsZero()},
		LastRunError:            sql.NullString{String: w.status.LastRunError, Valid: w.status.LastRunError != ""},
		TotalQuestionsGenerated: w.getTotalQuestionsGenerated(),
		TotalRuns:               len(w.history),
	}

	if err := w.workerService.UpdateWorkerStatus(ctx, w.instance, dbStatus); err != nil {
		w.logger.Error(ctx, "Failed to update worker status in database", err, map[string]interface{}{
			"instance": w.instance,
		})
	}
}

// getTotalQuestionsGenerated calculates total questions generated from run history
func (w *Worker) getTotalQuestionsGenerated() int {
	total := 0
	for _, record := range w.history {
		if record.Status == "Success" {
			// Parse details to count questions - simplified for now
			total++ // This would need to be enhanced to parse actual count
		}
	}
	return total
}

func (w *Worker) generateNeededQuestions(ctx context.Context) (result0 string, err error) {
	ctx, span := observability.TraceWorkerFunction(ctx, "generate_needed_questions",
		attribute.String("worker.instance", w.instance),
	)
	defer observability.FinishSpan(span, &err)

	// Check if globally paused BEFORE any work or logging
	globalPaused, err := w.workerService.IsGlobalPaused(ctx)
	if err != nil {
		span.RecordError(err)
		w.logger.Error(ctx, "Failed to check global pause status", err, map[string]interface{}{
			"instance": w.instance,
		})
		return "Error checking global pause status", err
	}
	if globalPaused {
		span.SetAttributes(attribute.Bool("globally_paused", true))
		w.logger.Info(ctx, "Worker skipping question generation (globally paused)", map[string]interface{}{
			"instance": w.instance,
		})
		return "Run paused globally", nil
	}

	aiUsers, err := w.getEligibleAIUsers(ctx)
	if err != nil {
		return "Error getting users", err
	}
	if len(aiUsers) == 0 {
		w.logger.Info(ctx, "Worker: No active users with AI provider configuration found for question generation", map[string]interface{}{
			"instance": w.instance,
		})
		return "No active users with AI provider configuration found", nil
	}

	var actions []string
	var checkedUsers []string
	var actuallyProcessedUsers []string
	var hadAttemptedOperations bool
	var hadFailures bool

	for _, user := range aiUsers {
		checkedUsers = append(checkedUsers, user.Username)
		shouldProcess, skipReason := w.shouldProcessUser(ctx, &user)
		if !shouldProcess {
			if skipReason != "" {
				w.logger.Info(ctx, "Worker user check", map[string]interface{}{
					"instance": w.instance,
					"username": user.Username,
					"reason":   skipReason,
				})
			}
			continue
		}
		actuallyProcessedUsers = append(actuallyProcessedUsers, user.Username)
		userActions, attempted, failed := w.processUserQuestionGeneration(ctx, &user)
		if attempted {
			hadAttemptedOperations = true
		}
		if failed {
			hadFailures = true
		}
		if userActions != "" {
			actions = append(actions, userActions)
		}
		w.logger.Info(ctx, "Worker completed check for user", map[string]interface{}{
			"instance": w.instance,
			"username": user.Username,
		})
	}

	w.updateActivity("")
	return w.summarizeRunActions(actions, checkedUsers, actuallyProcessedUsers, hadAttemptedOperations, hadFailures), nil
}

// getEligibleAIUsers returns users eligible for AI question generation
func (w *Worker) getEligibleAIUsers(ctx context.Context) (result0 []models.User, err error) {
	ctx, span := observability.TraceWorkerFunction(ctx, "get_eligible_ai_users",
		attribute.String("worker.instance", w.instance),
	)
	defer observability.FinishSpan(span, &err)

	users, err := w.userService.GetAllUsers(ctx)
	if err != nil {
		span.RecordError(err)
		return nil, err
	}
	var aiUsers []models.User
	for _, user := range users {
		if !user.AIEnabled.Valid || !user.AIEnabled.Bool {
			continue
		}
		userPaused, err := w.workerService.IsUserPaused(ctx, user.ID)
		if err == nil && userPaused {
			continue
		}
		hasAIProvider := user.AIProvider.Valid && user.AIProvider.String != ""
		hasAPIKey := false
		if hasAIProvider {
			savedKey, err := w.userService.GetUserAPIKey(ctx, user.ID, user.AIProvider.String)
			if err == nil && savedKey != "" {
				hasAPIKey = true
			}
		}
		if hasAPIKey || hasAIProvider {
			aiUsers = append(aiUsers, user)
		}
	}
	return aiUsers, nil
}

// shouldProcessUser encapsulates exponential backoff and pause checks
func (w *Worker) shouldProcessUser(ctx context.Context, user *models.User) (bool, string) {
	if !w.shouldRetryUser(user.ID) {
		w.failureMu.RLock()
		failure := w.userFailures[user.ID]
		nextRetry := time.Until(failure.NextRetryTime)
		w.failureMu.RUnlock()
		return false, fmt.Sprintf("Skipping due to exponential backoff (failure #%d, retry in %v)", failure.ConsecutiveFailures, nextRetry.Round(time.Second))
	}
	globalPaused, err := w.workerService.IsGlobalPaused(ctx)
	if err != nil {
		return false, "Error checking global pause status"
	}
	if globalPaused {
		return false, "Run paused globally"
	}
	status, err := w.workerService.GetWorkerStatus(ctx, w.instance)
	if err == nil && status != nil && status.IsPaused {
		return false, fmt.Sprintf("Worker instance %s paused", w.instance)
	}
	if ctx.Err() != nil {
		return false, "Shutdown initiated"
	}
	return true, ""
}

// Helper: get the count of eligible questions for a user (excludes questions answered correctly in the last 2 days)
func (w *Worker) getEligibleQuestionCount(ctx context.Context, userID int, language, level string, qType models.QuestionType) (result0 int, err error) {
	ctx, span := observability.TraceWorkerFunction(ctx, "get_eligible_question_count",
		observability.AttributeUserID(userID),
		attribute.String("language", language),
		attribute.String("level", level),
		attribute.String("question.type", string(qType)),
		attribute.String("worker.instance", w.instance),
	)
	defer observability.FinishSpan(span, &err)

	// Safe user lookup: tests may not wire userService
	userLookup := func(ctx context.Context, id int) (*models.User, error) {
		// Only use the concrete UserService implementation to avoid invoking mocks in unit tests
		if us, ok := w.userService.(*services.UserService); ok && us != nil {
			return us.GetUserByID(ctx, id)
		}
		// No userService available or not concrete - return nil so helper falls back to UTC
		return nil, nil
	}

	// Determine user-local 2-day window and pass UTC timestamps to query
	startUTC, endUTC, _, err := contextutils.UserLocalDayRange(ctx, userID, 2, userLookup)
	if err != nil {
		return 0, contextutils.WrapError(err, "failed to compute user local day range")
	}

	query := `
		SELECT COUNT(*)
		FROM questions q
		JOIN user_questions uq ON q.id = uq.question_id
		WHERE uq.user_id = $1
		  AND q.language = $2
		  AND q.level = $3
		  AND q.type = $4
		  AND q.status = 'active'
		  AND NOT EXISTS (
		        SELECT 1 FROM user_responses ur
		        WHERE ur.user_id = $1
		          AND ur.question_id = q.id
		          AND ur.is_correct = TRUE
		          AND ur.created_at >= $5 AND ur.created_at < $6
		  )
	`

	// Try to get the database from the question service
	var db *sql.DB
	if qs, ok := w.questionService.(*services.QuestionService); ok {
		db = qs.DB()
	} else {
		// For mock services or other implementations, we can't get the DB directly
		// This is expected in unit tests
		return 0, contextutils.ErrorWithContextf("cannot get database from question service implementation")
	}

	row := db.QueryRowContext(ctx, query, userID, language, level, qType, startUTC, endUTC)
	var count int
	if err := row.Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

func (w *Worker) processUserQuestionGeneration(ctx context.Context, user *models.User) (string, bool, bool) {
	ctx, span := observability.TraceWorkerFunction(ctx, "processUserQuestionGeneration",
		observability.AttributeUserID(user.ID),
		attribute.String("user.username", user.Username),
		attribute.String("worker.instance", w.instance),
	)
	defer observability.FinishSpan(span, nil)

	userLanguage := "italian"
	if user.PreferredLanguage.Valid && user.PreferredLanguage.String != "" {
		userLanguage = user.PreferredLanguage.String
		span.SetAttributes(attribute.String("user.language", userLanguage))
	}
	userLevel := "A1"
	if user.CurrentLevel.Valid && user.CurrentLevel.String != "" {
		userLevel = user.CurrentLevel.String
		span.SetAttributes(attribute.String("user.level", userLevel))
	}
	languages := []string{userLanguage}
	levels := []string{userLevel}
	questionTypes := []models.QuestionType{
		models.Vocabulary,
		models.FillInBlank,
		models.QuestionAnswer,
		models.ReadingComprehension,
	}

	// Reorder types based on active generation hints (hinted types first, stable order)
	if w.hintService != nil {
		if hints, err := w.hintService.GetActiveHintsForUser(ctx, user.ID); err == nil && len(hints) > 0 {
			hinted := make([]models.QuestionType, 0, len(hints))
			hintedSet := map[models.QuestionType]bool{}
			for _, h := range hints {
				qt := models.QuestionType(h.QuestionType)
				hinted = append(hinted, qt)
				hintedSet[qt] = true
			}
			rest := make([]models.QuestionType, 0, len(questionTypes))
			for _, qt := range questionTypes {
				if !hintedSet[qt] {
					rest = append(rest, qt)
				}
			}
			questionTypes = append(hinted, rest...)
		}
	}
	var actions []string
	var hadAttemptedOperations bool
	var hadFailures bool
	for _, language := range languages {
		for _, level := range levels {
			for _, qType := range questionTypes {
				activity := fmt.Sprintf("Checking questions for user %s: %s %s %s", user.Username, language, level, qType)
				w.updateActivity(activity)
				// Use eligible question count (not just total assigned)
				eligibleCount, err := w.getEligibleQuestionCount(ctx, user.ID, language, level, qType)
				if err != nil {
					span.RecordError(err)
					hadFailures = true
					continue // Continue to next question type
				}
				// If hinted, be more aggressive about generating for that type
				hinted := false
				if w.hintService != nil {
					if hints, err := w.hintService.GetActiveHintsForUser(ctx, user.ID); err == nil {
						for _, h := range hints {
							if models.QuestionType(h.QuestionType) == qType {
								hinted = true
								break
							}
						}
					}
				}

				refillThreshold := w.cfg.Server.QuestionRefillThreshold
				if hinted {
					// Treat as if pool is empty to trigger generation, but keep batch sizing logic
					eligibleCount = 0
				}

				if eligibleCount < refillThreshold {
					provider := "default"
					if user.AIProvider.Valid && user.AIProvider.String != "" {
						provider = user.AIProvider.String
					}
					// Base batch size from AI provider
					needed := w.aiService.GetQuestionBatchSize(provider)

					// Get user's learning preferences to use their personal FreshQuestionRatio
					userPrefs, prefsErr := w.learningService.GetUserLearningPreferences(ctx, user.ID)
					userFreshRatio := 0.7 // default fallback
					if prefsErr == nil && userPrefs != nil && userPrefs.FreshQuestionRatio > 0 {
						userFreshRatio = userPrefs.FreshQuestionRatio
					} else if prefsErr != nil {
						w.logger.Warn(ctx, "Failed to get user learning preferences, using default fresh ratio", map[string]interface{}{
							"user_id": user.ID,
							"error":   prefsErr.Error(),
						})
					}

					// Ensure at least enough fresh questions are available to meet the user's personal FreshQuestionRatio.
					// This ensures daily question assignment can respect the user's freshness preference.
					desiredFresh := int(math.Ceil(float64(refillThreshold) * userFreshRatio))
					freshCandidates := 0
					if qs, qerr := w.questionService.GetAdaptiveQuestionsForDaily(ctx, user.ID, language, level, 50); qerr == nil && qs != nil {
						for _, q := range qs {
							if q != nil && q.TotalResponses == 0 {
								freshCandidates++
							}
						}
					} else if qerr != nil {
						// Log but don't fail - we'll conservatively proceed with base batch size
						w.logger.Warn(ctx, "Failed to fetch adaptive questions for fresh-count check", map[string]interface{}{
							"user_id": user.ID,
							"error":   qerr.Error(),
						})
					}

					if missing := desiredFresh - freshCandidates; missing > 0 {
						needed += missing
						w.logger.Info(ctx, "Adjusting generation batch to meet user's personal fresh-question requirement", map[string]interface{}{
							"user_id":          user.ID,
							"language":         language,
							"level":            level,
							"question_type":    qType,
							"user_fresh_ratio": userFreshRatio,
							"base_batch_size":  w.aiService.GetQuestionBatchSize(provider),
							"desired_fresh":    desiredFresh,
							"fresh_candidates": freshCandidates,
							"added_to_batch":   missing,
							"final_batch_size": needed,
						})
					}
					hadAttemptedOperations = true
					action, err := w.GenerateQuestionsForUser(ctx, user, language, level, qType, needed, "")
					if err != nil {
						hadFailures = true
						// Continue to next question type instead of breaking all loops
						continue
					}
					if action != "" {
						actions = append(actions, action)
					}
					// Clear hint on successful generation attempt for this type
					if hinted && w.hintService != nil {
						_ = w.hintService.ClearHint(ctx, user.ID, language, level, qType)
					}
				}
			}
		}
	}
	return strings.Join(actions, "; "), hadAttemptedOperations, hadFailures
}

// summarizeRunActions builds the summary string for actions taken
func (w *Worker) summarizeRunActions(actions, checkedUsers, actuallyProcessedUsers []string, hadAttemptedOperations, hadFailures bool) string {
	userList := "No users with AI configuration found"
	if len(checkedUsers) > 0 {
		userList = fmt.Sprintf("Checked users: %s", strings.Join(checkedUsers, ", "))
	}
	if len(actions) == 0 {
		if len(actuallyProcessedUsers) == 0 {
			return fmt.Sprintf("No actions taken. All users in exponential backoff. %s", userList)
		}
		if hadAttemptedOperations && hadFailures && len(actions) == 0 {
			return fmt.Sprintf("No actions taken due to errors. %s", userList)
		}
		return fmt.Sprintf("No actions taken. All question types have sufficient questions. %s", userList)
	}
	userList = fmt.Sprintf("Processed users: %s", strings.Join(actuallyProcessedUsers, ", "))

	// Format actions with line breaks for better readability in UI
	if len(actions) == 1 {
		return fmt.Sprintf("%s\n%s", actions[0], userList)
	}

	formattedActions := strings.Join(actions, "\n")
	return fmt.Sprintf("%s\n%s", formattedActions, userList)
}

// GenerateQuestionsForUser generates questions for a specific user with the given parameters
func (w *Worker) GenerateQuestionsForUser(ctx context.Context, user *models.User, language, level string, qType models.QuestionType, count int, topic string) (result0 string, err error) {
	ctx, span := observability.TraceWorkerFunction(ctx, "generate_questions_for_user",
		observability.AttributeUserID(user.ID),
		attribute.String("user.username", user.Username),
		attribute.String("language", language),
		attribute.String("level", level),
		attribute.String("question.type", string(qType)),
		attribute.Int("question.count", count),
		attribute.String("topic", topic),
		attribute.String("worker.instance", w.instance),
	)
	defer observability.FinishSpan(span, &err)

	if count <= 0 {
		return "No questions needed", nil
	}

	// Gather priority data for variety selection
	priorityData := w.getPriorityGenerationData(ctx, user.ID, language, level, qType)
	var userWeakAreas []string
	if priorityData != nil && priorityData.FocusOnWeakAreas {
		userWeakAreas = priorityData.UserWeakAreas
	}
	var highPriorityTopics []string
	if priorityData != nil {
		highPriorityTopics = priorityData.HighPriorityTopics
	}
	var gapAnalysis map[string]int
	if priorityData != nil {
		gapAnalysis = priorityData.GapAnalysis
	}

	variety := w.aiService.VarietyService().SelectVarietyElements(ctx, level, highPriorityTopics, userWeakAreas, gapAnalysis)

	// Log priority generation decisions
	generationReasoning := w.getGenerationReasoning(priorityData, variety)

	var freshQuestionRatio float64
	if priorityData != nil {
		freshQuestionRatio = priorityData.FreshQuestionRatio
	}

	priorityLog := PriorityGenerationLog{
		UserID:              user.ID,
		Username:            user.Username,
		Language:            language,
		Level:               level,
		QuestionType:        string(qType),
		FocusOnWeakAreas:    priorityData != nil && priorityData.FocusOnWeakAreas,
		UserWeakAreas:       userWeakAreas,
		HighPriorityTopics:  highPriorityTopics,
		GapAnalysis:         gapAnalysis,
		FreshQuestionRatio:  freshQuestionRatio,
		SelectedVariety:     variety,
		GenerationReasoning: generationReasoning,
		Timestamp:           time.Now(),
	}
	w.logPriorityGeneration(ctx, priorityLog)

	aiReq, recentQuestions, err := w.buildAIQuestionGenRequest(ctx, user, language, level, qType, count, topic)
	if err != nil {
		w.logger.Warn(ctx, "Worker failed to get recent questions", map[string]interface{}{
			"instance": w.instance,
			"error":    err.Error(),
		})
		recentQuestions = []string{}
	}
	aiReq.RecentQuestionHistory = recentQuestions

	userConfig := w.getUserAIConfig(ctx, user)

	batchLogMsg := formatBatchLogMessage(user.Username, count, string(qType), language, level, variety, userConfig.Provider, userConfig.Model)
	w.logger.Info(ctx, batchLogMsg, map[string]interface{}{
		"instance": w.instance,
	})
	w.updateActivity(batchLogMsg)
	w.logActivity(ctx, "INFO", batchLogMsg, &user.ID, &user.Username)

	progressMsg, questions, errAI := w.handleAIQuestionStream(ctx, userConfig, aiReq, variety, count, language, level, qType, topic, user)

	if errAI != nil {
		w.recordUserFailure(ctx, user.ID, user.Username)
		return progressMsg, errAI
	}
	if len(questions) == 0 {
		w.recordUserFailure(ctx, user.ID, user.Username)
		return progressMsg, contextutils.WrapErrorf(contextutils.ErrAIResponseInvalid, "AI service returned 0 questions for %s %s %s", language, level, qType)
	}

	savedCount := w.saveGeneratedQuestions(ctx, user, questions, language, level, qType, topic, variety)

	if savedCount > 0 {
		w.recordUserSuccess(ctx, user.ID, user.Username)
	}
	if savedCount != len(questions) {
		w.recordUserFailure(ctx, user.ID, user.Username)
		return fmt.Sprintf("Generated %d/%d %s questions for %s %s", savedCount, len(questions), qType, language, level),
			contextutils.WrapErrorf(contextutils.ErrDatabaseQuery, "only saved %d out of %d generated questions", savedCount, len(questions))
	}
	return fmt.Sprintf("Generated %d %s questions for %s %s", savedCount, qType, language, level), nil
}

// buildAIQuestionGenRequest prepares the AI request and gets recent questions
func (w *Worker) buildAIQuestionGenRequest(ctx context.Context, user *models.User, language, level string, qType models.QuestionType, count int, _ string) (result0 *models.AIQuestionGenRequest, result1 []string, err error) {
	ctx, span := observability.TraceWorkerFunction(ctx, "build_ai_question_gen_request",
		observability.AttributeUserID(user.ID),
		attribute.String("user.username", user.Username),
		attribute.String("language", language),
		attribute.String("level", level),
		attribute.String("question.type", string(qType)),
		attribute.Int("question.count", count),
		attribute.String("worker.instance", w.instance),
	)
	defer observability.FinishSpan(span, &err)

	recentQuestions, err := w.questionService.GetRecentQuestionContentsForUser(ctx, user.ID, 10)
	if err != nil {
		span.RecordError(err)
		return nil, nil, err
	}
	aiReq := &models.AIQuestionGenRequest{
		Language:     language,
		Level:        level,
		QuestionType: qType,
		Count:        count,
	}

	aiReq.RecentQuestionHistory = recentQuestions

	return aiReq, recentQuestions, nil
}

// getUserAIConfig builds the UserAIConfig struct with API key
func (w *Worker) getUserAIConfig(ctx context.Context, user *models.User) *services.UserAIConfig {
	ctx, span := observability.TraceWorkerFunction(ctx, "get_user_ai_config",
		observability.AttributeUserID(user.ID),
		attribute.String("user.username", user.Username),
		attribute.String("worker.instance", w.instance),
	)
	defer observability.FinishSpan(span, nil)

	provider := ""
	if user.AIProvider.Valid {
		provider = user.AIProvider.String
		span.SetAttributes(attribute.String("ai.provider", provider))
	}
	model := ""
	if user.AIModel.Valid {
		model = user.AIModel.String
		span.SetAttributes(attribute.String("ai.model", model))
	}
	apiKey := ""
	if provider != "" {
		savedKey, err := w.userService.GetUserAPIKey(ctx, user.ID, provider)
		if err == nil && savedKey != "" {
			apiKey = savedKey
		}
	}
	return &services.UserAIConfig{
		Provider: provider,
		Model:    model,
		APIKey:   apiKey,
		Username: user.Username,
	}
}

// handleAIQuestionStream handles the AI streaming and collects questions
func (w *Worker) handleAIQuestionStream(ctx context.Context, userConfig *services.UserAIConfig, req *models.AIQuestionGenRequest, variety *services.VarietyElements, count int, language, level string, qType models.QuestionType, topic string, user *models.User) (result0 string, result1 []*models.Question, err error) {
	ctx, span := observability.TraceWorkerFunction(ctx, "handle_ai_question_stream",
		attribute.String("ai.provider", userConfig.Provider),
		attribute.String("ai.model", userConfig.Model),
		attribute.String("language", language),
		attribute.String("level", level),
		attribute.String("question.type", string(qType)),
		attribute.Int("question.count", count),
		attribute.String("topic", topic),
		attribute.String("user.username", user.Username),
		attribute.String("worker.instance", w.instance),
	)
	defer observability.FinishSpan(span, &err)

	progressChan := make(chan *models.Question)
	var questions []*models.Question
	var wg sync.WaitGroup
	var errAI error
	progressMsg := ""
	wg.Add(1)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				w.logger.Error(ctx, "Panic in AI question stream goroutine", nil, map[string]interface{}{
					"instance": w.instance,
					"panic":    fmt.Sprintf("%v", r),
				})
			}
			wg.Done()
		}()
		errAI = w.aiService.GenerateQuestionsStream(ctx, userConfig, req, progressChan, variety)
	}()
	generatedCount := 0
	for question := range progressChan {
		generatedCount++
		progressMsg = fmt.Sprintf("Generated %d/%d %s questions for %s %s", generatedCount, count, qType, language, level)
		if topic != "" {
			progressMsg = fmt.Sprintf("Generated %d/%d %s questions for %s %s (topic: %s)", generatedCount, count, qType, language, level, topic)
		}
		w.logger.Info(ctx, progressMsg, map[string]interface{}{
			"instance": w.instance,
		})
		w.updateActivity(progressMsg)
		w.logActivity(ctx, "INFO", progressMsg, &user.ID, &user.Username)
		questions = append(questions, question)
	}
	wg.Wait()
	return progressMsg, questions, errAI
}

// saveGeneratedQuestions saves questions to the DB and returns the count
func (w *Worker) saveGeneratedQuestions(ctx context.Context, user *models.User, questions []*models.Question, language, level string, qType models.QuestionType, topic string, variety *services.VarietyElements) int {
	ctx, span := observability.TraceWorkerFunction(ctx, "save_generated_questions",
		observability.AttributeUserID(user.ID),
		attribute.String("user.username", user.Username),
		attribute.String("language", language),
		attribute.String("level", level),
		attribute.String("question.type", string(qType)),
		attribute.Int("question.count", len(questions)),
		attribute.String("topic", topic),
		attribute.String("worker.instance", w.instance),
	)
	defer observability.FinishSpan(span, nil)

	savingMsg := fmt.Sprintf("Saving %d new %s questions for %s %s", len(questions), qType, language, level)
	if topic != "" {
		savingMsg = fmt.Sprintf("Saving %d new %s questions for %s %s (topic: %s)", len(questions), qType, language, level, topic)
	}
	w.logger.Info(ctx, savingMsg, map[string]interface{}{
		"instance": w.instance,
	})
	w.updateActivity(savingMsg)
	w.logActivity(ctx, "INFO", savingMsg, &user.ID, &user.Username)
	savedCount := 0
	for _, q := range questions {
		// Populate variety fields from the variety elements used during generation
		if variety != nil {
			q.TopicCategory = variety.TopicCategory
			q.GrammarFocus = variety.GrammarFocus
			q.VocabularyDomain = variety.VocabularyDomain
			q.Scenario = variety.Scenario
			q.StyleModifier = variety.StyleModifier
			q.DifficultyModifier = variety.DifficultyModifier
			q.TimeContext = variety.TimeContext
		}
		if err := w.questionService.SaveQuestion(ctx, q); err != nil {
			w.logger.Error(ctx, "Failed to save generated question", err, map[string]interface{}{
				"instance":      w.instance,
				"user_id":       user.ID,
				"language":      language,
				"level":         level,
				"question_type": qType,
			})
		} else {
			// Assign the question to the user after saving
			if err := w.questionService.AssignQuestionToUser(ctx, q.ID, user.ID); err != nil {
				w.logger.Error(ctx, "Failed to assign question to user", err, map[string]interface{}{
					"instance":    w.instance,
					"question_id": q.ID,
					"user_id":     user.ID,
				})
			} else {
				savedCount++
			}
		}
	}
	if savedCount > 0 {
		successMsg := fmt.Sprintf("Successfully saved %d new '%s' questions for %s %s", savedCount, qType, language, level)
		if topic != "" {
			successMsg = fmt.Sprintf("Successfully saved %d new '%s' questions for %s %s (topic: %s)", savedCount, qType, language, level, topic)
		}
		w.logActivity(ctx, "INFO", successMsg, &user.ID, &user.Username)
	}
	return savedCount
}

func (w *Worker) updateActivity(activity string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.status.CurrentActivity = activity
}

// logActivity adds an activity log entry
func (w *Worker) logActivity(_ context.Context, _, message string, userID *int, username *string) {
	w.mu.Lock()
	defer w.mu.Unlock()

	logEntry := ActivityLog{
		Timestamp: time.Now(),
		Level:     "INFO",
		Message:   message,
		UserID:    userID,
		Username:  username,
	}

	// Add to activity logs (circular buffer)
	w.activityLogs = append(w.activityLogs, logEntry)

	// Keep only the last maxActivityLogs entries
	if len(w.activityLogs) > w.cfg.Server.MaxActivityLogs {
		w.activityLogs = w.activityLogs[len(w.activityLogs)-w.cfg.Server.MaxActivityLogs:]
	}
}

// shouldRetryUser checks if enough time has passed since the last failure for exponential backoff
func (w *Worker) shouldRetryUser(userID int) bool {
	w.failureMu.RLock()
	defer w.failureMu.RUnlock()

	failure, exists := w.userFailures[userID]
	if !exists {
		return true // No previous failures, go ahead
	}

	return time.Now().After(failure.NextRetryTime)
}

// recordUserFailure records a failure and calculates the next retry time with exponential backoff
func (w *Worker) recordUserFailure(ctx context.Context, userID int, username string) {
	ctx, span := observability.TraceWorkerFunction(ctx, "record_user_failure",
		observability.AttributeUserID(userID),
		attribute.String("user.username", username),
		attribute.String("worker.instance", w.instance),
	)
	defer observability.FinishSpan(span, nil)

	w.failureMu.Lock()
	defer w.failureMu.Unlock()

	failure, exists := w.userFailures[userID]
	if !exists {
		failure = &UserFailureInfo{}
		w.userFailures[userID] = failure
	}

	failure.ConsecutiveFailures++
	failure.LastFailureTime = time.Now()

	// Exponential backoff: 2^failures seconds, max 1 hour
	backoffSeconds := int(math.Pow(2, float64(failure.ConsecutiveFailures)))
	if backoffSeconds > 3600 {
		backoffSeconds = 3600
	}
	failure.NextRetryTime = time.Now().Add(time.Duration(backoffSeconds) * time.Second)

	span.SetAttributes(
		attribute.Int("failure.count", failure.ConsecutiveFailures),
		attribute.Int("backoff.seconds", backoffSeconds),
	)

	w.logger.Info(ctx, "Worker recorded user failure", map[string]interface{}{
		"instance":           w.instance,
		"username":           username,
		"failure_count":      failure.ConsecutiveFailures,
		"next_retry_seconds": backoffSeconds,
	})
}

// recordUserSuccess clears the failure count for a user
func (w *Worker) recordUserSuccess(ctx context.Context, userID int, username string) {
	ctx, span := observability.TraceWorkerFunction(ctx, "record_user_success",
		observability.AttributeUserID(userID),
		attribute.String("user.username", username),
		attribute.String("worker.instance", w.instance),
	)
	defer observability.FinishSpan(span, nil)

	w.failureMu.Lock()
	defer w.failureMu.Unlock()

	failure, exists := w.userFailures[userID]
	if exists && failure.ConsecutiveFailures > 0 {
		span.SetAttributes(attribute.Int("previous_failures", failure.ConsecutiveFailures))
		w.logger.Info(ctx, "Worker user success after failures, resetting backoff", map[string]interface{}{
			"instance":          w.instance,
			"username":          username,
			"previous_failures": failure.ConsecutiveFailures,
		})
		delete(w.userFailures, userID)
	}
}

// formatBatchLogMessage creates a formatted log message for batch question generation
func formatBatchLogMessage(username string, count int, qType, language, level string, variety *services.VarietyElements, provider, model string) string {
	var summaryFields []string
	if variety != nil {
		if variety.GrammarFocus != "" {
			summaryFields = append(summaryFields, "grammar: "+variety.GrammarFocus)
		}
		if variety.TopicCategory != "" {
			summaryFields = append(summaryFields, "topic: "+variety.TopicCategory)
		}
		if variety.Scenario != "" {
			summaryFields = append(summaryFields, "scenario: "+variety.Scenario)
		}
		if variety.StyleModifier != "" {
			summaryFields = append(summaryFields, "style: "+variety.StyleModifier)
		}
		if variety.DifficultyModifier != "" {
			summaryFields = append(summaryFields, "difficulty: "+variety.DifficultyModifier)
		}
		if variety.VocabularyDomain != "" {
			summaryFields = append(summaryFields, "vocab: "+variety.VocabularyDomain)
		}
		if variety.TimeContext != "" {
			summaryFields = append(summaryFields, "time: "+variety.TimeContext)
		}
	}
	providerModel := "provider: " + provider + ", model: " + model
	if len(summaryFields) > 0 {
		summaryFields = append(summaryFields, providerModel)
	} else {
		summaryFields = []string{providerModel}
	}
	return fmt.Sprintf("Worker [user=%s]: Batch %d %s questions (lang: %s, level: %s) | %s", username, count, qType, language, level, strings.Join(summaryFields, " | "))
}

// PriorityGenerationData contains priority information to guide AI question generation
type PriorityGenerationData struct {
	UserWeakAreas        []string                        `json:"user_weak_areas,omitempty"`
	HighPriorityTopics   []string                        `json:"high_priority_topics,omitempty"`
	GapAnalysis          map[string]int                  `json:"gap_analysis,omitempty"`
	UserPreferences      *models.UserLearningPreferences `json:"user_preferences,omitempty"`
	PriorityDistribution map[string]int                  `json:"priority_distribution,omitempty"`
	FocusOnWeakAreas     bool                            `json:"focus_on_weak_areas"`
	FreshQuestionRatio   float64                         `json:"fresh_question_ratio"`
}

// PriorityGenerationLog contains structured data about priority-aware generation decisions
type PriorityGenerationLog struct {
	UserID              int                       `json:"user_id"`
	Username            string                    `json:"username"`
	Language            string                    `json:"language"`
	Level               string                    `json:"level"`
	QuestionType        string                    `json:"question_type"`
	FocusOnWeakAreas    bool                      `json:"focus_on_weak_areas"`
	UserWeakAreas       []string                  `json:"user_weak_areas,omitempty"`
	HighPriorityTopics  []string                  `json:"high_priority_topics,omitempty"`
	GapAnalysis         map[string]int            `json:"gap_analysis,omitempty"`
	FreshQuestionRatio  float64                   `json:"fresh_question_ratio"`
	SelectedVariety     *services.VarietyElements `json:"selected_variety"`
	GenerationReasoning string                    `json:"generation_reasoning"`
	Timestamp           time.Time                 `json:"timestamp"`
}

// logPriorityGeneration logs priority generation data as JSON
func (w *Worker) logPriorityGeneration(ctx context.Context, priorityLog PriorityGenerationLog) {
	ctx, span := observability.TraceWorkerFunction(ctx, "log_priority_generation",
		observability.AttributeUserID(priorityLog.UserID),
		attribute.String("user.username", priorityLog.Username),
		attribute.String("language", priorityLog.Language),
		attribute.String("level", priorityLog.Level),
		attribute.String("question.type", priorityLog.QuestionType),
		attribute.String("worker.instance", w.instance),
	)
	defer observability.FinishSpan(span, nil)

	logJSON, err := json.Marshal(priorityLog)
	if err != nil {
		span.RecordError(err)
		w.logger.Error(ctx, "Failed to marshal priority generation log", err, map[string]interface{}{
			"instance": w.instance,
		})
		return
	}
	w.logger.Info(ctx, "Worker priority generation", map[string]interface{}{
		"instance": w.instance,
		"data":     string(logJSON),
	})
}

// getGenerationReasoning provides a human-readable explanation of the generation strategy
func (w *Worker) getGenerationReasoning(priorityData *PriorityGenerationData, variety *services.VarietyElements) string {
	if priorityData == nil {
		return "standard generation"
	}
	var reasons []string

	if priorityData.FocusOnWeakAreas && len(priorityData.UserWeakAreas) > 0 {
		reasons = append(reasons, fmt.Sprintf("focusing on weak areas: %s", strings.Join(priorityData.UserWeakAreas, ", ")))
	}

	if len(priorityData.HighPriorityTopics) > 0 {
		reasons = append(reasons, fmt.Sprintf("high priority topics: %s", strings.Join(priorityData.HighPriorityTopics, ", ")))
	}

	if len(priorityData.GapAnalysis) > 0 {
		var gaps []string
		for topic, count := range priorityData.GapAnalysis {
			gaps = append(gaps, fmt.Sprintf("%s(%d)", topic, count))
		}
		reasons = append(reasons, fmt.Sprintf("gap analysis: %s", strings.Join(gaps, ", ")))
	}

	if priorityData.FreshQuestionRatio > 0 {
		reasons = append(reasons, fmt.Sprintf("fresh ratio: %.1f%%", priorityData.FreshQuestionRatio*100))
	}

	if variety != nil {
		var varietyElements []string
		if variety.TopicCategory != "" {
			varietyElements = append(varietyElements, fmt.Sprintf("topic:%s", variety.TopicCategory))
		}
		if variety.GrammarFocus != "" {
			varietyElements = append(varietyElements, fmt.Sprintf("grammar:%s", variety.GrammarFocus))
		}
		if variety.VocabularyDomain != "" {
			varietyElements = append(varietyElements, fmt.Sprintf("vocab:%s", variety.VocabularyDomain))
		}
		if variety.Scenario != "" {
			varietyElements = append(varietyElements, fmt.Sprintf("scenario:%s", variety.Scenario))
		}
		if len(varietyElements) > 0 {
			reasons = append(reasons, fmt.Sprintf("variety: %s", strings.Join(varietyElements, ", ")))
		}
	}

	if len(reasons) == 0 {
		return "standard generation"
	}

	return strings.Join(reasons, "; ")
}

// getPriorityGenerationData gathers priority data for AI question generation
func (w *Worker) getPriorityGenerationData(ctx context.Context, userID int, language, level string, questionType models.QuestionType) *PriorityGenerationData {
	// Get user preferences
	prefs, err := w.learningService.GetUserLearningPreferences(ctx, userID)
	if err != nil {
		w.logger.Warn(ctx, "Worker failed to get user preferences", map[string]interface{}{
			"instance": w.instance,
			"user_id":  userID,
			"error":    err.Error(),
		})
		prefs = w.getDefaultLearningPreferences()
	}

	// Get weak areas
	weakAreas, err := w.learningService.GetUserWeakAreas(ctx, userID, 5)
	if err != nil {
		w.logger.Warn(ctx, "Worker failed to get weak areas", map[string]interface{}{
			"instance": w.instance,
			"user_id":  userID,
			"error":    err.Error(),
		})
		weakAreas = []map[string]interface{}{}
	}

	// Convert weak areas to topic strings
	var weakAreaTopics []string
	for _, area := range weakAreas {
		if topic, ok := area["topic"].(string); ok && topic != "" {
			weakAreaTopics = append(weakAreaTopics, topic)
		}
	}

	// Get high priority topics
	highPriorityTopics, err := w.getHighPriorityTopics(ctx, userID, language, level, questionType)
	if err != nil {
		w.logger.Warn(ctx, "Worker failed to get high priority topics", map[string]interface{}{
			"instance": w.instance,
			"user_id":  userID,
			"error":    err.Error(),
		})
		highPriorityTopics = []string{}
	}

	// Get gap analysis
	gapAnalysis, err := w.getGapAnalysis(ctx, userID, language, level, questionType)
	if err != nil {
		w.logger.Warn(ctx, "Worker failed to get gap analysis", map[string]interface{}{
			"instance": w.instance,
			"user_id":  userID,
			"error":    err.Error(),
		})
		gapAnalysis = map[string]int{}
	}

	// Get priority distribution
	priorityDistribution, err := w.getPriorityDistribution(ctx, userID, language, level, questionType)
	if err != nil {
		w.logger.Warn(ctx, "Worker failed to get priority distribution", map[string]interface{}{
			"instance": w.instance,
			"user_id":  userID,
			"error":    err.Error(),
		})
		priorityDistribution = map[string]int{}
	}

	// Determine if we should focus on weak areas
	focusOnWeakAreas := len(weakAreaTopics) > 0 && prefs != nil && prefs.FocusOnWeakAreas

	return &PriorityGenerationData{
		UserWeakAreas:        weakAreaTopics,
		HighPriorityTopics:   highPriorityTopics,
		GapAnalysis:          gapAnalysis,
		UserPreferences:      prefs,
		PriorityDistribution: priorityDistribution,
		FocusOnWeakAreas:     focusOnWeakAreas,
		FreshQuestionRatio:   prefs.FreshQuestionRatio,
	}
}

// getDefaultLearningPreferences returns default learning preferences
func (w *Worker) getDefaultLearningPreferences() *models.UserLearningPreferences {
	return &models.UserLearningPreferences{
		FocusOnWeakAreas:   false,
		FreshQuestionRatio: 0.3,
		WeakAreaBoost:      1.5,
	}
}

// getHighPriorityTopics returns topics that have high average priority scores
func (w *Worker) getHighPriorityTopics(ctx context.Context, userID int, language, level string, questionType models.QuestionType) (result0 []string, err error) {
	return w.workerService.GetHighPriorityTopics(ctx, userID, language, level, string(questionType))
}

// getGapAnalysis identifies areas with insufficient questions available
func (w *Worker) getGapAnalysis(ctx context.Context, userID int, language, level string, questionType models.QuestionType) (result0 map[string]int, err error) {
	return w.workerService.GetGapAnalysis(ctx, userID, language, level, string(questionType))
}

// getPriorityDistribution returns the distribution of priority scores
func (w *Worker) getPriorityDistribution(ctx context.Context, userID int, language, level string, questionType models.QuestionType) (result0 map[string]int, err error) {
	return w.workerService.GetPriorityDistribution(ctx, userID, language, level, string(questionType))
}
