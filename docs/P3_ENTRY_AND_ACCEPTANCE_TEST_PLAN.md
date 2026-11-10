# P3 Entry Gates and Acceptance Test Plan

**Status:** design and test specification only. P3 implementation is gated.
**Basis:** `docs/ROADMAP.md`, `docs/PHASE_INDEX.md`, `docs/phases/P1-CORE-FOUNDATION.md`, `docs/phases/P2-HOSTING-ALPHA.md`, `docs/phases/P3-APP-PLATFORM.md`, `docs/BLUEPRINT.md`, `docs/THREAT_MODEL.md`, `docs/SECURITY.md`, `docs/ARCHITECTURE.md`, and `docs/TEST_STRATEGY.md`.
**Rule:** a draft PR, passing unit test, or unsigned developer VM smoke is not phase acceptance. Record evidence at the exact tested commit and do not enable a gated capability before its prerequisites pass.

## Current gate decision

**P3 entry: NOT PROVEN.** The Phase Index requires P1 before P2 and P2 before P3. The P3 plan specifically depends on secure site routing, authentication and quotas, backup, and deploy jobs. The current `main` checkpoint is `0aad68e`; it records P1 as incomplete. The current P2 plan says planned/not implemented. GitHub's open P2 drafts provide useful component and test evidence, but their limits still leave installer/service lifecycle, complete Files ACL/UI and quota wiring, full backup/restore operations, and overall P2 acceptance incomplete. Draft #84 reports a narrow MariaDB fresh-VM restore drill; it does not provide an offsite retention service or complete application/settings restore. Several drafts also say owner ADR/threat-model review is still pending. They remain drafts and are not merged foundation on `main`.

This specification is safe to review and refine without privileged operations. P3 implementation remains blocked until the entry evidence below is independently assembled and reviewed. No P3 feature, endpoint, runtime, build, installer, or host mutation is authorized by this document.

## Evidence protocol

For each gate, record the source commit SHA, PR and merge state, exact workflow/run URL and conclusion, test command, OS/architecture, disposable environment identity, expected and actual results, reviewer, and unresolved exclusions. Link artifacts/logs with secrets redacted. Distinguish code present, check authored, check executed, check passed, and gate accepted. A claim in a PR body is a lead to verify, not independent evidence by itself.

### P1 prerequisite matrix

All rows must be proven before P2 can be treated as a usable dependency for P3.

| Gate | Required evidence for P3 entry | Current evidence / status |
|---|---|---|
| P0 architecture and threats | Maintainer decisions recorded for applicable ADRs; threat register reviewed with owners, mitigations, issues, and acceptance tests | `main` ADR register labels decisions draft/not approved; threat model is unreviewed. **Open.** |
| Identity and tenant authorization | Production-intended auth/session/MFA/recovery contract; request, job, and agent authorization negative matrix, including suspended users/tenants and stale memberships | `main` says no real hosting product; historical checkpoints describe internal prototypes and unfinished external login/recovery. Open drafts do not supersede that checkpoint. **Open.** |
| Durable jobs and privilege boundary | Crash/restart/concurrency/idempotency/cancellation evidence; typed agent capabilities, fencing, audit and adversarial no-shell/path tests; independent security review | Existing prototypes and unit/hosted checks are not evidence of accepted privileged host operations; current `main` reports those gates open. **Open.** |
| Installer, supported host and recovery | Signed/provenance-checked package, exact supported OS matrix, clean install/upgrade/revert, failed-preflight no-mutation, backup-before-migration, recovery drill | PR #37/#45 describe unsigned developer-preview guest smoke and explicitly exclude installer, lifecycle, rollback, restore and OS certification. **Open.** |
| UI and operator usability | Responsive keyboard/a11y review and real role-based end-to-end workflows; no dead controls | Not established by the retrieved checkpoint. **Open.** |
| Dependency/supply-chain review | Pinned dependency inventory, vulnerability/licence findings and disposition, SBOM/signature plan, secrets redaction | Not established as a completed P1 exit gate. **Open.** |

### P2 prerequisite matrix

