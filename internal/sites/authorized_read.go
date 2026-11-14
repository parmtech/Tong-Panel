package sites

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/parmtech/tong-panel/internal/auth"
)

const membershipReadAuthorization = `SELECT NULL AS expires_at, 0 AS idle_deadline_unix
	WHERE EXISTS (
		SELECT 1 FROM memberships m
		JOIN tenants t ON t.id=m.tenant_id AND t.status='active'
		JOIN identities i ON i.id=m.identity_id AND i.disabled=0
		WHERE m.tenant_id=? AND m.identity_id=?
		AND m.role IN ('owner','admin','reseller','customer','auditor')
	)`

const sessionReadAuthorization = `SELECT s.expires_at, s.idle_deadline_unix
	FROM sessions s
	JOIN memberships m ON m.tenant_id=s.tenant_id AND m.identity_id=s.identity_id
	JOIN tenants t ON t.id=m.tenant_id AND t.status='active'
	JOIN identities i ON i.id=m.identity_id AND i.disabled=0
	WHERE s.token_sha256=? AND s.tenant_id=? AND s.identity_id=? AND s.revoked_at IS NULL
	AND m.role IN ('owner','admin','reseller','customer','auditor')`

type siteReadAuthorization struct {
	proof   *auth.SessionReadProof
	seen    bool
	expires sql.NullString
	idle    int64
}

func (c *Catalog) authorizedReadRows(ctx context.Context, tenantID, requester string, proof *auth.SessionReadProof, detailSQL string, detailArgs ...any) (*sql.Rows, *siteReadAuthorization, error) {
	validPrincipalID := func(id string) bool {
		return id != "" && utf8.ValidString(id) && !strings.ContainsRune(id, 0) && utf8.RuneCountInString(id) <= 128
	}
	if ctx == nil || c == nil || c.db == nil || !validPrincipalID(tenantID) || !validPrincipalID(requester) {
		return nil, nil, ErrCatalogUnavailable
	}
	authorizationSQL := membershipReadAuthorization
	args := []any{tenantID, requester}
	if proof != nil {
		if !proof.MatchesDatabase(c.db) {
			return nil, nil, ErrCatalogUnavailable
		}
		digest := proof.Digest()
		authorizationSQL = sessionReadAuthorization
		args = []any{digest[:], tenantID, requester}
	}
	query := "WITH authorized AS (" + authorizationSQL + "), detail AS (" + detailSQL +
		") SELECT authorized.expires_at,authorized.idle_deadline_unix,detail.* FROM authorized LEFT JOIN detail ON 1=1"
	args = append(args, detailArgs...)
	rows, err := c.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, nil, fmt.Errorf("read authorized site metadata: %w", err)
	}
	return rows, &siteReadAuthorization{proof: proof}, nil
}

func (a *siteReadAuthorization) observe(expires sql.NullString, idle int64) {
	a.seen = true
	a.expires = expires
	a.idle = idle
}

func (a *siteReadAuthorization) finish(ctx context.Context) error {
	if !a.seen {
		if a.proof != nil {
			return ErrSiteSessionDenied
		}
		return ErrSiteReadDenied
	}
	if a.proof == nil {
		return ctx.Err()
	}
	if !a.expires.Valid {
		return ErrSiteSessionDenied
	}
	expires, err := time.Parse(time.RFC3339Nano, a.expires.String)
	if err != nil {
		return ErrSiteSessionDenied
	}
	now := a.proof.Now()
	if !now.Before(expires) {
		if err := a.proof.RevokeObservedExpiry(ctx, a.expires.String, now, true); err != nil {
			return fmt.Errorf("record site read session expiry: %w", err)
		}
		return ErrSiteSessionDenied
	}
	if a.idle <= now.Unix() {
		if err := a.proof.RevokeObservedExpiry(ctx, a.expires.String, now, false); err != nil {
			return fmt.Errorf("record site read session idle expiry: %w", err)
		}
		return ErrSiteSessionDenied
	}
	return ctx.Err()
}

