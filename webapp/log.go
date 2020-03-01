package webapp

import (
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var (
	renderLog = log.Sample(&zerolog.BurstSampler{
		Burst:       5,
		Period:      1 * time.Second,
		NextSampler: &zerolog.BasicSampler{N: 100},
	})
)
