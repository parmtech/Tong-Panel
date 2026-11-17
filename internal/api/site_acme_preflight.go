package api

import (
	"net/http"

	"github.com/parmtech/tong-panel/internal/rbac"
)

func (s *Server) siteACMEHTTP01Preflight(w http.ResponseWriter, r *http.Request, p rbac.Principal, siteID string) {
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
	snapshot, err := s.SiteCatalog.ACMEHTTP01PreflightFor(r.Context(), p.TenantID, siteID, p.UserID, siteReadSessionProof(r))
	if err != nil {
		writeSiteError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, struct {
		SiteID           string   `json:"site_id"`
		Domains          []string `json:"domains"`
		Challenge        string   `json:"challenge"`
		SnapshotOnly     bool     `json:"snapshot_only"`
		ExecutionEnabled bool     `json:"execution_enabled"`
	}{SiteID: snapshot.SiteID, Domains: snapshot.Domains, Challenge: "http-01", SnapshotOnly: true, ExecutionEnabled: false})
}
