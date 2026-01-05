package services

import (
	"database/sql"
	"fmt"

	"github.com/75-hard-discord-bot/internal/logger"
)

// WeighInService handles weigh-in related operations
type WeighInService struct {
	db          *sql.DB
	userService *UserService
}

// NewWeighInService creates a new weigh-in service
func NewWeighInService(userService *UserService) *WeighInService {
	return &WeighInService{
		userService: userService,
	}
}

// Initialize initializes the service with database connection
func (s *WeighInService) Initialize(db *sql.DB) error {
	s.db = db
	return nil
}

// Name returns the service name
func (s *WeighInService) Name() string {
	return "WeighInService"
}

// Health checks the service health
func (s *WeighInService) Health() error {
	if s.db == nil {
		return fmt.Errorf("database not initialized")
	}
	return s.db.Ping()
}

// RecordWeighIn records a weigh-in for the user
func (s *WeighInService) RecordWeighIn(userID, username string, weightLbs float64, notes string) error {
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

	// Insert weigh-in (allows multiple per day)
	logger.DB("Recording weigh-in: user_id=%s, challenge_day=%d, weight=%.2f lbs", userID, challengeDay, weightLbs)
	_, err = s.db.Exec(
		`INSERT INTO weigh_ins (user_id, challenge_day, weight_lbs, notes)
		 VALUES ($1, $2, $3, $4)`,
		userID, challengeDay, weightLbs, notes,
	)
	if err != nil {
		logger.Error("Failed to record weigh-in: %v", err)
		return fmt.Errorf("failed to record weigh-in: %w", err)
	}

	logger.DB("Successfully recorded weigh-in for user_id=%s, challenge_day=%d, weight=%.2f lbs", userID, challengeDay, weightLbs)
	return nil
}

// GetLatestWeighIn gets the most recent weigh-in for a user
func (s *WeighInService) GetLatestWeighIn(userID string) (float64, int, error) {
	if s.db == nil {
		return 0, 0, fmt.Errorf("database not available")
	}

	var weight float64
	var challengeDay int
	err := s.db.QueryRow(
		`SELECT weight_lbs, challenge_day 
		 FROM weigh_ins 
		 WHERE user_id = $1 
		 ORDER BY weighed_at DESC 
		 LIMIT 1`,
		userID,
	).Scan(&weight, &challengeDay)

	if err == sql.ErrNoRows {
		return 0, 0, fmt.Errorf("no weigh-ins found for user")
	}
	if err != nil {
		logger.Error("Failed to get latest weigh-in: %v", err)
		return 0, 0, fmt.Errorf("failed to get latest weigh-in: %w", err)
	}

	return weight, challengeDay, nil
}

// GetWeighInHistory gets weigh-in history for a user (optional limit)
func (s *WeighInService) GetWeighInHistory(userID string, limit int) ([]map[string]interface{}, error) {
	if s.db == nil {
		return nil, fmt.Errorf("database not available")
	}

	if limit <= 0 {
		limit = 10 // Default to 10
	}

	rows, err := s.db.Query(
		`SELECT challenge_day, weight_lbs, weighed_at, notes
		 FROM weigh_ins 
		 WHERE user_id = $1 
		 ORDER BY weighed_at DESC 
		 LIMIT $2`,
		userID, limit,
	)
	if err != nil {
		logger.Error("Failed to query weigh-in history: %v", err)
		return nil, fmt.Errorf("failed to query weigh-in history: %w", err)
	}
	defer rows.Close()

	var history []map[string]interface{}
	for rows.Next() {
		var challengeDay int
		var weight float64
		var weighedAt sql.NullTime
		var notes sql.NullString

		err := rows.Scan(&challengeDay, &weight, &weighedAt, &notes)
		if err != nil {
			return nil, fmt.Errorf("failed to scan weigh-in row: %w", err)
		}

		entry := map[string]interface{}{
			"challenge_day": challengeDay,
			"weight_lbs":    weight,
			"weighed_at":    weighedAt.Time,
			"notes":         notes.String,
		}
		history = append(history, entry)
	}

	return history, nil
}
