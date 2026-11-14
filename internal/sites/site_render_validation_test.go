package sites

import (
	"errors"
	"strings"
	"testing"
)

func TestRenderersIgnorePendingAliasesAndKeepPrimaryCanonical(t *testing.T) {
	site := SiteRecord{
		TenantID: "tenant-a", SiteID: "site-a", DocumentRoot: "sites/site-a/public",
		Domains: []DomainClaim{
			{Name: "zeta.example.test", Kind: "alias", OwnershipState: "verified"},
			{Name: "www.example.test", Kind: "alias", OwnershipState: "pending"},
			{Name: "example.test", Kind: "primary", OwnershipState: "verified"},
			{Name: "alpha.example.test", Kind: "alias", OwnershipState: "verified"},
		},
	}
	paths := HostingPaths{TenantRoot: "/srv/tong", PHPSocketRoot: "/run/tong/php"}
	wantNames := "server_name example.test alpha.example.test zeta.example.test;"

	httpConfig, err := RenderNginxVhost(site, paths)
	if err != nil {
		t.Fatalf("RenderNginxVhost with a pending alias: %v", err)
	}
	if !strings.Contains(httpConfig, wantNames) || strings.Contains(httpConfig, "www.example.test") {
		t.Fatalf("HTTP vhost did not preserve primary and omit pending alias:\n%s", httpConfig)
	}

	tls := NginxTLSFiles{
		CertificateFile: "/etc/tong/certs/tenant-a/site-a/fullchain.pem",
		PrivateKeyFile:  "/etc/tong/certs/tenant-a/site-a/privkey.pem",
	}
	httpsConfig, err := RenderNginxHTTPSVhost(site, paths, tls)
	if err != nil {
		t.Fatalf("RenderNginxHTTPSVhost with a pending alias: %v", err)
	}
	if !strings.Contains(httpsConfig, wantNames) || strings.Contains(httpsConfig, "www.example.test") ||
		!strings.Contains(httpsConfig, "return 308 https://example.test$request_uri;") {
		t.Fatalf("HTTPS vhost did not preserve primary, omit pending alias, and redirect canonically:\n%s", httpsConfig)
	}

	if _, err := RenderPHPFPMPool(site, paths); err != nil {
		t.Fatalf("RenderPHPFPMPool with a pending alias: %v", err)
	}
}

func TestRenderersRequireVerifiedPrimaryAndRejectMalformedPendingAliases(t *testing.T) {
	paths := HostingPaths{TenantRoot: "/srv/tong", PHPSocketRoot: "/run/tong/php"}
	tls := NginxTLSFiles{
		CertificateFile: "/etc/tong/certs/tenant-a/site-a/fullchain.pem",
		PrivateKeyFile:  "/etc/tong/certs/tenant-a/site-a/privkey.pem",
	}
	renderers := []struct {
		name string
		run  func(SiteRecord) error
	}{
		{"HTTP vhost", func(site SiteRecord) error { _, err := RenderNginxVhost(site, paths); return err }},
		{"HTTPS vhost", func(site SiteRecord) error { _, err := RenderNginxHTTPSVhost(site, paths, tls); return err }},
		{"PHP-FPM pool", func(site SiteRecord) error { _, err := RenderPHPFPMPool(site, paths); return err }},
	}
	base := SiteRecord{
		TenantID: "tenant-a", SiteID: "site-a", DocumentRoot: "sites/site-a/public",
		Domains: []DomainClaim{
			{Name: "example.test", Kind: "primary", OwnershipState: "verified"},
			{Name: "www.example.test", Kind: "alias", OwnershipState: "verified"},
		},
	}
	for _, tc := range []struct {
		name string
		edit func(*SiteRecord)
		want error
	}{
		{"pending primary", func(site *SiteRecord) { site.Domains[0].OwnershipState = "pending" }, ErrSiteNotReady},
		{"unsafe pending alias", func(site *SiteRecord) { site.Domains[1].Name = "www.example.test; include /tmp/evil;" }, ErrInvalidSpec},
		{"unknown alias state", func(site *SiteRecord) { site.Domains[1].OwnershipState = "removed" }, ErrInvalidSpec},
	} {
		t.Run(tc.name, func(t *testing.T) {
			site := base
			site.Domains = append([]DomainClaim(nil), base.Domains...)
			tc.edit(&site)
			for _, renderer := range renderers {
				t.Run(renderer.name, func(t *testing.T) {
					if err := renderer.run(site); !errors.Is(err, tc.want) {
						t.Fatalf("render error=%v, want %v", err, tc.want)
					}
				})
			}
		})
	}
}
