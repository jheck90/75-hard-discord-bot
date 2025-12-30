package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/75-hard-discord-bot/internal/database"
)

// Webhook functionality commented out - using bot only
// type DiscordWebhookPayload struct {
// 	Content string `json:"content"`
// }

var db *sql.DB

func main() {
	// Initialize database connection (optional - app can run without DB)
	log.Println("🔌 Initializing database connection...")
	var err error
	db, err = database.ConnectOrSkip()
	if err != nil {
		log.Fatalf("❌ Failed to connect to database: %v", err)
	}
	if db != nil {
		log.Println("✅ Database connected and migrations applied")
		defer db.Close()
	} else {
		log.Println("⚠️  No database configured - database features will be unavailable")
	}

	// Webhook functionality commented out - using bot only
	// webhookURL := os.Getenv("DISCORD_WEBHOOK_URL")
	// if webhookURL == "" {
	// 	fmt.Println("Error: DISCORD_WEBHOOK_URL environment variable is not set")
	// 	os.Exit(1)
	// }
	//
	// fmt.Println("75 Hard Discord Bot - Webhook Test")
	// fmt.Println("==================================")
	// fmt.Printf("Webhook URL: %s\n", maskWebhookURL(webhookURL))
	// fmt.Println()
	//
	// // Send ping message
	// fmt.Println("Sending ping message to Discord...")
	// err = sendWebhookMessage(webhookURL, "🏓 **Ping!** Bot is alive and responding!")
	// if err != nil {
	// 	fmt.Printf("❌ Error sending message: %v\n", err)
	// 	os.Exit(1)
	// }
	//
	// fmt.Println("✅ Pong! Message sent successfully to Discord webhook")

	// Run bot by default
	runBot()
}

// Webhook functionality commented out - using bot only
// func sendWebhookMessage(webhookURL, message string) error {
// 	payload := DiscordWebhookPayload{
// 		Content: message,
// 	}
//
// 	jsonData, err := json.Marshal(payload)
// 	if err != nil {
// 		return fmt.Errorf("failed to marshal JSON: %w", err)
// 	}
//
// 	req, err := http.NewRequest("POST", webhookURL, bytes.NewBuffer(jsonData))
// 	if err != nil {
// 		return fmt.Errorf("failed to create request: %w", err)
// 	}
//
// 	req.Header.Set("Content-Type", "application/json")
//
// 	client := &http.Client{
// 		Timeout: 10 * time.Second,
// 	}
//
// 	resp, err := client.Do(req)
// 	if err != nil {
// 		return fmt.Errorf("failed to send request: %w", err)
// 	}
// 	defer resp.Body.Close()
//
// 	body, err := io.ReadAll(resp.Body)
// 	if err != nil {
// 		return fmt.Errorf("failed to read response: %w", err)
// 	}
//
// 	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
// 		return fmt.Errorf("webhook returned error status %d: %s", resp.StatusCode, string(body))
// 	}
//
// 	return nil
// }
//
// func maskWebhookURL(url string) string {
// 	if len(url) < 20 {
// 		return url
// 	}
// 	return url[:20] + "..." + url[len(url)-10:]
// }

// recordCheckInAndGetDBInfo records a check-in for the user and returns formatted DB entry info
func recordCheckInAndGetDBInfo(userID, username string) (string, error) {
	if db == nil {
		return "", fmt.Errorf("database not available")
	}

	// Ensure user exists in database (create if not exists)
	err := ensureUserExists(userID, username)
	if err != nil {
		return "", fmt.Errorf("failed to ensure user exists: %w", err)
	}

	// Get current challenge day for user
	challengeDay, err := getCurrentChallengeDay(userID)
	if err != nil {
		return "", fmt.Errorf("failed to get challenge day: %w", err)
	}

	// Record check-in (this will trigger auto-population of all feat tables)
	result, err := db.Exec(
		`INSERT INTO accountability_checkins (user_id, challenge_day, check_in_method) 
		 VALUES ($1, $2, $3) 
		 ON CONFLICT (user_id, challenge_day) DO UPDATE SET completed_at = NOW()`,
		userID, challengeDay, "emoji_reaction",
	)
	if err != nil {
		return "", fmt.Errorf("failed to record check-in: %w", err)
	}
	
	// Log if this was a new insert (trigger should fire)
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected > 0 {
		log.Printf("✅ Check-in recorded for user %s, day %d (trigger should fire)", userID, challengeDay)
	} else {
		log.Printf("⚠️ Check-in updated for user %s, day %d (trigger may not fire on UPDATE)", userID, challengeDay)
	}

	// Query all feat tables to show what was created
	dbInfo, err := getDBEntriesInfo(userID, challengeDay)
	if err != nil {
		return "", fmt.Errorf("failed to get DB entries info: %w", err)
	}

	return dbInfo, nil
}

