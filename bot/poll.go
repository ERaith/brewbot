package bot

import (
	"fmt"
	"strings"

	"github.com/ERaith/brewbot/db"
	"github.com/bwmarrin/discordgo"
)

// pollEmojis are used as reaction options. Max 10 date options.
var pollEmojis = []string{"1️⃣", "2️⃣", "3️⃣", "4️⃣", "5️⃣", "6️⃣", "7️⃣", "8️⃣", "9️⃣", "🔟"}

func (b *Bot) handleStartPoll(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	existing, err := b.db.GetOpenPoll(i.GuildID)
	if err != nil {
		return err
	}
	if existing != nil {
		return fmt.Errorf("there's already an open poll — use `/closepoll` to close it first")
	}

	proposed, err := b.db.GetProposedDates(i.GuildID)
	if err != nil {
		return err
	}
	if len(proposed) == 0 {
		return fmt.Errorf("no dates proposed yet — use `/propose` first")
	}

	// Deduplicate while preserving order
	seen := map[string]bool{}
	var dates []string
	for _, pd := range proposed {
		if !seen[pd.Date] {
			seen[pd.Date] = true
			dates = append(dates, pd.Date)
		}
	}
	if len(dates) > len(pollEmojis) {
		dates = dates[:len(pollEmojis)]
	}

	// Build poll message
	var sb strings.Builder
	sb.WriteString("🍺 **Vote for the next brew date!**\n\n")
	for idx, date := range dates {
		sb.WriteString(fmt.Sprintf("%s  %s\n", pollEmojis[idx], date))
	}
	sb.WriteString("\nReact with your choice. First option to reach majority wins!")

	// Ack the interaction ephemerally, then send the real poll message to the channel
	respond(s, i, "📊 Starting poll...")

	msg, err := s.ChannelMessageSend(i.ChannelID, sb.String())
	if err != nil {
		return fmt.Errorf("sending poll message: %w", err)
	}

	// Save poll to DB
	pollID, err := b.db.CreatePoll(i.GuildID, i.ChannelID)
	if err != nil {
		return err
	}
	b.db.SetPollMessage(pollID, msg.ID)
	for idx, date := range dates {
		b.db.AddPollOption(pollID, pollEmojis[idx], date)
	}

	// Prime reactions so users can click without typing
	for idx := range dates {
		s.MessageReactionAdd(i.ChannelID, msg.ID, pollEmojis[idx])
	}

	return nil
}

func (b *Bot) handleClosePoll(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	poll, err := b.db.GetOpenPoll(i.GuildID)
	if err != nil {
		return err
	}
	if poll == nil {
		return fmt.Errorf("no open poll")
	}
	respond(s, i, "Closing poll...")
	return b.closePollAndCreateChannel(s, i.GuildID, poll, "")
}

// checkPollReaction is called on every reaction — checks if majority has been reached.
func (b *Bot) checkPollReaction(s *discordgo.Session, r *discordgo.MessageReactionAdd) error {
	poll, opts, err := b.db.GetPollByMessage(r.MessageID)
	if err != nil || poll == nil || poll.Status != "open" {
		return err
	}

	// Determine majority threshold from active rotation members
	rotation, err := b.db.GetRotation(r.GuildID)
	if err != nil {
		return err
	}
	active := 0
	for _, m := range rotation {
		if m.Active {
			active++
		}
	}
	if active == 0 {
		active = 2 // sensible default if rotation not set up yet
	}
	majority := (active / 2) + 1

	// Fetch the message to read current reaction counts
	msg, err := s.ChannelMessage(r.ChannelID, r.MessageID)
	if err != nil {
		return err
	}

	for _, rxn := range msg.Reactions {
		// Subtract 1 for the bot's own priming reaction
		count := rxn.Count - 1
		if count >= majority {
			for _, o := range opts {
				if o.Emoji == rxn.Emoji.Name {
					return b.closePollAndCreateChannel(s, r.GuildID, poll, o.Date)
				}
			}
		}
	}
	return nil
}

func (b *Bot) closePollAndCreateChannel(s *discordgo.Session, guildID string, poll *db.Poll, winningDate string) error {
	// If no winner specified, pick the highest-voted option
	if winningDate == "" {
		winningDate = b.pickWinner(s, poll)
	}
	if winningDate == "" {
		winningDate = "TBD"
	}

	b.db.ClosePoll(poll.ID, winningDate)
	b.db.ClearProposedDates(guildID)

	brewer, err := b.db.NextBrewer(guildID)
	if err != nil {
		return err
	}
	if brewer == nil {
		s.ChannelMessageSend(poll.ChannelID, fmt.Sprintf(
			"🏆 **Poll closed!** Winning date: **%s**\n\n⚠️ No brewers in rotation — use `/rotation add @user` to set up the rotation.",
			winningDate,
		))
		return nil
	}

	// Create brew record
	brewID, err := b.db.CreateBrew(guildID, brewer.UserID, brewer.Username, winningDate)
	if err != nil {
		return err
	}

	// Get parent category from the next-brew channel
	pollChannel, err := s.Channel(poll.ChannelID)
	if err != nil {
		return err
	}

	// Channel name: brew-<brewer> (will be renamed when recipe submitted)
	slug := strings.ToLower(strings.ReplaceAll(brewer.Username, " ", "-"))
	channelName := fmt.Sprintf("brew-%s", slug)

	newCh, err := s.GuildChannelCreateComplex(guildID, discordgo.GuildChannelCreateData{
		Name:     channelName,
		Type:     discordgo.ChannelTypeGuildText,
		ParentID: pollChannel.ParentID,
	})
	if err != nil {
		return fmt.Errorf("creating brew channel: %w", err)
	}

	b.db.SetBrewChannel(brewID, newCh.ID)

	// Rotate brewer to end of queue so next person is up next time
	b.db.SkipBrewer(guildID, brewer.UserID)

	// Pin a welcome message in the new channel
	pinMsg, _ := s.ChannelMessageSend(newCh.ID, fmt.Sprintf(
		"📌 **Brew Planning**\n"+
			"🍺 **Brewer:** <@%s>\n"+
			"📅 **Date:** %s\n\n"+
			"<@%s> — run `/recipe submit` to add your recipe. The channel will be renamed to your brew name once you do!",
		brewer.UserID, winningDate, brewer.UserID,
	))
	if pinMsg != nil {
		s.ChannelMessagePin(newCh.ID, pinMsg.ID)
	}

	// Ensure blackboard channel exists
	go b.updateBlackboard(s, guildID)

	// Announce in the poll channel
	s.ChannelMessageSend(poll.ChannelID, fmt.Sprintf(
		"🏆 **Poll closed!** Winning date: **%s**\n"+
			"🍺 **Brewer:** <@%s>\n"+
			"📣 Brew channel: <#%s>\n\n"+
			"<@%s> head over and submit your recipe!",
		winningDate, brewer.UserID, newCh.ID, brewer.UserID,
	))

	return nil
}

func (b *Bot) pickWinner(s *discordgo.Session, poll *db.Poll) string {
	if poll.MessageID == "" {
		return ""
	}
	msg, err := s.ChannelMessage(poll.ChannelID, poll.MessageID)
	if err != nil {
		return ""
	}
	opts, _ := b.db.GetPollOptions(poll.ID)
	best := -1
	winner := ""
	for _, rxn := range msg.Reactions {
		count := rxn.Count - 1
		if count > best {
			best = count
			for _, o := range opts {
				if o.Emoji == rxn.Emoji.Name {
					winner = o.Date
				}
			}
		}
	}
	return winner
}
