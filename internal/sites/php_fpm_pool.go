package sites

import (
	"crypto/sha256"
	"encoding/base32"
	"fmt"
	"path"
	"regexp"
	"strings"
)

var validHostingPath = regexp.MustCompile(`^/[A-Za-z0-9._/-]+$`)

func validateHostingPaths(paths HostingPaths) error {
	for _, candidate := range []string{paths.TenantRoot, paths.PHPSocketRoot} {
		if !validHostingPath.MatchString(candidate) || path.Clean(candidate) != candidate ||
			candidate == "/" || strings.Contains(candidate, "//") {
			return ErrInvalidHostingPaths
		}
	}
	return nil
}

// RenderPHPFPMPool returns deterministic per-site PHP-FPM pool configuration
// for a verified site. It renders text only and never creates users, directories,
// sockets, or service configuration on the host.
func RenderPHPFPMPool(site SiteRecord, paths HostingPaths) (string, error) {
	if _, err := verifiedSiteDomains(site); err != nil {
		return "", err
	}
	if err := validateHostingPaths(paths); err != nil {
		return "", err
	}

	identity := phpFPMIdentity(site)
	root := path.Join(paths.TenantRoot, "tenants", site.TenantID, site.DocumentRoot)
	runtime := path.Join(paths.TenantRoot, "tenants", site.TenantID, "run", site.SiteID)
	socket := path.Join(paths.PHPSocketRoot, site.TenantID, site.SiteID+".sock")

	var b strings.Builder
	fmt.Fprintf(&b, "[%s]\n", identity)
	fmt.Fprintf(&b, "user = %s\ngroup = %s\n", identity, identity)
	fmt.Fprintf(&b, "listen = %s\n", socket)
	fmt.Fprintf(&b, "listen.owner = www-data\nlisten.group = %s\nlisten.mode = 0660\n", identity)
	b.WriteString("pm = ondemand\npm.max_children = 5\npm.process_idle_timeout = 10s\npm.max_requests = 500\n")
	b.WriteString("clear_env = yes\nsecurity.limit_extensions = .php\ncatch_workers_output = yes\nrequest_terminate_timeout = 60s\n")
	fmt.Fprintf(&b, "php_admin_value[open_basedir] = %s:%s:/tmp\n", root, runtime)
	fmt.Fprintf(&b, "php_admin_value[upload_tmp_dir] = %s/tmp\n", runtime)
	fmt.Fprintf(&b, "php_admin_value[session.save_path] = %s/sessions\n", runtime)
	b.WriteString("php_admin_flag[log_errors] = on\n")
	return b.String(), nil
}

func phpFPMIdentity(site SiteRecord) string {
	sum := sha256.Sum256([]byte(site.TenantID + "\x00" + site.SiteID))
	suffix := strings.ToLower(base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(sum[:16]))
	return "tp_" + suffix
}
