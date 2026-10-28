package api

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/parmtech/tong-panel/internal/rbac"
	"github.com/parmtech/tong-panel/internal/sites"
)

const maxSiteRequestBytes = 16 << 10

func (s *Server) siteCollection(w http.ResponseWriter, r *http.Request, p rbac.Principal) {
	if r.URL.Path != "/api/v1/sites" || r.URL.RawQuery != "" {
		fail(w, http.StatusNotFound, "not_found")
		return
	}
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
	if err := s.SiteCatalog.Create(r.Context(), spec); err != nil {
		writeSiteError(w, err)
		return
	}
	created, err := s.SiteCatalog.Get(r.Context(), p.TenantID, req.SiteID)
	if err != nil {
		writeSiteError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, created)
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
	instruction, err := s.SiteCatalog.BeginDomainVerification(r.Context(), p.TenantID, siteID, domain)
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
	err := s.SiteCatalog.CompleteDomainVerification(r.Context(), p.TenantID, siteID, domain, req.Value, s.DomainTXTLookup)
	if err != nil {
		writeSiteError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "verified"})
}

func writeSiteError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, sites.ErrInvalidSpec), errors.Is(err, sites.ErrInvalidDomain):
		fail(w, http.StatusBadRequest, "invalid_site_request")
	case errors.Is(err, sites.ErrSiteExists), errors.Is(err, sites.ErrDomainClaimed),
		errors.Is(err, sites.ErrDomainVerificationState):
		fail(w, http.StatusConflict, "site_conflict")
	case errors.Is(err, sites.ErrSiteNotFound), errors.Is(err, sites.ErrDomainNotFound):
		fail(w, http.StatusNotFound, "not_found")
	case errors.Is(err, sites.ErrPrimaryDomain):
		fail(w, http.StatusConflict, "primary_domain_required")
	case errors.Is(err, sites.ErrDomainProofMismatch):
		fail(w, http.StatusForbidden, "domain_proof_invalid")
	case errors.Is(err, sites.ErrDomainVerificationExpired):
		fail(w, http.StatusGone, "domain_proof_expired")
	case errors.Is(err, sites.ErrCatalogUnavailable), errors.Is(err, sites.ErrDomainVerifierUnavailable):
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
	if err := s.SiteCatalog.RemoveAliasDomain(r.Context(), p.TenantID, siteID, domain); err != nil {
		writeSiteError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
