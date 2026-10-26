# P1 — Core Platform Foundation
**Status:** PLANNED; limited non-privileged developer prototype may start in a feature branch.  
**Depends on:** P0 review/ADR and threats. **Delivers:** installable *dev* component baseline, not full hosting.

## Work packages
1. **P1-01 Repository/workflow:** stable monorepo layout; Go workspace and TS frontend; contribution policy; CI with formatting, tests, lints and dependency/security scanning.
2. **P1-02 Control-plane API:** Go service, configurable loopback bind, safe timeouts, version/health endpoint, request IDs, error schema; OpenAPI. Never expose root agent publicly.
3. **P1-03 Authentication:** bootstrap, safe sessions, MFA/TOTP/passkey evaluation, rate limits, recovery, secure cookies and password storage.
4. **P1-04 Identity/authorization:** owner/admin/reseller/customer/team; tenant/resource authorization in API and worker, not just UI.
5. **P1-05 Durable jobs:** typed requests, idempotency, lease/retry, locks, state transitions, cancellation and audit.
6. **P1-06 Local host agent:** Unix-domain socket, peer credential/attestation, strict op allowlist, input validation and per-action privilege; **no general shell**.
7. **P1-07 Data model:** schema/migrations, backend storage choice and backup/upgrade plan.
8. **P1-08 UI foundation:** mobile-like app grid, global search, settings, jobs/activity, notifications, safe modal/wizard primitives, accessibility.
9. **P1-09 Diagnostic tooling:** doctor/status/logs and redacted support bundle with no secrets.
10. **P1-10 Package/release lab:** signed experimental packages, clean test VMs, supply-chain manifest and CI report; not a public one-command production installer.

## Safety and non-goals
No public root API, no arbitrary command runner, no binding admin interface to 0.0.0.0 by default. P1 is NOT hosting support; do not show enabled “create domain” / “install app” until backend is functional. All tests run in isolated sandbox/VM; no production VPS writes.

## P1 acceptance gates
- CI passes unit tests for API, error handling, auth, role/tenant checks and job lifecycle.
- Root-like agent action cannot be invoked with arbitrary command or path, with fuzz/negative test evidence.
- Restart, crash, concurrency and malformed input safety confirmed.
- Test UI meets keyboard/navigation/a11y baseline; responsive grid works on desktop and mobile.
- Verifiable dependency pinning, signed artifacts plan and tested boot path in clean supported lab VM.
- Documentation and UPDATES journal includes commit/PR, test outputs and known limits.

## Success definition
A proven foundation that hosts **zero** real customers until P2 demonstrates site lifecycle and isolation.

## Handoff
Work only from scoped GitHub issues; consult docs/ARCHITECTURE.md, docs/THREAT_MODEL.md, docs/TEST_STRATEGY.md and ADR status. First small code task is a loopback-only unauthenticated liveness endpoint with no host management capabilities and basic unit tests.
