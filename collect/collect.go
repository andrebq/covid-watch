package collect

import (
	"context"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/dghubble/go-twitter/twitter"
	"github.com/dghubble/oauth1"
	"github.com/rs/zerolog/log"
)

func openClient() *twitter.Client {
	config := oauth1.NewConfig(os.Getenv("TWITTER_API_KEY"), os.Getenv("TWITTER_API_SECRET_KEY"))
	token := oauth1.NewToken(os.Getenv("TWITTER_ACCESS_TOKEN"), os.Getenv("TWITTER_ACCESS_TOKEN_SECRET"))
	httpClient := config.Client(oauth1.NoContext, token)

	return twitter.NewClient(httpClient)
}

func getTerms() []string {
	return strings.Split(os.Getenv("SEARCH_TERMS"), ";")
}

func Run(ctx context.Context) error {
	// TODO: rewrite to use errGroup
	log.Info().Msg("Connecting to Twitter")
	cli := openClient()
	log.Info().Msg("Client connected")
	terms := getTerms()

	var wg sync.WaitGroup
	ctx, cancel := context.WithCancel(ctx)
	wg.Add(3)
	tweets := make(chan *twitter.Tweet, 10)
	batch := make(chan []*twitter.Tweet)
	dw := DirWriter{
		Basedir: os.Getenv("COVID_INFLIGHT_STORAGE"),
		NewBasename: func(t time.Time) string {
			return defaultBasenameFn(t) + ".msgpack"
		},
		SplitAtBytes: 64 * 1000 * 1000,
	}

	if value, err := strconv.Atoi(os.Getenv("COVID_INFLIGHT_SPLIT_BYTES")); err == nil {
		dw.SplitAtBytes = value
	}
	err := dw.Init()
	if err != nil {
		panic(err)
	}
	go streamItems(ctx, tweets, cli, terms, wg.Done)
	go buffered(ctx, batch, 1000, tweets, wg.Done)
	go func(errPtr *error) {
		err := writeOutput(&dw, batch, "\n", wg.Done)
		if err != nil {
			log.Error().Err(err).Msg("Error writing output. Canceling all operations")
			cancel()
			*errPtr = err
		}
	}(&err)

	wg.Wait()
	return err
}
