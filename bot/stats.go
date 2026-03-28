package bot

import (
	"fmt"
	"strings"

	"github.com/ERaith/brewbot/db"
	"github.com/bwmarrin/discordgo"
)

// buildStatsCard builds the pinned stats message for a brew channel.
func buildStatsCard(brew *db.Brew, recipe *db.Recipe, ratings []db.Rating) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("📌 **%s**\n", brew.Name))
	sb.WriteString(fmt.Sprintf("**Brewer:** <@%s>", brew.BrewerID))
	if brew.Date != "" {
		sb.WriteString(fmt.Sprintf("  **Date:** %s", brew.Date))
	}
	sb.WriteString("\n")

	if recipe != nil {
		if recipe.Style != "" {
			sb.WriteString(fmt.Sprintf("**Style:** %s", recipe.Style))
		}
		if recipe.ABV > 0 {
			sb.WriteString(fmt.Sprintf("  **ABV:** %.1f%%", recipe.ABV))
		}
		if recipe.OG > 0 {
			sb.WriteString(fmt.Sprintf("  **OG:** %.3f  **FG:** %.3f", recipe.OG, recipe.FG))
		}
		if recipe.Style != "" || recipe.ABV > 0 {
			sb.WriteString("\n")
		}
	}

	sb.WriteString("─────────────────────\n")

	if len(ratings) == 0 {
		sb.WriteString("**Rating:** no ratings yet — use `/rate` after the session!\n")
	} else {
		sum := 0
		for _, r := range ratings {
			sum += r.Rating
		}
		avg := float64(sum) / float64(len(ratings))
		sb.WriteString(fmt.Sprintf("**Rating: %.1f / 5** (%d votes)\n", avg, len(ratings)))
		for _, r := range ratings {
			stars := strings.Repeat("⭐", r.Rating) + strings.Repeat("☆", 5-r.Rating)
			line := fmt.Sprintf("%s **%s** — %d/5", stars, r.Username, r.Rating)
			if r.Notes != "" {
				line += fmt.Sprintf(": _%s_", r.Notes)
			}
			sb.WriteString(line + "\n")
		}
	}

	return sb.String()
}

// postOrUpdateStatsCard posts the stats card if it doesn't exist, or edits it if it does.
func (b *Bot) postOrUpdateStatsCard(s *discordgo.Session, brew *db.Brew) error {
	recipe, err := b.db.GetRecipe(brew.ID)
	if err != nil {
		return err
	}
	ratings, err := b.db.GetRatings(brew.ID)
	if err != nil {
		return err
	}

	card := buildStatsCard(brew, recipe, ratings)

	if brew.StatsMessageID == "" {
		// First time — post and pin
		msg, err := s.ChannelMessageSend(brew.ChannelID, card)
		if err != nil {
			return err
		}
		s.ChannelMessagePin(brew.ChannelID, msg.ID)
		return b.db.SetBrewStatsMessage(brew.ID, msg.ID)
	}

	// Already exists — edit in place
	_, err = s.ChannelMessageEdit(brew.ChannelID, brew.StatsMessageID, card)
	return err
}
