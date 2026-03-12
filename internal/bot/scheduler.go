package bot

import (
	"context"
	"fmt"
	"log/slog"
	"time"
	"unicode/utf8"

	"github.com/jvsena42/memento/internal/config"
	"github.com/jvsena42/memento/internal/storage"
	"github.com/jvsena42/memento/internal/twitter"
)

const (
	maxTweetLength   = 280
	urlShorterLength = 23
	maxBatches       = 20
)

type Scheduler struct {
	Client       *twitter.Client
	CapsuleStore *storage.CapsuleStore
	Config       *config.Config
}

func (s *Scheduler) PublishDueCapsules(ctx context.Context) {

	for batch := range maxBatches {
		capsules, err := s.CapsuleStore.GetDueCapsules()

		if err != nil {
			slog.Error("error fetching capsules", "error", err)
			return
		}

		if len(capsules) == 0 {
			break
		}

		// Batch-fetch all tweet IDs in a single API call
		ids := make([]string, len(capsules))
		for i, c := range capsules {
			ids[i] = c.TweetID
		}

		existingTweets, _, err := s.Client.GetTweets(ctx, ids)
		if err != nil {
			slog.Error("error batch-fetching tweets", "error", err)
			return
		}

		for _, capsule := range capsules {
			sleepWithContext(ctx, 2*time.Second)

			if _, exists := existingTweets[capsule.TweetID]; exists {
				// Tweet still exists — post quote tweet
				_, err := s.Client.PostTweet(ctx, fmt.Sprintf("🕰️ %d years ago today... @%s", capsule.YearsDelay, capsule.RequesterHandle), capsule.TweetID, "")
				if err != nil {
					slog.Error("error publishing tweet", "error", err)
					if err := s.CapsuleStore.UpdateStatus(capsule.ID, "failed"); err != nil {
						slog.Error("failed to update capsule status", "capsule_id", capsule.ID, "error", err)
					}
				} else {
					if err := s.CapsuleStore.UpdateStatus(capsule.ID, "published"); err != nil {
						slog.Error("failed to update capsule status", "capsule_id", capsule.ID, "error", err)
					}
				}
			} else {
				// Tweet deleted or inaccessible — post snapshot
				s.publishDeletedCapsule(ctx, capsule)
			}
		}

		if batch == maxBatches-1 {
			slog.Warn("batch limit reached, will resume on next scheduler tick")
		}
	}
}

func (s *Scheduler) publishDeletedCapsule(ctx context.Context, capsule storage.Capsule) {
	prefix := fmt.Sprintf("🕰️ @%s saved this memory %d years ago, but the original tweet has been deleted 🕊️\n\nIt said: \"\"\n\nOriginal link: ", capsule.RequesterHandle, capsule.YearsDelay)
	prefixLength := utf8.RuneCountInString(prefix) + urlShorterLength

	availableChars := maxTweetLength - prefixLength

	truncatedText := truncate(capsule.TweetText, availableChars)

	text := fmt.Sprintf("🕰️ @%s saved this memory %d years ago, but the original tweet has been deleted 🕊️\n\nIt said: \"%s\"\n\nOriginal link: https://x.com/i/status/%s",
		capsule.RequesterHandle,
		capsule.YearsDelay,
		truncatedText,
		capsule.TweetID,
	)

	_, postErr := s.Client.PostTweet(ctx, text, "", "")
	if postErr != nil {
		slog.Error("error posting deleted capsule", "error", postErr)
		if err := s.CapsuleStore.UpdateStatus(capsule.ID, "failed"); err != nil {
			slog.Error("failed to update capsule status", "capsule_id", capsule.ID, "error", err)
		}
	} else {
		if err := s.CapsuleStore.UpdateStatus(capsule.ID, "published"); err != nil {
			slog.Error("failed to update capsule status", "capsule_id", capsule.ID, "error", err)
		}
	}
}

func (s *Scheduler) StartScheduler(ctx context.Context) {
	interval := 1 * time.Hour
	if s.Config.DevMode {
		interval = 1 * time.Minute
	}
	ticker := time.NewTicker(interval)

	defer ticker.Stop()

	s.PublishDueCapsules(ctx)

	for {
		select {
		case <-ticker.C:
			s.PublishDueCapsules(ctx)
		case <-ctx.Done():
			slog.Info("scheduler stopped")
			return
		}
	}
}

func truncate(s string, max int) string {
	runeCount := utf8.RuneCountInString(s)

	// 1. If it already fits, just return it.
	if runeCount <= max {
		return s
	}

	// 2. Edge case: if max is very small (less than the ellipsis itself)
	if max <= 3 {
		// Return just the dots up to the max, or an empty string
		dots := "..."
		return dots[:max]
	}

	// 3. Find the byte index for (max - 3) runes
	stopAt := max - 3
	count := 0
	for i := range s {
		if count == stopAt {
			return s[:i] + "..."
		}
		count++
	}

	return s
}

func sleepWithContext(ctx context.Context, d time.Duration) error {
	select {
	case <-time.After(d):
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
