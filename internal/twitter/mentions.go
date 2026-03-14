package twitter

import (
	"context"
	"encoding/json"
	"fmt"
)

func (c *Client) GetMentions(ctx context.Context) (*TweetsResponse, error) {
	params := map[string]string{
		"tweet.fields": "author_id,text,created_at,conversation_id,in_reply_to_user_id,referenced_tweets",
		"expansions":   "author_id,referenced_tweets.id,referenced_tweets.id.author_id",
	}
	if c.SinceID != "" {
		params["since_id"] = c.SinceID
	}

	var allTweets []Tweet
	var allUsers []User
	var allIncludedTweets []Tweet
	maxPages := c.MaxMentionPages
	if maxPages <= 0 {
		maxPages = 3
	}
	var response TweetsResponse
	for page := 0; page < maxPages; page++ {
		respBytes, err := c.doGet(ctx, fmt.Sprintf("/2/users/%s/mentions", c.BotUserID), params)
		if err != nil {
			return nil, err
		}
		if err := json.Unmarshal(respBytes, &response); err != nil {
			return nil, err
		}

		allTweets = append(allTweets, response.Tweets...)

		if response.Includes != nil {
			allUsers = append(allUsers, response.Includes.Users...)
			allIncludedTweets = append(allIncludedTweets, response.Includes.Tweets...)
		}

		if page == 0 && response.Meta != nil && response.Meta.NewestID != "" {
			c.SinceID = response.Meta.NewestID
		}

		if response.Meta == nil || response.Meta.NextToken == "" {
			break
		}

		params["pagination_token"] = response.Meta.NextToken
	}

	seen := make(map[string]struct{}, len(allTweets))
	deduped := allTweets[:0]
	for _, t := range allTweets {
		if _, ok := seen[t.ID]; !ok {
			seen[t.ID] = struct{}{}
			deduped = append(deduped, t)
		}
	}
	response.Tweets = deduped
	response.Includes = &Includes{
		Users:  allUsers,
		Tweets: allIncludedTweets,
	}

	return &response, nil
}
