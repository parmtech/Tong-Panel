package sites

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/parmtech/tong-panel/internal/auth"
)

// AddAliasDomainFor adds a pending claim and an attributed audit atomically.
func (c *Catalog) AddAliasDomainFor(ctx context.Context, tenant, siteID, rawDomain, requester string, proof *auth.SessionReadProof) (DomainClaim, error) {
	if !validID(tenant) || !validID(siteID) {
		return DomainClaim{}, ErrCatalogUnavailable
	}
	domain, err := NormalizeDomain(rawDomain)
	if err != nil {
		return DomainClaim{}, err
	}
	err = c.withAuthorizedSiteWrite(ctx, tenant, requester, proof, func(conn *sql.Conn) error {
		var exists bool
		if err := conn.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM sites WHERE tenant_id=? AND site_id=?)", tenant, siteID).Scan(&exists); err != nil {
			return err
		}
		if !exists {
			return ErrSiteNotFound
		}
		result, err := conn.ExecContext(ctx, "INSERT INTO site_domains(domain,tenant_id,site_id,kind,ownership_state) VALUES(?,?,?,'alias','pending') ON CONFLICT(domain) DO NOTHING", domain, tenant, siteID)
		if err != nil {
			return fmt.Errorf("insert alias: %w", err)
		}
		n, err := result.RowsAffected()
		if err != nil {
			return err
		}
		if n != 1 {
			return ErrDomainClaimed
		}
		return auditAlias(ctx, conn, tenant, requester, siteID+"/"+domain, "site.alias_added")
	})
	if err != nil {
		return DomainClaim{}, err
	}
	return DomainClaim{Name: domain, Kind: "alias", OwnershipState: "pending"}, nil
}

// RemoveAliasDomainFor removes an alias and records its actor atomically.
func (c *Catalog) RemoveAliasDomainFor(ctx context.Context, tenant, siteID, rawDomain, requester string, proof *auth.SessionReadProof) error {
	if !validID(tenant) || !validID(siteID) {
		return ErrCatalogUnavailable
	}
	domain, err := NormalizeDomain(rawDomain)
	if err != nil {
		return err
	}
	return c.withAuthorizedSiteWrite(ctx, tenant, requester, proof, func(conn *sql.Conn) error {
		var kind string
		err := conn.QueryRowContext(ctx, "SELECT kind FROM site_domains WHERE domain=? AND tenant_id=? AND site_id=?", domain, tenant, siteID).Scan(&kind)
		if errors.Is(err, sql.ErrNoRows) {
			return ErrDomainNotFound
		}
		if err != nil {
			return err
		}
		if kind == "primary" {
			return ErrPrimaryDomain
		}
		result, err := conn.ExecContext(ctx, "DELETE FROM site_domains WHERE domain=? AND tenant_id=? AND site_id=? AND kind='alias'", domain, tenant, siteID)
		if err != nil {
			return err
		}
		count, err := result.RowsAffected()
		if err != nil {
			return err
		}
		if count != 1 {
			return errors.New("alias removal did not affect one row")
		}
		return auditAlias(ctx, conn, tenant, requester, siteID+"/"+domain, "site.alias_removed")
	})
}

func auditAlias(ctx context.Context, conn *sql.Conn, tenant, requester, siteID, action string) error {
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return fmt.Errorf("generate alias audit ID: %w", err)
	}
	id := hex.EncodeToString(raw[:])
	if _, err := conn.ExecContext(ctx, `INSERT INTO audit_events(id,tenant_id,actor_identity_id,actor_kind,action,subject_id,request_id) VALUES(?,?,?,'user',?,?,?)`, id, tenant, requester, action, siteID, id); err != nil {
		return fmt.Errorf("audit alias mutation: %w", err)
	}
	return nil
}
