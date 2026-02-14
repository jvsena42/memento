package twitter

import "encoding/json"

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
