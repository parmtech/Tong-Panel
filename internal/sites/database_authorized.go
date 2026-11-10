package sites

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/parmtech/tong-panel/internal/auth"
)

// CreateDatabaseFor creates declarative metadata and attributed audit in one
// authorized writer transaction. It does not provision MariaDB or credentials.
func (c *Catalog) CreateDatabaseFor(ctx context.Context, input SiteDatabaseSpec, requester string, proof *auth.SessionReadProof) (SiteDatabaseRecord, error) {
	spec, err := input.Validate()
	if err != nil {
		return SiteDatabaseRecord{}, err
	}
	databaseName, username := siteDatabaseNames(spec)
	record := SiteDatabaseRecord{TenantID: spec.TenantID, SiteID: spec.SiteID, DatabaseID: spec.DatabaseID, DatabaseName: databaseName, Username: username}
	err = c.withAuthorizedSiteWrite(ctx, spec.TenantID, requester, proof, func(conn *sql.Conn) error {
		record.CreatedAt = c.now().Unix()
		result, err := conn.ExecContext(ctx, "INSERT INTO site_databases(tenant_id,site_id,database_id,database_name,username,created_at) SELECT ?,?,?,?,?,? FROM sites WHERE tenant_id=? AND site_id=? ON CONFLICT(tenant_id,site_id,database_id) DO NOTHING", spec.TenantID, spec.SiteID, spec.DatabaseID, databaseName, username, record.CreatedAt, spec.TenantID, spec.SiteID)
		if err != nil {
			if strings.Contains(err.Error(), "site_databases.database_name") || strings.Contains(err.Error(), "site_databases.username") {
				return fmt.Errorf("%w: %v", ErrSiteDatabaseNameCollision, err)
			}
			return fmt.Errorf("insert site database metadata: %w", err)
		}
		n, err := result.RowsAffected()
		if err != nil {
			return err
		}
		if n != 1 {
			var exists bool
			if err := conn.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM sites WHERE tenant_id=? AND site_id=?)", spec.TenantID, spec.SiteID).Scan(&exists); err != nil {
				return err
			}
			if !exists {
				return ErrSiteNotFound
			}
			return ErrSiteDatabaseExists
		}
		return auditSiteMutation(ctx, conn, spec.TenantID, requester, spec.SiteID+"/"+spec.DatabaseID, "site.database_metadata_created")
	})
	if err != nil {
		return SiteDatabaseRecord{}, err
	}
	return record, nil
}
