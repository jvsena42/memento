package twitter

import (
	"context"
	"encoding/json"
	"net/http"
	"sync/atomic"
	"testing"
)

func TestGetMentions_Success(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		resp := TweetsResponse{
			Tweets: []Tweet{
				{ID: "m1", AuthorID: "u1", Text: "@bot save"},
			},
			Includes: &Includes{
				Users: []User{{ID: "u1", UserName: "alice"}},
			},
			Meta: &Meta{NewestID: "m1", ResultCount: 1},
		}
		json.NewEncoder(w).Encode(resp)
	})

	got, err := client.GetMentions(context.Background())
	if err != nil {
		t.Fatalf("GetMentions() error: %v", err)
	}
	if len(got.Tweets) != 1 {
		t.Fatalf("got %d tweets, want 1", len(got.Tweets))
	}
	if got.Tweets[0].ID != "m1" {
		t.Errorf("tweet ID = %q, want m1", got.Tweets[0].ID)
	}
}

func TestGetMentions_UpdatesSinceID(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		resp := TweetsResponse{
			Tweets: []Tweet{{ID: "m5", Text: "hi"}},
			Meta:   &Meta{NewestID: "m5", ResultCount: 1},
		}
		json.NewEncoder(w).Encode(resp)
	})

	if client.SinceID != "" {
		t.Fatalf("SinceID should start empty, got %q", client.SinceID)
	}

	client.GetMentions(context.Background())

	if client.SinceID != "m5" {
		t.Errorf("SinceID = %q, want m5", client.SinceID)
	}
}

func TestGetMentions_Pagination(t *testing.T) {
	var page atomic.Int32
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		p := page.Add(1)
		var resp TweetsResponse
		if p == 1 {
			resp = TweetsResponse{
				Tweets: []Tweet{{ID: "m1", Text: "first"}},
				Meta:   &Meta{NewestID: "m1", NextToken: "page2token", ResultCount: 1},
			}
		} else {
			resp = TweetsResponse{
				Tweets: []Tweet{{ID: "m2", Text: "second"}},
				Meta:   &Meta{ResultCount: 1},
			}
		}
		json.NewEncoder(w).Encode(resp)
	})

	got, err := client.GetMentions(context.Background())
	if err != nil {
		t.Fatalf("GetMentions() error: %v", err)
	}
	if len(got.Tweets) != 2 {
		t.Fatalf("got %d tweets, want 2 across pages", len(got.Tweets))
	}
	if client.SinceID != "m1" {
		t.Errorf("SinceID = %q, want m1 (from first page)", client.SinceID)
	}
}

func TestGetMentions_UsesSinceID(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		sinceID := r.URL.Query().Get("since_id")
		if sinceID != "prev100" {
			t.Errorf("since_id param = %q, want prev100", sinceID)
		}
		resp := TweetsResponse{
			Tweets: []Tweet{{ID: "m200", Text: "new"}},
			Meta:   &Meta{NewestID: "m200", ResultCount: 1},
		}
		json.NewEncoder(w).Encode(resp)
	})

	client.SinceID = "prev100"
	_, err := client.GetMentions(context.Background())
	if err != nil {
		t.Fatalf("GetMentions() error: %v", err)
	}
}
