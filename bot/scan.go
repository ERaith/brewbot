package bot

import (
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
)

// recipeScan looks at the last few messages in the channel, finds an image or
// text recipe, sends it to Claude, and pins the extracted recipe card.
func (b *Bot) recipeScan(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	if b.cfg.AnthropicKey == "" {
		return fmt.Errorf("ANTHROPIC_API_KEY not set — cannot scan recipes")
	}

	brew, err := b.db.GetBrewByChannel(i.ChannelID)
	if err != nil {
		return err
	}
	if brew == nil {
		return fmt.Errorf("this command only works in a brew channel")
	}

	// Acknowledge — Claude can take a second
	respond(s, i, "🔍 Scanning for a recipe, one sec...")

	// Fetch recent messages to find an image or text recipe
	messages, err := s.ChannelMessages(i.ChannelID, 20, "", "", "")
	if err != nil {
		return fmt.Errorf("fetching messages: %w", err)
	}

	var extracted *extractedRecipe

	// First pass: look for an image attachment
	for _, msg := range messages {
		if msg.Author.ID == s.State.User.ID {
			continue
		}
		for _, att := range msg.Attachments {
			if !strings.HasPrefix(att.ContentType, "image/") {
				continue
			}
			extracted, err = b.parseRecipeFromImage(att.URL, att.ContentType)
			if err != nil {
				return fmt.Errorf("image scan failed: %w", err)
			}
			break
		}
		if extracted != nil {
			break
		}
	}

	// Second pass: look for a long text message that looks like a recipe
	if extracted == nil {
		for _, msg := range messages {
			if msg.Author.ID == s.State.User.ID {
				continue
			}
			if len(msg.Content) > 100 {
				extracted, err = b.parseRecipeFromText(msg.Content)
				if err != nil {
					return fmt.Errorf("text scan failed: %w", err)
				}
				break
			}
		}
	}

	if extracted == nil {
		s.ChannelMessageSend(i.ChannelID, "❌ Couldn't find a recipe image or text in recent messages. Post the recipe first, then run `/recipe scan`.")
		return nil
	}

	// Use brew name from DB if Claude didn't find one
	name := extracted.Name
	if name == "" {
		name = brew.Name
	}
	if name == "" {
		name = "Unnamed Brew"
	}

	// Calculate ABV if we have both gravities
	abv := extracted.ABV()
	if err := b.db.SetBrewName(brew.ID, name); err != nil {
		return err
	}
	if err := b.db.UpsertRecipe(brew.ID, extracted.Style, extracted.OG, extracted.FG, abv, extracted.Ingredients, extracted.Notes); err != nil {
		return err
	}

	// Rename channel to brew name
	slug := strings.ToLower(strings.ReplaceAll(name, " ", "-"))
	s.ChannelEdit(i.ChannelID, &discordgo.ChannelEdit{Name: fmt.Sprintf("brew-%s", slug)})

	// Post and pin the stats card
	updated, err := b.db.GetBrewByChannel(i.ChannelID)
	if err != nil {
		return err
	}
	b.postOrUpdateStatsCard(s, updated)

	// Confirm in channel
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("✅ **Recipe extracted for %s**\n", name))
	if extracted.Style != "" {
		sb.WriteString(fmt.Sprintf("Style: %s  ", extracted.Style))
	}
	if extracted.OG > 0 {
		sb.WriteString(fmt.Sprintf("OG: %.3f  ", extracted.OG))
	}
	if extracted.FG > 0 {
		sb.WriteString(fmt.Sprintf("FG: %.3f  ABV: %.1f%%", extracted.FG, abv))
	} else if extracted.OG > 0 {
		sb.WriteString("FG: TBD — run `/recipe fg` at kegging")
	}
	s.ChannelMessageSend(i.ChannelID, sb.String())

	return nil
}

// ABV calculates ABV from the extracted OG/FG.
func (r *extractedRecipe) ABV() float64 {
	if r.OG > r.FG && r.FG > 0 {
		return (r.OG - r.FG) * 131.25
	}
	return 0
}
