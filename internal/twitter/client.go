package twitter

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"

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
		BotUserID:     "",
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

	req, err := http.NewRequest("GET", url.String(), nil)

	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	body, err := c.doRequestWithRetry(req)

	return body, err
}

func (c *Client) doPost(endpoint string, params interface{}) ([]byte, error) {
	url := c.BaseUrl + endpoint

	jsonBody, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("Failed to marshal params: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewReader(jsonBody))

	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	body, err := c.doRequestWithRetry(req)

	return body, err
}

func (c *Client) doRequest(req *http.Request) ([]byte, int, error) {
	// Make the request
	resp, err := c.Authenticated.Do(req)
	if err != nil {
		return nil, 0, err // Network error, no status code
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, err
	}

	return body, resp.StatusCode, nil
}

func (c *Client) doRequestWithRetry(req *http.Request) ([]byte, error) {
	maxRetries := 3

	for attempt := 0; attempt <= maxRetries; attempt++ {
		body, statusCode, err := c.doRequest(req)

		//Network error
		if err != nil {
			delay(math.Pow(2, float64(attempt))) * time.Seconds
			continue
		}

		// Rate Limit -> wait until reset time
		if statusCode == 429 {
			resp := bytes.NewReader(body)
			delay(resp.body.rateLimit+1) * time.Seconds
			continue
		}

		// Server error -> wait and retry
		if statusCode >= 500 {
			delay(math.Pow(2, float64(attempt))) * time.Seconds
			continue
		}

		// Client error (400, 401, 404) → don't retry
		if statusCode < 200 || statusCode >= 300 {
			return nil, fmt.Errorf("api error (status %d): %s", statusCode, string(body))
		}

		// Success
		return body, nil
	}

	return nil, fmt.Errorf("max retries exceeded")
}
