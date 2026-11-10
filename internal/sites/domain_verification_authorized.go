package sites

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/parmtech/tong-panel/internal/auth"
)

// BeginDomainVerificationFor atomically authorizes challenge replacement and audit.
func (c *Catalog) BeginDomainVerificationFor(ctx context.Context, tenant, siteID, rawDomain, requester string, proof *auth.SessionReadProof) (DomainVerificationInstruction, error) {
	if !validID(tenant) || !validID(siteID) {
		return DomainVerificationInstruction{}, ErrCatalogUnavailable
	}
	domain, err := NormalizeDomain(rawDomain)
	if err != nil {
		return DomainVerificationInstruction{}, err
	}
	var secret [32]byte
	if _, err := rand.Read(secret[:]); err != nil {
		return DomainVerificationInstruction{}, fmt.Errorf("generate domain proof: %w", err)
	}
	value := base64.RawURLEncoding.EncodeToString(secret[:])
	digest := sha256.Sum256([]byte(value))
	digestHex := hex.EncodeToString(digest[:])
	var expires int64
	err = c.withAuthorizedSiteWrite(ctx, tenant, requester, proof, func(conn *sql.Conn) error {
		var state string
		err := conn.QueryRowContext(ctx, "SELECT ownership_state FROM site_domains WHERE domain=? AND tenant_id=? AND site_id=?", domain, tenant, siteID).Scan(&state)
		if errors.Is(err, sql.ErrNoRows) {
			return ErrSiteNotFound
		}
		if err != nil {
			return err
		}
		if state != "pending" {
			return ErrDomainVerificationState
		}
		expires = c.now().Add(DomainProofLifetime).Unix()
		result, err := conn.ExecContext(ctx, "UPDATE site_domains SET verification_token_sha256=?,verification_expires_at=? WHERE domain=? AND tenant_id=? AND site_id=? AND ownership_state='pending'", digestHex, expires, domain, tenant, siteID)
		if err != nil {
			return err
		}
		n, err := result.RowsAffected()
		if err != nil {
			return err
		}
		if n != 1 {
			return ErrDomainVerificationState
		}
		return auditSiteMutation(ctx, conn, tenant, requester, siteID+"/"+domain, "site.domain_verification_started")
	})
	if err != nil {
		return DomainVerificationInstruction{}, err
	}
	return DomainVerificationInstruction{Name: domainProofLabel + "." + domain, Value: value, ExpiresAt: expires}, nil
}

// CompleteDomainVerificationFor authorizes a short state read, resolves DNS
// without a database lock, then reauthorizes and atomically verifies and audits.
func (c *Catalog) CompleteDomainVerificationFor(ctx context.Context, tenant, siteID, rawDomain, value, requester string, proof *auth.SessionReadProof, lookup TXTLookup) error {
	if !validID(tenant) || !validID(siteID) || value == "" || lookup == nil {
		return ErrDomainVerifierUnavailable
	}
	domain, err := NormalizeDomain(rawDomain)
	if err != nil {
		return err
	}
	var stored sql.NullString
	var expiry sql.NullInt64
	err = c.withAuthorizedSiteWrite(ctx, tenant, requester, proof, func(conn *sql.Conn) error {
		var state string
		err := conn.QueryRowContext(ctx, "SELECT ownership_state,verification_token_sha256,verification_expires_at FROM site_domains WHERE domain=? AND tenant_id=? AND site_id=?", domain, tenant, siteID).Scan(&state, &stored, &expiry)
		if errors.Is(err, sql.ErrNoRows) {
			return ErrSiteNotFound
		}
		if err != nil {
			return err
		}
		if state != "pending" || !stored.Valid || stored.String == "" || !expiry.Valid {
			return ErrDomainVerificationState
		}
		if c.now().Unix() >= expiry.Int64 {
			return ErrDomainVerificationExpired
		}
		digest := sha256.Sum256([]byte(value))
		if subtle.ConstantTimeCompare([]byte(stored.String), []byte(hex.EncodeToString(digest[:]))) != 1 {
			return ErrDomainProofMismatch
		}
		return nil
	})
	if err != nil {
		return err
	}
	records, err := lookup.LookupTXT(ctx, domainProofLabel+"."+domain)
	if err != nil {
		return fmt.Errorf("lookup domain proof TXT: %w", err)
	}
	found := false
	for _, record := range records {
		if subtle.ConstantTimeCompare([]byte(record), []byte(value)) == 1 {
			found = true
		}
	}
	if !found {
		return ErrDomainProofMismatch
	}
	return c.withAuthorizedSiteWrite(ctx, tenant, requester, proof, func(conn *sql.Conn) error {
		result, err := conn.ExecContext(ctx, "UPDATE site_domains SET ownership_state='verified',verification_token_sha256=NULL,verification_expires_at=NULL WHERE domain=? AND tenant_id=? AND site_id=? AND ownership_state='pending' AND verification_token_sha256=? AND verification_expires_at>? AND verification_expires_at=?", domain, tenant, siteID, stored.String, c.now().Unix(), expiry.Int64)
		if err != nil {
			return err
		}
		n, err := result.RowsAffected()
		if err != nil {
			return err
		}
		if n != 1 {
			return ErrDomainVerificationExpired
		}
		return auditSiteMutation(ctx, conn, tenant, requester, siteID+"/"+domain, "site.domain_verified")
	})
}
