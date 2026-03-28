package bot

import (
	_ "embed"

	"github.com/bwmarrin/discordgo"
)

//go:embed help.md
var helpMessage string

// PostHelpChannel sends the command reference to the given channel.
func (b *Bot) PostHelpChannel(channelID string) error {
	_, err := b.session.ChannelMessageSend(channelID, helpMessage)
	return err
}

// SetupHelpChannel creates #brewbot-commands and posts the reference.
func (b *Bot) SetupHelpChannel(guildID, parentID string) (string, error) {
	ch, err := b.session.GuildChannelCreateComplex(guildID, discordgo.GuildChannelCreateData{
		Name:     "brewbot-commands",
		Type:     discordgo.ChannelTypeGuildText,
		Topic:    "Brewbot command reference — all slash commands listed here",
		ParentID: parentID,
	})
	if err != nil {
		return "", err
	}
	if err := b.PostHelpChannel(ch.ID); err != nil {
		return ch.ID, err
	}
	return ch.ID, nil
}
