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
	"sync"
	"testing"
	"time"

	"github.com/parmtech/tong-panel/internal/auth"
	"github.com/parmtech/tong-panel/internal/files"
	"github.com/parmtech/tong-panel/internal/rbac"
)

func attachTestTenantFiles(t *testing.T, s *Server) (string, string) {
	t.Helper()
	rootAPath, rootBPath := t.TempDir(), t.TempDir()
	if err := os.WriteFile(filepath.Join(rootAPath, "private.txt"), []byte("tenant-a-secret"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(rootBPath, "private.txt"), []byte("tenant-b-secret"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(rootAPath, "public"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(rootAPath, "public", "index.html"), []byte("safe"), 0600); err != nil {
		t.Fatal(err)
	}
	external := filepath.Join(t.TempDir(), "outside.txt")
	if err := os.WriteFile(external, []byte("outside-secret"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(external, filepath.Join(rootAPath, "escape")); err != nil {
		t.Fatal(err)
	}

	rootA, err := files.OpenRoot(rootAPath)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = rootA.Close() })
	rootB, err := files.OpenRoot(rootBPath)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = rootB.Close() })
	roots, err := files.NewTenantRoots(map[string]*files.Root{"tenant-a": rootA, "tenant-b": rootB})
	if err != nil {
		t.Fatal(err)
	}
	s.Files = NewTenantFileService(roots)
	return rootAPath, rootBPath
}

func sessionGET(h http.Handler, path, token string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "https://panel.test"+path, nil)
	req.TLS = &tls.ConnectionState{}
	req.AddCookie(&http.Cookie{Name: auth.BrowserCookieName, Value: token})
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

type testFileQuota struct {
	mu       sync.Mutex
	limit    int64
	used     map[string]int64
	reserved map[string]int64
}

func newTestFileQuota(limit int64) *testFileQuota {
	return &testFileQuota{limit: limit, used: make(map[string]int64), reserved: make(map[string]int64)}
}

func (q *testFileQuota) Reserve(tenantID string, bytes int64) (FileQuotaReservation, error) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.used[tenantID]+q.reserved[tenantID]+bytes > q.limit {
		return nil, ErrFileQuotaExceeded
	}
	q.reserved[tenantID] += bytes
	return &testFileQuotaReservation{quota: q, tenantID: tenantID, bytes: bytes}, nil
}

type testFileQuotaReservation struct {
	quota    *testFileQuota
	tenantID string
	bytes    int64
	done     bool
}

type uncertainWriteFiles struct {
	TenantFileService
	bytes int64
}

func (f uncertainWriteFiles) WriteNewForTenant(string, string, io.Reader, int64) (int64, error) {
	return f.bytes, ErrFilePublishedDurabilityUnconfirmed
}

type failingCommitQuota struct{ reservation *failingCommitReservation }

func (q failingCommitQuota) Reserve(string, int64) (FileQuotaReservation, error) {
	return q.reservation, nil
}

type failingCommitReservation struct {
	commitCalls  int
	releaseCalls int
}

func (r *failingCommitReservation) Commit(int64) error {
	r.commitCalls++
	return io.ErrClosedPipe
}

func (r *failingCommitReservation) Release() error {
	r.releaseCalls++
	return nil
}

func (r *testFileQuotaReservation) Commit(actualBytes int64) error {
	r.quota.mu.Lock()
	defer r.quota.mu.Unlock()
	if r.done || actualBytes < 0 || actualBytes > r.bytes {
		panic("invalid quota reservation commit")
	}
	r.quota.reserved[r.tenantID] -= r.bytes
	r.quota.used[r.tenantID] += actualBytes
	r.done = true
	return nil
}

func (r *testFileQuotaReservation) Release() error {
	r.quota.mu.Lock()
	defer r.quota.mu.Unlock()
	if r.done {
		return nil
	}
	r.quota.reserved[r.tenantID] -= r.bytes
	r.done = true
	return nil
}

func (q *testFileQuota) snapshot(tenantID string) (used, reserved int64) {
	q.mu.Lock()
	defer q.mu.Unlock()
	return q.used[tenantID], q.reserved[tenantID]
}

