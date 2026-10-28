package sites

import (
	"errors"
	"os"
	"strings"
	"testing"
)

func TestRenderNginxHTTPSVhostRequiresVerifiedDomainsAndCanonicalRedirect(t *testing.T) {
	site := SiteRecord{
		TenantID: "tenant-a", SiteID: "site-a", DocumentRoot: "sites/site-a/public",
		Domains: []DomainClaim{
			{Name: "tenant-a.example.test", Kind: "primary", OwnershipState: "verified"},
			{Name: "www.tenant-a.example.test", Kind: "alias", OwnershipState: "verified"},
		},
	}
	paths := HostingPaths{TenantRoot: "/srv/tong", PHPSocketRoot: "/run/tong/php"}
	tls := NginxTLSFiles{
		CertificateFile: "/etc/tong/certs/tenant-a/site-a/fullchain.pem",
		PrivateKeyFile:  "/etc/tong/certs/tenant-a/site-a/privkey.pem",
	}
	configA, err := RenderNginxHTTPSVhost(site, paths, tls)
	if err != nil {
		t.Fatal(err)
	}
	siteB := SiteRecord{
		TenantID: "tenant-b", SiteID: "site-b", DocumentRoot: "sites/site-b/public",
		Domains: []DomainClaim{{Name: "tenant-b.example.test", Kind: "primary", OwnershipState: "verified"}},
	}
	tlsB := NginxTLSFiles{
		CertificateFile: "/etc/tong/certs/tenant-b/site-b/fullchain.pem",
		PrivateKeyFile:  "/etc/tong/certs/tenant-b/site-b/privkey.pem",
	}
	configB, err := RenderNginxHTTPSVhost(siteB, paths, tlsB)
	if err != nil {
		t.Fatal(err)
	}
	config := configA + "\n" + configB
httpBlockEnd := strings.Index(configA, "\n}\n\n")
if httpBlockEnd < 0 || !strings.Contains(configA[:httpBlockEnd], "root /srv/tong/tenants/tenant-a/sites/site-a/public;") {
	t.Fatalf("HTTP ACME server does not serve from the verified tenant/site root:\n%s", configA)
}
	if output := os.Getenv("TONG_NGINX_HTTPS_PREVIEW_PATH"); output != "" {
		if err := os.WriteFile(output, []byte(config), 0600); err != nil {
			t.Fatalf("write HTTPS vhost preview for clean-VM syntax check: %v", err)
		}
	}
	for _, expected := range []string{
		"listen 80;",
		"server_name tenant-a.example.test www.tenant-a.example.test;",
		"return 308 https://tenant-a.example.test$request_uri;",
		"location ^~ /.well-known/acme-challenge/",
		"listen 443 ssl;",
		"ssl_certificate /etc/tong/certs/tenant-a/site-a/fullchain.pem;",
		"ssl_certificate_key /etc/tong/certs/tenant-a/site-a/privkey.pem;",
		"ssl_protocols TLSv1.2 TLSv1.3;",
		"root /srv/tong/tenants/tenant-a/sites/site-a/public;",
		"fastcgi_pass unix:/run/tong/php/tenant-a/site-a.sock;",
		"location ~ /\\. {",
	} {
		if !strings.Contains(config, expected) {
			t.Errorf("rendered HTTPS config missing %q:\n%s", expected, config)
		}
	}
}

func TestRenderNginxHTTPSVhostRejectsUnsafeInputs(t *testing.T) {
	base := SiteRecord{
		TenantID: "tenant-a", SiteID: "site-a", DocumentRoot: "sites/site-a/public",
		Domains: []DomainClaim{{Name: "example.com", Kind: "primary", OwnershipState: "verified"}},
	}
	baseTLS := NginxTLSFiles{
		CertificateFile: "/etc/tong/certs/tenant-a/site-a/fullchain.pem",
		PrivateKeyFile:  "/etc/tong/certs/tenant-a/site-a/privkey.pem",
	}
	for _, tc := range []struct {
		name string
		edit func(*SiteRecord, *HostingPaths, *NginxTLSFiles)
		want error
	}{
		{"unverified domain", func(s *SiteRecord, _ *HostingPaths, _ *NginxTLSFiles) { s.Domains[0].OwnershipState = "pending" }, ErrSiteNotReady},
		{"injected domain", func(s *SiteRecord, _ *HostingPaths, _ *NginxTLSFiles) {
			s.Domains[0].Name = "example.com; include /tmp/evil;"
		}, ErrInvalidSpec},
		{"TLS path traversal", func(_ *SiteRecord, _ *HostingPaths, tls *NginxTLSFiles) {
			tls.PrivateKeyFile = "/etc/tong/certs/tenant-a/../privkey.pem"
		}, ErrInvalidTLSFiles},
		{"cross-tenant certificate", func(_ *SiteRecord, _ *HostingPaths, tls *NginxTLSFiles) {
			tls.CertificateFile = "/etc/tong/certs/tenant-b/site-a/fullchain.pem"
		}, ErrInvalidTLSFiles},
		{"certificate directive injection", func(_ *SiteRecord, _ *HostingPaths, tls *NginxTLSFiles) {
			tls.CertificateFile += "; include /tmp/evil;"
		}, ErrInvalidTLSFiles},
		{"unsafe tenant root", func(_ *SiteRecord, p *HostingPaths, _ *NginxTLSFiles) { p.TenantRoot = "/srv/tong; include /tmp/evil;" }, ErrInvalidHostingPaths},
	} {
		t.Run(tc.name, func(t *testing.T) {
			site := base
			site.Domains = append([]DomainClaim(nil), base.Domains...)
			paths := HostingPaths{TenantRoot: "/srv/tong", PHPSocketRoot: "/run/tong/php"}
			tls := baseTLS
			tc.edit(&site, &paths, &tls)
			if _, err := RenderNginxHTTPSVhost(site, paths, tls); !errors.Is(err, tc.want) {
				t.Fatalf("RenderNginxHTTPSVhost error=%v, want %v", err, tc.want)
			}
		})
	}
}
