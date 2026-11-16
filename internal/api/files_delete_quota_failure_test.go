//go:build linux

package api

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"
)

type failingRemoveUsedQuota struct {
	*testFileQuota
	calls    int
	tenantID string
	bytes    int64
}

func (q *failingRemoveUsedQuota) RemoveUsed(tenantID string, bytes int64) error {
	q.calls++
	q.tenantID = tenantID
	q.bytes = bytes
	return io.ErrClosedPipe
}

func TestFilesDeleteReportsQuotaFailureAfterVisibleRemoval(t *testing.T) {
	s, _, _, token, csrf := setupCookieAPI(t)
	rootA, _ := attachTestTenantFiles(t, s)
	quota := newTestFileQuota(16)
	s.FileQuota = quota
	h := s.Handler()

	const path = "quota-failure-delete.txt"
	const content = "delete"
	if rec := fileWriteCall(h, "/api/v1/files/content?path="+path, token, csrf, content); rec.Code != http.StatusCreated {
		t.Fatalf("seed write status=%d body=%s", rec.Code, rec.Body.String())
	}
	usedBefore, reservedBefore := quota.snapshot("tenant-a")
	if usedBefore != int64(len(content)) || reservedBefore != 0 {
		t.Fatalf("seed quota used=%d reserved=%d", usedBefore, reservedBefore)
	}

	failingQuota := &failingRemoveUsedQuota{testFileQuota: quota}
	s.FileQuota = failingQuota
	rec := fileDeleteCall(h, "/api/v1/files/content?path="+path, token, csrf)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("delete status=%d body=%s, want 503", rec.Code, rec.Body.String())
	}
	var response struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil || response.Error.Code != "quota_unavailable" {
		t.Fatalf("delete error=%s code=%q err=%v, want quota_unavailable", rec.Body.String(), response.Error.Code, err)
	}
	if failingQuota.calls != 1 || failingQuota.tenantID != "tenant-a" || failingQuota.bytes != int64(len(content)) {
		t.Fatalf("quota removal calls=%d tenant=%q bytes=%d", failingQuota.calls, failingQuota.tenantID, failingQuota.bytes)
	}
	if _, err := os.Stat(filepath.Join(rootA, path)); !os.IsNotExist(err) {
		t.Fatalf("file should be visibly removed despite quota error: %v", err)
	}
	usedAfter, reservedAfter := quota.snapshot("tenant-a")
	if usedAfter != usedBefore || reservedAfter != reservedBefore {
		t.Fatalf("failed quota accounting changed usage: before=%d/%d after=%d/%d", usedBefore, reservedBefore, usedAfter, reservedAfter)
	}
}
