package database

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestOpenPreservesLiteralFilename(t *testing.T) {
	names := []string{"literal#fragment.db", "literal%23hash.db", "literal%3Fquery.db", "literal%00nul.db", "space and + plus.db"}
	if runtime.GOOS != "windows" {
		names = append(names, "literal?query.db", "literal?_pragma=foreign_keys(0).db")
	}
	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			path := filepath.Join(root, name)
			db, err := Open(context.Background(), path)
			if err != nil {
				t.Fatalf("literal filename rejected: %v", err)
			}
			defer db.Close()
			var sequence int
			var schema, opened string
			if err := db.QueryRow("PRAGMA database_list").Scan(&sequence, &schema, &opened); err != nil {
				t.Fatal(err)
			}
			if filepath.Clean(opened) != filepath.Clean(path) {
				t.Fatal("SQLite opened a different literal path")
			}
			for pragma, want := range map[string]int{"foreign_keys": 1, "synchronous": 2, "busy_timeout": 5000, "user_version": SchemaVersion} {
				var got int
				if err := db.QueryRow("PRAGMA " + pragma).Scan(&got); err != nil || got != want {
					t.Fatalf("required pragma %s changed", pragma)
				}
			}
			var journal string
			if err := db.QueryRow("PRAGMA journal_mode").Scan(&journal); err != nil || journal != "wal" {
				t.Fatal("WAL was not enabled")
			}
			if _, err := db.Exec("INSERT INTO tenants(id,name) VALUES('literal-path','Persisted fixture')"); err != nil {
				t.Fatal(err)
			}
			if err := db.Close(); err != nil {
				t.Fatal(err)
			}
			if _, err := os.Stat(path); err != nil {
				t.Fatal("intended database file was not created")
			}
			entries, err := os.ReadDir(root)
			if err != nil || len(entries) != 1 || entries[0].Name() != name {
				t.Fatal("unexpected alternate database file created")
			}
			db, err = Open(context.Background(), path)
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()
			var persisted string
			if err := db.QueryRow("SELECT name FROM tenants WHERE id='literal-path'").Scan(&persisted); err != nil || persisted != "Persisted fixture" {
				t.Fatal("literal path did not preserve data on reopen")
			}
		})
	}
}
