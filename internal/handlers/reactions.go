package handlers

import (
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/75-hard-discord-bot/internal/logger"
	"github.com/75-hard-discord-bot/internal/services"
)

// ReactionHandler handles message reaction events
type ReactionHandler struct {
	services *services.ServiceRegistry
}

// NewReactionHandler creates a new reaction handler
func NewReactionHandler(serviceRegistry *services.ServiceRegistry) *ReactionHandler {
	return &ReactionHandler{
		services: serviceRegistry,
	}
}

// HandleMessageReaction handles message reaction add events
func (h *ReactionHandler) HandleMessageReaction(s *discordgo.Session, r *discordgo.MessageReactionAdd) {
	// Ignore bot's own reactions
	if r.UserID == s.State.User.ID {
		return
	}

	// Get user information
	user, err := s.User(r.UserID)
	if err != nil {
		logger.Error("Error getting user: %v", err)
		return
	}

	// Get the message to check if it's our check-in message
	message, err := s.ChannelMessage(r.ChannelID, r.MessageID)
	if err != nil {
		logger.Error("Error getting message: %v", err)
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

		// Build confirmation message (only in dev mode)
		var confirmation string
		if logger.IsDevMode() {
			confirmation = fmt.Sprintf("✅ **Confirmation**\n"+
				"**User:** %s\n"+
				"**Emoji:** %s\n"+
				"Reaction received!", user.Username, emojiName)
		} else {
			// In production, just acknowledge silently or with minimal message
			confirmation = "✅ Check-in recorded!"
		}

		// If database is available and emoji is ✅ (or white_check_mark), record check-in
		emojiNameLower := strings.ToLower(r.Emoji.Name)
		isCheckMark := emojiNameLower == "✅" || emojiNameLower == "white_check_mark" || emojiNameLower == "check"

		// Get check-in service from registry
		var checkInService *services.CheckInService
		for _, svc := range h.services.GetServices() {
			if cs, ok := svc.(*services.CheckInService); ok {
				checkInService = cs
				break
			}
		}

		if checkInService != nil && isCheckMark {
			logger.Info("Processing check-in for user: %s (user_id=%s)", user.Username, r.UserID)
			dbInfo, err := checkInService.RecordCheckIn(r.UserID, user.Username)
			if err != nil {
				logger.Error("Error recording check-in: %v", err)
				if logger.IsDevMode() {
					confirmation += "\n\n⚠️ Database recording failed (see logs)"
				}
			} else if logger.IsDevMode() && dbInfo != "" {
				// Only show DB entries in dev mode
				confirmation += "\n\n" + dbInfo
			}
		}

		// Only send confirmation message in dev mode
		if logger.IsDevMode() {
			_, err = s.ChannelMessageSend(r.ChannelID, confirmation)
			if err != nil {
				logger.Error("Error sending confirmation: %v", err)
			}
		}
	}
}
