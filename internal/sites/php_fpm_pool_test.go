package sites

import (
	"errors"
	"os"
	"strings"
	"sync"
	"testing"
)

func phpFPMFixture() SiteRecord {
	return SiteRecord{
		TenantID: "tenant-a", SiteID: "site-a", DocumentRoot: "sites/site-a/public",
		Domains: []DomainClaim{{Name: "example.com", Kind: "primary", OwnershipState: "verified"}},
	}
}

func TestRenderPHPFPMPoolRequiresVerifiedIsolatedSite(t *testing.T) {
	site := phpFPMFixture()
	paths := HostingPaths{TenantRoot: "/srv/tong", PHPSocketRoot: "/run/tong/php"}
	config, err := RenderPHPFPMPool(site, paths)
	if err != nil {
		t.Fatal(err)
	}
	if preview := os.Getenv("TONG_PHP_FPM_PREVIEW_PATH"); preview != "" {
		if err := os.WriteFile(preview, []byte(config), 0600); err != nil {
			t.Fatalf("write PHP-FPM preview for clean-VM syntax check: %v", err)
		}
	}
	for _, expected := range []string{
		"[tp_", "user = tp_", "group = tp_", "listen = /run/tong/php/tenant-a/site-a.sock",
		"listen.owner = www-data", "listen.mode = 0660", "pm = ondemand",
		"pm.max_children = 5", "clear_env = yes", "security.limit_extensions = .php",
		"open_basedir = /srv/tong/tenants/tenant-a/sites/site-a/public:/srv/tong/tenants/tenant-a/run/site-a:/tmp",
		"upload_tmp_dir] = /srv/tong/tenants/tenant-a/run/site-a/tmp",
		"session.save_path] = /srv/tong/tenants/tenant-a/run/site-a/sessions",
		"request_terminate_timeout = 60s",
	} {
		if !strings.Contains(config, expected) {
			t.Errorf("pool config missing %q:\n%s", expected, config)
		}
	}
	if strings.Contains(config, "php_admin_value[disable_functions]") ||
		strings.Contains(config, "listen = 127.0.0.1") {
		t.Fatalf("unexpected unsafe or TCP listener in generated pool:\n%s", config)
	}
	again, err := RenderPHPFPMPool(site, paths)
	if err != nil || again != config {
		t.Fatalf("render must be stable: err=%v equal=%v", err, again == config)
	}
	other, err := RenderPHPFPMPool(SiteRecord{
		TenantID: "tenant-b", SiteID: "site-b", DocumentRoot: "sites/site-b/public",
		Domains: site.Domains,
	}, paths)
	if err != nil {
		t.Fatal(err)
	}
	if other == config || strings.Contains(other, "tenant-a") || strings.Contains(other, "site-a") {
		t.Fatalf("pool identity or paths were shared across tenants:\n%s", other)
	}
}

func TestRenderPHPFPMPoolRejectsUnverifiedAndInjectedInputs(t *testing.T) {
	base := phpFPMFixture()
	for _, tc := range []struct {
		name string
		edit func(*SiteRecord, *HostingPaths)
		want error
	}{
		{"pending domain", func(s *SiteRecord, _ *HostingPaths) { s.Domains[0].OwnershipState = "pending" }, ErrSiteNotReady},
		{"injected domain", func(s *SiteRecord, _ *HostingPaths) { s.Domains[0].Name = "example.com; include /tmp/evil;" }, ErrInvalidSpec},
		{"duplicate primary", func(s *SiteRecord, _ *HostingPaths) { s.Domains = append(s.Domains, s.Domains[0]) }, ErrInvalidSpec},
		{"missing primary", func(s *SiteRecord, _ *HostingPaths) { s.Domains[0].Kind = "alias" }, ErrInvalidSpec},
		{"wrong document root", func(s *SiteRecord, _ *HostingPaths) { s.DocumentRoot = "sites/other/public" }, ErrInvalidSpec},
		{"bad tenant identifier", func(s *SiteRecord, _ *HostingPaths) { s.TenantID = "../tenant" }, ErrInvalidSpec},
		{"root traversal", func(_ *SiteRecord, p *HostingPaths) { p.TenantRoot = "/srv/../etc" }, ErrInvalidHostingPaths},
		{"root injection", func(_ *SiteRecord, p *HostingPaths) { p.TenantRoot = "/srv/tong; include /tmp/evil;" }, ErrInvalidHostingPaths},
		{"socket injection", func(_ *SiteRecord, p *HostingPaths) { p.PHPSocketRoot = "/run/php;}" }, ErrInvalidHostingPaths},
	} {
		t.Run(tc.name, func(t *testing.T) {
			site := base
			site.Domains = append([]DomainClaim(nil), base.Domains...)
			paths := HostingPaths{TenantRoot: "/srv/tong", PHPSocketRoot: "/run/tong/php"}
			tc.edit(&site, &paths)
			if _, err := RenderPHPFPMPool(site, paths); !errors.Is(err, tc.want) {
				t.Fatalf("RenderPHPFPMPool error=%v, want %v", err, tc.want)
			}
		})
	}
}

func TestRenderPHPFPMPoolConcurrentTenantIsolation(t *testing.T) {
	const count = 32
	paths := HostingPaths{TenantRoot: "/srv/tong", PHPSocketRoot: "/run/tong/php"}
	var wg sync.WaitGroup
	errs := make(chan string, count)
	for i := 0; i < count; i++ {
		tenant := fmt.Sprintf("tenant-%02d", i)
		siteID := fmt.Sprintf("site-%02d", i)
		wg.Add(1)
		go func(tenant, siteID string) {
			defer wg.Done()
			site := SiteRecord{
				TenantID: tenant, SiteID: siteID, DocumentRoot: "sites/" + siteID + "/public",
				Domains: []DomainClaim{{Name: "example.com", Kind: "primary", OwnershipState: "verified"}},
			}
			config, err := RenderPHPFPMPool(site, paths)
			if err != nil || !strings.Contains(config, "/tenants/"+tenant+"/sites/"+siteID+"/public") ||
				!strings.Contains(config, "/"+tenant+"/"+siteID+".sock") ||
				strings.Contains(config, "tenant-a") || strings.Contains(config, "tenant-b") {
				errs <- fmt.Sprintf("%s/%s: err=%v config=%q", tenant, siteID, err, config)
			}
		}(tenant, siteID)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Error(err)
	}
}
