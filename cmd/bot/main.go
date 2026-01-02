package main

import (
	"database/sql"
	"os"
	"os/signal"
	"syscall"

	"github.com/75-hard-discord-bot/internal/bot"
	"github.com/75-hard-discord-bot/internal/config"
	"github.com/75-hard-discord-bot/internal/database"
	"github.com/75-hard-discord-bot/internal/logger"
	"github.com/75-hard-discord-bot/internal/services"
)

func main() {
	// Initialize logger
	logLevel := logger.GetLogLevelFromEnv()
	devMode := logger.GetDevModeFromEnv()
	logger.Init(logLevel, devMode)

	// Load configuration
	logger.Info("Loading configuration...")
	cfg, err := config.Load()
	if err != nil {
		logger.Fatal("Failed to load configuration: %v", err)
	}

	// Initialize database connection (optional - app can run without DB)
	logger.Info("🔌 Initializing database connection...")
	var db *sql.DB
	if cfg.Database != nil {
		dbConfig := &database.Config{
			Host:     cfg.Database.Host,
			Port:     cfg.Database.Port,
			User:     cfg.Database.User,
			Password: cfg.Database.Password,
			DBName:   cfg.Database.DBName,
			SSLMode:  cfg.Database.SSLMode,
		}
		db, err = database.Connect(dbConfig)
		if err != nil {
			logger.Fatal("❌ Failed to connect to database: %v", err)
		}
		logger.Info("✅ Database connected and migrations applied")
		defer db.Close()
	} else {
		logger.Info("⚠️  No database configured - database features will be unavailable")
	}

	// Create service registry
	serviceRegistry := services.NewServiceRegistry()

	// Create and register services
	userService := services.NewUserService()
	serviceRegistry.Register(userService)

	checkInService := services.NewCheckInService(userService)
	serviceRegistry.Register(checkInService)

	exerciseService := services.NewExerciseService(userService)
	serviceRegistry.Register(exerciseService)

	summaryService := services.NewSummaryService()
	serviceRegistry.Register(summaryService)

	// Initialize all services
	if db != nil {
		logger.Info("Initializing services...")
		if err := serviceRegistry.InitializeAll(db); err != nil {
			logger.Fatal("Failed to initialize services: %v", err)
		}
		logger.Info("✅ All services initialized")
	}

	// Create and start bot
	logger.Info("Creating bot instance...")
	discordBot, err := bot.NewBot(cfg, db, serviceRegistry)
	if err != nil {
		logger.Fatal("Failed to create bot: %v", err)
	}

	// Start bot
	if err := discordBot.Start(); err != nil {
		logger.Fatal("Failed to start bot: %v", err)
	}
	defer discordBot.Stop()

	// Wait for interrupt signal
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	logger.Info("\nShutting down...")
}
