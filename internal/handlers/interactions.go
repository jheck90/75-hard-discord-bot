package handlers

import (
	"fmt"
	"strconv"
	"strings"
	"time"

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
	case "weighin":
		h.handleWeighInCommand(s, i)
	case "start":
		h.handleStartCommand(s, i)
	case "water":
		h.handleWaterCommand(s, i)
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

// handleWeighInCommand handles the /weighin slash command
func (h *InteractionHandler) handleWeighInCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	userID := i.Member.User.ID
	username := i.Member.User.Username

	// Get weigh-in service from registry
	var weighInService *services.WeighInService
	for _, svc := range h.services.GetServices() {
		if ws, ok := svc.(*services.WeighInService); ok {
			weighInService = ws
			break
		}
	}

	if weighInService == nil {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "❌ Weigh-in service not available.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	// Get weight from options
	var weight float64
	var notes string
	for _, option := range i.ApplicationCommandData().Options {
		switch option.Name {
		case "weight":
			weight = option.FloatValue()
		case "notes":
			notes = option.StringValue()
		}
	}

	// Validate weight
	if weight <= 0 || weight >= 1000 {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "❌ Weight must be between 0.01 and 999.99 pounds.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	// Record weigh-in
	err := weighInService.RecordWeighIn(userID, username, weight, notes)
	if err != nil {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("❌ Error recording weigh-in: %v", err),
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	// Get latest weigh-in for comparison
	latestWeight, challengeDay, err := weighInService.GetLatestWeighIn(userID)
	responseText := fmt.Sprintf("✅ **Weigh-in recorded!**\n**Weight:** %.2f lbs", weight)
	if err == nil && latestWeight != weight {
		diff := weight - latestWeight
		if diff > 0 {
			responseText += fmt.Sprintf("\n📈 **Change:** +%.2f lbs from last weigh-in (Day %d)", diff, challengeDay)
		} else {
			responseText += fmt.Sprintf("\n📉 **Change:** %.2f lbs from last weigh-in (Day %d)", diff, challengeDay)
		}
	}
	if notes != "" {
		responseText += fmt.Sprintf("\n📝 **Notes:** %s", notes)
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: responseText,
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}

// HandleButtonClick handles button click interactions
func (h *InteractionHandler) HandleButtonClick(s *discordgo.Session, i *discordgo.InteractionCreate) {
	customID := i.MessageComponentData().CustomID

	if strings.HasPrefix(customID, "start_confirm_") {
		h.handleStartConfirmation(s, i, customID)
	} else if strings.HasPrefix(customID, "start_cancel_") {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseUpdateMessage,
			Data: &discordgo.InteractionResponseData{
				Content: "❌ Challenge start cancelled.",
				Flags:   discordgo.MessageFlagsEphemeral,
				Components: []discordgo.MessageComponent{},
			},
		})
	}
}

// handleStartConfirmation handles the confirmation button click for starting challenge
func (h *InteractionHandler) handleStartConfirmation(s *discordgo.Session, i *discordgo.InteractionCreate, customID string) {
	userID := i.Member.User.ID
	username := i.Member.User.Username

	// Parse custom ID: start_confirm_{userID}_{timestamp}
	parts := strings.Split(customID, "_")
	if len(parts) < 4 {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "❌ Invalid confirmation. Please try /start again.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	// Get timestamp from custom ID
	timestampStr := parts[3]
	timestamp, err := strconv.ParseInt(timestampStr, 10, 64)
	if err != nil {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "❌ Invalid confirmation. Please try /start again.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	// Load MST location
	mst, err := time.LoadLocation("America/Denver")
	if err != nil {
		mst = time.FixedZone("MST", -7*3600)
	}

	startDate := time.Unix(timestamp, 0).In(mst)
	startDate = time.Date(startDate.Year(), startDate.Month(), startDate.Day(), 0, 0, 0, 0, mst)

	// Get user service
	var userService *services.UserService
	for _, svc := range h.services.GetServices() {
		if us, ok := svc.(*services.UserService); ok {
			userService = us
			break
		}
	}

	if userService == nil {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "❌ User service not available.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	// Start the challenge
	actualStartDate, endDate, err := userService.StartChallenge(userID, username, startDate)
	if err != nil {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseUpdateMessage,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("❌ Error starting challenge: %v", err),
				Flags:   discordgo.MessageFlagsEphemeral,
				Components: []discordgo.MessageComponent{},
			},
		})
		return
	}

	// Calculate challenge day (should be 1 on start date)
	challengeDay := 1
	now := time.Now().In(mst)
	if now.After(actualStartDate) {
		daysSinceStart := int(now.Sub(actualStartDate).Hours() / 24)
		if daysSinceStart >= 0 {
			challengeDay = daysSinceStart + 1
		}
	}

	startDateStr := actualStartDate.Format("January 2, 2006")
	endDateStr := endDate.Format("January 2, 2006")

	// Update the confirmation message
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("✅ **Challenge Started!**\n\n"+
				"📅 **Start Date:** %s (MST)\n"+
				"🏁 **End Date:** %s (MST)\n"+
				"📊 **Current Day:** Day %d\n\n"+
				"Good luck! You've got this! 💪", startDateStr, endDateStr, challengeDay),
			Flags:      discordgo.MessageFlagsEphemeral,
			Components: []discordgo.MessageComponent{},
		},
	})

	// Send public announcement
	announcement := fmt.Sprintf("🎉 **%s** has started the 75 Half Chub Challenge!\n\n"+
		"📅 Started on: **%s** (MST)\n"+
		"🏁 Challenge will complete on: **%s** (MST)\n"+
		"📊 Currently on: **Day %d**\n\n"+
		"Let's support them on this journey! 💪", username, startDateStr, endDateStr, challengeDay)

	_, err = s.ChannelMessageSend(i.ChannelID, announcement)
	if err != nil {
		logger.Error("Failed to send announcement: %v", err)
	}
}

