package twitter

import (
	"testing"

	"github.com/D8-X/twitter-counter/src/mocks"
	"github.com/stretchr/testify/assert"
)

func TestProcessDirectUserInteractions(t *testing.T) {

	tests := []struct {
		name          string
		userId        string
		expectResult  *UserInteractions
		inputResponse *TweetsResponse
		inputResult   *UserInteractions
	}{
		{
			name:   "ok",
			userId: "123",
			expectResult: &UserInteractions{
				UserTwitterId: "123",
				RepliesToOtherUsers: map[string]uint{
					"other-user-1": 5,
					"other-user-2": 1,
				},
				RetweetsToOtherUsers: map[string]uint{
					"other-user-1": 1,
					"other-user-2": 3,
				},
				UserLikedTweets: map[string]uint{},
			},
			inputResponse: &TweetsResponse{
				Data: []Tweet{
					// Replies
					{
						InReplyToUserId: "other-user-1",
						AuthorUserId:    "123",
					},
					{
						InReplyToUserId: "other-user-1",
						AuthorUserId:    "123",
					},
					{
						InReplyToUserId: "other-user-1",
						AuthorUserId:    "123",
					},
					{
						InReplyToUserId: "other-user-1",
						AuthorUserId:    "123",
					},
					{
						InReplyToUserId: "other-user-1",
						AuthorUserId:    "123",
					},
					{
						InReplyToUserId: "other-user-2",
						AuthorUserId:    "123",
					},

					// Retweets
					{
						InReplyToUserId: "",
						ReferencedTweets: []ReferencedTweetMeta{
							{
								Type: Retweet,
								Id:   "rt-original-1",
							},
							{
								Type: Retweet,
								Id:   "rt-original-2",
							},
						},
						AuthorUserId: "123",
					},
					{
						InReplyToUserId: "",
						ReferencedTweets: []ReferencedTweetMeta{
							{
								Type: Retweet,
								Id:   "rt-original-3",
							},
							{
								Type: Retweet,
								Id:   "rt-original-4",
							},
						},
						AuthorUserId: "123",
					},
				},
				Includes: TweetIncludes{
					Tweets: []Tweet{
						{
							AuthorUserId: "other-user-1",
							TweetId:      "rt-original-1",
						},
						{
							AuthorUserId: "other-user-2",
							TweetId:      "rt-original-2",
						},
						{
							AuthorUserId: "other-user-2",
							TweetId:      "rt-original-3",
						},
						{
							AuthorUserId: "other-user-2",
							TweetId:      "rt-original-4",
						},
					},
				},
			},
			inputResult: func() *UserInteractions {
				o := NewUserInteractionsObject()
				o.UserTwitterId = "123"
				return o
			}(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := NewDevAnalyzer(nil)
			a.ProcessUserInteractions(tt.userId, tt.inputResponse, tt.inputResult)

			assert.Equal(t, tt.expectResult, tt.inputResult)
		})
	}
}

func TestProcessUserLikes(t *testing.T) {
	tests := []struct {
		name          string
		userId        string
		expectResult  *UserInteractions
		inputResponse *TweetsResponse
		inputResult   *UserInteractions
	}{
		{
			name:   "ok",
			userId: "123",
			expectResult: &UserInteractions{
				UserTwitterId:        "123",
				RepliesToOtherUsers:  map[string]uint{},
				RetweetsToOtherUsers: map[string]uint{},
				UserLikedTweets: map[string]uint{
					"other-user-1": 1,
					"other-user-2": 5,
				},
			},
			inputResponse: &TweetsResponse{
				Data: []Tweet{
					{AuthorUserId: "other-user-1"},
					{AuthorUserId: "other-user-2"},
					{AuthorUserId: "other-user-2"},
					{AuthorUserId: "other-user-2"},
					{AuthorUserId: "other-user-2"},
					{AuthorUserId: "other-user-2"},
				},
			},
			inputResult: func() *UserInteractions {
				o := NewUserInteractionsObject()
				o.UserTwitterId = "123"
				return o
			}(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			a := NewDevAnalyzer(nil)
			a.ProcessUserLikes(tt.inputResponse, tt.inputResult)

			assert.Equal(t, tt.expectResult, tt.inputResult)

		})
	}
}

