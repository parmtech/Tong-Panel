package sites

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// ErrInvalidPage indicates an invalid page size or site-ID boundary.
var ErrInvalidPage = errors.New("invalid site page")

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
	defer rows.Close()
	items := make([]SiteSummary, 0, limit+1)
	for rows.Next() {
		var item SiteSummary
		var domain, state sql.NullString
		if err := rows.Scan(&item.SiteID, &domain, &state); err != nil {
			return nil, "", fmt.Errorf("scan site page: %w", err)
		}
		if !domain.Valid || !state.Valid {
			return nil, "", errors.New("site primary domain missing")
		}
		item.PrimaryDomain, item.OwnershipState = domain.String, state.String
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, "", fmt.Errorf("iterate site page: %w", err)
	}
	next := ""
	if len(items) > limit {
		items = items[:limit]
		next = items[len(items)-1].SiteID
	}
	return items, next, nil
}
