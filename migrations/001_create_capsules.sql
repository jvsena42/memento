CREATE TABLE IF NOT EXISTS capsules (
    id               INTEGER PRIMARY KEY AUTOINCREMENT,
    requester_id     TEXT      NOT NULL,
    requester_handle TEXT      NOT NULL,
    tweet_id         TEXT      NOT NULL UNIQUE,
    tweet_author     TEXT      NOT NULL,
    tweet_text       TEXT      NOT NULL,
    is_reply         BOOLEAN   NOT NULL DEFAULT 0,
    created_at       TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    republish_at     TIMESTAMP NOT NULL,
    status           TEXT      NOT NULL DEFAULT 'pending',
    published_at     TIMESTAMP
);

-- Fast lookups for the scheduler
CREATE INDEX IF NOT EXISTS idx_capsules_republish
    ON capsules (status, republish_at);

-- Rate limit check: did this user already save today?
CREATE INDEX IF NOT EXISTS idx_capsules_requester_date
    ON capsules (requester_id, created_at);