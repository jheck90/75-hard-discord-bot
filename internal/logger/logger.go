package logger

import (
	"log"
	"os"
	"strings"
)

// LogLevel represents the logging level
type LogLevel int

const (
	ERROR LogLevel = iota
	INFO
)

var (
	currentLevel LogLevel = ERROR
	isDevMode    bool     = false
)

// Init initializes the logger with the specified log level and dev mode
func Init(logLevel string, devMode bool) {
	isDevMode = devMode
	
	switch strings.ToUpper(logLevel) {
	case "INFO":
		currentLevel = INFO
	case "ERROR":
		currentLevel = ERROR
	default:
		currentLevel = ERROR
	}
	
	// Log initialization message (always log this, even at ERROR level)
	if isDevMode {
		log.Printf("[INFO] Logger initialized in DEV mode with level: %s", logLevel)
	} else {
		log.Printf("[INFO] Logger initialized in PROD mode with level: %s", logLevel)
	}
}

// IsDevMode returns whether the app is running in dev mode
func IsDevMode() bool {
	return isDevMode
}

// Info logs an informational message if log level is INFO or higher
func Info(format string, v ...interface{}) {
	if currentLevel >= INFO {
		log.Printf("[INFO] "+format, v...)
	}
}

// Error logs an error message (always logged)
func Error(format string, v ...interface{}) {
	log.Printf("[ERROR] "+format, v...)
}

// Fatal logs a fatal error and exits
func Fatal(format string, v ...interface{}) {
	log.Fatalf("[FATAL] "+format, v...)
}

// DB logs database operations at INFO level
func DB(format string, v ...interface{}) {
	if currentLevel >= INFO {
		log.Printf("[DB] "+format, v...)
	}
}

// GetLogLevelFromEnv reads LOG_LEVEL from environment, defaults to ERROR
func GetLogLevelFromEnv() string {
	level := os.Getenv("LOG_LEVEL")
	if level == "" {
		return "ERROR"
	}
	return level
}

// GetDevModeFromEnv reads DEV_MODE from environment
func GetDevModeFromEnv() bool {
	mode := strings.ToLower(os.Getenv("DEV_MODE"))
	return mode == "dev" || mode == "development" || mode == "true" || mode == "1"
}
