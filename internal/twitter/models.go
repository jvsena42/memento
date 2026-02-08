package twitter

import "time"

type Tweet struct {
	ID        string    `json:"id"`
	AuthorID  string    `json:"author_id"`
	Text      string    `json:"text"`
	CreatedAt time.Time `json:"created_at"`
}

type User struct {
	ID       string `json:"id"`
	UserName string `json:"username"`
}

type APIError struct {
	Title  string `json:"title"`
	Type   string `json:"type"`
	Detail string `json:"detail"`
	Status int    `json:"status"`
}

type Meta struct {
	NewestID      string `json:"newest_id"`
	NextToken     string `json:"next_token"`
	OldestID      string `json:"oldest_id"`
	PreviousToken string `json:"previous_token"`
	ResultCount   int    `json:"result_count"`
}

type Includes struct {
	Tweets []Tweet `json:"tweets"`
	Users  []User  `json:"users"`
}

type TweetResponse struct {
	Tweet    Tweet       `json:"data"`
	Includes []*Includes `json:"includes"`
	Meta     []*Meta     `json:"meta"`
	Errors   []APIError  `json:"errors"`
}

type TweetsResponse struct {
	Tweets   []Tweet     `json:"data"`
	Includes []*Includes `json:"includes"`
	Meta     []*Meta     `json:"meta"`
	Errors   []APIError  `json:"errors"`
}
