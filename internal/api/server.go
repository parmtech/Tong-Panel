// Package api exposes an intentionally local, limited P1 developer API.
// Static bearer tokens are development-only: no production login is implemented.
package api

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/parmtech/tong-panel/internal/auth"
	"github.com/parmtech/tong-panel/internal/jobs"
	"github.com/parmtech/tong-panel/internal/rbac"
)

type Server struct {
	Keys *auth.Store
	Jobs *jobs.Store
}
type failure struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
func fail(w http.ResponseWriter, status int, code string) {
	var e failure
	e.Error.Code = code
	e.Error.Message = http.StatusText(status)
	writeJSON(w, status, e)
}
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			w.Header().Set("Allow", "GET")
			fail(w, 405, "method_not_allowed")
			return
		}
		writeJSON(w, 200, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("/api/v1/version", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			w.Header().Set("Allow", "GET")
			fail(w, 405, "method_not_allowed")
			return
		}
		writeJSON(w, 200, map[string]string{"product": "Tong Panel", "stage": "experimental-core", "api": "v1"})
	})
	mux.HandleFunc("/api/v1/me", s.protected(s.me))
	mux.HandleFunc("/api/v1/jobs", s.protected(s.jobListCreate))
	mux.HandleFunc("/api/v1/jobs/", s.protected(s.jobGet))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var raw [8]byte
		_, _ = rand.Read(raw[:])
		w.Header().Set("X-Request-ID", hex.EncodeToString(raw[:]))
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")
		mux.ServeHTTP(w, r)
	})
}

type secured func(http.ResponseWriter, *http.Request, rbac.Principal)

func (s *Server) protected(h secured) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if s.Keys == nil {
			fail(w, 503, "auth_unconfigured")
			return
		}
		p, ok := s.Keys.Check(r.Header.Get("Authorization"))
		if !ok {
			w.Header().Set("WWW-Authenticate", "Bearer")
			fail(w, 401, "unauthorized")
			return
		}
		h(w, r, p)
	}
}
func (s *Server) me(w http.ResponseWriter, r *http.Request, p rbac.Principal) {
	if r.Method != "GET" {
		fail(w, 405, "method_not_allowed")
		return
	}
	writeJSON(w, 200, p)
}
func (s *Server) jobListCreate(w http.ResponseWriter, r *http.Request, p rbac.Principal) {
	if s.Jobs == nil {
		fail(w, 503, "jobs_unconfigured")
		return
	}
	switch r.Method {
	case "GET":
		if !p.Can("jobs:read", p.TenantID) {
			fail(w, 403, "forbidden")
			return
		}
		writeJSON(w, 200, map[string]any{"jobs": s.Jobs.List(p.TenantID)})
	case "POST":
		if !p.Can("jobs:create", p.TenantID) {
			fail(w, 403, "forbidden")
			return
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			fail(w, 415, "json_required")
			return
		}
		key := r.Header.Get("Idempotency-Key")
		if key == "" {
			fail(w, 400, "idempotency_required")
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, 2048)
		var req struct {
			Kind string `json:"kind"`
		}
		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&req); err != nil {
			fail(w, 400, "invalid_request")
			return
		}
		if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
			fail(w, 400, "invalid_request")
			return
		}
		j, err := s.Jobs.Create(p.TenantID, req.Kind, key)
		if err != nil {
			fail(w, 400, "invalid_job")
			return
		}
		writeJSON(w, 202, j)
	default:
		w.Header().Set("Allow", "GET, POST")
		fail(w, 405, "method_not_allowed")
	}
}
func (s *Server) jobGet(w http.ResponseWriter, r *http.Request, p rbac.Principal) {
	if r.Method != "GET" {
		w.Header().Set("Allow", "GET")
		fail(w, 405, "method_not_allowed")
		return
	}
	if !p.Can("jobs:read", p.TenantID) {
		fail(w, 403, "forbidden")
		return
	}
	if s.Jobs == nil {
		fail(w, 503, "jobs_unconfigured")
		return
	}
	id := strings.TrimPrefix(r.URL.Path, "/api/v1/jobs/")
	if len(id) != 32 || strings.ContainsAny(id, "/\\") {
		fail(w, 404, "not_found")
		return
	}
	j, err := s.Jobs.Get(p.TenantID, id)
	if err != nil {
		fail(w, 404, "not_found")
		return
	}
	writeJSON(w, 200, j)
}
