package bot

import (
	"sync"
	"time"
)

type rateLimitEntry struct {
	count     int
	windowStart time.Time
}

type MentionLimiter struct {
	mu       sync.Mutex
	seen     map[string]rateLimitEntry
	maxCount int
	window   time.Duration
	maxSize  int
}

func NewMentionLimiter(maxCount int, window time.Duration) *MentionLimiter {
	return &MentionLimiter{
		seen:     make(map[string]rateLimitEntry),
		maxCount: maxCount,
		window:   window,
		maxSize:  10000,
	}
}

// Allow returns true if the user has not exceeded maxCount mentions within the window.
func (l *MentionLimiter) Allow(userID string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	entry, exists := l.seen[userID]

	if exists && now.Sub(entry.windowStart) < l.window {
		if entry.count >= l.maxCount {
			return false
		}
		entry.count++
		l.seen[userID] = entry
		return true
	}

	// New window
	if len(l.seen) >= l.maxSize {
		l.cleanup(now)
	}

	l.seen[userID] = rateLimitEntry{
		count:       1,
		windowStart: now,
	}
	return true
}

func (l *MentionLimiter) cleanup(now time.Time) {
	for id, entry := range l.seen {
		if now.Sub(entry.windowStart) >= l.window {
			delete(l.seen, id)
		}
	}
}
