// Package sites contains unprivileged validation for proposed hosted-site
// configuration. It does not claim domains, configure services, or issue TLS certificates.
package sites

import (
	"errors"
	"fmt"
	"net"
	"path"
	"strings"
)

const MaxAliases = 32

var (
	ErrInvalidDomain = errors.New("invalid site domain")
	ErrInvalidSpec   = errors.New("invalid site specification")
)

// SiteSpec is declarative metadata for a site. DocumentRoot is relative to the
// tenant's pinned site storage root and is constrained to sites/<site-id>/public.
type SiteSpec struct {
	TenantID      string
	SiteID        string
	PrimaryDomain string
	Aliases       []string
	DocumentRoot  string
}

// NormalizeDomain returns a lower-case ASCII DNS name suitable for exact
// ownership lookup. Unicode input is rejected; callers must canonicalize it with a vetted IDNA library first.
func NormalizeDomain(input string) (string, error) {
	domain := strings.TrimSuffix(input, ".")
	if domain == "" || len(domain) > 253 || strings.TrimSpace(input) != input {
		return "", ErrInvalidDomain
	}
	if strings.ContainsAny(domain, " /\\:\t\r\n\x00") || net.ParseIP(domain) != nil {
		return "", ErrInvalidDomain
	}
	domain = strings.ToLower(domain)
	labels := strings.Split(domain, ".")
	if len(labels) < 2 {
		return "", ErrInvalidDomain
	}
	for _, label := range labels {
		if len(label) == 0 || len(label) > 63 || label[0] == '-' || label[len(label)-1] == '-' {
			return "", ErrInvalidDomain
		}
		for _, ch := range label {
			if (ch < 'a' || ch > 'z') && (ch < '0' || ch > '9') && ch != '-' {
				return "", ErrInvalidDomain
			}
		}
	}
	return domain, nil
}

// Validate returns a normalized copy, rejecting ambiguous names and unsafe
// tenant-relative document roots before any future adapter sees the spec.
func (s SiteSpec) Validate() (SiteSpec, error) {
	if !validID(s.TenantID) || !validID(s.SiteID) {
		return SiteSpec{}, fmt.Errorf("%w: invalid tenant or site id", ErrInvalidSpec)
	}
	primary, err := NormalizeDomain(s.PrimaryDomain)
	if err != nil {
		return SiteSpec{}, fmt.Errorf("%w: primary domain: %v", ErrInvalidSpec, err)
	}
	if s.DocumentRoot != path.Join("sites", s.SiteID, "public") ||
		path.IsAbs(s.DocumentRoot) || strings.Contains(s.DocumentRoot, "\\") {
		return SiteSpec{}, fmt.Errorf("%w: document root must be sites/<site-id>/public", ErrInvalidSpec)
	}
	if len(s.Aliases) > MaxAliases {
		return SiteSpec{}, fmt.Errorf("%w: too many domain aliases", ErrInvalidSpec)
	}
	seen := map[string]struct{}{primary: {}}
	aliases := make([]string, 0, len(s.Aliases))
	for _, raw := range s.Aliases {
		alias, err := NormalizeDomain(raw)
		if err != nil {
			return SiteSpec{}, fmt.Errorf("%w: alias domain: %v", ErrInvalidSpec, err)
		}
		if _, exists := seen[alias]; exists {
			return SiteSpec{}, fmt.Errorf("%w: duplicate domain %q", ErrInvalidSpec, alias)
		}
		seen[alias] = struct{}{}
		aliases = append(aliases, alias)
	}
	s.PrimaryDomain = primary
	s.Aliases = aliases
	return s, nil
}

func validID(value string) bool {
	if value == "" || len(value) > 63 || strings.TrimSpace(value) != value ||
		value[0] == '-' || value[len(value)-1] == '-' {
		return false
	}
	for _, ch := range value {
		if (ch < 'a' || ch > 'z') && (ch < 'A' || ch > 'Z') &&
			(ch < '0' || ch > '9') && ch != '-' && ch != '_' {
			return false
		}
	}
	return true
}
