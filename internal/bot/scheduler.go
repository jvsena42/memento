package bot

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"
	"unicode/utf8"

	"github.com/jvsena42/memento/internal/config"
	"github.com/jvsena42/memento/internal/storage"
	"github.com/jvsena42/memento/internal/twitter"
)

const (
	maxTweetLength           = 280
	urlShorterLength         = 23
	maxBatches               = 20
	maxPublishPerUserPerTick = 3
)

type Scheduler struct {
	Client       TwitterClient
	CapsuleStore CapsuleStorage
	Config       *config.Config
}

func (s *Scheduler) PublishDueCapsules(ctx context.Context) {
	publishedPerUser := make(map[string]int)

	for batch := range maxBatches {
		select {
		case <-ctx.Done():
			return
		default:
		}

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
			if publishedPerUser[capsule.RequesterID] >= maxPublishPerUserPerTick {
				slog.Debug("per-user publish limit reached, deferring", "user_id", capsule.RequesterID)
				continue
			}

			if err := sleepWithContext(ctx, 2*time.Second); err != nil {
				return
			}

			if _, exists := existingTweets[capsule.TweetID]; exists {
				// Tweet still exists — post quote tweet
				_, err := s.Client.PostTweet(ctx, fmt.Sprintf("🕰️ %d years ago today... @%s", capsule.YearsDelay, capsule.RequesterHandle), capsule.TweetID, "")
				if err != nil {
					if errors.Is(err, twitter.ErrQuoteNotAllowed) {
						slog.Warn("quoting not allowed, falling back to repost", "capsule_id", capsule.ID)
						if err := s.publishRepost(ctx, capsule); err != nil {
							slog.Error("error publishing repost fallback", "error", err)
						} else {
							publishedPerUser[capsule.RequesterID]++
						}
					} else {
						slog.Error("error publishing tweet", "error", err)
						if err := s.CapsuleStore.UpdateStatus(capsule.ID, "failed"); err != nil {
							slog.Error("failed to update capsule status", "capsule_id", capsule.ID, "error", err)
						}
					}
				} else {
					if err := s.CapsuleStore.UpdateStatus(capsule.ID, "published"); err != nil {
						slog.Error("failed to update capsule status", "capsule_id", capsule.ID, "error", err)
					}
					publishedPerUser[capsule.RequesterID]++
				}
			} else {
				// Tweet deleted or inaccessible — post snapshot
				if err := s.publishDeletedCapsule(ctx, capsule); err != nil {
					slog.Error("error publishing deleted capsule", "error", err)
				} else {
					publishedPerUser[capsule.RequesterID]++
				}
			}
		}

		if batch == maxBatches-1 {
			slog.Warn("batch limit reached, will resume on next scheduler tick")
		}
	}
}

func (s *Scheduler) publishRepost(ctx context.Context, capsule storage.Capsule) error {
	prefix := fmt.Sprintf("🕰️ @%s saved this memory %d years ago:\n\n\"\"\n\n", capsule.RequesterHandle, capsule.YearsDelay)
	prefixLength := utf8.RuneCountInString(prefix) + urlShorterLength

	availableChars := maxTweetLength - prefixLength
	truncatedText := truncate(capsule.TweetText, availableChars)

	text := fmt.Sprintf("🕰️ @%s saved this memory %d years ago:\n\n\"%s\"\n\nhttps://x.com/i/status/%s",
		capsule.RequesterHandle,
		capsule.YearsDelay,
		truncatedText,
		capsule.TweetID,
	)

	_, postErr := s.Client.PostTweet(ctx, text, "", capsule.MentionID)
	if postErr != nil {
		if err := s.CapsuleStore.UpdateStatus(capsule.ID, "failed"); err != nil {
			slog.Error("failed to update capsule status", "capsule_id", capsule.ID, "error", err)
		}
		return postErr
	}

	if err := s.CapsuleStore.UpdateStatus(capsule.ID, "published"); err != nil {
		slog.Error("failed to update capsule status", "capsule_id", capsule.ID, "error", err)
	}
	return nil
}

func (s *Scheduler) publishDeletedCapsule(ctx context.Context, capsule storage.Capsule) error {
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

	_, postErr := s.Client.PostTweet(ctx, text, "", capsule.MentionID)
	if postErr != nil {
		if err := s.CapsuleStore.UpdateStatus(capsule.ID, "failed"); err != nil {
			slog.Error("failed to update capsule status", "capsule_id", capsule.ID, "error", err)
		}
		return postErr
	}

	if err := s.CapsuleStore.UpdateStatus(capsule.ID, "published"); err != nil {
		slog.Error("failed to update capsule status", "capsule_id", capsule.ID, "error", err)
	}
	return nil
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
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
