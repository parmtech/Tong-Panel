package api

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/parmtech/tong-panel/internal/auth"
	"github.com/parmtech/tong-panel/internal/rbac"
	"github.com/parmtech/tong-panel/internal/sites"
)

const maxSiteRequestBytes = 16 << 10

func (s *Server) siteCollection(w http.ResponseWriter, r *http.Request, p rbac.Principal) {
	if r.URL.Path != "/api/v1/sites" || (r.Method != http.MethodGet && r.URL.RawQuery != "") {
		fail(w, http.StatusNotFound, "not_found")
		return
	}
	if r.Method == http.MethodGet {
		s.siteList(w, r, p)
		return
	}
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "GET, POST")
		fail(w, http.StatusMethodNotAllowed, "method_not_allowed")
		return
	}
	if !p.Can("sites:write", p.TenantID) {
		fail(w, http.StatusForbidden, "forbidden")
		return
	}
	if s.SiteCatalog == nil {
		fail(w, http.StatusServiceUnavailable, "sites_unavailable")
		return
	}
	if r.Header.Get("Content-Type") != "application/json" {
		fail(w, http.StatusUnsupportedMediaType, "json_required")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxSiteRequestBytes)
	var req struct {
		SiteID        string   `json:"site_id"`
		PrimaryDomain string   `json:"primary_domain"`
		Aliases       []string `json:"aliases"`
	}
	if err := decodeSiteJSON(r.Body, &req); err != nil {
		fail(w, http.StatusBadRequest, "invalid_site_request")
		return
	}
	spec := sites.SiteSpec{
		TenantID: p.TenantID, SiteID: req.SiteID, PrimaryDomain: req.PrimaryDomain,
		Aliases: req.Aliases, DocumentRoot: "sites/" + req.SiteID + "/public",
	}
	var proof *auth.SessionReadProof
	if value, ok := r.Context().Value(sessionProofKey{}).(auth.SessionReadProof); ok {
		proof = &value
	}
	created, err := s.SiteCatalog.CreateFor(r.Context(), spec, p.UserID, proof)
	if err != nil {
		writeSiteError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

// siteList always returns a bounded tenant-scoped page. Cursors are ordering
// boundaries and never select a tenant or confer access.
func (s *Server) siteList(w http.ResponseWriter, r *http.Request, p rbac.Principal) {
	if !p.Can("sites:read", p.TenantID) {
		fail(w, http.StatusForbidden, "forbidden")
		return
	}
	if s.SiteCatalog == nil {
		fail(w, http.StatusServiceUnavailable, "sites_unavailable")
		return
	}
	values, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil {
		fail(w, http.StatusBadRequest, "invalid_page")
		return
	}
	for key, entries := range values {
		if (key != "limit" && key != "after") || len(entries) != 1 || entries[0] == "" {
			fail(w, http.StatusBadRequest, "invalid_page")
			return
		}
	}
	limit := 50
	if v, ok := values["limit"]; ok {
		for _, digit := range v[0] {
			if digit < '0' || digit > '9' {
				fail(w, http.StatusBadRequest, "invalid_page")
				return
			}
		}
		limit, err = strconv.Atoi(v[0])
		if err != nil || limit < 1 || limit > 50 {
			fail(w, http.StatusBadRequest, "invalid_page")
			return
		}
	}
	after := values.Get("after")
	var items []sites.SiteSummary
	var next string
	if proof, ok := r.Context().Value(sessionProofKey{}).(auth.SessionReadProof); ok {
		items, next, err = s.SiteCatalog.ListPageForSession(r.Context(), p.TenantID, p.UserID, after, limit, proof)
	} else {
		items, next, err = s.SiteCatalog.ListPageFor(r.Context(), p.TenantID, p.UserID, after, limit)
	}
	if errors.Is(err, sites.ErrInvalidPage) {
		fail(w, http.StatusBadRequest, "invalid_page")
		return
	}
	if errors.Is(err, sites.ErrSiteReadDenied) {
		fail(w, http.StatusForbidden, "forbidden")
		return
	}
	if errors.Is(err, sites.ErrSiteSessionDenied) {
		fail(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if err != nil {
		writeSiteError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"sites": items, "next_after": next})
}

func decodeSiteJSON(body io.Reader, dst any) error {
	decoder := json.NewDecoder(body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return errors.New("request must contain exactly one JSON value")
	}
	return nil
}

func (s *Server) siteResource(w http.ResponseWriter, r *http.Request, p rbac.Principal) {
	if r.URL.RawQuery != "" {
		fail(w, http.StatusBadRequest, "invalid_site_request")
		return
	}
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/v1/sites/"), "/")
	if len(parts) == 1 && parts[0] != "" {
		s.siteGet(w, r, p, parts[0])
		return
	}
	if len(parts) == 3 && parts[0] != "" && parts[1] == "preview" && parts[2] == "nginx" {
		s.siteNginxPreview(w, r, p, parts[0])
		return
	}
	if len(parts) == 2 && parts[0] != "" && parts[1] == "domains" {
		s.siteDomainAdd(w, r, p, parts[0])
		return
	}
	if len(parts) == 2 && parts[0] != "" && parts[1] == "databases" {
		s.siteDatabaseCollection(w, r, p, parts[0])
		return
	}
	if len(parts) == 3 && parts[0] != "" && parts[1] == "databases" && parts[2] != "" {
		s.siteDatabaseResource(w, r, p, parts[0], parts[2])
		return
	}
	if len(parts) == 3 && parts[0] != "" && parts[1] == "domains" && parts[2] != "" {
		s.siteDomainRemove(w, r, p, parts[0], parts[2])
		return
	}
	if len(parts) == 4 && parts[0] != "" && parts[1] == "domains" && parts[2] != "" && parts[3] == "verification" {
		s.siteDomainVerificationBegin(w, r, p, parts[0], parts[2])
		return
	}
	if len(parts) == 5 && parts[0] != "" && parts[1] == "domains" && parts[2] != "" &&
		parts[3] == "verification" && parts[4] == "complete" {
		s.siteDomainVerificationComplete(w, r, p, parts[0], parts[2])
		return
	}
	fail(w, http.StatusNotFound, "not_found")
}

func (s *Server) siteDatabaseCollection(w http.ResponseWriter, r *http.Request, p rbac.Principal, siteID string) {
	if r.URL.RawQuery != "" {
		fail(w, http.StatusBadRequest, "invalid_database_request")
		return
	}
	switch r.Method {
	case http.MethodGet:
		if !p.Can("sites:read", p.TenantID) {
			fail(w, http.StatusForbidden, "forbidden")
			return
		}
		if s.SiteCatalog == nil {
			fail(w, http.StatusServiceUnavailable, "sites_unavailable")
			return
		}
		records, err := s.SiteCatalog.ListDatabases(r.Context(), p.TenantID, siteID)
		if err != nil {
			writeSiteError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"databases": records})
	case http.MethodPost:
		if !p.Can("sites:write", p.TenantID) {
			fail(w, http.StatusForbidden, "forbidden")
			return
		}
		if s.SiteCatalog == nil {
			fail(w, http.StatusServiceUnavailable, "sites_unavailable")
			return
		}
		if r.Header.Get("Content-Type") != "application/json" {
			fail(w, http.StatusUnsupportedMediaType, "json_required")
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, 4096)
		var req struct {
			DatabaseID string `json:"database_id"`
		}
		if err := decodeSiteJSON(r.Body, &req); err != nil {
			fail(w, http.StatusBadRequest, "invalid_database_request")
			return
		}
		spec := sites.SiteDatabaseSpec{TenantID: p.TenantID, SiteID: siteID, DatabaseID: req.DatabaseID}
		var proof *auth.SessionReadProof
		if value, ok := r.Context().Value(sessionProofKey{}).(auth.SessionReadProof); ok {
			proof = &value
		}
		record, err := s.SiteCatalog.CreateDatabaseFor(r.Context(), spec, p.UserID, proof)
		if err != nil {
			writeSiteError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, record)
	default:
		w.Header().Set("Allow", "GET, POST")
		fail(w, http.StatusMethodNotAllowed, "method_not_allowed")
	}
}

func (s *Server) siteDatabaseResource(w http.ResponseWriter, r *http.Request, p rbac.Principal, siteID, databaseID string) {
	if r.URL.RawQuery != "" {
		fail(w, http.StatusBadRequest, "invalid_database_request")
		return
	}
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		fail(w, http.StatusMethodNotAllowed, "method_not_allowed")
		return
	}
	if !p.Can("sites:read", p.TenantID) {
		fail(w, http.StatusForbidden, "forbidden")
		return
	}
	if s.SiteCatalog == nil {
		fail(w, http.StatusServiceUnavailable, "sites_unavailable")
		return
	}
	record, err := s.SiteCatalog.GetDatabase(r.Context(), p.TenantID, siteID, databaseID)
	if err != nil {
		writeSiteError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, record)
}

