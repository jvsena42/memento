package twitter

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync/atomic"
	"testing"
	"time"
)

func newTestClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	return &Client{
		Authenticated:   server.Client(),
		BaseUrl:         server.URL,
		BotUserID:       "bot123",
		MaxMentionPages: 3,
	}
}

func TestDoRequestWithRetry_Success(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte(`{"ok":true}`))
	})

	body, err := client.doRequestWithRetry(context.Background(), "GET", client.BaseUrl+"/test", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(body) != `{"ok":true}` {
		t.Errorf("body = %q, want {\"ok\":true}", body)
	}
}

func TestDoRequestWithRetry_ServerError_Retries(t *testing.T) {
	var attempts atomic.Int32
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		n := attempts.Add(1)
		if n == 1 {
			w.WriteHeader(500)
			return
		}
		w.WriteHeader(200)
		w.Write([]byte(`ok`))
	})

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	body, err := client.doRequestWithRetry(ctx, "GET", client.BaseUrl+"/test", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(body) != "ok" {
		t.Errorf("body = %q, want ok", body)
	}
	if got := attempts.Load(); got < 2 {
		t.Errorf("expected at least 2 attempts, got %d", got)
	}
}

func TestDoRequestWithRetry_RateLimit(t *testing.T) {
	var attempts atomic.Int32
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		n := attempts.Add(1)
		if n == 1 {
			resetTime := time.Now().Unix() // reset immediately
			w.Header().Set("x-rate-limit-reset", strconv.FormatInt(resetTime, 10))
			w.WriteHeader(429)
			return
		}
		w.WriteHeader(200)
		w.Write([]byte(`ok`))
	})

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	body, err := client.doRequestWithRetry(ctx, "GET", client.BaseUrl+"/test", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(body) != "ok" {
		t.Errorf("body = %q, want ok", body)
	}
}

func TestDoRequestWithRetry_NotFound(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
	})

	_, err := client.doRequestWithRetry(context.Background(), "GET", client.BaseUrl+"/test", nil)
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestDoRequestWithRetry_Forbidden(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(403)
		w.Write([]byte("forbidden"))
	})

	_, err := client.doRequestWithRetry(context.Background(), "GET", client.BaseUrl+"/test", nil)
	if err == nil {
		t.Fatal("expected error for 403")
	}
	if errors.Is(err, ErrNotFound) {
		t.Error("should not be ErrNotFound")
	}
}

func TestDoRequestWithRetry_ClientError_NoRetry(t *testing.T) {
	var attempts atomic.Int32
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		attempts.Add(1)
		w.WriteHeader(400)
		w.Write([]byte("bad request"))
	})

	_, err := client.doRequestWithRetry(context.Background(), "GET", client.BaseUrl+"/test", nil)
	if err == nil {
		t.Fatal("expected error for 400")
	}
	if got := attempts.Load(); got != 1 {
		t.Errorf("expected 1 attempt (no retry), got %d", got)
	}
}

func TestDoRequestWithRetry_ContextCanceled(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	})

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	_, err := client.doRequestWithRetry(ctx, "GET", client.BaseUrl+"/test", nil)
	if err == nil {
		t.Fatal("expected error for canceled context")
	}
}

func TestDoRequestWithRetry_MaxRetriesExceeded(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	})

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, err := client.doRequestWithRetry(ctx, "GET", client.BaseUrl+"/test", nil)
	if err == nil {
		t.Fatal("expected error after max retries")
	}
	expected := "max retries exceeded"
	if fmt.Sprintf("%v", err) != expected {
		t.Errorf("error = %q, want %q", err, expected)
	}
}
