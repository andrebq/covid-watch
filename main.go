package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"sync"

	"github.com/andrebq/covid-watch/collect"
	"github.com/andrebq/covid-watch/webapp"
	_ "github.com/joho/godotenv/autoload"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var (
	debug = flag.Bool("debug", false, "Enable debug level")
)

func watchSignal(ctx context.Context, sig os.Signal) context.Context {
	ctx, cancel := context.WithCancel(ctx)
	go func() {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, os.Interrupt)
		defer signal.Stop(sig)
		select {
		case s := <-sig:
			cancel()
			log.Info().Str("signal", s.String()).Msg("Caught signal and context closed. Waiting for stream to finish")
		case <-ctx.Done():
			break
		}
	}()
	return ctx
}

func runServer(ctx context.Context, server func(context.Context) error, serverDescription string, cancelFn func(), done func()) {
	defer done()
	err := server(ctx)
	if err != nil {
		log.Error().Err(err).Str("server", serverDescription).Send()
		cancelFn()
	}
}

func main() {
	flag.Parse()
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	if *debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}

	ctx, cancel := context.WithCancel(context.Background())
	ctx = watchSignal(ctx, os.Interrupt)
	var wg sync.WaitGroup
	wg.Add(2)
	go runServer(ctx, collect.Run, "CollectTweets", cancel, wg.Done)
	go runServer(ctx, webapp.StartServer, "WebServer", cancel, wg.Done)

	select {
	case <-ctx.Done():
		log.Error().Msg("Early cancelation")
	}
	wg.Wait()
}
