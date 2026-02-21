package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/jvsena42/memento/internal/bot"
	"github.com/jvsena42/memento/internal/config"
	"github.com/jvsena42/memento/internal/storage"
	"github.com/jvsena42/memento/internal/twitter"
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

	if err := db.Migrate("migrations"); err != nil {
		slog.Error("failed to run migrations", "error", err)
		os.Exit(1)
	}

	slog.Info("database ready")

	capsuleStore := storage.NewCapsuleStore(db)

	twitterClient := twitter.NewClient(cfg)

	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup

	botHandler := bot.Handler{
		Client:       twitterClient,
		CapsuleStore: capsuleStore,
		Config:       cfg,
	}

	// Launch goroutines with wg tracking:
	wg.Add(1)
	go func() {
		defer wg.Done()
		botHandler.StartPoller(ctx)
	}()

	botScheduler := bot.Scheduler{
		Client:       twitterClient,
		CapsuleStore: capsuleStore,
		Config:       cfg,
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		botScheduler.StartScheduler(ctx)
	}()

	slog.Info("memento bot started 🕰️")

	// Set up signal listening:
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Block until signal received (replaces select{}):
	<-sigChan
	slog.Info("shutting down...")

	// Trigger shutdown:
	cancel()   // signals both goroutines via ctx.Done()
	wg.Wait()  // waits until both goroutines return
	db.Close() // clean up database
	slog.Info("shutdown complete")
}