Every P2 acceptance scenario in `docs/phases/P2-HOSTING-ALPHA.md` remains required. P3 may not assume P2 contract behavior from isolated prototype packages.

| P2 contract | Required proof before P3 consumes it | Current GitHub signal (open draft; not gate acceptance) |
|---|---|---|
| Tenant/site identity and routing | Two-tenant API, job, file, database and route isolation; verified domain/TLS route with teardown; authorization repeated at job/agent boundary | Current P2 plan requires two-tenant lifecycle tests; latest `main` lacks accepted P2. **Open.** |
| Site lifecycle and safe host changes | Idempotent create/update/remove, staged config validation, reload failure rollback, port conflict and unsupported-host fail-before-mutation | Draft #80 reports isolated Nginx staging/validation/reload rollback tests on clean Ubuntu/Debian guests, but says the daemon does not apply host config. P2 plan requires integrated lifecycle end-to-end. Narrow evidence; gate **Open**. |
| Files, ACL and quotas | Authenticated list/read/write/delete/upload and archive flows; object/tenant ACL, race and quota enforcement; crash reconciliation and safe cleanup | PR #51 is read-only and unconfigured; #52 upload depends on it; #53 quota remains opt-in/unwired and requires trusted reconciliation. All are drafts. **Open.** |
| Backup and restore | Encrypted offsite backup, retention, destroy disposable source VM, clean target restore and data/settings verification; interrupted/partial restore recovery | Draft #84 reports passing Ubuntu 24.04/Debian 12 fresh-VM encrypted MariaDB dump/restore and wrong-tenant/key denial. PR #91 exact head `01ae2237b8bdb4240fb79ca45842f9e6d4f549a1` also has a completed-success clean-VM run #42633862164 on Ubuntu 24.04 and Debian 12; the PR text still calls those jobs pending, so its narrative needs reconciliation. These are narrow unsigned-preview lab results; the PR limits exclude production key custody, offsite retention/provider, schedules and complete managed service restore. Full P2 backup gate remains **Open**. |
| Authentication/MFA/recovery | Publicly usable protected workflow and abuse/recovery tests; no credential or tenant crossover; auditable secret redaction | Main P1 checkpoint and open P2 draft limits do not establish this. **Open.** |
| Installer/OS support | Exact supported OS/architecture, signed artifact validation and clean install/upgrade/rollback in disposable VMs | PR #45 calls guest smoke unsigned developer preview and excludes installer/certification. **Open.** |
| P2 operator acceptance | Complete documented alpha gate, visible limitations, reproducible evidence and maintainer/security review | Not recorded as accepted. **Open.** |

The WordPress tenant runtime draft #89 exact head `51e12abb99d919e5e8cb10a69e868dc65c4ed9a` has a completed-success clean-VM run #42631444449 on Ubuntu 24.04 and Debian 12. The PR description still says fresh validation is pending despite that run; reconcile the journal with the run before counting its detailed assertions. The run uses the unsigned developer preview and does not establish install/upgrade/rollback or P2 phase acceptance.

## Cross-phase contracts P3 must consume

These are contract requirements to validate and settle against the accepted P1/P2 implementation before P3 coding. They do not assert that the interfaces already exist.

