package twitter

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"testing"
)

func TestGetTweet_Success(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		resp := TweetResponse{
			Tweet: Tweet{ID: "123", AuthorID: "a1", Text: "hello"},
			Includes: &Includes{
				Users: []User{{ID: "a1", UserName: "alice"}},
			},
		}
		json.NewEncoder(w).Encode(resp)
	})

	got, err := client.GetTweet(context.Background(), "123")
	if err != nil {
		t.Fatalf("GetTweet() error: %v", err)
	}
	if got.Tweet.ID != "123" {
		t.Errorf("ID = %q, want 123", got.Tweet.ID)
	}
	if got.Tweet.Text != "hello" {
		t.Errorf("Text = %q, want hello", got.Tweet.Text)
	}
	if len(got.Includes.Users) != 1 || got.Includes.Users[0].UserName != "alice" {
		t.Error("expected includes with user alice")
	}
}

func TestGetTweet_NotFound(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
	})

	_, err := client.GetTweet(context.Background(), "999")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestGetTweets_Success(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		resp := TweetsResponse{
			Tweets: []Tweet{
				{ID: "1", AuthorID: "a1", Text: "one"},
				{ID: "2", AuthorID: "a2", Text: "two"},
			},
			Includes: &Includes{
				Users: []User{
					{ID: "a1", UserName: "alice"},
					{ID: "a2", UserName: "bob"},
				},
			},
		}
		json.NewEncoder(w).Encode(resp)
	})

	tweets, users, err := client.GetTweets(context.Background(), []string{"1", "2"})
	if err != nil {
		t.Fatalf("GetTweets() error: %v", err)
	}
	if len(tweets) != 2 {
		t.Errorf("got %d tweets, want 2", len(tweets))
	}
	if tweets["1"].Text != "one" {
		t.Errorf("tweet 1 text = %q, want one", tweets["1"].Text)
	}
	if len(users) != 2 {
		t.Errorf("got %d users, want 2", len(users))
	}
}

func TestGetTweets_PartialResults(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		resp := TweetsResponse{
			Tweets: []Tweet{
				{ID: "1", Text: "one"},
			},
		}
		json.NewEncoder(w).Encode(resp)
	})

	tweets, _, err := client.GetTweets(context.Background(), []string{"1", "2"})
	if err != nil {
		t.Fatalf("GetTweets() error: %v", err)
	}
	if len(tweets) != 1 {
		t.Errorf("got %d tweets, want 1", len(tweets))
	}
	if _, ok := tweets["2"]; ok {
		t.Error("tweet 2 should be absent")
	}
}

func TestPostTweet_Success(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("method = %s, want POST", r.Method)
		}
		body, _ := io.ReadAll(r.Body)
		var req PostTweetRequest
		json.Unmarshal(body, &req)
		if req.Text != "hello world" {
			t.Errorf("text = %q, want hello world", req.Text)
		}
		resp := TweetResponse{Tweet: Tweet{ID: "new1", Text: "hello world"}}
		json.NewEncoder(w).Encode(resp)
	})

	got, err := client.PostTweet(context.Background(), "hello world", "", "")
	if err != nil {
		t.Fatalf("PostTweet() error: %v", err)
	}
	if got.Tweet.ID != "new1" {
		t.Errorf("ID = %q, want new1", got.Tweet.ID)
	}
}

func TestPostTweet_WithQuote(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var req PostTweetRequest
		json.Unmarshal(body, &req)
		if req.QuoteTweetID != "qt1" {
			t.Errorf("quote_tweet_id = %q, want qt1", req.QuoteTweetID)
		}
		if req.Reply != nil {
			t.Error("expected no reply config for quote tweet")
		}
		json.NewEncoder(w).Encode(TweetResponse{Tweet: Tweet{ID: "new1"}})
	})

	_, err := client.PostTweet(context.Background(), "quoting", "qt1", "")
	if err != nil {
		t.Fatalf("PostTweet() error: %v", err)
	}
}

func TestPostTweet_WithReply(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var req PostTweetRequest
		json.Unmarshal(body, &req)
		if req.Reply == nil || req.Reply.InReplyToTweetID != "reply1" {
			t.Errorf("expected reply to reply1, got %+v", req.Reply)
		}
		if req.QuoteTweetID != "" {
			t.Error("expected empty quote_tweet_id for reply")
		}
		json.NewEncoder(w).Encode(TweetResponse{Tweet: Tweet{ID: "new1"}})
	})

	_, err := client.PostTweet(context.Background(), "replying", "", "reply1")
	if err != nil {
		t.Fatalf("PostTweet() error: %v", err)
	}
}
