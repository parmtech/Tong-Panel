# Certificate operation authorization and key custody

**Status: development contract and review proposal, 13 November 2026.** This document does not approve a certificate executor, role grant, public CA, persistent key store or host adapter. Current foundations include domain-verification evidence and cancellation of an isolated HTTP-01 provider attempt. A separate test-only extension investigates interrupted responses and recovery of the same local-CA order. The rest of this contract specifies the evidence needed for the next implementation stages.

## 1. Current boundary and implemented foundations

The [ACME preflight](P1_ACME_PREFLIGHT.md) is an informational `sites:read` snapshot. Its response is not an issuance permit. The existing job executor still accepts only `validation.noop`; the host agent remains ping-only. No certificate kind, role permission, API mutation, scheduled renewal or host command is added by this continuation.

The domain catalog now retains three server-authored facts after a successful DNS proof: when the matching DNS evidence was observed, the verification time sampled inside the final writer transaction, and a fresh opaque receipt identifying that verification. The verification timestamp is not an exact commit timestamp. The transient proof token remains hashed while pending and is cleared on success. The receipt is independent of the token and is not a secret or capability.

Freshness must use the **DNS observation time**. Waiting for a SQLite writer does not make an older DNS result fresh. The verification time records the later successful state transition. A usable record must satisfy `0 < observed_at <= verified_at <= now`; a caller supplies a positive maximum evidence age explicitly. No production maximum-age policy is selected here. Old verified rows retain unknown evidence, and malformed, future or stale records fail closed in the internal evidence reader. Pending aliases retain the existing preflight exclusion behavior; an otherwise verified name must not be silently omitted just because its evidence is unusable.

The internal evidence read rechecks current membership and a supplied cookie-session proof in the same SQLite statement as site/domain metadata. It is still an informational read under the existing read policy. It is not continuous authorization, a domain lock, proof of current DNS control, or an operation permission. A later operation must reload and compare the receipt for every requested name after its own authorization check.

The separate [HTTP-01 experiment](../tools/acme-prototype/README.md) supports an attempt context and terminal `Close`. Closing/canceling clears its active challenge and prevents later presentation or response admission. Response I/O does not hold the provider's state mutex. A response already admitted before closure can finish; this primitive cannot recall transmitted bytes or cancel an order already accepted by a CA. Keys remain in RAM in the loopback Pebble fixture.

## 2. Decisions required before an executable certificate operation

| Decision | Proposed boundary | Required recorded outcome |
|---|---|---|
| Operation permission | A separately named `certificates:issue`/`certificates:renew` policy, default denied until selected | Explicit role and tenant/site matrix; read/site-write/job permissions do not imply issuance |
| Interactive authentication | Actual TLS, authoritative cookie session, existing CSRF/origin protection and reviewed step-up assurance | ADR-009 authentication/assurance and reverse-proxy decisions |
| Renewal identity | A dedicated, revocable service grant scoped to the selected tenant/site and managed certificate | Grant creation, expiry, suspension, revocation and recovery policy; no indefinite reuse of an old browser session |
| Domain freshness | Explicit maximum observation age and re-verification mechanism | Owner/security selection of policy and re-verification UX; no invented freshness for legacy records |
| Resource and retry limits | Bounded names, concurrent attempts, challenge lifetime, request duration and retries per tenant/site | Reviewed abuse limits, CA retry handling and observable exhaustion; no unbounded background retry loop |
| Account scope | Tenant/account isolation and an allowlisted CA directory profile | Account sharing or isolation policy, terms/contact handling and recovery ownership |
| Key custody | Separate account, certificate and wrapping keys with tenant-bound storage | ADR-007 key source, rotation, backup, recovery, access and deletion decisions |
| Host installation | A separate typed install operation with staged validation and rollback | Reviewed identity/path binding and supported-VM evidence before privileged changes |

All entries remain proposals until recorded through [the ADR register](ADR_REGISTER.md). The first-release target requires free/open-source integrated components; this contract does not select a paid vault, external identity service or proprietary certificate manager.

## 3. Proposed operation binding and rechecks

An operation record should bind immutable operation ID, tenant/site IDs, canonical requested names, each verification receipt/observation time, operation-policy revision, selected CA/account reference, and opaque certificate-key reference. A claimed attempt additionally binds worker identity, lease expiry and a monotonically advancing generation. Raw keys, challenge values, account secrets, browser cookies and bearer capabilities do not belong in API-visible job JSON or audit rows.

The current owner/session/role, active tenant, site ownership, complete requested-name set, evidence freshness and explicit certificate permission must be rechecked:

1. Before preparing the operation.
2. When a worker claims/starts the attempt.
3. Before publishing a challenge and before sending a finalize request.
4. Before committing a returned certificate artifact.
5. Separately before any later host installation or renewal.

Bind a changed receipt to a new authorization decision. Removing/re-adding or re-verifying the same domain must not silently validate an old attempt. A still-readable site or a CA's cached authorization cannot replace panel authorization. No network or response I/O should hold a SQLite writer lock.

The existing job lease code provides useful concurrency examples, but its `validation.noop` contract must not be widened to execute certificate work. A new typed executor needs its own reviewed resource locks, fenced state transitions and tests. A lost lease, expired/revoked identity, changed evidence, unavailable authority store or cancellation must prevent the old worker from committing or publishing later state.

## 4. Proposed key and challenge custody

