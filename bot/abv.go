package bot

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
)

func (b *Bot) handleABV(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	opts := i.ApplicationCommandData().Options
	og := opts[0].FloatValue()
	fg := opts[1].FloatValue()

	if og <= fg {
		return fmt.Errorf("OG (%.3f) must be greater than FG (%.3f)", og, fg)
	}
	if og < 1.0 || og > 1.200 {
		return fmt.Errorf("OG looks off — expected something like 1.040–1.120")
	}

	abv := (og - fg) * 131.25
	attenuation := ((og - fg) / (og - 1.0)) * 100

	respond(s, i, fmt.Sprintf(
		"🧪 **ABV Calculator**\n`OG %.3f` → `FG %.3f`\n**ABV: %.1f%%**\nApparent attenuation: %.1f%%",
		og, fg, abv, attenuation,
	))
	return nil
}
