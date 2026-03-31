package bot

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/jvsena42/memento/internal/config"
	"github.com/jvsena42/memento/internal/storage"
	"github.com/jvsena42/memento/internal/twitter"
)

func TestTruncate(t *testing.T) {
	tests := []struct {
		name string
		s    string
		max  int
		want string
	}{
		{"short string", "hello", 10, "hello"},
		{"exact length", "hello", 5, "hello"},
		{"needs truncation", "hello world", 8, "hello..."},
		{"max equals 3", "hello", 3, "..."},
		{"max equals 2", "hello", 2, ".."},
		{"max equals 1", "hello", 1, "."},
		{"max equals 0", "hello", 0, ""},
		{"empty string", "", 10, ""},
		{"unicode chars", "héllo wörld", 8, "héllo..."},
		{"unicode exact", "héllo", 5, "héllo"},
		{"unicode truncated", "日本語テスト", 5, "日本..."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncate(tt.s, tt.max)
			if got != tt.want {
				t.Errorf("truncate(%q, %d) = %q, want %q", tt.s, tt.max, got, tt.want)
			}
		})
	}
}

// --- PublishDueCapsules tests ---

func newTestScheduler(tc *mockTwitterClient, cs *mockCapsuleStorage) *Scheduler {
	return &Scheduler{
		Client:       tc,
		CapsuleStore: cs,
		Config:       &config.Config{DevMode: true},
	}
}

func TestPublishDueCapsules_NoDueCapsules(t *testing.T) {
	tc := &mockTwitterClient{}
	cs := &mockCapsuleStorage{
		getDueCapsulesFn: func() ([]storage.Capsule, error) {
			return nil, nil
		},
	}
	s := newTestScheduler(tc, cs)

	// Should return without error or posting anything
	s.PublishDueCapsules(context.Background())
}

func TestPublishDueCapsules_TweetExists_QuoteTweet(t *testing.T) {
	var postedText string
	var postedQuoteID string

	tc := &mockTwitterClient{
		getTweetsFn: func(ctx context.Context, ids []string) (map[string]twitter.Tweet, []twitter.User, error) {
			return map[string]twitter.Tweet{
				"tweet1": {ID: "tweet1", Text: "original"},
			}, nil, nil
		},
		postTweetFn: func(ctx context.Context, text, quoteTweetID, replyToID string) (*twitter.TweetResponse, error) {
			postedText = text
			postedQuoteID = quoteTweetID
			return &twitter.TweetResponse{}, nil
		},
	}

	callCount := 0
	cs := &mockCapsuleStorage{
		getDueCapsulesFn: func() ([]storage.Capsule, error) {
			callCount++
			if callCount == 1 {
				return []storage.Capsule{
					{ID: 1, RequesterID: "u1", RequesterHandle: "alice", TweetID: "tweet1", TweetText: "original", YearsDelay: 3},
				}, nil
			}
			return nil, nil
		},
		updateStatusFn: func(id int64, status string) error {
			if status != "published" {
				t.Errorf("expected published status, got %q", status)
			}
			return nil
		},
	}
	s := newTestScheduler(tc, cs)

	s.PublishDueCapsules(context.Background())

	if postedQuoteID != "tweet1" {
		t.Errorf("quote tweet ID = %q, want tweet1", postedQuoteID)
	}
	if !strings.Contains(postedText, "@alice") {
		t.Errorf("text should mention @alice, got %q", postedText)
	}
	matched := false
	for _, tmpl := range quoteTweetTemplates {
		expected := fmt.Sprintf(tmpl, int64(3), "alice")
		if postedText == expected {
			matched = true
			break
		}
	}
	if !matched {
		t.Errorf("text should match a quote tweet template, got %q", postedText)
	}
}

func TestPublishDueCapsules_TweetDeleted_PostsSnapshot(t *testing.T) {
	var postedText string

	tc := &mockTwitterClient{
		getTweetsFn: func(ctx context.Context, ids []string) (map[string]twitter.Tweet, []twitter.User, error) {
			return map[string]twitter.Tweet{}, nil, nil // empty = tweet deleted
		},
		postTweetFn: func(ctx context.Context, text, quoteTweetID, replyToID string) (*twitter.TweetResponse, error) {
			postedText = text
			return &twitter.TweetResponse{}, nil
		},
	}

	callCount := 0
	cs := &mockCapsuleStorage{
		getDueCapsulesFn: func() ([]storage.Capsule, error) {
			callCount++
			if callCount == 1 {
				return []storage.Capsule{
					{ID: 1, RequesterID: "u1", RequesterHandle: "alice", TweetID: "tweet1", TweetText: "deleted tweet text", YearsDelay: 2},
				}, nil
			}
			return nil, nil
		},
		updateStatusFn: func(id int64, status string) error { return nil },
	}
	s := newTestScheduler(tc, cs)

	s.PublishDueCapsules(context.Background())

	if !strings.Contains(postedText, "deleted tweet text") {
		t.Errorf("snapshot should contain original text, got %q", postedText)
	}
	if !strings.Contains(postedText, "@alice") {
		t.Errorf("snapshot should mention @alice, got %q", postedText)
	}
}