| Material | Proposed custody | Exposure and recovery requirements |
|---|---|---|
| ACME account private key | An opaque account-key reference accessible only to the reviewed issuer/key service | Never return to browser/jobs/logs; account identity and CA profile bound to storage; test rotation and recovery |
| Certificate private key | A separate per-artifact key reference, with an explicit reuse/rotation policy | Never reuse an account key; bind tenant/site and artifact identity; install only through the reviewed target adapter |
| Wrapping/master key | An approved source outside the metadata database and ordinary backup payload | No generated fallback/default key; rotation generations, backup-key recovery and loss behavior must be tested |
| HTTP-01 token/key authorization | Attempt-owned, bounded lifetime and exact host/path scope | Remove on completion/failure/cancel; an old attempt cannot remove or replace a newer attempt's challenge |
| Certificate/chain | A bounded artifact associated with the exact request and key reference | Verify requested SANs, chain, validity and private-key match before acceptance; retain previous working artifact for rollback |
| Audit/support evidence | Fixed event codes and scoped lifecycle identifiers | No PEM/private keys, proof tokens, cookies, request bodies or arbitrary upstream error strings |

The existing [MFA secret box](../internal/auth/mfa_secret_box.go) shows authenticated encryption with an off-database key, identity binding and envelope versions. It is a bounded TOTP-seed primitive, **not** a certificate-key store. Do not reuse its wrapping key, copy its size limits to PEM data, or claim persistent ACME key custody exists. A certificate envelope needs a separately reviewed format, component-specific associated data, bounded parsing and tests for cross-tenant/account/artifact swaps, corruption and unknown generations. No storage implementation or algorithm expansion is introduced here.

ACME separates account-authentication keys from certificate keys. Account rollover and authorization management are protocol operations; local worker cancellation alone is not certificate revocation. The HTTP-01 resource should be removed after challenge completion. These protocol distinctions come from [RFC 8555 sections 7.3.5, 7.5, 7.6, 8.3 and 11.1](https://www.rfc-editor.org/rfc/rfc8555). The storage, role and worker policies in this document are Tong Panel proposals, not requirements selected by that RFC.

## 5. Failure, restart and renewal behavior to prove

| Failure boundary | Required behavior before acceptance |
|---|---|
| Canceled before challenge publication | Close the attempt, publish nothing, deny later presentation and retain a redacted terminal status |
| Canceled during challenge response | Stop new admissions promptly; allow that bytes already admitted may have been sent; remove the owned challenge without waiting for a blocked response writer |
| Failure after a remote order/finalize request | Record an uncertain remote outcome and reconcile through the trusted CA/account profile; do not claim that no certificate exists |
| Process crash or expired lease | Close/fence the old attempt, reconcile owned challenges and uncertain orders, and reauthorize before any retry; never blindly replay a finalize/install action |
| New owner, replaced proof or revoked grant | Deny the prior attempt even if domain text and site ID are unchanged; never transfer key access implicitly |
| Key source or authoritative DB unavailable | Fail closed; no plaintext/default-key fallback and no stale authorization grace |
| Failed host install/reload | Preserve the previous working certificate/configuration, verify rollback, and distinguish issuance from successful installation |
| Renewal/revocation | Use current explicit service authority and current domain evidence; respect bounded retry/rate policy and distinguish renewal, account-key rotation and certificate revocation |

Local idempotency keys need bounded retention and exact intent matching. Local retry limits and CA response/backoff handling must be reviewed together; retrying must not mint unlimited accounts/orders. No automatic renewal interval, CA-specific limit or production retry budget is chosen by this proposal.

The [local interruption experiment](../tools/acme-prototype/LOCAL_E2E.md#interrupted-order-response-and-reconciliation) exercises one uncertainty boundary: a finalize request accepted by Pebble whose response is withheld, followed by same-account recovery of the saved order without another order/finalize. The account and separate certificate key stay in RAM, and a new bounded context performs recovery. Its test-only transport cancels outgoing requests and bodies; lego polling sleeps can still continue until their configured bound. This evidence does not provide durable restart recovery, persistent key custody or production authorization.

## 6. Next implementation stages and acceptance

| Stage | Concrete deliverable | Evidence required |
|---|---|---|
| Evidence and attempt cleanup — current slice | Catalog proof observations/receipts; cancellation-scoped HTTP-01 provider | Legacy migration, transaction/audit rollback, stale/future/changed evidence, current tenant/session checks, blocked response I/O, race tests and local Pebble cancellation |
| Explicit policy and key-custody decisions | Reviewed role/assurance/freshness/account/key profiles | Actual accepted/revised decisions; no inferred permission from current read APIs |
| Typed local issuer harness | Separate operation record/attempt protocol using fixed loopback CA and RAM keys | Revocation/evidence change at every boundary, lease fencing, canceled/replayed attempts, uncertain-order reconciliation and no-op-worker separation |
| Persistent key and recovery adapter | Approved encrypted store plus recovery/rotation interfaces | Tamper/cross-tenant denial, unknown-generation handling, crash consistency and fresh-host recovery |
| Host install and renewal | Reviewed typed target adapter and bounded renewal service | Disposable supported-OS integration, failure/rollback, expiry and recovery tests, independent security and owner acceptance |

Keep the public preflight response `snapshot_only: true` and `execution_enabled: false` until a separate reviewed implementation changes that contract. Update [UPDATES.md](../UPDATES.md) and the [resume handoff](RESUME_HANDOFF_2026-11-12.md) with executed results and exact commits; a proposed test is not passing evidence.
