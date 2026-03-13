package bot

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/jvsena42/memento/internal/config"
	"github.com/jvsena42/memento/internal/storage"
	"github.com/jvsena42/memento/internal/twitter"
)

func TestParseYear(t *testing.T) {
	tests := []struct {
		name string
		text string
		min  int
		max  int
		want int
	}{
		{"valid in range", "@mementobot_x 3", 1, 5, 3},
		{"min boundary", "1 year", 1, 5, 1},
		{"max boundary", "5 years", 1, 5, 5},
		{"below min returns max", "@mementobot_x 0", 1, 5, 5},
		{"above max returns max", "@mementobot_x 10", 1, 5, 5},
		{"negative returns max", "@mementobot_x -1", 1, 5, 5},
		{"no number returns max", "@mementobot_x save this", 1, 5, 5},
		{"multiple numbers takes first", "2 years not 4", 1, 5, 2},
		{"number embedded in word ignored", "abc3def 2", 1, 5, 2},
		{"empty text returns max", "", 1, 5, 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseYear(tt.text, tt.min, tt.max)
			if got != tt.want {
				t.Errorf("parseYear(%q, %d, %d) = %d, want %d", tt.text, tt.min, tt.max, got, tt.want)
			}
		})
	}
}

func TestFindUser(t *testing.T) {
	users := []twitter.User{
		{ID: "1", UserName: "alice"},
		{ID: "2", UserName: "bob"},
	}

	t.Run("found", func(t *testing.T) {
		got := findUser(users, "2")
		if got != "bob" {
			t.Errorf("findUser() = %q, want %q", got, "bob")
		}
	})

	t.Run("not found", func(t *testing.T) {
		got := findUser(users, "99")
		if got != "" {
			t.Errorf("findUser() = %q, want empty", got)
		}
	})

	t.Run("empty slice", func(t *testing.T) {
		got := findUser(nil, "1")
		if got != "" {
			t.Errorf("findUser(nil) = %q, want empty", got)
		}
	})
}

func TestFindTweet(t *testing.T) {
	tweets := []twitter.Tweet{
		{ID: "100", Text: "hello"},
		{ID: "200", Text: "world"},
	}

	t.Run("found", func(t *testing.T) {
		got, ok := findTweet(tweets, "200")
		if !ok || got.Text != "world" {
			t.Errorf("findTweet() = (%v, %v), want (world tweet, true)", got, ok)
		}
	})

	t.Run("not found", func(t *testing.T) {
		_, ok := findTweet(tweets, "999")
		if ok {
			t.Error("findTweet() should return false for missing ID")
		}
	})
}

func TestFindRepliedToTweet(t *testing.T) {
	t.Run("has replied_to", func(t *testing.T) {
		refs := []twitter.ReferencedTweet{
			{Type: "quoted", ID: "1"},
			{Type: "replied_to", ID: "2"},
		}
		got := findRepliedToTweet(refs)
		if got == nil || got.ID != "2" {
			t.Errorf("findRepliedToTweet() = %v, want replied_to with ID 2", got)
		}
	})

	t.Run("no replied_to type", func(t *testing.T) {
		refs := []twitter.ReferencedTweet{
			{Type: "quoted", ID: "1"},
		}
		got := findRepliedToTweet(refs)
		if got != nil {
			t.Errorf("findRepliedToTweet() = %v, want nil", got)
		}
	})

	t.Run("empty slice", func(t *testing.T) {
		got := findRepliedToTweet(nil)
		if got != nil {
			t.Errorf("findRepliedToTweet(nil) = %v, want nil", got)
		}
	})
}

// --- ProcessMention tests ---

func newTestHandler(tc *mockTwitterClient, cs *mockCapsuleStorage) *Handler {
	return &Handler{
		Client:       tc,
		CapsuleStore: cs,
		Config: &config.Config{
			MaxCapsulesPerUserDay: 5,
			MaxCapsulesPerDay:     100,
		},
		Limiter: NewMentionLimiter(100, 24*time.Hour),
	}
}

