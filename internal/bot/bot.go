package bot

import (
	"database/sql"
	"fmt"

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

	logger.Info("75 Hard Discord Bot")
	logger.Info("===================")
	if b.db != nil {
		logger.Info("✅ Database connected - check-ins will be recorded")
	} else {
		logger.Info("⚠️  No database configured - check-ins will not be recorded")
	}
	logger.Info("Bot is now running and listening for commands and reactions...")

	// Send introduction message
	if err := b.SendIntroduction(b.config.DiscordChannelID); err != nil {
		return fmt.Errorf("failed to send introduction: %w", err)
	}

	// Send the check-in message
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

// SendCheckInMessage sends the daily check-in message to the channel
func (b *Bot) SendCheckInMessage(channelID string) error {
	checkInMessage := "Check this message to confirm you completed the challenges today"
	logger.DB("Sending check-in message to channel_id=%s", channelID)
	msg, err := b.session.ChannelMessageSend(channelID, checkInMessage)
	if err != nil {
		return fmt.Errorf("error sending check-in message: %w", err)
	}

	// Add a self-reaction so users can easily click it
	err = b.session.MessageReactionAdd(channelID, msg.ID, "✅")
	if err != nil {
		logger.Error("⚠️  Warning: Could not add self-reaction: %v", err)
		logger.Info("   Users can still react manually")
	}

	logger.Info("✅ Check-in message sent to channel %s", channelID)
	logger.Info("   Message ID: %s", msg.ID)
	logger.Info("   Bot has added ✅ reaction - users can click it to check in!")

	return nil
}
