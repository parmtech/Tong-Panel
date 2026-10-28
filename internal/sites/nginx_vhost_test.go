package sites

import (
	"errors"
	"os"
	"strings"
	"testing"
)

func TestRenderNginxVhostRequiresVerifiedCanonicalDomains(t *testing.T) {
	site := SiteRecord{
		TenantID: "tenant-a", SiteID: "site-a", DocumentRoot: "sites/site-a/public",
		Domains: []DomainClaim{
			{Name: "example.com", Kind: "primary", OwnershipState: "verified"},
			{Name: "www.example.com", Kind: "alias", OwnershipState: "verified"},
		},
	}
	config, err := RenderNginxVhost(site, HostingPaths{TenantRoot: "/srv/tong", PHPSocketRoot: "/run/tong/php"})
	if err != nil {
		t.Fatal(err)
	}
	if output := os.Getenv("TONG_NGINX_PREVIEW_PATH"); output != "" {
		if err := os.WriteFile(output, []byte(config), 0600); err != nil {
			t.Fatalf("write nginx config preview for clean-VM syntax check: %v", err)
		}
	}
	for _, expected := range []string{
		"server_name example.com www.example.com;",
		"root /srv/tong/tenants/tenant-a/sites/site-a/public;",
		"fastcgi_pass unix:/run/tong/php/tenant-a/site-a.sock;",
		"location ^~ /.well-known/acme-challenge/",
		"location ~ /\\. {",
		"try_files $uri $uri/ /index.php?$query_string;",
		"fastcgi_param HTTP_PROXY \"\";",
	} {
		if !strings.Contains(config, expected) {
			t.Errorf("rendered config missing %q:\n%s", expected, config)
		}
	}
	if strings.Contains(config, "listen 443") || strings.Contains(config, "ssl_certificate") {
		t.Fatal("renderer implied TLS before certificate lifecycle exists")
	}
}

func TestRenderNginxVhostRejectsUnverifiedAndUnsafeRecords(t *testing.T) {
	base := SiteRecord{
		TenantID: "tenant-a", SiteID: "site-a", DocumentRoot: "sites/site-a/public",
		Domains: []DomainClaim{{Name: "example.com", Kind: "primary", OwnershipState: "verified"}},
	}
	for _, tc := range []struct {
		name string
		edit func(*SiteRecord, *HostingPaths)
		want error
	}{
		{"unverified", func(s *SiteRecord, _ *HostingPaths) { s.Domains[0].OwnershipState = "pending" }, ErrSiteNotReady},
		{"injected domain", func(s *SiteRecord, _ *HostingPaths) { s.Domains[0].Name = "example.com; include /tmp/evil;" }, ErrInvalidSpec},
		{"duplicate primary", func(s *SiteRecord, _ *HostingPaths) { s.Domains = append(s.Domains, s.Domains[0]) }, ErrInvalidSpec},
		{"no primary", func(s *SiteRecord, _ *HostingPaths) { s.Domains[0].Kind = "alias" }, ErrInvalidSpec},
		{"invalid root path", func(s *SiteRecord, p *HostingPaths) { p.TenantRoot = "/srv/tong; include /tmp/evil;" }, ErrInvalidHostingPaths},
		{"traversal root", func(s *SiteRecord, p *HostingPaths) { p.TenantRoot = "/srv/../etc" }, ErrInvalidHostingPaths},
		{"injected socket path", func(_ *SiteRecord, p *HostingPaths) { p.PHPSocketRoot = "/run/php;}" }, ErrInvalidHostingPaths},
		{"wrong document root", func(s *SiteRecord, _ *HostingPaths) { s.DocumentRoot = "sites/other/public" }, ErrInvalidSpec},
		{"invalid identifier", func(s *SiteRecord, _ *HostingPaths) { s.TenantID = "../tenant" }, ErrInvalidSpec},
	} {
		t.Run(tc.name, func(t *testing.T) {
			site := base
			site.Domains = append([]DomainClaim(nil), base.Domains...)
			paths := HostingPaths{TenantRoot: "/srv/tong", PHPSocketRoot: "/run/tong/php"}
			tc.edit(&site, &paths)
			if _, err := RenderNginxVhost(site, paths); !errors.Is(err, tc.want) {
				t.Fatalf("RenderNginxVhost error=%v, want %v", err, tc.want)
			}
		})
	}
}
