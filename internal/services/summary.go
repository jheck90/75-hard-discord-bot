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

	query := `
		SELECT 
			u.user_id,
			u.username,
			u.challenge_start_date,
			u.current_challenge_end_date,
			u.days_added,
			COUNT(DISTINCT a.challenge_day) as checkin_days,
			COUNT(DISTINCT e.challenge_day) as exercise_days,
			COUNT(DISTINCT d.challenge_day) as diet_days,
			COUNT(DISTINCT w.challenge_day) as water_days,
			COUNT(DISTINCT si.challenge_day) as self_improvement_days,
			COUNT(DISTINCT CASE WHEN f.compliance_status = 'compliant' THEN f.challenge_day END) as finances_days
		FROM users u
		LEFT JOIN accountability_checkins a ON a.user_id = u.user_id
		LEFT JOIN exercise_completions e ON e.user_id = u.user_id
		LEFT JOIN diet_completions d ON d.user_id = u.user_id
		LEFT JOIN water_completions w ON w.user_id = u.user_id
		LEFT JOIN self_improvement_completions si ON si.user_id = u.user_id
		LEFT JOIN finances_completions f ON f.user_id = u.user_id
		GROUP BY u.user_id, u.username, u.challenge_start_date, u.current_challenge_end_date, u.days_added
		ORDER BY checkin_days DESC, u.username
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
		var daysAdded, checkinDays, exerciseDays, dietDays, waterDays, selfImproveDays, financesDays sql.NullInt64

		err := rows.Scan(&userID, &username, &startDate, &endDate, &daysAdded, &checkinDays, &exerciseDays, &dietDays, &waterDays, &selfImproveDays, &financesDays)
		if err != nil {
			return "", fmt.Errorf("failed to scan user row: %w", err)
		}

		totalDays := int(endDate.Sub(startDate).Hours() / 24)
		currentDay := int(time.Since(startDate).Hours()/24) + 1
		if currentDay > totalDays {
			currentDay = totalDays
		}

		summary.WriteString(fmt.Sprintf("**%s** (Day %d/%d", username, currentDay, totalDays))
		if daysAdded.Int64 > 0 {
			summary.WriteString(fmt.Sprintf(" +%d", daysAdded.Int64))
		}
		summary.WriteString(")\n")
		summary.WriteString(fmt.Sprintf("  ✅ Check-ins: %d | 💪 Exercise: %d | 🍽️ Diet: %d | 💧 Water: %d | 📚 Self-Improve: %d | 💰 Finances: %d\n\n",
			checkinDays.Int64, exerciseDays.Int64, dietDays.Int64, waterDays.Int64, selfImproveDays.Int64, financesDays.Int64))
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
			COUNT(DISTINCT a.challenge_day) as checkin_days,
			COUNT(DISTINCT e.challenge_day) as exercise_days,
			COUNT(DISTINCT d.challenge_day) as diet_days,
			COUNT(DISTINCT w.challenge_day) as water_days,
			COUNT(DISTINCT si.challenge_day) as self_improvement_days,
			COUNT(DISTINCT CASE WHEN f.compliance_status = 'compliant' THEN f.challenge_day END) as finances_days
		FROM users u
		LEFT JOIN accountability_checkins a ON a.user_id = u.user_id
		LEFT JOIN exercise_completions e ON e.user_id = u.user_id
		LEFT JOIN diet_completions d ON d.user_id = u.user_id
		LEFT JOIN water_completions w ON w.user_id = u.user_id
		LEFT JOIN self_improvement_completions si ON si.user_id = u.user_id
		LEFT JOIN finances_completions f ON f.user_id = u.user_id
		WHERE LOWER(u.username) = LOWER($1)
		GROUP BY u.user_id, u.username, u.challenge_start_date, u.current_challenge_end_date, u.days_added
	`

	logger.DB("Querying summary for user: %s", username)
	var userID, dbUsername string
	var startDate, endDate time.Time
	var daysAdded, checkinDays, exerciseDays, dietDays, waterDays, selfImproveDays, financesDays sql.NullInt64

	err := s.db.QueryRow(query, username).Scan(&userID, &dbUsername, &startDate, &endDate, &daysAdded, &checkinDays, &exerciseDays, &dietDays, &waterDays, &selfImproveDays, &financesDays)
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
	if daysAdded.Int64 > 0 {
		summary.WriteString(fmt.Sprintf(" (+%d days added)", daysAdded.Int64))
	}
	summary.WriteString(fmt.Sprintf("\n**Started:** %s\n\n", startDate.Format("January 2, 2006")))

	summary.WriteString("**Feat Completions:**\n")
	summary.WriteString(fmt.Sprintf("  ✅ Accountability Check-ins: %d days\n", checkinDays.Int64))
	summary.WriteString(fmt.Sprintf("  💪 Exercise: %d days\n", exerciseDays.Int64))
	summary.WriteString(fmt.Sprintf("  🍽️  Diet: %d days\n", dietDays.Int64))
	summary.WriteString(fmt.Sprintf("  💧 Water: %d days\n", waterDays.Int64))
	summary.WriteString(fmt.Sprintf("  📚 Self-Improvement: %d days\n", selfImproveDays.Int64))
	summary.WriteString(fmt.Sprintf("  💰 Finances: %d days\n", financesDays.Int64))

	// Calculate completion percentage
	allFeatsCompleted := min(checkinDays.Int64, exerciseDays.Int64, dietDays.Int64, waterDays.Int64, selfImproveDays.Int64, financesDays.Int64)
	completionRate := float64(allFeatsCompleted) / float64(totalDays) * 100
	summary.WriteString(fmt.Sprintf("\n**Overall Progress:** %.1f%% (%d/%d days with all feats completed)", completionRate, allFeatsCompleted, totalDays))

	return summary.String(), nil
}

// min returns the minimum of multiple int64 values
func min(values ...int64) int64 {
	if len(values) == 0 {
		return 0
	}
	minVal := values[0]
	for _, v := range values[1:] {
		if v < minVal {
			minVal = v
		}
	}
	return minVal
}
