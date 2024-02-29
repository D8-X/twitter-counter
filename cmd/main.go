package main

import (
	"fmt"
	"log"
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
	var client twitter.Client = twitter.NewAuthBearerClient(viper.GetString("TWITTER_AUTH_BEARER"), twitter.APIPlanBasic)

	// Generate the user interaction graph
	a := twitter.NewProductionAnalyzer(client)

	resp, err := a.AnalyzeNewUserInteractions(
		"339061487", // "APompliano"
		[]string{
			"21839417",            // 	"NateAFischer",
			"859484337850523648",  // 	"crypto_rand",
			"1343212349344395264", // 	"BlackLabelAdvsr",
			"1404217022137970691", // 	"SaiyanFinance",
			"1610295867617034241", // 	"limitlessio",
			"1674917913415806977", // 	"NotLarryFink",
			"1536675997180911616", // 	"YaroslavKraskoo",
		})
	if err != nil {
		log.Fatal(err)
	}
	ids, ranks := resp.Ranked()
	fmt.Printf("Rankings: %+v %+v\n", ids, ranks)

	// // d8x_exchange user id
	// result, _ := a.CreateUserInteractionGraph("1593204306206932993")

	// // Print out the ranked user ids and interaction counts
	// rankedUserIds, rankedUserValues := result.Ranked()
	// for i, userId := range rankedUserIds {
	// 	fmt.Printf("Rank #%d user id \t%s number or interactions\t%d\n", i+1, userId, rankedUserValues[i])
	// }
}