func (s *Server) siteNginxPreview(w http.ResponseWriter, r *http.Request, p rbac.Principal, siteID string) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		fail(w, http.StatusMethodNotAllowed, "method_not_allowed")
		return
	}
	if !p.Can("sites:read", p.TenantID) {
		fail(w, http.StatusForbidden, "forbidden")
		return
	}
	if s.SiteCatalog == nil || s.SiteHostingPaths == nil {
		fail(w, http.StatusServiceUnavailable, "site_preview_unavailable")
		return
	}
	record, err := s.SiteCatalog.Get(r.Context(), p.TenantID, siteID)
	if err != nil {
		writeSiteError(w, err)
		return
	}
	config, err := sites.RenderNginxVhost(record, *s.SiteHostingPaths)
	if err != nil {
		writeSiteError(w, err)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusOK)
	_, _ = io.WriteString(w, config)
}
func (s *Server) siteGet(w http.ResponseWriter, r *http.Request, p rbac.Principal, siteID string) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		fail(w, http.StatusMethodNotAllowed, "method_not_allowed")
		return
	}
	if !p.Can("sites:read", p.TenantID) {
		fail(w, http.StatusForbidden, "forbidden")
		return
	}
	if s.SiteCatalog == nil {
		fail(w, http.StatusServiceUnavailable, "sites_unavailable")
		return
	}
	record, err := s.SiteCatalog.Get(r.Context(), p.TenantID, siteID)
	if err != nil {
		writeSiteError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, record)
}

