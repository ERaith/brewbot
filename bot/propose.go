package bot

import (
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
)

func (b *Bot) handlePropose(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	raw := i.ApplicationCommandData().Options[0].StringValue()

	parts := strings.Split(raw, ",")
	var dates []string
	for _, p := range parts {
		if d := strings.TrimSpace(p); d != "" {
			dates = append(dates, d)
		}
	}
	if len(dates) == 0 {
		return fmt.Errorf("no dates found — use comma-separated dates, e.g. `March 15, March 22`")
	}

	username := displayName(i)
	if err := b.db.AddProposedDates(i.GuildID, username, dates); err != nil {
		return fmt.Errorf("saving dates: %w", err)
	}

	all, err := b.db.GetProposedDates(i.GuildID)
	if err != nil {
		return err
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("📅 **%s** proposed: %s\n\n", username, strings.Join(dates, ", ")))
	sb.WriteString("**All proposed dates so far:**\n")
	for _, pd := range all {
		sb.WriteString(fmt.Sprintf("• %s _(by %s)_\n", pd.Date, pd.ProposedBy))
	}
	sb.WriteString("\nRun `/startpoll` when everyone has proposed to kick off the vote.")

	respondPublic(s, i, sb.String())
	return nil
}

func displayName(i *discordgo.InteractionCreate) string {
	if i.Member == nil {
		return "unknown"
	}
	if i.Member.Nick != "" {
		return i.Member.Nick
	}
	return i.Member.User.Username
}
