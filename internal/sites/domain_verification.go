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
	"net"
	"time"
)

const DomainProofLifetime = time.Hour
const domainProofLabel = "_tong-panel-challenge"

var (
	ErrDomainVerificationState   = errors.New("domain is not eligible for verification")
	ErrDomainVerificationExpired = errors.New("domain verification token expired or replaced")
	ErrDomainProofMismatch       = errors.New("domain DNS proof did not match")
	ErrDomainVerifierUnavailable = errors.New("domain DNS verifier unavailable")
)

// TXTLookup resolves DNS TXT records for a validated challenge name.
// Implementations must honor cancellation and return complete TXT values.
type TXTLookup interface {
	LookupTXT(ctx context.Context, name string) ([]string, error)
}

// NetTXTLookup uses the host's configured DNS resolver. It performs no HTTP
// requests and is not a domain-ownership or DNSSEC certification service.
type NetTXTLookup struct {
	Resolver *net.Resolver
}

func (l NetTXTLookup) LookupTXT(ctx context.Context, name string) ([]string, error) {
	resolver := l.Resolver
	if resolver == nil {
		resolver = net.DefaultResolver
	}
	return resolver.LookupTXT(ctx, name)
}

// DomainVerificationInstruction contains a one-time token. The caller must
// display/store it once and instruct the domain owner to publish it as the
// value of TXT record Name. The catalog stores only a SHA-256 digest.
type DomainVerificationInstruction struct {
	Name      string
	Value     string
	ExpiresAt int64
}

func (c *Catalog) BeginDomainVerification(ctx context.Context, tenantID, siteID, rawDomain string) (DomainVerificationInstruction, error) {
	if ctx == nil || c == nil || c.db == nil || !validID(tenantID) || !validID(siteID) {
		return DomainVerificationInstruction{}, ErrCatalogUnavailable
	}
	domain, err := NormalizeDomain(rawDomain)
	if err != nil {
		return DomainVerificationInstruction{}, err
	}
	var state string
	if err := c.db.QueryRowContext(ctx,
		"SELECT ownership_state FROM site_domains WHERE domain=? AND tenant_id=? AND site_id=?",
		domain, tenantID, siteID).Scan(&state); errors.Is(err, sql.ErrNoRows) {
		return DomainVerificationInstruction{}, ErrSiteNotFound
	} else if err != nil {
		return DomainVerificationInstruction{}, fmt.Errorf("read domain verification state: %w", err)
	}
	if state != "pending" {
		return DomainVerificationInstruction{}, ErrDomainVerificationState
	}
	var secret [32]byte
	if _, err := rand.Read(secret[:]); err != nil {
		return DomainVerificationInstruction{}, fmt.Errorf("generate domain proof token: %w", err)
	}
	value := base64.RawURLEncoding.EncodeToString(secret[:])
	digest := sha256.Sum256([]byte(value))
	digestHex := hex.EncodeToString(digest[:])
	now := c.now()
	expires := now.Add(DomainProofLifetime).Unix()
	result, err := c.db.ExecContext(ctx,
		"UPDATE site_domains SET verification_token_sha256=?, verification_expires_at=? WHERE domain=? AND tenant_id=? AND site_id=? AND ownership_state='pending'",
		digestHex, expires, domain, tenantID, siteID)
	if err != nil {
		return DomainVerificationInstruction{}, fmt.Errorf("store domain proof digest: %w", err)
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return DomainVerificationInstruction{}, fmt.Errorf("read domain proof update result: %w", err)
	}
	if changed != 1 {
		return DomainVerificationInstruction{}, ErrDomainVerificationState
	}
	return DomainVerificationInstruction{Name: domainProofLabel + "." + domain, Value: value, ExpiresAt: expires}, nil
}

// CompleteDomainVerification confirms a short-lived token only after the exact
// value is visible at its tenant-specific DNS TXT challenge name. It rechecks
// the digest and expiry during the final update to reject token replacement and
// concurrent replay. No API route or certificate action is wired to this method.
func (c *Catalog) CompleteDomainVerification(ctx context.Context, tenantID, siteID, rawDomain, value string, lookup TXTLookup) error {
	if ctx == nil || c == nil || c.db == nil || lookup == nil || !validID(tenantID) || !validID(siteID) || value == "" {
		return ErrDomainVerifierUnavailable
	}
	domain, err := NormalizeDomain(rawDomain)
	if err != nil {
		return err
	}
	var state, storedDigest string
	var expires sql.NullInt64
	err = c.db.QueryRowContext(ctx,
		"SELECT ownership_state,verification_token_sha256,verification_expires_at FROM site_domains WHERE domain=? AND tenant_id=? AND site_id=?",
		domain, tenantID, siteID).Scan(&state, &storedDigest, &expires)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrSiteNotFound
	}
	if err != nil {
		return fmt.Errorf("read domain proof state: %w", err)
	}
	if state != "pending" || storedDigest == "" || !expires.Valid {
		return ErrDomainVerificationState
	}
	now := c.now().Unix()
	if now >= expires.Int64 {
		return ErrDomainVerificationExpired
	}
	digest := sha256.Sum256([]byte(value))
	digestHex := hex.EncodeToString(digest[:])
	if subtle.ConstantTimeCompare([]byte(storedDigest), []byte(digestHex)) != 1 {
		return ErrDomainProofMismatch
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
	result, err := c.db.ExecContext(ctx,
		"UPDATE site_domains SET ownership_state='verified', verification_token_sha256=NULL, verification_expires_at=NULL WHERE domain=? AND tenant_id=? AND site_id=? AND ownership_state='pending' AND verification_token_sha256=? AND verification_expires_at>? AND verification_expires_at=?",
		domain, tenantID, siteID, storedDigest, c.now().Unix(), expires.Int64)
	if err != nil {
		return fmt.Errorf("commit domain verification: %w", err)
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read domain verification result: %w", err)
	}
	if changed != 1 {
		return ErrDomainVerificationExpired
	}
	return nil
}
