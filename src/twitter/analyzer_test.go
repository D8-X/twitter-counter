package twitter

import (
	"testing"

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
			a.ProcessDirectUserInteractions(tt.inputResponse, tt.inputResult)

			assert.Equal(t, tt.expectResult, tt.inputResult)
		})
	}
}
