package main

import (
	"log/slog"
	"os"

	"github.com/jvsena42/memento/internal/config"
	"github.com/jvsena42/memento/internal/storage"
)

func main() {

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	slog.Info("configuration loaded",
		"dev_mode", cfg.DevMode,
		"poll_interval", cfg.PollInterval,
		"republish_delay", cfg.RepublishDelay,
		"database", cfg.DatabasePath,
	)

	// Initialize DB
	db, err := storage.New(cfg.DatabasePath)
	if err != nil {
		slog.Error("failed to open database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	if err := db.Migrate("migrations"); err != nil {
		slog.Error("failed to run migrations", "error", err)
		os.Exit(1)
	}

	slog.Info("database ready")

	// Initialize capsule store
	_ = storage.NewCapsuleStore(db)

	// TODO Phase 2: Initialize Twitter client
	// TODO Phase 3: Start mention poller
	// TODO Phase 4: Start scheduler

	slog.Info("memento bot started 🕰️")

	// Block forever (will be replaced with signal handling in Phase 5)
	select {}
}
