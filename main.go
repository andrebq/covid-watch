package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"strings"
	"sync"

	"github.com/dghubble/go-twitter/twitter"
	"github.com/dghubble/oauth1"
	_ "github.com/joho/godotenv/autoload"
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

func streamItems(ctx context.Context, cli *twitter.Client, terms []string, done func()) {
	defer done()
	var stall bool
	sfp := &twitter.StreamFilterParams{
		StallWarnings: &stall,
		Track:         terms,
	}
	log.Info().Strs("searchTerms", terms).Msg("Filter configured")
	stream, err := cli.Streams.Filter(sfp)
	if err != nil {
		log.Error().Err(err).Msg("Error opening Stream.Filter")
	}
	mux := twitter.NewSwitchDemux()
	mux.Tweet = func(t *twitter.Tweet) {
		log.Info().Str("text", t.Text).Str("user", t.User.ScreenName).Send()
	}
	mux.StreamLimit = func(l *twitter.StreamLimit) {
		log.Warn().Int64("overQuota", l.Track).Send()
	}
	mux.Warning = func(w *twitter.StallWarning) {
		log.Warn().Int("percentageFull", w.PercentFull).Send()
	}
	go func() {
		<-ctx.Done()
		stream.Stop()
	}()
	mux.HandleChan(stream.Messages)
}

func main() {
	flag.Parse()
	log.Info().Msg("Connecting to Twitter")

	cli := openClient()

	log.Info().Msg("Client connected")
	terms := getTerms()

	var wg sync.WaitGroup
	ctx, cancel := context.WithCancel(context.Background())
	wg.Add(1)
	go streamItems(ctx, cli, terms, wg.Done)

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)
	<-sig
	signal.Stop(sig)
	cancel()
	log.Info().Msg("Caught signal and context closed. Waiting for stream to finish")
	wg.Wait()
}
