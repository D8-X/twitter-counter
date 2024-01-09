package twitter

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"golang.org/x/time/rate"
)

// UserInteractions defines the interaction graph structure for a single user
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

// NewDevAnalyzer creates a Analyzer with sensible defaults for development. For
// production usage please create a new Analyzer manually.
func NewDevAnalyzer(c Client) *Analyzer {
	return &Analyzer{
		Client:                 c,
		MaxTweetsPerRequest:    5,
		UserTweetsToFetch:      200,
		UserLikedTweetsToFetch: 200,
		Logger:                 slog.Default(),
		FetchTweetsLimit:       rate.NewLimiter(rate.Every(time.Minute*15), 5),
	}
}

func NewProductionAnalyzer(c Client) *Analyzer {
	return &Analyzer{
		Client:                 c,
		MaxTweetsPerRequest:    100,
		UserTweetsToFetch:      1000,
		UserLikedTweetsToFetch: 1000,
		Logger:                 slog.Default(),
	}
}

// Analyzer utilizes provided client to construct the interaction graph for a
// user id
type Analyzer struct {
	Client Client
	Logger *slog.Logger

	// Number of user tweets to fetch in a single request. Number between 5 and
	// 100 (max_results query parameter)
	MaxTweetsPerRequest uint

	FetchTweetsLimit *rate.Limiter

	// How many timeline tweets to check for user. Maximum number is 3200
	// (limitation of twitter API). This only controls how many tweets will be
	// retrieved from the timeline in a single analysis run. The actual number
	// of tweets to be processed might be larger (likes, other user like and
	// retweet checks).
	UserTweetsToFetch uint

	// How many user liked tweets to fetch. Similar to UserTweetsToFetch, but
	// fetches tweets which user in question liked.
	UserLikedTweetsToFetch uint
}

// ProcessDirectUserInteractions processes and counts the direct user
// interactions from given r. These include: replies to other users, retweets of
// other user tweets
func (a *Analyzer) ProcessDirectUserInteractions(r *TweetsResponse, result *UserInteractions) {
	for _, tweet := range r.Data {
		// Process replies and increment reply counters. Replies contain
		// InReplyToUserId field and can be used directly
		if tweet.InReplyToUserId != "" {
			if _, ok := result.RepliesToOtherUsers[tweet.InReplyToUserId]; !ok {
				result.RepliesToOtherUsers[tweet.InReplyToUserId] = 0
			}
			result.RepliesToOtherUsers[tweet.InReplyToUserId]++
		}

		// Retweets/Quoted RTs will contain only the reference to the original
		// tweet. We need to collect the user id of the original tweet from
		// includes.
		if tweet.InReplyToUserId == "" && len(tweet.ReferencedTweets) > 0 {
			for _, referencedTweet := range tweet.ReferencedTweets {
				// Find the referenced tweet from includes
				originalTweet := r.FindReferencedTweet(referencedTweet.Id)
				if originalTweet == nil {
					a.Logger.Warn("referenced tweet not found in includes",
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

func collectAllWithPaginationAndThrottling[T any](fn func(nextToken string)) {

}

// CreateUserInteractionGraph runs a full interaction check for a given user id.
// Note that due to rate limitin completing the interaction run might take a
// long time. Make sure you use sensible values for limits.
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

	for {
		if reservation := a.FetchTweetsLimit.Reserve(); reservation.OK() {

			tweets, err := a.Client.FetchUserTweets(userTwitterId,
				append(opts, OptApplyMaxResults(strconv.Itoa(int(a.MaxTweetsPerRequest))))...,
			)
			if err != nil {
				slog.Error("failed to fetch user tweets", err)
				// TODO do not kill the process, but handle the possible rate
				// limiting error
				break
			}

			// Process direct interactions
			a.ProcessDirectUserInteractions(tweets, result)

			for _, tweet := range tweets.Data {
				// Collect the tweet ids for replies, retweets and likes checking by
				// other user_ids.
				collectedUserTweetIds = append(collectedUserTweetIds, tweet.TweetId)
			}

			// Stop when we don't have more results or we reached our defined
			// limit
			if len(tweets.Data) < int(a.MaxTweetsPerRequest) || tweets.Meta.NextToken == "" || len(collectedUserTweetIds) >= int(a.UserTweetsToFetch) {
				break
			}

		} else {
			a.Logger.Info("rate limited, waiting")
			a.FetchTweetsLimit.Wait(context.Background())
		}
	}

	fmt.Printf("Collected tweet ids: %+v\n len: %d\n", collectedUserTweetIds, len(collectedUserTweetIds))

	// Collect user likes
	// a.Client.FetchUserLikedTweets(userTwitterId)

	// For each collected user tweet find other user

	// likes,

	// replies,

	// retweets

}
