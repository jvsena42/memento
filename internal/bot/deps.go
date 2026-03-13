package bot

import (
	"context"

	"github.com/jvsena42/memento/internal/storage"
	"github.com/jvsena42/memento/internal/twitter"
)

// TwitterClient defines the twitter operations used by Handler and Scheduler.
type TwitterClient interface {
	GetTweet(ctx context.Context, id string) (*twitter.TweetResponse, error)
	GetTweets(ctx context.Context, ids []string) (map[string]twitter.Tweet, []twitter.User, error)
	GetMentions(ctx context.Context) (*twitter.TweetsResponse, error)
	PostTweet(ctx context.Context, text string, quoteTweetID string, replyToID string) (*twitter.TweetResponse, error)
	GetBotUserID() string
	GetSinceID() string
	SetSinceID(id string)
}

// CapsuleStorage defines the storage operations used by Handler and Scheduler.
type CapsuleStorage interface {
	Create(c *storage.Capsule) error
	TweetAlreadySaved(tweetID string) (bool, error)
	UserCapsulesToday(requesterID string) (int, error)
	CapsulesToday() (int, error)
	GetDueCapsules() ([]storage.Capsule, error)
	UpdateStatus(id int64, status string) error
	GetValue(key string) (string, error)
	SetValue(key string, value string) error
}
