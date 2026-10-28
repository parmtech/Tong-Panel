package sites

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/base32"
	"errors"
	"fmt"
	"strings"
)

var (
	ErrInvalidDatabaseSpec       = errors.New("invalid site database specification")
	ErrSiteDatabaseExists        = errors.New("site database already exists")
	ErrSiteDatabaseNotFound      = errors.New("site database not found")
	ErrSiteDatabaseNameCollision = errors.New("generated site database name collision")
)

type SiteDatabaseSpec struct {
	TenantID   string
	SiteID     string
	DatabaseID string
}

func (s SiteDatabaseSpec) Validate() (SiteDatabaseSpec, error) {
	if !validID(s.TenantID) || !validID(s.SiteID) || !validID(s.DatabaseID) {
		return SiteDatabaseSpec{}, ErrInvalidDatabaseSpec
	}
	return s, nil
}

// SiteDatabaseRecord contains names and ownership metadata only. It never stores
// a database password or asserts that a MariaDB database/user has been created.
type SiteDatabaseRecord struct {
	TenantID     string
	SiteID       string
	DatabaseID   string
	DatabaseName string
	Username     string
	CreatedAt    int64
}

func (c *Catalog) CreateDatabase(ctx context.Context, input SiteDatabaseSpec) error {
	if ctx == nil || c == nil || c.db == nil {
		return ErrCatalogUnavailable
	}
	spec, err := input.Validate()
	if err != nil {
		return err
	}
	databaseName, username := siteDatabaseNames(spec)
	tx, err := c.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin site database catalog create: %w", err)
	}
	defer tx.Rollback()

	result, err := tx.ExecContext(ctx,
		"INSERT INTO site_databases(tenant_id,site_id,database_id,database_name,username,created_at) "+
			"SELECT ?,?,?,?,?,? FROM sites WHERE tenant_id=? AND site_id=? "+
			"ON CONFLICT(tenant_id,site_id,database_id) DO NOTHING",
		spec.TenantID, spec.SiteID, spec.DatabaseID, databaseName, username, c.now().Unix(),
		spec.TenantID, spec.SiteID)
	if err != nil {
		if strings.Contains(err.Error(), "site_databases.database_name") ||
			strings.Contains(err.Error(), "site_databases.username") {
			return fmt.Errorf("%w: %v", ErrSiteDatabaseNameCollision, err)
		}
		return fmt.Errorf("insert site database metadata: %w", err)
	}
	inserted, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read site database insert result: %w", err)
	}
	if inserted != 1 {
		var siteExists bool
		if err := tx.QueryRowContext(ctx,
			"SELECT EXISTS(SELECT 1 FROM sites WHERE tenant_id=? AND site_id=?)",
			spec.TenantID, spec.SiteID).Scan(&siteExists); err != nil {
			return fmt.Errorf("check site database owner after insert: %w", err)
		}
		if !siteExists {
			return ErrSiteNotFound
		}
		return ErrSiteDatabaseExists
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit site database metadata: %w", err)
	}
	return nil
}

func (c *Catalog) GetDatabase(ctx context.Context, tenantID, siteID, databaseID string) (SiteDatabaseRecord, error) {
	if ctx == nil || c == nil || c.db == nil {
		return SiteDatabaseRecord{}, ErrCatalogUnavailable
	}
	spec, err := (SiteDatabaseSpec{TenantID: tenantID, SiteID: siteID, DatabaseID: databaseID}).Validate()
	if err != nil {
		return SiteDatabaseRecord{}, err
	}
	var record SiteDatabaseRecord
	err = c.db.QueryRowContext(ctx,
		"SELECT tenant_id,site_id,database_id,database_name,username,created_at "+
			"FROM site_databases WHERE tenant_id=? AND site_id=? AND database_id=?",
		spec.TenantID, spec.SiteID, spec.DatabaseID).Scan(
		&record.TenantID, &record.SiteID, &record.DatabaseID,
		&record.DatabaseName, &record.Username, &record.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return SiteDatabaseRecord{}, ErrSiteDatabaseNotFound
	}
	if err != nil {
		return SiteDatabaseRecord{}, fmt.Errorf("read site database metadata: %w", err)
	}
	return record, nil
}

func siteDatabaseNames(spec SiteDatabaseSpec) (databaseName, username string) {
	sum := sha256.Sum256([]byte(spec.TenantID + "\x00" + spec.SiteID + "\x00" + spec.DatabaseID))
	suffix := strings.ToLower(base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(sum[:16]))
	return "tp_" + suffix, "tu_" + suffix
}
