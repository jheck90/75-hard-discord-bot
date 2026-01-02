package services

import (
	"database/sql"
	"fmt"

	"github.com/75-hard-discord-bot/internal/logger"
)

// ExerciseService handles exercise-related operations
type ExerciseService struct {
	db          *sql.DB
	userService *UserService
}

// NewExerciseService creates a new exercise service
func NewExerciseService(userService *UserService) *ExerciseService {
	return &ExerciseService{
		userService: userService,
	}
}

// Initialize initializes the service with database connection
func (s *ExerciseService) Initialize(db *sql.DB) error {
	s.db = db
	return nil
}

// Name returns the service name
func (s *ExerciseService) Name() string {
	return "ExerciseService"
}

// Health checks the service health
func (s *ExerciseService) Health() error {
	if s.db == nil {
		return fmt.Errorf("database not initialized")
	}
	return s.db.Ping()
}

// LogExerciseQuick logs exercise with default values
func (s *ExerciseService) LogExerciseQuick(userID, username string) error {
	return s.LogExerciseDetailed(userID, username, 30, "general", "indoor", 10, "general")
}

// LogExerciseDetailed logs exercise with provided details
func (s *ExerciseService) LogExerciseDetailed(userID, username string, workoutDuration int, workoutType, workoutLocation string, coreDuration int, coreType string) error {
	if s.db == nil {
		return fmt.Errorf("database not available")
	}

	// Ensure user exists
	err := s.userService.EnsureUserExists(userID, username)
	if err != nil {
		return fmt.Errorf("failed to ensure user exists: %w", err)
	}

	// Get current challenge day
	challengeDay, err := s.userService.GetCurrentChallengeDay(userID)
	if err != nil {
		return fmt.Errorf("failed to get challenge day: %w", err)
	}

	// Insert or update exercise completion (mark as manual entry)
	logger.DB("Logging exercise: user_id=%s, challenge_day=%d, workout=%dmin, core=%dmin", userID, challengeDay, workoutDuration, coreDuration)
	_, err = s.db.Exec(
		`INSERT INTO exercise_completions 
		 (user_id, challenge_day, workout_duration_minutes, workout_type, workout_location, core_mobility_duration_minutes, core_mobility_type, autopopulated)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, false)
		 ON CONFLICT (user_id, challenge_day) 
		 DO UPDATE SET 
			workout_duration_minutes = EXCLUDED.workout_duration_minutes,
			workout_type = EXCLUDED.workout_type,
			workout_location = EXCLUDED.workout_location,
			core_mobility_duration_minutes = EXCLUDED.core_mobility_duration_minutes,
			core_mobility_type = EXCLUDED.core_mobility_type,
			autopopulated = false,
			completed_at = NOW()`,
		userID, challengeDay, workoutDuration, workoutType, workoutLocation, coreDuration, coreType,
	)
	if err != nil {
		logger.Error("Failed to log exercise: %v", err)
	} else {
		logger.DB("Successfully logged exercise for user_id=%s, challenge_day=%d", userID, challengeDay)
	}
	return err
}
