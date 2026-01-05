package services

import (
	"database/sql"
	"fmt"

	"github.com/75-hard-discord-bot/internal/logger"
)

// WaterService handles water intake tracking operations
type WaterService struct {
	db          *sql.DB
	userService *UserService
}

// NewWaterService creates a new water service
func NewWaterService(userService *UserService) *WaterService {
	return &WaterService{
		userService: userService,
	}
}

// Initialize initializes the service with database connection
func (s *WaterService) Initialize(db *sql.DB) error {
	s.db = db
	return nil
}

// Name returns the service name
func (s *WaterService) Name() string {
	return "WaterService"
}

// Health checks the service health
func (s *WaterService) Health() error {
	if s.db == nil {
		return fmt.Errorf("database not initialized")
	}
	return s.db.Ping()
}

// AddWater adds water intake for the user
func (s *WaterService) AddWater(userID, username string, ounces float64) (float64, float64, error) {
	if s.db == nil {
		return 0, 0, fmt.Errorf("database not available")
	}

	if ounces <= 0 {
		return 0, 0, fmt.Errorf("ounces must be greater than 0")
	}

	// Ensure user exists
	err := s.userService.EnsureUserExists(userID, username)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to ensure user exists: %w", err)
	}

	// Get current challenge day
	challengeDay, err := s.userService.GetCurrentChallengeDay(userID)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get challenge day: %w", err)
	}

	// Get current water amount for today
	var currentAmount sql.NullFloat64
	err = s.db.QueryRow(
		`SELECT amount_ounces FROM water_completions WHERE user_id = $1 AND challenge_day = $2`,
		userID, challengeDay,
	).Scan(&currentAmount)

	currentTotal := 0.0
	if err == nil && currentAmount.Valid {
		currentTotal = currentAmount.Float64
	} else if err != sql.ErrNoRows {
		logger.Error("Failed to query current water amount: %v", err)
		return 0, 0, fmt.Errorf("failed to query current water amount: %w", err)
	}

	// Calculate new total (cap at 128oz)
	newTotal := currentTotal + ounces
	if newTotal > 128.0 {
		newTotal = 128.0
		ounces = 128.0 - currentTotal // Only add what fits
	}

	// Insert or update water completion
	logger.DB("Adding water: user_id=%s, challenge_day=%d, adding=%.2f oz, new_total=%.2f oz", userID, challengeDay, ounces, newTotal)
	if currentTotal == 0 {
		// Insert new record
		_, err = s.db.Exec(
			`INSERT INTO water_completions (user_id, challenge_day, amount_ounces, is_plain_water, completed_at)
			 VALUES ($1, $2, $3, true, NOW())`,
			userID, challengeDay, newTotal,
		)
	} else {
		// Update existing record
		_, err = s.db.Exec(
			`UPDATE water_completions 
			 SET amount_ounces = LEAST(amount_ounces + $3, 128.0),
			     completed_at = NOW()
			 WHERE user_id = $1 AND challenge_day = $2`,
			userID, challengeDay, ounces,
		)
	}
	if err != nil {
		logger.Error("Failed to add water: %v", err)
		return 0, 0, fmt.Errorf("failed to add water: %w", err)
	}

	logger.DB("Successfully added water for user_id=%s, challenge_day=%d, total=%.2f oz", userID, challengeDay, newTotal)
	return ounces, newTotal, nil
}

// SubtractWater subtracts water intake for the user
func (s *WaterService) SubtractWater(userID, username string, ounces float64) (float64, float64, error) {
	if s.db == nil {
		return 0, 0, fmt.Errorf("database not available")
	}

	if ounces <= 0 {
		return 0, 0, fmt.Errorf("ounces must be greater than 0")
	}

	// Ensure user exists
	err := s.userService.EnsureUserExists(userID, username)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to ensure user exists: %w", err)
	}

	// Get current challenge day
	challengeDay, err := s.userService.GetCurrentChallengeDay(userID)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get challenge day: %w", err)
	}

	// Get current water amount for today
	var currentAmount sql.NullFloat64
	err = s.db.QueryRow(
		`SELECT amount_ounces FROM water_completions WHERE user_id = $1 AND challenge_day = $2`,
		userID, challengeDay,
	).Scan(&currentAmount)

	currentTotal := 0.0
	if err == nil && currentAmount.Valid {
		currentTotal = currentAmount.Float64
	} else if err != sql.ErrNoRows {
		logger.Error("Failed to query current water amount: %v", err)
		return 0, 0, fmt.Errorf("failed to query current water amount: %w", err)
	}

	// Calculate new total (can't go below 0)
	newTotal := currentTotal - ounces
	if newTotal < 0 {
		newTotal = 0
		ounces = currentTotal // Only subtract what exists
	}

	// Update water completion
	logger.DB("Subtracting water: user_id=%s, challenge_day=%d, subtracting=%.2f oz, new_total=%.2f oz", userID, challengeDay, ounces, newTotal)
	_, err = s.db.Exec(
		`UPDATE water_completions 
		 SET amount_ounces = GREATEST(amount_ounces - $3, 0),
		     completed_at = NOW()
		 WHERE user_id = $1 AND challenge_day = $2`,
		userID, challengeDay, ounces,
	)
	if err != nil {
		logger.Error("Failed to subtract water: %v", err)
		return 0, 0, fmt.Errorf("failed to subtract water: %w", err)
	}

	logger.DB("Successfully subtracted water for user_id=%s, challenge_day=%d, total=%.2f oz", userID, challengeDay, newTotal)
	return ounces, newTotal, nil
}

// GetWaterIntake gets the current water intake for the user today
func (s *WaterService) GetWaterIntake(userID string) (float64, error) {
	if s.db == nil {
		return 0, fmt.Errorf("database not available")
	}

	// Get current challenge day
	challengeDay, err := s.userService.GetCurrentChallengeDay(userID)
	if err != nil {
		return 0, fmt.Errorf("failed to get challenge day: %w", err)
	}

	var amount sql.NullFloat64
	err = s.db.QueryRow(
		`SELECT amount_ounces FROM water_completions WHERE user_id = $1 AND challenge_day = $2`,
		userID, challengeDay,
	).Scan(&amount)

	if err == sql.ErrNoRows {
		return 0, nil // No water logged yet today
	}
	if err != nil {
		logger.Error("Failed to get water intake: %v", err)
		return 0, fmt.Errorf("failed to get water intake: %w", err)
	}

	if amount.Valid {
		return amount.Float64, nil
	}
	return 0, nil
}
