package database

import (
	"context"
	"database/sql"
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestCreateSnapshotPrivateUniqueAndNoClobber(t *testing.T) {
	ctx := context.Background()
	db, err := Open(ctx, filepath.Join(t.TempDir(), "source.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec("INSERT INTO tenants(id,name) VALUES('snapshot','Before')"); err != nil {
		t.Fatal(err)
	}
	parent := filepath.Join(t.TempDir(), "literal#%23 parent")
	if err := os.Mkdir(parent, 0700); err != nil {
		t.Fatal(err)
	}
	sentinel := filepath.Join(parent, "metadata.db")
	if err := os.WriteFile(sentinel, []byte("sentinel"), 0600); err != nil {
		t.Fatal(err)
	}
	first, err := CreateSnapshot(ctx, db, parent)
	if err != nil {
		t.Fatal(err)
	}
	second, err := CreateSnapshot(ctx, db, parent)
	if err != nil {
		t.Fatal(err)
	}
	if !filepath.IsAbs(first) || first == second || filepath.Dir(first) == filepath.Dir(second) || filepath.Dir(filepath.Dir(first)) != parent {
		t.Fatal("snapshot destinations are not unique owned directories")
	}
	if bytes, err := os.ReadFile(sentinel); err != nil || string(bytes) != "sentinel" {
		t.Fatal("snapshot clobbered existing destination")
	}
	if runtime.GOOS != "windows" {
		for path, mode := range map[string]os.FileMode{first: 0600, filepath.Dir(first): 0700} {
			info, err := os.Stat(path)
			if err != nil || info.Mode().Perm() != mode {
				t.Fatal("snapshot permission mode is not private")
			}
		}
	}
	if _, err := db.Exec("UPDATE tenants SET name='After' WHERE id='snapshot'"); err != nil {
		t.Fatal(err)
	}
	uriPath := filepath.ToSlash(first)
	if !strings.HasPrefix(uriPath, "/") {
		uriPath = "/" + uriPath
	}
	image, err := sql.Open("sqlite", (&url.URL{Scheme: "file", Path: uriPath, RawQuery: "mode=ro"}).String())
	if err != nil {
		t.Fatal(err)
	}
	defer image.Close()
	var name string
	if err := image.QueryRow("SELECT name FROM tenants WHERE id='snapshot'").Scan(&name); err != nil || name != "Before" {
		t.Fatal("snapshot changed with source")
	}
}

func TestCreateSnapshotCancellationAfterAllocationCleansArtifact(t *testing.T) {
	db, err := Open(context.Background(), filepath.Join(t.TempDir(), "source.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	// Occupy the source's sole connection so VACUUM waits after allocating its
	// output, without a production hook or an assumed scheduling delay.
	connection, err := db.Conn(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer connection.Close()
	parent := t.TempDir()
	if err := os.WriteFile(filepath.Join(parent, "sentinel"), []byte("keep"), 0600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	type result struct {
		path string
		err  error
	}
	done := make(chan result, 1)
	go func() { path, err := CreateSnapshot(ctx, db, parent); done <- result{path, err} }()
	ticker := time.NewTicker(5 * time.Millisecond)
	defer ticker.Stop()
	timeout := time.NewTimer(5 * time.Second)
	defer timeout.Stop()
	allocated := false
	for !allocated {
		entries, err := os.ReadDir(parent)
		if err != nil {
			t.Fatal(err)
		}
		for _, entry := range entries {
			if entry.IsDir() && strings.HasPrefix(entry.Name(), "tong-metadata-") {
				if _, err := os.Stat(filepath.Join(parent, entry.Name(), "metadata.db")); err == nil {
					allocated = true
				}
			}
		}
		if allocated {
			break
		}
		select {
		case <-ticker.C:
		case <-timeout.C:
			t.Fatal("snapshot output allocation timed out")
		}
	}
	cancel()
	select {
	case result := <-done:
		if result.path != "" || !errors.Is(result.err, context.Canceled) {
			t.Fatal("allocated snapshot did not preserve cancellation")
		}
	case <-timeout.C:
		t.Fatal("snapshot cancellation did not return")
	}
	entries, err := os.ReadDir(parent)
	if err != nil || len(entries) != 1 || entries[0].Name() != "sentinel" {
		t.Fatal("canceled snapshot left artifact or removed caller file")
	}
}

func TestCreateSnapshotFailuresLeaveParentUntouched(t *testing.T) {
	for _, kind := range []string{"nil context", "nil database", "canceled", "closed", "wrong version", "foreign key"} {
		t.Run(kind, func(t *testing.T) {
			ctx := context.Background()
			db, err := Open(ctx, filepath.Join(t.TempDir(), "source.db"))
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()
			parent := t.TempDir()
			if err := os.WriteFile(filepath.Join(parent, "sentinel"), []byte("keep"), 0600); err != nil {
				t.Fatal(err)
			}
			switch kind {
			case "nil context":
				ctx = nil
			case "nil database":
				db.Close()
				db = nil
			case "canceled":
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			case "closed":
				db.Close()
			case "wrong version":
				if _, err := db.Exec("PRAGMA user_version=999"); err != nil {
					t.Fatal(err)
				}
			case "foreign key":
				if _, err := db.Exec("PRAGMA foreign_keys=OFF; INSERT INTO memberships(tenant_id,identity_id,role) VALUES('missing','missing','owner')"); err != nil {
					t.Fatal(err)
				}
			}
			path, err := CreateSnapshot(ctx, db, parent)
			if err == nil || path != "" {
				t.Fatal("invalid snapshot was published")
			}
			if kind == "canceled" && !errors.Is(err, context.Canceled) {
				t.Fatal("cancellation was not preserved")
			}
			entries, err := os.ReadDir(parent)
			if err != nil || len(entries) != 1 || entries[0].Name() != "sentinel" {
				t.Fatal("failed snapshot left an artifact or removed caller file")
			}
		})
	}
}

func TestCreateSnapshotRequiresExistingDirectory(t *testing.T) {
	db, err := Open(context.Background(), filepath.Join(t.TempDir(), "source.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	root := t.TempDir()
	file := filepath.Join(root, "file")
	if err := os.WriteFile(file, []byte("keep"), 0600); err != nil {
		t.Fatal(err)
	}
	for _, parent := range []string{"", filepath.Join(root, "missing"), file} {
		if path, err := CreateSnapshot(context.Background(), db, parent); err == nil || path != "" {
			t.Fatal("invalid snapshot parent accepted")
		}
	}
	entries, err := os.ReadDir(root)
	if err != nil || len(entries) != 1 {
		t.Fatal("snapshot created missing parent")
	}
}
