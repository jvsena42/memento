package twitter

import (
	"encoding/json"
	"fmt"
)

func (c *Client) GetMentions(sinceID string) (*TweetsResponse, error) {
	params := map[string]string{
		"tweet.fields": "author_id,text,created_at,conversation_id,in_reply_to_user_id",
		"expansions":   "author_id",
	}
	if sinceID != "" {
		params["since_id"] = sinceID
	}
	respBytes, err := c.doGet(fmt.Sprintf("/2/users/%s/mentions", c.BotUserID), params)
	if err != nil {
		return nil, err
	}

	var response TweetsResponse
	if err := json.Unmarshal(respBytes, &response); err != nil {
		return nil, err
	}

	if response.Meta != nil {
		c.SinceID = response.Meta.NewestID
	}

	return &response, nil
}
