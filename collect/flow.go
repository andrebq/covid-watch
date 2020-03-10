package collect

import (
	"context"
	"time"

	"github.com/vmihailenco/msgpack/v4"

	"github.com/dghubble/go-twitter/twitter"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog/log"
)

var (
	tweetsReceived = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "tweets_received",
		Help: "How many tweets were received from Twitter",
	})

	bytesWritten = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "tweet_bytes_written",
		Help: "How many bytes were written so far",
	})

	batchSize = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "tweet_batch_size",
		Help:    "How many tweets were on a given batch",
		Buckets: []float64{1, 10, 50, 500, 1000},
	})

	writeDuration = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "tweet_write_duration_seconds",
		Help:    "How log it took to write one batch to the disk",
		Buckets: prometheus.ExponentialBuckets(0.0001, 10, 5),
	})
)

func init() {
	prometheus.MustRegister(bytesWritten)
	prometheus.MustRegister(tweetsReceived)
	prometheus.MustRegister(writeDuration)
	prometheus.MustRegister(batchSize)
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
		tweetsReceived.Add(1)
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

func writeOutput(out *DirWriter, input <-chan []*twitter.Tweet, done func()) error {
	defer done()
	writeBatch := func(batch []*twitter.Tweet) (int, error) {
		p, err := NewPacket("TweetBatch", TweetBatch{Items: batch}, msgpack.Marshal)
		if err != nil {
			return 0, err
		}
		sz, err := WritePacket(out, p, msgpack.Marshal)
		if err != nil {
			return sz, err
		}
		return sz, nil

	}
	for batch := range input {
		// too lazy here, let's just burn memory
		now := time.Now()
		sz, err := writeBatch(batch)
		bytesWritten.Add(float64(sz))
		writeDuration.Observe(time.Since(now).Seconds())
		if err != nil {
			return err
		}
	}
	return out.Close()
}
