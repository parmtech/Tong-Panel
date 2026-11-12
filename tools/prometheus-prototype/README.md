# Prometheus client prototype

This isolated proof of concept uses the Prometheus Go client to expose a constant `tong_panel_prototype_info` marker through an in-memory HTTP handler. It is **not connected to tongd**, does not listen on a port, and does not report real service health. The marker only shows the handler responds. Do not expose it to a network without an approved authentication, authorization, tenancy and information-disclosure design.

Upstream: https://github.com/prometheus/client_golang, v1.23.2, tag commit `8179a560819f2c64ef6ade70e6ae4c73aecaca3c`, Apache-2.0; transitive dependencies are pinned by go.mod/go.sum and still need complete notice, security and SBOM review before release. No third-party source is vendored.

Windows validation with Go 1.27.3: `go mod tidy`, `go test -count=1 ./...`, `go vet ./...` and `go mod verify` passed. The preexisting shared Go module cache produced an inconsistent checksum tile; a fresh temporary module cache worked with the default checksum database enabled. No shared cache was deleted or verification disabled.

Next: owner/security review of endpoint access and actual health semantics; review all dependencies and advisories; test production isolation, quotas and deployment before wiring into the application. Existing Phase Zero gates remain open.
