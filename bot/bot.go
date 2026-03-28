package bot

import (
	"fmt"
	"log"

	"github.com/ERaith/brewbot/config"
	"github.com/ERaith/brewbot/db"
	"github.com/bwmarrin/discordgo"
)

type Bot struct {
	cfg     *config.Config
	db      *db.DB
	session *discordgo.Session
}

func New(cfg *config.Config, database *db.DB) (*Bot, error) {
	s, err := discordgo.New("Bot " + cfg.Token)
	if err != nil {
		return nil, fmt.Errorf("discordgo: %w", err)
	}
	return &Bot{cfg: cfg, db: database, session: s}, nil
}

func (b *Bot) Start() error {
	b.session.AddHandler(b.onInteraction)
	b.session.AddHandler(b.onReactionAdd)
	b.session.AddHandler(b.onReady)
	b.session.Identify.Intents = discordgo.IntentsGuilds |
		discordgo.IntentsGuildMessages |
		discordgo.IntentsGuildMessageReactions

	if err := b.session.Open(); err != nil {
		return fmt.Errorf("open: %w", err)
	}

	if err := b.registerCommands(); err != nil {
		return fmt.Errorf("register commands: %w", err)
	}

	log.Println("bot started, commands registered")
	return nil
}

func (b *Bot) Stop() {
	b.session.Close()
}
