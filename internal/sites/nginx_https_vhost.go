package sites

import (
	"errors"
	"fmt"
	"path"
	"strings"
)

var ErrInvalidTLSFiles = errors.New("invalid site TLS certificate paths")

// NginxTLSFiles identifies the only certificate and private-key paths accepted
// for a site. The files must already exist under the service-owned tenant/site
// directory; this renderer neither requests certificates nor writes files.
type NginxTLSFiles struct {
	CertificateFile string
	PrivateKeyFile  string
}

// RenderNginxHTTPSVhost renders a verified-domain HTTP-to-HTTPS vhost and its TLS
// server block. It only returns text; it does not install files or reload Nginx.
func RenderNginxHTTPSVhost(site SiteRecord, paths HostingPaths, tls NginxTLSFiles) (string, error) {
	domains, err := verifiedSiteDomains(site)
	if err != nil {
		return "", err
	}
	if err := validateHostingPaths(paths); err != nil {
		return "", err
	}
	certDir := path.Join("/etc/tong/certs", site.TenantID, site.SiteID)
	if tls.CertificateFile != path.Join(certDir, "fullchain.pem") ||
		tls.PrivateKeyFile != path.Join(certDir, "privkey.pem") {
		return "", ErrInvalidTLSFiles
	}

	root := path.Join(paths.TenantRoot, "tenants", site.TenantID, site.DocumentRoot)
	socket := path.Join(paths.PHPSocketRoot, site.TenantID, site.SiteID+".sock")
	serverNames := strings.Join(domains, " ")
	canonical := domains[0]

	var b strings.Builder
	fmt.Fprintf(&b, "server {\n    listen 80;\n    server_name %s;\n    root %s;\n", serverNames, root)
	b.WriteString("    location ^~ /.well-known/acme-challenge/ {\n        try_files $uri =404;\n    }\n")
	fmt.Fprintf(&b, "    location / {\n        return 308 https://%s$request_uri;\n    }\n}\n\n", canonical)

	fmt.Fprintf(&b, "server {\n    listen 443 ssl;\n    server_name %s;\n", serverNames)
	fmt.Fprintf(&b, "    ssl_certificate %s;\n    ssl_certificate_key %s;\n", tls.CertificateFile, tls.PrivateKeyFile)
	b.WriteString("    ssl_protocols TLSv1.2 TLSv1.3;\n    ssl_session_tickets off;\n")
	fmt.Fprintf(&b, "    root %s;\n    index index.php index.html;\n\n", root)
	b.WriteString("    location ~ /\\. {\n        deny all;\n    }\n\n")
	b.WriteString("    location / {\n        try_files $uri $uri/ /index.php?$query_string;\n    }\n\n")
	fmt.Fprintf(&b, "    location ~ \\.php$ {\n        try_files $uri =404;\n        include /etc/nginx/fastcgi_params;\n        fastcgi_param SCRIPT_FILENAME $document_root$fastcgi_script_name;\n        fastcgi_param HTTP_PROXY \"\";\n        fastcgi_pass unix:%s;\n    }\n}\n", socket)
	return b.String(), nil
}
