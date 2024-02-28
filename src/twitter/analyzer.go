package twitter

import (
	"errors"
	"io"
	"log"
	"log/slog"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"sync"
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

	// Same as bove, just given by other users to current UserTwitterId
	RetweetsFromOtherUsers map[string]uint

	// Same as bove, just given by other users to current UserTwitterId
	RepliesFromOtherUsers map[string]uint

	// Likes given by current UserTwtiterId. Key is other user id, Value is
	// number of likes for that particular user id.
	UserLikedTweets map[string]uint
}

// Ranked returns ranked list of user ids and their interaction values based on
// given UserInteractions data. First returned slice is the user ids, second is
// the interaction counts.
func (u *UserInteractions) Ranked() ([]string, []uint) {
	all := map[string]uint{}

	setHelper := func(k string, v uint) {
		if _, ok := all[k]; !ok {
			all[k] = 0
		}
		all[k] += v
	}

	for k, v := range u.RepliesToOtherUsers {
		setHelper(k, v)
	}
	for k, v := range u.RetweetsToOtherUsers {
		setHelper(k, v)
	}
	for k, v := range u.UserLikedTweets {
		setHelper(k, v)
	}
	for k, v := range u.RepliesFromOtherUsers {
		setHelper(k, v)
	}
	for k, v := range u.RetweetsFromOtherUsers {
		setHelper(k, v)
	}

	userIds := make([]string, 0, len(all))
	userValues := make([]uint, len(all))

	for k := range all {
		userIds = append(userIds, k)
	}

	// Order in descending order
	sort.Slice(userIds, func(i, j int) bool {
		return all[userIds[i]] > all[userIds[j]]
	})

	for i, userId := range userIds {
		userValues[i] = all[userId]
	}

	return userIds, userValues
}

func NewUserInteractionsObject() *UserInteractions {
	return &UserInteractions{
		RepliesToOtherUsers:    map[string]uint{},
		RetweetsToOtherUsers:   map[string]uint{},
		UserLikedTweets:        map[string]uint{},
		RetweetsFromOtherUsers: map[string]uint{},
		RepliesFromOtherUsers:  map[string]uint{},
	}
}

// NewDevAnalyzer creates a Analyzer with sensible defaults for development with
// BASIC API plan. For production usage please create a new Analyzer manually or
// use NewProductionAnalyzer.
func NewDevAnalyzer(c Client) *Analyzer {
	return &Analyzer{
		Client:                 c,
		MaxTweetsPerRequest:    100,
		UserTweetsToFetch:      300,
		UserLikedTweetsToFetch: 300,
		Logger:                 slog.Default(),
		// 10 requests per 15 minutes for timeline requests per app
		TimelineLimiter:    NewRateLimiter(10, time.Minute*15),
		LikedTweetsLimiter: NewRateLimiter(5, time.Minute*15),
	}
}

