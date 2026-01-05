package services

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/75-hard-discord-bot/internal/logger"
)

// SummaryService handles summary-related operations
type SummaryService struct {
	db *sql.DB
}

// NewSummaryService creates a new summary service
func NewSummaryService() *SummaryService {
	return &SummaryService{}
}

// Initialize initializes the service with database connection
func (s *SummaryService) Initialize(db *sql.DB) error {
	s.db = db
	return nil
}

// Name returns the service name
func (s *SummaryService) Name() string {
	return "SummaryService"
}

// Health checks the service health
func (s *SummaryService) Health() error {
	if s.db == nil {
		return fmt.Errorf("database not initialized")
	}
	return s.db.Ping()
}

// GetProgressSummary returns a formatted progress summary
func (s *SummaryService) GetProgressSummary(targetUsername string) (string, error) {
	if targetUsername == "" {
		return s.GetAllUsersSummary()
	}
	return s.GetUserSummary(targetUsername)
}

// GetAllUsersSummary returns summary for all users
func (s *SummaryService) GetAllUsersSummary() (string, error) {
	if s.db == nil {
		return "", fmt.Errorf("database not available")
	}

	// Count distinct challenge days completed (using check-ins as the source of truth)
	query := `
		SELECT 
			u.user_id,
			u.username,
			u.challenge_start_date,
			u.current_challenge_end_date,
			u.days_added,
			COUNT(DISTINCT CASE WHEN a.challenge_day >= 1 AND a.challenge_day <= GREATEST(1, (CURRENT_DATE::date - u.challenge_start_date::date) + 1) THEN a.challenge_day END) as days_completed
		FROM users u
		LEFT JOIN accountability_checkins a ON a.user_id = u.user_id
		GROUP BY u.user_id, u.username, u.challenge_start_date, u.current_challenge_end_date, u.days_added
		ORDER BY days_completed DESC, u.username
	`

	logger.DB("Querying summary for all users")
	rows, err := s.db.Query(query)
	if err != nil {
		logger.Error("Failed to query users: %v", err)
		return "", fmt.Errorf("failed to query users: %w", err)
	}
	defer rows.Close()

	var summary strings.Builder
	summary.WriteString("📊 **Challenge Progress Summary (All Users)**\n\n")

	for rows.Next() {
		var userID, username string
		var startDate, endDate time.Time
		var daysAdded int
		var daysCompleted sql.NullInt64

		err := rows.Scan(&userID, &username, &startDate, &endDate, &daysAdded, &daysCompleted)
		if err != nil {
			return "", fmt.Errorf("failed to scan user row: %w", err)
		}

		totalDays := int(endDate.Sub(startDate).Hours() / 24)
		currentDay := int(time.Since(startDate).Hours()/24) + 1
		if currentDay > totalDays {
			currentDay = totalDays
		}

		summary.WriteString(fmt.Sprintf("**%s** (Day %d/%d", username, currentDay, totalDays))
		if daysAdded > 0 {
			summary.WriteString(fmt.Sprintf(" +%d", daysAdded))
		}
		summary.WriteString(")\n")
		summary.WriteString(fmt.Sprintf("  ✅ Days Completed: %d\n\n", daysCompleted.Int64))
	}

	if summary.Len() == len("📊 **Challenge Progress Summary (All Users)**\n\n") {
		summary.WriteString("No users found.")
	}

	return summary.String(), nil
}

// GetUserSummary returns summary for a specific user
func (s *SummaryService) GetUserSummary(username string) (string, error) {
	if s.db == nil {
		return "", fmt.Errorf("database not available")
	}

	query := `
		SELECT 
			u.user_id,
			u.username,
			u.challenge_start_date,
			u.current_challenge_end_date,
			u.days_added,
			COUNT(DISTINCT CASE WHEN a.challenge_day >= 1 AND a.challenge_day <= GREATEST(1, (CURRENT_DATE::date - u.challenge_start_date::date) + 1) THEN a.challenge_day END) as days_completed
		FROM users u
		LEFT JOIN accountability_checkins a ON a.user_id = u.user_id
		WHERE LOWER(u.username) = LOWER($1)
		GROUP BY u.user_id, u.username, u.challenge_start_date, u.current_challenge_end_date, u.days_added
	`

	logger.DB("Querying summary for user: %s", username)
	var userID, dbUsername string
	var startDate, endDate time.Time
	var daysAdded int
	var daysCompleted sql.NullInt64

	err := s.db.QueryRow(query, username).Scan(&userID, &dbUsername, &startDate, &endDate, &daysAdded, &daysCompleted)
	if err == sql.ErrNoRows {
		logger.DB("User not found: %s", username)
		return fmt.Sprintf("❌ User '%s' not found.", username), nil
	}
	if err != nil {
		logger.Error("Failed to query user: %v", err)
		return "", fmt.Errorf("failed to query user: %w", err)
	}

	totalDays := int(endDate.Sub(startDate).Hours() / 24)
	currentDay := int(time.Since(startDate).Hours()/24) + 1
	if currentDay > totalDays {
		currentDay = totalDays
	}

	var summary strings.Builder
	summary.WriteString(fmt.Sprintf("📊 **Challenge Progress Summary: %s**\n\n", dbUsername))
	summary.WriteString(fmt.Sprintf("**Challenge:** Day %d/%d", currentDay, totalDays))
	if daysAdded > 0 {
		summary.WriteString(fmt.Sprintf(" (+%d days added)", daysAdded))
	}
	summary.WriteString(fmt.Sprintf("\n**Started:** %s\n\n", startDate.Format("January 2, 2006")))

	summary.WriteString(fmt.Sprintf("**Days Completed:** %d\n", daysCompleted.Int64))

	// Calculate completion percentage
	completionRate := float64(daysCompleted.Int64) / float64(totalDays) * 100
	summary.WriteString(fmt.Sprintf("\n**Progress:** %.1f%% (%d/%d days)", completionRate, daysCompleted.Int64, totalDays))

	return summary.String(), nil
}
