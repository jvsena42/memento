package storage

import (
	"database/sql"
	"fmt"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

const migrationSQL = `
CREATE TABLE IF NOT EXISTS capsules (
    id               INTEGER PRIMARY KEY AUTOINCREMENT,
    requester_id     TEXT      NOT NULL,
    requester_handle TEXT      NOT NULL,
    tweet_id         TEXT      NOT NULL UNIQUE,
    tweet_author     TEXT      NOT NULL,
    tweet_text       TEXT      NOT NULL,
    is_reply         BOOLEAN   NOT NULL DEFAULT 0,
    mention_id       TEXT      NOT NULL DEFAULT '',
    created_at       TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    republish_at     TIMESTAMP NOT NULL,
    status           TEXT      NOT NULL DEFAULT 'pending',
    published_at     TIMESTAMP,
    years_delay      INTEGER   NOT NULL DEFAULT 5
);
CREATE INDEX IF NOT EXISTS idx_capsules_republish ON capsules (status, republish_at);
CREATE INDEX IF NOT EXISTS idx_capsules_requester_date ON capsules (requester_id, created_at);
CREATE INDEX IF NOT EXISTS idx_capsules_created_at ON capsules (created_at);
CREATE TABLE IF NOT EXISTS key_value (
    key   TEXT PRIMARY KEY,
    value TEXT NOT NULL
);
`

func setupTestDB(t *testing.T) *CapsuleStore {
	t.Helper()
	conn, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("opening test db: %v", err)
	}
	t.Cleanup(func() { conn.Close() })

	if _, err := conn.Exec(migrationSQL); err != nil {
		t.Fatalf("running migrations: %v", err)
	}

	db := &DB{Conn: conn}
	return NewCapsuleStore(db)
}

func newTestCapsule(tweetID string) *Capsule {
	return &Capsule{
		RequesterID:     "user1",
		RequesterHandle: "alice",
		TweetID:         tweetID,
		TweetAuthor:     "bob",
		TweetText:       "hello world",
		IsReply:         false,
		RepublishAt:     time.Now().UTC().Add(365 * 24 * time.Hour),
		YearsDelay:      1,
	}
}

func TestCreate_Success(t *testing.T) {
	store := setupTestDB(t)
	c := newTestCapsule("tweet1")

	if err := store.Create(c); err != nil {
		t.Fatalf("Create() error: %v", err)
	}
	if c.ID == 0 {
		t.Error("expected non-zero ID after Create")
	}
}

func TestCreate_DuplicateTweetID(t *testing.T) {
	store := setupTestDB(t)
	c1 := newTestCapsule("tweet1")
	if err := store.Create(c1); err != nil {
		t.Fatalf("first Create() error: %v", err)
	}

	c2 := newTestCapsule("tweet1")
	if err := store.Create(c2); err == nil {
		t.Error("expected error on duplicate tweet_id, got nil")
	}
}

func TestTweetAlreadySaved_Exists(t *testing.T) {
	store := setupTestDB(t)
	c := newTestCapsule("tweet1")
	store.Create(c)

	saved, err := store.TweetAlreadySaved("tweet1")
	if err != nil {
		t.Fatalf("TweetAlreadySaved() error: %v", err)
	}
	if !saved {
		t.Error("expected true for saved tweet")
	}
}

func TestTweetAlreadySaved_NotExists(t *testing.T) {
	store := setupTestDB(t)

	saved, err := store.TweetAlreadySaved("nonexistent")
	if err != nil {
		t.Fatalf("TweetAlreadySaved() error: %v", err)
	}
	if saved {
		t.Error("expected false for unsaved tweet")
	}
}

func TestUserCapsulesToday(t *testing.T) {
	store := setupTestDB(t)

	store.Create(newTestCapsule("t1"))
	store.Create(&Capsule{
		RequesterID: "user2", RequesterHandle: "carol",
		TweetID: "t2", TweetAuthor: "dave", TweetText: "other",
		RepublishAt: time.Now().UTC().Add(365 * 24 * time.Hour), YearsDelay: 1,
	})

	count, err := store.UserCapsulesToday("user1")
	if err != nil {
		t.Fatalf("UserCapsulesToday() error: %v", err)
	}
	if count != 1 {
		t.Errorf("UserCapsulesToday() = %d, want 1", count)
	}
}

func TestCapsulesToday(t *testing.T) {
	store := setupTestDB(t)

	store.Create(newTestCapsule("t1"))
	store.Create(&Capsule{
		RequesterID: "user2", RequesterHandle: "carol",
		TweetID: "t2", TweetAuthor: "dave", TweetText: "other",
		RepublishAt: time.Now().UTC().Add(365 * 24 * time.Hour), YearsDelay: 1,
	})

	count, err := store.CapsulesToday()
	if err != nil {
		t.Fatalf("CapsulesToday() error: %v", err)
	}
	if count != 2 {
		t.Errorf("CapsulesToday() = %d, want 2", count)
	}
}

func TestGetDueCapsules_ReturnsPending(t *testing.T) {
	store := setupTestDB(t)

	// Due capsule (republish_at in the past)
	c := &Capsule{
		RequesterID: "user1", RequesterHandle: "alice",
		TweetID: "t1", TweetAuthor: "bob", TweetText: "past",
		RepublishAt: time.Now().UTC().Add(-1 * time.Hour), YearsDelay: 1,
	}
	store.Create(c)

	capsules, err := store.GetDueCapsules()
	if err != nil {
		t.Fatalf("GetDueCapsules() error: %v", err)
	}
	if len(capsules) != 1 {
		t.Fatalf("GetDueCapsules() returned %d, want 1", len(capsules))
	}
	if capsules[0].TweetID != "t1" {
		t.Errorf("got tweet_id %q, want t1", capsules[0].TweetID)
	}
}

