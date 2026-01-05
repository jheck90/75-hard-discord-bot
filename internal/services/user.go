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

// StartChallenge starts or updates a user's challenge with a specific start date
func (s *UserService) StartChallenge(userID, username string, startDate time.Time) (time.Time, time.Time, error) {
	if s.db == nil {
		return time.Time{}, time.Time{}, fmt.Errorf("database not available")
	}

	endDate := startDate.AddDate(0, 0, 75)
	startDateStr := startDate.Format("2006-01-02")
	endDateStr := endDate.Format("2006-01-02")

	logger.DB("Starting challenge: user_id=%s, username=%s, start_date=%s", userID, username, startDateStr)
	_, err := s.db.Exec(
		`INSERT INTO users (user_id, username, challenge_start_date, original_challenge_end_date, current_challenge_end_date)
		 VALUES ($1, $2, $3, $4, $5)
		 ON CONFLICT (user_id) DO UPDATE SET 
			username = EXCLUDED.username,
			challenge_start_date = EXCLUDED.challenge_start_date,
			original_challenge_end_date = EXCLUDED.original_challenge_end_date,
			current_challenge_end_date = EXCLUDED.current_challenge_end_date,
			days_added = 0`,
		userID, username, startDateStr, endDateStr, endDateStr,
	)
	if err != nil {
		logger.Error("Failed to start challenge: %v", err)
		return time.Time{}, time.Time{}, fmt.Errorf("failed to start challenge: %w", err)
	}

	logger.DB("Successfully started challenge for user_id=%s, start_date=%s, end_date=%s", userID, startDateStr, endDateStr)
	return startDate, endDate, nil
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

// ActiveUser represents a user currently participating in the challenge
type ActiveUser struct {
	UserID      string
	Username    string
	StartDate   time.Time
	EndDate     time.Time
	CurrentDay  int
	TotalDays   int
	DaysAdded   int
}

// GetActiveUsers returns all users currently participating in the challenge
func (s *UserService) GetActiveUsers() ([]ActiveUser, error) {
	if s.db == nil {
		return nil, fmt.Errorf("database not available")
	}

	// Load MST location for consistent date handling
	mst, err := time.LoadLocation("America/Denver")
	if err != nil {
		mst = time.FixedZone("MST", -7*3600)
	}

	// Get today's date in MST (normalized to midnight)
	nowMST := time.Now().In(mst)
	todayMST := time.Date(nowMST.Year(), nowMST.Month(), nowMST.Day(), 0, 0, 0, 0, mst)
	
	// Use date-only comparison (cast to date in SQL)
	query := `
		SELECT 
			user_id,
			username,
			challenge_start_date,
			current_challenge_end_date,
			days_added
		FROM users
		WHERE challenge_start_date::date <= $1::date
		  AND current_challenge_end_date::date >= $1::date
		ORDER BY challenge_start_date ASC, username ASC
	`

	rows, err := s.db.Query(query, todayMST)
	if err != nil {
		logger.Error("Failed to query active users: %v", err)
		return nil, fmt.Errorf("failed to query active users: %w", err)
	}
	defer rows.Close()

	var activeUsers []ActiveUser
	for rows.Next() {
		var userID, username string
		var startDate, endDate time.Time
		var daysAdded int

		err := rows.Scan(&userID, &username, &startDate, &endDate, &daysAdded)
		if err != nil {
			logger.Error("Failed to scan active user row: %v", err)
			continue
		}

		// Normalize dates to MST midnight for accurate day calculations
		startDateMST := time.Date(startDate.Year(), startDate.Month(), startDate.Day(), 0, 0, 0, 0, mst)
		endDateMST := time.Date(endDate.Year(), endDate.Month(), endDate.Day(), 0, 0, 0, 0, mst)

		// Calculate days since start using MST dates
		daysSinceStart := int(todayMST.Sub(startDateMST).Hours() / 24)
		if daysSinceStart < 0 {
			daysSinceStart = 0
		}
		currentDay := daysSinceStart + 1
		totalDays := int(endDateMST.Sub(startDateMST).Hours() / 24)
		if currentDay > totalDays {
			currentDay = totalDays
		}

		activeUsers = append(activeUsers, ActiveUser{
			UserID:     userID,
			Username:   username,
			StartDate:  startDateMST,
			EndDate:    endDateMST,
			CurrentDay: currentDay,
			TotalDays:  totalDays,
			DaysAdded:  daysAdded,
		})
	}

	return activeUsers, nil
}
