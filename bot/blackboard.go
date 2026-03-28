package bot

import (
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
)

func (b *Bot) handleBlackboard(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	entries, err := b.db.GetBlackboard(i.GuildID)
	if err != nil {
		return err
	}
	if len(entries) == 0 {
		respond(s, i, "📋 No completed brews yet. Mark a brew complete with `/complete`.")
		return nil
	}

	var sb strings.Builder
	sb.WriteString("📋 **The Blackboard**\n\n")

	for _, e := range entries {
		// Star rating display
		stars := ""
		if e.AvgRating > 0 {
			full := int(e.AvgRating + 0.5)
			stars = " " + strings.Repeat("⭐", full)
		}

		name := e.Brew.Name
		if name == "" {
			name = "Unnamed Brew"
		}

		sb.WriteString(fmt.Sprintf("**%s**%s\n", name, stars))
		sb.WriteString(fmt.Sprintf("  Brewer: <@%s>", e.Brew.BrewerID))
		if e.Brew.Date != "" {
			sb.WriteString(fmt.Sprintf(" | Date: %s", e.Brew.Date))
		}
		if e.Recipe != nil {
			if e.Recipe.Style != "" {
				sb.WriteString(fmt.Sprintf(" | Style: %s", e.Recipe.Style))
			}
			if e.Recipe.ABV > 0 {
				sb.WriteString(fmt.Sprintf(" | ABV: %.1f%%", e.Recipe.ABV))
			}
		}
		if e.AvgRating > 0 {
			sb.WriteString(fmt.Sprintf(" | Rating: %.1f/5 (%d votes)", e.AvgRating, len(e.Ratings)))
		}
		sb.WriteString("\n\n")
	}

	// Discord message limit is 2000 chars — truncate gracefully
	out := sb.String()
	if len(out) > 1900 {
		out = out[:1900] + "\n_(showing recent brews — list truncated)_"
	}

	respond(s, i, out)
	return nil
}
