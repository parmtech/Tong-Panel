package main

import (
	"context"
	"errors"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/parmtech/tong-panel/internal/jobs"
)

var errDevelopmentWorkerConfiguration = errors.New("invalid development no-op worker configuration")

func parseDevelopmentNoopWorker(value string, sqliteMode bool) (bool, error) {
	if value == "" {
		return false, nil
	}
	if value != "yes" || !sqliteMode {
		return false, errDevelopmentWorkerConfiguration
	}
	return true, nil
}

// runDevelopmentServer binds before launching any worker. It joins both
// goroutines before returning so its caller can safely close the SQLite DB.
func runDevelopmentServer(ctx context.Context, server *http.Server, queue *jobs.SQLiteStore, enabled bool, logger *log.Logger) error {
	if server == nil || ctx == nil || (enabled && queue == nil) {
		return errDevelopmentWorkerConfiguration
	}
	if ctx.Err() != nil {
		return nil
	}
	listenAddress, err := validateDevelopmentListenAddress(server.Addr, true)
	if err != nil {
		return err
	}
	listener, err := net.Listen("tcp4", listenAddress)
	if err != nil {
		return errors.New("development listener unavailable")
	}
	workerContext, stopWorker := context.WithCancel(ctx)
	defer stopWorker()
	serveDone := make(chan error, 1)
	go func() { serveDone <- server.Serve(listener) }()
	if logger != nil {
		logger.Printf("experimental local API listening on %s", listener.Addr())
	}
	var workerDone chan error
	if enabled {
		workerDone = make(chan error, 1)
		go func() { workerDone <- queue.RunDevelopmentNoopWorker(workerContext) }()
		if logger != nil {
			logger.Print("opt-in development no-op worker enabled; metadata transitions only")
		}
	}
	var result error
	serveReceived, workerReceived := false, false
	select {
	case <-ctx.Done():
	case err := <-serveDone:
		serveReceived = true
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			result = errors.New("development HTTP server failed")
		}
	case err := <-workerDone:
		workerReceived = true
		if err != nil {
			result = jobs.ErrDevelopmentNoopWorker
		} else if ctx.Err() == nil {
			result = errors.New("development no-op worker stopped unexpectedly")
		}
	}
	stopWorker()
	shutdownContext, stopShutdown := context.WithTimeout(context.Background(), 5*time.Second)
	shutdownErr := server.Shutdown(shutdownContext)
	stopShutdown()
	if shutdownErr != nil {
		_ = server.Close()
		if result == nil {
			result = errors.New("development HTTP shutdown failed")
		}
	}
	if !serveReceived {
		<-serveDone
	}
	if enabled && !workerReceived {
		<-workerDone
	}
	return result
}
