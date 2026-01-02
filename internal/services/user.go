package services

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/75-hard-discord-bot/internal/logger"
)

// UserService handles user-related operations
type UserService struct {
	db *sql.DB
}

// NewUserService creates a new user service
func NewUserService() *UserService {
	return &UserService{}
}

// Initialize initializes the service with database connection
func (s *UserService) Initialize(db *sql.DB) error {
	s.db = db
	return nil
}

// Name returns the service name
func (s *UserService) Name() string {
	return "UserService"
}

// Health checks the service health
func (s *UserService) Health() error {
	if s.db == nil {
		return fmt.Errorf("database not initialized")
	}
	return s.db.Ping()
}

// EnsureUserExists creates a user record if it doesn't exist
func (s *UserService) EnsureUserExists(userID, username string) error {
	if s.db == nil {
		return fmt.Errorf("database not available")
	}

	now := time.Now()
	startDate := now.Format("2006-01-02")
	endDate := now.AddDate(0, 0, 75).Format("2006-01-02")

	logger.DB("Executing INSERT/UPDATE on users table: user_id=%s, username=%s, start_date=%s", userID, username, startDate)
	_, err := s.db.Exec(
		`INSERT INTO users (user_id, username, challenge_start_date, original_challenge_end_date, current_challenge_end_date)
		 VALUES ($1, $2, $3, $4, $5)
		 ON CONFLICT (user_id) DO UPDATE SET username = EXCLUDED.username`,
		userID, username, startDate, endDate, endDate,
	)
	if err != nil {
		logger.Error("Failed to ensure user exists: %v", err)
	}
	return err
}

// GetCurrentChallengeDay calculates the current challenge day for a user
func (s *UserService) GetCurrentChallengeDay(userID string) (int, error) {
	if s.db == nil {
		return 0, fmt.Errorf("database not available")
	}

	logger.DB("Querying challenge_start_date for user_id=%s", userID)
	var startDate time.Time
	err := s.db.QueryRow(
		`SELECT challenge_start_date FROM users WHERE user_id = $1`,
		userID,
	).Scan(&startDate)
	if err != nil {
		logger.Error("Failed to get challenge start date: %v", err)
		return 0, err
	}

	daysSinceStart := int(time.Since(startDate).Hours() / 24)
	if daysSinceStart < 0 {
		daysSinceStart = 0
	}
	challengeDay := daysSinceStart + 1
	logger.DB("Calculated challenge_day=%d for user_id=%s", challengeDay, userID)
	return challengeDay, nil
}
