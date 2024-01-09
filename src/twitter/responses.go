package twitter

import "encoding/json"

// TweetsResponse represents the data from tweets endpoints. It will include the
// tweets data as well as meta data, pagination (next_token), etc.
type TweetsResponse struct {
	Data []Tweet `json:"data"`
	Meta Meta    `json:"meta"`

	// Includes will contain users,tweets and other objects that are referenced
	// in Data tweets.
	Includes TweetIncludes `json:"includes"`

	// Raw is the raw json response from the API
	Raw json.RawMessage `json:"-"`
}

// FindReferencedTweet finds the referenced tweet by id. Returns nil if not
// found.
func (u *TweetsResponse) FindReferencedTweet(referencedTweetId string) *Tweet {
	for _, tweet := range u.Includes.Tweets {
		tweet := tweet
		if tweet.TweetId == referencedTweetId {
			return &tweet
		}
	}

	return nil
}

// Tweets includes object
type TweetIncludes struct {
	// Included referenced tweets
	Tweets []Tweet `json:"tweets"`
}

type Meta struct {
	NewestID string `json:"newest_id"`
	// NextToken is passed as pagination_token query parameter for the
	// next pagination page
	NextToken     string `json:"next_token"`
	OldestID      string `json:"oldest_id"`
	PreviousToken string `json:"previous_token"`
	ResultCount   int    `json:"result_count"`
}

type Tweet struct {
	AuthorUserId string `json:"author_id"`
	CreatedAt    string `json:"created_at"`
	TweetId      string `json:"id"`
	TweetText    string `json:"text"`

	// Original tweet id if this is a reply. See
	// https://developer.twitter.com/en/docs/twitter-api/tweets/timelines/api-reference/get-users-id-tweets
	ConversationTweetId string `json:"conversation_id"`

	// User id of the conversation tweet (this tweet is a reply to a tweet of
	// this InReplyToUserId user id). Will be empty when tweet is not a reply
	InReplyToUserId string `json:"in_reply_to_user_id"`

	// Referenced tweets refs. UserTweetsResponse.Includes.Tweets will include
	// full details of referenced tweets (linked via Id)
	ReferencedTweets []ReferencedTweetMeta `json:"referenced_tweets"`
}

type ReferencedTweetType string

const (
	Retweet ReferencedTweetType = "retweeted"
	Quoted  ReferencedTweetType = "quoted"
	Reply   ReferencedTweetType = "replied_to"
)

type ReferencedTweetMeta struct {
	Type ReferencedTweetType `json:"type"`
	Id   string              `json:"id"`
}

func (t Tweet) IsReply() bool {
	return t.ConversationTweetId != ""
}

// UserInteractorsResponse is a response from endpoints which return a list of
// users who interacted with something. For example likers or retweeters of a
// tweet.
//
// See
// https://developer.twitter.com/en/docs/twitter-api/tweets/likes/api-reference/get-tweets-id-liking_users
type UserInteractorsResponse struct {
	Data []UserDetail `json:"data"`
	// Only for next_token
	Meta Meta `json:"meta"`

	// Raw is the raw json response from the API
	Raw json.RawMessage `json:"-"`
}

// See
// https://developer.twitter.com/en/docs/twitter-api/users/lookup/api-reference/get-users-by
type UserLookupResponse struct {
	Data []UserDetail `json:"data"`

	Raw json.RawMessage `json:"-"`
}

type UserDetail struct {
	Id       string `json:"id"`
	Name     string `json:"name"`
	Username string `json:"username"`
}
