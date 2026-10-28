package sites

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteTwoTenantSitePreviewsForCleanVM(t *testing.T) {
	out := os.Getenv("TONG_TWO_TENANT_SITE_PREVIEW_DIR")
	if out == "" {
		return
	}
	paths := HostingPaths{TenantRoot: "/srv/tong", PHPSocketRoot: "/run/tong/php"}
	sites := []SiteRecord{
		{
			TenantID: "tenant-a", SiteID: "site-a", DocumentRoot: "sites/site-a/public",
			Domains: []DomainClaim{{Name: "tenant-a.example.test", Kind: "primary", OwnershipState: "verified"}},
		},
		{
			TenantID: "tenant-b", SiteID: "site-b", DocumentRoot: "sites/site-b/public",
			Domains: []DomainClaim{{Name: "tenant-b.example.test", Kind: "primary", OwnershipState: "verified"}},
		},
	}
	var nginx, pools []string
	for _, site := range sites {
		nginxConfig, err := RenderNginxVhost(site, paths)
		if err != nil {
			t.Fatalf("render Nginx vhost for %s: %v", site.TenantID, err)
		}
		poolConfig, err := RenderPHPFPMPool(site, paths)
		if err != nil {
			t.Fatalf("render PHP-FPM pool for %s: %v", site.TenantID, err)
		}
		nginx = append(nginx, nginxConfig)
		pools = append(pools, poolConfig)
	}
	if err := os.MkdirAll(out, 0700); err != nil {
		t.Fatal(err)
	}
	for name, data := range map[string]string{
		"nginx-sites.conf": strings.Join(nginx, "\n"),
		"php-fpm-pools.conf": strings.Join(pools, "\n"),
	} {
		if err := os.WriteFile(filepath.Join(out, name), []byte(data), 0600); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
}
