package config

import (
	"fmt"
	"os"
	"time"

	"github.com/joho/godotenv"
)

const (
	defaultPollInterval  = 30 * time.Second
	defaultRepublishDev  = 5 * time.Minute
	defaultRepublishProd = 5 * 365 * 24 * time.Hour // ~5 years
)

type Config struct {
	TwitterAPIKey       string
	TwitterAPISecret    string
	TwitterAccessToken  string
	TwitterAccessSecret string
	BotHandle           string
	DatabasePath        string
	DevMode             bool
	PollInterval        time.Duration
	RepublishDelay      time.Duration
}

func Load() (*Config, error) {
	// Load .env file if it exists, ignore error if missing
	_ = godotenv.Load()

	cfg := &Config{
		TwitterAPIKey:       os.Getenv("TWITTER_API_KEY"),
		TwitterAPISecret:    os.Getenv("TWITTER_API_SECRET"),
		TwitterAccessToken:  os.Getenv("TWITTER_ACCESS_TOKEN"),
		TwitterAccessSecret: os.Getenv("TWITTER_ACCESS_SECRET"),
		BotHandle:           os.Getenv("BOT_HANDLE"),
		DatabasePath:        os.Getenv("DATABASE_PATH"),
		DevMode:             os.Getenv("DEV_MODE") == "true",
	}

	if cfg.BotHandle == "" {
		cfg.BotHandle = "MementoBot"
	}

	if cfg.DatabasePath == "" {
		cfg.DatabasePath = "./memento.db"
	}

	// Poll interval
	if v := os.Getenv("POLL_INTERVAL"); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil {
			return nil, fmt.Errorf("invalid POLL_INTERVAL %q: %w", v, err)
		}
		cfg.PollInterval = d
	} else {
		cfg.PollInterval = defaultPollInterval
	}

	// Republish delay

	if cfg.DevMode {
		if v := os.Getenv("REPUBLISH_DELAY"); v != "" {
			d, err := time.ParseDuration(v)
			if err != nil {
				return nil, fmt.Errorf("invalid REPUBLISH_DELAY %q: %w", v, err)
			}
			cfg.RepublishDelay = d
		} else {
			cfg.RepublishDelay = defaultRepublishDev
		}
	} else {
		cfg.RepublishDelay = defaultRepublishProd
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

func (c *Config) validate() error {
	required := map[string]string{
		"TWITTER_API_KEY":       c.TwitterAPIKey,
		"TWITTER_API_SECRET":    c.TwitterAPISecret,
		"TWITTER_ACCESS_TOKEN":  c.TwitterAccessToken,
		"TWITTER_ACCESS_SECRET": c.TwitterAccessSecret,
	}

	for name, value := range required {
		if value == "" {
			return fmt.Errorf("missing required environment variable: %s", name)
		}
	}

	return nil
}
