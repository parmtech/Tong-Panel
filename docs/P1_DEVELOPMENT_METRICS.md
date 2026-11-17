# Optional development Prometheus endpoint

The loopback-only development daemon can expose `GET /metrics` when the operator explicitly supplies `TONG_DEV_METRICS_TOKEN`. With no value, the route remains 404. This is a small integration of the earlier isolated `tools/prometheus-prototype` experiment; it is not service health, production monitoring, a phase exit or security approval.

The token must be a separate random secret of 32–4096 printable non-space ASCII characters and must differ from `TONG_DEV_TOKEN`. Invalid configuration stops startup before opening the development state store. The scrape requires exactly one `Authorization: Bearer <metrics-token>` header, an actual IPv4 loopback peer, GET and no query parameters. Tokens in URLs, cookies and forwarding headers do not authorize scraping. Existing listener validation still rejects public/wildcard/hostname/IPv6 binding. Reverse proxy exposure is unsupported: a proxy could make external traffic appear local, so loopback checks do not authorize public proxying.

The only metric is `tong_panel_development_metrics_info 1`. A private registry registers no default Go/process collectors; it has no tenant, identity, site, token, URL, job, filesystem or host labels. The value says the opt-in scrape endpoint responds, and does not establish dependency health. Scrapes are bounded to one in flight and a five-second timeout, with `Cache-Control: no-store`. The separate metrics credential grants no API privilege; the API developer credential does not grant scrape access.

Set the metrics token through the local process environment, using your secret generator rather than a literal copied from documentation. Keep the daemon bound to its existing loopback listener. A local Prometheus configuration may use a protected credential file, kept outside git:

```yaml
scrape_configs:
  - job_name: tong-development
    static_configs:
      - targets: ['127.0.0.1:8090']
    authorization:
      type: Bearer
      credentials_file: /private/local/path/tong-development-metrics-token
```

No Prometheus server is installed or configured by Tong Panel. The listener is plaintext and development only. Do not place credentials in query strings, committed configuration, screenshots or shell command history. Broader monitoring, alerting, authentication, transport, proxy and access-policy decisions remain open.

## Dependency and review evidence

The main module pins `github.com/prometheus/client_golang v1.23.2`, tag commit `8179a560819f2c64ef6ade70e6ae4c73aecaca3c`, module checksum `h1:Je96obch5RDVy3FDMndoUsjAhG5Edi49h0RJWRi/o0o=`, using Go's module checksum verification. Upstream is [Apache-2.0](https://github.com/prometheus/client_golang/blob/v1.23.2/LICENSE); its original LICENSE and NOTICE are retained in `docs/third-party/prometheus-client-golang-v1.23.2/`. No upstream implementation is copied into the new integration file. Exact direct/transitive versions and checksums are in the main go.mod/go.sum. Full distribution/SBOM/transitive-notice review remains required before release.

Verification on 2026-11-12 uses official Go 1.27.4 windows/amd64 with default checksum security and a fresh process-scoped cache. Focused daemon tests passed, including an actual ephemeral loopback HTTP scrape, default-disabled behavior, separate credentials, duplicate authorization rejection, nonloopback/forwarded peer rejection, method/query/path restrictions, minimal metric families and invalid-config-before-state startup. No production process, live credential or real host was used.

Full main-module `go test ./... -count=1 -timeout=180s`, `go vet ./...`, `go mod verify` and `git diff --check` also passed. Nested experiment modules are not included in that main-module test command.

`govulncheck ./cmd/tongd` reported zero reachable vulnerabilities in this configuration, with one existing imported-package finding (`golang.org/x/sys/windows` GO-2026-5024) and 20 existing module-only x/crypto findings that its call graph does not reach. The existing x/sys and x/crypto pins were not changed by this metrics integration. This does not clear those dependencies for other uses or platforms. Race detection is unverified in this Windows environment because cgo/compiler support is absent. Hosted Linux/race checks and review remain required.
