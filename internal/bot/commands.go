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
