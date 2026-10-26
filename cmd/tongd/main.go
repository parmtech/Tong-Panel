// tongd is a local-only development control-plane prototype.
// It does not manage hosting services or run privileged commands.
package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/parmtech/tong-panel/internal/health"
)

func main() {
	logger := log.New(os.Stderr, "tongd: ", log.LstdFlags)
	server := &http.Server{
		Addr:              "127.0.0.1:8090",
		Handler:           health.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       30 * time.Second,
	}

	stopCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	serverErrors := make(chan error, 1)
	go func() {
		logger.Printf("developer liveness API listening on %s (loopback only)", server.Addr)
		serverErrors <- server.ListenAndServe()
	}()

	select {
	case err := <-serverErrors:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Printf("server stopped unexpectedly: %v", err)
			os.Exit(1)
		}
	case <-stopCtx.Done():
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(ctx); err != nil {
			logger.Printf("graceful shutdown: %v", err)
			os.Exit(1)
		}
	}
}
