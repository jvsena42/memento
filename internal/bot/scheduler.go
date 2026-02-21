package bot

import (
	"fmt"
	"log/slog"

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

		if err != nil {
			slog.Error("error fetching twwet", "error", err)
			s.CapsuleStore.UpdateStatus(capsule.ID, "deleted")
			continue
		}

		if response != nil { // Tweet exists
			_, err := s.Client.PostTweet(fmt.Sprintf("🕰️ 5 years ago today... @%s", capsule.RequesterHandle), capsule.TweetID, "")
			if err != nil {
				slog.Error("error fetching twwet", "error", err)
				s.CapsuleStore.UpdateStatus(capsule.ID, "failed")

			} else {
				s.CapsuleStore.UpdateStatus(capsule.ID, "published")
			}
		} else {
			text := fmt.Sprintf("🕰️ @%s saved this memory 5 years ago, but the original tweet has been deleted 🕊️\n\nIt said: \"%s\"\n\nOriginal link: https://x.com/i/status/%s",
				capsule.RequesterHandle,
				capsule.TweetText,
				capsule.TweetID,
			)
			_, err := s.Client.PostTweet(text, "", "")
			if err != nil {
				slog.Error("error fetching twwet", "error", err)
				s.CapsuleStore.UpdateStatus(capsule.ID, "failed")

			} else {
				s.CapsuleStore.UpdateStatus(capsule.ID, "published")
			}
		}

	}

}
