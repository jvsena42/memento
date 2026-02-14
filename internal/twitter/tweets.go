package twitter

import "encoding/json"

type PostTweetRequest struct {
	Text         string       `json:"text"`
	QuoteTweetID string       `json:"quote_tweet_id,omitempty"`
	Reply        *ReplyConfig `json:"reply,omitempty"`
}

type ReplyConfig struct {
	InReplyToTweetID string `json:"in_reply_to_tweet_id"`
}

func (c *Client) GetTweet(id string) (*TweetResponse, error) {
	params := map[string]string{
		"tweet.fields": "author_id,text,created_at,conversation_id,in_reply_to_user_id",
		"expansions":   "author_id",
	}
	respBytes, err := c.doGet("/2/tweets/"+id, params)
	if err != nil {
		return nil, err
	}

	var response TweetResponse
	if err := json.Unmarshal(respBytes, &response); err != nil {
		return nil, err
	}

	return &response, nil
}

func (c *Client) PostTweet(text string, quoteTweetID string, replyToID string) (*TweetResponse, error) {
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

	respBytes, err := c.doPost("/2/tweets", request)
	if err != nil {
		return nil, err
	}

	var response TweetResponse
	if err := json.Unmarshal(respBytes, &response); err != nil {
		return nil, err
	}

	return &response, nil
}