func TestProcessMention_SkipsSelfMention(t *testing.T) {
	tc := &mockTwitterClient{botUserID: "bot1"}
	cs := &mockCapsuleStorage{}
	h := newTestHandler(tc, cs)

	mention := twitter.Tweet{ID: "m1", AuthorID: "bot1", Text: "self"}
	err := h.ProcessMention(context.Background(), mention, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestProcessMention_SkipsRateLimited(t *testing.T) {
	tc := &mockTwitterClient{botUserID: "bot1"}
	cs := &mockCapsuleStorage{}
	h := newTestHandler(tc, cs)
	h.Limiter = NewMentionLimiter(0, 24*time.Hour) // block all

	mention := twitter.Tweet{ID: "m1", AuthorID: "user1", Text: "save"}
	err := h.ProcessMention(context.Background(), mention, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestProcessMention_SkipsAlreadySaved(t *testing.T) {
	tc := &mockTwitterClient{botUserID: "bot1"}
	cs := &mockCapsuleStorage{
		tweetAlreadySavedFn: func(tweetID string) (bool, error) {
			return true, nil
		},
	}
	h := newTestHandler(tc, cs)

	mention := twitter.Tweet{ID: "m1", AuthorID: "user1", Text: "save"}
	err := h.ProcessMention(context.Background(), mention, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestProcessMention_SkipsUserDailyLimit(t *testing.T) {
	tc := &mockTwitterClient{botUserID: "bot1"}
	cs := &mockCapsuleStorage{
		userCapsulesTodayFn: func(requesterID string) (int, error) {
			return 5, nil // at limit
		},
	}
	h := newTestHandler(tc, cs)

	users := []twitter.User{{ID: "user1", UserName: "alice"}}
	mention := twitter.Tweet{ID: "m1", AuthorID: "user1", Text: "save"}
	err := h.ProcessMention(context.Background(), mention, users, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestProcessMention_SavesDirectMention(t *testing.T) {
	var createdCapsule bool
	var postedReply bool

	tc := &mockTwitterClient{
		botUserID: "bot1",
		postTweetFn: func(ctx context.Context, text, quoteTweetID, replyToID string) (*twitter.TweetResponse, error) {
			postedReply = true
			if replyToID != "m1" {
				t.Errorf("reply to = %q, want m1", replyToID)
			}
			return &twitter.TweetResponse{}, nil
		},
	}
	cs := &mockCapsuleStorage{
		createFn: func(c *storage.Capsule) error {
			createdCapsule = true
			if c.TweetID != "m1" {
				t.Errorf("capsule TweetID = %q, want m1", c.TweetID)
			}
			return nil
		},
	}
	h := newTestHandler(tc, cs)

	users := []twitter.User{
		{ID: "user1", UserName: "alice"},
	}
	mention := twitter.Tweet{ID: "m1", AuthorID: "user1", Text: "@mementobot_x 3"}

	err := h.ProcessMention(context.Background(), mention, users, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !createdCapsule {
		t.Error("expected capsule to be created")
	}
	if !postedReply {
		t.Error("expected confirmation reply to be posted")
	}
}

func TestProcessMention_SavesRepliedToTweet(t *testing.T) {
	var savedTweetID string

	tc := &mockTwitterClient{botUserID: "bot1"}
	cs := &mockCapsuleStorage{
		createFn: func(c *storage.Capsule) error {
			savedTweetID = c.TweetID
			return nil
		},
	}
	h := newTestHandler(tc, cs)

	users := []twitter.User{
		{ID: "user1", UserName: "alice"},
		{ID: "author1", UserName: "bob"},
	}
	includedTweets := []twitter.Tweet{
		{ID: "target1", AuthorID: "author1", Text: "original tweet"},
	}
	mention := twitter.Tweet{
		ID:       "m1",
		AuthorID: "user1",
		Text:     "@mementobot_x 2",
		ReferencedTweets: []twitter.ReferencedTweet{
			{Type: "replied_to", ID: "target1"},
		},
	}

	err := h.ProcessMention(context.Background(), mention, users, includedTweets)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if savedTweetID != "target1" {
		t.Errorf("saved TweetID = %q, want target1", savedTweetID)
	}
}

func TestProcessMention_FallbackFetchesTweet(t *testing.T) {
	var fetchedID string

	tc := &mockTwitterClient{
		botUserID: "bot1",
		getTweetFn: func(ctx context.Context, id string) (*twitter.TweetResponse, error) {
			fetchedID = id
			return &twitter.TweetResponse{
				Tweet:    twitter.Tweet{ID: id, AuthorID: "author1", Text: "fetched tweet"},
				Includes: &twitter.Includes{Users: []twitter.User{{ID: "author1", UserName: "bob"}}},
			}, nil
		},
	}
	cs := &mockCapsuleStorage{}
	h := newTestHandler(tc, cs)

	users := []twitter.User{{ID: "user1", UserName: "alice"}}
	mention := twitter.Tweet{
		ID:       "m1",
		AuthorID: "user1",
		Text:     "@mementobot_x",
		ReferencedTweets: []twitter.ReferencedTweet{
			{Type: "replied_to", ID: "target99"},
		},
	}

	// target99 is NOT in includedTweets, so it should fallback to GetTweet
	err := h.ProcessMention(context.Background(), mention, users, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fetchedID != "target99" {
		t.Errorf("fetched ID = %q, want target99", fetchedID)
	}
}

func TestProcessMention_GlobalLimitReached(t *testing.T) {
	var created bool

	tc := &mockTwitterClient{botUserID: "bot1"}
	cs := &mockCapsuleStorage{
		capsulesTodayFn: func() (int, error) {
			return 100, nil // at limit
		},
		createFn: func(c *storage.Capsule) error {
			created = true
			return nil
		},
	}
	h := newTestHandler(tc, cs)

	users := []twitter.User{{ID: "user1", UserName: "alice"}}
	mention := twitter.Tweet{ID: "m1", AuthorID: "user1", Text: "@mementobot_x 3"}

	err := h.ProcessMention(context.Background(), mention, users, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if created {
		t.Error("should not create capsule when global limit reached")
	}
}

func TestProcessMention_EmptyTweetText(t *testing.T) {
	var created bool

	tc := &mockTwitterClient{botUserID: "bot1"}
	cs := &mockCapsuleStorage{
		createFn: func(c *storage.Capsule) error {
			created = true
			return nil
		},
	}
	h := newTestHandler(tc, cs)

	users := []twitter.User{{ID: "user1", UserName: "alice"}}
	mention := twitter.Tweet{ID: "m1", AuthorID: "user1", Text: "   "} // whitespace only

	err := h.ProcessMention(context.Background(), mention, users, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if created {
		t.Error("should not create capsule for empty tweet text")
	}
}

func TestProcessMention_PostsConfirmation(t *testing.T) {
	var postedText string

	tc := &mockTwitterClient{
		botUserID: "bot1",
		postTweetFn: func(ctx context.Context, text, quoteTweetID, replyToID string) (*twitter.TweetResponse, error) {
			postedText = text
			return &twitter.TweetResponse{}, nil
		},
	}
	cs := &mockCapsuleStorage{}
	h := newTestHandler(tc, cs)

	users := []twitter.User{{ID: "user1", UserName: "alice"}}
	mention := twitter.Tweet{ID: "m1", AuthorID: "user1", Text: "@mementobot_x 2"}

	h.ProcessMention(context.Background(), mention, users, nil)

	if !strings.Contains(postedText, "@alice") {
		t.Errorf("confirmation should mention @alice, got %q", postedText)
	}
	if !strings.Contains(postedText, "Saved!") {
		t.Errorf("confirmation should contain 'Saved!', got %q", postedText)
	}
}

// need to suppress the unused import warning
var _ = fmt.Sprintf
