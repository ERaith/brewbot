package bot

import (
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
)

// updateBlackboard finds or creates the #blackboard channel and edits the live message.
// Call this after any rating, recipe submit, fg update, or brew completion.
func (b *Bot) updateBlackboard(s *discordgo.Session, guildID string) {
	channelID := b.db.GetConfig(guildID, "blackboard_channel_id")
	messageID := b.db.GetConfig(guildID, "blackboard_message_id")

	content, err := b.buildBlackboard(guildID)
	if err != nil {
		return
	}

	// Create channel if we don't have one yet
	if channelID == "" {
		ch, err := s.GuildChannelCreate(guildID, "blackboard", discordgo.ChannelTypeGuildText)
		if err != nil {
			return
		}
		channelID = ch.ID
		b.db.SetConfig(guildID, "blackboard_channel_id", channelID)
	}

	// Post or edit the message
	if messageID == "" {
		msg, err := s.ChannelMessageSend(channelID, content)
		if err != nil {
			return
		}
		s.ChannelMessagePin(channelID, msg.ID)
		b.db.SetConfig(guildID, "blackboard_message_id", msg.ID)
	} else {
		if _, err := s.ChannelMessageEdit(channelID, messageID, content); err != nil {
			// Message was deleted — post a new one
			msg, err := s.ChannelMessageSend(channelID, content)
			if err != nil {
				return
			}
			s.ChannelMessagePin(channelID, msg.ID)
			b.db.SetConfig(guildID, "blackboard_message_id", msg.ID)
		}
	}
}

func (b *Bot) buildBlackboard(guildID string) (string, error) {
	entries, err := b.db.GetBlackboard(guildID)
	if err != nil {
		return "", err
	}

	var sb strings.Builder
	sb.WriteString("# 📋 The Blackboard\n\n")

	if len(entries) == 0 {
		sb.WriteString("_No completed brews yet._")
		return sb.String(), nil
	}

	for _, e := range entries {
		name := e.Brew.Name
		if name == "" {
			name = "Unnamed Brew"
		}

		sb.WriteString(fmt.Sprintf("## 🍺 %s\n", name))
		sb.WriteString(fmt.Sprintf("**Brewer:** <@%s>", e.Brew.BrewerID))
		if e.Brew.Date != "" {
			sb.WriteString(fmt.Sprintf("  **Date:** %s", e.Brew.Date))
		}
		if e.Recipe != nil && e.Recipe.Style != "" {
			sb.WriteString(fmt.Sprintf("  **Style:** %s", e.Recipe.Style))
		}
		if e.Recipe != nil && e.Recipe.ABV > 0 {
			sb.WriteString(fmt.Sprintf("  **ABV:** %.1f%%", e.Recipe.ABV))
		}
		sb.WriteString("\n")

		if e.AvgRating > 0 {
			full := int(e.AvgRating + 0.5)
			stars := strings.Repeat("⭐", full) + strings.Repeat("☆", 5-full)
			sb.WriteString(fmt.Sprintf("**Rating:** %s %.1f / 5 (%d votes)\n", stars, e.AvgRating, len(e.Ratings)))
		} else {
			sb.WriteString("**Rating:** _no votes yet_\n")
		}

		if e.Brew.ChannelID != "" {
			sb.WriteString(fmt.Sprintf("**Channel:** <#%s>\n", e.Brew.ChannelID))
		}
		sb.WriteString("\n")
	}

	sb.WriteString("_Updates automatically when ratings or recipes change._")
	return sb.String(), nil
}
