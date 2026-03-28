package bot

import (
	"github.com/bwmarrin/discordgo"
)

var ratingChoices = []*discordgo.ApplicationCommandOptionChoice{
	{Name: "5 ⭐⭐⭐⭐⭐ — Best brew ever", Value: 5},
	{Name: "4 ⭐⭐⭐⭐  — Great, would brew again", Value: 4},
	{Name: "3 ⭐⭐⭐   — Solid, drinkable", Value: 3},
	{Name: "2 ⭐⭐     — Meh, needs work", Value: 2},
	{Name: "1 ⭐       — Drain pour", Value: 1},
}

func (b *Bot) onAutocomplete(s *discordgo.Session, i *discordgo.InteractionCreate) {
	data := i.ApplicationCommandData()

	switch data.Name {
	case "rate":
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionApplicationCommandAutocompleteResult,
			Data: &discordgo.InteractionResponseData{
				Choices: ratingChoices,
			},
		})
	}
}