| Producer contract | P3 consumer | Required compatibility and failure behavior |
|---|---|---|
| Authenticated principal, tenant membership, role/permission decision, session and CSRF/origin policy (P1) | App/repository/runtime/secret/deployment APIs and queued deploy jobs | Derive tenant and actor from verified server context; reauthorize when a worker executes; reject stale/revoked membership; never trust caller-supplied tenant IDs. |
| Durable typed job lifecycle, idempotency, cancellation, audit, retry and recovery (P1) | Fetch, build, stage, health-check, switch-route, rollback, and cleanup jobs | Every transition is durable and attributable; retries cannot publish twice; cancellation and process restart reconcile or fail closed; privileged effects stay within typed agent capabilities. |
| Site, domain, verified TLS route, health and reverse-proxy lifecycle (P2) | Attach Node/Python runtime to a customer hostname and switch active release | Use stable site/tenant ownership and centrally reserved ports; preserve routing/TLS ownership checks; stage and health-check before traffic switch; revert on apply/reload failure. |
| File ownership, ACL, quota accounting and safe staging (P2) | Source checkout, build workspace, release artifacts and app data | Keep build/release paths separate from customer data; account all bounded storage including concurrent builds/artifacts; prevent path/symlink escape; retain reservations through uncertain publication until reconciled. |
| Secret reference, encryption/key custody, rotation and redaction (P1/P2) | Git credentials, build-time variables, runtime environment and app secret rotation | Pass only scoped references to the intended process; define whether build-time and runtime secrets differ; deny reads from other apps; redact logs/process diagnostics; define backup and restore semantics before use. |
| Backup/restore and retention (P2) | App release history, app data, environment configuration and rollback | Specify which artifacts/configuration are included, secret/key dependencies, point-in-time consistency, retention and restore ordering. P3 rollback is not a substitute for backup; data deletion/removal must remain explicit and recoverable. |
| Approved runtime/build supply-chain policy (P1/P3) | Node/Python toolchains, dependencies, optional container images and declarative recipes | Pin immutable sources and digests; define signature/provenance trust roots and revocation; produce dependency/licence inventory; fail closed on unknown or changed artifacts. |

For each row, publish a versioned schema or typed contract, ownership rule, error/state model, security review, and compatibility tests before the first integration consumes it. If P2 exposes no stable contract, record the blocker and keep P3 work at design/test-fixture level.

## P3 design decisions requiring review

These questions are intentionally unresolved; answer them through the repository's ADR/issue review process before making a production-facing contract. A prototype must label its assumptions and must not silently turn one answer into a supported policy.

| Decision | Options/questions that need evidence | Minimum review evidence |
|---|---|---|
| Build network egress | Deny by default, allow only reviewed destinations, or allow broader egress with explicit user consent? How are DNS rebinding, redirects, private/link-local ranges, and package registries handled? | Threat review, deterministic network tests, documented exception UX, and proof that panel/agent/metadata endpoints remain unreachable. |
| Build-time secrets | Should untrusted dependency/build scripts receive any application runtime secrets? Can credential forwarding be avoided through a separate fetch phase? | Explicit secret lifecycle diagram; tests prove least exposure and absence from logs, argv, environment snapshots accessible to other apps, and failed-build diagnostics. |
| Build and runtime identities | Separate identity per app, per build, or per build attempt? Which paths and process metadata can tenants observe? | OS capability/UID model, systemd sandbox profile, process/file visibility tests, cleanup and crash-recovery evidence. |
| Runtime/toolchain provenance | Which official sources and pinning/signature mechanism establish Node/Python runtime trust? Who can revoke a compromised release? | Maintained version matrix, key/trust-root ownership and rotation/revocation plan, tamper and stale-metadata tests, SBOM/licence output. |
| Artifact immutability and retention | Where are source snapshots, intermediate outputs and promoted releases stored? What prevents in-place mutation, and which retention policy is compatible with rollback and quotas? | Storage threat model, tenant/quota accounting, digest verification, concurrent publication and garbage-collection recovery tests. |
| Deploy state machine and route switch | Which exact state transitions are durable? When does a build become the active release, and how are concurrent deploy, cancel, timeout, restart and uncertain proxy reload resolved? | Versioned state/event model, idempotency rules, crash injection at each boundary and externally verified old/new route behavior. |
| Webhook and Git authorization | Which source/ref/repository is bound to an app? How are delivery replay, credential scope, credential revocation and provider outage represented? | Threat model, signature/replay tests, least-scope provider configuration, and redacted audit evidence. |
| Catalogue trust and removal | Who signs first-party recipes, how is key compromise/revocation handled, and what exact data is owned by a recipe versus a customer? | Maintainer-approved permissions and provenance policy, tamper/revocation tests, explicit data ownership/removal UX and restore proof. |

## Hostile fixture catalog

