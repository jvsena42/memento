# Memento 🕰️

A Twitter/X bot that acts as a time capsule. Mention `@mementobot_x` on any tweet and it will republish it **1–5 years later**, bringing back memories from the past.

## How It Works

1. A user mentions `@mementobot_x` on a tweet (either as a reply or directly on a root tweet), optionally specifying a number of years (1–5)
2. The bot saves a snapshot of the target tweet
3. It replies with a confirmation: *"📸 Saved! I'll bring this back on 05/Feb/2031, @user!"*
4. On the scheduled date, the bot republishes the tweet as a quote tweet, tagging the original requester. If quoting isn't allowed, it replies to the original mention instead with the saved text
5. If the original tweet was deleted, the bot replies to the original mention with the saved snapshot and a note that the tweet was lost

Each user can save up to **5 tweets per day** (configurable via `MAX_CAPSULES_PER_USER_PER_DAY`). Each tweet can only be saved **once** — first come, first served. If someone tries to save an already-captured tweet, the bot replies: *"This one's already saved! ⏳"*

## Example

**Saving a memory (default 5 years):**

> **@someone:** just shipped my first open source project 🚀
>
> **@you:** @MementoBot
>
> **@MementoBot:** 📸 Saved! I'll bring this back on 05/Feb/2031, @you!

**Saving a memory with a custom delay:**

> **@you:** @MementoBot 2
>
> **@MementoBot:** 📸 Saved! I'll bring this back on 05/Feb/2028, @you!

**On the scheduled date:**

> **@MementoBot:** 🕰️ 5 years ago today... @you
>
> *(quote tweet of the original post)*

**If quoting is restricted (reply to the original mention):**

> **@you:** @MementoBot *(5 years ago)*
>
> &nbsp;&nbsp;&nbsp;&nbsp;**@MementoBot:** 🕰️ @you saved this memory 5 years ago:
>
> &nbsp;&nbsp;&nbsp;&nbsp;"just shipped my first open source project 🚀"
>
> &nbsp;&nbsp;&nbsp;&nbsp;https://x.com/i/status/123456789

**If the original was deleted (reply to the original mention):**

> **@you:** @MementoBot *(5 years ago)*
>
> &nbsp;&nbsp;&nbsp;&nbsp;**@MementoBot:** 🕰️ @you saved this memory 5 years ago, but the original tweet has been deleted 🕊️
>
> &nbsp;&nbsp;&nbsp;&nbsp;It said: *"just shipped my first open source project 🚀"*
>
> &nbsp;&nbsp;&nbsp;&nbsp;Original link: https://x.com/i/status/123456789

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
│   │   ├── client.go          # OAuth 1.0a HTTP client with retry/backoff
│   │   ├── mentions.go        # Polling the mentions timeline (paginated)
│   │   ├── tweets.go          # Fetch, post, and quote tweets
│   │   ├── models.go          # Twitter API v2 response types
│   │   └── errors.go          # Sentinel errors (ErrNotFound, ErrForbidden, ErrQuoteNotAllowed)
│   ├── bot/
│   │   ├── handler.go         # Mention processing and capsule creation
│   │   └── scheduler.go       # Hourly job to republish due capsules
│   └── storage/
│       ├── db.go              # SQLite connection and migrations
│       └── capsules.go        # CRUD + key/value store
├── migrations/
│   ├── 001_create_capsules.sql
│   ├── 002_create_key_value.sql
│   ├── 003_add_years_delay.sql
│   ├── 004_add_created_at_index.sql
│   └── 005_add_mention_id.sql
├── .env.example
├── Dockerfile
├── go.mod
└── README.md
```

## Requirements

- Go 1.25+
- A Twitter/X Developer account with API v2 access (Basic tier is sufficient)
- SQLite

## Configuration

Copy `.env.example` to `.env` and fill in your credentials:

```env
TWITTER_API_KEY=your_api_key
TWITTER_API_SECRET=your_api_secret
TWITTER_ACCESS_TOKEN=your_access_token
TWITTER_ACCESS_SECRET=your_access_secret
BOT_USER_ID=your_bot_numeric_user_id
BOT_HANDLE=MementoBot
DATABASE_PATH=./memento.db
DEV_MODE=false
POLL_INTERVAL=30s
REPUBLISH_DELAY=5m              # Base delay per year in DEV_MODE (e.g. 5m × 2 years = 10m). Ignored in production.
MAX_CAPSULES_PER_DAY=100        # Global daily capsule creation limit
MAX_CAPSULES_PER_USER_PER_DAY=5 # Per-user daily capsule limit
MAX_MENTION_PAGES=3             # Max pages to fetch when polling mentions
```

### Dev Mode

Set `DEV_MODE=true` to use a short republish delay instead of real years. The `REPUBLISH_DELAY` value (default 5 minutes) is multiplied by the number of years requested — so a 2-year capsule is republished in 10 minutes, a 5-year capsule in 25 minutes. The scheduler also runs every minute instead of every hour. Useful for testing the full pipeline end to end.

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

| Column             | Type      | Description                                         |
|--------------------|-----------|-----------------------------------------------------|
| `id`               | INTEGER   | Primary key                                         |
| `requester_id`     | TEXT      | Twitter user ID of who tagged the bot               |
| `requester_handle` | TEXT      | @handle for tagging on republish                    |
| `tweet_id`         | TEXT      | Target tweet ID (unique)                            |
| `tweet_author`     | TEXT      | Author of the target tweet                          |
| `tweet_text`       | TEXT      | Snapshot of the tweet text (fallback)               |
| `is_reply`         | BOOLEAN   | Whether the mention was a reply or root tweet       |
| `mention_id`       | TEXT      | Tweet ID of the bot mention (used to reply in-thread on fallback) |
| `years_delay`      | INTEGER   | Number of years until republish (1–5, default 5)    |
| `created_at`       | TIMESTAMP | When the capsule was created                        |
| `republish_at`     | TIMESTAMP | When the tweet should be republished                |
| `status`           | TEXT      | `pending` / `published` / `deleted` / `failed`      |
| `published_at`     | TIMESTAMP | When the tweet was actually republished             |

### Key-Value Table

A small `key_value` table is used to persist the mention poller's `last_mention_id` high watermark across restarts, so no mentions are lost or reprocessed.

## Deployment

```bash
# Build the Docker image
docker build -t memento .

