//go:build linux

package api

import (
	"context"
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/parmtech/tong-panel/internal/auth"
)

type blockedContextFileRemoval struct {
	TenantFileService
	ContextTenantFileService

	entered    chan struct{}
	release    chan struct{}
	tenantID   string
	relative   string
	contextErr error
}

func (f *blockedContextFileRemoval) RemoveForTenantContext(ctx context.Context, tenantID, relative string) (int64, error) {
	f.tenantID = tenantID
	f.relative = relative
	close(f.entered)
	select {
	case <-ctx.Done():
		f.contextErr = ctx.Err()
		return f.ContextTenantFileService.RemoveForTenantContext(ctx, tenantID, relative)
	case <-f.release:
		f.contextErr = ctx.Err()
		return 0, context.Canceled
	}
}

type recordingRemoveQuota struct {
	*testFileQuota
	removeCalls  int
	removeTenant string
	removeBytes  int64
}

func (q *recordingRemoveQuota) RemoveUsed(tenantID string, bytes int64) error {
	q.removeCalls++
	q.removeTenant = tenantID
	q.removeBytes = bytes
	return q.testFileQuota.RemoveUsed(tenantID, bytes)
}

// This proves the API forwards request cancellation through the context-aware
// adapter and the adapter honors an already-canceled request before removal.
// It does not inject cancellation during the root's quarantine/unlink sequence.
func TestFilesDeletePassesCancellationToContextService(t *testing.T) {
	s, _, _, token, csrf := setupCookieAPI(t)
	rootA, _ := attachTestTenantFiles(t, s)
	quota := &recordingRemoveQuota{testFileQuota: newTestFileQuota(32)}
	s.FileQuota = quota

	baseFiles := s.Files
	h := s.Handler()
	const path = "delete-after-cancel.txt"
	if rec := fileWriteCall(h, "/api/v1/files/content?path="+path, token, csrf, "delete"); rec.Code != http.StatusCreated {
		t.Fatalf("seed write status=%d body=%s", rec.Code, rec.Body.String())
	}
	usedBefore, reservedBefore := quota.snapshot("tenant-a")
	if usedBefore != int64(len("delete")) || reservedBefore != 0 {
		t.Fatalf("seed quota used=%d reserved=%d", usedBefore, reservedBefore)
	}

	contextFiles, ok := baseFiles.(ContextTenantFileService)
	if !ok {
		t.Fatal("tenant Files adapter does not implement ContextTenantFileService")
	}
	blocked := &blockedContextFileRemoval{
		TenantFileService:        baseFiles,
		ContextTenantFileService: contextFiles,
		entered:                  make(chan struct{}),
		release:                  make(chan struct{}),
	}
	s.Files = blocked
	h = s.Handler()

	ctx, cancel := context.WithCancel(context.Background())
	req := httptest.NewRequest(http.MethodDelete, "https://panel.test/api/v1/files/content?path="+path, nil).WithContext(ctx)
	req.TLS = &tls.ConnectionState{}
	req.AddCookie(&http.Cookie{Name: auth.BrowserCookieName, Value: token})
	req.Header.Set("X-CSRF-Token", csrf)
	req.Header.Set("Origin", "https://panel.test")
	rec := httptest.NewRecorder()
	done := make(chan struct{})
	go func() {
		h.ServeHTTP(rec, req)
		close(done)
	}()
	var releaseOnce sync.Once
	releaseBlocked := func() { releaseOnce.Do(func() { close(blocked.release) }) }
	defer func() {
		cancel()
		releaseBlocked()
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Errorf("DELETE handler goroutine did not stop during test cleanup")
		}
	}()

	select {
	case <-blocked.entered:
	case <-done:
		t.Fatal("DELETE finished without entering RemoveForTenantContext")
	case <-time.After(5 * time.Second):
		cancel()
		releaseBlocked()
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Fatal("DELETE did not enter RemoveForTenantContext or stop after release")
		}
		t.Fatal("DELETE did not enter RemoveForTenantContext")
	}
	if blocked.tenantID != "tenant-a" || blocked.relative != path {
		t.Fatalf("removal arguments tenant=%q path=%q, want tenant-a and %q", blocked.tenantID, blocked.relative, path)
	}

	cancel()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		releaseBlocked()
		select {
		case <-done:
			t.Errorf("DELETE handler stopped only after test release; request cancellation was not propagated")
		case <-time.After(5 * time.Second):
			t.Fatal("DELETE handler did not stop after cancellation or test release")
		}
	}
	if blocked.contextErr != context.Canceled {
		t.Fatalf("removal context error=%v, want context.Canceled", blocked.contextErr)
	}
	if rec.Body.Len() != 0 {
		t.Fatalf("canceled DELETE wrote a response: %s", rec.Body.String())
	}
	if content, err := os.ReadFile(filepath.Join(rootA, path)); err != nil || string(content) != "delete" {
		t.Fatalf("canceled DELETE changed file: content=%q err=%v", content, err)
	}
	if quota.removeCalls != 0 {
		t.Fatalf("canceled DELETE called RemoveUsed %d times for tenant=%q bytes=%d", quota.removeCalls, quota.removeTenant, quota.removeBytes)
	}
	usedAfter, reservedAfter := quota.snapshot("tenant-a")
	if usedAfter != usedBefore || reservedAfter != reservedBefore {
		t.Fatalf("canceled DELETE changed quota: before=%d/%d after=%d/%d", usedBefore, reservedBefore, usedAfter, reservedAfter)
	}
}
