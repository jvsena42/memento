package bot

import (
	"context"
	"fmt"
	"log/slog"
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

	if mention.AuthorID == h.Client.BotUserID {
		return nil
	}

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

	var tweetUsers []twitter.User
	if targetTweet.Includes != nil {
		tweetUsers = targetTweet.Includes.Users
	}
	tweetAuthor := findUser(tweetUsers, targetTweet.Tweet.AuthorID)

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

	if err := h.CapsuleStore.Create(&capsule); err != nil {
		return fmt.Errorf("failed to create capsule: %w", err)
	}

	date := capsule.RepublishAt.Format("02/Jan/2006")
	h.Client.PostTweet(
		fmt.Sprintf("📸 Saved! I'll bring this back on %s, @%s!", date, requesterHandler),
		"", mention.ID,
	)

	return nil
}

func (h *Handler) StartPoller(ctx context.Context) {

	ticker := time.NewTicker(h.Config.PollInterval)

	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			tweetsResponse, err := h.Client.GetMentions()

			if err != nil {
				slog.Error("error fetching mentions", "error", err)
				continue
			}

			for _, err := range tweetsResponse.Errors {
				slog.Error("error for tweetsResponse", "error", err)
			}

			if tweetsResponse.Tweets == nil || len(tweetsResponse.Tweets) == 0 {
				continue
			}

			var users []twitter.User
			if tweetsResponse.Includes != nil {
				users = tweetsResponse.Includes.Users
			}

			for _, tweet := range tweetsResponse.Tweets {
				if err := h.ProcessMention(tweet, users); err != nil {
					slog.Error("error processing mention", "tweet_id", tweet.ID, "error", err)
				}
			}
		case <-ctx.Done():
			slog.Info("pooler stopped")
			return
		}
	}
}

func findUser(users []twitter.User, userID string) string {
	for _, user := range users {
		if user.ID == userID {
			return user.UserName
		}
	}
	return ""
}
