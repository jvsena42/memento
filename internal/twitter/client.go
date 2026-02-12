package twitter

import (
	"context"
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
	fullUrl, err := url.Parse(c.BaseUrl + endpoint)
	if err != nil {
		return nil, err
	}

	query := fullUrl.Query()
	for key, value := range params {
		query.Set(key, value)
	}

	fullUrl.RawQuery = query.Encode()

	resp, err := c.Authenticated.Get(fullUrl.String())
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
