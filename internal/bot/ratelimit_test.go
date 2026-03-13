package bot

import (
	"testing"
	"time"
)

func TestMentionLimiter_AllowWithinLimit(t *testing.T) {
	limiter := NewMentionLimiter(5, 24*time.Hour)

	for i := 0; i < 5; i++ {
		if !limiter.Allow("user1") {
			t.Fatalf("expected Allow to return true on attempt %d", i+1)
		}
	}
}

func TestMentionLimiter_BlockAfterLimit(t *testing.T) {
	limiter := NewMentionLimiter(5, 24*time.Hour)

	for i := 0; i < 5; i++ {
		limiter.Allow("user1")
	}

	if limiter.Allow("user1") {
		t.Fatal("expected Allow to return false after limit reached")
	}
}

func TestMentionLimiter_SeparateUsers(t *testing.T) {
	limiter := NewMentionLimiter(1, 24*time.Hour)

	if !limiter.Allow("user1") {
		t.Fatal("expected Allow to return true for user1")
	}
	if limiter.Allow("user1") {
		t.Fatal("expected Allow to return false for user1 after limit")
	}
	if !limiter.Allow("user2") {
		t.Fatal("expected Allow to return true for user2")
	}
}

func TestMentionLimiter_WindowExpiry(t *testing.T) {
	limiter := NewMentionLimiter(1, 50*time.Millisecond)

	if !limiter.Allow("user1") {
		t.Fatal("expected Allow to return true")
	}
	if limiter.Allow("user1") {
		t.Fatal("expected Allow to return false within window")
	}

	time.Sleep(60 * time.Millisecond)

	if !limiter.Allow("user1") {
		t.Fatal("expected Allow to return true after window expired")
	}
}

func TestMentionLimiter_Cleanup(t *testing.T) {
	limiter := NewMentionLimiter(1, 10*time.Millisecond)
	limiter.maxSize = 2

	limiter.Allow("user1")
	limiter.Allow("user2")

	time.Sleep(20 * time.Millisecond)

	// This should trigger cleanup since maxSize=2 and we're adding a 3rd
	limiter.Allow("user3")

	limiter.mu.Lock()
	size := len(limiter.seen)
	limiter.mu.Unlock()

	if size != 1 {
		t.Fatalf("expected 1 entry after cleanup, got %d", size)
	}
}
