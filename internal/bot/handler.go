package bot

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/jvsena42/memento/internal/config"
	"github.com/jvsena42/memento/internal/storage"
	"github.com/jvsena42/memento/internal/twitter"
	"modernc.org/sqlite"
)

const (
	lastMentionId = "last_mention_id"
)

type Handler struct {
	Client       TwitterClient
	CapsuleStore CapsuleStorage
	Config       *config.Config
	Limiter      *MentionLimiter
}

func (h *Handler) ProcessMention(ctx context.Context, mention twitter.Tweet, users []twitter.User, includedTweets []twitter.Tweet) error {

	if mention.AuthorID == h.Client.GetBotUserID() {
		return nil
	}

	if h.Limiter != nil && !h.Limiter.Allow(mention.AuthorID) {
		slog.Debug("mention rate limited", "user_id", mention.AuthorID)
		return nil
	}

	referencedTweet := findRepliedToTweet(mention.ReferencedTweets)

	targetID := mention.ID
	if referencedTweet != nil {
		targetID = referencedTweet.ID
	}

	tweetAlreadySaved, err := h.CapsuleStore.TweetAlreadySaved(targetID)
	if err != nil {
		return fmt.Errorf("failed to check tweet: %w", err)
	}
	if tweetAlreadySaved {
		slog.Debug("tweet already saved, skipping reply", "tweet_id", targetID)
		return nil
	}

	userCount, err := h.CapsuleStore.UserCapsulesToday(mention.AuthorID)
	if err != nil {
		return fmt.Errorf("failed to check user daily limit: %w", err)
	}

	requesterHandler := findUser(users, mention.AuthorID)

	if userCount >= h.Config.MaxCapsulesPerUserDay {
		slog.Debug("user daily limit reached, skipping", "user_id", mention.AuthorID, "count", userCount)
		return nil
	}

	// Try to find the target tweet in the included tweets from GetMentions expansion
	var target twitter.Tweet
	var found bool
	if targetID == mention.ID {
		// The mention itself is the target
		target = mention
		found = true
	} else {
		target, found = findTweet(includedTweets, targetID)
	}

	if found {
		tweetAuthor := findUser(users, target.AuthorID)
		return h.saveCapsule(ctx, mention, target, tweetAuthor, requesterHandler)
	}

	// Fallback: fetch individually if not in includes (e.g., across pagination boundaries)
	targetTweet, err := h.Client.GetTweet(ctx, targetID)
	if err != nil {
		return fmt.Errorf("failed to fetch target tweet: %w", err)
	}

	var tweetUsers []twitter.User
	if targetTweet.Includes != nil {
		tweetUsers = targetTweet.Includes.Users
	}
	tweetAuthor := findUser(tweetUsers, targetTweet.Tweet.AuthorID)
	return h.saveCapsule(ctx, mention, targetTweet.Tweet, tweetAuthor, requesterHandler)
}

