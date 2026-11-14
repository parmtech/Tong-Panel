package sites

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/parmtech/tong-panel/internal/auth"
	"github.com/parmtech/tong-panel/internal/database"
	"github.com/parmtech/tong-panel/internal/rbac"
)

// CreateFor authorizes and records a site creation under one SQLite writer
// lock. A nil session proof is for the explicitly configured bearer path.
// It writes metadata only and does not provision a host or verify a domain.
func (c *Catalog) CreateFor(ctx context.Context, input SiteSpec, requester string, proof *auth.SessionReadProof) (SiteRecord, error) {
	if ctx == nil || c == nil || c.db == nil || requester == "" {
		return SiteRecord{}, ErrCatalogUnavailable
	}
	spec, err := input.Validate()
	if err != nil {
		return SiteRecord{}, err
	}
	var createdAt int64
	err = c.withAuthorizedSiteWrite(ctx, spec.TenantID, requester, proof, func(conn *sql.Conn) error {
		createdAt = c.now().Unix()
		if err := insertSite(ctx, conn, spec, createdAt); err != nil {
			return err
		}
		var eventID [16]byte
		if _, err := rand.Read(eventID[:]); err != nil {
			return fmt.Errorf("generate site audit ID: %w", err)
		}
		id := hex.EncodeToString(eventID[:])
		if _, err := conn.ExecContext(ctx, `INSERT INTO audit_events(id,tenant_id,actor_identity_id,actor_kind,action,subject_id,request_id)
			VALUES(?,?,?,'user','site.created',?,?)`, id, spec.TenantID, requester, spec.SiteID, id); err != nil {
			return fmt.Errorf("audit site creation: %w", err)
		}
		return nil
	})
	if err != nil {
		return SiteRecord{}, err
	}
	record := SiteRecord{TenantID: spec.TenantID, SiteID: spec.SiteID, DocumentRoot: spec.DocumentRoot, CreatedAt: createdAt}
	for i, domain := range append([]string{spec.PrimaryDomain}, spec.Aliases...) {
		kind := "alias"
		if i == 0 {
			kind = "primary"
		}
		record.Domains = append(record.Domains, DomainClaim{Name: domain, Kind: kind, OwnershipState: "pending"})
	}
	sort.Slice(record.Domains[1:], func(i, j int) bool { return record.Domains[i+1].Name < record.Domains[j+1].Name })
	return record, nil
}

// withAuthorizedSiteWrite serializes current authorization, the mutation and
// its audit record with other SQLite writers.
func (c *Catalog) withAuthorizedSiteWrite(ctx context.Context, tenantID, requester string, proof *auth.SessionReadProof, mutate func(*sql.Conn) error) error {
	if ctx == nil || c == nil || c.db == nil || requester == "" {
		return ErrCatalogUnavailable
	}
	if proof != nil && !proof.MatchesDatabase(c.db) {
		return ErrCatalogUnavailable
	}
	conn, err := c.db.Conn(ctx)
	if err != nil {
		return fmt.Errorf("open site mutation connection: %w", err)
	}
	defer conn.Close()
	restoreBusyTimeout, err := beginSiteWrite(ctx, conn)
	if err != nil {
		return fmt.Errorf("begin guarded site mutation: %w", err)
	}
	defer restoreBusyTimeout()
	defer func() { _, _ = conn.ExecContext(context.Background(), "ROLLBACK") }()
	recordExpired := func(expiry string, now, expires time.Time) error {
		if _, err := conn.ExecContext(context.Background(), "ROLLBACK"); err != nil {
			return fmt.Errorf("rollback expired site mutation: %w", err)
		}
		restoreBusyTimeout()
		if err := conn.Close(); err != nil {
			return fmt.Errorf("release expired site mutation connection: %w", err)
		}
		if err := proof.RevokeObservedExpiry(ctx, expiry, now, !now.Before(expires)); err != nil {
			return fmt.Errorf("record expired site session: %w", err)
		}
		return ErrSiteSessionDenied
	}
	var role string
	err = conn.QueryRowContext(ctx, `SELECT m.role FROM memberships m
		JOIN tenants t ON t.id=m.tenant_id AND t.status='active'
		JOIN identities i ON i.id=m.identity_id AND i.disabled=0
		WHERE m.tenant_id=? AND m.identity_id=?`, tenantID, requester).Scan(&role)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrSiteReadDenied
	}
	if err != nil {
		return fmt.Errorf("authorize site mutation: %w", err)
	}
	if !(rbac.Principal{UserID: requester, TenantID: tenantID, Role: rbac.Role(role)}).Can("sites:write", tenantID) {
		return ErrSiteReadDenied
	}
	var expiry string
	var deadline int64
	var expires time.Time
	if proof != nil {
		digest := proof.Digest()
		err = conn.QueryRowContext(ctx, `SELECT expires_at,idle_deadline_unix FROM sessions
			WHERE token_sha256=? AND tenant_id=? AND identity_id=? AND revoked_at IS NULL`, digest[:], tenantID, requester).Scan(&expiry, &deadline)
		if errors.Is(err, sql.ErrNoRows) {
			return ErrSiteSessionDenied
		}
		if err != nil {
			return fmt.Errorf("read site mutation session: %w", err)
		}
		expires, err = time.Parse(time.RFC3339Nano, expiry)
		if err != nil {
			return ErrSiteSessionDenied
		}
		if now := proof.Now(); !now.Before(expires) || deadline <= now.Unix() {
			return recordExpired(expiry, now, expires)
		}
	}
	if err := mutate(conn); err != nil {
		return err
	}
	if proof != nil {
		if now := proof.Now(); !now.Before(expires) || deadline <= now.Unix() {
			return recordExpired(expiry, now, expires)
		}
	}
	if _, err := conn.ExecContext(ctx, "COMMIT"); err != nil {
		return fmt.Errorf("commit guarded site mutation: %w", err)
	}
	return nil
}

func beginSiteWrite(ctx context.Context, conn *sql.Conn) (func(), error) {
	return database.BeginImmediate(ctx, conn, 5*time.Second)
}
