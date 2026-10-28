package api

import (
	"errors"
	"net/http"
	"net/url"
	"strconv"

	"github.com/parmtech/tong-panel/internal/rbac"
)

const (
	maxAPIFilesReadBytes  int64 = 16 << 20
	defaultAPIFilesLimit       = 200
	maxAPIFilesLimit            = 1000
)

var (
	ErrFilesTenantDenied = errors.New("tenant file access denied")
	ErrFilePathInvalid   = errors.New("invalid file path")
	ErrFileNotFound      = errors.New("file not found")
	ErrFileReadTooLarge  = errors.New("file exceeds read limit")
	ErrFileListTooLarge  = errors.New("directory exceeds list limit")
	ErrFilesUnavailable  = errors.New("file service unavailable")
)

// TenantFileEntry exposes only a name and non-following type/size metadata.
// It never contains an absolute server path or a client-selected tenant ID.
type TenantFileEntry struct {
	Name string `json:"name"`
	Kind string `json:"kind"`
	Size int64  `json:"size"`
}

// TenantFileService must select the tenant root using the server-authenticated
// tenant ID supplied by the handler. This interface is read-only by design.
type TenantFileService interface {
	ListForTenant(tenantID, relative string, limit int) ([]TenantFileEntry, error)
	ReadForTenant(tenantID, relative string, maxBytes int64) ([]byte, error)
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
	default:
		fail(w, http.StatusServiceUnavailable, "files_unavailable")
	}
}