func TestCollectAndProcessEndpoint(t *testing.T) {
	tests := []struct {
		name               string
		inputEndpointName  string
		expectLimiterCalls func(*mocks.MockApiRateLimiter)
		inputFetchFunc     func(*testing.T) func(opts []ApiRequestOption) (*TweetsResponse, error)
		// Must return false eventually to stop the loop
		inputProcessAndContinueAssert func(*testing.T) func(*TweetsResponse) bool
	}{
		{
			name:              "ok no limiter",
			inputEndpointName: "test-endpoint",
			inputFetchFunc: func(T *testing.T) func(opts []ApiRequestOption) (*TweetsResponse, error) {
				return func(opts []ApiRequestOption) (*TweetsResponse, error) {
					assert.Len(T, opts, 0)

					return &TweetsResponse{
						Data: []Tweet{
							{
								AuthorUserId: "123",
								TweetId:      "321",
								TweetText:    "some-text",
							},
						},
					}, nil
				}
			},
			inputProcessAndContinueAssert: func(t *testing.T) func(*TweetsResponse) bool {
				return func(tr *TweetsResponse) bool {
					expected := &TweetsResponse{
						Data: []Tweet{
							{
								AuthorUserId: "123",
								TweetId:      "321",
								TweetText:    "some-text",
							},
						},
					}
					assert.Equal(t, expected, tr)

					return false
				}
			},
			expectLimiterCalls: func(marl *mocks.MockApiRateLimiter) {
				marl.EXPECT().Allow().Return(true).Times(1)
			},
		},
		{
			name:              "ok next page token appended",
			inputEndpointName: "test-endpoint",
			inputFetchFunc: func(T *testing.T) func(opts []ApiRequestOption) (*TweetsResponse, error) {
				i := 0
				return func(opts []ApiRequestOption) (*TweetsResponse, error) {

					if i == 0 {
					} else {
						assert.Len(T, opts, 1)

						// Assert that next token pagination opt is used
						opt, ok := opts[0].(*OptApplyQueryParam)
						assert.True(T, ok)
						assert.Equal(T, "next-page-token", opt.Value)
					}

					return &TweetsResponse{
						Data: []Tweet{
							{
								AuthorUserId: "123",
								TweetId:      "321",
								TweetText:    "some-text",
							},
						},
						Meta: Meta{
							NextToken: "next-page-token",
						},
					}, nil
				}
			},
			inputProcessAndContinueAssert: func(t *testing.T) func(*TweetsResponse) bool {

				i := 0
				return func(tr *TweetsResponse) bool {
					i++
					expected := &TweetsResponse{
						Data: []Tweet{
							{
								AuthorUserId: "123",
								TweetId:      "321",
								TweetText:    "some-text",
							},
						},
						Meta: Meta{
							NextToken: "next-page-token",
						},
					}
					assert.Equal(t, expected, tr)

					return i <= 1
				}
			},
			expectLimiterCalls: func(marl *mocks.MockApiRateLimiter) {
				marl.EXPECT().Allow().Return(true)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			limiter := mocks.NewMockApiRateLimiter(t)

			if tt.expectLimiterCalls != nil {
				tt.expectLimiterCalls(limiter)
			}

			a := NewDevAnalyzer(nil)
			a.CollectAndProcessEndpoint(
				tt.inputEndpointName,
				limiter,
				tt.inputFetchFunc(t),
				tt.inputProcessAndContinueAssert(t),
			)
		})
	}
}

func TestUserInteractionRanked(t *testing.T) {

	u := &UserInteractions{
		UserTwitterId: "123",
		RepliesToOtherUsers: map[string]uint{
			"other-user-1": 5,
			"other-user-2": 1,
			"other-user-3": 178,
		},
		RetweetsToOtherUsers: map[string]uint{
			"other-user-1": 678,
			"other-user-2": 34,
			"other-user-3": 2,
		},
		UserLikedTweets: map[string]uint{
			"other-user-1": 1,
			"other-user-2": 44,
			"other-user-3": 72,
		},
	}

	ids, interactions := u.Ranked()

	wantIds := []string{
		"other-user-1",
		"other-user-3",
		"other-user-2",
	}

	wantInteractions := []uint{
		684,
		252,
		79,
	}

	assert.Equal(t, wantIds, ids)
	assert.Equal(t, wantInteractions, interactions)

}
