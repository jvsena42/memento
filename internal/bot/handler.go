package bot

import (
	"fmt"
	"time"

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

	tweetAuthor := findUser(targetTweet.Includes.Users, targetTweet.Tweet.AuthorID)

	requesterHandler := findUser(users, mention.AuthorID)

	saved, err := h.CapsuleStore.TweetAlreadySaved(targetTweet.Tweet.ID)
	if err != nil {
		return fmt.Errorf("failed to check tweet: %w", err)
	}
	if saved {
		h.Client.PostTweet("This one's already saved! ⏳", "", mention.ID)
		return nil
	}

	saved, err = h.CapsuleStore.UserSavedToday(mention.AuthorID)
	if err != nil {
		return fmt.Errorf("failed to check tweet: %w", err)
	}
	if saved {
		h.Client.PostTweet("Come back tomorrow! 🕰️", "", mention.ID)
		return nil
	}

	capsule := storage.Capsule{
		RequesterID:     mention.AuthorID,
		RequesterHandle: requesterHandler,
		TweetID:         targetTweet.Tweet.ID,
		TweetAuthor:     tweetAuthor,
		TweetText:       targetTweet.Tweet.Text,
		IsReply:         mention.InReplyToUserID != nil,
		RepublishAt:     time.Now().Add(h.Config.RepublishDelay),
	}

	h.CapsuleStore.Create(&capsule)
	return nil
}

func findUser(users []twitter.User, userID string) string {
	for _, user := range users {
		if user.ID == userID {
			return user.UserName
		}
	}
	return ""
}
