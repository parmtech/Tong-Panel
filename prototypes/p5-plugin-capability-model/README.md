# P5 plugin capability model (prototype)

This standalone standard-library-only Go module models one narrow acceptance property: a disabled plugin grant and any mismatched plugin/install/tenant/resource/capability binding are denied. It has no network, persistent storage, host integration, secrets, sandbox, or Tong Panel runtime dependency.

`Grant` and `Request` must be created by a trusted policy/identity layer in any future implementation. The plugin must not provide either as authoritative input. This prototype does not define grant storage, signature verification, global plugins, capability schemas, cache invalidation, data preservation, or process isolation. Exact string matching is only a candidate decision rule for review; it is not a plugin security boundary or an approved architecture.

## Automated and local checks

Root-module `go test ./...` does not traverse this separate module. The [offline model and fixture workflow](../../.github/workflows/offline-model-checks.yml) enters it explicitly and checks formatting, race-enabled tests and vet on Ubuntu 24.04 with Go 1.27.4. Module downloads are disabled and manifests must remain unchanged. Passing these checks verifies modeled grant comparisons only, not a plugin sandbox, security boundary or P5 acceptance.

From this directory, with Go 1.27.4 and a C compiler on `PATH`:

```sh
export CGO_ENABLED=1 GOTOOLCHAIN=local GOFLAGS=-mod=readonly
export GOWORK=off GOPROXY=off GOSUMDB=off
go test -race -count=1 -timeout=120s ./...
go vet ./...
```
