package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sync"
	"time"

	"modernc.org/sqlite"
)

const sqliteBeginRetryInterval = 25 * time.Millisecond

// BeginImmediate reserves SQLite's single writer without allowing the
// connection's native busy timeout to mask caller cancellation. The caller
// owns the transaction and must call the returned idempotent cleanup after
// commit or rollback. A positive maxWait bounds only lock acquisition; when
// that internal limit expires, the last SQLite busy error is returned.
func BeginImmediate(ctx context.Context, conn *sql.Conn, maxWait time.Duration) (restore func(), err error) {
	if ctx == nil || conn == nil || maxWait < 0 {
		return nil, errors.New("invalid SQLite writer reservation arguments")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	var busyTimeout int
	// This connection-local integer read does not wait for a database lock.
	// Let its rows close before observing cancellation: modernc SQLite v1.38.2
	// can lose a prepared statement when its QueryContext is interrupted after
	// creating rows, keeping the database file open even after pool closure.
	if err := conn.QueryRowContext(context.WithoutCancel(ctx), "PRAGMA busy_timeout").Scan(&busyTimeout); err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	var restoreOnce sync.Once
	restore = func() {
		restoreOnce.Do(func() {
			_, _ = conn.ExecContext(context.Background(), fmt.Sprintf("PRAGMA busy_timeout=%d", busyTimeout))
		})
	}
	if _, err := conn.ExecContext(ctx, "PRAGMA busy_timeout=0"); err != nil {
		restore()
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, err
	}
	waitCtx := ctx
	cancel := func() {}
	if maxWait > 0 {
		waitCtx, cancel = context.WithTimeout(ctx, maxWait)
	}
	defer cancel()

	var lastBusy error
	for {
		if _, err := conn.ExecContext(waitCtx, "BEGIN IMMEDIATE"); err == nil {
			return restore, nil
		} else if !isSQLiteBusy(err) {
			restore()
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}
			if waitCtx.Err() != nil && lastBusy != nil {
				return nil, lastBusy
			}
			return nil, err
		} else {
			lastBusy = err
		}
		timer := time.NewTimer(sqliteBeginRetryInterval)
		select {
		case <-waitCtx.Done():
			timer.Stop()
			restore()
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}
			return nil, lastBusy
		case <-timer.C:
		}
	}
}

func isSQLiteBusy(err error) bool {
	var sqliteErr *sqlite.Error
	return errors.As(err, &sqliteErr) && sqliteErr.Code()&0xff == 5
}
