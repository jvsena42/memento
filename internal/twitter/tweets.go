package twitter

import (
	"context"
	"encoding/json"
	"strings"
)

type PostTweetRequest struct {
	Text         string       `json:"text"`
	QuoteTweetID string       `json:"quote_tweet_id,omitempty"`
	Reply        *ReplyConfig `json:"reply,omitempty"`
}

type ReplyConfig struct {
	InReplyToTweetID string `json:"in_reply_to_tweet_id"`
}

func (c *Client) GetTweet(ctx context.Context, id string) (*TweetResponse, error) {
	params := map[string]string{
		"tweet.fields": "author_id,text,created_at,conversation_id,in_reply_to_user_id,referenced_tweets",
		"expansions":   "author_id",
	}
	respBytes, err := c.doGet(ctx, "/2/tweets/"+id, params)
	if err != nil {
		return nil, err
	}

	var response TweetResponse
	if err := json.Unmarshal(respBytes, &response); err != nil {
		return nil, err
	}

	return &response, nil
}

// GetTweets fetches multiple tweets by ID in a single API call (up to 100).
// Tweets that are not found or deleted will be absent from the returned map.
func (c *Client) GetTweets(ctx context.Context, ids []string) (map[string]Tweet, []User, error) {
	params := map[string]string{
		"ids":          strings.Join(ids, ","),
		"tweet.fields": "author_id,text,created_at,conversation_id,in_reply_to_user_id,referenced_tweets",
		"expansions":   "author_id",
	}
	respBytes, err := c.doGet(ctx, "/2/tweets", params)
	if err != nil {
		return nil, nil, err
	}

	var response TweetsResponse
	if err := json.Unmarshal(respBytes, &response); err != nil {
		return nil, nil, err
	}

	tweetMap := make(map[string]Tweet, len(response.Tweets))
	for _, t := range response.Tweets {
		tweetMap[t.ID] = t
	}

	var users []User
	if response.Includes != nil {
		users = response.Includes.Users
	}

	return tweetMap, users, nil
}

func (c *Client) PostTweet(ctx context.Context, text string, quoteTweetID string, replyToID string) (*TweetResponse, error) {
	request := PostTweetRequest{
		Text: text,
	}

	if quoteTweetID != "" {
		request.QuoteTweetID = quoteTweetID
	}

	if replyToID != "" {
		request.Reply = &ReplyConfig{
			InReplyToTweetID: replyToID,
		}
	}

	respBytes, err := c.doPost(ctx, "/2/tweets", request)
	if err != nil {
		return nil, err
	}

	var response TweetResponse
	if err := json.Unmarshal(respBytes, &response); err != nil {
		return nil, err
	}

	return &response, nil
}
