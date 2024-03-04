# Twitter API interactions generator

This is a small library which can collect and build user interactions with
Twitter API.

It fetches timeline tweets and liked tweets of provided user id  and builds a
ranking list of of twitter user ids with which the provided user id interacted
the most.

Note that due to very restrictive Twitter API rate limits, you will most likely
need to upgrade the plan.

## Usage 

Create a client with `Bearer` token from your Twitter API dashboard and set the
API plan.

```go
var client twitter.Client = twitter.NewAuthBearerClient("<YOUR_BEARER_TOKEN>", twitter.APIPlanBasic)
```

Create analyzer with client and provide the user id + referring users list of
ids you want to run analysis on

```go
analyzer := twitter.NewProductionAnalyzer(client)
result, err := analyzer.CreateUserInteractionGraph("<TWITTER_USER_ID>", "<REFERRAL_USERS_LIST>")
rankedUserIds, rankedUserValues := result.Ranked()
```

Note that you will most likely need to customize `Analyzer` rate limits depending
on your used API plan.

There is a helper method `FindUserDetails` in `twitter.Client` which you can use
to get the user ids by twitter usernames.


## Counting

idX->idY Count: corresponds to the number of occurrences counted for idY when querying `analyzer.CreateUserInteractionGraph(idX)` compared to the count
before the action took place.

| **Action**                                    | **id1->id2 Count** | **id2->id1 Count** | **id1->id3 Count** | **id3->id2 Count** | **id3->id1 Count** | **id2->id3 Count** |
|-----------------------------------------------|--------------------|--------------------|--------------------|--------------------|--------------------|--------------------|
|                        id1 likes tweet of id2 |         +1         |         +0         |         +0         |         +0         |         +0         |         +0         |
| id1 comments on id2's tweet                   |         +1         |         +0         |         +0         |         +0         |         +0         |         +0         |
| id1 retweets id2's tweet                      |         +1         |         +0         |         +0         |         +0         |         +0         |         +0         |
|      id1 likes id2's re-tweet of id3's tweet  |         +1         |         +0         |         +0         |         +0         |         +0         |         +1         |
| id1 comments on id2's re-tweet of id3's tweet |         +1         |         +0         |         +0         |         +0         |         +0         |         +1         |
| id1 retweets id2's re-tweet of id3's tweet    |         +1         |         +0         |         +0         |         +0         |         +0         |         +1         |



### How interactions are counted in `analyzer.CreateDirectUserInteractionGraph`

`analyzer.CreateDirectUserInteractionGraph` fetches and calculates interaction
between provided list of `referringUserIds` and `newUserTwitterId` by using
fine-grained search API. Search API endpoints limit results to recent tweets
only (7 days on basic plan). This method also fetches the direct likes of the
provided `newUserTwitterId` id. Returned result is `UserInteractions` struct
which can be used to get the ranked list of interactions.


### How interactions are counted in `analyzer.CreateDirectUserInteractionGraph`

`analyzer.CreateDirectUserInteractionGraph` fetches and calculates interaction from
direct user tweets and direct user's liked tweets .

For given Twitter user ID, we fetch all the timeline tweets of that ID. Timeline
tweets will include normal tweets, retweets, replies (comments) to other tweets.
Only replies and retweets matter, since from those tweets we can gather the
information about the referenced user ids that our user ID interacted with.

These tweets are processed from the provided ID side and a list of direct
interaction user ids are collected.

**Note** that higher than 1 level interactions like id1->id2->id3 (id1->id3) ar
not counted since this information is not directly available from Twitter API.

Same goes for user likes. Liked tweets are fetched and referenced user ids are
counted.

Tweets of user are checked in one-way direction. Meaning that when user's normal
tweets are fetched - we don't check for interactions with that tweet from other
users. There is functionality in this library to do that though:
`Client.FetchTweetLikers` and `Client.FetchTweetRetweeters` could be used to
gather details about other users who interacted with out user's normal tweets,
but due to restrictive rate limits it is not feasible to do this and is not
implemented in `analyzer.CreateDirectUserInteractionGraph`.

## Examples

See `cmd/main.go`
