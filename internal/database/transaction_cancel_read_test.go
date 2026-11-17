package database

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"modernc.org/sqlite"
)

type cancelPragmaConnector struct {
	path        string
	cancel      context.CancelFunc
	readContext context.Context
	rowsClosed  bool
	writes      int
}

func (c *cancelPragmaConnector) Driver() driver.Driver { return &sqlite.Driver{} }
func (c *cancelPragmaConnector) Connect(context.Context) (driver.Conn, error) {
	conn, err := c.Driver().Open(c.path)
	if err != nil {
		return nil, err
	}
	return &cancelPragmaConn{Conn: conn, state: c}, nil
}

type cancelPragmaConn struct {
	driver.Conn
	state *cancelPragmaConnector
}

func (c *cancelPragmaConn) QueryContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	if query == "PRAGMA busy_timeout" && c.state.cancel != nil {
		c.state.readContext = ctx
		c.state.cancel()
		c.state.cancel = nil
		rows, err := c.Conn.(driver.QueryerContext).QueryContext(ctx, query, args)
		if err != nil {
			return nil, err
		}
		return &cancelPragmaRows{Rows: rows, state: c.state}, nil
	}
	return c.Conn.(driver.QueryerContext).QueryContext(ctx, query, args)
}
func (c *cancelPragmaConn) ExecContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	c.state.writes++
	return c.Conn.(driver.ExecerContext).ExecContext(ctx, query, args)
}

type cancelPragmaRows struct {
	driver.Rows
	state *cancelPragmaConnector
}

func (r *cancelPragmaRows) Close() error { r.state.rowsClosed = true; return r.Rows.Close() }

func TestBeginImmediateCancellationDuringLocalTimeoutRead(t *testing.T) {
	path := filepath.Join(t.TempDir(), "cancel-read.db")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	// Wrap the real SQLite driver to cancel precisely as the local timeout
	// query begins, without relying on scheduler timing or a cancellation sleep.
	connector := &cancelPragmaConnector{path: path + "?_pragma=busy_timeout%281234%29", cancel: cancel}
	db := sql.OpenDB(connector)
	defer db.Close()
	conn, err := db.Conn(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	restore, err := BeginImmediate(ctx, conn, 0)
	if restore != nil {
		restore()
		t.Fatal("canceled timeout read returned a transaction cleanup")
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error=%v, want context.Canceled", err)
	}
	if connector.readContext == nil || connector.readContext.Err() != nil || connector.readContext.Done() != nil {
		t.Fatal("connection-local timeout read was interrupted by caller cancellation")
	}
	if !connector.rowsClosed {
		t.Fatal("local timeout read did not close its SQLite rows")
	}
	if connector.writes != 0 {
		t.Fatalf("canceled timeout read executed %d write statements", connector.writes)
	}
	assertBusyTimeout(t, conn, 1234)
	if err := conn.Close(); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(path); err != nil {
		t.Fatalf("SQLite file remains open after close: %v", err)
	}
}
