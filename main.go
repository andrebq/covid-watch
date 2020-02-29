package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/dghubble/go-twitter/twitter"
	"github.com/dghubble/oauth1"
	_ "github.com/joho/godotenv/autoload"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var (
	debug = flag.Bool("debug", false, "Enable debug level")
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

func streamItems(ctx context.Context, out chan<- *twitter.Tweet, cli *twitter.Client, terms []string, done func()) {
	defer close(out)
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
		out <- t
	}
	mux.StreamLimit = func(l *twitter.StreamLimit) {
		log.Debug().Int64("overQuota", l.Track).Msg("Your query is to generic and generated more data than your account can access")
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
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	if *debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}
	log.Info().Msg("Connecting to Twitter")

	cli := openClient()

	log.Info().Msg("Client connected")
	terms := getTerms()

	var wg sync.WaitGroup
	ctx, cancel := context.WithCancel(context.Background())
	wg.Add(3)
	tweets := make(chan *twitter.Tweet, 10)
	batch := make(chan []*twitter.Tweet)
	dw := DirWriter{
		Basedir: os.Getenv("COVID_INFLIGHT_STORAGE"),
		NewBasename: func(t time.Time) string {
			return defaultBasenameFn(t) + ".json"
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
	go func() {
		err := writeOutput(&dw, batch, "\n", wg.Done)
		if err != nil {
			log.Error().Err(err).Msg("Error writing output. Canceling all operations")
			cancel()
		}
	}()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)
	select {
	case <-sig:
		signal.Stop(sig)
		cancel()
		log.Info().Msg("Caught signal and context closed. Waiting for stream to finish")
	case <-ctx.Done():
		log.Error().Msg("Early cancelation")
		signal.Stop(sig)
	}
	wg.Wait()
}
