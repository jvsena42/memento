package twitter

import (
	"encoding/json"
	"fmt"
)

func (c *Client) GetMentions() (*TweetsResponse, error) {
	params := map[string]string{
		"tweet.fields": "author_id,text,created_at,conversation_id,in_reply_to_user_id",
		"expansions":   "author_id",
	}
	if c.SinceID != "" {
		params["since_id"] = c.SinceID
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
