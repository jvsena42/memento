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
	"time"

	"github.com/dghubble/oauth1"

	"github.com/jvsena42/memento/internal/config"
)

type Client struct {
	Authenticated *http.Client
	BotUserID     string
	BaseUrl       string
	SinceID       string
}

func NewClient(cfg *config.Config) *Client {

	config := oauth1.Config{
		ConsumerKey:    cfg.TwitterAPIKey,
		ConsumerSecret: cfg.TwitterAPISecret,
	}
	token := oauth1.NewToken(cfg.TwitterAccessToken, cfg.TwitterAccessSecret)

	return &Client{
		Authenticated: config.Client(context.Background(), token),
		BaseUrl:       "https://api.twitter.com",
		BotUserID:     cfg.BotUserID,
		SinceID:       "",
	}
}

func (c *Client) doGet(endpoint string, params map[string]string) ([]byte, error) {
	url, err := url.Parse(c.BaseUrl + endpoint)
	if err != nil {
		return nil, err
	}

	query := url.Query()
	for key, value := range params {
		query.Set(key, value)
	}

	url.RawQuery = query.Encode()

	body, err := c.doRequestWithRetry("GET", url.String(), nil)

	return body, err
}

func (c *Client) doPost(endpoint string, params interface{}) ([]byte, error) {
	url := c.BaseUrl + endpoint

	jsonBody, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("Failed to marshal params: %w", err)
	}

	body, err := c.doRequestWithRetry("POST", url, jsonBody)

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

func (c *Client) doRequestWithRetry(method string, url string, body []byte) ([]byte, error) {
	maxRetries := 3

	for attempt := 0; attempt <= maxRetries; attempt++ {
		req, err := http.NewRequest(method, url, bytes.NewReader(body))

		if err != nil {
			slog.Warn("request creation failed", "error", err)
			time.Sleep(time.Duration(math.Pow(2, float64(attempt))) * time.Second)
			continue
		}

		if method == "POST" {
			req.Header.Set("Content-Type", "application/json")
		}
		respBody, statusCode, header, err := c.doRequest(req)

		//Network error
		if err != nil {
			time.Sleep(time.Duration(math.Pow(2, float64(attempt))) * time.Second)
			continue
		}

		// Rate Limit -> wait until reset time
		if statusCode == 429 {
			resetStr := header.Get("x-rate-limit-reset")
			resetUnix, _ := strconv.ParseInt(resetStr, 10, 64)
			waitTime := time.Until(time.Unix(resetUnix, 0)) + 1*time.Second
			slog.Warn("rate limited, waiting", "seconds", waitTime.Seconds())
			time.Sleep(waitTime)
			continue
		}

		// Server error -> wait and retry
		if statusCode >= 500 {
			time.Sleep(time.Duration(math.Pow(2, float64(attempt))) * time.Second)
			continue
		}

		// Client error (400, 401, 404) → don't retry
		if statusCode < 200 || statusCode >= 300 {
			return nil, fmt.Errorf("api error (status %d): %s", statusCode, string(respBody))
		}

		// Success
		return respBody, nil
	}

	return nil, fmt.Errorf("max retries exceeded")
}
