package bot

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/75-hard-discord-bot/internal/config"
	"github.com/75-hard-discord-bot/internal/handlers"
	"github.com/75-hard-discord-bot/internal/logger"
	"github.com/75-hard-discord-bot/internal/services"
)

// Bot represents the Discord bot instance
type Bot struct {
	session  *discordgo.Session
	config   *config.Config
	db       *sql.DB
	services *services.ServiceRegistry
}

// NewBot creates a new bot instance
func NewBot(cfg *config.Config, db *sql.DB, serviceRegistry *services.ServiceRegistry) (*Bot, error) {
	// Create Discord session
	session, err := discordgo.New("Bot " + cfg.DiscordBotToken)
	if err != nil {
		return nil, fmt.Errorf("error creating Discord session: %w", err)
	}

	// Register intents needed for slash commands and interactions
	session.Identify.Intents = discordgo.IntentsGuildMessages | discordgo.IntentsGuildMessageReactions | discordgo.IntentsGuilds

	bot := &Bot{
		session:  session,
		config:   cfg,
		db:       db,
		services: serviceRegistry,
	}

	return bot, nil
}

// Start starts the bot and registers handlers
func (b *Bot) Start() error {
	// Create handlers
	interactionHandler := handlers.NewInteractionHandler(b.services)
	modalHandler := handlers.NewModalHandler(b.services)
	reactionHandler := handlers.NewReactionHandler(b.services)

	// Register handlers
	b.session.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		if i.Type == discordgo.InteractionApplicationCommand {
			interactionHandler.HandleSlashCommand(s, i)
		}
	})

	b.session.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		if i.Type == discordgo.InteractionModalSubmit {
			modalHandler.HandleModalSubmit(s, i)
		} else if i.Type == discordgo.InteractionMessageComponent {
			interactionHandler.HandleButtonClick(s, i)
		}
	})

	b.session.AddHandler(func(s *discordgo.Session, r *discordgo.MessageReactionAdd) {
		reactionHandler.HandleMessageReaction(s, r)
	})

	// Open websocket connection
	logger.Info("Opening Discord websocket connection...")
	err := b.session.Open()
	if err != nil {
		return fmt.Errorf("error opening connection: %w", err)
	}

	// Register slash commands
	if err := RegisterCommands(b.session); err != nil {
		return fmt.Errorf("failed to register commands: %w", err)
	}

	logger.Info("75 Half Chub Discord Bot")
	logger.Info("===================")
	if b.db != nil {
		logger.Info("✅ Database connected - check-ins will be recorded")
		
		// Query and display active users
		if err := b.DisplayActiveUsers(b.config.DiscordChannelID); err != nil {
			logger.Error("Failed to display active users: %v", err)
		}
	} else {
		logger.Info("⚠️  No database configured - check-ins will not be recorded")
	}
	logger.Info("Bot is now running and listening for commands and reactions...")

	// Send introduction message
	if err := b.SendIntroduction(b.config.DiscordChannelID); err != nil {
		return fmt.Errorf("failed to send introduction: %w", err)
	}

	// Send the check-in message (pinned, datestamped)
	if err := b.SendCheckInMessage(b.config.DiscordChannelID); err != nil {
		return fmt.Errorf("failed to send check-in message: %w", err)
	}

	return nil
}

// Stop gracefully shuts down the bot
func (b *Bot) Stop() error {
	logger.Info("Shutting down bot...")
	return b.session.Close()
}

// SendIntroduction sends a one-sentence introduction message to the channel
func (b *Bot) SendIntroduction(channelID string) error {
	introMessage := "👋 75 Half Chub Bot here! I'll help you track your daily challenge progress."
	logger.Info("Sending introduction message to channel_id=%s", channelID)
	_, err := b.session.ChannelMessageSend(channelID, introMessage)
	if err != nil {
		return fmt.Errorf("error sending introduction: %w", err)
	}
	logger.Info("✅ Introduction message sent")
	return nil
}

