package bot

import (
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
)

func (b *Bot) handleRecipe(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	sub := i.ApplicationCommandData().Options[0]
	switch sub.Name {
	case "submit":
		return b.recipeSubmit(s, i, sub)
	case "scan":
		return b.recipeScan(s, i)
	case "fg":
		return b.recipeFG(s, i, sub)
	case "view":
		return b.recipeView(s, i)
	}
	return nil
}

func (b *Bot) recipeSubmit(s *discordgo.Session, i *discordgo.InteractionCreate, sub *discordgo.ApplicationCommandInteractionDataOption) error {
	brew, err := b.db.GetBrewByChannel(i.ChannelID)
	if err != nil {
		return err
	}
	if brew == nil {
		return fmt.Errorf("this command only works in a brew channel")
	}

	optMap := map[string]*discordgo.ApplicationCommandInteractionDataOption{}
	for _, o := range sub.Options {
		optMap[o.Name] = o
	}

	name := optMap["name"].StringValue()
	style, ingredients, notes := "", "", ""
	var og, fg, abv float64

	if o, ok := optMap["style"]; ok {
		style = o.StringValue()
	}
	if o, ok := optMap["og"]; ok {
		og = o.FloatValue()
	}
	if o, ok := optMap["fg"]; ok {
		fg = o.FloatValue()
	}
	if og > 0 && fg > 0 && og > fg {
		abv = (og - fg) * 131.25
	}
	if o, ok := optMap["ingredients"]; ok {
		ingredients = o.StringValue()
	}
	if o, ok := optMap["notes"]; ok {
		notes = o.StringValue()
	}

	if err := b.db.SetBrewName(brew.ID, name); err != nil {
		return err
	}
	if err := b.db.UpsertRecipe(brew.ID, style, og, fg, abv, ingredients, notes); err != nil {
		return err
	}

	// Rename the channel to the brew name
	slug := strings.ToLower(strings.ReplaceAll(name, " ", "-"))
	newName := fmt.Sprintf("brew-%s", slug)
	s.ChannelEdit(i.ChannelID, &discordgo.ChannelEdit{Name: newName})

	// Build and post the recipe card
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("🍺 **%s** — Recipe\n", name))
	sb.WriteString(fmt.Sprintf("**Brewer:** <@%s>\n", brew.BrewerID))
	if brew.Date != "" {
		sb.WriteString(fmt.Sprintf("**Brew Date:** %s\n", brew.Date))
	}
	if style != "" {
		sb.WriteString(fmt.Sprintf("**Style:** %s\n", style))
	}
	if og > 0 {
		sb.WriteString(fmt.Sprintf("**OG:** %.3f", og))
		if fg > 0 {
			sb.WriteString(fmt.Sprintf("  **FG:** %.3f  **ABV:** %.1f%%", fg, abv))
		} else {
			sb.WriteString("  _FG TBD — run `/recipe fg` at kegging to lock in ABV_")
		}
		sb.WriteString("\n")
	}
	if ingredients != "" {
		sb.WriteString(fmt.Sprintf("\n**Ingredients:**\n%s\n", ingredients))
	}
	if notes != "" {
		sb.WriteString(fmt.Sprintf("\n**Notes:** %s\n", notes))
	}
	// Post/update the pinned stats card
	// Re-fetch brew to get latest state (name just set above)
	updated, err := b.db.GetBrewByChannel(i.ChannelID)
	if err == nil && updated != nil {
		b.postOrUpdateStatsCard(s, updated)
	}

	respondPublic(s, i, sb.String())
	return nil
}

func (b *Bot) recipeView(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	brew, err := b.db.GetBrewByChannel(i.ChannelID)
	if err != nil {
		return err
	}
	if brew == nil {
		return fmt.Errorf("this command only works in a brew channel")
	}

	recipe, err := b.db.GetRecipe(brew.ID)
	if err != nil {
		return err
	}
	if recipe == nil {
		respond(s, i, "No recipe submitted yet. Use `/recipe submit` to add one.")
		return nil
	}

	ratings, err := b.db.GetRatings(brew.ID)
	if err != nil {
		return err
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("🍺 **%s** — Recipe\n", brew.Name))
	sb.WriteString(fmt.Sprintf("**Brewer:** <@%s>  **Date:** %s\n", brew.BrewerID, brew.Date))
	if recipe.Style != "" {
		sb.WriteString(fmt.Sprintf("**Style:** %s\n", recipe.Style))
	}
	if recipe.OG > 0 {
		sb.WriteString(fmt.Sprintf("**OG:** %.3f  **FG:** %.3f  **ABV:** %.1f%%\n", recipe.OG, recipe.FG, recipe.ABV))
	}

	// Ratings summary
	if len(ratings) > 0 {
		sum := 0
		for _, r := range ratings {
			sum += r.Rating
		}
		avg := float64(sum) / float64(len(ratings))
		sb.WriteString(fmt.Sprintf("**Rating:** %.1f/5 (%d votes)\n", avg, len(ratings)))
		for _, r := range ratings {
			stars := strings.Repeat("⭐", r.Rating) + strings.Repeat("☆", 5-r.Rating)
			line := fmt.Sprintf("  %s %s — %d/5", stars, r.Username, r.Rating)
			if r.Notes != "" {
				line += fmt.Sprintf(": _%s_", r.Notes)
			}
			sb.WriteString(line + "\n")
		}
	} else {
		sb.WriteString("**Rating:** no ratings yet\n")
	}

	if recipe.Ingredients != "" {
		sb.WriteString(fmt.Sprintf("\n**Ingredients:**\n%s\n", recipe.Ingredients))
	}
	if recipe.Notes != "" {
		sb.WriteString(fmt.Sprintf("\n**Notes:** %s\n", recipe.Notes))
	}

	respondPublic(s, i, sb.String())
	return nil
}

func (b *Bot) recipeFG(s *discordgo.Session, i *discordgo.InteractionCreate, sub *discordgo.ApplicationCommandInteractionDataOption) error {
	brew, err := b.db.GetBrewByChannel(i.ChannelID)
	if err != nil {
		return err
	}
	if brew == nil {
		return fmt.Errorf("this command only works in a brew channel")
	}

	recipe, err := b.db.GetRecipe(brew.ID)
	if err != nil {
		return err
	}
	if recipe == nil {
		return fmt.Errorf("no recipe submitted yet — run `/recipe submit` first")
	}
	if recipe.OG == 0 {
		return fmt.Errorf("no OG on record — submit the recipe with OG first so ABV can be calculated")
	}

	fg := sub.Options[0].FloatValue()
	if fg >= recipe.OG {
		return fmt.Errorf("FG (%.3f) must be less than OG (%.3f)", fg, recipe.OG)
	}

	abv, err := b.db.SetFinalGravity(brew.ID, fg)
	if err != nil {
		return err
	}

	attenuation := ((recipe.OG - fg) / (recipe.OG - 1.0)) * 100

	msg := fmt.Sprintf(
		"🍺 **%s** — FG locked in!\n**OG:** %.3f → **FG:** %.3f\n**ABV: %.1f%%**  _(apparent attenuation: %.1f%%)_",
		brew.Name, recipe.OG, fg, abv, attenuation,
	)
	respondPublic(s, i, msg)

	// Update the pinned stats card with final ABV
	updated, err := b.db.GetBrewByChannel(i.ChannelID)
	if err == nil && updated != nil {
		b.postOrUpdateStatsCard(s, updated)
	}
	return nil
}