// GetFor reads site details only while the requester's current membership (and
// session proof, when provided) is valid in the same SQLite statement snapshot.
func (c *Catalog) GetFor(ctx context.Context, tenantID, siteID, requester string, proof *auth.SessionReadProof) (SiteRecord, error) {
	if !validID(tenantID) || !validID(siteID) {
		return SiteRecord{}, ErrInvalidSpec
	}
	detail := `SELECT s.tenant_id,s.site_id,s.document_root,s.created_at,d.domain,d.kind,d.ownership_state
		FROM sites s LEFT JOIN site_domains d ON d.tenant_id=s.tenant_id AND d.site_id=s.site_id
		WHERE s.tenant_id=? AND s.site_id=? ORDER BY CASE d.kind WHEN 'primary' THEN 0 ELSE 1 END,d.domain`
	rows, authz, err := c.authorizedReadRows(ctx, tenantID, requester, proof, detail, tenantID, siteID)
	if err != nil {
		return SiteRecord{}, err
	}
	defer rows.Close()
	var record SiteRecord
	found := false
	for rows.Next() {
		var expiry sql.NullString
		var idle int64
		var recordTenant, recordSite, root, domain, kind, state sql.NullString
		var created sql.NullInt64
		if err := rows.Scan(&expiry, &idle, &recordTenant, &recordSite, &root, &created, &domain, &kind, &state); err != nil {
			return SiteRecord{}, fmt.Errorf("scan authorized site details: %w", err)
		}
		authz.observe(expiry, idle)
		if !recordSite.Valid {
			continue
		}
		if !found {
			record = SiteRecord{TenantID: recordTenant.String, SiteID: recordSite.String, DocumentRoot: root.String, CreatedAt: created.Int64}
			found = true
		}
		if domain.Valid {
			record.Domains = append(record.Domains, DomainClaim{Name: domain.String, Kind: kind.String, OwnershipState: state.String})
		}
	}
	if err := rows.Err(); err != nil {
		return SiteRecord{}, fmt.Errorf("iterate authorized site details: %w", err)
	}
	if err := rows.Close(); err != nil {
		return SiteRecord{}, fmt.Errorf("close authorized site details: %w", err)
	}
	if err := authz.finish(ctx); err != nil {
		return SiteRecord{}, err
	}
	if !found {
		return SiteRecord{}, ErrSiteNotFound
	}
	return record, nil
}

// GetDatabaseFor reads one database metadata record under current membership
// and, for cookie requests, current session state in the same statement.
func (c *Catalog) GetDatabaseFor(ctx context.Context, tenantID, siteID, databaseID, requester string, proof *auth.SessionReadProof) (SiteDatabaseRecord, error) {
	spec, err := (SiteDatabaseSpec{TenantID: tenantID, SiteID: siteID, DatabaseID: databaseID}).Validate()
	if err != nil {
		return SiteDatabaseRecord{}, err
	}
	detail := `SELECT tenant_id,site_id,database_id,database_name,username,created_at
		FROM site_databases WHERE tenant_id=? AND site_id=? AND database_id=?`
	rows, authz, err := c.authorizedReadRows(ctx, tenantID, requester, proof, detail,
		spec.TenantID, spec.SiteID, spec.DatabaseID)
	if err != nil {
		return SiteDatabaseRecord{}, err
	}
	defer rows.Close()
	var record SiteDatabaseRecord
	found := false
	for rows.Next() {
		var expiry sql.NullString
		var idle int64
		var tenant, site, id, name, username sql.NullString
		var created sql.NullInt64
		if err := rows.Scan(&expiry, &idle, &tenant, &site, &id, &name, &username, &created); err != nil {
			return SiteDatabaseRecord{}, fmt.Errorf("scan authorized site database: %w", err)
		}
		authz.observe(expiry, idle)
		if id.Valid {
			record = SiteDatabaseRecord{TenantID: tenant.String, SiteID: site.String, DatabaseID: id.String,
				DatabaseName: name.String, Username: username.String, CreatedAt: created.Int64}
			found = true
		}
	}
	if err := rows.Err(); err != nil {
		return SiteDatabaseRecord{}, fmt.Errorf("iterate authorized site database: %w", err)
	}
	if err := rows.Close(); err != nil {
		return SiteDatabaseRecord{}, fmt.Errorf("close authorized site database: %w", err)
	}
	if err := authz.finish(ctx); err != nil {
		return SiteDatabaseRecord{}, err
	}
	if !found {
		return SiteDatabaseRecord{}, ErrSiteDatabaseNotFound
	}
	return record, nil
}