func fileWriteCall(h http.Handler, path, token, csrf, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "https://panel.test"+path, strings.NewReader(body))
	req.TLS = &tls.ConnectionState{}
	req.AddCookie(&http.Cookie{Name: auth.BrowserCookieName, Value: token})
	req.Header.Set("X-CSRF-Token", csrf)
	req.Header.Set("Origin", "https://panel.test")
	req.Header.Set("Content-Type", "application/octet-stream")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestFilesWriteAPIReservesQuotaBeforeCreateAndStaysTenantScoped(t *testing.T) {
	s, sessions, _, tokenA, csrfA := setupCookieAPI(t)
	rootA, rootB := attachTestTenantFiles(t, s)
	quota := newTestFileQuota(5)
	s.FileQuota = quota
	tokenB, csrfB, _, err := sessions.Issue(rbac.Principal{UserID: "u2", TenantID: "tenant-b", Role: rbac.Customer}, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	h := s.Handler()

	created := fileWriteCall(h, "/api/v1/files/content?path=created.txt", tokenA, csrfA, "12345")
	if created.Code != http.StatusCreated || !strings.Contains(created.Body.String(), `"bytes":5`) {
		t.Fatalf("write status=%d body=%s", created.Code, created.Body.String())
	}
	if data, err := os.ReadFile(filepath.Join(rootA, "created.txt")); err != nil || string(data) != "12345" {
		t.Fatalf("tenant A file=%q err=%v", data, err)
	}
	if _, err := os.Stat(filepath.Join(rootB, "created.txt")); !os.IsNotExist(err) {
		t.Fatalf("tenant B unexpectedly received tenant A file: %v", err)
	}
	if used, reserved := quota.snapshot("tenant-a"); used != 5 || reserved != 0 {
		t.Fatalf("quota after commit used=%d reserved=%d", used, reserved)
	}

	denied := fileWriteCall(h, "/api/v1/files/content?path=overflow.txt", tokenA, csrfA, "x")
	if denied.Code != http.StatusInsufficientStorage {
		t.Fatalf("quota denial status=%d body=%s", denied.Code, denied.Body.String())
	}
	if _, err := os.Stat(filepath.Join(rootA, "overflow.txt")); !os.IsNotExist(err) {
		t.Fatalf("quota-denied file was published: %v", err)
	}

	createdB := fileWriteCall(h, "/api/v1/files/content?path=created.txt", tokenB, csrfB, "b")
	if createdB.Code != http.StatusCreated {
		t.Fatalf("tenant B write status=%d body=%s", createdB.Code, createdB.Body.String())
	}
	if data, err := os.ReadFile(filepath.Join(rootB, "created.txt")); err != nil || string(data) != "b" {
		t.Fatalf("tenant B file=%q err=%v", data, err)
	}
	if used, reserved := quota.snapshot("tenant-b"); used != 1 || reserved != 0 {
		t.Fatalf("tenant B quota used=%d reserved=%d", used, reserved)
	}
}

func TestFilesWriteAPIFailureReleasesQuotaAndRejectsRolesAndPaths(t *testing.T) {
	s, sessions, _, token, csrf := setupCookieAPI(t)
	root, _ := attachTestTenantFiles(t, s)
	quota := newTestFileQuota(64)
	s.FileQuota = quota
	h := s.Handler()

	first := fileWriteCall(h, "/api/v1/files/content?path=existing.txt", token, csrf, "first")
	if first.Code != http.StatusCreated {
		t.Fatalf("initial write status=%d body=%s", first.Code, first.Body.String())
	}
	duplicate := fileWriteCall(h, "/api/v1/files/content?path=existing.txt", token, csrf, "again")
	if duplicate.Code != http.StatusConflict {
		t.Fatalf("duplicate write status=%d body=%s", duplicate.Code, duplicate.Body.String())
	}
	if used, reserved := quota.snapshot("tenant-a"); used != 5 || reserved != 0 {
		t.Fatalf("failed write quota used=%d reserved=%d", used, reserved)
	}
	for _, path := range []string{
		"/api/v1/files/content?path=../escape",
		"/api/v1/files/content?path=one&path=two",
		"/api/v1/files/content?path=good&tenant=tenant-b",
	} {
		rec := fileWriteCall(h, path, token, csrf, "blocked")
		if rec.Code != http.StatusBadRequest {
			t.Errorf("invalid path %q status=%d body=%s", path, rec.Code, rec.Body.String())
		}
	}
	auditorToken, auditorCSRF, _, err := sessions.Issue(rbac.Principal{UserID: "audit", TenantID: "tenant-a", Role: rbac.Auditor}, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if rec := fileWriteCall(h, "/api/v1/files/content?path=auditor.txt", auditorToken, auditorCSRF, "blocked"); rec.Code != http.StatusForbidden {
		t.Fatalf("auditor write status=%d body=%s", rec.Code, rec.Body.String())
	}
	if _, err := os.Stat(filepath.Join(root, "auditor.txt")); !os.IsNotExist(err) {
		t.Fatalf("auditor file was published: %v", err)
	}
}

func TestFilesWriteAPIRequiresQuotaAndCSRF(t *testing.T) {
	s, _, _, token, csrf := setupCookieAPI(t)
	attachTestTenantFiles(t, s)
	h := s.Handler()

	if rec := fileWriteCall(h, "/api/v1/files/content?path=blocked.txt", token, csrf, "data"); rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("missing quota status=%d body=%s", rec.Code, rec.Body.String())
	}
	s.FileQuota = newTestFileQuota(32)
	if rec := fileWriteCall(h, "/api/v1/files/content?path=blocked.txt", token, "wrong", "data"); rec.Code != http.StatusForbidden {
		t.Fatalf("missing CSRF status=%d body=%s", rec.Code, rec.Body.String())
	}
	unauthenticated := httptest.NewRequest(http.MethodPost, "https://panel.test/api/v1/files/content?path=blocked.txt", strings.NewReader("data"))
	unauthenticated.TLS = &tls.ConnectionState{}
	unauthenticated.Header.Set("Content-Type", "application/octet-stream")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, unauthenticated)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated write status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestFilesWriteAPIRejectsDeclaredOversizeBeforeQuotaReservation(t *testing.T) {
	s, _, _, token, csrf := setupCookieAPI(t)
	attachTestTenantFiles(t, s)
	quota := newTestFileQuota(maxAPIFilesWriteBytes * 2)
	s.FileQuota = quota
	req := httptest.NewRequest(http.MethodPost, "https://panel.test/api/v1/files/content?path=oversize.txt", strings.NewReader("small"))
	req.TLS = &tls.ConnectionState{}
	req.AddCookie(&http.Cookie{Name: auth.BrowserCookieName, Value: token})
	req.Header.Set("X-CSRF-Token", csrf)
	req.Header.Set("Origin", "https://panel.test")
	req.Header.Set("Content-Type", "application/octet-stream")
	req.ContentLength = maxAPIFilesWriteBytes + 1
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("oversize write status=%d body=%s", rec.Code, rec.Body.String())
	}
	if used, reserved := quota.snapshot("tenant-a"); used != 0 || reserved != 0 {
		t.Fatalf("oversize request reached quota: used=%d reserved=%d", used, reserved)
	}
}

