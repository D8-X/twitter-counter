package svc

import (
	"log/slog"
	"os"

	"github.com/D8-X/twitter-referral-system/src/env"
	"github.com/D8-X/twitter-referral-system/src/twitter"
	"github.com/spf13/viper"
)

func init() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		AddSource: true,
	})))
}

func RunTwitterSocialGraphService() {
	viper.SetConfigFile(".env")
	if err := viper.ReadInConfig(); err != nil {
		slog.Warn(".env file not found")
	}

	viper.AutomaticEnv()

	required := []string{
		env.TWITTER_AUTH_BEARER,
	}

	for _, e := range required {
		if !viper.IsSet(e) {
			slog.Error("missing required environment variable", slog.String("env", e))
			os.Exit(1)
		}
	}

	// Build the Twitter client
	var client twitter.Client = twitter.NewAuthBearerClient(viper.GetString(env.TWITTER_AUTH_BEARER))

	// Find user details
	userDetails, err := client.FindUserDetails([]string{"d8x_exchange"})
	if err != nil {
		slog.Error("failed to find user details", err)
	} else {
		for _, user := range userDetails.Data {
			slog.Info("user details",
				slog.String("user_id", user.Id),
				slog.String("name", user.Name),
				slog.String("username", user.Username),
			)
		}
	}

	a := twitter.NewDevAnalyzer(client)
	a.CreateUserInteractionGraph("1593204306206932993")

	// tweets, err := client.FetchUserTweets(userId, twitter.OptApplyMaxResults("5"))
	// if err != nil {
	// 	slog.Error("failed to fetch user tweets", err)
	// 	return
	// }
	// fmt.Printf("tweets raw: %+s\n\n", tweets.Raw)

	// userId := "1593204306206932993"
	// analyzer := &twitter.Analyzer{Client: client}
	// analyzer.CreateUserInteractionGraph(userId)

	// resp, err := client.FetchUserLikedTweets(userId)
	// if err != nil {
	// 	if errors.Is(err, twitter.ErrRateLimited) {
	// 		fmt.Printf("we got rate limited!!!\n\n\n")
	// 	}

	// 	slog.Error("failed to fetch user liked tweets")
	// } else {
	// 	for _, tweet := range resp.Data {
	// 		fmt.Printf("Liked tweet author and id: %s %s\n", tweet.AuthorUserId, tweet.TweetId)
	// 	}
	// }

	// for {
	// 	likers, err := client.FetchTweetLikers("1627601297083092992")
	// 	if err != nil {
	// 		log.Fatal(err)
	// 	}

	// 	fmt.Printf("Likes raw: %+s\n\n", likers.Raw)
	// 	for _, u := range likers.Data {
	// 		fmt.Printf("Liker: %s id: %s\n", u.Username, u.Id)
	// 	}
	// }

	// client.FetchTweetDetails(tweetId) // twitter.OptApplyMaxResults("5"),

	// a := &twitter.Analyzer{Client: client}
	// a.CreateUserInteractionGraph(userId)
	// if err != nil {
	// slog.Error("failed to fetch user tweets", err)
	// return
	// }

	// fmt.Printf("Tweets: %+v\n", tweets)
}