// ensureUserExists creates a user record if it doesn't exist
func ensureUserExists(userID, username string) error {
	now := time.Now()
	startDate := now.Format("2006-01-02")
	endDate := now.AddDate(0, 0, 75).Format("2006-01-02")

	_, err := db.Exec(
		`INSERT INTO users (user_id, username, challenge_start_date, original_challenge_end_date, current_challenge_end_date)
		 VALUES ($1, $2, $3, $4, $5)
		 ON CONFLICT (user_id) DO UPDATE SET username = EXCLUDED.username`,
		userID, username, startDate, endDate, endDate,
	)
	return err
}

// getCurrentChallengeDay calculates the current challenge day for a user
func getCurrentChallengeDay(userID string) (int, error) {
	var startDate time.Time
	err := db.QueryRow(
		`SELECT challenge_start_date FROM users WHERE user_id = $1`,
		userID,
	).Scan(&startDate)
	if err != nil {
		return 0, err
	}

	daysSinceStart := int(time.Since(startDate).Hours() / 24)
	if daysSinceStart < 0 {
		daysSinceStart = 0
	}
	return daysSinceStart + 1, nil
}

// getDBEntriesInfo queries all feat tables and returns formatted info
func getDBEntriesInfo(userID string, challengeDay int) (string, error) {
	var info strings.Builder
	info.WriteString("📊 **Database Entries Created:**\n```\n")

	// Check accountability check-in
	var checkInTime time.Time
	err := db.QueryRow(
		`SELECT completed_at FROM accountability_checkins WHERE user_id = $1 AND challenge_day = $2`,
		userID, challengeDay,
	).Scan(&checkInTime)
	if err == nil {
		info.WriteString(fmt.Sprintf("✅ Accountability Check-in: %s\n", checkInTime.Format("2006-01-02 15:04:05")))
	}

	// Check exercise
	var exerciseWorkout, exerciseCore sql.NullInt64
	err = db.QueryRow(
		`SELECT workout_duration_minutes, core_mobility_duration_minutes 
		 FROM exercise_completions WHERE user_id = $1 AND challenge_day = $2`,
		userID, challengeDay,
	).Scan(&exerciseWorkout, &exerciseCore)
	if err == nil {
		info.WriteString(fmt.Sprintf("💪 Exercise: %d min workout + %d min core/mobility\n",
			exerciseWorkout.Int64, exerciseCore.Int64))
	}

	// Check diet
	var dietCheatMeal, dietAlcohol sql.NullBool
	err = db.QueryRow(
		`SELECT cheat_meal, alcohol_consumed FROM diet_completions WHERE user_id = $1 AND challenge_day = $2`,
		userID, challengeDay,
	).Scan(&dietCheatMeal, &dietAlcohol)
	if err == nil {
		info.WriteString("🍽️  Diet: Compliant (no cheat meals, no alcohol)\n")
	}

	// Check water
	var waterAmount sql.NullFloat64
	err = db.QueryRow(
		`SELECT amount_ounces FROM water_completions WHERE user_id = $1 AND challenge_day = $2`,
		userID, challengeDay,
	).Scan(&waterAmount)
	if err == nil {
		info.WriteString(fmt.Sprintf("💧 Water: %.2f oz (1 gallon)\n", waterAmount.Float64))
	}

	// Check self-improvement
	var selfImproveDuration sql.NullInt64
	err = db.QueryRow(
		`SELECT duration_minutes FROM self_improvement_completions WHERE user_id = $1 AND challenge_day = $2`,
		userID, challengeDay,
	).Scan(&selfImproveDuration)
	if err == nil {
		info.WriteString(fmt.Sprintf("📚 Self-Improvement: %d minutes\n", selfImproveDuration.Int64))
	}

	// Check finances
	var financesStatus sql.NullString
	err = db.QueryRow(
		`SELECT compliance_status FROM finances_completions WHERE user_id = $1 AND challenge_day = $2`,
		userID, challengeDay,
	).Scan(&financesStatus)
	if err == nil {
		info.WriteString(fmt.Sprintf("💰 Finances: %s\n", financesStatus.String))
	}

	info.WriteString("```")
	return info.String(), nil
}

