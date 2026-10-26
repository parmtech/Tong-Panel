# Tong Panel — QA and Test Strategy
**Status:** PLANNED. There are no validated distro install tests at time of creation.

## Test pyramid and environments
- Pure unit tests for validation, permission decisions, contract serialization and resource models (per commit).
- API contract tests including RBAC/tenant matrix, session and rate limit, JSON schema/OpenAPI, deterministic error codes.
- Agent and service-adapter tests in isolated Linux containers/VMs; never run privileged integration suite against a developer's live web host.
- E2E flows in fresh ephemeral supported distro VMs (not a pre-configured shared host).
- Browser automation across mobile/tablet/desktop with keyboard and screenreader audits.
- Fault injection/chaos, race/concurrency, workload limits, network outages, rollback and backup restore.
- Security SAST/dependency scans, SBOM/license provenance, fuzzing and third-party audit prior to stable release.

## CI lanes
1. **Docs and style** (every PR): links/format, broken references, licensing declarations, UPDATES change status.
2. **Go controls** (once code lands): gofmt, go vet, unit tests, race detector, fuzz targets.
3. **Web UI** (once code lands): TS checks, lint, unit tests, a11y and build.
4. **API contract**: OpenAPI semantic version/compatibility and authorization matrix.
5. **Service/host**: Nginx/PHP/DB and agent integration with temp disposable hosts.
6. **Release**: supported distro CI, signed artifacts, install/upgrade/restore/rollback.
7. **Scheduled security**: dependency advisories, app recipe pins, regression against previously found vulnerabilities.

## Required bad-path test cases
- Invalid auth/malformed requests/rate limiting; no accidental sensitive response data.
- Suspended tenant and stale permissions cannot modify files/jobs/databases.
- Command injection, symlink file escape, ZIP Slip and decompression bomb.
- Crash during job, config update, certificate issue, install and restore.
- Unavailable upstream DNS/cloud/ACME/Git provider (timeouts/retries bounded).
- Quota overflow, full disk, huge upload, database unavailable, partial network.
- Old plugin or API client incompatible version; no silent destructive data conversion.
- Corrupt signature/image/manifests fail closed.
- Duplicate webhook and concurrent admin action deduplicate instead of double-apply.

## Phase gate evidence format
Record: test name; commit SHA; OS/arch/profile; command/script; pass/fail counts; failure links/log paths with secrets redacted; reviewer; known exclusions. A written “we tested everything” is not evidence.

## Supported OS matrix
Ubuntu 24.04 LTS x86_64 and Debian 12 x86_64 are *candidate lab targets*, NOT verified production support. Add Debian 13 and arm64 based on evidence. No distro marked supported until clean install, upgrades, rollback, recovery and relevant service E2E pass.

## Manual acceptance by role
First-time novice, web developer, hosting reseller and server operator should complete key tasks with minimal help. Specifically verify mobile icon grid, global search, contextual tours, file Explorer operations, setting domain+SSL, importing app, error recovery and accessibility. Feature only accepted when UI, API, privilege layer and data layer all function coherently.

## Test status log
2026-10-26: Strategy drafted only. No real integration, CI or installer tests executed in this file.
