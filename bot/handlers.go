package bot

import (
	"log"

	"github.com/bwmarrin/discordgo"
)

func (b *Bot) onInteraction(s *discordgo.Session, i *discordgo.InteractionCreate) {
	switch i.Type {
	case discordgo.InteractionApplicationCommandAutocomplete:
		b.onAutocomplete(s, i)
		return
	case discordgo.InteractionApplicationCommand:
		// handled below
	default:
		return
	}

	var err error
	switch i.ApplicationCommandData().Name {
	case "propose":
		err = b.handlePropose(s, i)
	case "startpoll":
		err = b.handleStartPoll(s, i)
	case "closepoll":
		err = b.handleClosePoll(s, i)
	case "rotation":
		err = b.handleRotation(s, i)
	case "recipe":
		err = b.handleRecipe(s, i)
	case "rate":
		err = b.handleRate(s, i)
	case "complete":
		err = b.handleComplete(s, i)
	case "abv":
		err = b.handleABV(s, i)
	}

	if err != nil {
		log.Printf("/%s error: %v", i.ApplicationCommandData().Name, err)
		respond(s, i, "❌ "+err.Error())
	}
}

func (b *Bot) onReactionAdd(s *discordgo.Session, r *discordgo.MessageReactionAdd) {
	if r.UserID == s.State.User.ID {
		return // ignore bot's own reactions
	}
	if err := b.checkPollReaction(s, r); err != nil {
		log.Printf("poll reaction check error: %v", err)
	}
}

// respond sends an ephemeral reply (only visible to the user who ran the command)
func respond(s *discordgo.Session, i *discordgo.InteractionCreate, msg string) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: msg,
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}

// respondPublic sends a public reply visible to everyone
func respondPublic(s *discordgo.Session, i *discordgo.InteractionCreate, msg string) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: msg,
		},
	})
}
