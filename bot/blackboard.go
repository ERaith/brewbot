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
		name := e.Brew.Name
		if name == "" {
			name = "Unnamed Brew"
		}

		// Rating pill
		rating := "no ratings"
		if e.AvgRating > 0 {
			full := int(e.AvgRating + 0.5)
			rating = fmt.Sprintf("%.1f/5 %s", e.AvgRating, strings.Repeat("⭐", full))
		}

		// Channel link if available
		channelRef := ""
		if e.Brew.ChannelID != "" {
			channelRef = fmt.Sprintf(" · <#%s>", e.Brew.ChannelID)
		}

		line := fmt.Sprintf("🍺 **%s**", name)
		if e.Brew.Date != "" {
			line += fmt.Sprintf(" · %s", e.Brew.Date)
		}
		line += fmt.Sprintf(" · <@%s>", e.Brew.BrewerID)
		if e.Recipe != nil && e.Recipe.Style != "" {
			line += fmt.Sprintf(" · %s", e.Recipe.Style)
		}
		if e.Recipe != nil && e.Recipe.ABV > 0 {
			line += fmt.Sprintf(" · %.1f%% ABV", e.Recipe.ABV)
		}
		line += fmt.Sprintf(" · %s", rating)
		line += channelRef
		sb.WriteString(line + "\n")
	}

	// Discord message limit is 2000 chars — truncate gracefully
	out := sb.String()
	if len(out) > 1900 {
		out = out[:1900] + "\n_(showing recent brews — list truncated)_"
	}

	respond(s, i, out)
	return nil
}