func (s *Server) siteDomainVerificationBegin(w http.ResponseWriter, r *http.Request, p rbac.Principal, siteID, domain string) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		fail(w, http.StatusMethodNotAllowed, "method_not_allowed")
		return
	}
	if !p.Can("sites:write", p.TenantID) {
		fail(w, http.StatusForbidden, "forbidden")
		return
	}
	if s.SiteCatalog == nil {
		fail(w, http.StatusServiceUnavailable, "sites_unavailable")
		return
	}
	var proof *auth.SessionReadProof
	if value, ok := r.Context().Value(sessionProofKey{}).(auth.SessionReadProof); ok {
		proof = &value
	}
	instruction, err := s.SiteCatalog.BeginDomainVerificationFor(r.Context(), p.TenantID, siteID, domain, p.UserID, proof)
	if err != nil {
		writeSiteError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, instruction)
}

func (s *Server) siteDomainVerificationComplete(w http.ResponseWriter, r *http.Request, p rbac.Principal, siteID, domain string) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		fail(w, http.StatusMethodNotAllowed, "method_not_allowed")
		return
	}
	if !p.Can("sites:write", p.TenantID) {
		fail(w, http.StatusForbidden, "forbidden")
		return
	}
	if s.SiteCatalog == nil || s.DomainTXTLookup == nil {
		fail(w, http.StatusServiceUnavailable, "domain_verifier_unavailable")
		return
	}
	if r.Header.Get("Content-Type") != "application/json" {
		fail(w, http.StatusUnsupportedMediaType, "json_required")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 4096)
	var req struct {
		Value string `json:"value"`
	}
	if err := decodeSiteJSON(r.Body, &req); err != nil || req.Value == "" {
		fail(w, http.StatusBadRequest, "invalid_domain_proof")
		return
	}
	var proof *auth.SessionReadProof
	if value, ok := r.Context().Value(sessionProofKey{}).(auth.SessionReadProof); ok {
		proof = &value
	}
	err := s.SiteCatalog.CompleteDomainVerificationFor(r.Context(), p.TenantID, siteID, domain, req.Value, p.UserID, proof, s.DomainTXTLookup)
	if err != nil {
		writeSiteError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "verified"})
}

