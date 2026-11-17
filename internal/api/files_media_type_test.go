//go:build linux

package api

import (
	"crypto/tls"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/parmtech/tong-panel/internal/auth"
)

type reservationCountingQuota struct {
	*testFileQuota
	reserveCalls int
}

func (q *reservationCountingQuota) Reserve(tenantID string, bytes int64) (FileQuotaReservation, error) {
	q.reserveCalls++
	return q.testFileQuota.Reserve(tenantID, bytes)
}

func TestFilesWriteRejectsParameterizedContentTypeBeforeReservation(t *testing.T) {
	s, _, _, token, csrf := setupCookieAPI(t)
	rootA, _ := attachTestTenantFiles(t, s)
	quota := &reservationCountingQuota{testFileQuota: newTestFileQuota(64)}
	s.FileQuota = quota

	const path = "invalid-media-type.txt"
	body := &countingReader{reader: strings.NewReader("payload")}
	req := httptest.NewRequest(http.MethodPost, "https://panel.test/api/v1/files/content?path="+path, nil)
	req.Body = io.NopCloser(body)
	req.ContentLength = -1
	req.TLS = &tls.ConnectionState{}
	req.AddCookie(&http.Cookie{Name: auth.BrowserCookieName, Value: token})
	req.Header.Set("X-CSRF-Token", csrf)
	req.Header.Set("Origin", "https://panel.test")
	req.Header.Set("Content-Type", "application/octet-stream; charset=binary")
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("parameterized media type status=%d body=%s, want 415", rec.Code, rec.Body.String())
	}
	var response struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil || response.Error.Code != "binary_required" {
		t.Fatalf("media type error=%s code=%q err=%v, want binary_required", rec.Body.String(), response.Error.Code, err)
	}
	if body.read != 0 {
		t.Fatalf("rejected media type consumed %d request bytes", body.read)
	}
	if quota.reserveCalls != 0 {
		t.Fatalf("rejected media type attempted %d quota reservations", quota.reserveCalls)
	}
	if used, reserved := quota.snapshot("tenant-a"); used != 0 || reserved != 0 {
		t.Fatalf("rejected media type changed quota: used=%d reserved=%d", used, reserved)
	}
	if _, err := os.Lstat(filepath.Join(rootA, path)); !os.IsNotExist(err) {
		t.Fatalf("rejected media type created a file: %v", err)
	}
}
