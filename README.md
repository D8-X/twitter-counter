# Twitter API interactions generator

This is a small library which can collect and build user interactions with
Twitter API.

It fetches timeline tweets and liked tweets of provided user id  and builds a
ranking list of of twitter user ids with which the provided user id interacted
the most.

Note that due to very restrictive Twitter API rate limits, you will most likely
need to upgrade the plan.

## Usage 

Create a client with `Bearer` token from your Twitter API dashboard.

```go
var client twitter.Client = twitter.NewAuthBearerClient("<YOUR_BEARER_TOKEN>")
```

Create analyzer with client and provide the user id you want to run analysis on

```go
analyzer := twitter.NewProductionAnalyzer(client)
result, err := analyzer.CreateUserInteractionGraph("<TWITTER_USER_ID>")
rankedUserIds, rankedUserValues := result.Ranked()
```

Note that you will most likely need to customize `Analyzer` rate limits depending
on your used API plan.

There is a helper method `FindUserDetails` in `twitter.Client` which you can use
to get the user ids by twitter usernames.

## Counting

idX->idY Count: corresponds to the number of occurrences counted for idY when querying `analyzer.CreateUserInteractionGraph(idX)` compared to the count
before the action took place.

TODO:
| **Action**                                    | **id1->id2 Count** | **id2->id1 Count** | **id1->id3 Count** | **id3->id2 Count** | **id3->id1 Count** |
|-----------------------------------------------|--------------------|--------------------|--------------------|--------------------|--------------------|
|                        id1 likes tweet of id2 |         +1         |         +0         |                    |          -         |          -         |
| id1 comments on id2's tweet                   |         +1         |         +1         |                    |          -         |          -         |
| id1 retweets id2's tweet                      |         +1         |         +1         |                    |          -         |          -         |
|      id1 likes id2's re-tweet of id3's tweet  |         +1         |                    |                    |         +0         |         +0         |
| id1 comments on id2's re-tweet of id3's tweet |                    |                    |                    |                    |                    |
| id1 retweets id2's re-tweet of id3's tweet    |                    |                    |                    |                    |                    |

## Examples

See `cmd/main.go`
