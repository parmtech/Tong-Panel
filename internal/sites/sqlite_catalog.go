package sites

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

var (
	ErrCatalogUnavailable = errors.New("site catalog unavailable")
	ErrSiteExists         = errors.New("site already exists")
	ErrSiteNotFound       = errors.New("site not found")
	ErrDomainClaimed      = errors.New("domain is already claimed")
	ErrDomainNotFound     = errors.New("site domain not found")
	ErrPrimaryDomain      = errors.New("primary domain cannot be removed")
)

// Catalog persists declarative site metadata only. A pending domain row is not
// proof of control and must never authorize certificate issuance or host changes.
type Catalog struct {
	db  *sql.DB
	now func() time.Time
}

type DomainClaim struct {
	Name           string
	Kind           string
	OwnershipState string
}

type SiteRecord struct {
	TenantID     string
	SiteID       string
	DocumentRoot string
	CreatedAt    int64
	Domains      []DomainClaim
}

func NewCatalog(db *sql.DB) (*Catalog, error) {
	if db == nil {
		return nil, ErrCatalogUnavailable
	}
	return &Catalog{db: db, now: time.Now}, nil
}

// Create atomically inserts one site and its globally unique domain claims.
// Domain claims stay pending; ownership verification is deliberately separate.
func (c *Catalog) Create(ctx context.Context, input SiteSpec) error {
	if ctx == nil || c == nil || c.db == nil {
		return ErrCatalogUnavailable
	}
	spec, err := input.Validate()
	if err != nil {
		return err
	}
	tx, err := c.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin site catalog create: %w", err)
	}
	defer tx.Rollback()

	result, err := tx.ExecContext(ctx,
		"INSERT INTO sites(tenant_id,site_id,document_root,created_at) VALUES(?,?,?,?) ON CONFLICT(tenant_id,site_id) DO NOTHING",
		spec.TenantID, spec.SiteID, spec.DocumentRoot, c.now().Unix())
	if err != nil {
		return fmt.Errorf("insert site metadata: %w", err)
	}
	inserted, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read site insert result: %w", err)
	}
	if inserted != 1 {
		return ErrSiteExists
	}

	domains := append([]string{spec.PrimaryDomain}, spec.Aliases...)
	for i, domain := range domains {
		kind := "alias"
		if i == 0 {
			kind = "primary"
		}
		result, err := tx.ExecContext(ctx,
			"INSERT INTO site_domains(domain,tenant_id,site_id,kind,ownership_state) VALUES(?,?,?,?, 'pending') ON CONFLICT(domain) DO NOTHING",
			domain, spec.TenantID, spec.SiteID, kind)
		if err != nil {
			return fmt.Errorf("insert site domain claim: %w", err)
		}
		inserted, err := result.RowsAffected()
		if err != nil {
			return fmt.Errorf("read domain claim insert result: %w", err)
		}
		if inserted != 1 {
			return fmt.Errorf("%w: %s", ErrDomainClaimed, domain)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit site catalog create: %w", err)
	}
	return nil
}

// Get returns only the requested tenant's site record.
func (c *Catalog) Get(ctx context.Context, tenantID, siteID string) (SiteRecord, error) {
	if ctx == nil || c == nil || c.db == nil || !validID(tenantID) || !validID(siteID) {
		return SiteRecord{}, ErrCatalogUnavailable
	}
	var record SiteRecord
	err := c.db.QueryRowContext(ctx,
		"SELECT tenant_id,site_id,document_root,created_at FROM sites WHERE tenant_id=? AND site_id=?",
		tenantID, siteID).Scan(&record.TenantID, &record.SiteID, &record.DocumentRoot, &record.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return SiteRecord{}, ErrSiteNotFound
	}
	if err != nil {
		return SiteRecord{}, fmt.Errorf("read site catalog: %w", err)
	}
	rows, err := c.db.QueryContext(ctx,
		"SELECT domain,kind,ownership_state FROM site_domains WHERE tenant_id=? AND site_id=? ORDER BY CASE kind WHEN 'primary' THEN 0 ELSE 1 END, domain",
		tenantID, siteID)
	if err != nil {
		return SiteRecord{}, fmt.Errorf("read site domains: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var claim DomainClaim
		if err := rows.Scan(&claim.Name, &claim.Kind, &claim.OwnershipState); err != nil {
			return SiteRecord{}, fmt.Errorf("scan site domain: %w", err)
		}
		record.Domains = append(record.Domains, claim)
	}
	if err := rows.Err(); err != nil {
		return SiteRecord{}, fmt.Errorf("iterate site domains: %w", err)
	}
	return record, nil
}

// RemoveAliasDomain releases an alias claim for the requested tenant/site.
// The primary domain is immutable through this operation. This changes metadata
// only; callers must not infer that any host configuration was changed.
func (c *Catalog) RemoveAliasDomain(ctx context.Context, tenantID, siteID, rawDomain string) error {
	if ctx == nil || c == nil || c.db == nil || !validID(tenantID) || !validID(siteID) {
		return ErrCatalogUnavailable
	}
	domain, err := NormalizeDomain(rawDomain)
	if err != nil {
		return err
	}
	result, err := c.db.ExecContext(ctx,
		"DELETE FROM site_domains WHERE domain=? AND tenant_id=? AND site_id=? AND kind='alias'",
		domain, tenantID, siteID)
	if err != nil {
		return fmt.Errorf("remove site alias domain: %w", err)
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read site alias removal result: %w", err)
	}
	if changed == 1 {
		return nil
	}
	var kind string
	err = c.db.QueryRowContext(ctx,
		"SELECT kind FROM site_domains WHERE domain=? AND tenant_id=? AND site_id=?",
		domain, tenantID, siteID).Scan(&kind)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrDomainNotFound
	}
	if err != nil {
		return fmt.Errorf("check site domain removal result: %w", err)
	}
	if kind == "primary" {
		return ErrPrimaryDomain
	}
	return ErrDomainNotFound
}