// NewProductionAnalyzer constructs a new analyzer with rate limiting for PRO
// API plan.
func NewProductionAnalyzer(c Client) *Analyzer {
	return &Analyzer{
		Client:                 c,
		TimelineLimiter:        NewRateLimiter(75, time.Minute*15),
		LikedTweetsLimiter:     NewRateLimiter(75, time.Minute*15),
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
	TimelineLimiter ApiRateLimiter

	// Rate limiter for user's liked tweets endpoint
	LikedTweetsLimiter ApiRateLimiter

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

// ProcessUserInteractions processes and counts the bidirectional user
// interactions from given r for userTwitterId. These include: replies from and
// to other users, retweets of other user tweets and retweets of user's tweets
// by other users.
func (a *Analyzer) ProcessUserInteractions(userTwitterId string, r *TweetsResponse, result *UserInteractions) {
	for _, tweet := range r.Data {
		// Process replies and increment reply counters. Replies contain
		// InReplyToUserId field and can be used directly to determine if reply
		// is done by our userTwitterId for by other user
		if tweet.InReplyToUserId != "" {
			// Reply from other user to userTwitterId
			if tweet.InReplyToUserId == userTwitterId {
				if _, ok := result.RepliesFromOtherUsers[tweet.AuthorUserId]; !ok {
					result.RepliesFromOtherUsers[tweet.AuthorUserId] = 0
				}
				result.RepliesFromOtherUsers[tweet.AuthorUserId]++
			} else {
				// Reply from userTwitterId to other user
				if _, ok := result.RepliesToOtherUsers[tweet.InReplyToUserId]; !ok {
					result.RepliesToOtherUsers[tweet.InReplyToUserId] = 0
				}
				result.RepliesToOtherUsers[tweet.InReplyToUserId]++
			}

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

// ProcessUserLikes collects author user ids of user's liked tweets
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

// CollectAndProcessEndpoint collects paginated data via fetchFunc and processes
// the responses via processAndContinue. This function also handles pagination
// and rate limiting automatically.
func (a *Analyzer) CollectAndProcessEndpoint(endpointName string, rateLimiter ApiRateLimiter, fetchFunc func(opts []ApiRequestOption) (*TweetsResponse, error), processAndContinue func(*TweetsResponse) bool) {
	apiRequestOpts := []ApiRequestOption{}
	for {
		if rateLimiter.Allow() {
			tweets, err := fetchFunc(apiRequestOpts)
			if err != nil {
				// When we get limited from the API, set the limiter to limited
				// state and set the next available run time to the reset
				// timestamp if available.
				if erl, ok := err.(*ErrRateLimited); ok {
					rateLimiter.MarkLimited()
					if erl.ResetTimestamp > 0 {
						rateLimiter.SetAvailableTime(erl.ResetTimestamp)
					}

					a.Logger.Warn(
						"rate limited, waiting to run next request",
						slog.Time("next_reset_from_api", time.Unix(erl.ResetTimestamp, 0)),
						slog.String("endpoint", endpointName),
					)
					continue
				}

				// Exit on other errors
				break
			}

			// Process the result and exit when done
			if !processAndContinue(tweets) {
				break
			}

			// If next token is still available - set it in options
			apiRequestOpts = []ApiRequestOption{OptApplyPaginationToken(tweets.Meta.NextToken)}

		} else {
			wt := rateLimiter.WaitTime()
			a.Logger.Info("rate limit reached, waiting to run next request",
				slog.Duration("wait_time", wt),
				slog.Time("next_run", time.Now().Add(wt)),
				slog.String("endpoint", endpointName),
			)
			time.Sleep(wt)
		}
	}
}

// CreateUserInteractionGraph runs a full interaction check for a given user id.
// Note that due to rate limiting, completing the interaction run might take a
// long time. Make sure you use sensible values for limits.
func (a *Analyzer) CreateUserInteractionGraph(userTwitterId string) (*UserInteractions, error) {
	result := NewUserInteractionsObject()
	result.UserTwitterId = userTwitterId
	userInteractionsMu := sync.Mutex{}

	collectedTweetsNum := 0
	collectedLikedTweetsNum := 0

	wg := sync.WaitGroup{}

	// Process the tweets timeline
	wg.Add(1)
	go func() {
		a.CollectAndProcessEndpoint("user-timeline-tweets", a.TimelineLimiter,
			func(opts []ApiRequestOption) (*TweetsResponse, error) {
				return a.Client.FetchUserTweets(userTwitterId,
					append(opts, OptApplyMaxResults(strconv.Itoa(int(a.MaxTweetsPerRequest))))...,
				)
			},
			func(tweets *TweetsResponse) bool {
				userInteractionsMu.Lock()
				defer userInteractionsMu.Unlock()

				// Process direct interactions
				a.ProcessUserInteractions(userTwitterId, tweets, result)

				collectedTweetsNum += len(tweets.Data)
				a.Logger.Info("collected timeline tweets", slog.Int("count", collectedTweetsNum))

				// Stop when we don't have more results or we reached our defined
				// limit
				if len(tweets.Data) < int(a.MaxTweetsPerRequest) || tweets.Meta.NextToken == "" || collectedTweetsNum >= int(a.UserTweetsToFetch) {
					return false
				}
				return true
			},
		)
		wg.Done()
	}()

	// Process user liked tweets
	wg.Add(1)
	go func() {
		a.CollectAndProcessEndpoint("user-liked-tweets", a.LikedTweetsLimiter,
			func(opts []ApiRequestOption) (*TweetsResponse, error) {
				return a.Client.FetchUserLikedTweets(userTwitterId,
					append(opts, OptApplyMaxResults(strconv.Itoa(int(a.MaxTweetsPerRequest))))...,
				)
			},
			func(tweets *TweetsResponse) bool {
				userInteractionsMu.Lock()
				defer userInteractionsMu.Unlock()

				a.ProcessUserLikes(tweets, result)

				collectedLikedTweetsNum += len(tweets.Data)
				a.Logger.Info("collected liked tweets", slog.Int("count", collectedLikedTweetsNum))

				// Stop when we don't have more results or we reached our defined
				// limit
				if len(tweets.Data) < int(a.MaxTweetsPerRequest) || tweets.Meta.NextToken == "" || collectedLikedTweetsNum >= int(a.UserLikedTweetsToFetch) {
					return false
				}
				return true
			},
		)
		wg.Done()
	}()

	wg.Wait()

	// Remove all current user entries from the result
	delete(result.RepliesToOtherUsers, userTwitterId)
	delete(result.RetweetsToOtherUsers, userTwitterId)
	delete(result.UserLikedTweets, userTwitterId)

	return result, nil
}

func (a *Analyzer) AnalyzeNewUserInteractions(newUserTwitterId string, referringUserIds []string) (*UserInteractions, error) {
	result := NewUserInteractionsObject()

	// TODO Timestamp used to determine whether we should stop processing tweets
	// for current batch of referrringUserIds and move the currentReferrersIndex
	// tweetsTimeLimit := -1

	// Assuming the average twitter id length is 15 characters, the maximum
	// number of referring users per query is 3. This can be calculated by
	// running the following test code:
	//
	// arr := []string{} for i := 0; i < 20; i++ {
	//  id := "123456789011123"
	//  arr = append(arr, id)
	//  q := buildSearchQuery(id, arr)
	//  fmt.Printf("Number of ids %d Query length: %d\n", i+1, len(q), q)
	// }
	//
	// The ids might be longer or shorter, for more recent users it seems to be
	// around 18 chars. We'll start with 6 referring users per query and back
	// off
	numbReferrersPerQuery := 6
	currentReferrersIndex := 0
	currentNumReferrersToUse := numbReferrersPerQuery

	// Process all referringUserIds for newUserTwitterId
	for {
		var resp *TweetsResponse
		var err error

		// Find the correct number of referring users to use per query
		for {
			resp, err = a.Client.FetchUserInteractionsWithSearch(newUserTwitterId, referringUserIds[currentReferrersIndex:currentReferrersIndex+currentNumReferrersToUse])
			if err != nil {
				if errors.Is(err, ErrSearchQueryTooLong) {
					currentNumReferrersToUse--
					a.Logger.Warn("search query too long, reducing number of referring users per query", slog.Int("next_referring_users_num", currentNumReferrersToUse))
				} else {
					return nil, err
				}
			} else {
				break
			}
		}
		a.ProcessUserInteractions(newUserTwitterId, resp, result)

		// Once we don't have any more results or we reached time threshold,
		// move on to the next batch of referring users
		if resp.Meta.NextToken == "" {
			// TODO check the tweetsTimeLimit
			// if resp.Data[len(resp.Data)-1].CreatedAt < tweetsTimeLimit {
			// }

			// Reset and continue with the next batch of referring users
			currentReferrersIndex += max(currentReferrersIndex+currentNumReferrersToUse, len(referringUserIds))
			currentNumReferrersToUse = numbReferrersPerQuery
			a.Logger.Info(
				"no more results, moving to the next batch of referring users",
				slog.Int("next_referring_users_index", currentReferrersIndex),
			)

			// We're done
			if currentReferrersIndex >= len(referringUserIds) {
				break
			}
		}
	}

	return result, nil
}

func printer(body []byte) {
	// TMP print stuff
	f, err := os.OpenFile("tmp.json", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		log.Fatal(err)
	}
	defer os.Remove("tmp.json")
	_, err = io.Copy(f, strings.NewReader(string(body)))
	if err != nil {
		log.Fatal(err)
	}
	cmd := exec.Command("jq", ".", "tmp.json")
	cmd.Stdout = os.Stdout
	if err := cmd.Run(); err != nil {
		log.Fatal(err)
	}
}
