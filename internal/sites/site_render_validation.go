package sites

import (
	"path"
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
	domains := make([]string, 0, len(site.Domains))
	seen := make(map[string]struct{}, len(site.Domains))
	primaryCount := 0
	for _, claim := range site.Domains {
		domain, err := NormalizeDomain(claim.Name)
		if err != nil || domain != claim.Name {
			return nil, ErrInvalidSpec
		}
		if claim.OwnershipState != "verified" {
			return nil, ErrSiteNotReady
		}
		if _, exists := seen[domain]; exists {
			return nil, ErrInvalidSpec
		}
		seen[domain] = struct{}{}
		switch claim.Kind {
		case "primary":
			primaryCount++
		case "alias":
		default:
			return nil, ErrInvalidSpec
		}
		domains = append(domains, domain)
	}
	if primaryCount != 1 {
		return nil, ErrInvalidSpec
	}
	return domains, nil
}
