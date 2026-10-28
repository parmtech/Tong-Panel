package api

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strconv"

	"github.com/parmtech/tong-panel/internal/rbac"
)

const maxAPIFilesReadBytes int64 = 16 << 20
const maxAPIFilesWriteBytes int64 = 16 << 20
const defaultAPIFilesLimit = 200
const maxAPIFilesLimit = 1000

var (
	ErrFilesTenantDenied                  = errors.New("tenant file access denied")
	ErrFilePathInvalid                    = errors.New("invalid file path")
	ErrFileNotFound                       = errors.New("file not found")
	ErrFileReadTooLarge                   = errors.New("file exceeds read limit")
	ErrFileListTooLarge                   = errors.New("directory exceeds list limit")
	ErrFilesUnavailable                   = errors.New("file service unavailable")
	ErrFileQuotaExceeded                  = errors.New("tenant file quota exceeded")
	ErrFileAlreadyExists                  = errors.New("file already exists")
	ErrFilePublishedDurabilityUnconfirmed = errors.New("file was published but durability is unconfirmed")
)

// TenantFileEntry exposes only a name and non-following type/size metadata.
// It never contains an absolute server path or a client-selected tenant ID.
type TenantFileEntry struct {
	Name string `json:"name"`
	Kind string `json:"kind"`
	Size int64  `json:"size"`
}

// TenantFileService must select the tenant root using the server-authenticated
// tenant ID supplied by the handler.
type TenantFileService interface {
	ListForTenant(tenantID, relative string, limit int) ([]TenantFileEntry, error)
	ReadForTenant(tenantID, relative string, maxBytes int64) ([]byte, error)
	WriteNewForTenant(tenantID, relative string, content io.Reader, maxBytes int64) (int64, error)
	RemoveForTenant(tenantID, relative string) (int64, error)
}

// FileQuotaReservation accounts for bytes admitted for one upload. Commit must
// atomically replace the reservation with actualBytes; if it fails, the
// reservation must remain in force for later reconciliation. Release must be
// safe after failed or canceled writes and retain the reservation if it fails.
type FileQuotaReservation interface {
	Commit(actualBytes int64) error
	Release() error
}

// TenantFileQuota reserves capacity before any request body is consumed. A
// durable implementation must serialize reservations across workers and
// reconcile existing files before enabling this API.
type TenantFileQuota interface {
	Reserve(tenantID string, bytes int64) (FileQuotaReservation, error)
	RemoveUsed(tenantID string, bytes int64) error
}

// FileMutationBarrier coordinates quota-sensitive file mutations with
// cross-process maintenance. Acquire(shared) must cover reservation, disk
// mutation and quota finalization; maintenance uses exclusive mode across
// scanning and reconciliation. It is required for all mutating Files routes.
type FileMutationBarrier interface {
	Acquire(ctx context.Context, tenantID string, exclusive bool) (release func(), err error)
}

func parseTenantFileQuery(r *http.Request, allowLimit, requirePath bool) (string, int, error) {
	values, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil {
		return "", 0, ErrFilePathInvalid
	}
	for key := range values {
		if key != "path" && !(allowLimit && key == "limit") {
			return "", 0, ErrFilePathInvalid
		}
	}
	path := ""
	if pathValues, ok := values["path"]; ok {
		if len(pathValues) != 1 {
			return "", 0, ErrFilePathInvalid
		}
		path = pathValues[0]
	}
	if requirePath && path == "" {
		return "", 0, ErrFilePathInvalid
	}
	limit := defaultAPIFilesLimit
	if limitValues, ok := values["limit"]; ok {
		if len(limitValues) != 1 {
			return "", 0, ErrFilePathInvalid
		}
		limit, err = strconv.Atoi(limitValues[0])
		if err != nil || limit < 1 || limit > maxAPIFilesLimit {
			return "", 0, ErrFilePathInvalid
		}
	}
	return path, limit, nil
}

func (s *Server) fileList(w http.ResponseWriter, r *http.Request, p rbac.Principal) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		fail(w, http.StatusMethodNotAllowed, "method_not_allowed")
		return
	}
	if !p.Can("files:read", p.TenantID) {
		fail(w, http.StatusForbidden, "forbidden")
		return
	}
	if s.Files == nil {
		fail(w, http.StatusServiceUnavailable, "files_unavailable")
		return
	}
	relative, limit, err := parseTenantFileQuery(r, true, false)
	if err != nil {
		fail(w, http.StatusBadRequest, "invalid_file_request")
		return
	}
	entries, err := s.Files.ListForTenant(p.TenantID, relative, limit)
	if err != nil {
		writeFilesError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"entries": entries})
}

func (s *Server) fileContent(w http.ResponseWriter, r *http.Request, p rbac.Principal) {
	switch r.Method {
	case http.MethodGet:
		s.fileRead(w, r, p)
	case http.MethodPost:
		s.fileWrite(w, r, p)
	case http.MethodDelete:
		s.fileDelete(w, r, p)
	default:
		w.Header().Set("Allow", "GET, POST, DELETE")
		fail(w, http.StatusMethodNotAllowed, "method_not_allowed")
	}
}

