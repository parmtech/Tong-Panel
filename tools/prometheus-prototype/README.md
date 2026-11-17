# Prometheus client prototype

This isolated proof of concept uses the Prometheus Go client to expose a constant `tong_panel_prototype_info` marker through an in-memory HTTP handler. It is **not connected to tongd**, does not listen on a port, and does not report real service health. The marker only shows the handler responds. Do not expose it to a network without an approved authentication, authorization, tenancy and information-disclosure design.

Upstream: https://github.com/prometheus/client_golang, v1.23.2, tag commit `8179a560819f2c64ef6ade70e6ae4c73aecaca3c`, Apache-2.0; transitive dependencies are pinned by go.mod/go.sum and still need complete notice, security and SBOM review before release. No third-party source is vendored.

Windows validation with Go 1.27.3: `go mod tidy`, `go test -count=1 ./...`, `go vet ./...` and `go mod verify` passed. The preexisting shared Go module cache produced an inconsistent checksum tile; a fresh temporary module cache worked with the default checksum database enabled. No shared cache was deleted or verification disabled.

Next: owner/security review of endpoint access and actual health semantics; review all dependencies and advisories; test production isolation, quotas and deployment before wiring into the application. Existing Phase Zero gates remain open.

## Linux continuous validation

The root module's `go test ./...` skips this separate module. [Isolated Go prototype checks](../../.github/workflows/prototype-go-checks.yml) explicitly tests it on relevant pull requests and main-branch pushes, or by manual dispatch. The job uses pinned Go 1.27.4 on Ubuntu 24.04 with cgo enabled, `go mod download`, `go mod verify`, `go test -race -count=1 -timeout=120s ./...`, and `go vet ./...`. It then uses pinned `govulncheck v1.9.0` with `-test -scan symbol` to scan source and tests, and checks that `go.mod` and `go.sum` remain unchanged. Reachable findings and tool failures fail the job; module-only findings remain visible for review.

No network listener, service health collector, production secret or deployment is added. The workflow has read-only repository permissions and does not persist checkout credentials. Confirm the matrix job result for the exact commit before claiming Linux/race validation; adding CI does not certify production monitoring or resolve the outstanding review gates.

### Executed local Linux evidence for this continuation

With the verified official Go 1.27.4 linux/amd64 toolchain and cgo enabled, the race suite passed in 1.023s, `go vet` and module verification passed, and the module manifests remained unchanged. `govulncheck v1.9.0` with `-test -scan symbol` reported zero reachable and zero imported-package vulnerabilities using the database updated at `2026-11-11T02:51:11Z`.

The scan separately reported one required-module finding: [GO-2026-5024](https://pkg.go.dev/vuln/GO-2026-5024), a Windows `NewNTUnicodeString` overflow in `golang.org/x/sys v0.35.0`, fixed in v0.44.0. That Windows package is not imported by this Linux prototype; the finding remains recorded for dependency and platform review, without changing or suppressing the scanner. These local results do not establish a completely cleared dependency inventory or replace exact-commit hosted checks.
