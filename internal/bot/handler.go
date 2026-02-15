package bot

import (
	"fmt"
	"os/user"

	"github.com/jvsena42/memento/internal/config"
	"github.com/jvsena42/memento/internal/storage"
	"github.com/jvsena42/memento/internal/twitter"
)

type Handler struct {
	Client       *twitter.Client
	CapsuleStore *storage.CapsuleStore
	Config       *config.Config
}

func (h *Handler) ProcessMention(mention twitter.Tweet, users []twitter.User) error {

	var targetTweet *twitter.TweetResponse
	var err error

	if mention.InReplyToUserID != nil {
		targetTweet, err = h.Client.GetTweet(mention.ConversationID)
	} else {
		targetTweet, err = h.Client.GetTweet(mention.ID)
	}

	if err != nil {
		return fmt.Errorf("failed to fetch target tweet: %w", err)
	}

	tweetAuthor := findUser(targetTweet.Includes.Users, targetTweet.Tweet.AuthorID)

	requesterHandler := findUser(users, mention.AuthorID)

	if h.CapsuleStore.TweetAlreadySaved(targetTweet.Tweet.ID) {
		h.Client.PostTweet("This one's already saved! ⏳", "", mention.ID)
		return nil
	}

	if h.CapsuleStore.UserSavedToday()(mention.AuthorID) {
		h.Client.PostTweet("Come back tomorrow! 🕰️", "", mention.ID)
		return nil
	}

	
}

func findUser(users []twitter.User, userID string) string {
	for _, user : = range users {
		if user.ID == userID {
			return user.UserName
		}
	}
	return ""
}
