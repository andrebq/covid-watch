package collect

import (
	"context"
	"io"

	"github.com/vmihailenco/msgpack/v4"

	"github.com/dghubble/go-twitter/twitter"
	"github.com/rs/zerolog/log"
)

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

func buffered(ctx context.Context, output chan<- []*twitter.Tweet, maxBuf int, input <-chan *twitter.Tweet, done func()) {
	buf := make([]*twitter.Tweet, 0, maxBuf/2)
	var out chan<- []*twitter.Tweet
	defer close(output)
	defer done()
	for {
		select {
		case <-ctx.Done():
			if len(buf) > 0 {
				select {
				case output <- buf:
				default:
				}
			}
			return
		case out <- buf:
			buf = make([]*twitter.Tweet, 0, maxBuf/2)
			// disable output, since we just flushed
			out = nil
		case data, open := <-input:
			if !open {
				input = nil
				continue
			}
			if len(buf) == maxBuf {
				// drop head
				copy(buf[0:], buf[1:])
			}
			buf = append(buf, data)

			// enable output again
			out = output
		}
	}
}

func writeOutput(out *DirWriter, input <-chan []*twitter.Tweet, separator string, done func()) error {
	defer done()
	for batch := range input {
		// too lazy here, let's just burn memory
		for _, v := range batch {
			data, err := msgpack.Marshal(v)
			if err != nil {
				// too lazy to handle encoding errors here, skip
				continue
			}
			if len(data) == 0 {
				continue
			}
			_, err = out.Write(data)
			if err != nil {
				log.Error().Err(err).Msg("Error writing data to disk.")
				return err
			}
			_, err = io.WriteString(out, separator)
			if err != nil {
				log.Error().Err(err).Msg("Error writing data to disk.")
				return err
			}
		}
	}
	return out.Close()
}