func TestFilesWriteQuotaDenialDoesNotConsumeRequestBody(t *testing.T) {
	s, _, _, token, csrf := setupCookieAPI(t)
	attachTestTenantFiles(t, s)
	s.FileQuota = newTestFileQuota(0)
	body := &countingReader{reader: strings.NewReader("denied")}
	req := httptest.NewRequest(http.MethodPost, "https://panel.test/api/v1/files/content?path=denied.txt", nil)
	req.Body = io.NopCloser(body)
	req.ContentLength = -1
	req.TLS = &tls.ConnectionState{}
	req.AddCookie(&http.Cookie{Name: auth.BrowserCookieName, Value: token})
	req.Header.Set("X-CSRF-Token", csrf)
	req.Header.Set("Origin", "https://panel.test")
	req.Header.Set("Content-Type", "application/octet-stream")
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusInsufficientStorage {
		t.Fatalf("quota-denied write status=%d body=%s", rec.Code, rec.Body.String())
	}
	if body.read != 0 {
		t.Fatalf("quota denial consumed %d request bytes", body.read)
	}
}

func TestFilesWriteKeepsQuotaAccountedWhenPublishedDurabilityIsUncertain(t *testing.T) {
	s, _, _, token, csrf := setupCookieAPI(t)
	attachTestTenantFiles(t, s)
	s.Files = uncertainWriteFiles{TenantFileService: s.Files, bytes: 3}
	quota := newTestFileQuota(16)
	s.FileQuota = quota
	rec := fileWriteCall(s.Handler(), "/api/v1/files/content?path=possibly-published.txt", token, csrf, "abc")
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("uncertain durability status=%d body=%s", rec.Code, rec.Body.String())
	}
	if used, reserved := quota.snapshot("tenant-a"); used != 3 || reserved != 0 {
		t.Fatalf("uncertain publication quota used=%d reserved=%d", used, reserved)
	}
}