// DisplayActiveUsers queries and displays currently active challenge participants
func (b *Bot) DisplayActiveUsers(channelID string) error {
	if b.db == nil {
		return nil // No database, skip
	}

	// Get user service from registry
	var userService *services.UserService
	for _, svc := range b.services.GetServices() {
		if us, ok := svc.(*services.UserService); ok {
			userService = us
			break
		}
	}

	if userService == nil {
		return fmt.Errorf("user service not available")
	}

	activeUsers, err := userService.GetActiveUsers()
	if err != nil {
		return fmt.Errorf("failed to get active users: %w", err)
	}

	if len(activeUsers) == 0 {
		logger.Info("📊 No active users found")
		return nil
	}

	// Load MST location for date formatting
	mst, err := time.LoadLocation("America/Denver")
	if err != nil {
		mst = time.FixedZone("MST", -7*3600)
	}
	today := time.Now().In(mst).Format("January 2, 2006")

	var message strings.Builder
	message.WriteString(fmt.Sprintf("📊 **Active Challenge Participants** - %s (MST)\n\n", today))

	for _, user := range activeUsers {
		// Dates are already in MST from GetActiveUsers
		startDateStr := user.StartDate.Format("Jan 2, 2006")
		endDateStr := user.EndDate.Format("Jan 2, 2006")
		
		message.WriteString(fmt.Sprintf("**%s** - Day %d/%d", user.Username, user.CurrentDay, user.TotalDays))
		if user.DaysAdded > 0 {
			message.WriteString(fmt.Sprintf(" (+%d)", user.DaysAdded))
		}
		message.WriteString(fmt.Sprintf("\n  Started: %s | Ends: %s\n\n", startDateStr, endDateStr))
	}

	message.WriteString(fmt.Sprintf("_Total active participants: %d_", len(activeUsers)))

	logger.Info("Displaying active users to channel_id=%s", channelID)
	_, err = b.session.ChannelMessageSend(channelID, message.String())
	if err != nil {
		return fmt.Errorf("error sending active users message: %w", err)
	}

	logger.Info("✅ Displayed %d active users", len(activeUsers))
	return nil
}

// SendCheckInMessage sends the daily check-in message to the channel (pinned, datestamped)
func (b *Bot) SendCheckInMessage(channelID string) error {
	// Load MST location for date formatting
	mst, err := time.LoadLocation("America/Denver")
	if err != nil {
		mst = time.FixedZone("MST", -7*3600)
	}
	today := time.Now().In(mst)
	dateStr := today.Format("January 2, 2006")

	// Try to find and unpin existing check-in messages
	b.CleanupOldCheckInMessages(channelID)

	checkInMessage := fmt.Sprintf("📅 **Daily Check-In - %s (MST)**\n\nCheck this message to confirm you completed the challenges today", dateStr)
	logger.DB("Sending check-in message to channel_id=%s", channelID)
	msg, err := b.session.ChannelMessageSend(channelID, checkInMessage)
	if err != nil {
		return fmt.Errorf("error sending check-in message: %w", err)
	}

	// Pin the message
	err = b.session.ChannelMessagePin(channelID, msg.ID)
	if err != nil {
		logger.Error("⚠️  Warning: Could not pin check-in message: %v", err)
		logger.Info("   Message sent but not pinned")
	}

	// Add a self-reaction so users can easily click it
	err = b.session.MessageReactionAdd(channelID, msg.ID, "✅")
	if err != nil {
		logger.Error("⚠️  Warning: Could not add self-reaction: %v", err)
		logger.Info("   Users can still react manually")
	}

	logger.Info("✅ Check-in message sent and pinned to channel %s", channelID)
	logger.Info("   Message ID: %s", msg.ID)
	logger.Info("   Date: %s", dateStr)
	logger.Info("   Bot has added ✅ reaction - users can click it to check in!")

	return nil
}

// CleanupOldCheckInMessages finds and unpins old check-in messages
func (b *Bot) CleanupOldCheckInMessages(channelID string) {
	// Get pinned messages
	pins, err := b.session.ChannelMessagesPinned(channelID)
	if err != nil {
		logger.Error("Failed to get pinned messages: %v", err)
		return
	}

	botID := b.session.State.User.ID
	for _, pin := range pins {
		// Only unpin messages from the bot that look like check-in messages
		if pin.Author.ID == botID && strings.Contains(pin.Content, "Daily Check-In") {
			err := b.session.ChannelMessageUnpin(channelID, pin.ID)
			if err != nil {
				logger.Error("Failed to unpin old check-in message %s: %v", pin.ID, err)
			} else {
				logger.DB("Unpinned old check-in message: %s", pin.ID)
			}
		}
	}
}
