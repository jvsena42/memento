# Memento 🕰️

A Twitter/X bot that acts as a time capsule. Mention `@MementoBot` on any tweet, and it will republish it **5 years later**, bringing back memories from the past.

## How It Works

1. A user mentions `@MementoBot` on a tweet (either as a reply or directly on a root tweet)
2. The bot saves a snapshot of the target tweet
3. It replies with a confirmation: *"📸 Saved! I'll bring this back on 2031-02-05, @user!"*
4. Five years later, the bot republishes the tweet as a quote tweet, tagging the original requester
5. If the original tweet was deleted, the bot posts the saved snapshot with a message noting it was lost

Each user can only save **one tweet per day** to prevent spam.

## Example

**Saving a memory:**

> **@someone:** just mass mass mass shipped my first open source project 🚀
>
> **@you:** @MementoBot save this one
>
> **@MementoBot:** 📸 Saved! I'll bring this back on 2031-02-05, @you!

**Five years later:**

> **@MementoBot:** 🕰️ 5 years ago today... @you
>
> *(quote tweet of the original post)*

**If the original was deleted:**

> **@MementoBot:** 🕰️ @you saved this memory 5 years ago, but the original tweet has been deleted 🕊️
>
> It said: *"just mass mass mass shipped my first open source project 🚀"*
>
> Original link: https://x.com/i/status/123456789

## Project Structure

```
memento/
├── cmd/
│   └── memento/
│       └── main.go            # Entry point, wires everything together
├── internal/
│   ├── config/
│   │   └── config.go          # Environment-based configuration
│   ├── twitter/
│   │   ├── client.go          # OAuth and HTTP client setup
│   │   ├── mentions.go        # Polling the mentions timeline
│   │   ├── tweets.go          # Fetch, post, and quote tweets
│   │   └── models.go          # Twitter API response types
│   ├── bot/
│   │   ├── handler.go         # Mention processing and capsule creation
│   │   └── scheduler.go       # Daily job to republish due capsules
│   └── storage/
│       ├── db.go              # SQLite connection and migrations
│       └── capsules.go        # CRUD operations for capsules
├── migrations/
│   └── 001_create_capsules.sql
├── .env.example
├── Dockerfile
├── go.mod
└── README.md
```

## Requirements

- Go 1.21+
- A Twitter/X Developer account with API v2 access (Basic tier is sufficient)
- SQLite

## Configuration

Copy `.env.example` to `.env` and fill in your credentials:

```env
TWITTER_API_KEY=your_api_key
TWITTER_API_SECRET=your_api_secret
TWITTER_ACCESS_TOKEN=your_access_token
TWITTER_ACCESS_SECRET=your_access_secret
BOT_HANDLE=MementoBot
DATABASE_PATH=./memento.db
DEV_MODE=false
POLL_INTERVAL=30s
REPUBLISH_DELAY=5m  # Only used when DEV_MODE=true, otherwise defaults to 5 years
```

### Dev Mode

Set `DEV_MODE=true` to use a short republish delay (default 5 minutes) instead of 5 years. Useful for testing the full pipeline end to end.

## Getting Started

```bash
# Clone the repository
git clone https://github.com/yourusername/memento.git
cd memento

# Install dependencies
go mod download

# Set up your environment
cp .env.example .env
# Edit .env with your Twitter API credentials

# Run the bot
go run ./cmd/memento

# Or build and run
go build -o memento ./cmd/memento
./memento
```

## Database

Memento uses SQLite to store capsules. The schema is applied automatically on startup via the migration files in `migrations/`.

### Capsules Table

| Column             | Type      | Description                                  |
|--------------------|-----------|----------------------------------------------|
| `id`               | INTEGER   | Primary key                                  |
| `requester_id`     | TEXT      | Twitter user ID of who tagged the bot        |
| `requester_handle` | TEXT      | @handle for tagging on republish             |
| `tweet_id`         | TEXT      | Target tweet ID (unique)                     |
| `tweet_author`     | TEXT      | Author of the target tweet                   |
| `tweet_text`       | TEXT      | Snapshot of the tweet text (fallback)        |
| `is_reply`         | BOOLEAN   | Whether the mention was a reply or root       |
| `created_at`       | TIMESTAMP | When the capsule was created                 |
| `republish_at`     | TIMESTAMP | When the tweet should be republished         |
| `status`           | TEXT      | `pending` / `published` / `deleted` / `failed` |
| `published_at`     | TIMESTAMP | When the tweet was actually republished      |

## Deployment

```bash
# Build the Docker image
docker build -t memento .

# Run
docker run --env-file .env -v $(pwd)/data:/data memento
```

The bot is designed to run as a long-lived process. It starts two loops:

- **Mention Poller** — checks for new mentions at the configured interval
- **Scheduler** — runs once per hour, publishes any capsules that are due

## Rate Limits

- **Per user:** 1 capsule per day
- **Twitter API:** The bot respects Twitter's rate limits with exponential back-off on 429 responses

## Edge Cases

| Scenario                          | Behavior                                                  |
|-----------------------------------|-----------------------------------------------------------|
| Original tweet deleted            | Posts snapshot text + original link + "lost memory" message |
| User already tagged today         | Replies with a friendly "come back tomorrow" message       |
| Bot tagged on a root tweet        | Treats that tweet itself as the capsule target             |
| Duplicate tag on the same tweet   | Ignored (tweet_id has a unique constraint)                 |
| Protected/suspended account       | Skipped gracefully, status set to `failed`                 |

## Tech Stack

- **Go** — core application
- **SQLite** — storage (`modernc.org/sqlite`)
- **Twitter API v2** — mentions, tweet lookup, posting

## License

MIT
