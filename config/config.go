package config

import "os"

type Config struct {
	Token  string
	AppID  string
	DBPath string
}

func Load() *Config {
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "brewbot.db"
	}
	return &Config{
		Token:  os.Getenv("DISCORD_TOKEN"),
		AppID:  os.Getenv("DISCORD_APP_ID"),
		DBPath: dbPath,
	}
}