// ListDatabasesFor reads one site's database metadata under current membership
// and, for cookie requests, current session state in the same statement.
func (c *Catalog) ListDatabasesFor(ctx context.Context, tenantID, siteID, requester string, proof *auth.SessionReadProof) ([]SiteDatabaseRecord, error) {
	if !validID(tenantID) || !validID(siteID) {
		return nil, ErrInvalidDatabaseSpec
	}
	detail := `SELECT s.site_id AS parent_site_id,d.tenant_id,d.site_id,d.database_id,d.database_name,d.username,d.created_at
		FROM sites s LEFT JOIN site_databases d ON d.tenant_id=s.tenant_id AND d.site_id=s.site_id
		WHERE s.tenant_id=? AND s.site_id=? ORDER BY d.created_at,d.database_id`
	rows, authz, err := c.authorizedReadRows(ctx, tenantID, requester, proof, detail, tenantID, siteID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	records := make([]SiteDatabaseRecord, 0)
	siteFound := false
	for rows.Next() {
		var expiry sql.NullString
		var idle int64
		var parent, tenant, site, id, name, username sql.NullString
		var created sql.NullInt64
		if err := rows.Scan(&expiry, &idle, &parent, &tenant, &site, &id, &name, &username, &created); err != nil {
			return nil, fmt.Errorf("scan authorized site database list: %w", err)
		}
		authz.observe(expiry, idle)
		if parent.Valid {
			siteFound = true
		}
		if id.Valid {
			records = append(records, SiteDatabaseRecord{TenantID: tenant.String, SiteID: site.String, DatabaseID: id.String,
				DatabaseName: name.String, Username: username.String, CreatedAt: created.Int64})
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate authorized site database list: %w", err)
	}
	if err := rows.Close(); err != nil {
		return nil, fmt.Errorf("close authorized site database list: %w", err)
	}
	if err := authz.finish(ctx); err != nil {
		return nil, err
	}
	if !siteFound {
		return nil, ErrSiteNotFound
	}
	return records, nil
}

// ListDatabasePageFor returns a bounded, ID-ordered page from the same SQLite
// snapshot as the current membership and optional session proof. The cursor is
// an ordering boundary, not authority; pages are not a consistent snapshot.
func (c *Catalog) ListDatabasePageFor(ctx context.Context, tenantID, siteID, requester, after string, limit int, proof *auth.SessionReadProof) ([]SiteDatabaseRecord, string, error) {
	if limit < 1 || limit > 50 || (after != "" && !validID(after)) {
		return nil, "", ErrInvalidPage
	}
	if !validID(tenantID) || !validID(siteID) {
		return nil, "", ErrInvalidDatabaseSpec
	}
	detail := `SELECT s.site_id AS parent_site_id,d.tenant_id,d.site_id,d.database_id,d.database_name,d.username,d.created_at
        FROM sites s LEFT JOIN site_databases d ON d.tenant_id=s.tenant_id AND d.site_id=s.site_id AND d.database_id>? COLLATE BINARY
        WHERE s.tenant_id=? AND s.site_id=? ORDER BY d.database_id COLLATE BINARY LIMIT ?`
	rows, authz, err := c.authorizedReadRows(ctx, tenantID, requester, proof, detail, after, tenantID, siteID, limit+1)
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()
	records := make([]SiteDatabaseRecord, 0, limit+1)
	siteFound := false
	for rows.Next() {
		var expiry sql.NullString
		var idle int64
		var parent, tenant, site, id, name, username sql.NullString
		var created sql.NullInt64
		if err := rows.Scan(&expiry, &idle, &parent, &tenant, &site, &id, &name, &username, &created); err != nil {
			return nil, "", fmt.Errorf("scan authorized database page: %w", err)
		}
		authz.observe(expiry, idle)
		if parent.Valid {
			siteFound = true
		}
		if id.Valid {
			records = append(records, SiteDatabaseRecord{TenantID: tenant.String, SiteID: site.String, DatabaseID: id.String, DatabaseName: name.String, Username: username.String, CreatedAt: created.Int64})
		}
	}
	if err := rows.Err(); err != nil {
		return nil, "", fmt.Errorf("iterate authorized database page: %w", err)
	}
	if err := rows.Close(); err != nil {
		return nil, "", fmt.Errorf("close authorized database page: %w", err)
	}
	if err := authz.finish(ctx); err != nil {
		return nil, "", err
	}
	if !siteFound {
		return nil, "", ErrSiteNotFound
	}
	sort.Slice(records, func(i, j int) bool { return records[i].DatabaseID < records[j].DatabaseID })
	next := ""
	if len(records) > limit {
		records = records[:limit]
		next = records[len(records)-1].DatabaseID
	}
	return records, next, nil
}
