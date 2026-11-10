package sites

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"
)

// ErrInvalidPage indicates an invalid page size or site-ID boundary.
var ErrInvalidPage = errors.New("invalid site page")

// ErrSiteReadDenied means the current catalog database state does not grant
// the requester tenant-scoped site reads.
var ErrSiteReadDenied = errors.New("site read denied")

// SiteSummary is catalog metadata only; a verified domain does not mean the
// site has been provisioned or that a host service is running.
type SiteSummary struct {
	SiteID         string `json:"site_id"`
	PrimaryDomain  string `json:"primary_domain"`
	OwnershipState string `json:"ownership_state"`
}

// ListPage returns at most limit summaries for one tenant, ordered by binary
// site ID. A nonempty next cursor is the last returned ID when more rows exist.
// The cursor is only an ordering boundary, not an authorization token. Pages do
// not form a snapshot: a fresh listing is needed for earlier concurrent inserts.
func (c *Catalog) ListPage(ctx context.Context, tenantID, after string, limit int) ([]SiteSummary, string, error) {
	if limit < 1 || limit > 50 || (after != "" && !validID(after)) {
		return nil, "", ErrInvalidPage
	}
	if ctx == nil || c == nil || c.db == nil || !validID(tenantID) {
		return nil, "", ErrCatalogUnavailable
	}
	rows, err := c.db.QueryContext(ctx, `SELECT s.site_id,d.domain,d.ownership_state
		FROM sites s LEFT JOIN site_domains d
		ON d.tenant_id=s.tenant_id AND d.site_id=s.site_id AND d.kind='primary'
		WHERE s.tenant_id=? AND s.site_id>? COLLATE BINARY
		ORDER BY s.site_id COLLATE BINARY LIMIT ?`, tenantID, after, limit+1)
	if err != nil {
		return nil, "", fmt.Errorf("read site page: %w", err)
	}
	return scanSitePage(rows, limit, false)
}

// ListPageFor checks current membership in the same SQLite statement and
// snapshot as the returned site rows. It does not revalidate a session token.
func (c *Catalog) ListPageFor(ctx context.Context, tenantID, requester, after string, limit int) ([]SiteSummary, string, error) {
	if limit < 1 || limit > 50 || (after != "" && !validID(after)) {
		return nil, "", ErrInvalidPage
	}
	validPrincipalID := func(id string) bool {
		return id != "" && utf8.ValidString(id) && !strings.ContainsRune(id, 0) && utf8.RuneCountInString(id) <= 128
	}
	if ctx == nil || c == nil || c.db == nil || !validPrincipalID(tenantID) || !validPrincipalID(requester) {
		return nil, "", ErrCatalogUnavailable
	}
	rows, err := c.db.QueryContext(ctx, `WITH authorized AS (
		SELECT 1 AS allowed FROM memberships m
		JOIN tenants t ON t.id=m.tenant_id AND t.status='active'
		JOIN identities i ON i.id=m.identity_id AND i.disabled=0
		WHERE m.tenant_id=? AND m.identity_id=? AND m.role IN ('owner','admin','reseller','customer','auditor')
	), page AS (
		SELECT s.site_id,d.domain,d.ownership_state FROM sites s LEFT JOIN site_domains d
		ON d.tenant_id=s.tenant_id AND d.site_id=s.site_id AND d.kind='primary'
		WHERE s.tenant_id=? AND s.site_id>? COLLATE BINARY
		ORDER BY s.site_id COLLATE BINARY LIMIT ?
	)
	SELECT page.site_id,page.domain,page.ownership_state FROM authorized
	LEFT JOIN page ON 1=1 ORDER BY page.site_id COLLATE BINARY`, tenantID, requester, tenantID, after, limit+1)
	if err != nil {
		return nil, "", fmt.Errorf("read authorized site page: %w", err)
	}
	return scanSitePage(rows, limit, true)
}

func scanSitePage(rows *sql.Rows, limit int, requireAuthorization bool) ([]SiteSummary, string, error) {
	defer rows.Close()
	items := make([]SiteSummary, 0, limit+1)
	seen := false
	for rows.Next() {
		var item SiteSummary
		var id, domain, state sql.NullString
		if err := rows.Scan(&id, &domain, &state); err != nil {
			return nil, "", fmt.Errorf("scan site page: %w", err)
		}
		seen = true
		if !id.Valid {
			continue // Authorized empty page is represented by a LEFT JOIN row.
		}
		if !domain.Valid || !state.Valid {
			return nil, "", errors.New("site primary domain missing")
		}
		item.SiteID, item.PrimaryDomain, item.OwnershipState = id.String, domain.String, state.String
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, "", fmt.Errorf("iterate site page: %w", err)
	}
	if requireAuthorization && !seen {
		return nil, "", ErrSiteReadDenied
	}
	next := ""
	if len(items) > limit {
		items = items[:limit]
		next = items[len(items)-1].SiteID
	}
	return items, next, nil
}
