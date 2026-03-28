package bot

import (
	"log"

	"github.com/bwmarrin/discordgo"
)

var minRating = 1.0

var commands = []*discordgo.ApplicationCommand{
	{
		Name:        "propose",
		Description: "Propose dates for the next brew day",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "dates",
				Description: "Comma-separated dates, e.g. \"March 15, March 22\"",
				Required:    true,
			},
		},
	},
	{
		Name:        "startpoll",
		Description: "Start a vote from all proposed dates",
	},
	{
		Name:        "closepoll",
		Description: "Close the current poll and pick the winner (use if no majority reached)",
	},
	{
		Name:        "rotation",
		Description: "Manage the brewer rotation",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Name:        "list",
				Description: "Show the current rotation order",
				Type:        discordgo.ApplicationCommandOptionSubCommand,
			},
			{
				Name:        "add",
				Description: "Add a brewer to the rotation",
				Type:        discordgo.ApplicationCommandOptionSubCommand,
				Options: []*discordgo.ApplicationCommandOption{
					{
						Type:        discordgo.ApplicationCommandOptionUser,
						Name:        "user",
						Description: "The user to add",
						Required:    true,
					},
				},
			},
			{
				Name:        "skip",
				Description: "Skip a brewer this round — moves them to end of queue",
				Type:        discordgo.ApplicationCommandOptionSubCommand,
				Options: []*discordgo.ApplicationCommandOption{
					{
						Type:        discordgo.ApplicationCommandOptionUser,
						Name:        "user",
						Description: "The user to skip",
						Required:    true,
					},
					{
						Type:        discordgo.ApplicationCommandOptionString,
						Name:        "reason",
						Description: "Reason for skipping",
						Required:    false,
					},
				},
			},
			{
				Name:        "next",
				Description: "Show who is brewing next",
				Type:        discordgo.ApplicationCommandOptionSubCommand,
			},
		},
	},
	{
		Name:        "recipe",
		Description: "Submit or view the recipe for this brew channel",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Name:        "submit",
				Description: "Submit your brew day recipe (OG, ingredients, notes — FG comes later)",
				Type:        discordgo.ApplicationCommandOptionSubCommand,
				Options: []*discordgo.ApplicationCommandOption{
					{
						Type:        discordgo.ApplicationCommandOptionString,
						Name:        "name",
						Description: "Brew name — will rename this channel",
						Required:    true,
					},
					{
						Type:        discordgo.ApplicationCommandOptionString,
						Name:        "style",
						Description: "Beer style, e.g. IPA, Stout, Hefeweizen",
						Required:    false,
					},
					{
						Type:        discordgo.ApplicationCommandOptionNumber,
						Name:        "og",
						Description: "Original gravity, e.g. 1.065",
						Required:    false,
					},
					{
						Type:        discordgo.ApplicationCommandOptionNumber,
						Name:        "fg",
						Description: "Final gravity, e.g. 1.012",
						Required:    false,
					},
					{
						Type:        discordgo.ApplicationCommandOptionString,
						Name:        "ingredients",
						Description: "Ingredient list (hops, malts, yeast, etc.)",
						Required:    false,
					},
					{
						Type:        discordgo.ApplicationCommandOptionString,
						Name:        "notes",
						Description: "Brewing notes or process details",
						Required:    false,
					},
				},
			},
			{
				Name:        "fg",
				Description: "Set final gravity at kegging — calculates and locks in ABV",
				Type:        discordgo.ApplicationCommandOptionSubCommand,
				Options: []*discordgo.ApplicationCommandOption{
					{
						Type:        discordgo.ApplicationCommandOptionNumber,
						Name:        "fg",
						Description: "Final gravity reading, e.g. 1.012",
						Required:    true,
					},
				},
			},
			{
				Name:        "view",
				Description: "View the recipe for this brew channel",
				Type:        discordgo.ApplicationCommandOptionSubCommand,
			},
		},
	},
	{
		Name:        "rate",
		Description: "Rate the current brew",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionInteger,
				Name:        "rating",
				Description: "Rating from 1 to 5",
				Required:    true,
				Autocomplete: true,
			},
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "notes",
				Description: "Tasting notes",
				Required:    false,
			},
		},
	},
	{
		Name:        "complete",
		Description: "Mark this brew as complete and add it to the blackboard",
	},
	{
		Name:        "abv",
		Description: "Calculate ABV from original and final gravity",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionNumber,
				Name:        "og",
				Description: "Original gravity, e.g. 1.065",
				Required:    true,
			},
			{
				Type:        discordgo.ApplicationCommandOptionNumber,
				Name:        "fg",
				Description: "Final gravity, e.g. 1.012",
				Required:    true,
			},
		},
	},
}

func (b *Bot) registerCommands() error {
	for _, cmd := range commands {
		if _, err := b.session.ApplicationCommandCreate(b.cfg.AppID, "", cmd); err != nil {
			return err
		}
		log.Printf("registered /%s", cmd.Name)
	}
	return nil
}
