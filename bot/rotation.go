package bot

import (
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
)

func (b *Bot) handleRotation(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	sub := i.ApplicationCommandData().Options[0]
	switch sub.Name {
	case "list":
		return b.rotationList(s, i)
	case "add":
		return b.rotationAdd(s, i, sub)
	case "skip":
		return b.rotationSkip(s, i, sub)
	case "next":
		return b.rotationNext(s, i)
	}
	return nil
}

func (b *Bot) rotationList(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	members, err := b.db.GetRotation(i.GuildID)
	if err != nil {
		return err
	}
	if len(members) == 0 {
		respond(s, i, "No brewers in rotation yet. Use `/rotation add @user` to add members.")
		return nil
	}

	var sb strings.Builder
	sb.WriteString("**🍺 Brew Rotation**\n")
	for idx, m := range members {
		marker := "   "
		if idx == 0 && m.Active {
			marker = "👉"
		}
		status := ""
		if !m.Active {
			status = " _(inactive)_"
		}
		sb.WriteString(fmt.Sprintf("%s %d. <@%s>%s\n", marker, idx+1, m.UserID, status))
	}
	respondPublic(s, i, sb.String())
	return nil
}

func (b *Bot) rotationAdd(s *discordgo.Session, i *discordgo.InteractionCreate, sub *discordgo.ApplicationCommandInteractionDataOption) error {
	user := sub.Options[0].UserValue(s)
	if err := b.db.AddRotationMember(i.GuildID, user.ID, user.Username); err != nil {
		return err
	}
	respondPublic(s, i, fmt.Sprintf("✅ <@%s> added to the rotation.", user.ID))
	return nil
}

func (b *Bot) rotationSkip(s *discordgo.Session, i *discordgo.InteractionCreate, sub *discordgo.ApplicationCommandInteractionDataOption) error {
	user := sub.Options[0].UserValue(s)
	reason := ""
	if len(sub.Options) > 1 {
		reason = sub.Options[1].StringValue()
	}

	if err := b.db.SkipBrewer(i.GuildID, user.ID); err != nil {
		return err
	}

	msg := fmt.Sprintf("⏭️ <@%s> skipped this round — moved to end of rotation.", user.ID)
	if reason != "" {
		msg += fmt.Sprintf("\n> _%s_", reason)
	}
	respondPublic(s, i, msg)
	return nil
}

func (b *Bot) rotationNext(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	brewer, err := b.db.NextBrewer(i.GuildID)
	if err != nil {
		return err
	}
	if brewer == nil {
		respond(s, i, "No brewers in rotation. Use `/rotation add @user` to add members.")
		return nil
	}
	respondPublic(s, i, fmt.Sprintf("🍺 Next up to brew: <@%s>", brewer.UserID))
	return nil
}
