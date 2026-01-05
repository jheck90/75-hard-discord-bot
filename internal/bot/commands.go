package bot

import (
	"github.com/bwmarrin/discordgo"
	"github.com/75-hard-discord-bot/internal/logger"
)

// RegisterCommands registers all slash commands with Discord
func RegisterCommands(session *discordgo.Session) error {
	commands := []*discordgo.ApplicationCommand{
		{
			Name:        "exercise",
			Description: "Log your daily exercise (workout + core/mobility)",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "quick",
					Description: "Quick log with defaults (30min workout, 10min core)",
				},
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "detailed",
					Description: "Log with full details (opens a form)",
				},
			},
		},
		{
			Name:        "summary",
			Description: "View challenge progress summary",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "user",
					Description: "Username to view summary for (leave empty for all users)",
					Required:    false,
				},
			},
		},
		{
			Name:        "weighin",
			Description: "Record your daily weigh-in",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionNumber,
					Name:        "weight",
					Description: "Your weight in pounds",
					Required:    true,
				},
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "notes",
					Description: "Optional notes about your weigh-in",
					Required:    false,
					MaxLength:   500,
				},
			},
		},
		{
			Name:        "start",
			Description: "Start your 75 Hard challenge",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "date",
					Description: "Start date (YYYY-MM-DD) - defaults to today (MST)",
					Required:    false,
				},
			},
		},
		{
			Name:        "water",
			Description: "Track your daily water intake",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "summary",
					Description: "View today's total water intake",
				},
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "add",
					Description: "Add water to today's total",
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionNumber,
							Name:        "ounces",
							Description: "Amount of water in ounces to add",
							Required:    true,
						},
					},
				},
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "subtract",
					Description: "Subtract water from today's total",
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionNumber,
							Name:        "ounces",
							Description: "Amount of water in ounces to subtract",
							Required:    true,
						},
					},
				},
			},
		},
	}

	// Register commands
	logger.Info("Registering slash commands...")
	for _, cmd := range commands {
		_, err := session.ApplicationCommandCreate(session.State.User.ID, "", cmd)
		if err != nil {
			logger.Error("Cannot create command '%s': %v", cmd.Name, err)
			return err
		}
		logger.Info("✅ Registered command: /%s", cmd.Name)
	}

	return nil
}
