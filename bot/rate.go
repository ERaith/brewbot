package bot

import (
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
)

func (b *Bot) handleRate(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	brew, err := b.db.GetBrewByChannel(i.ChannelID)
	if err != nil {
		return err
	}
	if brew == nil {
		return fmt.Errorf("this command only works in a brew channel")
	}

	opts := i.ApplicationCommandData().Options
	rating := int(opts[0].IntValue())
	notes := ""
	if len(opts) > 1 {
		notes = opts[1].StringValue()
	}

	username := displayName(i)
	if err := b.db.UpsertRating(brew.ID, i.Member.User.ID, username, rating, notes); err != nil {
		return err
	}

	stars := strings.Repeat("⭐", rating) + strings.Repeat("☆", 5-rating)
	msg := fmt.Sprintf("%s **%s** rated **%s** — %d/5", stars, username, brew.Name, rating)
	if notes != "" {
		msg += fmt.Sprintf("\n> %s", notes)
	}
	if brew.Status == "complete" {
		msg += "\n_Brew is complete — rating added to the blackboard._"
	}
	respondPublic(s, i, msg)

	// Update pinned stats card + blackboard
	b.postOrUpdateStatsCard(s, brew)
	go b.updateBlackboard(s, i.GuildID)
	return nil
}

func (b *Bot) handleComplete(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	brew, err := b.db.GetBrewByChannel(i.ChannelID)
	if err != nil {
		return err
	}
	if brew == nil {
		return fmt.Errorf("this command only works in a brew channel")
	}
	if brew.Status == "complete" {
		return fmt.Errorf("this brew is already marked complete")
	}

	if err := b.db.CompleteBrew(brew.ID); err != nil {
		return err
	}

	ratings, _ := b.db.GetRatings(brew.ID)
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("🏆 **%s** is complete — added to the blackboard!\n", brew.Name))

	if len(ratings) > 0 {
		sum := 0
		for _, r := range ratings {
			sum += r.Rating
		}
		avg := float64(sum) / float64(len(ratings))
		sb.WriteString(fmt.Sprintf("**Final rating:** %.1f/5 (%d votes)\n", avg, len(ratings)))
	} else {
		sb.WriteString("No ratings yet — everyone can still `/rate` this brew.\n")
	}
	sb.WriteString("\nThe blackboard has been updated.")

	respondPublic(s, i, sb.String())
	return nil
}
