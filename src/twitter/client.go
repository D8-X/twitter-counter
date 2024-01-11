package twitter

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"github.com/go-resty/resty/v2"
)

// Twitter V2 API endpoint with trailing slash
const TwitterV2API = "https://api.twitter.com/2/"

// Whenever HTTP 429 is returned from API, this error will be returned from
// Client func calls.
type ErrRateLimited struct {
	ResetTimestamp int64
}

func (e *ErrRateLimited) Error() string {
	return "rate limited"
}

// Client queries twitter API endpoints and fetches API data. Client is not
// responsible for any rate limiting or throttling. It only issues requests and
// parses the responses. User is responsible for any error handling. Twitter API
// is inherently quite restrictive and even processing something like 1000
// tweets will get rate limited pretty fast.
type Client interface {
	// FetchUserTweets fetches timeline tweets which include tweets, retweets,
	// replies, quote tweets for given userId
	//
	// Limitations are: maximum 3200 in the past. Up to 100 tweets per single
	// request. Rate limits based on the subscription plan.
	//
	// For more, see
	// https://developer.twitter.com/en/docs/twitter-api/tweets/timelines/introduction
	FetchUserTweets(userId string, options ...ApiRequestOption) (*TweetsResponse, error)

	// FetchUserLikedTweets fetches the tweets liked by the user.
	//
	// Rate limits based on the subscription plan. For pro plan 5/15 mins
	//
	FetchUserLikedTweets(userId string, options ...ApiRequestOption) (*TweetsResponse, error)

	// FetchTweetLikers fetches users who liked the given tweetId.
	//
	// Limitations: 100 items per request. Rate limits based on the
	// subscription. For pro plan: 25/15min
	FetchTweetLikers(tweetId string, options ...ApiRequestOption) (*UserInteractorsResponse, error)

	// FetchTweetRetweeters fetches users who retweeted the given tweetId tweet.
	//
	// Limitations: 100 items per request. Rate limits based on the
	// subscription. Pro plan: 5/15min
	FetchTweetRetweeters(tweetId string, options ...ApiRequestOption) (*UserInteractorsResponse, error)

	// FindUserDetails is a helper method to find user ids by names.
	FindUserDetails(userNames []string) (*UserLookupResponse, error)
}

func NewAuthBearerClient(authBearer string) *twitterHTTPClient {
	r := resty.New()
	r.SetHeader("Authorization", "Bearer "+authBearer)

	return &twitterHTTPClient{
		authBearer: authBearer,
		r:          r,
	}
}

var _ Client = (*twitterHTTPClient)(nil)

// twitterHTTPClient is the basic http client for Twitter V2 API authenticating
// with authBearer
type twitterHTTPClient struct {
	authBearer string
	r          *resty.Client
}

func (t *twitterHTTPClient) sendGet(endpoint string, options ...ApiRequestOption) ([]byte, error) {
	req := t.r.R()

	for _, opt := range options {
		opt.Apply(req)
	}

	resp, err := req.Get(endpoint)
	if err != nil {
		return nil, err
	}
	body := resp.Body()

	fullEndpoint := endpoint + "?" + req.QueryParam.Encode()
	slog.Info("sent a GET twitter API request",
		slog.String("endpoint", fullEndpoint),
	)

	if resp.StatusCode() != 200 {
		slog.Error(
			"response failed",
			slog.String("endpoint", fullEndpoint),
			slog.Int("status", resp.StatusCode()),
			slog.String("repsonse", string(body)),
		)

		// For rate limited requests -
		if resp.StatusCode() == 429 {
			resetTimeInt := int64(0)
			if resetTime := resp.Header().Get("x-rate-limit-reset"); resetTime != "" {
				if ri, err := strconv.ParseInt(resetTime, 10, 64); err == nil {
					resetTimeInt = ri
				}
			}

			return nil, &ErrRateLimited{ResetTimestamp: resetTimeInt}
		}

		return nil, fmt.Errorf("response failed: %d", resp.StatusCode())
	}

	return body, nil
}