// runBot starts the Discord bot
func runBot() {
	botToken := os.Getenv("DISCORD_BOT_TOKEN")
	if botToken == "" {
		fmt.Println("Error: DISCORD_BOT_TOKEN environment variable is not set")
		fmt.Println("To run the bot, you need a Discord bot token.")
		fmt.Println("Create a bot at https://discord.com/developers/applications")
		os.Exit(1)
	}

	channelID := os.Getenv("DISCORD_CHANNEL_ID")
	if channelID == "" {
		fmt.Println("Error: DISCORD_CHANNEL_ID environment variable is not set")
		fmt.Println("Please set DISCORD_CHANNEL_ID to the channel where the bot should operate")
		os.Exit(1)
	}

	// Create a new Discord session
	dg, err := discordgo.New("Bot " + botToken)
	if err != nil {
		fmt.Printf("Error creating Discord session: %v\n", err)
		os.Exit(1)
	}

	// Register intents needed for slash commands and interactions
	dg.Identify.Intents = discordgo.IntentsGuildMessages | discordgo.IntentsGuildMessageReactions | discordgo.IntentsGuilds

	// Register slash command handler
	dg.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		if i.Type == discordgo.InteractionApplicationCommand {
			cmdName := i.ApplicationCommandData().Name
			if cmdName == "exercise" {
				handleExerciseCommand(s, i)
			} else if cmdName == "summary" {
				handleSummaryCommand(s, i)
			}
		}
	})

	// Register modal submit handler
	dg.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		if i.Type == discordgo.InteractionModalSubmit {
			if i.ModalSubmitData().CustomID == "exercise_modal" {
				handleExerciseModal(s, i)
			}
		}
	})

	// Register message reaction add handler
	dg.AddHandler(func(s *discordgo.Session, r *discordgo.MessageReactionAdd) {
		// Ignore bot's own reactions
		if r.UserID == s.State.User.ID {
			return
		}

		// Get user information
		user, err := s.User(r.UserID)
		if err != nil {
			fmt.Printf("Error getting user: %v\n", err)
			return
		}

		// Get the message to check if it's our check-in message
		message, err := s.ChannelMessage(r.ChannelID, r.MessageID)
		if err != nil {
			fmt.Printf("Error getting message: %v\n", err)
			return
		}

		// Check if this is our check-in message
		if message.Author.ID == s.State.User.ID && 
		   message.Content == "emoji this message to ping" {
			// Format emoji name
			emojiName := r.Emoji.Name
			if r.Emoji.ID != "" {
				emojiName = fmt.Sprintf("<:%s:%s>", r.Emoji.Name, r.Emoji.ID)
			}

			// Build confirmation message
			confirmation := fmt.Sprintf("✅ **Confirmation**\n"+
				"**User:** %s\n"+
				"**Emoji:** %s\n"+
				"Reaction received!", user.Username, emojiName)

			// If database is available and emoji is ✅ (or white_check_mark), record check-in and show DB entries
			emojiNameLower := strings.ToLower(r.Emoji.Name)
			isCheckMark := emojiNameLower == "✅" || emojiNameLower == "white_check_mark" || emojiNameLower == "check"
			if db != nil && isCheckMark {
				dbInfo, err := recordCheckInAndGetDBInfo(r.UserID, user.Username)
				if err != nil {
					log.Printf("Error recording check-in: %v\n", err)
					confirmation += "\n\n⚠️ Database recording failed (see logs)"
				} else {
					confirmation += "\n\n" + dbInfo
				}
			}

			_, err = s.ChannelMessageSend(r.ChannelID, confirmation)
			if err != nil {
				fmt.Printf("Error sending confirmation: %v\n", err)
			}
		}
	})

	// Open websocket connection
	err = dg.Open()
	if err != nil {
		fmt.Printf("Error opening connection: %v\n", err)
		os.Exit(1)
	}
	defer dg.Close()

	// Register slash commands
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
	for _, cmd := range commands {
		_, err := dg.ApplicationCommandCreate(dg.State.User.ID, "", cmd)
		if err != nil {
			log.Printf("Cannot create command '%s': %v", cmd.Name, err)
		} else {
			log.Printf("✅ Registered command: /%s", cmd.Name)
		}
	}

	fmt.Println("75 Hard Discord Bot")
	fmt.Println("===================")
	if db != nil {
		fmt.Println("✅ Database connected - check-ins will be recorded")
	} else {
		fmt.Println("⚠️  No database configured - check-ins will not be recorded")
	}
	fmt.Println("Bot is now running and listening for commands and reactions...")
	fmt.Println()

	// Send the check-in message
	testMessage := "emoji this message to ping"
	msg, err := dg.ChannelMessageSend(channelID, testMessage)
	if err != nil {
		fmt.Printf("❌ Error sending check-in message: %v\n", err)
		os.Exit(1)
	}

	// Add a self-reaction so users can easily click it
	err = dg.MessageReactionAdd(channelID, msg.ID, "✅")
	if err != nil {
		fmt.Printf("⚠️  Warning: Could not add self-reaction: %v\n", err)
		fmt.Println("   Users can still react manually")
	}

	fmt.Printf("✅ Check-in message sent to channel %s\n", channelID)
	fmt.Printf("   Message ID: %s\n", msg.ID)
	fmt.Println("   Bot has added ✅ reaction - users can click it to check in!")
	fmt.Println()
	fmt.Println("Waiting for check-ins... (Press Ctrl+C to stop)")

	// Wait for interrupt signal
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	fmt.Println("\nShutting down...")
}

