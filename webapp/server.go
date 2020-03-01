package webapp

import (
	"context"
	"net/http"
	"os"
	"time"

	"github.com/rs/zerolog/log"
)

func StartServer(ctx context.Context) error {
	addr := os.Getenv("COVID_HTTP_BIND_ADDR")
	if len(addr) == 0 {
		addr = "0.0.0.0:8080"
	}
	router := NewRouter()
	server := &http.Server{
		Addr:              addr,
		ReadTimeout:       time.Second * 10,
		WriteTimeout:      time.Second * 10,
		ReadHeaderTimeout: time.Second * 4,
		Handler:           router,
	}
	log.Info().Str("addr", addr).Msg("Starting server")
	err := make(chan error)
	go func() { err <- server.ListenAndServe() }()

	select {
	case e := <-err:
		return e
	case <-ctx.Done():
		timeout := time.Second * 3
		log.Info().Dur("timeoutIn", timeout).Msg("Graceful shutdown")
		ctx, _ := context.WithTimeout(context.Background(), timeout)
		return server.Shutdown(ctx)
	}
}