func TestGetDueCapsules_SkipsFuture(t *testing.T) {
	store := setupTestDB(t)

	c := &Capsule{
		RequesterID: "user1", RequesterHandle: "alice",
		TweetID: "t1", TweetAuthor: "bob", TweetText: "future",
		RepublishAt: time.Now().UTC().Add(24 * time.Hour), YearsDelay: 1,
	}
	store.Create(c)

	capsules, err := store.GetDueCapsules()
	if err != nil {
		t.Fatalf("GetDueCapsules() error: %v", err)
	}
	if len(capsules) != 0 {
		t.Errorf("GetDueCapsules() returned %d, want 0", len(capsules))
	}
}

func TestGetDueCapsules_SkipsPublished(t *testing.T) {
	store := setupTestDB(t)

	c := &Capsule{
		RequesterID: "user1", RequesterHandle: "alice",
		TweetID: "t1", TweetAuthor: "bob", TweetText: "done",
		RepublishAt: time.Now().UTC().Add(-1 * time.Hour), YearsDelay: 1,
	}
	store.Create(c)
	store.UpdateStatus(c.ID, "published")

	capsules, err := store.GetDueCapsules()
	if err != nil {
		t.Fatalf("GetDueCapsules() error: %v", err)
	}
	if len(capsules) != 0 {
		t.Errorf("GetDueCapsules() returned %d, want 0", len(capsules))
	}
}

func TestGetDueCapsules_BatchLimit(t *testing.T) {
	store := setupTestDB(t)

	for i := 0; i < 55; i++ {
		c := &Capsule{
			RequesterID: "user1", RequesterHandle: "alice",
			TweetID: fmt.Sprintf("t%d", i), TweetAuthor: "bob", TweetText: "text",
			RepublishAt: time.Now().UTC().Add(-1 * time.Hour), YearsDelay: 1,
		}
		store.Create(c)
	}

	capsules, err := store.GetDueCapsules()
	if err != nil {
		t.Fatalf("GetDueCapsules() error: %v", err)
	}
	if len(capsules) != 50 {
		t.Errorf("GetDueCapsules() returned %d, want 50", len(capsules))
	}
}

func TestUpdateStatus_Published(t *testing.T) {
	store := setupTestDB(t)
	c := newTestCapsule("t1")
	store.Create(c)

	if err := store.UpdateStatus(c.ID, "published"); err != nil {
		t.Fatalf("UpdateStatus() error: %v", err)
	}

	got, _ := store.GetByID(c.ID)
	if got.Status != "published" {
		t.Errorf("status = %q, want published", got.Status)
	}
	if got.PublishedAt == nil {
		t.Error("expected published_at to be set")
	}
}

func TestUpdateStatus_Failed(t *testing.T) {
	store := setupTestDB(t)
	c := newTestCapsule("t1")
	store.Create(c)

	if err := store.UpdateStatus(c.ID, "failed"); err != nil {
		t.Fatalf("UpdateStatus() error: %v", err)
	}

	got, _ := store.GetByID(c.ID)
	if got.Status != "failed" {
		t.Errorf("status = %q, want failed", got.Status)
	}
	if got.PublishedAt != nil {
		t.Error("expected published_at to be nil for failed status")
	}
}

func TestGetByID_Found(t *testing.T) {
	store := setupTestDB(t)
	c := newTestCapsule("t1")
	store.Create(c)

	got, err := store.GetByID(c.ID)
	if err != nil {
		t.Fatalf("GetByID() error: %v", err)
	}
	if got == nil {
		t.Fatal("GetByID() returned nil")
	}
	if got.TweetID != "t1" {
		t.Errorf("TweetID = %q, want t1", got.TweetID)
	}
}

func TestGetByID_NotFound(t *testing.T) {
	store := setupTestDB(t)

	got, err := store.GetByID(999)
	if err != nil {
		t.Fatalf("GetByID() error: %v", err)
	}
	if got != nil {
		t.Error("expected nil for missing ID")
	}
}

func TestGetSetValue(t *testing.T) {
	store := setupTestDB(t)

	if err := store.SetValue("key1", "val1"); err != nil {
		t.Fatalf("SetValue() error: %v", err)
	}

	got, err := store.GetValue("key1")
	if err != nil {
		t.Fatalf("GetValue() error: %v", err)
	}
	if got != "val1" {
		t.Errorf("GetValue() = %q, want val1", got)
	}
}

func TestGetValue_Missing(t *testing.T) {
	store := setupTestDB(t)

	got, err := store.GetValue("nonexistent")
	if err != nil {
		t.Fatalf("GetValue() error: %v", err)
	}
	if got != "" {
		t.Errorf("GetValue() = %q, want empty", got)
	}
}

func TestSetValue_Upsert(t *testing.T) {
	store := setupTestDB(t)

	store.SetValue("key1", "val1")
	store.SetValue("key1", "val2")

	got, _ := store.GetValue("key1")
	if got != "val2" {
		t.Errorf("GetValue() = %q, want val2 after upsert", got)
	}
}
