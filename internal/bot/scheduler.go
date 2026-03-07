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

const MAX_TWEET_LENGTH = 280
const URL_SHORTEN_LENGTH = 23

type Scheduler struct {
	Client       *twitter.Client
	CapsuleStore *storage.CapsuleStore
	Config       *config.Config
}

func (s *Scheduler) PublishDueCapsules(ctx context.Context) {

	for {
		capsules, err := s.CapsuleStore.GetDueCapsules()

		if err != nil {
			slog.Error("error fetching capsules", "error", err)
			return
		}

		if len(capsules) == 0 {
			break
		}

		for _, capsule := range capsules {
			response, err := s.Client.GetTweet(ctx, capsule.TweetID)

			if errors.Is(err, twitter.ErrForbidden) {
				slog.Error("error publishing capsule", "error", err)
				if err := s.CapsuleStore.UpdateStatus(capsule.ID, "failed"); err != nil {
					slog.Error("failed to update capsule status", "capsule_id", capsule.ID, "error", err)
				}
				continue
			}

			if errors.Is(err, twitter.ErrNotFound) {

				prefix := fmt.Sprintf("🕰️ @%s saved this memory 5 years ago, but the original tweet has been deleted 🕊️\n\nIt said: \"\"\n\nOriginal link: ", capsule.RequesterHandle)
				prefixLength := utf8.RuneCountInString(prefix) + URL_SHORTEN_LENGTH

				availableChars := MAX_TWEET_LENGTH - prefixLength

				truncatedText := truncate(capsule.TweetText, availableChars)

				text := fmt.Sprintf("🕰️ @%s saved this memory 5 years ago, but the original tweet has been deleted 🕊️\n\nIt said: \"%s\"\n\nOriginal link: https://x.com/i/status/%s",
					capsule.RequesterHandle,
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

				continue
			}

			if err != nil {
				slog.Error("error fetching tweet", "error", err)
				continue
			}

			if response != nil { // Tweet exists
				_, err := s.Client.PostTweet(ctx, fmt.Sprintf("🕰️ 5 years ago today... @%s", capsule.RequesterHandle), capsule.TweetID, "")
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
			}

			time.Sleep(2 * time.Second)
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
