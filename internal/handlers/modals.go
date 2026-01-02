package handlers

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
	"github.com/75-hard-discord-bot/internal/logger"
	"github.com/75-hard-discord-bot/internal/services"
)

// ModalHandler handles modal submission interactions
type ModalHandler struct {
	services *services.ServiceRegistry
}

// NewModalHandler creates a new modal handler
func NewModalHandler(serviceRegistry *services.ServiceRegistry) *ModalHandler {
	return &ModalHandler{
		services: serviceRegistry,
	}
}

// HandleModalSubmit routes modal submissions to appropriate handlers
func (h *ModalHandler) HandleModalSubmit(s *discordgo.Session, i *discordgo.InteractionCreate) {
	customID := i.ModalSubmitData().CustomID

	switch customID {
	case "exercise_modal":
		h.handleExerciseModal(s, i)
	default:
		logger.Error("Unknown modal: %s", customID)
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("❌ Unknown modal: %s", customID),
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
	}
}

// handleExerciseModal handles the exercise modal submission
func (h *ModalHandler) handleExerciseModal(s *discordgo.Session, i *discordgo.InteractionCreate) {
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

	data := i.ModalSubmitData()
	workoutDurationStr := data.Components[0].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value
	workoutType := data.Components[1].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value
	workoutLocation := data.Components[2].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value
	coreDurationStr := data.Components[3].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value
	coreType := data.Components[4].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value

	// Parse durations
	var workoutDuration, coreDuration int
	fmt.Sscanf(workoutDurationStr, "%d", &workoutDuration)
	fmt.Sscanf(coreDurationStr, "%d", &coreDuration)

	// Validate minimums
	if workoutDuration < 30 {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "❌ Workout duration must be at least 30 minutes.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	if coreDuration < 10 {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "❌ Core/mobility duration must be at least 10 minutes.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	// Set defaults for empty fields
	if workoutType == "" {
		workoutType = "general"
	}
	if workoutLocation == "" {
		workoutLocation = "indoor"
	}
	if coreType == "" {
		coreType = "general"
	}

	err := exerciseService.LogExerciseDetailed(userID, username, workoutDuration, workoutType, workoutLocation, coreDuration, coreType)
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
			Content: fmt.Sprintf("✅ **Exercise logged!**\n"+
				"**Workout:** %d minutes (%s, %s)\n"+
				"**Core/Mobility:** %d minutes (%s)",
				workoutDuration, workoutType, workoutLocation, coreDuration, coreType),
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})
}
