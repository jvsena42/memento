package storage

import (
	"database/sql"
	"errors"
	"fmt"
	"time"
)

type Capsule struct {
	ID              int64
	RequesterID     string
	RequesterHandle string
	TweetID         string
	TweetAuthor     string
	TweetText       string
	IsReply         bool
	CreatedAt       time.Time
	RepublishAt     time.Time
	Status          string
	PublishedAt     *time.Time
}

type CapsuleStore struct {
	db *DB
}

func NewCapsuleStore(db *DB) *CapsuleStore {
	return &CapsuleStore{db: db}
}

func (s *CapsuleStore) Create(c *Capsule) error {
	result, err := s.db.Conn.Exec(`
		INSERT INTO capsules (requester_id, requester_handle, tweet_id, tweet_author, tweet_text, is_reply, republish_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, c.RequesterID, c.RequesterHandle, c.TweetID, c.TweetAuthor, c.TweetText, c.IsReply, c.RepublishAt)
	if err != nil {
		return fmt.Errorf("inserting capsule: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("getting last insert id: %w", err)
	}
	c.ID = id

	return nil
}

func (s *CapsuleStore) TweetAlreadySaved(TweetID string) (bool, error) {
	var count int
	err := s.db.Conn.QueryRow("SELECT COUNT(*) FROM capsules WHERE tweet_id = ?", TweetID).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("checking tweet existence: %w", err)
	}

	return count > 0, nil
}

func (s *CapsuleStore) UserSavedToday(requesterID string) (bool, error) {
	today := time.Now().UTC().Truncate(24 * time.Hour)
	tomorrow := today.Add(24 * time.Hour)

	var count int
	err := s.db.Conn.QueryRow(`
		SELECT COUNT(*) FROM capsules
		WHERE requester_id = ? AND created_at >= ? AND created_at < ?
	`, requesterID, today, tomorrow).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("checking daily rate limit: %w", err)
	}
	return count > 0, nil
}

// GetDueCapsules returns all pending capsules that are due for republishing
func (s *CapsuleStore) GetDueCapsules() ([]Capsule, error) {
	rows, err := s.db.Conn.Query(`
		SELECT id, requester_id, requester_handle, tweet_id, tweet_author, tweet_text, is_reply, created_at, republish_at, status
		FROM capsules
		WHERE status = 'pending' AND republish_at <= ?
		ORDER BY republish_at ASC
	`, time.Now().UTC())
	if err != nil {
		return nil, fmt.Errorf("querying due capsules: %w", err)
	}
	defer rows.Close()

	var capsules []Capsule
	for rows.Next() {
		var c Capsule
		if err := rows.Scan(&c.ID, &c.RequesterID, &c.RequesterHandle, &c.TweetID, &c.TweetAuthor, &c.TweetText, &c.IsReply, &c.CreatedAt, &c.RepublishAt, &c.Status); err != nil {
			return nil, fmt.Errorf("scanning capsule: %w", err)
		}
		capsules = append(capsules, c)
	}

	return capsules, rows.Err()
}

// UpdateStatus updates the status of a capsule and optionally sets published_at
func (s *CapsuleStore) UpdateStatus(id int64, status string) error {
	var err error
	if status == "published" || status == "deleted" {
		now := time.Now().UTC()
		_, err = s.db.Conn.Exec(`
			UPDATE capsules SET status = ?, published_at = ? WHERE id = ?
		`, status, now, id)
	} else {
		_, err = s.db.Conn.Exec(`
			UPDATE capsules SET status = ? WHERE id = ?
		`, status, id)
	}
	if err != nil {
		return fmt.Errorf("updating capsule status: %w", err)
	}
	return nil
}

func (s *CapsuleStore) GetByID(id int64) (*Capsule, error) {
	var c Capsule
	err := s.db.Conn.QueryRow(`
		SELECT id, requester_id, requester_handle, tweet_id, tweet_author, tweet_text, is_reply, created_at, republish_at, status, published_at
		FROM capsules WHERE id = ?
	`, id).Scan(&c.ID, &c.RequesterID, &c.RequesterHandle, &c.TweetID, &c.TweetAuthor, &c.TweetText, &c.IsReply, &c.CreatedAt, &c.RepublishAt, &c.Status, &c.PublishedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("getting capsule by id: %w", err)
	}
	return &c, nil
}

func (s *CapsuleStore) GetValue(key string) (string, error) {
	var value string
	err := s.db.Conn.QueryRow("SELECT value FROM key_value WHERE key = ?", key).Scan(&value)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("getting key %s: %w", key, err)
	}
	return value, nil
}

func (s *CapsuleStore) SetValue(key string, value string) error {
	if _, err := s.db.Conn.Exec("INSERT OR REPLACE INTO key_value (key, value) VALUES (?, ?)", key, value); err != nil {
		return fmt.Errorf("error setting value %s: %w", value, err)
	}
	return nil
}
