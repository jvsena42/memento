package twitter

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/dghubble/oauth1"

	"github.com/jvsena42/memento/internal/config"
)

type Client struct {
	Authenticated   *http.Client
	BotUserID       string
	BaseUrl         string
	SinceID         string
	MaxMentionPages int
}

func NewClient(ctx context.Context, cfg *config.Config) *Client {

	config := oauth1.Config{
		ConsumerKey:    cfg.TwitterAPIKey,
		ConsumerSecret: cfg.TwitterAPISecret,
	}
	token := oauth1.NewToken(cfg.TwitterAccessToken, cfg.TwitterAccessSecret)
	configClient := config.Client(ctx, token)
	configClient.Timeout = 30 * time.Second
	return &Client{
		Authenticated:   configClient,
		BaseUrl:         "https://api.twitter.com",
		BotUserID:       cfg.BotUserID,
		SinceID:         "",
		MaxMentionPages: cfg.MaxMentionPages,
	}
}

func (c *Client) doGet(ctx context.Context, endpoint string, params map[string]string) ([]byte, error) {
	url, err := url.Parse(c.BaseUrl + endpoint)
	if err != nil {
		return nil, err
	}

	query := url.Query()
	for key, value := range params {
		query.Set(key, value)
	}

	url.RawQuery = query.Encode()

	body, err := c.doRequestWithRetry(ctx, "GET", url.String(), nil)

	return body, err
}

func (c *Client) doPost(ctx context.Context, endpoint string, params interface{}) ([]byte, error) {
	url := c.BaseUrl + endpoint

	jsonBody, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("Failed to marshal params: %w", err)
	}

	body, err := c.doRequestWithRetry(ctx, "POST", url, jsonBody)

	return body, err
}

func (c *Client) doRequest(req *http.Request) ([]byte, int, http.Header, error) {
	// Make the request
	resp, err := c.Authenticated.Do(req)
	if err != nil {
		return nil, 0, nil, err // Network error, no status code
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, resp.Header, err
	}

	return body, resp.StatusCode, resp.Header, nil
}

func (c *Client) doRequestWithRetry(ctx context.Context, method string, url string, body []byte) ([]byte, error) {
	maxRetries := 3

	for attempt := 0; attempt <= maxRetries; attempt++ {

		var bodyReader io.Reader = http.NoBody
		if body != nil {
			bodyReader = bytes.NewReader(body)
		}

		req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)

		if err != nil {
			slog.Warn("request creation failed", "error", err)
			if err := sleepWithContext(ctx, time.Duration(math.Pow(2, float64(attempt)))*time.Second); err != nil {
				return nil, err
			}
			continue
		}

		if method == "POST" {
			req.Header.Set("Content-Type", "application/json")
		}
		respBody, statusCode, header, err := c.doRequest(req)

		//Network error
		if err != nil {
			if err := sleepWithContext(ctx, time.Duration(math.Pow(2, float64(attempt)))*time.Second); err != nil {
				return nil, err
			}
			continue
		}

		// Rate Limit -> wait until reset time
		if statusCode == 429 {
			resetStr := header.Get("x-rate-limit-reset")
			resetUnix, _ := strconv.ParseInt(resetStr, 10, 64)
			waitTime := time.Until(time.Unix(resetUnix, 0)) + 1*time.Second
			if waitTime < 1*time.Second {
				waitTime = 1 * time.Second
			}
			slog.Warn("rate limited, waiting", "seconds", waitTime.Seconds())
			if err := sleepWithContext(ctx, waitTime); err != nil {
				return nil, err
			}
			continue
		}

		// Server error -> wait and retry
		if statusCode >= 500 {
			if err := sleepWithContext(ctx, time.Duration(math.Pow(2, float64(attempt)))*time.Second); err != nil {
				return nil, err
			}
			continue
		}

		if statusCode == 404 {
			return nil, ErrNotFound
		}

		if statusCode == 403 {
			if strings.Contains(string(respBody), "Quoting this post is not allowed") {
				return nil, ErrQuoteNotAllowed
			}
			return nil, ErrForbidden
		}

		// Other client errors (400, 401, etc.) → don't retry
		if statusCode < 200 || statusCode >= 300 {
			return nil, fmt.Errorf("api error (status %d): %s", statusCode, string(respBody))
		}

		// Success
		return respBody, nil
	}

	return nil, fmt.Errorf("max retries exceeded")
}

func (c *Client) GetBotUserID() string { return c.BotUserID }
func (c *Client) GetSinceID() string   { return c.SinceID }
func (c *Client) SetSinceID(id string) { c.SinceID = id }

func sleepWithContext(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
