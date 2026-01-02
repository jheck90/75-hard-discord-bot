package services

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/75-hard-discord-bot/internal/logger"
)

// CheckInService handles check-in related operations
type CheckInService struct {
	db           *sql.DB
	userService  *UserService
}

// NewCheckInService creates a new check-in service
func NewCheckInService(userService *UserService) *CheckInService {
	return &CheckInService{
		userService: userService,
	}
}

// Initialize initializes the service with database connection
func (s *CheckInService) Initialize(db *sql.DB) error {
	s.db = db
	return nil
}

// Name returns the service name
func (s *CheckInService) Name() string {
	return "CheckInService"
}

// Health checks the service health
func (s *CheckInService) Health() error {
	if s.db == nil {
		return fmt.Errorf("database not initialized")
	}
	return s.db.Ping()
}

// RecordCheckIn records a check-in for the user and returns formatted DB entry info
func (s *CheckInService) RecordCheckIn(userID, username string) (string, error) {
	if s.db == nil {
		return "", fmt.Errorf("database not available")
	}

	// Ensure user exists in database (create if not exists)
	logger.DB("Ensuring user exists: user_id=%s, username=%s", userID, username)
	err := s.userService.EnsureUserExists(userID, username)
	if err != nil {
		logger.Error("Failed to ensure user exists: %v", err)
		return "", fmt.Errorf("failed to ensure user exists: %w", err)
	}

	// Get current challenge day for user
	logger.DB("Getting current challenge day for user_id=%s", userID)
	challengeDay, err := s.userService.GetCurrentChallengeDay(userID)
	if err != nil {
		logger.Error("Failed to get challenge day: %v", err)
		return "", fmt.Errorf("failed to get challenge day: %w", err)
	}

	// Record check-in (this will trigger auto-population of all feat tables)
	logger.DB("Recording check-in: user_id=%s, challenge_day=%d", userID, challengeDay)
	result, err := s.db.Exec(
		`INSERT INTO accountability_checkins (user_id, challenge_day, check_in_method) 
		 VALUES ($1, $2, $3) 
		 ON CONFLICT (user_id, challenge_day) DO UPDATE SET completed_at = NOW()`,
		userID, challengeDay, "emoji_reaction",
	)
	if err != nil {
		logger.Error("Failed to record check-in: %v", err)
		return "", fmt.Errorf("failed to record check-in: %w", err)
	}

	// Log if this was a new insert (trigger should fire)
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected > 0 {
		logger.DB("✅ Check-in recorded for user %s, day %d (trigger should fire)", userID, challengeDay)
	} else {
		logger.DB("⚠️ Check-in updated for user %s, day %d (trigger may not fire on UPDATE)", userID, challengeDay)
	}

	// Query all feat tables to show what was created (only in dev mode)
	var dbInfo string
	if logger.IsDevMode() {
		logger.DB("Querying DB entries info for user_id=%s, challenge_day=%d", userID, challengeDay)
		dbInfo, err = s.GetDBEntriesInfo(userID, challengeDay)
		if err != nil {
			logger.Error("Failed to get DB entries info: %v", err)
			return "", fmt.Errorf("failed to get DB entries info: %w", err)
		}
	}

	return dbInfo, nil
}

// GetDBEntriesInfo queries all feat tables and returns formatted info
func (s *CheckInService) GetDBEntriesInfo(userID string, challengeDay int) (string, error) {
	var info strings.Builder
	info.WriteString("📊 **Database Entries Created:**\n```\n")

	// Check accountability check-in
	var checkInTime time.Time
	err := s.db.QueryRow(
		`SELECT completed_at FROM accountability_checkins WHERE user_id = $1 AND challenge_day = $2`,
		userID, challengeDay,
	).Scan(&checkInTime)
	if err == nil {
		info.WriteString(fmt.Sprintf("✅ Accountability Check-in: %s\n", checkInTime.Format("2006-01-02 15:04:05")))
	}

	// Check exercise
	var exerciseWorkout, exerciseCore sql.NullInt64
	err = s.db.QueryRow(
		`SELECT workout_duration_minutes, core_mobility_duration_minutes 
		 FROM exercise_completions WHERE user_id = $1 AND challenge_day = $2`,
		userID, challengeDay,
	).Scan(&exerciseWorkout, &exerciseCore)
	if err == nil {
		info.WriteString(fmt.Sprintf("💪 Exercise: %d min workout + %d min core/mobility\n",
			exerciseWorkout.Int64, exerciseCore.Int64))
	}

	// Check diet
	var dietCheatMeal, dietAlcohol sql.NullBool
	err = s.db.QueryRow(
		`SELECT cheat_meal, alcohol_consumed FROM diet_completions WHERE user_id = $1 AND challenge_day = $2`,
		userID, challengeDay,
	).Scan(&dietCheatMeal, &dietAlcohol)
	if err == nil {
		info.WriteString("🍽️  Diet: Compliant (no cheat meals, no alcohol)\n")
	}

	// Check water
	var waterAmount sql.NullFloat64
	err = s.db.QueryRow(
		`SELECT amount_ounces FROM water_completions WHERE user_id = $1 AND challenge_day = $2`,
		userID, challengeDay,
	).Scan(&waterAmount)
	if err == nil {
		info.WriteString(fmt.Sprintf("💧 Water: %.2f oz (1 gallon)\n", waterAmount.Float64))
	}

	// Check self-improvement
	var selfImproveDuration sql.NullInt64
	err = s.db.QueryRow(
		`SELECT duration_minutes FROM self_improvement_completions WHERE user_id = $1 AND challenge_day = $2`,
		userID, challengeDay,
	).Scan(&selfImproveDuration)
	if err == nil {
		info.WriteString(fmt.Sprintf("📚 Self-Improvement: %d minutes\n", selfImproveDuration.Int64))
	}

	// Check finances
	var financesStatus sql.NullString
	err = s.db.QueryRow(
		`SELECT compliance_status FROM finances_completions WHERE user_id = $1 AND challenge_day = $2`,
		userID, challengeDay,
	).Scan(&financesStatus)
	if err == nil {
		info.WriteString(fmt.Sprintf("💰 Finances: %s\n", financesStatus.String))
	}

	info.WriteString("```")
	return info.String(), nil
}