Create synthetic repositories and test applications only after the runner boundary is chosen. Run these fixtures solely in disposable guests with no production credentials or network access to real services. Fixture code is adversarial test input and must never be run on the Windows developer host.

| Fixture ID | Build/runtime behavior | Required oracle |
|---|---|---|
| `app-cross-tenant-read` | Tries known synthetic paths and IDs belonging to a second tenant, including secrets, source, artifacts and backup fixtures | Every access denied; canary values absent from stdout/stderr, logs and API responses; victim files unchanged. |
| `app-process-probe` | Enumerates process metadata and attempts to signal a sibling app/build identity | Only the fixture's permitted process metadata is visible; cross-identity inspection/signals fail; no host or control-plane process can be affected. |
| `build-limit-cpu-memory` | Bounded CPU loop and incremental memory allocation until the configured cgroup limit is reached | Worker terminates within policy; guest remains responsive; prior release and other app health remain unchanged; job has a typed resource-limit result. |
| `build-limit-disk-process-time` | Writes only within its workspace, creates bounded child processes and blocks on a controllable timer | Each configured limit trips independently; workspace/quota accounting is reconciled; no orphan process or reservation remains after cleanup. |
| `build-network-boundary` | Attempts loopback/private/link-local and synthetic metadata targets, then follows controlled redirect/DNS-rebinding fixtures | Requests outside the reviewed allowlist fail; no credential or internal response is returned; logs redact request secrets. |
| `build-secret-canary` | Searches its own environment, argv and workspace for a synthetic canary and emits discovered values | Only the explicitly approved secret stage receives the canary; no unapproved phase receives it; all logs and audit output are redacted. |
| `release-tamper` | Alters one byte in a staged artifact or supplies an artifact under another tenant/app identity | Digest/ownership verification rejects promotion before route change; old healthy release stays active; rejection is audited without secret data. |
| `health-failure` | Starts but returns wrong status/body, hangs, or exits after initial readiness | Readiness timeout is bounded; active traffic stays on the previous release; rollback/job state is durable and operator-visible. |
| `webhook-replay-invalid` | Sends invalid signature, duplicate delivery ID, stale timestamp, wrong ref and malformed/oversized payloads | No build job is accepted; response is bounded and non-sensitive; audit has an attributable denial reason. |
| `recipe-untrusted` | Uses unsigned, revoked, mutable-only, digest-mismatched, unknown-license or overprivileged definitions | Catalogue/install path refuses before fetch or execution; no files, ports, services, or records are left behind. |

Each fixture should have a pinned source revision and deterministic expected result. Record the guest image digest, runner policy, test command, output digest, and cleanup result. A fixture pass is only evidence for its named assertion; it does not imply overall P3 or P2 acceptance.

## Phase-plan module traceability

This crosswalk follows all ten modules in `docs/phases/P3-APP-PLATFORM.md`. It is a coverage map for the plan, not a statement that any module is implemented.

| P3 plan module | Covered here | Fixture/acceptance evidence |
|---|---|---|
| 1. Versioned Node.js/Python runtimes | P3-01, runtime/toolchain provenance decision | Runtime compatibility matrix, pinned source/digest and tamper rejection; Node and Python exit scenarios 1–2. |
| 2. Git deploy and webhook verification | P3-02, webhook/Git authorization decision | `webhook-replay-invalid`, least-scope disposable credential, revocation/redaction; exit scenarios 1 and 7. |
| 3. Native process manager and site UID sandbox | P3-05, build/runtime identities decision | `app-process-probe`; process visibility, resource policy, crash/restart and bounded cleanup; exit scenarios 3–4. |
| 4. Isolated build runners and resource limits | P3-03, build network and secret decisions | `build-limit-cpu-memory`, `build-limit-disk-process-time`, `build-network-boundary`, `build-secret-canary`; exit scenarios 3–4 and 7. |
| 5. Python WSGI/ASGI profiles | P3-01, P3-05, P3-07 | Python ASGI fixture with isolated dependencies, dedicated identity, health and proxy checks; exit scenario 2. Add WSGI only when separately listed in the compatibility matrix. |
| 6. Node app profiles and reverse proxy | P3-01, P3-05, P3-07 | Node fixture with pinned runtime, centrally reserved port, verified route/health, release switch and rollback; exit scenario 1. |
| 7. Per-app secrets and audit | P3-06, build-time secrets decision | `build-secret-canary`, cross-tenant fixture, secret rotation/redaction assertions; exit scenarios 3 and 7. |
| 8. Deployment history, health and rollback | P3-04, P3-07, deploy state-machine decision | `release-tamper`, `health-failure`, cancellation/restart and immutable prior release; exit scenarios 1–2 and 7. |
| 9. Optional rootless containers | P3-08, runtime/toolchain provenance decision | Separate container-profile escape, digest, network, volume and resource-limit tests; exit scenarios 3–5 when that optional profile is claimed. Native runtime acceptance remains independently required. |
| 10. Signed declarative app catalogue | P3-09, catalogue trust/removal decision | `recipe-untrusted`, signature/revocation/digest/permission checks, data-preserving removal; exit scenarios 5–6 and 8. |

