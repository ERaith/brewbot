package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/ERaith/brewbot/bot"
	"github.com/ERaith/brewbot/config"
	"github.com/ERaith/brewbot/db"
	"github.com/joho/godotenv"
)

// Version is set at build time via -ldflags.
var Version = "dev"

func main() {
	log.Printf("brewbot %s starting", Version)
	if err := godotenv.Load(); err != nil {
		log.Println("no .env file, reading from environment")
	}

	cfg := config.Load()
	if cfg.Token == "" {
		log.Fatal("DISCORD_TOKEN not set")
	}

	database, err := db.New(cfg.DBPath)
	if err != nil {
		log.Fatalf("db: %v", err)
	}
	defer database.Close()

	b, err := bot.New(cfg, database)
	if err != nil {
		log.Fatalf("bot: %v", err)
	}

	if err := b.Start(); err != nil {
		log.Fatalf("start: %v", err)
	}
	log.Println("brewbot running — Ctrl+C to stop")

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM)
	<-sc

	log.Println("shutting down...")
	b.Stop()
}
