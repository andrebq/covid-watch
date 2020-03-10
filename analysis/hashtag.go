package analysis

import (
	"github.com/andrebq/covid-watch/collect"
	"github.com/rs/zerolog/log"
	"github.com/vmihailenco/msgpack"
)

type (
	// Hashtags takes all data from a given set of tweets
	// and computes all the hashtags and the respective count
	// of each one
	Hashtags struct {
		Tags map[string]int
	}
)

func DiscoverHashtags(pattern string) (*Hashtags, error) {
	s := &collect.Stream{Glob: pattern}
	h := &Hashtags{
		Tags: make(map[string]int),
	}
	err := s.Stream(func(p collect.Packet) (bool, error) {
		var tb *collect.TweetBatch
		// TODO: leaking from collect that we keep data using msgpack (fix this)
		err := msgpack.Unmarshal(p.Buf, &tb)
		if err != nil {
			log.Error().Str("analysis", "discoverHashtags").Err(err).Msg("Problems parsing data from file")
			return true, nil
		}
		for _, t := range tb.Items {
			for _, tag := range t.Entities.Hashtags {
				h.Tags[tag.Text] += 1
			}
		}
		return true, nil
	})
	if err != nil {
		return nil, err
	}
	return h, nil
}