Any change to the phase plan must update this crosswalk and its linked test evidence in the same reviewed change. A framework, OS, architecture, or optional container profile is out of the compatibility claim until its corresponding exact-head test record exists.

## P3 contract and test specifications

Use disposable Linux VMs corresponding to the eventually approved OS matrix. No test may target the developer workstation, a live server, production credentials, or a non-disposable shared host. Test fixtures must be synthetic. Start from a clean snapshot, capture evidence, and destroy the VM after the run. Never relax isolation merely to make a scenario pass.

### P3-01: Runtime pinning and compatibility

- Define a small explicit Node.js LTS and Python version matrix; pin exact runtime/package sources and checksums/digests per app release.
- Build/deploy a minimal Node HTTP app and Python ASGI app from known fixture revisions. Record runtime, architecture, dependency lock state, build artifact digest, and health endpoint.
- Verify per-app runtime selection does not modify system Python/Node packages and that two apps can pin different supported versions concurrently.
- Reject unsupported, changed, or unverifiable runtime artifacts before executing them; preserve prior healthy deployment.
- Evidence: compatibility table, clean guest logs, artifact digests, negative tamper test, and rollback result.

### P3-02: Git credentials, webhook authenticity, and deploy idempotency

- Use a disposable repository and least-privilege, read-only deploy credential. Prove token material never appears in process arguments, URLs, build output, panel logs, audit records, error pages, or support bundles.
- Accept a valid signed webhook once. Reject invalid signature, missing signature, wrong repository/ref, stale timestamp, replayed delivery ID, oversized body, and malformed payload without queuing a build.
- Revoke/rotate the credential and verify future fetches fail closed while historical logs remain redacted.
- Verify cancellation and retry do not duplicate publication or leave ambiguous current-release state.
- Evidence: captured redacted audit/log assertions, replay/negative test results, and exact credential scopes (never the secret value).

### P3-03: Isolated build execution and hostile build inputs

- Run builds under a dedicated unprivileged per-app/build identity or the reviewed isolated runner contract; no root, host-root mounts, Docker socket, panel DB, agent socket, signing key, or other tenant paths.
- Apply explicit CPU, memory, disk, wall-clock, process-count, and network policy. Test each limit independently and in combination; the worker and host must remain healthy after termination.
- Hostile fixtures attempt cross-tenant reads, process inspection/signals, symlink/path traversal, archive traversal/bomb expansion, metadata-service/internal-network access, credential scraping, fork/resource exhaustion, and persistence after build cleanup.
- Default network policy is deny or an explicitly reviewed allowlist; test DNS/IP rebinding and redirects if outbound access is permitted.
- Verify abort, timeout, worker crash, disk pressure, and retry leave no published partial artifact and no orphan process/mount/reservation.
- Evidence: isolation boundary review, adversarial test matrix, measured limits and host health before/after, cleanup and retry state.

### P3-04: Immutable artifact and release history