func TestFilesWriteDoesNotReleaseReservationWhenQuotaCommitIsUncertain(t *testing.T) {
	s, _, _, token, csrf := setupCookieAPI(t)
	attachTestTenantFiles(t, s)
	s.Files = uncertainWriteFiles{TenantFileService: s.Files, bytes: 3}
	reservation := &failingCommitReservation{}
	s.FileQuota = failingCommitQuota{reservation: reservation}
	rec := fileWriteCall(s.Handler(), "/api/v1/files/content?path=possibly-published.txt", token, csrf, "abc")
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("quota commit error status=%d body=%s", rec.Code, rec.Body.String())
	}
	if reservation.commitCalls != 1 || reservation.releaseCalls != 0 {
		t.Fatalf("quota finalization commit=%d release=%d", reservation.commitCalls, reservation.releaseCalls)
	}
}

func TestFilesWriteAPIConcurrentQuotaAndCreateOnlyPublication(t *testing.T) {
	s, _, _, token, csrf := setupCookieAPI(t)
	root, _ := attachTestTenantFiles(t, s)
	quota := newTestFileQuota(8)
	s.FileQuota = quota
	h := s.Handler()
	const writers = 8
	results := make(chan int, writers)
	var wg sync.WaitGroup
	for i := 0; i < writers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			results <- fileWriteCall(h, "/api/v1/files/content?path=race.txt", token, csrf, "12345678").Code
		}()
	}
	wg.Wait()
	close(results)
	winners, quotaDenied, conflicts := 0, 0, 0
	for status := range results {
		switch status {
		case http.StatusCreated:
			winners++
		case http.StatusInsufficientStorage:
			quotaDenied++
		case http.StatusConflict:
			conflicts++
		default:
			t.Errorf("unexpected concurrent write status %d", status)
		}
	}
	if winners != 1 || quotaDenied+conflicts != writers-1 {
		t.Fatalf("concurrent write outcomes winners=%d quota=%d conflicts=%d", winners, quotaDenied, conflicts)
	}
	if data, err := os.ReadFile(filepath.Join(root, "race.txt")); err != nil || string(data) != "12345678" {
		t.Fatalf("race winner content=%q err=%v", data, err)
	}
	if used, reserved := quota.snapshot("tenant-a"); used != 8 || reserved != 0 {
		t.Fatalf("concurrent quota used=%d reserved=%d", used, reserved)
	}
}