func (t *twitterHTTPClient) FindUserDetails(userNames []string) (*UserLookupResponse, error) {
	endpoint := TwitterV2API + "users/by"

	body, err := t.sendGet(endpoint, &OptApplyQueryParam{
		Key:   "usernames",
		Value: strings.Join(userNames, ","),
	})
	if err != nil {
		return nil, err
	}

	ret := &UserLookupResponse{
		Raw: body,
	}
	if err := json.Unmarshal(body, ret); err != nil {
		return nil, fmt.Errorf("parsing user lookup response: %w", err)
	}

	return ret, nil
}

// FetchUserTweets sends a user tweets request and parses it. Collected
// iformation includes tweet text, tweet id, conversation id,
func (t *twitterHTTPClient) FetchUserTweets(userId string, options ...ApiRequestOption) (*TweetsResponse, error) {
	endpoint := TwitterV2API + "users/" + userId + "/tweets"
	body, err := t.sendGet(endpoint,
		append(
			options,
			// Append the conversation_id expansion to get the information if
			// tweet is a reply in conversation. For simple tweets the
			// conversation_id should be the same tweet id
			&OptApplyQueryParam{
				Key:   "tweet.fields",
				Value: "conversation_id,referenced_tweets",
			},
			// Append information about conversation tweet author
			// (in_reply_to_user_id) and referenced tweets and author_id (any of
			// these might be empty too if a tweet is just a simple tweet)
			&OptApplyQueryParam{
				Key:   "expansions",
				Value: "in_reply_to_user_id,referenced_tweets.id,referenced_tweets.id.author_id",
			},
		)...,
	)
	if err != nil {
		return nil, err
	}

	ret := &TweetsResponse{
		Raw: body,
	}
	if err := json.Unmarshal(body, ret); err != nil {
		return nil, fmt.Errorf("parsing user tweets response: %w", err)
	}

	return ret, nil
}

// FetchUserTweets send a user tweets request and parses it. Collected
// iformation includes tweet text, tweet id, author id, and might also
// includes.users. Tweet author ids are the most important for data processing.
// Up to 100 results per request.
func (t *twitterHTTPClient) FetchUserLikedTweets(userId string, options ...ApiRequestOption) (*TweetsResponse, error) {
	endpoint := TwitterV2API + "users/" + userId + "/liked_tweets"
	body, err := t.sendGet(endpoint,
		append(
			options,
			// Append information about conversation tweet author user id
			&OptApplyQueryParam{
				Key:   "expansions",
				Value: "author_id",
			},
		)...,
	)
	if err != nil {
		return nil, err
	}

	ret := &TweetsResponse{
		Raw: body,
	}
	if err := json.Unmarshal(body, ret); err != nil {
		return nil, fmt.Errorf("parsing user liked tweets response: %w", err)
	}

	return ret, nil
}

// FetchTweetLikers finds the users who liked given tweetId tweet. Limitations
func (t twitterHTTPClient) FetchTweetLikers(tweetId string, options ...ApiRequestOption) (*UserInteractorsResponse, error) {
	endpoint := TwitterV2API + "tweets/" + tweetId + "/liking_users"
	body, err := t.sendGet(endpoint, options...)
	if err != nil {
		return nil, err
	}

	ret := &UserInteractorsResponse{
		Raw: body,
	}
	if err := json.Unmarshal(body, ret); err != nil {
		return nil, fmt.Errorf("parsing user tweet likers response: %w", err)
	}

	return ret, nil
}

func (t twitterHTTPClient) FetchTweetRetweeters(tweetId string, options ...ApiRequestOption) (*UserInteractorsResponse, error) {
	endpoint := TwitterV2API + "tweets/" + tweetId + "/retweeted_by"
	body, err := t.sendGet(endpoint, options...)
	if err != nil {
		return nil, err
	}

	ret := &UserInteractorsResponse{
		Raw: body,
	}
	if err := json.Unmarshal(body, ret); err != nil {
		return nil, fmt.Errorf("parsing tweet retweets response: %w", err)
	}

	return ret, nil

}
