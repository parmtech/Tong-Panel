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
	if proof != nil && !proof.MatchesDatabase(c.db) {
		return SiteRecord{}, ErrCatalogUnavailable
	}
	conn, err := c.db.Conn(ctx)
	if err != nil {
		return SiteRecord{}, fmt.Errorf("open site creation connection: %w", err)
	}
	defer conn.Close()
	if _, err = conn.ExecContext(ctx, "BEGIN IMMEDIATE"); err != nil {
		return SiteRecord{}, fmt.Errorf("begin guarded site creation: %w", err)
	}
	defer func() { _, _ = conn.ExecContext(context.Background(), "ROLLBACK") }()
	recordExpired := func(expiry string, now, expires time.Time) error {
		if _, err := conn.ExecContext(context.Background(), "ROLLBACK"); err != nil {
			return fmt.Errorf("rollback expired site creation: %w", err)
		}
		if err := conn.Close(); err != nil {
			return fmt.Errorf("release expired site creation connection: %w", err)
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
		WHERE m.tenant_id=? AND m.identity_id=?`, spec.TenantID, requester).Scan(&role)
	if errors.Is(err, sql.ErrNoRows) {
		return SiteRecord{}, ErrSiteReadDenied
	}
	if err != nil {
		return SiteRecord{}, fmt.Errorf("authorize site creation: %w", err)
	}
	if !(rbac.Principal{UserID: requester, TenantID: spec.TenantID, Role: rbac.Role(role)}).Can("sites:write", spec.TenantID) {
		return SiteRecord{}, ErrSiteReadDenied
	}
	var expiry string
	var deadline int64
	var expires time.Time
	if proof != nil {
		digest := proof.Digest()
		err = conn.QueryRowContext(ctx, `SELECT expires_at,idle_deadline_unix FROM sessions
			WHERE token_sha256=? AND tenant_id=? AND identity_id=? AND revoked_at IS NULL`, digest[:], spec.TenantID, requester).Scan(&expiry, &deadline)
		if errors.Is(err, sql.ErrNoRows) {
			return SiteRecord{}, ErrSiteSessionDenied
		}
		if err != nil {
			return SiteRecord{}, fmt.Errorf("read site creation session: %w", err)
		}
		expires, err = time.Parse(time.RFC3339Nano, expiry)
		if err != nil {
			return SiteRecord{}, ErrSiteSessionDenied
		}
		if now := proof.Now(); !now.Before(expires) || deadline <= now.Unix() {
			return SiteRecord{}, recordExpired(expiry, now, expires)
		}
	}
	createdAt := c.now().Unix()
	if err := insertSite(ctx, conn, spec, createdAt); err != nil {
		return SiteRecord{}, err
	}
	var eventID [16]byte
	if _, err := rand.Read(eventID[:]); err != nil {
		return SiteRecord{}, fmt.Errorf("generate site audit ID: %w", err)
	}
	id := hex.EncodeToString(eventID[:])
	if _, err := conn.ExecContext(ctx, `INSERT INTO audit_events(id,tenant_id,actor_identity_id,actor_kind,action,subject_id,request_id)
		VALUES(?,?,?,'user','site.created',?,?)`, id, spec.TenantID, requester, spec.SiteID, id); err != nil {
		return SiteRecord{}, fmt.Errorf("audit site creation: %w", err)
	}
	if proof != nil {
		if now := proof.Now(); !now.Before(expires) || deadline <= now.Unix() {
			return SiteRecord{}, recordExpired(expiry, now, expires)
		}
	}
	if _, err := conn.ExecContext(ctx, "COMMIT"); err != nil {
		return SiteRecord{}, fmt.Errorf("commit guarded site creation: %w", err)
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