# Run
docker run --env-file .env -v $(pwd)/data:/data memento
```

The bot is designed to run as a long-lived process. It starts two loops:

- **Mention Poller** — checks for new mentions at the configured interval; persists the high watermark to the database
- **Scheduler** — runs once per hour (once per minute in dev mode), publishes any capsules that are due in batches of up to 50

## Rate Limits

- **Per user:** 5 capsules per day (configurable via `MAX_CAPSULES_PER_USER_PER_DAY`)
- **Global:** 100 capsules per day (configurable via `MAX_CAPSULES_PER_DAY`)
- **Per tweet:** 1 capsule ever (first come, first served)
- **Scheduler:** Max 3 capsules published per user per scheduler tick
- **Twitter API:** The bot respects Twitter's rate limits with exponential backoff on 429 responses and up to 3 retries on 5xx errors

## Edge Cases

| Scenario                          | Behavior                                                                     |
|-----------------------------------|------------------------------------------------------------------------------|
| Original tweet deleted            | Replies to original mention with snapshot text + original link               |
| Quote tweet forbidden (403)       | Replies to original mention with saved tweet text + original link            |
| User hit daily limit (5/day)      | Replies with a friendly "come back tomorrow" message                          |
| Bot tagged on a root tweet        | Treats that tweet itself as the capsule target                               |
| Tweet already saved by someone    | Replies: *"This one's already saved! ⏳"*                                    |
| Protected/suspended account       | Skipped gracefully, status set to `failed`                                   |
| Bot mentions itself               | Ignored silently                                                             |
| Malformed/empty mentions          | Skipped with a warning log                                                   |
| Large backlog of due capsules     | Processed in batches of 50 per scheduler tick (up to 20 batches per run)     |
| Scheduler republish message       | Includes the actual `years_delay` value, e.g. *"🕰️ 2 years ago today..."*   |

## Custom Delay

When mentioning the bot, you can include a number (1–5) in your message to set how many years before the tweet is brought back:

```
@mementobot_x 3
```

If no valid number is found, or the value is out of range, it defaults to **5 years**.

## Tech Stack

- **Go** — core application
- **SQLite** — storage (`modernc.org/sqlite`, pure Go, no CGO)
- **Twitter API v2** — mentions, tweet lookup, posting
- **OAuth 1.0a** — request signing via `github.com/dghubble/oauth1`

## License

MIT