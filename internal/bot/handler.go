package bot

import (
	"fmt"

	"github.com/jvsena42/memento/internal/config"
	"github.com/jvsena42/memento/internal/storage"
	"github.com/jvsena42/memento/internal/twitter"
)

type Handler struct {
	Client       *twitter.Client
	CapsuleStore *storage.CapsuleStore
	Config       *config.Config
}

func (h *Handler) ProcessMention(mention twitter.Tweet, users []twitter.User) error {

	var targetTweet *twitter.TweetResponse
	var err error

	if mention.InReplyToUserID != nil {
		targetTweet, err = h.Client.GetTweet(mention.ConversationID)
	} else {
		targetTweet, err = h.Client.GetTweet(mention.ID)
	}

	if err != nil {
		return fmt.Errorf("failed to fetch target tweet: %w", err)
	}

}
