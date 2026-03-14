package twitter

import "errors"

var (
	ErrNotFound        = errors.New("tweet not found or deleted")
	ErrForbidden       = errors.New("tweet is protected or account suspended")
	ErrQuoteNotAllowed = errors.New("quoting this post is not allowed")
)
