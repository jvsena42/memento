package bot

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/jvsena42/memento/internal/config"
	"github.com/jvsena42/memento/internal/storage"
	"github.com/jvsena42/memento/internal/twitter"
)

type Scheduler struct {
	Client       *twitter.Client
	CapsuleStore *storage.CapsuleStore
	Config       *config.Config
}

func (s *Scheduler) PublishDueCapsules() {
	capsules, err := s.CapsuleStore.GetDueCapsules()

	if err != nil {
		slog.Error("error fetching capsules", "error", err)
		return
	}

	for _, capsule := range capsules {
		response, err := s.Client.GetTweet(capsule.TweetID)

		if errors.Is(err, twitter.ErrForbidden) {
			slog.Error("error publishing capsule", "error", err)
			if err := s.CapsuleStore.UpdateStatus(capsule.ID, "failed"); err != nil {
				slog.Error("failed to update capsule status", "capsule_id", capsule.ID, "error", err)
			}
			continue
		}

		if errors.Is(err, twitter.ErrNotFound) {

			text := fmt.Sprintf("🕰️ @%s saved this memory 5 years ago, but the original tweet has been deleted 🕊️\n\nIt said: \"%s\"\n\nOriginal link: https://x.com/i/status/%s",
				capsule.RequesterHandle,
				capsule.TweetText,
				capsule.TweetID,
			)

			_, postErr := s.Client.PostTweet(text, "", "")
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
			_, err := s.Client.PostTweet(fmt.Sprintf("🕰️ 5 years ago today... @%s", capsule.RequesterHandle), capsule.TweetID, "")
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
	}
}

func (s *Scheduler) StartScheduler(ctx context.Context) {
	interval := 1 * time.Hour
	if s.Config.DevMode {
		interval = 1 * time.Minute
	}
	ticker := time.NewTicker(interval)

	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.PublishDueCapsules()
		case <-ctx.Done():
			slog.Info("scheduler stopped")
			return
		}
	}
}
