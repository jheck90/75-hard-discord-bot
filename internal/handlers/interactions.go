package handlers

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
	"github.com/75-hard-discord-bot/internal/logger"
	"github.com/75-hard-discord-bot/internal/services"
)

// InteractionHandler handles slash command interactions
type InteractionHandler struct {
	services *services.ServiceRegistry
}

// NewInteractionHandler creates a new interaction handler
func NewInteractionHandler(serviceRegistry *services.ServiceRegistry) *InteractionHandler {
	return &InteractionHandler{
		services: serviceRegistry,
	}
}

// HandleSlashCommand routes slash commands to appropriate handlers
func (h *InteractionHandler) HandleSlashCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	cmdName := i.ApplicationCommandData().Name

	switch cmdName {
	case "exercise":
		h.handleExerciseCommand(s, i)
	case "summary":
		h.handleSummaryCommand(s, i)
	default:
		logger.Error("Unknown command: %s", cmdName)
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("❌ Unknown command: %s", cmdName),
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
	}
}

// handleExerciseCommand handles the /exercise slash command
func (h *InteractionHandler) handleExerciseCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	userID := i.Member.User.ID
	username := i.Member.User.Username

	// Get exercise service from registry
	var exerciseService *services.ExerciseService
	for _, svc := range h.services.GetServices() {
		if es, ok := svc.(*services.ExerciseService); ok {
			exerciseService = es
			break
		}
	}

	if exerciseService == nil {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "❌ Exercise service not available.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	subcommand := i.ApplicationCommandData().Options[0].Name

	if subcommand == "quick" {
		// Quick log with defaults
		err := exerciseService.LogExerciseQuick(userID, username)
		if err != nil {
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: fmt.Sprintf("❌ Error logging exercise: %v", err),
					Flags:   discordgo.MessageFlagsEphemeral,
				},
			})
			return
		}

		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "✅ **Exercise logged!**\n" +
					"Workout: 30 minutes\n" +
					"Core/Mobility: 10 minutes\n\n" +
					"Use `/exercise detailed` for custom durations.",
				Flags: discordgo.MessageFlagsEphemeral,
			},
		})
	} else if subcommand == "detailed" {
		// Show modal for detailed input
		err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseModal,
			Data: &discordgo.InteractionResponseData{
				CustomID: "exercise_modal",
				Title:    "Log Exercise",
				Components: []discordgo.MessageComponent{
					discordgo.ActionsRow{
						Components: []discordgo.MessageComponent{
							discordgo.TextInput{
								CustomID:    "workout_duration",
								Label:       "Workout Duration (minutes)",
								Style:       discordgo.TextInputShort,
								Placeholder: "30",
								Required:    true,
								MinLength:   1,
								MaxLength:   3,
							},
						},
					},
					discordgo.ActionsRow{
						Components: []discordgo.MessageComponent{
							discordgo.TextInput{
								CustomID:    "workout_type",
								Label:       "Workout Type",
								Style:       discordgo.TextInputShort,
								Placeholder: "e.g., running, weights, cycling",
								Required:    false,
								MaxLength:   50,
							},
						},
					},
					discordgo.ActionsRow{
						Components: []discordgo.MessageComponent{
							discordgo.TextInput{
								CustomID:    "workout_location",
								Label:       "Location (indoor/outdoor)",
								Style:       discordgo.TextInputShort,
								Placeholder: "indoor or outdoor",
								Required:    false,
								MaxLength:   10,
							},
						},
					},
					discordgo.ActionsRow{
						Components: []discordgo.MessageComponent{
							discordgo.TextInput{
								CustomID:    "core_duration",
								Label:       "Core/Mobility Duration (minutes)",
								Style:       discordgo.TextInputShort,
								Placeholder: "10",
								Required:    true,
								MinLength:   1,
								MaxLength:   3,
							},
						},
					},
					discordgo.ActionsRow{
						Components: []discordgo.MessageComponent{
							discordgo.TextInput{
								CustomID:    "core_type",
								Label:       "Core/Mobility Type",
								Style:       discordgo.TextInputShort,
								Placeholder: "e.g., abs, planks, stretching, yoga",
								Required:    false,
								MaxLength:   50,
							},
						},
					},
				},
			},
		})
		if err != nil {
			logger.Error("Error responding to exercise command: %v", err)
		}
	}
}

// handleSummaryCommand handles the /summary slash command
func (h *InteractionHandler) handleSummaryCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// Get summary service from registry
	var summaryService *services.SummaryService
	for _, svc := range h.services.GetServices() {
		if ss, ok := svc.(*services.SummaryService); ok {
			summaryService = ss
			break
		}
	}

	if summaryService == nil {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "❌ Summary service not available.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	// Get optional user parameter
	var targetUsername string
	if len(i.ApplicationCommandData().Options) > 0 {
		targetUsername = i.ApplicationCommandData().Options[0].StringValue()
	}

	summary, err := summaryService.GetProgressSummary(targetUsername)
	if err != nil {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("❌ Error getting summary: %v", err),
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: summary,
		},
	})
}