func (s *Server) fileDelete(w http.ResponseWriter, r *http.Request, p rbac.Principal) {
	if !p.Can("files:write", p.TenantID) {
		fail(w, http.StatusForbidden, "forbidden")
		return
	}
	if s.Files == nil || s.FileQuota == nil || s.FileMutationBarrier == nil {
		fail(w, http.StatusServiceUnavailable, "files_unavailable")
		return
	}
	relative, _, err := parseTenantFileQuery(r, false, true)
	if err != nil {
		fail(w, http.StatusBadRequest, "invalid_file_request")
		return
	}
	release, err := s.FileMutationBarrier.Acquire(r.Context(), p.TenantID, false)
	if err != nil {
		fail(w, http.StatusServiceUnavailable, "files_unavailable")
		return
	}
	defer release()
	removed, err := s.Files.RemoveForTenant(p.TenantID, relative)
	if err != nil {
		writeFilesError(w, err)
		return
	}
	if err := s.FileQuota.RemoveUsed(p.TenantID, removed); err != nil {
		fail(w, http.StatusServiceUnavailable, "quota_unavailable")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) fileRead(w http.ResponseWriter, r *http.Request, p rbac.Principal) {
	if !p.Can("files:read", p.TenantID) {
		fail(w, http.StatusForbidden, "forbidden")
		return
	}
	if s.Files == nil {
		fail(w, http.StatusServiceUnavailable, "files_unavailable")
		return
	}
	relative, _, err := parseTenantFileQuery(r, false, true)
	if err != nil {
		fail(w, http.StatusBadRequest, "invalid_file_request")
		return
	}
	content, err := s.Files.ReadForTenant(p.TenantID, relative, maxAPIFilesReadBytes)
	if err != nil {
		writeFilesError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Length", strconv.Itoa(len(content)))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(content)
}

func (s *Server) fileWrite(w http.ResponseWriter, r *http.Request, p rbac.Principal) {
	if !p.Can("files:write", p.TenantID) {
		fail(w, http.StatusForbidden, "forbidden")
		return
	}
	if s.Files == nil || s.FileQuota == nil || s.FileMutationBarrier == nil {
		fail(w, http.StatusServiceUnavailable, "files_unavailable")
		return
	}
	relative, _, err := parseTenantFileQuery(r, false, true)
	if err != nil {
		fail(w, http.StatusBadRequest, "invalid_file_request")
		return
	}
	if r.Header.Get("Content-Type") != "application/octet-stream" {
		fail(w, http.StatusUnsupportedMediaType, "binary_required")
		return
	}
	reserveBytes := r.ContentLength
	if reserveBytes < 0 {
		reserveBytes = maxAPIFilesWriteBytes
	}
	if reserveBytes > maxAPIFilesWriteBytes {
		fail(w, http.StatusRequestEntityTooLarge, "file_limit_exceeded")
		return
	}
	release, err := s.FileMutationBarrier.Acquire(r.Context(), p.TenantID, false)
	if err != nil {
		fail(w, http.StatusServiceUnavailable, "files_unavailable")
		return
	}
	defer release()
	reservation, err := s.FileQuota.Reserve(p.TenantID, reserveBytes)
	if err != nil {
		if errors.Is(err, ErrFileQuotaExceeded) {
			fail(w, http.StatusInsufficientStorage, "file_quota_exceeded")
		} else {
			fail(w, http.StatusServiceUnavailable, "files_unavailable")
		}
		return
	}
	if reservation == nil {
		fail(w, http.StatusServiceUnavailable, "files_unavailable")
		return
	}
	committed := false
	defer func() {
		if !committed {
			_ = reservation.Release()
		}
	}()

	maxBytes := reserveBytes
	if r.ContentLength < 0 {
		maxBytes = maxAPIFilesWriteBytes
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
	written, err := s.Files.WriteNewForTenant(p.TenantID, relative, r.Body, maxBytes)
	if err != nil {
		if errors.Is(err, ErrFilePublishedDurabilityUnconfirmed) {
			committed = true
			if quotaErr := reservation.Commit(written); quotaErr != nil {
				fail(w, http.StatusServiceUnavailable, "quota_unavailable")
				return
			}
		} else {
			committed = true
			if quotaErr := reservation.Release(); quotaErr != nil {
				fail(w, http.StatusServiceUnavailable, "quota_unavailable")
				return
			}
		}
		var maxBytesErr *http.MaxBytesError
		if errors.Is(err, ErrFileReadTooLarge) || errors.As(err, &maxBytesErr) {
			fail(w, http.StatusRequestEntityTooLarge, "file_limit_exceeded")
		} else {
			writeFilesError(w, err)
		}
		return
	}
	committed = true
	if err := reservation.Commit(written); err != nil {
		fail(w, http.StatusServiceUnavailable, "quota_unavailable")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"bytes": written})
}

func writeFilesError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrFilesTenantDenied):
		fail(w, http.StatusForbidden, "forbidden")
	case errors.Is(err, ErrFilePathInvalid):
		fail(w, http.StatusBadRequest, "invalid_file_request")
	case errors.Is(err, ErrFileNotFound):
		fail(w, http.StatusNotFound, "not_found")
	case errors.Is(err, ErrFileReadTooLarge), errors.Is(err, ErrFileListTooLarge):
		fail(w, http.StatusRequestEntityTooLarge, "file_limit_exceeded")
	case errors.Is(err, ErrFileQuotaExceeded):
		fail(w, http.StatusInsufficientStorage, "file_quota_exceeded")
	case errors.Is(err, ErrFileAlreadyExists):
		fail(w, http.StatusConflict, "already_exists")
	case errors.Is(err, ErrFilePublishedDurabilityUnconfirmed):
		fail(w, http.StatusServiceUnavailable, "file_durability_unconfirmed")
	default:
		fail(w, http.StatusServiceUnavailable, "files_unavailable")
	}
}
