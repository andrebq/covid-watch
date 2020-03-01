package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"sync"

	"github.com/andrebq/covid-watch/collect"
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

func main() {
	flag.Parse()
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	if *debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}

	ctx := context.Background()
	ctx = watchSignal(ctx, os.Interrupt)
	var wg sync.WaitGroup
	go collect.Run(ctx, wg.Done)

	select {
	case <-ctx.Done():
		log.Error().Msg("Early cancelation")
	}
}
