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
	waitCtx := ctx
	cancel := func() {}
	if maxWait > 0 {
		waitCtx, cancel = context.WithTimeout(ctx, maxWait)
	}
	defer cancel()

	var busyTimeout int
	if err := conn.QueryRowContext(waitCtx, "PRAGMA busy_timeout").Scan(&busyTimeout); err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, err
	}
	var restoreOnce sync.Once
	restore = func() {
		restoreOnce.Do(func() {
			_, _ = conn.ExecContext(context.Background(), fmt.Sprintf("PRAGMA busy_timeout=%d", busyTimeout))
		})
	}
	if _, err := conn.ExecContext(waitCtx, "PRAGMA busy_timeout=0"); err != nil {
		restore()
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, err
	}

	var lastBusy error
	for {
		if _, err := conn.ExecContext(waitCtx, "BEGIN IMMEDIATE"); err == nil {
			return restore, nil
		} else if !isSQLiteBusy(err) {
			restore()
			if ctx.Err() != nil {
				return nil, ctx.Err()
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