func TestFilesReadAPIUsesAuthenticatedTenantAndRejectsAmbiguousPaths(t *testing.T) {
	s, sessions, _, tokenA, _ := setupCookieAPI(t)
	rootA, rootB := attachTestTenantFiles(t, s)
	tokenB, _, _, err := sessions.Issue(rbac.Principal{UserID: "u3", TenantID: "tenant-b", Role: rbac.Customer}, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	h := s.Handler()

	for _, tc := range []struct{ token, want string }{{tokenA, "tenant-a-secret"}, {tokenB, "tenant-b-secret"}} {
		rec := sessionGET(h, "/api/v1/files/content?path=private.txt", tc.token)
		if rec.Code != http.StatusOK || rec.Body.String() != tc.want {
			t.Fatalf("tenant content status=%d body=%q; want %q", rec.Code, rec.Body.String(), tc.want)
		}
		if rec.Header().Get("Content-Type") != "application/octet-stream" ||
			rec.Header().Get("Cache-Control") != "no-store" ||
			rec.Header().Get("X-Content-Type-Options") != "nosniff" {
			t.Fatalf("unsafe file response headers: %#v", rec.Header())
		}
	}
	if _, err := os.Stat(filepath.Join(rootA, "private.txt")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(rootB, "private.txt")); err != nil {
		t.Fatal(err)
	}

	listing := sessionGET(h, "/api/v1/files?path=public", tokenA)
	if listing.Code != http.StatusOK {
		t.Fatalf("list status=%d body=%s", listing.Code, listing.Body.String())
	}
	var body struct {
		Entries []TenantFileEntry `json:"entries"`
	}
	if err := json.Unmarshal(listing.Body.Bytes(), &body); err != nil || len(body.Entries) != 1 || body.Entries[0].Name != "index.html" {
		t.Fatalf("listing=%s entries=%#v err=%v", listing.Body.String(), body.Entries, err)
	}

	for _, tc := range []struct {
		path string
		want int
	}{
		{"/api/v1/files/content?path=../tenant-b/private.txt", http.StatusBadRequest},
		{"/api/v1/files/content?path=private.txt&path=escape", http.StatusBadRequest},
		{"/api/v1/files/content?path=private.txt&tenant=tenant-b", http.StatusBadRequest},
		{"/api/v1/files/content?path=escape", http.StatusBadRequest},
		{"/api/v1/files/content?path=missing.txt", http.StatusNotFound},
		{"/api/v1/files?path=public&limit=0", http.StatusBadRequest},
	} {
		rec := sessionGET(h, tc.path, tokenA)
		if rec.Code != tc.want || rec.Body.String() == "tenant-b-secret" || rec.Body.String() == "outside-secret" {
			t.Errorf("file request %q returned %d %q; want %d", tc.path, rec.Code, rec.Body.String(), tc.want)
		}
	}
}

func TestFilesReadAPIDefaultClosedAuthRoleAndReadOnlyMethods(t *testing.T) {
	s, sessions, _, token, csrf := setupCookieAPI(t)
	attachTestTenantFiles(t, s)
	h := s.Handler()

	unauthenticated := httptest.NewRequest(http.MethodGet, "https://panel.test/api/v1/files", nil)
	unauthenticated.TLS = &tls.ConnectionState{}
	unauthenticatedResponse := httptest.NewRecorder()
	h.ServeHTTP(unauthenticatedResponse, unauthenticated)
	if unauthenticatedResponse.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated file list status=%d, want 401", unauthenticatedResponse.Code)
	}

	if rec := cookieCall(h, http.MethodPost, "/api/v1/files?path=write.txt", token, csrf, "https://panel.test", ""); rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("file mutation method status=%d body=%s, want 405", rec.Code, rec.Body.String())
	}
	if rec := sessionGET(h, "/api/v1/files", token); rec.Code != http.StatusOK {
		t.Fatalf("authenticated list status=%d body=%s", rec.Code, rec.Body.String())
	}

	auditorToken, _, _, err := sessions.Issue(rbac.Principal{UserID: "u4", TenantID: "tenant-a", Role: rbac.Auditor}, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if rec := sessionGET(h, "/api/v1/files", auditorToken); rec.Code != http.StatusForbidden {
		t.Fatalf("auditor file list status=%d body=%s, want 403", rec.Code, rec.Body.String())
	}

	s.Files = nil
	if rec := sessionGET(s.Handler(), "/api/v1/files", token); rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("unconfigured file service status=%d body=%s, want 503", rec.Code, rec.Body.String())
	}
}
