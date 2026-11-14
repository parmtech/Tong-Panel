package sites

import (
	"context"

	"github.com/parmtech/tong-panel/internal/auth"
)

// ACMEHTTP01Preflight is an informational snapshot, never an execution permit.
// It contains no CA URL, challenge token, key, path, job or issuance capability.
// A future executor must reauthorize/reload ownership at execution time.
type ACMEHTTP01Preflight struct {
	TenantID string
	SiteID   string
	Domains  []string
}

// ACMEHTTP01PreflightFor connects existing catalog authorization and verified
// domain selection to the input shape needed by a future HTTP-01 adapter.
// Pending aliases are excluded; a pending primary fails closed. This method
// neither constructs a provider nor queues or performs certificate operations.
// GetFor also enforces current session state when a proof is supplied.
func (c *Catalog) ACMEHTTP01PreflightFor(ctx context.Context, tenantID, siteID, requester string, proof *auth.SessionReadProof) (ACMEHTTP01Preflight, error) {
	record, err := c.GetFor(ctx, tenantID, siteID, requester, proof)
	if err != nil {
		return ACMEHTTP01Preflight{}, err
	}
	domains, err := verifiedSiteDomains(record)
	if err != nil {
		return ACMEHTTP01Preflight{}, err
	}
	return ACMEHTTP01Preflight{TenantID: record.TenantID, SiteID: record.SiteID, Domains: domains}, nil
}
