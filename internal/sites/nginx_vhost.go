package sites

import (
	"errors"
	"fmt"
	"path"
	"regexp"
	"strings"
)

var (
	ErrInvalidHostingPaths = errors.New("invalid hosting paths")
	ErrSiteNotReady        = errors.New("site domains are not verified")
)

type HostingPaths struct {
	TenantRoot    string
	PHPSocketRoot string
}

var safeAbsolutePath = regexp.MustCompile(`^/[A-Za-z0-9._/-]+$`)

// RenderNginxVhost returns deterministic HTTP virtual-host configuration for a
// verified site. It only renders text; it does not write files, reload Nginx,
// provision PHP-FPM, or enable the result.
func RenderNginxVhost(site SiteRecord, paths HostingPaths) (string, error) {
	if !validID(site.TenantID) || !validID(site.SiteID) {
		return "", ErrInvalidSpec
	}
	if !safeAbsolutePath.MatchString(paths.TenantRoot) || path.Clean(paths.TenantRoot) != paths.TenantRoot ||
		paths.TenantRoot == "/" || strings.Contains(paths.TenantRoot, "//") ||
		!safeAbsolutePath.MatchString(paths.PHPSocketRoot) || path.Clean(paths.PHPSocketRoot) != paths.PHPSocketRoot ||
		paths.PHPSocketRoot == "/" || strings.Contains(paths.PHPSocketRoot, "//") {
		return "", ErrInvalidHostingPaths
	}
	if site.DocumentRoot != path.Join("sites", site.SiteID, "public") {
		return "", ErrInvalidSpec
	}
	if len(site.Domains) == 0 || len(site.Domains) > MaxAliases+1 {
		return "", ErrSiteNotReady
	}
	domains := make([]string, 0, len(site.Domains))
	seen := make(map[string]struct{}, len(site.Domains))
	primaryCount := 0
	for _, claim := range site.Domains {
		domain, err := NormalizeDomain(claim.Name)
		if err != nil || domain != claim.Name {
			return "", ErrInvalidSpec
		}
		if claim.OwnershipState != "verified" {
			return "", ErrSiteNotReady
		}
		if _, exists := seen[domain]; exists {
			return "", ErrInvalidSpec
		}
		seen[domain] = struct{}{}
		switch claim.Kind {
		case "primary":
			primaryCount++
		case "alias":
		default:
			return "", ErrInvalidSpec
		}
		domains = append(domains, domain)
	}
	if primaryCount != 1 {
		return "", ErrInvalidSpec
	}
	root := path.Join(paths.TenantRoot, "tenants", site.TenantID, site.DocumentRoot)
	socket := path.Join(paths.PHPSocketRoot, site.TenantID, site.SiteID+".sock")
	var b strings.Builder
	fmt.Fprintf(&b, "server {\n    listen 80;\n    server_name %s;\n    root %s;\n    index index.php index.html;\n\n", strings.Join(domains, " "), root)
	b.WriteString("    location ^~ /.well-known/acme-challenge/ {\n        try_files $uri =404;\n    }\n\n")
	b.WriteString("    location ~ /\\. {\n        deny all;\n    }\n\n")
	b.WriteString("    location / {\n        try_files $uri $uri/ /index.php?$query_string;\n    }\n\n")
	fmt.Fprintf(&b, "    location ~ \\.php$ {\n        try_files $uri =404;\n        include /etc/nginx/fastcgi_params;\n        fastcgi_param SCRIPT_FILENAME $document_root$fastcgi_script_name;\n        fastcgi_param HTTP_PROXY \"\";\n        fastcgi_pass unix:%s;\n    }\n", socket)
	b.WriteString("}\n")
	return b.String(), nil
}
