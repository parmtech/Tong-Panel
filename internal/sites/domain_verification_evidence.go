package sites

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/parmtech/tong-panel/internal/auth"
)

// ErrDomainEvidenceUnavailable covers missing, invalid, future or stale proof
// evidence without disclosing any domain or receipt from a rejected snapshot.
var ErrDomainEvidenceUnavailable = errors.New("fresh domain verification evidence unavailable")

// VerifiedDomainEvidence records one completed DNS proof. VerificationID is
// random metadata identifying that observation, not a token or authorization.
// Timestamps have UTC Unix-second precision; ObservedAt records DNS observation
// and VerifiedAt records the subsequent serialized catalog update.
type VerifiedDomainEvidence struct {
	Domain         string
	ObservedAt     time.Time
	VerifiedAt     time.Time
	VerificationID string
}

// DomainVerificationEvidence is a detached informational view. It does not
// grant certificate-operation permission, establish current DNS control, or
// fence future execution. Callers must recheck current evidence and authority
// at the point of any separately approved operation.
type DomainVerificationEvidence struct {
	TenantID string
	SiteID   string
	Domains  []VerifiedDomainEvidence
}

// DomainVerificationEvidenceFor returns only fresh recorded evidence under the
// existing sites:read membership/session policy. There is deliberately no
// production max-age default: a trusted caller must supply a positive policy.
// Membership, session and evidence are read in one SQLite statement snapshot.
// The session proof's trusted clock governs session expiry; the catalog's
// trusted clock governs evidence age. A clock rollback fails closed.
func (c *Catalog) DomainVerificationEvidenceFor(ctx context.Context, tenantID, siteID, requester string, proof *auth.SessionReadProof, maxAge time.Duration) (DomainVerificationEvidence, error) {
	if !validID(tenantID) || !validID(siteID) || maxAge <= 0 {
		return DomainVerificationEvidence{}, ErrInvalidSpec
	}
	detail := `SELECT s.tenant_id,s.site_id,s.document_root,s.created_at,d.domain,d.kind,d.ownership_state,
		d.ownership_observed_at,d.ownership_verified_at,d.ownership_verification_id
		FROM sites s LEFT JOIN site_domains d ON d.tenant_id=s.tenant_id AND d.site_id=s.site_id
		WHERE s.tenant_id=? AND s.site_id=? ORDER BY CASE d.kind WHEN 'primary' THEN 0 ELSE 1 END,d.domain`
	rows, authz, err := c.authorizedReadRows(ctx, tenantID, requester, proof, detail, tenantID, siteID)
	if err != nil {
		return DomainVerificationEvidence{}, err
	}
	defer rows.Close()
	var record SiteRecord
	found := false
	type storedEvidence struct {
		observed, verified sql.NullInt64
		id                 sql.NullString
	}
	evidence := make(map[string]storedEvidence)
	for rows.Next() {
		var expiry sql.NullString
		var idle int64
		var recordTenant, recordSite, root, domain, kind, state sql.NullString
		var created sql.NullInt64
		var item storedEvidence
		if err := rows.Scan(&expiry, &idle, &recordTenant, &recordSite, &root, &created, &domain, &kind, &state,
			&item.observed, &item.verified, &item.id); err != nil {
			return DomainVerificationEvidence{}, fmt.Errorf("scan domain evidence: %w", err)
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
			evidence[domain.String] = item
		}
	}
	if err := rows.Err(); err != nil {
		return DomainVerificationEvidence{}, fmt.Errorf("iterate domain evidence: %w", err)
	}
	if err := rows.Close(); err != nil {
		return DomainVerificationEvidence{}, fmt.Errorf("close domain evidence: %w", err)
	}
	if err := authz.finish(ctx); err != nil {
		return DomainVerificationEvidence{}, err
	}
	if !found {
		return DomainVerificationEvidence{}, ErrSiteNotFound
	}
	domains, err := verifiedSiteDomains(record)
	if err != nil {
		return DomainVerificationEvidence{}, err
	}
	now := c.now().UTC()
	result := DomainVerificationEvidence{TenantID: record.TenantID, SiteID: record.SiteID}
	for _, domain := range domains {
		item := evidence[domain]
		if !item.observed.Valid || !item.verified.Valid || !item.id.Valid ||
			item.observed.Int64 <= 0 || item.observed.Int64 > item.verified.Int64 || item.verified.Int64 > now.Unix() ||
			!validVerificationID(item.id.String) {
			return DomainVerificationEvidence{}, ErrDomainEvidenceUnavailable
		}
		observed := time.Unix(item.observed.Int64, 0).UTC()
		if now.Sub(observed) >= maxAge {
			return DomainVerificationEvidence{}, ErrDomainEvidenceUnavailable
		}
		result.Domains = append(result.Domains, VerifiedDomainEvidence{
			Domain: domain, ObservedAt: observed, VerifiedAt: time.Unix(item.verified.Int64, 0).UTC(), VerificationID: item.id.String,
		})
	}
	if err := ctx.Err(); err != nil {
		return DomainVerificationEvidence{}, err
	}
	return result, nil
}

func validVerificationID(value string) bool {
	if len(value) != 32 {
		return false
	}
	for _, ch := range value {
		if (ch < '0' || ch > '9') && (ch < 'a' || ch > 'f') {
			return false
		}
	}
	return true
}

// completeDomainProof runs only with the final SQLite writer reservation held.
// DNS observation time was sampled outside the lock; waiting must not make an
// old observation appear new. Token expiry is checked against the final writer
// transaction's verification time.
func (c *Catalog) completeDomainProof(ctx context.Context, conn *sql.Conn, tenant, siteID, domain, digest string, expiry int64, observedAt time.Time) error {
	verifiedAt := c.now().UTC()
	if observedAt.Unix() <= 0 || observedAt.After(verifiedAt) {
		return ErrDomainEvidenceUnavailable
	}
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return fmt.Errorf("generate domain verification ID: %w", err)
	}
	result, err := conn.ExecContext(ctx, `UPDATE site_domains
		SET ownership_state='verified',verification_token_sha256=NULL,verification_expires_at=NULL,
		    ownership_observed_at=?,ownership_verified_at=?,ownership_verification_id=?
		WHERE domain=? AND tenant_id=? AND site_id=? AND ownership_state='pending'
		    AND verification_token_sha256=? AND verification_expires_at>? AND verification_expires_at=?`,
		observedAt.Unix(), verifiedAt.Unix(), hex.EncodeToString(raw[:]), domain, tenant, siteID, digest, verifiedAt.Unix(), expiry)
	if err != nil {
		return fmt.Errorf("record domain verification evidence: %w", err)
	}
	n, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read domain evidence update result: %w", err)
	}
	if n != 1 {
		return ErrDomainVerificationExpired
	}
	return nil
}
