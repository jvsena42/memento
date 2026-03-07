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
	"modernc.org/sqlite"
)

const LAST_MENTION_ID = "last_mention_id"

type Handler struct {
	Client       *twitter.Client
	CapsuleStore *storage.CapsuleStore
	Config       *config.Config
}

func (h *Handler) ProcessMention(ctx context.Context, mention twitter.Tweet, users []twitter.User) error {

	if mention.AuthorID == h.Client.BotUserID {
		return nil
	}

	var targetTweet *twitter.TweetResponse
	var err error

	if mention.InReplyToUserID != nil {
		targetTweet, err = h.Client.GetTweet(ctx, mention.ConversationID)
	} else {
		targetTweet, err = h.Client.GetTweet(ctx, mention.ID)
	}

	if err != nil {
		return fmt.Errorf("failed to fetch target tweet: %w", err)
	}

	var tweetUsers []twitter.User
	if targetTweet.Includes != nil {
		tweetUsers = targetTweet.Includes.Users
	}
	tweetAuthor := findUser(tweetUsers, targetTweet.Tweet.AuthorID)

	if tweetAuthor == "" {
		slog.Warn("tweetAuthor not found", "mentionID", mention.ID, "authorID", targetTweet.Tweet.AuthorID)
		return nil
	}

	requesterHandler := findUser(users, mention.AuthorID)

	if requesterHandler == "" {
		slog.Warn("requesterHandler not found", "mentionID", mention.ID, "authorID", mention.AuthorID)
		return nil
	}

	saved, err := h.CapsuleStore.TweetAlreadySaved(targetTweet.Tweet.ID)
	if err != nil {
		return fmt.Errorf("failed to check tweet: %w", err)
	}
	if saved {
		if _, err := h.Client.PostTweet(ctx, "This one's already saved! ⏳", "", mention.ID); err != nil {
			slog.Warn("failed to reply 'already saved'", "error", err)
		}
		return nil
	}

	saved, err = h.CapsuleStore.UserSavedToday(mention.AuthorID)
	if err != nil {
		return fmt.Errorf("failed to check tweet: %w", err)
	}
	if saved {
		if _, err := h.Client.PostTweet(ctx, "Come back tomorrow! 🕰️", "", mention.ID); err != nil {
			slog.Warn("failed to reply 'come back tomorrow'", "error", err)
		}
		return nil
	}

	capsule := storage.Capsule{
		RequesterID:     mention.AuthorID,
		RequesterHandle: requesterHandler,
		TweetID:         targetTweet.Tweet.ID,
		TweetAuthor:     tweetAuthor,
		TweetText:       targetTweet.Tweet.Text,
		IsReply:         mention.InReplyToUserID != nil,
		RepublishAt:     time.Now().UTC().Add(h.Config.RepublishDelay),
	}

	err = h.CapsuleStore.Create(&capsule)

	if err != nil {
		var sqliteErr *sqlite.Error
		if errors.As(err, &sqliteErr) && sqliteErr.Code() == 2067 { // 2067 = SQLITE_CONSTRAINT_UNIQUE
			slog.Debug("duplicate capsule, skipping", "tweet_id", capsule.TweetID)
			return nil
		}

		return fmt.Errorf("failed to create capsule: %w", err)
	}

	date := capsule.RepublishAt.Format("02/Jan/2006")
	if _, err := h.Client.PostTweet(ctx, fmt.Sprintf("📸 Saved! I'll bring this back on %s, @%s!", date, requesterHandler),
		"", mention.ID); err != nil {
		slog.Warn("failed to reply with confirmation", "error", err)
	}

	return nil
}

func (h *Handler) StartPoller(ctx context.Context) {
	sinceID, err := h.CapsuleStore.GetValue(LAST_MENTION_ID)
	if err != nil {
		slog.Warn("failed to load last mention id", "error", err)
	} else {
		h.Client.SinceID = sinceID
	}

	ticker := time.NewTicker(h.Config.PollInterval)

	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			tweetsResponse, err := h.Client.GetMentions(ctx)

			if err != nil {
				slog.Error("error fetching mentions", "error", err)
				continue
			}

			if h.Client.SinceID != "" {
				if err := h.CapsuleStore.SetValue(LAST_MENTION_ID, h.Client.SinceID); err != nil {
					slog.Error("failed to save last mention id", "error", err)
				}
			}

			for _, err := range tweetsResponse.Errors {
				slog.Error("error for tweetsResponse", "error", err)
			}

			if len(tweetsResponse.Tweets) == 0 {
				continue
			}

			var users []twitter.User
			if tweetsResponse.Includes != nil {
				users = tweetsResponse.Includes.Users
			}

			for _, tweet := range tweetsResponse.Tweets {
				if err := h.ProcessMention(ctx, tweet, users); err != nil {
					slog.Error("error processing mention", "tweet_id", tweet.ID, "error", err)
				}
			}
		case <-ctx.Done():
			slog.Info("poller stopped")
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
