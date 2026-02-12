package twitter

import (
	"context"
	"net/http"

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
