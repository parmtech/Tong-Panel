//go:build linux

package api

import (
	"context"
	"crypto/tls"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/parmtech/tong-panel/internal/auth"
)

// Check the actual on-disk boundary at each existing cancellation check, rather
// than depending on call counts or scheduling. Embedding a real cancelable
// context keeps Done and Err consistent once the callback cancels it.
type quarantineCancellationContext struct {
	context.Context
	mu    sync.Mutex
	check func()
}

func (c *quarantineCancellationContext) Err() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.Context.Err() == nil {
		c.check()
	}
	return c.Context.Err()
}

type quarantineCancellationRemoval struct {
	TenantFileService
	ContextTenantFileService
	check   func()
	called  bool
	removed int64
	err     error
}

func (f *quarantineCancellationRemoval) RemoveForTenantContext(ctx context.Context, tenantID, relative string) (int64, error) {
	f.called = true
	f.removed, f.err = f.ContextTenantFileService.RemoveForTenantContext(&quarantineCancellationContext{Context: ctx, check: f.check}, tenantID, relative)
	return f.removed, f.err
}

func TestFilesDeleteCancellationAfterQuarantineRestoresEntryAndQuota(t *testing.T) {
	s, _, _, token, csrf := setupCookieAPI(t)
	root, _ := attachTestTenantFiles(t, s)
	quota := &recordingRemoveQuota{testFileQuota: newTestFileQuota(32)}
	s.FileQuota = quota
	const name, content = "keep-after-quarantine.txt", "keep these bytes"
	if rec := fileWriteCall(s.Handler(), "/api/v1/files/content?path="+name, token, csrf, content); rec.Code != http.StatusCreated {
		t.Fatalf("seed write status=%d body=%s", rec.Code, rec.Body.String())
	}
	target := filepath.Join(root, name)
	original, err := os.Lstat(target)
	if err != nil {
		t.Fatal(err)
	}
	usedBefore, reservedBefore := quota.snapshot("tenant-a")
	if usedBefore != int64(len(content)) || reservedBefore != 0 {
		t.Fatalf("seed quota used=%d reserved=%d", usedBefore, reservedBefore)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	quarantined := false
	contextFiles, ok := s.Files.(ContextTenantFileService)
	if !ok {
		t.Fatal("tenant Files adapter does not implement ContextTenantFileService")
	}
	removal := &quarantineCancellationRemoval{TenantFileService: s.Files, ContextTenantFileService: contextFiles}
	removal.check = func() {
		if _, err := os.Lstat(target); err == nil {
			return // The pre-quarantine checks must leave the context active.
		} else if !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("inspect original path: %v", err)
		}
		entries, err := os.ReadDir(root)
		if err != nil {
			t.Fatal(err)
		}
		for _, entry := range entries {
			if !strings.HasPrefix(entry.Name(), ".tong-stage-") {
				continue
			}
			stage := filepath.Join(root, entry.Name())
			staged, err := os.Lstat(filepath.Join(stage, "entry"))
			if err != nil || !os.SameFile(original, staged) {
				t.Fatalf("quarantined entry is not original inode: stat=%v err=%v", staged, err)
			}
			if data, err := os.ReadFile(filepath.Join(stage, "entry")); err != nil || string(data) != content {
				t.Fatalf("quarantined content=%q err=%v", data, err)
			}
			quarantined = true
			cancel()
			return
		}
		t.Fatal("original path disappeared without a quarantined entry")
	}
	s.Files = removal
	req := httptest.NewRequest(http.MethodDelete, "https://panel.test/api/v1/files/content?path="+name, nil).WithContext(ctx)
	req.TLS = &tls.ConnectionState{}
	req.AddCookie(&http.Cookie{Name: auth.BrowserCookieName, Value: token})
	req.Header.Set("X-CSRF-Token", csrf)
	req.Header.Set("Origin", "https://panel.test")
	rec := httptest.NewRecorder()
	response := &trackingResponseWriter{ResponseWriter: rec}
	s.Handler().ServeHTTP(response, req)

	if !removal.called || !quarantined || !errors.Is(removal.err, context.Canceled) || removal.removed != 0 {
		t.Fatalf("removal called=%t quarantined=%t bytes=%d err=%v", removal.called, quarantined, removal.removed, removal.err)
	}
	restored, err := os.Lstat(target)
	if err != nil || !os.SameFile(original, restored) {
		t.Fatalf("restored entry is not original inode: stat=%v err=%v", restored, err)
	}
	if data, err := os.ReadFile(target); err != nil || string(data) != content {
		t.Fatalf("restored content=%q err=%v", data, err)
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".tong-stage-") {
			t.Fatalf("canceled deletion left private stage %q", entry.Name())
		}
	}
	if quota.removeCalls != 0 {
		t.Fatalf("canceled deletion called RemoveUsed %d times", quota.removeCalls)
	}
	if used, reserved := quota.snapshot("tenant-a"); used != usedBefore || reserved != reservedBefore {
		t.Fatalf("quota changed from %d/%d to %d/%d", usedBefore, reservedBefore, used, reserved)
	}
	// Recorder.Code defaults to 200 even when the handler writes nothing; check
	// actual writes so an implicit recorder status cannot masquerade as success.
	if response.wroteHeader || response.wroteBody || rec.Body.Len() != 0 {
		t.Fatalf("canceled deletion wrote HTTP response: header=%t body=%t status=%d body=%q", response.wroteHeader, response.wroteBody, rec.Code, rec.Body.String())
	}
}
