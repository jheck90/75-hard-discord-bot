package database

import (
	"database/sql"
	"fmt"
	"os"

	_ "github.com/lib/pq" // PostgreSQL driver

	"github.com/75-hard-discord-bot/internal/database/migrations"
)

// Config holds database connection configuration
type Config struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
	SSLMode  string
}

// GetConfigFromEnv reads database configuration from environment variables
// Returns nil if database is not configured (for auto-provisioned mode)
func GetConfigFromEnv() *Config {
	host := os.Getenv("DB_HOST")
	if host == "" {
		return nil // No database configured, will use auto-provisioned mode
	}

	config := &Config{
		Host:     host,
		Port:     getEnvOrDefault("DB_PORT", "5432"),
		User:     getEnvOrDefault("DB_USER", "postgres"),
		Password: os.Getenv("DB_PASSWORD"),
		DBName:   getEnvOrDefault("DB_NAME", "hard75"),
		SSLMode:  getEnvOrDefault("DB_SSLMODE", "require"),
	}

	if config.Password == "" {
		return nil // Password required if host is set
	}

	return config
}

// getEnvOrDefault returns environment variable value or default
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// BuildDSN builds a PostgreSQL connection string from config
func (c *Config) BuildDSN() string {
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.DBName, c.SSLMode,
	)
}

// Connect establishes a database connection and runs migrations
func Connect(config *Config) (*sql.DB, error) {
	if config == nil {
		return nil, fmt.Errorf("database configuration is required")
	}

	dsn := config.BuildDSN()

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}

	// Test connection
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Run migrations
	mgr := migrations.NewManager(db)
	if err := mgr.Run(); err != nil {
		db.Close()
		return nil, fmt.Errorf("migration failed: %w", err)
	}

	// Ensure trigger function exists (applied separately due to migration complexity)
	if err := ensureAutoPopulateTrigger(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ensure trigger: %w", err)
	}

	return db, nil
}

// ConnectOrSkip attempts to connect to database if configured, otherwise returns nil
// This allows the app to run without a database (for testing webhook functionality)
func ConnectOrSkip() (*sql.DB, error) {
	config := GetConfigFromEnv()
	if config == nil {
		return nil, nil // No database configured, skip
	}

	return Connect(config)
}

