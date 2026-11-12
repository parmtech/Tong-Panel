# ACME catalog preflight boundary

The existing local-only ACME experiment has passed actual HTTP-01 validation against pinned Pebble, including invalid-challenge cleanup. That is protocol evidence, not permission to make the daemon an issuer. The main queue and development worker currently accept only validation.noop, and deliberately have no executor callback, command or host adapter. Reusing that job kind for certificate operations would violate its metadata-only contract.

The next implemented step is a detached informational snapshot: Catalog.ACMEHTTP01PreflightFor in internal/sites/acme_preflight.go. It loads the existing site through GetFor, which checks current tenant membership and supplied session proof in the same SQLite statement snapshot, then reuses the established verified-domain validator. A pending primary rejects the request; pending aliases are omitted; verified aliases are sorted after the primary. Invalid site/domain metadata fails closed. The requester and tenant inputs must come from an authenticated trusted caller, just as for other catalog methods. Passing nil session proof is for the existing membership-checked developer-token flow, never a cookie-authentication bypass.

The returned tenant/site/domain list contains no CA URL, certificate request, challenge token, key, filesystem path, executable or job identifier. It cannot execute anything. It is not a capability or a durable authorization record: ownership, membership, role, session and policy must be checked again when a future executor acts. It does not query DNS or establish freshness of prior domain proofs. Existing session-read behavior may record an observed session expiry; the method adds no certificate/job/host mutations.

No HTTP route or worker behavior is changed. The tested lego provider remains in its isolated module, with no main-module dependency. No certificate issuance, account creation, network access, challenge publication, key storage, configuration write or service reload is introduced by this method. An API would expose this snapshot only after its own route/authentication contract and tests are reviewed. Informational read permission must never be treated as issue/renew permission.

## Executed verification — 2026-11-12

Using official Go 1.27.4 windows/amd64 with a process-scoped module cache and checksum verification enabled:

- New preflight tests cover verified primary/alias selection, pending primary rejection, pending alias omission, cross-tenant requester rejection, tenant-scoped missing-site response, ownership/membership revocation, valid then revoked session proof, caller cancellation, detached result mutation and no queued jobs.
- go test ./internal/sites ./internal/jobs -count=1 -timeout=180s: PASS (sites 4.555s; jobs 7.825s), preserving the existing no-op worker boundary.
- go vet ./internal/sites ./internal/jobs and git diff --check: PASS.
- Race detection remains unverified in this Windows environment without cgo/compiler support.

Next concrete review: a read-only authenticated site-preflight route, or a separately named local-test-only workflow with a fixed loopback test CA. Before any executable certificate job, define explicit operation permission, domain recheck at claim/start, cancellation/lease fencing, account/key custody, failed-order reconciliation and renewal/expiry handling. Public CA access and host effects remain outside this step and existing P0/security/phase gates remain open.
