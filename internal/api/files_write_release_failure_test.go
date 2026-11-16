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

type releaseFailingQuota struct {
	*testFileQuota
	reserveCalls  int
	reserveTenant string
	reserveBytes  int64
	reservation   *releaseFailingReservation
}

func (q *releaseFailingQuota) Reserve(tenantID string, bytes int64) (FileQuotaReservation, error) {
	reservation, err := q.testFileQuota.Reserve(tenantID, bytes)
	if err != nil {
		return nil, err
	}
	q.reserveCalls++
	q.reserveTenant = tenantID
	q.reserveBytes = bytes
	q.reservation = &releaseFailingReservation{FileQuotaReservation: reservation}
	return q.reservation, nil
}

type releaseFailingReservation struct {
	FileQuotaReservation
	releaseCalls int
	commitCalls  int
}

func (r *releaseFailingReservation) Commit(bytes int64) error {
	r.commitCalls++
	return r.FileQuotaReservation.Commit(bytes)
}

func (r *releaseFailingReservation) Release() error {
	r.releaseCalls++
	return io.ErrClosedPipe
}

func TestFilesWriteRetainsReservationWhenReleaseFailsAfterCreateConflict(t *testing.T) {
	s, _, _, token, csrf := setupCookieAPI(t)
	rootA, _ := attachTestTenantFiles(t, s)
	quota := newTestFileQuota(64)
	s.FileQuota = quota
	h := s.Handler()

	const path = "existing.txt"
	const original = "original"
	const upload = "new-bytes"
	if rec := fileWriteCall(h, "/api/v1/files/content?path="+path, token, csrf, original); rec.Code != http.StatusCreated {
		t.Fatalf("seed write status=%d body=%s", rec.Code, rec.Body.String())
	}
	usedBefore, reservedBefore := quota.snapshot("tenant-a")
	if usedBefore != int64(len(original)) || reservedBefore != 0 {
		t.Fatalf("seed quota used=%d reserved=%d", usedBefore, reservedBefore)
	}

	failingQuota := &releaseFailingQuota{testFileQuota: quota}
	s.FileQuota = failingQuota
	rec := fileWriteCall(h, "/api/v1/files/content?path="+path, token, csrf, upload)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("conflicting write status=%d body=%s, want 503", rec.Code, rec.Body.String())
	}
	var response struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil || response.Error.Code != "quota_unavailable" {
		t.Fatalf("write error=%s code=%q err=%v, want quota_unavailable", rec.Body.String(), response.Error.Code, err)
	}
	if failingQuota.reserveCalls != 1 || failingQuota.reserveTenant != "tenant-a" || failingQuota.reserveBytes != int64(len(upload)) {
		t.Fatalf("reserve calls=%d tenant=%q bytes=%d", failingQuota.reserveCalls, failingQuota.reserveTenant, failingQuota.reserveBytes)
	}
	if failingQuota.reservation == nil || failingQuota.reservation.releaseCalls != 1 || failingQuota.reservation.commitCalls != 0 {
		t.Fatalf("reservation=%+v, want one failed release and no commit", failingQuota.reservation)
	}
	if content, err := os.ReadFile(filepath.Join(rootA, path)); err != nil || string(content) != original {
		t.Fatalf("create conflict changed existing file: content=%q err=%v", content, err)
	}
	usedAfter, reservedAfter := quota.snapshot("tenant-a")
	if usedAfter != usedBefore || reservedAfter != int64(len(upload)) {
		t.Fatalf("uncertain release accounting: before=%d/%d after=%d/%d", usedBefore, reservedBefore, usedAfter, reservedAfter)
	}
}