// ensureAutoPopulateTrigger creates or updates the auto-populate trigger function
// This is applied separately from migrations due to complexity with dollar-quoted strings
func ensureAutoPopulateTrigger(db *sql.DB) error {
	// Check if autopopulated column exists (migration might not have run yet)
	var hasAutopopulated bool
	err := db.QueryRow(
		`SELECT EXISTS(
			SELECT 1 FROM information_schema.columns 
			WHERE table_name = 'exercise_completions' AND column_name = 'autopopulated'
		)`,
	).Scan(&hasAutopopulated)
	if err != nil {
		return fmt.Errorf("failed to check for autopopulated column: %w", err)
	}

	// Build function SQL based on whether autopopulated column exists
	var functionSQL string
	if hasAutopopulated {
		// Use autopopulated-aware logic
		functionSQL = `
			CREATE OR REPLACE FUNCTION auto_populate_feats_on_checkin()
			RETURNS TRIGGER AS $$
			BEGIN
				-- Insert or update exercise completion (only if doesn't exist or was autopopulated)
				INSERT INTO exercise_completions (user_id, challenge_day, completed_at, autopopulated)
				VALUES (NEW.user_id, NEW.challenge_day, NEW.completed_at, true)
				ON CONFLICT (user_id, challenge_day) 
				DO UPDATE SET 
					completed_at = NEW.completed_at,
					autopopulated = true
				WHERE exercise_completions.autopopulated IS NULL OR exercise_completions.autopopulated = true;

				-- Insert or update diet completion (only if doesn't exist or was autopopulated)
				INSERT INTO diet_completions (user_id, challenge_day, completed_at, autopopulated)
				VALUES (NEW.user_id, NEW.challenge_day, NEW.completed_at, true)
				ON CONFLICT (user_id, challenge_day) 
				DO UPDATE SET 
					completed_at = NEW.completed_at,
					autopopulated = true
				WHERE diet_completions.autopopulated IS NULL OR diet_completions.autopopulated = true;

				-- Insert or update water completion (only if doesn't exist or was autopopulated)
				INSERT INTO water_completions (user_id, challenge_day, completed_at, autopopulated)
				VALUES (NEW.user_id, NEW.challenge_day, NEW.completed_at, true)
				ON CONFLICT (user_id, challenge_day) 
				DO UPDATE SET 
					completed_at = NEW.completed_at,
					autopopulated = true
				WHERE water_completions.autopopulated IS NULL OR water_completions.autopopulated = true;

				-- Insert or update self-improvement completion (only if doesn't exist or was autopopulated)
				INSERT INTO self_improvement_completions (user_id, challenge_day, completed_at, autopopulated)
				VALUES (NEW.user_id, NEW.challenge_day, NEW.completed_at, true)
				ON CONFLICT (user_id, challenge_day) 
				DO UPDATE SET 
					completed_at = NEW.completed_at,
					autopopulated = true
				WHERE self_improvement_completions.autopopulated IS NULL OR self_improvement_completions.autopopulated = true;

				-- Insert or update finances completion (only if doesn't exist or was autopopulated)
				INSERT INTO finances_completions (user_id, challenge_day, completed_at, autopopulated)
				VALUES (NEW.user_id, NEW.challenge_day, NEW.completed_at, true)
				ON CONFLICT (user_id, challenge_day) 
				DO UPDATE SET 
					completed_at = NEW.completed_at,
					autopopulated = true
				WHERE finances_completions.autopopulated IS NULL OR finances_completions.autopopulated = true;

				RETURN NEW;
			END;
			$$ LANGUAGE plpgsql;
		`
	} else {
		// Fallback: simple insert without autopopulated column (for backwards compatibility)
		functionSQL = `
			CREATE OR REPLACE FUNCTION auto_populate_feats_on_checkin()
			RETURNS TRIGGER AS $$
			BEGIN
				-- Insert exercise completion with defaults
				INSERT INTO exercise_completions (user_id, challenge_day, completed_at)
				VALUES (NEW.user_id, NEW.challenge_day, NEW.completed_at)
				ON CONFLICT (user_id, challenge_day) DO NOTHING;

				-- Insert diet completion with defaults
				INSERT INTO diet_completions (user_id, challenge_day, completed_at)
				VALUES (NEW.user_id, NEW.challenge_day, NEW.completed_at)
				ON CONFLICT (user_id, challenge_day) DO NOTHING;

				-- Insert water completion with defaults
				INSERT INTO water_completions (user_id, challenge_day, completed_at)
				VALUES (NEW.user_id, NEW.challenge_day, NEW.completed_at)
				ON CONFLICT (user_id, challenge_day) DO NOTHING;

				-- Insert self-improvement completion with defaults
				INSERT INTO self_improvement_completions (user_id, challenge_day, completed_at)
				VALUES (NEW.user_id, NEW.challenge_day, NEW.completed_at)
				ON CONFLICT (user_id, challenge_day) DO NOTHING;

				-- Insert finances completion with defaults
				INSERT INTO finances_completions (user_id, challenge_day, completed_at)
				VALUES (NEW.user_id, NEW.challenge_day, NEW.completed_at)
				ON CONFLICT (user_id, challenge_day) DO NOTHING;

				RETURN NEW;
			END;
			$$ LANGUAGE plpgsql;
		`
	}

	_, err = db.Exec(functionSQL)
	if err != nil {
		return fmt.Errorf("failed to create trigger function: %w", err)
	}

	// Check if trigger already exists
	var exists bool
	err = db.QueryRow(
		`SELECT EXISTS(
			SELECT 1 FROM pg_trigger 
			WHERE tgname = 'trigger_auto_populate_feats'
		)`,
	).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check if trigger exists: %w", err)
	}

	if !exists {
		// Create the trigger (fires on both INSERT and UPDATE)
		triggerSQL := `
			CREATE TRIGGER trigger_auto_populate_feats
				AFTER INSERT OR UPDATE ON accountability_checkins
				FOR EACH ROW
				EXECUTE FUNCTION auto_populate_feats_on_checkin();
		`

		_, err = db.Exec(triggerSQL)
		if err != nil {
			return fmt.Errorf("failed to create trigger: %w", err)
		}
		fmt.Println("✅ Created auto-populate trigger")
	} else {
		// Update existing trigger to also fire on UPDATE
		dropSQL := `DROP TRIGGER IF EXISTS trigger_auto_populate_feats ON accountability_checkins;`
		_, err = db.Exec(dropSQL)
		if err != nil {
			return fmt.Errorf("failed to drop existing trigger: %w", err)
		}
		
		triggerSQL := `
			CREATE TRIGGER trigger_auto_populate_feats
				AFTER INSERT OR UPDATE ON accountability_checkins
				FOR EACH ROW
				EXECUTE FUNCTION auto_populate_feats_on_checkin();
		`
		_, err = db.Exec(triggerSQL)
		if err != nil {
			return fmt.Errorf("failed to recreate trigger: %w", err)
		}
		// Note: Logger not available in this package, using fmt for critical messages
		fmt.Println("✅ Updated auto-populate trigger to fire on INSERT and UPDATE")
	}

	return nil
}
