# Isolated ACME HTTP-01 provider experiment

This original AGPL-3.0 prototype evaluates the lego HTTP-01 provider contract in a separate Go module. The provider and default unit tests remain network-free. The explicitly selected local end-to-end test adds ephemeral loopback DNS/HTTP/HTTPS fixtures, a Pebble test CA, local account/orders, and disposable .test certificates with keys held only in RAM. See [local E2E evidence](LOCAL_E2E.md). No daemon integration, public ACME service, real domain, persistent key, host change or production lifecycle acceptance is established.

One provider instance is configured with one trusted canonical ASCII hostname, a maximum ten-minute lifetime and one active token. The future trusted caller must separately authorize the tenant and domain before construction. Tokens and key authorizations are bounded and validated. Requests require the configured host, exact challenge path and GET; unknown paths, hosts and methods return 404. Identical Present retries do not extend expiry, replacement while active is rejected, and stale cleanup cannot remove a later challenge. State is memory only and lost on restart. Unicode/IDNA, wildcard and IP hosts are deliberately unsupported in this experiment.

## Choice and provenance

Lego provides the small explicit provider contract needed for this experiment. CertMagic also offers full automated TLS management, storage, listeners and renewal; those are broader than the behavior being evaluated here. This choice is an experiment, not a production architecture decision.

- lego `github.com/go-acme/lego/v4 v4.34.0`, exact tag commit `f3c686a890e6fa36b92614112e8717832894faa7`, [official source](https://github.com/go-acme/lego/tree/v4.34.0), [MIT license](https://github.com/go-acme/lego/blob/v4.34.0/LICENSE). Module checksum `h1:oRsIuPJ4ORX7ufviXvelUpBSez2XxeKGwo5pNG9BVeY=`; go.mod checksum `h1:gsmdlx/ZS6OUeXbOj0U+VnCLLfEFj4WCYRkcGpZw+pc=`.
- Imported dependency closure: `github.com/cenkalti/backoff/v5 v5.0.3` (MIT) and `github.com/go-jose/go-jose/v4 v4.1.8` (Apache-2.0; raised from v4.1.4 by Pebble). Exact module and test dependency checksums are recorded in go.sum. License texts are retained in `third-party-notices/`; no upstream implementation was copied into the provider.
- The earlier research reference lego v4.28.1 was rejected after the scanner identified webroot traversal [GO-2026-5593](https://pkg.go.dev/vuln/GO-2026-5593), fixed in v4.34.0. This prototype never imports its webroot provider. The go-jose pin addresses [GO-2026-4945](https://pkg.go.dev/vuln/GO-2026-4945). Go 1.27.4 addresses the Go 1.27.3 net/http finding [GO-2026-6859](https://pkg.go.dev/vuln/GO-2026-6859).

## Initial provider verification — 2026-11-12

Using official automatically downloaded Go 1.27.4 windows/amd64, with default `GOPROXY=https://proxy.golang.org,direct` and `GOSUMDB=sum.golang.org`:

- `go test -count=1 ./...`: PASS (four tests; final run 1.504s), covering presentation/removal, host/path/method isolation, input bounds, expiry, retry, stale cleanup and concurrent operations.
- `go vet ./...`: PASS.
- `go mod verify`: PASS, all modules verified.
- `govulncheck -show verbose ./...`: PASS, no vulnerabilities found in the four-module closure and Go 1.27.4 standard library at scan time.
- Race detection: NOT VERIFIED. `go test -race` could not run because this Windows environment lacks cgo/compiler support; concurrent functional testing is not a race detector substitute.

The existing shared module cache returned an inconsistent checksum tile for another experiment. Verification used a fresh process-scoped `GOMODCACHE=$env:TEMP\tong-prometheus-verified-cache-20261112` and `GOTOOLCHAIN=go1.27.4`; no checksum controls were disabled and no permanent Go environment or global toolchain settings were changed.

The local test-server experiment is now implemented below. Next: account/key custody, interruption/restart reconciliation, renewal/expiry and authorization/host integration review. No live issuance or production certificate readiness is established here. Production selection still needs packaging/SBOM, full dependency license review, independent security review and supported lifecycle acceptance.
