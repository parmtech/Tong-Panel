// tongd is an experimental loopback-only dev API. It performs no host mutations.
package main

import (
	"context"
	"errors"
	"github.com/parmtech/tong-panel/internal/api"
	"github.com/parmtech/tong-panel/internal/auth"
	"github.com/parmtech/tong-panel/internal/database"
	"github.com/parmtech/tong-panel/internal/jobs"
	"github.com/parmtech/tong-panel/internal/rbac"
	"github.com/parmtech/tong-panel/internal/sites"
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
	stopCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := runDevelopmentDaemon(stopCtx, logger); err != nil {
		logger.Print(err)
		stop()
		os.Exit(1)
	}
}

func runDevelopmentDaemon(ctx context.Context, logger *log.Logger) error {
	if ctx.Err() != nil {
		return nil
	}
	listenAddress, err := parseDevelopmentListenAddress(os.Getenv("TONG_DEV_LISTEN_ADDR"))
	if err != nil {
		return err
	}
	metrics, err := newDevelopmentMetrics(os.Getenv("TONG_DEV_METRICS_TOKEN"), os.Getenv("TONG_DEV_TOKEN"))
	if err != nil {
		return err
	}
	sqlitePath := os.Getenv("TONG_DEV_SQLITE_DB")
	workerEnabled, err := parseDevelopmentNoopWorker(os.Getenv("TONG_DEV_RUN_NOOP_WORKER"), sqlitePath != "")
	if err != nil {
		return err
	}
	var keys *auth.Store
	token := os.Getenv("TONG_DEV_TOKEN")
	if token != "" {
		var err error
		keys, err = auth.NewDevStore(map[string]rbac.Principal{token: {UserID: "local-dev", TenantID: "dev-tenant", Role: rbac.Owner}})
		if err != nil {
			return errors.New("invalid TONG_DEV_TOKEN; expected random token >= 32 characters")
		}
	}
	dataDir := os.Getenv("TONG_DATA_DIR")
	if dataDir == "" {
		dataDir = filepath.Join(os.TempDir(), "tong-panel-dev")
	}
	// The SQL mode is opt-in for local development only. No identity, password,
	// session or default administrator is silently provisioned by this server.
	var q *jobs.Store
	var sqlJobs *jobs.SQLiteStore
	var siteCatalog *sites.Catalog
	var domainTXTLookup sites.TXTLookup
	if sqlitePath != "" {
		if keys == nil {
			return errors.New("TONG_DEV_SQLITE_DB requires TONG_DEV_TOKEN in this local-only prototype")
		}
		db, err := database.Open(ctx, sqlitePath)
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return errors.New("development SQLite store unavailable")
		}
		defer db.Close()
		sqlJobs, err = jobs.NewSQLiteStore(db)
		if err != nil {
			return errors.New("development SQLite jobs unavailable")
		}
		siteCatalog, err = sites.NewCatalog(db)
		if err != nil {
			return errors.New("development site catalog unavailable")
		}
		domainTXTLookup = sites.NetTXTLookup{}
		logger.Print("opt-in dev SQLite mode enabled; identity and membership must exist already; no provisioning performed")
	} else {
		var err error
		q, err = jobs.Open(filepath.Join(dataDir, "jobs.json"))
		if err != nil {
			return errors.New("job journal unavailable")
		}
	}
	server := &http.Server{Addr: listenAddress, Handler: withDevelopmentMetrics(publicHandler(&api.Server{Keys: keys, Jobs: q, SQLJobs: sqlJobs, SiteCatalog: siteCatalog, DomainTXTLookup: domainTXTLookup}), metrics), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 10 * time.Second, WriteTimeout: 10 * time.Second, IdleTimeout: 30 * time.Second}
	return runDevelopmentServer(ctx, server, sqlJobs, workerEnabled, logger)
}