// handleExerciseCommand handles the /exercise slash command
func handleExerciseCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	userID := i.Member.User.ID
	username := i.Member.User.Username

	// Check if database is available
	if db == nil {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "❌ Database not configured. Exercise logging is unavailable.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	subcommand := i.ApplicationCommandData().Options[0].Name

	if subcommand == "quick" {
		// Quick log with defaults
		err := logExerciseQuick(userID, username)
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
								MinLength:    1,
								MaxLength:    3,
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
								MinLength:    1,
								MaxLength:    3,
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
			log.Printf("Error responding to exercise command: %v", err)
		}
	}
}

// handleExerciseModal handles the exercise modal submission
func handleExerciseModal(s *discordgo.Session, i *discordgo.InteractionCreate) {
	userID := i.Member.User.ID
	username := i.Member.User.Username

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

	err := logExerciseDetailed(userID, username, workoutDuration, workoutType, workoutLocation, coreDuration, coreType)
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

// logExerciseQuick logs exercise with default values
func logExerciseQuick(userID, username string) error {
	return logExerciseDetailed(userID, username, 30, "general", "indoor", 10, "general")
}

// logExerciseDetailed logs exercise with provided details
func logExerciseDetailed(userID, username string, workoutDuration int, workoutType, workoutLocation string, coreDuration int, coreType string) error {
	if db == nil {
		return fmt.Errorf("database not available")
	}

	// Ensure user exists
	err := ensureUserExists(userID, username)
	if err != nil {
		return fmt.Errorf("failed to ensure user exists: %w", err)
	}

	// Get current challenge day
	challengeDay, err := getCurrentChallengeDay(userID)
	if err != nil {
		return fmt.Errorf("failed to get challenge day: %w", err)
	}

	// Insert or update exercise completion (mark as manual entry)
	_, err = db.Exec(
		`INSERT INTO exercise_completions 
		 (user_id, challenge_day, workout_duration_minutes, workout_type, workout_location, core_mobility_duration_minutes, core_mobility_type, autopopulated)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, false)
		 ON CONFLICT (user_id, challenge_day) 
		 DO UPDATE SET 
			workout_duration_minutes = EXCLUDED.workout_duration_minutes,
			workout_type = EXCLUDED.workout_type,
			workout_location = EXCLUDED.workout_location,
			core_mobility_duration_minutes = EXCLUDED.core_mobility_duration_minutes,
			core_mobility_type = EXCLUDED.core_mobility_type,
			autopopulated = false,
			completed_at = NOW()`,
		userID, challengeDay, workoutDuration, workoutType, workoutLocation, coreDuration, coreType,
	)
	return err
}

// handleSummaryCommand handles the /summary slash command
func handleSummaryCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if db == nil {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "❌ Database not configured. Summary is unavailable.",
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

	summary, err := getProgressSummary(targetUsername)
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

// getProgressSummary returns a formatted progress summary
func getProgressSummary(targetUsername string) (string, error) {
	if targetUsername == "" {
		return getAllUsersSummary()
	}
	return getUserSummary(targetUsername)
}

// getAllUsersSummary returns summary for all users
func getAllUsersSummary() (string, error) {
	query := `
		SELECT 
			u.user_id,
			u.username,
			u.challenge_start_date,
			u.current_challenge_end_date,
			u.days_added,
			COUNT(DISTINCT a.challenge_day) as checkin_days,
			COUNT(DISTINCT e.challenge_day) as exercise_days,
			COUNT(DISTINCT d.challenge_day) as diet_days,
			COUNT(DISTINCT w.challenge_day) as water_days,
			COUNT(DISTINCT si.challenge_day) as self_improvement_days,
			COUNT(DISTINCT CASE WHEN f.compliance_status = 'compliant' THEN f.challenge_day END) as finances_days
		FROM users u
		LEFT JOIN accountability_checkins a ON a.user_id = u.user_id
		LEFT JOIN exercise_completions e ON e.user_id = u.user_id
		LEFT JOIN diet_completions d ON d.user_id = u.user_id
		LEFT JOIN water_completions w ON w.user_id = u.user_id
		LEFT JOIN self_improvement_completions si ON si.user_id = u.user_id
		LEFT JOIN finances_completions f ON f.user_id = u.user_id
		GROUP BY u.user_id, u.username, u.challenge_start_date, u.current_challenge_end_date, u.days_added
		ORDER BY checkin_days DESC, u.username
	`

	rows, err := db.Query(query)
	if err != nil {
		return "", fmt.Errorf("failed to query users: %w", err)
	}
	defer rows.Close()

	var summary strings.Builder
	summary.WriteString("📊 **Challenge Progress Summary (All Users)**\n\n")

	for rows.Next() {
		var userID, username string
		var startDate, endDate time.Time
		var daysAdded, checkinDays, exerciseDays, dietDays, waterDays, selfImproveDays, financesDays sql.NullInt64

		err := rows.Scan(&userID, &username, &startDate, &endDate, &daysAdded, &checkinDays, &exerciseDays, &dietDays, &waterDays, &selfImproveDays, &financesDays)
		if err != nil {
			return "", fmt.Errorf("failed to scan user row: %w", err)
		}

		totalDays := int(endDate.Sub(startDate).Hours() / 24)
		currentDay := int(time.Since(startDate).Hours()/24) + 1
		if currentDay > totalDays {
			currentDay = totalDays
		}

		summary.WriteString(fmt.Sprintf("**%s** (Day %d/%d", username, currentDay, totalDays))
		if daysAdded.Int64 > 0 {
			summary.WriteString(fmt.Sprintf(" +%d", daysAdded.Int64))
		}
		summary.WriteString(")\n")
		summary.WriteString(fmt.Sprintf("  ✅ Check-ins: %d | 💪 Exercise: %d | 🍽️ Diet: %d | 💧 Water: %d | 📚 Self-Improve: %d | 💰 Finances: %d\n\n",
			checkinDays.Int64, exerciseDays.Int64, dietDays.Int64, waterDays.Int64, selfImproveDays.Int64, financesDays.Int64))
	}

	if summary.Len() == len("📊 **Challenge Progress Summary (All Users)**\n\n") {
		summary.WriteString("No users found.")
	}

	return summary.String(), nil
}

// getUserSummary returns summary for a specific user
func getUserSummary(username string) (string, error) {
	query := `
		SELECT 
			u.user_id,
			u.username,
			u.challenge_start_date,
			u.current_challenge_end_date,
			u.days_added,
			COUNT(DISTINCT a.challenge_day) as checkin_days,
			COUNT(DISTINCT e.challenge_day) as exercise_days,
			COUNT(DISTINCT d.challenge_day) as diet_days,
			COUNT(DISTINCT w.challenge_day) as water_days,
			COUNT(DISTINCT si.challenge_day) as self_improvement_days,
			COUNT(DISTINCT CASE WHEN f.compliance_status = 'compliant' THEN f.challenge_day END) as finances_days
		FROM users u
		LEFT JOIN accountability_checkins a ON a.user_id = u.user_id
		LEFT JOIN exercise_completions e ON e.user_id = u.user_id
		LEFT JOIN diet_completions d ON d.user_id = u.user_id
		LEFT JOIN water_completions w ON w.user_id = u.user_id
		LEFT JOIN self_improvement_completions si ON si.user_id = u.user_id
		LEFT JOIN finances_completions f ON f.user_id = u.user_id
		WHERE LOWER(u.username) = LOWER($1)
		GROUP BY u.user_id, u.username, u.challenge_start_date, u.current_challenge_end_date, u.days_added
	`

	var userID, dbUsername string
	var startDate, endDate time.Time
	var daysAdded, checkinDays, exerciseDays, dietDays, waterDays, selfImproveDays, financesDays sql.NullInt64

	err := db.QueryRow(query, username).Scan(&userID, &dbUsername, &startDate, &endDate, &daysAdded, &checkinDays, &exerciseDays, &dietDays, &waterDays, &selfImproveDays, &financesDays)
	if err == sql.ErrNoRows {
		return fmt.Sprintf("❌ User '%s' not found.", username), nil
	}
	if err != nil {
		return "", fmt.Errorf("failed to query user: %w", err)
	}

	totalDays := int(endDate.Sub(startDate).Hours() / 24)
	currentDay := int(time.Since(startDate).Hours()/24) + 1
	if currentDay > totalDays {
		currentDay = totalDays
	}

	var summary strings.Builder
	summary.WriteString(fmt.Sprintf("📊 **Challenge Progress Summary: %s**\n\n", dbUsername))
	summary.WriteString(fmt.Sprintf("**Challenge:** Day %d/%d", currentDay, totalDays))
	if daysAdded.Int64 > 0 {
		summary.WriteString(fmt.Sprintf(" (+%d days added)", daysAdded.Int64))
	}
	summary.WriteString(fmt.Sprintf("\n**Started:** %s\n\n", startDate.Format("January 2, 2006")))

	summary.WriteString("**Feat Completions:**\n")
	summary.WriteString(fmt.Sprintf("  ✅ Accountability Check-ins: %d days\n", checkinDays.Int64))
	summary.WriteString(fmt.Sprintf("  💪 Exercise: %d days\n", exerciseDays.Int64))
	summary.WriteString(fmt.Sprintf("  🍽️  Diet: %d days\n", dietDays.Int64))
	summary.WriteString(fmt.Sprintf("  💧 Water: %d days\n", waterDays.Int64))
	summary.WriteString(fmt.Sprintf("  📚 Self-Improvement: %d days\n", selfImproveDays.Int64))
	summary.WriteString(fmt.Sprintf("  💰 Finances: %d days\n", financesDays.Int64))

	// Calculate completion percentage
	allFeatsCompleted := min(checkinDays.Int64, exerciseDays.Int64, dietDays.Int64, waterDays.Int64, selfImproveDays.Int64, financesDays.Int64)
	completionRate := float64(allFeatsCompleted) / float64(totalDays) * 100
	summary.WriteString(fmt.Sprintf("\n**Overall Progress:** %.1f%% (%d/%d days with all feats completed)", completionRate, allFeatsCompleted, totalDays))

	return summary.String(), nil
}

// min returns the minimum of multiple int64 values
func min(values ...int64) int64 {
	if len(values) == 0 {
		return 0
	}
	minVal := values[0]
	for _, v := range values[1:] {
		if v < minVal {
			minVal = v
		}
	}
	return minVal
}

