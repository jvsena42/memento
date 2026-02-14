package twitter

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/dghubble/oauth1"

	"github.com/jvsena42/memento/internal/config"
)

type Client struct {
	Authenticated *http.Client
	BotUserID     string
	BaseUrl       string
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

	resp, err := c.Authenticated.Get(url.String())
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("api error (status %d): %s", resp.StatusCode, string(body))
	}

	return body, err
}

func (c *Client) doPost(endpoint string, params interface{}) ([]byte, error) {
	url := c.BaseUrl + endpoint

	jsonBody, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("Failed to marshal params: %w", err)
	}

	resp, err := c.Authenticated.Post(url, "application/json", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("api error (status %d): %s", resp.StatusCode, string(body))
	}

	return body, nil
}