func writeSiteError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		return
	case errors.Is(err, sites.ErrSiteReadDenied):
		fail(w, http.StatusForbidden, "forbidden")
	case errors.Is(err, sites.ErrSiteSessionDenied):
		fail(w, http.StatusUnauthorized, "unauthorized")
	case errors.Is(err, sites.ErrSiteNotReady):
		fail(w, http.StatusConflict, "site_not_ready")
	case errors.Is(err, sites.ErrInvalidHostingPaths):
		fail(w, http.StatusServiceUnavailable, "site_preview_unavailable")
	case errors.Is(err, sites.ErrInvalidSpec), errors.Is(err, sites.ErrInvalidDomain):
		fail(w, http.StatusBadRequest, "invalid_site_request")
	case errors.Is(err, sites.ErrInvalidDatabaseSpec):
		fail(w, http.StatusBadRequest, "invalid_database_request")
	case errors.Is(err, sites.ErrSiteExists), errors.Is(err, sites.ErrDomainClaimed),
		errors.Is(err, sites.ErrDomainVerificationState), errors.Is(err, sites.ErrSiteDatabaseExists):
		fail(w, http.StatusConflict, "site_conflict")
	case errors.Is(err, sites.ErrSiteNotFound), errors.Is(err, sites.ErrDomainNotFound), errors.Is(err, sites.ErrSiteDatabaseNotFound):
		fail(w, http.StatusNotFound, "not_found")
	case errors.Is(err, sites.ErrPrimaryDomain):
		fail(w, http.StatusConflict, "primary_domain_required")
	case errors.Is(err, sites.ErrDomainProofMismatch):
		fail(w, http.StatusForbidden, "domain_proof_invalid")
	case errors.Is(err, sites.ErrDomainVerificationExpired):
		fail(w, http.StatusGone, "domain_proof_expired")
	case errors.Is(err, sites.ErrCatalogUnavailable), errors.Is(err, sites.ErrDomainVerifierUnavailable),
		errors.Is(err, sites.ErrSiteDatabaseNameCollision):
		fail(w, http.StatusServiceUnavailable, "sites_unavailable")
	default:
		fail(w, http.StatusServiceUnavailable, "sites_unavailable")
	}
}

func (s *Server) siteDomainRemove(w http.ResponseWriter, r *http.Request, p rbac.Principal, siteID, domain string) {
	if r.Method != http.MethodDelete {
		w.Header().Set("Allow", http.MethodDelete)
		fail(w, http.StatusMethodNotAllowed, "method_not_allowed")
		return
	}
	if !p.Can("sites:write", p.TenantID) {
		fail(w, http.StatusForbidden, "forbidden")
		return
	}
	if s.SiteCatalog == nil {
		fail(w, http.StatusServiceUnavailable, "sites_unavailable")
		return
	}
	var proof *auth.SessionReadProof
	if value, ok := r.Context().Value(sessionProofKey{}).(auth.SessionReadProof); ok {
		proof = &value
	}
	if err := s.SiteCatalog.RemoveAliasDomainFor(r.Context(), p.TenantID, siteID, domain, p.UserID, proof); err != nil {
		writeSiteError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) siteDomainAdd(w http.ResponseWriter, r *http.Request, p rbac.Principal, siteID string) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		fail(w, http.StatusMethodNotAllowed, "method_not_allowed")
		return
	}
	if !p.Can("sites:write", p.TenantID) {
		fail(w, http.StatusForbidden, "forbidden")
		return
	}
	if s.SiteCatalog == nil {
		fail(w, http.StatusServiceUnavailable, "sites_unavailable")
		return
	}
	if r.Header.Get("Content-Type") != "application/json" {
		fail(w, http.StatusUnsupportedMediaType, "json_required")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 4096)
	var req struct {
		Domain string `json:"domain"`
	}
	if err := decodeSiteJSON(r.Body, &req); err != nil || req.Domain == "" {
		fail(w, http.StatusBadRequest, "invalid_site_request")
		return
	}
	var proof *auth.SessionReadProof
	if value, ok := r.Context().Value(sessionProofKey{}).(auth.SessionReadProof); ok {
		proof = &value
	}
	claim, err := s.SiteCatalog.AddAliasDomainFor(r.Context(), p.TenantID, siteID, req.Domain, p.UserID, proof)
	if err != nil {
		writeSiteError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, claim)
}
