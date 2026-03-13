package bot

import (
	"context"

	"github.com/jvsena42/memento/internal/storage"
	"github.com/jvsena42/memento/internal/twitter"
)

type mockTwitterClient struct {
	getTweetFn    func(ctx context.Context, id string) (*twitter.TweetResponse, error)
	getTweetsFn   func(ctx context.Context, ids []string) (map[string]twitter.Tweet, []twitter.User, error)
	getMentionsFn func(ctx context.Context) (*twitter.TweetsResponse, error)
	postTweetFn   func(ctx context.Context, text, quoteTweetID, replyToID string) (*twitter.TweetResponse, error)
	botUserID     string
	sinceID       string
}

func (m *mockTwitterClient) GetTweet(ctx context.Context, id string) (*twitter.TweetResponse, error) {
	if m.getTweetFn != nil {
		return m.getTweetFn(ctx, id)
	}
	return &twitter.TweetResponse{}, nil
}

func (m *mockTwitterClient) GetTweets(ctx context.Context, ids []string) (map[string]twitter.Tweet, []twitter.User, error) {
	if m.getTweetsFn != nil {
		return m.getTweetsFn(ctx, ids)
	}
	return map[string]twitter.Tweet{}, nil, nil
}

func (m *mockTwitterClient) GetMentions(ctx context.Context) (*twitter.TweetsResponse, error) {
	if m.getMentionsFn != nil {
		return m.getMentionsFn(ctx)
	}
	return &twitter.TweetsResponse{}, nil
}

func (m *mockTwitterClient) PostTweet(ctx context.Context, text, quoteTweetID, replyToID string) (*twitter.TweetResponse, error) {
	if m.postTweetFn != nil {
		return m.postTweetFn(ctx, text, quoteTweetID, replyToID)
	}
	return &twitter.TweetResponse{}, nil
}

func (m *mockTwitterClient) GetBotUserID() string { return m.botUserID }
func (m *mockTwitterClient) GetSinceID() string   { return m.sinceID }
func (m *mockTwitterClient) SetSinceID(id string) { m.sinceID = id }

type mockCapsuleStorage struct {
	createFn            func(c *storage.Capsule) error
	tweetAlreadySavedFn func(tweetID string) (bool, error)
	userCapsulesTodayFn func(requesterID string) (int, error)
	capsulesTodayFn     func() (int, error)
	getDueCapsulesFn    func() ([]storage.Capsule, error)
	updateStatusFn      func(id int64, status string) error
	getValueFn          func(key string) (string, error)
	setValueFn          func(key string, value string) error
}

func (m *mockCapsuleStorage) Create(c *storage.Capsule) error {
	if m.createFn != nil {
		return m.createFn(c)
	}
	return nil
}

func (m *mockCapsuleStorage) TweetAlreadySaved(tweetID string) (bool, error) {
	if m.tweetAlreadySavedFn != nil {
		return m.tweetAlreadySavedFn(tweetID)
	}
	return false, nil
}

func (m *mockCapsuleStorage) UserCapsulesToday(requesterID string) (int, error) {
	if m.userCapsulesTodayFn != nil {
		return m.userCapsulesTodayFn(requesterID)
	}
	return 0, nil
}

func (m *mockCapsuleStorage) CapsulesToday() (int, error) {
	if m.capsulesTodayFn != nil {
		return m.capsulesTodayFn()
	}
	return 0, nil
}

func (m *mockCapsuleStorage) GetDueCapsules() ([]storage.Capsule, error) {
	if m.getDueCapsulesFn != nil {
		return m.getDueCapsulesFn()
	}
	return nil, nil
}

func (m *mockCapsuleStorage) UpdateStatus(id int64, status string) error {
	if m.updateStatusFn != nil {
		return m.updateStatusFn(id, status)
	}
	return nil
}

func (m *mockCapsuleStorage) GetValue(key string) (string, error) {
	if m.getValueFn != nil {
		return m.getValueFn(key)
	}
	return "", nil
}

func (m *mockCapsuleStorage) SetValue(key string, value string) error {
	if m.setValueFn != nil {
		return m.setValueFn(key, value)
	}
	return nil
}
