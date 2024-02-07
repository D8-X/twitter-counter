package main

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/D8-X/twitter-counter/src/twitter"
	"github.com/spf13/viper"
)

// Example usage of twitter social graph ranking
func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		AddSource: true,
	})))

	runInteractionsAnalyzer()
}

func runInteractionsAnalyzer() {
	viper.SetConfigFile(".env")
	if err := viper.ReadInConfig(); err != nil {
		slog.Warn(".env file not found")
	}

	viper.AutomaticEnv()

	required := []string{
		"TWITTER_AUTH_BEARER",
	}

	for _, e := range required {
		if !viper.IsSet(e) {
			slog.Error("missing required environment variable", slog.String("env", e))
			os.Exit(1)
		}
	}

	// Build the Twitter client
	var client twitter.Client = twitter.NewAuthBearerClient(viper.GetString("TWITTER_AUTH_BEARER"))

	// Generate the user interaction graph
	a := twitter.NewProductionAnalyzer(client)
	// d8x_exchange user id
	result, _ := a.CreateUserInteractionGraph("1593204306206932993")

	// Print out the ranked user ids and interaction counts
	rankedUserIds, rankedUserValues := result.Ranked()
	for i, userId := range rankedUserIds {
		fmt.Printf("Rank #%d user id \t%s number or interactions\t%d\n", i+1, userId, rankedUserValues[i])
	}
}