- Publish a build only after successful completion and integrity verification; associate source revision, runtime/dependency lock, builder identity, output digest, and deployment job.
- Confirm failed builds never replace the active artifact. Concurrent deploys serialize or fail with a clear conflict; duplicate idempotency keys resolve to one operation.
- Retain immutable release history sufficient for rollback. Reject missing, altered, cross-app, cross-tenant, or untrusted artifacts.
- Verify retention and deletion behavior against backup/restore policy; do not silently remove user data when removing an app recipe.

### P3-05: Runtime identity, process manager, ports, and resource controls

- Run Node/Python processes as dedicated site identities with reviewed systemd sandbox/resource controls. Assert effective UID/GID, filesystem visibility, capabilities, namespaces, limits and writable paths from inside a test process.
- Reserve ports centrally. Test collision, stale reservation, restart, simultaneous allocation, and rollback after failed start; no listener should bind unintended interfaces.
- Exercise start, stop, restart, crash loop, health timeout, resource exhaustion, host reboot, and graceful shutdown. Check bounded restart behavior and operator-visible diagnostics.
- Verify process arguments, environment inspection, `/proc` visibility, and logs do not disclose another app's secret or command content.

### P3-06: Per-app secrets and audit

- Store secret references through the approved secret boundary; expose values only to the intended app process. Test cross-tenant/app denial, rotation, deletion, failed decryption, and backup/restore behavior.
- Inject canary secrets and assert absence from API responses, build output, deploy logs, runtime logs, exceptions, process listings visible to other tenants, and audit exports.
- Audit actor, app, action, release, and outcome without recording secret values. Ensure authorization is rechecked at deploy-job execution time.

### P3-07: Routing, health checks, rollback, and TLS integration

- Only after P2 routing/TLS is accepted, deploy Node and Python fixtures behind a custom domain and verified HTTPS route. Assert host/path routing cannot cross app or tenant boundaries.
- Test health success, wrong response, timeout, dependency failure, and app crash. Failed health gates must preserve old traffic/release and report a bounded remediation path.
- Upgrade to a new release, verify externally, then rollback to a pinned previous artifact. Test rollback when prior artifact is corrupt/unavailable and ensure the active route is not silently pointed at an unverified release.
- Confirm domain ownership/TLS and existing P2 proxy controls are not bypassed by app-provided Host, forwarded headers, or arbitrary target URLs.

### P3-08: Optional rootless-container profile

- Treat this as optional, independently gated scope. No Docker socket, privileged mode, host PID/network namespace, host root mount, or unreviewed device access.
- Pin image digests and verify provenance; reject mutable-only tags and signature/digest mismatch by default.
- Test restricted network, volume allowlists, cgroups/resource limits, user namespace behavior, teardown, restart, and hostile container escape attempts in disposable guests.
- If runtime support or isolation cannot be proven on a candidate OS, mark that profile unsupported and keep it disabled; this does not excuse failure of the native Node/Python profile.

### P3-09: Declarative catalogue and recipes

- Catalogue definition must be data-only and schema-validated; default deny untrusted sources. Verify signature, immutable version/digest, declared permissions, resource requirements, licence/provenance, health check, backup/removal behavior, and rollback metadata.
- Malformed, unsigned, replayed, revoked, overprivileged, licence-unknown, or digest-mismatched recipes fail before installation. Signature key rotation/revocation and stale metadata are tested.
- Removing a recipe must not delete customer-created data unless a separately authorized, explicit, recoverable operation chooses that outcome.
- Start with a narrow supported compatibility list; adding a framework or marketplace claim requires its own repeatable test evidence.

## Phase exit dossier

P3 can be proposed for acceptance only when all prerequisite rows are proven and every in-scope P3 acceptance scenario has exact-commit evidence:

1. Node app from Git, custom domain/TLS, verified route, upgrade and rollback in a lab.
2. Python ASGI app, isolated dependency/runtime version, health and route verification.
3. Two hostile apps cannot read each other's secrets, processes, backups, files, or artifacts.
4. Build resource limits fail gracefully and leave the host healthy.
5. Unverified public tags and untrusted app definitions cannot install by default.
6. Recipe removal preserves user-created data absent a separately authorized backup/export/delete decision.
7. API/job/agent tenant reauthorization, audit/redaction, crash/cancellation/retry, and rollback tests pass.
8. Supported OS/architecture and framework matrix is limited to measured configurations; dependency licences/provenance and security review are recorded.
9. User-facing deployment status, health, logs, and error recovery have accessibility and manual developer/operator acceptance.

