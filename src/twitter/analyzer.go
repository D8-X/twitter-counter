package twitter

import (
	"fmt"
	"log/slog"
)

// How many tweets to fetch in a single FetchUserTweets request. Must be a
// numeric string up to 100.
const TweetsPerRequestLimit = "25"

// UserInteractions defines the interaction graph for a single user
type UserInteractions struct {
	UserTwitterId string

	// Replies made by current UserTwitterId to other user ids. Data collected
	// directly from FetchUserTweets whenever in in_reply_to_user_id is present.
	// Key: other user id, Value: number of replies
	RepliesToOtherUsers map[string]uint

	// Retweets made by current UserTwitterId of other user ids posts. Data
	// collected directly from FetchUserTweets whenever in referenced_tweets
	RetweetsToOtherUsers map[string]uint

	// Likes given by current UserTwtiterId. Key is other user id, Value is
	// number of likes for that particular user id.
	UserLikedTweets map[string]uint
}

func NewUserInteractionsObject() *UserInteractions {
	return &UserInteractions{
		RepliesToOtherUsers:  map[string]uint{},
		RetweetsToOtherUsers: map[string]uint{},
	}
}

type Analyzer struct {
	Client Client
}

// ProcessDirectUserInteractions processes and counts the direct user
// interactions from given r. These include: replies to other users, retweets of
// other user tweets
func ProcessDirectUserInteractions(r *UserTweetsResponse, result *UserInteractions) {
	for _, tweet := range r.Data {
		// Process replies and increment reply counters. Replies contain
		// InReplyToUserId field and can be used directly
		if tweet.InReplyToUserId != "" {
			if _, ok := result.RepliesToOtherUsers[tweet.InReplyToUserId]; !ok {
				result.RepliesToOtherUsers[tweet.InReplyToUserId] = 0
			}
			result.RepliesToOtherUsers[tweet.InReplyToUserId]++
		}

		// Retweets will contain only the reference to the original tweet. We
		// need to collect the user id of the original tweet from includes.
		if tweet.InReplyToUserId == "" && len(tweet.ReferencedTweets) > 0 {
			for _, referencedTweet := range tweet.ReferencedTweets {
				// Find the referenced tweet from includes
				originalTweet := r.FindReferencedTweet(referencedTweet.Id)
				if originalTweet == nil {
					slog.Warn("referenced tweet not found in includes",
						slog.String("referenced_tweet_id", referencedTweet.Id),
						slog.String("tweet_id", tweet.TweetId),
					)
					continue
				}

				// Increment the retweet counter
				if _, ok := result.RetweetsToOtherUsers[originalTweet.AuthorUserId]; !ok {
					result.RetweetsToOtherUsers[originalTweet.AuthorUserId] = 0
				}
				result.RetweetsToOtherUsers[originalTweet.AuthorUserId]++
			}
		}
	}

}

// CreateUserInteractionGraph runs a full interaction check for a given user id.
func (a *Analyzer) CreateUserInteractionGraph(userTwitterId string) {

	result := NewUserInteractionsObject()
	result.UserTwitterId = userTwitterId

	// Print the result at the exit
	defer func() {
		fmt.Printf("The Result: %+v\n", result)
	}()

	// Start with collecting the user's tweets up to specific tweets limit or
	// date
	collectedUserTweetIds := []string{}
	opts := []ApiRequestOption{}

	// TODO introduce rate limiting
	for i := 0; i < 2; i++ {
		tweets, err := a.Client.FetchUserTweets(userTwitterId,
			append(opts, OptApplyMaxResults(TweetsPerRequestLimit))...,
		)
		if err != nil {
			slog.Error("failed to fetch user tweets", err)
			// TODO do not kill the process, but handle the possible rate
			// limiting error
			return
		}

		// Process direct interactions
		ProcessDirectUserInteractions(tweets, result)

		for _, tweet := range tweets.Data {
			// Collect the tweet ids for replies, retweets and likes checking by
			// other user_ids.
			collectedUserTweetIds = append(collectedUserTweetIds, tweet.TweetId)
		}
	}

	fmt.Printf("Collected tweet ids: %+v\n", collectedUserTweetIds)

	// Collect user likes

	// For each collected user tweet find other user

	// likes,

	// replies,

	// retweets

}