// handleWaterCommand handles the /water slash command
func (h *InteractionHandler) handleWaterCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	userID := i.Member.User.ID
	username := i.Member.User.Username

	// Get water service from registry
	var waterService *services.WaterService
	for _, svc := range h.services.GetServices() {
		if ws, ok := svc.(*services.WaterService); ok {
			waterService = ws
			break
		}
	}

	if waterService == nil {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "❌ Water service not available.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	// Get subcommand
	subcommand := i.ApplicationCommandData().Options[0].Name

	if subcommand == "summary" {
		// Show today's total
		currentTotal, err := waterService.GetWaterIntake(userID)
		if err != nil {
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: fmt.Sprintf("❌ Error getting water intake: %v", err),
					Flags:   discordgo.MessageFlagsEphemeral,
				},
			})
			return
		}

		responseText := fmt.Sprintf("💧 **Today's Water Intake**\n**Total:** %.2f / 128 oz", currentTotal)
		if currentTotal >= 128.0 {
			responseText += "\n\n🎉 **Goal reached!** You've hit 1 gallon (128 oz)!"
		} else {
			remaining := 128.0 - currentTotal
			responseText += fmt.Sprintf("\n📊 **Remaining:** %.2f oz to reach 1 gallon", remaining)
		}

		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: responseText,
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	// Get ounces from subcommand options
	var ounces float64
	for _, option := range i.ApplicationCommandData().Options[0].Options {
		if option.Name == "ounces" {
			ounces = option.FloatValue()
			break
		}
	}

	// Validate ounces
	if ounces <= 0 {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "❌ Ounces must be greater than 0.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	var responseText string
	var err error
	var actualAmount, newTotal float64

	if subcommand == "subtract" {
		actualAmount, newTotal, err = waterService.SubtractWater(userID, username, ounces)
		if err != nil {
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: fmt.Sprintf("❌ Error subtracting water: %v", err),
					Flags:   discordgo.MessageFlagsEphemeral,
				},
			})
			return
		}
		responseText = fmt.Sprintf("💧 **Water subtracted!**\n**Subtracted:** %.2f oz\n**Total today:** %.2f / 128 oz", actualAmount, newTotal)
	} else if subcommand == "add" {
		actualAmount, newTotal, err = waterService.AddWater(userID, username, ounces)
		if err != nil {
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: fmt.Sprintf("❌ Error adding water: %v", err),
					Flags:   discordgo.MessageFlagsEphemeral,
				},
			})
			return
		}
		responseText = fmt.Sprintf("💧 **Water added!**\n**Added:** %.2f oz\n**Total today:** %.2f / 128 oz", actualAmount, newTotal)
		
		if newTotal >= 128.0 {
			responseText += "\n\n🎉 **Goal reached!** You've hit 1 gallon (128 oz)!"
		} else {
			remaining := 128.0 - newTotal
			responseText += fmt.Sprintf("\n📊 **Remaining:** %.2f oz to reach 1 gallon", remaining)
		}
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: responseText,
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}

