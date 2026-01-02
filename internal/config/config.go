package config

import (
	"fmt"
	"os"
)

// Config holds all application configuration
type Config struct {
	DiscordBotToken  string
	DiscordChannelID string
	Database         *DatabaseConfig
}

// DatabaseConfig holds database configuration
type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
	SSLMode  string
}

// Load loads configuration from environment variables
func Load() (*Config, error) {
	cfg := &Config{
		DiscordBotToken:  os.Getenv("DISCORD_BOT_TOKEN"),
		DiscordChannelID: os.Getenv("DISCORD_CHANNEL_ID"),
	}

	// Validate required Discord config
	if cfg.DiscordBotToken == "" {
		return nil, fmt.Errorf("DISCORD_BOT_TOKEN environment variable is not set")
	}
	if cfg.DiscordChannelID == "" {
		return nil, fmt.Errorf("DISCORD_CHANNEL_ID environment variable is not set")
	}

	// Load database config (optional)
	dbHost := os.Getenv("DB_HOST")
	if dbHost != "" {
		dbPassword := os.Getenv("DB_PASSWORD")
		if dbPassword == "" {
			return nil, fmt.Errorf("DB_PASSWORD is required when DB_HOST is set")
		}

		cfg.Database = &DatabaseConfig{
			Host:     dbHost,
			Port:     getEnvOrDefault("DB_PORT", "5432"),
			User:     getEnvOrDefault("DB_USER", "postgres"),
			Password: dbPassword,
			DBName:   getEnvOrDefault("DB_NAME", "hard75"),
			SSLMode:  getEnvOrDefault("DB_SSLMODE", "require"),
		}
	}

	return cfg, nil
}

// getEnvOrDefault returns environment variable value or default
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