func TestPublishDueCapsules_UpdatesStatusFailed(t *testing.T) {
	var updatedStatus string

	tc := &mockTwitterClient{
		getTweetsFn: func(ctx context.Context, ids []string) (map[string]twitter.Tweet, []twitter.User, error) {
			return map[string]twitter.Tweet{
				"tweet1": {ID: "tweet1"},
			}, nil, nil
		},
		postTweetFn: func(ctx context.Context, text, quoteTweetID, replyToID string) (*twitter.TweetResponse, error) {
			return nil, errors.New("twitter api error")
		},
	}

	callCount := 0
	cs := &mockCapsuleStorage{
		getDueCapsulesFn: func() ([]storage.Capsule, error) {
			callCount++
			if callCount == 1 {
				return []storage.Capsule{
					{ID: 1, RequesterID: "u1", RequesterHandle: "alice", TweetID: "tweet1", TweetText: "text", YearsDelay: 1},
				}, nil
			}
			return nil, nil
		},
		updateStatusFn: func(id int64, status string) error {
			updatedStatus = status
			return nil
		},
	}
	s := newTestScheduler(tc, cs)

	s.PublishDueCapsules(context.Background())

	if updatedStatus != "failed" {
		t.Errorf("status = %q, want failed", updatedStatus)
	}
}

func TestPublishDueCapsules_PerUserLimit(t *testing.T) {
	postCount := 0

	tc := &mockTwitterClient{
		getTweetsFn: func(ctx context.Context, ids []string) (map[string]twitter.Tweet, []twitter.User, error) {
			m := make(map[string]twitter.Tweet)
			for _, id := range ids {
				m[id] = twitter.Tweet{ID: id}
			}
			return m, nil, nil
		},
		postTweetFn: func(ctx context.Context, text, quoteTweetID, replyToID string) (*twitter.TweetResponse, error) {
			postCount++
			return &twitter.TweetResponse{}, nil
		},
	}

	callCount := 0
	cs := &mockCapsuleStorage{
		getDueCapsulesFn: func() ([]storage.Capsule, error) {
			callCount++
			if callCount == 1 {
				// 5 capsules from same user — should only publish 3
				capsules := make([]storage.Capsule, 5)
				for i := range capsules {
					capsules[i] = storage.Capsule{
						ID: int64(i + 1), RequesterID: "u1", RequesterHandle: "alice",
						TweetID: strings.Repeat("t", i+1), TweetText: "text", YearsDelay: 1,
					}
				}
				return capsules, nil
			}
			return nil, nil
		},
		updateStatusFn: func(id int64, status string) error { return nil },
	}
	s := newTestScheduler(tc, cs)

	s.PublishDueCapsules(context.Background())

	if postCount != 3 {
		t.Errorf("posted %d tweets, want 3 (per-user limit)", postCount)
	}
}

func TestPublishDeletedCapsule_Success(t *testing.T) {
	var postedText string
	var updatedStatus string

	tc := &mockTwitterClient{
		postTweetFn: func(ctx context.Context, text, quoteTweetID, replyToID string) (*twitter.TweetResponse, error) {
			postedText = text
			return &twitter.TweetResponse{}, nil
		},
	}
	cs := &mockCapsuleStorage{
		updateStatusFn: func(id int64, status string) error {
			updatedStatus = status
			return nil
		},
	}
	s := newTestScheduler(tc, cs)

	capsule := storage.Capsule{
		ID: 1, RequesterHandle: "alice", TweetID: "tweet1",
		TweetText: "hello world", YearsDelay: 3,
	}
	s.publishDeletedCapsule(context.Background(), capsule)

	if updatedStatus != "published" {
		t.Errorf("status = %q, want published", updatedStatus)
	}
	if !strings.Contains(postedText, "hello world") {
		t.Errorf("text should contain original tweet text, got %q", postedText)
	}
	if !strings.Contains(postedText, "tweet1") {
		t.Errorf("text should contain tweet ID link, got %q", postedText)
	}
}

func TestPublishDeletedCapsule_PostError(t *testing.T) {
	var updatedStatus string

	tc := &mockTwitterClient{
		postTweetFn: func(ctx context.Context, text, quoteTweetID, replyToID string) (*twitter.TweetResponse, error) {
			return nil, errors.New("post failed")
		},
	}
	cs := &mockCapsuleStorage{
		updateStatusFn: func(id int64, status string) error {
			updatedStatus = status
			return nil
		},
	}
	s := newTestScheduler(tc, cs)

	capsule := storage.Capsule{
		ID: 1, RequesterHandle: "alice", TweetID: "tweet1",
		TweetText: "hello", YearsDelay: 1,
	}
	s.publishDeletedCapsule(context.Background(), capsule)

	if updatedStatus != "failed" {
		t.Errorf("status = %q, want failed", updatedStatus)
	}
}