// handleStartCommand handles the /start slash command
func (h *InteractionHandler) handleStartCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	userID := i.Member.User.ID

	// Get user service from registry
	var userService *services.UserService
	for _, svc := range h.services.GetServices() {
		if us, ok := svc.(*services.UserService); ok {
			userService = us
			break
		}
	}

	if userService == nil {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "❌ User service not available.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	// Parse date (default to today MST)
	var startDate time.Time
	dateStr := ""
	for _, option := range i.ApplicationCommandData().Options {
		if option.Name == "date" {
			dateStr = option.StringValue()
		}
	}

	// Load MST location
	mst, err := time.LoadLocation("America/Denver")
	if err != nil {
		mst = time.FixedZone("MST", -7*3600) // Fallback to UTC-7
	}

	if dateStr == "" {
		// Default to today in MST
		now := time.Now().In(mst)
		startDate = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, mst)
	} else {
		// Parse provided date (assume MST)
		parsedDate, err := time.ParseInLocation("2006-01-02", dateStr, mst)
		if err != nil {
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "❌ Invalid date format. Use YYYY-MM-DD (e.g., 2024-01-15)",
					Flags:   discordgo.MessageFlagsEphemeral,
				},
			})
			return
		}
		startDate = parsedDate
	}

	endDate := startDate.AddDate(0, 0, 75)
	startDateStr := startDate.Format("January 2, 2006")
	endDateStr := endDate.Format("January 2, 2006")

	// Show confirmation with rules
	rulesText := fmt.Sprintf("**75 Half Chub Challenge Rules:**\n\n"+
		"1. Follow a diet (no cheat meals, no alcohol)\n"+
		"2. One 30+ minute workout (indoor/outdoor doesn't matter; walking only counts with weight vest)\n"+
		"3. 10+ minutes of core/mobility\n"+
		"4. Drink 1 gallon of water (doesn't have to be plain)\n"+
		"5. 30 minutes of intentional self-improvement (reading, learning, journaling, studying, etc.)\n"+
		"6. Daily check-in (react with ✅)\n"+
		"7. Weekly progress photo\n"+
		"8. Finances: necessities only\n\n"+
		"**Challenge Details:**\n"+
		"📅 **Start Date:** %s (MST)\n"+
		"🏁 **End Date:** %s (MST)\n"+
		"📊 **Duration:** 75 days (base)\n\n"+
		"⚠️ **Failure Rule:** If you miss any task, add 7 days to your end date. You may publicly request forgiveness for emergencies (sick kids, etc.) to waive penalties.\n\n"+
		"Ready to begin?", startDateStr, endDateStr)

	// Store start date in custom ID for button handler
	customID := fmt.Sprintf("start_confirm_%s_%d", userID, startDate.Unix())

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: rulesText,
			Flags:   discordgo.MessageFlagsEphemeral,
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.Button{
							Label:    "Yes, Start Challenge",
							Style:    discordgo.SuccessButton,
							CustomID: customID,
						},
						discordgo.Button{
							Label:    "Cancel",
							Style:    discordgo.DangerButton,
							CustomID: fmt.Sprintf("start_cancel_%s", userID),
						},
					},
				},
			},
		},
	})
}
