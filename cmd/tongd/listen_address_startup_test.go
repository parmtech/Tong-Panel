package main

import (
	"context"
	"errors"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/parmtech/tong-panel/internal/jobs"
)

func TestDevelopmentInvalidListenAddressFailsBeforeOpeningState(t *testing.T) {
	for _, sqlite := range []bool{false, true} {
		t.Run(map[bool]string{false: "journal", true: "SQLite"}[sqlite], func(t *testing.T) {
			root := t.TempDir()
			dataDir := filepath.Join(root, "must-not-create-journal")
			database := filepath.Join(root, "must-not-create-sqlite", "panel.db")
			t.Setenv("TONG_DATA_DIR", dataDir)
			t.Setenv("TONG_DEV_LISTEN_ADDR", "secret-bind-canary:8090")
			t.Setenv("TONG_DEV_TOKEN", "long-random-looking-fixture-token-for-bind-test")
			t.Setenv("TONG_DEV_SQLITE_DB", "")
			t.Setenv("TONG_DEV_RUN_NOOP_WORKER", "")
			if sqlite {
				t.Setenv("TONG_DEV_SQLITE_DB", database)
				t.Setenv("TONG_DEV_RUN_NOOP_WORKER", "yes")
			}
			if err := runDevelopmentDaemon(context.Background(), log.New(io.Discard, "", 0)); !errors.Is(err, errDevelopmentListenAddress) {
				t.Fatal("invalid listener configuration was not rejected first")
			}
			entries, err := os.ReadDir(root)
			if err != nil || len(entries) != 0 {
				t.Fatal("invalid bind configuration created state files or directories")
			}
		})
	}
}

func TestDevelopmentServerRejectsInvalidExposureBeforeWorker(t *testing.T) {
	queue, db, id := developmentWorkerQueue(t)
	for _, address := range []string{":8090", "0.0.0.0:8090", "192.0.2.1:8090", "[::ffff:127.0.0.1]:8090"} {
		if err := runDevelopmentServer(context.Background(), &http.Server{Addr: address}, queue, true, nil); !errors.Is(err, errDevelopmentListenAddress) {
			t.Fatal("invalid exposure was not rejected before worker startup")
		}
	}
	job, err := queue.Get("tenant-a", id)
	if err != nil || job.State != jobs.Queued {
		t.Fatal("invalid listener configuration allowed worker activity")
	}
	var attempts int
	if err := db.QueryRow("SELECT count(*) FROM job_attempts").Scan(&attempts); err != nil || attempts != 0 {
		t.Fatal("invalid listener configuration recorded a worker attempt")
	}
}
