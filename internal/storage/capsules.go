
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

fun NewCapsuleStore(db *DB) *CapsuleStore {
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

func (s *CapsuleStore) TweetAlreadySaved (TweetID string) (bool,err) {
	var count int
	err := s.db.Conn.QueryRow("SELECT COUNT(*) FROM capsules WHERE tweet_id = ?", tweetID).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("checking tweet existence: %w", err)
	}

	return count > 0, nil
}