func (h *Handler) saveCapsule(ctx context.Context, mention twitter.Tweet, target twitter.Tweet, tweetAuthor string, requesterHandler string) error {

	globalCount, err := h.CapsuleStore.CapsulesToday()
	if err != nil {
		return fmt.Errorf("failed to check daily cap: %w", err)
	}
	if globalCount >= h.Config.MaxCapsulesPerDay {
		slog.Warn("global daily capsule limit reached", "count", globalCount)
		return nil
	}

	if tweetAuthor == "" {
		slog.Warn("tweetAuthor not found", "mentionID", mention.ID, "authorID", target.AuthorID)
		return nil
	}

	if requesterHandler == "" {
		slog.Warn("requesterHandler not found", "mentionID", mention.ID, "authorID", mention.AuthorID)
		return nil
	}

	trimmedText := strings.TrimSpace(target.Text)
	if trimmedText == "" {
		slog.Warn("tweet text is empty, skipping", "tweet_id", target.ID)
		return nil
	}

	years := parseYear(mention.Text, config.MinRepublishDelayYear, config.MaxRepublishDelayYear)
	capsule := storage.Capsule{
		RequesterID:     mention.AuthorID,
		RequesterHandle: requesterHandler,
		TweetID:         target.ID,
		TweetAuthor:     tweetAuthor,
		TweetText:       trimmedText,
		IsReply:         mention.InReplyToUserID != nil,
		RepublishAt:     time.Now().UTC().AddDate(years, 0, 0),
		YearsDelay:      int64(years),
	}

	err = h.CapsuleStore.Create(&capsule)

	if err != nil {
		var sqliteErr *sqlite.Error
		if errors.As(err, &sqliteErr) && sqliteErr.Code() == 2067 { // 2067 = SQLITE_CONSTRAINT_UNIQUE
			slog.Debug("duplicate capsule, skipping", "tweet_id", capsule.TweetID)
			return nil
		}

		return fmt.Errorf("failed to create capsule: %w", err)
	}

	date := capsule.RepublishAt.Format("02/Jan/2006")
	if _, err := h.Client.PostTweet(ctx, fmt.Sprintf("📸 Saved! I'll bring this back on %s, @%s!", date, requesterHandler),
		"", mention.ID); err != nil {
		slog.Warn("failed to reply with confirmation", "error", err)
	}

	return nil
}

func (h *Handler) StartPoller(ctx context.Context) {
	sinceID, err := h.CapsuleStore.GetValue(lastMentionId)
	if err != nil {
		slog.Warn("failed to load last mention id", "error", err)
	} else {
		h.Client.SetSinceID(sinceID)
	}

	ticker := time.NewTicker(h.Config.PollInterval)

	defer ticker.Stop()
	h.pollMentions(ctx)
	for {
		select {
		case <-ticker.C:
			h.pollMentions(ctx)
		case <-ctx.Done():
			slog.Info("poller stopped")
			return
		}
	}
}

func (h *Handler) pollMentions(ctx context.Context) {
	tweetsResponse, err := h.Client.GetMentions(ctx)

	if err != nil {
		slog.Error("error fetching mentions", "error", err)
		return
	}

	if h.Client.GetSinceID() != "" {
		if err := h.CapsuleStore.SetValue(lastMentionId, h.Client.GetSinceID()); err != nil {
			slog.Error("failed to save last mention id", "error", err)
		}
	}

	for _, err := range tweetsResponse.Errors {
		slog.Error("error for tweetsResponse", "error", err)
	}

	if len(tweetsResponse.Tweets) == 0 {
		return
	}

	var users []twitter.User
	var includedTweets []twitter.Tweet
	if tweetsResponse.Includes != nil {
		users = tweetsResponse.Includes.Users
		includedTweets = tweetsResponse.Includes.Tweets
	}

	for _, tweet := range tweetsResponse.Tweets {
		if err := h.ProcessMention(ctx, tweet, users, includedTweets); err != nil {
			slog.Error("error processing mention", "tweet_id", tweet.ID, "error", err)
		}
	}
}

func findUser(users []twitter.User, userID string) string {
	for _, user := range users {
		if user.ID == userID {
			return user.UserName
		}
	}
	return ""
}

func findTweet(tweets []twitter.Tweet, id string) (twitter.Tweet, bool) {
	for _, t := range tweets {
		if t.ID == id {
			return t, true
		}
	}
	return twitter.Tweet{}, false
}

func findRepliedToTweet(referencedTweets []twitter.ReferencedTweet) *twitter.ReferencedTweet {
	for _, tweet := range referencedTweets {
		if tweet.Type == "replied_to" {
			return &tweet
		}
	}
	return nil
}

// parseYear extracts the first number separated by spaces from the text and returns it as int.
func parseYear(text string, min int, max int) int {
	// strings.Fields splits the string around each instance of one or more consecutive white space characters.
	words := strings.Fields(text)

	var year int
	for _, word := range words {
		// Attempt to convert the word to an integer
		if val, err := strconv.Atoi(word); err == nil {
			year = val
			break
		}
	}

	if year < min || year > max {
		return max
	}

	return year
}
