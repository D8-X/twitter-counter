package twitter

import (
	"log/slog"
	"strconv"
	"time"
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
		UserLikedTweets:      map[string]uint{},
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
		// 10 requests per 15 minutes for timeline requests per app
		TimelineLimiter:    NewRateLimiter(10, time.Minute*15),
		LikedTweetsLimiter: NewRateLimiter(5, time.Minute*15),
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

	// Rate limiter for user's timeline endpoint
	TimelineLimiter *TwitterRateLimiter

	// Rate limiter for user's liked tweets endpoint
	LikedTweetsLimiter *TwitterRateLimiter

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

func (a *Analyzer) ProcessUserLikes(likedTweets *TweetsResponse, result *UserInteractions) {
	for _, tweet := range likedTweets.Data {
		if tweet.AuthorUserId != "" {
			if _, ok := result.UserLikedTweets[tweet.AuthorUserId]; !ok {
				result.UserLikedTweets[tweet.AuthorUserId] = 0
			}
			result.UserLikedTweets[tweet.AuthorUserId]++
		}
	}
}

// CreateUserInteractionGraph runs a full interaction check for a given user id.
// Note that due to rate limitin completing the interaction run might take a
// long time. Make sure you use sensible values for limits.
func (a *Analyzer) CreateUserInteractionGraph(userTwitterId string) (*UserInteractions, error) {
	result := NewUserInteractionsObject()
	result.UserTwitterId = userTwitterId

	timelineOpts := []ApiRequestOption{}
	collectedTweetsNum := 0
	collectedLikedTweetsNum := 0

	// Process the tweets timeline
	for {
		if a.TimelineLimiter.Allow() {
			tweets, err := a.Client.FetchUserTweets(userTwitterId,
				append(timelineOpts, OptApplyMaxResults(strconv.Itoa(int(a.MaxTweetsPerRequest))))...,
			)
			if err != nil {
				// When we got limited from the API, set the limiter to limited
				// state and set the next available run time if available.
				if erl, ok := err.(*ErrRateLimited); ok {
					a.TimelineLimiter.MarkLimited()
					if erl.ResetTimestamp > 0 {
						a.TimelineLimiter.SetAvailableTime(erl.ResetTimestamp)
					}

					a.Logger.Warn("call rate limited, waiting to run next request")
					continue
				}

				// Exit on other errors
				break
			}

			// Process direct interactions
			a.ProcessDirectUserInteractions(tweets, result)

			collectedTweetsNum += len(tweets.Data)

			// Stop when we don't have more results or we reached our defined
			// limit
			if len(tweets.Data) < int(a.MaxTweetsPerRequest) || tweets.Meta.NextToken == "" || collectedTweetsNum >= int(a.UserTweetsToFetch) {
				break
			}

			// If next token is still available - set it in options
			timelineOpts = []ApiRequestOption{OptApplyPaginationToken(tweets.Meta.NextToken)}

		} else {
			wt := a.TimelineLimiter.WaitTime()
			a.Logger.Info("collecting user timeline tweets, rate limit reached, waiting to run next request", slog.Duration("wait_time", wt))
			time.Sleep(wt)
		}
	}

	likedTweetsOpts := []ApiRequestOption{}
	// Process liked tweets
	for {
		if a.LikedTweetsLimiter.Allow() {
			tweets, err := a.Client.FetchUserLikedTweets(userTwitterId,
				append(likedTweetsOpts, OptApplyMaxResults(strconv.Itoa(int(a.MaxTweetsPerRequest))))...,
			)
			if err != nil {
				// When we got limited from the API, set the limiter to limited
				// state and set the next available run time if available.
				if erl, ok := err.(*ErrRateLimited); ok {
					a.LikedTweetsLimiter.MarkLimited()
					if erl.ResetTimestamp > 0 {
						a.LikedTweetsLimiter.SetAvailableTime(erl.ResetTimestamp)
					}

					a.Logger.Warn("call rate limited, waiting to run next request")
					continue
				}

				// Exit on other errors
				break
			}

			a.ProcessUserLikes(tweets, result)

			collectedLikedTweetsNum += len(tweets.Data)

			// Stop when we don't have more results or we reached our defined
			// limit
			if len(tweets.Data) < int(a.MaxTweetsPerRequest) || tweets.Meta.NextToken == "" || collectedLikedTweetsNum >= int(a.UserLikedTweetsToFetch) {
				break
			}

			// If next token is still available - set it in options
			likedTweetsOpts = []ApiRequestOption{OptApplyPaginationToken(tweets.Meta.NextToken)}

		} else {
			wt := a.TimelineLimiter.WaitTime()
			a.Logger.Info("collecting user liked tweets, rate limit reached, waiting to run next request", slog.Duration("wait_time", wt))
			time.Sleep(wt)
		}
	}

	return result, nil
}
