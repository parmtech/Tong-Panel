package main

import (
	"context"
	"database/sql"
	"errors"
	"io"
	"log"
	"net"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/parmtech/tong-panel/internal/database"
	"github.com/parmtech/tong-panel/internal/jobs"
)

func developmentWorkerQueue(t *testing.T) (*jobs.SQLiteStore, *sql.DB, string) {
	t.Helper()
	db, err := database.Open(context.Background(), filepath.Join(t.TempDir(), "panel.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	if _, err := db.Exec("INSERT INTO tenants(id,name) VALUES('tenant-a','A'); INSERT INTO identities(id,username) VALUES('user-a','alice'); INSERT INTO memberships(tenant_id,identity_id,role) VALUES('tenant-a','user-a','owner')"); err != nil {
		t.Fatal(err)
	}
	queue, err := jobs.NewSQLiteStore(db)
	if err != nil {
		t.Fatal(err)
	}
	job, err := queue.CreateFor("tenant-a", "user-a", jobs.KindNoop, "supervised-worker-0001")
	if err != nil {
		t.Fatal(err)
	}
	return queue, db, job.ID
}

func TestDevelopmentNoopWorkerOptInIsStrictAndDisabledByDefault(t *testing.T) {
	for _, tc := range []struct {
		value                    string
		sqlite, enabled, invalid bool
	}{
		{"", false, false, false}, {"", true, false, false},
		{"yes", true, true, false}, {"yes", false, false, true},
		{"true", true, false, true}, {"YES", true, false, true},
		{"secret-opt-in-canary", true, false, true},
	} {
		enabled, err := parseDevelopmentNoopWorker(tc.value, tc.sqlite)
		if enabled != tc.enabled || (err != nil) != tc.invalid {
			t.Fatal("incorrect development opt-in decision")
		}
		if err != nil && (err != errDevelopmentWorkerConfiguration || strings.Contains(err.Error(), tc.value)) {
			t.Fatal("configuration error disclosed the supplied value")
		}
	}
}

func TestDevelopmentServerBindFailureDoesNotClaimJob(t *testing.T) {
	queue, db, id := developmentWorkerQueue(t)
	occupied, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer occupied.Close()
	server := &http.Server{Addr: occupied.Addr().String()}
	if err := runDevelopmentServer(context.Background(), server, queue, true, nil); err == nil {
		t.Fatal("occupied listener was accepted")
	}
	current, err := queue.Get("tenant-a", id)
	if err != nil || current.State != jobs.Queued {
		t.Fatal("bind failure allowed worker activity")
	}
	var attempts int
	if err := db.QueryRow("SELECT count(*) FROM job_attempts").Scan(&attempts); err != nil || attempts != 0 {
		t.Fatal("bind failure created an attempt")
	}
}

type developmentAddressWriter struct{ address chan string }

func (w developmentAddressWriter) Write(data []byte) (int, error) {
	line := string(data)
	const prefix = "experimental local API listening on "
	if strings.HasPrefix(line, prefix) {
		w.address <- strings.TrimSpace(strings.TrimPrefix(line, prefix))
	}
	return len(data), nil
}

func TestDevelopmentServerDisabledWorkerAndGracefulStop(t *testing.T) {
	queue, db, id := developmentWorkerQueue(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	addresses := make(chan string, 1)
	server := &http.Server{Addr: "127.0.0.1:0", Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusNoContent) })}
	done := make(chan error, 1)
	go func() {
		done <- runDevelopmentServer(ctx, server, queue, false, log.New(developmentAddressWriter{addresses}, "", 0))
	}()
	var address string
	select {
	case address = <-addresses:
	case <-time.After(3 * time.Second):
		t.Fatal("server did not bind")
	}
	client := &http.Client{Timeout: time.Second}
	response, err := client.Get("http://" + address + "/")
	if err != nil {
		t.Fatal(err)
	}
	io.Copy(io.Discard, response.Body)
	response.Body.Close()
	client.CloseIdleConnections()
	current, err := queue.Get("tenant-a", id)
	if err != nil || current.State != jobs.Queued {
		t.Fatal("default-disabled worker ran a job")
	}
	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("server failed to join on normal shutdown")
	}
	if connection, err := net.DialTimeout("tcp4", address, time.Second); err == nil {
		connection.Close()
		t.Fatal("server still accepted connections after shutdown")
	}
	var attempts int
	if err := db.QueryRow("SELECT count(*) FROM job_attempts").Scan(&attempts); err != nil || attempts != 0 {
		t.Fatal("disabled worker created an attempt")
	}
}

func TestDevelopmentServerWorkerFailureStopsHTTPWithoutClosingCallerDB(t *testing.T) {
	queue, db, id := developmentWorkerQueue(t)
	if _, err := db.Exec("CREATE TRIGGER reject_supervised_complete BEFORE INSERT ON audit_events WHEN NEW.action='job.completed' BEGIN SELECT RAISE(ABORT,'private-audit-error'); END"); err != nil {
		t.Fatal(err)
	}
	server := &http.Server{Addr: "127.0.0.1:0"}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- runDevelopmentServer(ctx, server, queue, true, nil) }()
	select {
	case err := <-done:
		if !errors.Is(err, jobs.ErrDevelopmentNoopWorker) || strings.Contains(err.Error(), "private-audit-error") {
			t.Fatal("worker failure was not generic")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("fatal worker did not stop and join the server")
	}
	current, err := queue.Get("tenant-a", id)
	if err != nil || current.State != jobs.Running {
		t.Fatal("failed worker completed job or caller DB closed before supervision ended")
	}
}

func TestDevelopmentServerAlreadyCancelledDoesNotStartWorker(t *testing.T) {
	queue, _, id := developmentWorkerQueue(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := runDevelopmentServer(ctx, &http.Server{Addr: "127.0.0.1:0"}, queue, true, nil); err != nil {
		t.Fatal(err)
	}
	current, err := queue.Get("tenant-a", id)
	if err != nil || current.State != jobs.Queued {
		t.Fatal("cancelled server started a worker")
	}
}
