package auth

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/parmtech/tong-panel/internal/database"
)

func limiterFixture(t *testing.T) (string, *sql.DB, *SQLiteLoginLimiter, *time.Time) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "limiter.db")
	db, err := database.Open(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	limiter, err := NewSQLiteLoginLimiter(db, bytes.Repeat([]byte{0x77}, 32))
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 10, 28, 0, 0, 0, 0, time.UTC)
	limiter.now = func() time.Time { return now }
	return path, db, limiter, &now
}

func TestSQLiteLoginLimiterBlocksAccountAcrossSourcesAndResetsWindow(t *testing.T) {
	_, db, limiter, clock := limiterFixture(t)
	ctx := context.Background()
	for i := 0; i < maxAccountAttempts; i++ {
		ip := "192.0.2." + string(rune('1'+i))
		if allowed, err := limiter.Admit(ctx, "tenant-1", "alice", ip); !allowed || err != nil {
			t.Fatalf("attempt %d denied: %v %v", i, allowed, err)
		}
	}
	if allowed, err := limiter.Admit(ctx, "tenant-1", "ALICE", "198.51.100.1"); allowed || err != nil {
		t.Fatalf("case-folded account budget bypassed: %v %v", allowed, err)
	}
	var buckets int
	if err := db.QueryRow("SELECT count(*) FROM login_attempt_buckets").Scan(&buckets); err != nil || buckets != 6 {
		t.Fatalf("unexpected account + source bucket count %d err=%v", buckets, err)
	}
	*clock = clock.Add(loginWindow)
	if allowed, err := limiter.Admit(ctx, "tenant-1", "alice", "192.0.2.1"); !allowed || err != nil {
		t.Fatalf("expired throttle window did not reset: %v %v", allowed, err)
	}
}

func TestSQLiteLoginLimiterSourceLimitAcrossDifferentAccounts(t *testing.T) {
	_, db, limiter, _ := limiterFixture(t)
	for i := 0; i < maxSourceAttempts; i++ {
		username := "user" + string(rune('a'+i))
		if allowed, err := limiter.Admit(context.Background(), "tenant-a", username, "203.0.113.20"); !allowed || err != nil {
			t.Fatalf("source attempt %d refused: %v %v", i, allowed, err)
		}
	}
	if allowed, err := limiter.Admit(context.Background(), "tenant-a", "extra", "203.0.113.20"); allowed || err != nil {
		t.Fatalf("source limit bypassed: %v %v", allowed, err)
	}
	// No raw names or IPs are permitted in the persisted throttle records.
	var raw string
	if err := db.QueryRow("SELECT hex(bucket_sha256) FROM login_attempt_buckets LIMIT 1").Scan(&raw); err != nil ||
		len(raw) != 64 || strings.Contains(raw, "203.0.113") {
		t.Fatalf("privacy-keyed bucket unavailable or leaked peer info: %q %v", raw, err)
	}
}

func TestSQLiteLoginLimiterConcurrentConnectionsShareBudget(t *testing.T) {
	path, db, first, clock := limiterFixture(t)
	secondDB, err := database.Open(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	defer secondDB.Close()
	second, err := NewSQLiteLoginLimiter(secondDB, bytes.Repeat([]byte{0x77}, 32))
	if err != nil {
		t.Fatal(err)
	}
	second.now = func() time.Time { return *clock }
	const tries = 14
	results := make(chan struct {
		allowed bool
		err     error
	}, tries)
	var wg sync.WaitGroup
	start := make(chan struct{})
	for i := 0; i < tries; i++ {
		wg.Add(1)
		service := first
		if i%2 != 0 {
			service = second
		}
		go func(l *SQLiteLoginLimiter) {
			defer wg.Done()
			<-start
			allowed, err := l.Admit(context.Background(), "tenant-a", "victim", "192.0.2.90")
			results <- struct {
				allowed bool
				err     error
			}{allowed, err}
		}(service)
	}
	close(start)
	wg.Wait()
	close(results)
	winners := 0
	for outcome := range results {
		if outcome.err != nil {
			t.Fatalf("concurrent login throttling failed: %v", outcome.err)
		}
		if outcome.allowed {
			winners++
		}
	}
	if winners != maxAccountAttempts {
		t.Fatalf("cross-connection throttle oversubscribed: %d winners", winners)
	}
	var maxCount int
	if err := db.QueryRow("SELECT max(attempts) FROM login_attempt_buckets").Scan(&maxCount); err != nil ||
		maxCount != maxAccountAttempts {
		t.Fatalf("incorrect committed throttle count: %d %v", maxCount, err)
	}
}

func TestSQLiteLoginLimiterInvalidSourcesKeysAndClockRegression(t *testing.T) {
	_, db, limiter, clock := limiterFixture(t)
	if _, err := NewSQLiteLoginLimiter(nil, bytes.Repeat([]byte{0x45}, 32)); !errors.Is(err, ErrInvalidLoginLimiter) {
		t.Fatalf("nil limiter DB accepted: %v", err)
	}
	if _, err := NewSQLiteLoginLimiter(db, []byte("tiny")); !errors.Is(err, ErrInvalidLoginLimiter) {
		t.Fatalf("weak HMAC key accepted: %v", err)
	}
	for _, item := range []struct{ tenant, username, source string }{
		{"", "alice", "192.0.2.1"},
		{"tenant", "ab", "192.0.2.1"},
		{"tenant", "alice", "127.0.0.0.0"},
		{"tenant", "alice", "0.0.0.0"},
		{"tenant", "alice", "evil.example"},
		{"tenant", "alice", "1.2.3.4\n"},
	} {
		if ok, err := limiter.Admit(context.Background(), item.tenant, item.username, item.source); ok || !errors.Is(err, ErrInvalidLoginLimiter) {
			t.Fatalf("untrusted source/identity allowed: %+v %v %v", item, ok, err)
		}
	}
	ok, err := limiter.Admit(context.Background(), "tenant", "alice", "192.0.2.1")
	if !ok || err != nil {
		t.Fatal(err)
	}
	*clock = clock.Add(-time.Minute)
	if ok, err := limiter.Admit(context.Background(), "tenant", "alice", "192.0.2.1"); ok || err != nil {
		t.Fatalf("backward clock reset account limit: %v %v", ok, err)
	}
}
