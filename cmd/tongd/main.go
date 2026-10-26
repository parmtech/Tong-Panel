// tongd is an experimental loopback-only dev API. It performs no host mutations.
package main

import (
	"context"
	"errors"
	"github.com/parmtech/tong-panel/internal/api"
	"github.com/parmtech/tong-panel/internal/auth"
	"github.com/parmtech/tong-panel/internal/jobs"
	"github.com/parmtech/tong-panel/internal/rbac"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"
)

func main() {
	logger := log.New(os.Stderr, "tongd: ", log.LstdFlags)
	var keys *auth.Store
	token := os.Getenv("TONG_DEV_TOKEN")
	if token != "" {
		var err error
		keys, err = auth.NewDevStore(map[string]rbac.Principal{token: {UserID: "local-dev", TenantID: "dev-tenant", Role: rbac.Owner}})
		if err != nil {
			logger.Fatal("invalid TONG_DEV_TOKEN; expected random token >= 32 characters")
		}
	}
	dataDir := os.Getenv("TONG_DATA_DIR")
	if dataDir == "" {
		dataDir = filepath.Join(os.TempDir(), "tong-panel-dev")
	}
	q, err := jobs.Open(filepath.Join(dataDir, "jobs.json"))
	if err != nil {
		logger.Fatal("job journal: ", err)
	}
	server := &http.Server{Addr: "127.0.0.1:8090", Handler: publicHandler(&api.Server{Keys: keys, Jobs: q}), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 10 * time.Second, WriteTimeout: 10 * time.Second, IdleTimeout: 30 * time.Second}
	stopCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	errs := make(chan error, 1)
	go func() {
		logger.Printf("experimental local API listening on %s", server.Addr)
		errs <- server.ListenAndServe()
	}()
	select {
	case err := <-errs:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Fatal(err)
		}
	case <-stopCtx.Done():
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(ctx); err != nil {
			logger.Fatal(err)
		}
	}
}
