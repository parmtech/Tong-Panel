package sites

import (
	"errors"
	"fmt"
	"path"
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

// RenderNginxVhost returns deterministic HTTP virtual-host configuration for a
// verified site. It only renders text; it does not write files, reload Nginx,
// provision PHP-FPM, or enable the result.
func RenderNginxVhost(site SiteRecord, paths HostingPaths) (string, error) {
	domains, err := verifiedSiteDomains(site)
	if err != nil {
		return "", err
	}
	if err := validateHostingPaths(paths); err != nil {
		return "", err
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