Each result must be classified **pass / fail / not run / excluded with rationale**. Any cross-tenant access, leaked canary secret, host breakout, untrusted artifact execution, or non-recoverable failed deployment blocks exit. Do not use a percentage-complete claim.

## Manual developer acceptance walkthroughs

Run only after P1/P2 entry gates, the P3 runner boundary, and the relevant UI/API contracts have been accepted. These steps describe outcomes rather than prescribing unreviewed screens or endpoints. Use a disposable lab, synthetic tenants, a disposable Git repository, synthetic credentials, and lab-only DNS/TLS. Never copy production secrets or point tests at a live host.

### Node Git deployment and rollback

1. As a tenant developer, create an app from the pinned Node fixture repository and choose a supported runtime. The app is visibly scoped to the current tenant and repository credential access is least-privileged.
2. Deploy a known-good commit. Confirm the UI/job history identifies the source revision, runtime pin, build, artifact digest, health result and active release. Verify the app through its assigned lab hostname/TLS route.
3. Push a second valid commit and deploy it. Verify the new release becomes active only after its health check passes; retain the previous release for rollback.
4. Push a commit that intentionally fails readiness. Confirm the deployment reports a bounded error and remediation hint, the prior release continues serving, and the failure log contains no canary secret or Git credential.
5. Roll back to the pinned previous release. Verify route content and release history agree, and the rollback is attributed and auditable.
6. Rotate a synthetic runtime secret and repeat a deployment/restart. Verify only the intended app receives the new value; API views, build output, process listings visible to another tenant, runtime logs and audit exports do not expose it.
7. In a second tenant session, attempt to enumerate the app, inspect its job/log/history/artifact, and access its route-management action. Every cross-tenant request is denied without leaking whether protected objects exist.

### Python ASGI deployment and dependency isolation

1. As a different tenant developer, deploy the pinned FastAPI/ASGI fixture with the approved Python runtime and its lockfile.
2. Verify health and lab hostname/TLS routing. Record runtime and dependency digests; confirm no host Python packages changed.
3. Deploy a second app using a different supported Python pin and dependency set. Confirm each app starts under its assigned identity and neither can read the other's tree, process secrets or artifacts.
4. Submit a package/runtime digest mismatch and an unsupported runtime. Both must fail before execution and leave the healthy route/release unchanged.
5. Trigger a bounded build limit and a failed readiness check. Verify job state is clear, the previous route remains healthy, and cleanup leaves no process, workspace reservation or partial artifact.

### Operator review during both walkthroughs

- The app owner can find deployment state, source revision, health, relevant redacted logs, and a concrete recovery action without access to another tenant's information.
- An operator can distinguish build failure, health failure, route failure, unsupported runtime and policy denial; diagnostics must not suggest disabling security controls.
- Keyboard-only operation and accessible status announcements work for long-running deploy, health, failure and rollback states. Any UI-only claim is labeled until the backend operation is verified.
- Observe guest CPU, memory, disk and process health before, during and after hostile/resource-limit fixtures. Record actual measurements; do not invent thresholds in this plan.

For every walkthrough, record operator role, tenant fixture IDs, source commit, environment and guest image digests, browser/runtime versions, exact job/test IDs, route verification method, expected/actual results per step, redacted evidence locations, reviewer and exclusions. A successful manual walkthrough complements automated negative tests; it does not replace them or establish phase acceptance by itself.

## Current next action

Maintain this as a reviewable P3 design/test artifact while P1/P2 remain gated. The next implementation action must be selected only after GitHub records the P1 exit decisions and the complete P2 contracts stabilize. Recheck open PRs and current branch heads immediately before that selection to avoid duplicating active work.
