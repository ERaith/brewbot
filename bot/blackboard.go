package bot

import (
	"fmt"
	"log"
	"strings"

	"github.com/bwmarrin/discordgo"
)

// updateBlackboard finds or creates the #blackboard channel and edits the live message.
// Call this after any rating, recipe submit, fg update, or brew completion.
func (b *Bot) updateBlackboard(s *discordgo.Session, guildID string) {
	channelID := b.db.GetConfig(guildID, "blackboard_channel_id")
	messageID := b.db.GetConfig(guildID, "blackboard_message_id")
	log.Printf("blackboard: guild=%s channel=%s message=%s", guildID, channelID, messageID)

	content, err := b.buildBlackboard(guildID)
	if err != nil {
		log.Printf("blackboard: buildBlackboard error: %v", err)
		return
	}
	log.Printf("blackboard: content length=%d", len(content))

	// Create channel if we don't have one yet
	if channelID == "" {
		ch, err := s.GuildChannelCreate(guildID, "blackboard", discordgo.ChannelTypeGuildText)
		if err != nil {
			log.Printf("blackboard: create channel error: %v", err)
			return
		}
		channelID = ch.ID
		b.db.SetConfig(guildID, "blackboard_channel_id", channelID)
	}

	// Post or edit the message
	if messageID == "" {
		msg, err := s.ChannelMessageSend(channelID, content)
		if err != nil {
			log.Printf("blackboard: send message error: %v", err)
			return
		}
		s.ChannelMessagePin(channelID, msg.ID)
		b.db.SetConfig(guildID, "blackboard_message_id", msg.ID)
		log.Printf("blackboard: posted message %s", msg.ID)
	} else {
		if _, err := s.ChannelMessageEdit(channelID, messageID, content); err != nil {
			log.Printf("blackboard: edit error: %v — reposting", err)
			msg, err := s.ChannelMessageSend(channelID, content)
			if err != nil {
				log.Printf("blackboard: repost error: %v", err)
				return
			}
			s.ChannelMessagePin(channelID, msg.ID)
			b.db.SetConfig(guildID, "blackboard_message_id", msg.ID)
		}
	}
}

func (b *Bot) buildBlackboard(guildID string) (string, error) {
	log.Printf("blackboard: querying entries for guild %s", guildID)
	entries, err := b.db.GetBlackboard(guildID)
	if err != nil {
		return "", err
	}
	log.Printf("blackboard: got %d entries", len(entries))

	var sb strings.Builder
	sb.WriteString("### 📋 Brews\n\n")

	if len(entries) == 0 {
		sb.WriteString("_No completed brews yet._")
		return sb.String(), nil
	}

	for idx, e := range entries {
		if idx > 0 {
			sb.WriteString("───────────────────\n")
		}

		name := e.Brew.Name
		if name == "" {
			name = "Unnamed Brew"
		}

		sb.WriteString(fmt.Sprintf("🍺 **%s**\n", name))

		sb.WriteString(fmt.Sprintf("Brewer: <@%s>", e.Brew.BrewerID))
		if e.Brew.Date != "" {
			sb.WriteString(fmt.Sprintf(" · %s", e.Brew.Date))
		}
		sb.WriteString("\n")

		if e.Recipe != nil && (e.Recipe.Style != "" || e.Recipe.ABV > 0) {
			if e.Recipe.Style != "" {
				sb.WriteString(fmt.Sprintf("Style: %s", e.Recipe.Style))
			}
			if e.Recipe.ABV > 0 {
				if e.Recipe.Style != "" {
					sb.WriteString(fmt.Sprintf(" · ABV: %.1f%%", e.Recipe.ABV))
				} else {
					sb.WriteString(fmt.Sprintf("ABV: %.1f%%", e.Recipe.ABV))
				}
			}
			sb.WriteString("\n")
		}

		if e.AvgRating > 0 {
			full := int(e.AvgRating + 0.5)
			stars := strings.Repeat("⭐", full) + strings.Repeat("☆", 5-full)
			sb.WriteString(fmt.Sprintf("%s %.1f / 5 (%d votes)\n", stars, e.AvgRating, len(e.Ratings)))
		} else {
			sb.WriteString("_no votes yet_\n")
		}

		if e.Brew.ChannelID != "" {
			sb.WriteString(fmt.Sprintf("<#%s>\n", e.Brew.ChannelID))
		}
	}

	sb.WriteString("\n_Updates automatically when ratings or recipes change._")
	return sb.String(), nil
}
