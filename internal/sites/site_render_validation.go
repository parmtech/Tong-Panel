package sites

import (
	"path"
	"sort"
)

func verifiedSiteDomains(site SiteRecord) ([]string, error) {
	if !validID(site.TenantID) || !validID(site.SiteID) {
		return nil, ErrInvalidSpec
	}
	if site.DocumentRoot != path.Join("sites", site.SiteID, "public") {
		return nil, ErrInvalidSpec
	}
	if len(site.Domains) == 0 || len(site.Domains) > MaxAliases+1 {
		return nil, ErrSiteNotReady
	}
	aliases := make([]string, 0, len(site.Domains)-1)
	seen := make(map[string]struct{}, len(site.Domains))
	primary := ""
	for _, claim := range site.Domains {
		domain, err := NormalizeDomain(claim.Name)
		if err != nil || domain != claim.Name {
			return nil, ErrInvalidSpec
		}
		if _, exists := seen[domain]; exists {
			return nil, ErrInvalidSpec
		}
		seen[domain] = struct{}{}

		switch claim.Kind {
		case "primary":
			if primary != "" {
				return nil, ErrInvalidSpec
			}
			if claim.OwnershipState == "pending" {
				return nil, ErrSiteNotReady
			}
			if claim.OwnershipState != "verified" {
				return nil, ErrInvalidSpec
			}
			primary = domain
		case "alias":
			switch claim.OwnershipState {
			case "verified":
				aliases = append(aliases, domain)
			case "pending":
				// Pending claims are metadata only; never activate them in a vhost.
			default:
				return nil, ErrInvalidSpec
			}
		default:
			return nil, ErrInvalidSpec
		}
	}
	if primary == "" {
		return nil, ErrInvalidSpec
	}
	sort.Strings(aliases)
	domains := make([]string, 1, 1+len(aliases))
	domains[0] = primary
	return append(domains, aliases...), nil
}

