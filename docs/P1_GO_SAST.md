# Go source static security scan

The Go SAST workflow runs pinned `gosec` v2.30.0 against the Linux amd64,
CGO-disabled application packages and test files. It writes a machine-readable
JSON report and uploads it as a short-lived workflow artifact. The workflow
uses read-only repository permission and no deployment, signing, or security
event credentials.

The report is advisory while the project establishes a reviewed baseline and
security finding policy. Scanner findings do not currently fail the job;
`gosec` runs with `-no-fail` so the report remains available for review. The
workflow separately rejects non-empty report-level analyzer errors, and fatal
setup, package-processing, report-writing, missing or malformed report, and
artifact-upload errors fail the workflow. Do not add `#nosec` suppressions or
treat a clean run as security approval without documented review.

The scan covers the Go source and tests selected for Linux amd64 with CGO
disabled. It does not cover other build tags, platforms, toolchains, generated
assets, JavaScript, dependencies as deployed, or runtime behavior. The existing
`govulncheck` workflow remains the separate known-vulnerability call-path
check. This scan is not a penetration test, full security audit, or substitute
for the pending P0 threat-model and owner/security review. Making its result a
required merge check also requires repository ruleset configuration; this
workflow does not change that configuration.

Reproduce the scan from the repository root after installing Go 1.27.3:

```sh
go mod download
go mod verify
GOBIN="$(go env GOPATH)/bin" go install github.com/securego/gosec/v2/cmd/gosec@v2.30.0
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 GOTOOLCHAIN=local gosec -no-fail -tests -fmt=json -out gosec-report.json ./...
```
