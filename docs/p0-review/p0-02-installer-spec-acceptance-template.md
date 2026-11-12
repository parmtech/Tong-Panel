> **Preservation draft:** historical working evidence; factual statements and links require review. No approval or phase completion is implied. Machine-local paths and private coordination details have been omitted.

# P0-02 Installer Specification Acceptance Record — Blank Template

**Status:** BLANK TEMPLATE — no installer design acceptance, OS certification, security signoff, release decision, or P0 acceptance is recorded here.

This record is for reviewing the installer specification and its documentation claims against exact revisions and evidence. The canonical P0 plan labels UX and installer specifications P0-02; issue #4 is a related UX-flow anchor, not installer approval. The current specification says it is proposed and that no installer exists. Keep candidate validation targets distinct from supported or certified platforms.

This record covers only the P0-02 installer-spec slice. It does not by itself satisfy issue #2, which also requires reviewed security-design deliverables linked to acceptance tests and P1 work items, with findings recorded in `UPDATES.md`.

## 1. Exact review inputs and scope

- Repository:
- Canonical baseline SHA: `0aad68e81138a0f226e83e401f90bae4d0b6bddc` (verify before review):
- `docs/phases/P0-PLANNING.md` revision reviewed:
- `docs/INSTALLER_SPEC.md` revision reviewed (expected blob at baseline: `7eeff85c57173ee5db481fb176d25bb80e7d133a`):
- README / AGENTS / PHASE_INDEX / other installer entrypoint revisions reviewed:
- Related ADRs and exact revisions reviewed (including ADR-005/008; include ADR-006 when optional mail/DNS is in scope, otherwise record explicit out-of-scope status):
- Related P1/P2/P3/P5 plans, checklists, or artifacts and exact revisions:
- Issue #2 security-design linkage: exact P0-06 review revision, covered flows/threat IDs and reviewer dispositions; mapped P1 work-item IDs/revisions, linked acceptance-test contract revisions, and their dispositions:
- Related P0-08 installer/recovery routing: exact T12/T15 route status and phase-entry gate, or explicit unmapped/downstream status; P0-08 crosswalk revision/hash and linked P0-06 contract/disposition references:
- Issue #2 exact body/revision and explicit Acceptance clause reviewed (including required `UPDATES.md` security findings reference); keep distinct from installer/recovery Deliverables and cite any authorized owner-adopted acceptance extension or updated issue text:
- Exact `UPDATES.md` entry/anchor and revision where installer findings are reported (leave blank until reviewed findings exist):
- Exact source/spec/design revisions and retained evidence locations:
- Related P0-09 provenance record ID/revision/hash for any included bootstrap or distribution artifact, or explicitly open/not applicable; this reference does not block review of a proposal-only specification:
- Intended product profile, OS/architecture/runtime and resource scope under review:
- Exact owner-accepted P0-07 OS/architecture/runtime/resource profile decision revision and status (or explicitly open/unknown); keep candidate validation targets distinct from an accepted profile:
- Explicitly excluded profiles, services, distributions, and later decision gates:
- Review date/time (UTC):

## Source-audit evidence supplied for verification (not a disposition)

The following source-audit note is retained unchanged as review input. The reviewer must verify its cited revisions and evidence for the scope being assessed; it does not prefill this record's findings or acceptance decision.

> Source audit (pinned local `main@0aad68e81138a0f226e83e401f90bae4d0b6bddc`): P0 task P0-02 is marked `[x]` for producing “UX and installer specifications, docs entrypoint” at `docs/phases/P0-PLANNING.md:9–13`; this is planning-artifact completion, not an install/acceptance pass. The pinned root `README.md` links `docs/INSTALLER_SPEC.md` at line 38 using the label “Verified fresh-Linux setup, one-command goal, recovery and update strategy,” which may imply completed verification despite README line 49 saying installation is not available and INSTALLER_SPEC lines 1–2 calling the document proposed/no installer. The current uncommitted README at line 39 changes that label to “Proposed fresh-Linux setup”; this is a local draft correction, not a published or owner-approved change. `docs/README.md` is absent (no docs README candidate). README line 7 calls the panel/installer nonfunctioning. No installer success or support certification is claimed.
>
> Concrete source inconsistency: INSTALLER_SPEC.md:7–8 proposes initial validation of Debian 12/13 + Ubuntu 24.04 x86_64; canonical ADR-005 at `docs/ADR_REGISTER.md:32–36` orders Ubuntu 24.04 and Debian 12 first and Debian 13 only after verified CI. P0-PLANNING exit criterion at lines 25/28 requires no contradictory stack/install claims and distro/runtime promises remain proposed until clean VM CI. Treat spec’s Debian 13 timing as unresolved P0-07, not a support promise. Also ARCHITECTURE/BLUEPRINT wording conflicts are already in P0-07 findings.
>
> Objective evidence still absent from INSTALLER_SPEC (it is a requirement list, not results): signed immutable artifact/hash/key verification; successful + negative preflight runs; fresh clean-VM install on selected exact OS/arch/resource profiles; upgrade/resume/uninstall/backup-restore/rollback; failure injection at each journal checkpoint (disk/network/package/interruption/existing panel/firewall/DNS/ACME/revoked key/restore); SSH survivability and no data loss; permission confinement; exact checks/log/artifact hashes. These are listed as future requirements in lines 11–19, 27–34 but no executed evidence is recorded. Current UPDATES lines 11–12 state P0 remains unaccepted and P1 experimental; line 21 records the installer cross-gate; lines 27–28 distinguish proposed support/resources from exit evidence. No edit made.

## 2. Review authority and disposition

- Decision owner / maintainer, name, role, and authority:
- Installer/release reviewer, name, role, and authority:
- Independent security reviewer, name, role, and authority (if required):
- QA/operations reviewer, name, role, and authority:
- Reviewer independence / conflicts:
- Review method and exact evidence examined:
- Overall specification disposition: [ ] Accept for named design scope  [ ] Needs revision  [ ] Reject  [ ] Defer
- Exact scope and conditions covered:
- Findings, required changes, and follow-up issue/document references:
- Decision record location and immutable revision:

## 3. Entrypoint and documentation-claim review

| Path / claim | Exact source revision and line/section | Expected wording or behavior | Observed wording / link / command | Finding, correction, or limitation | Reviewer disposition |
|---|---|---|---|---|---|
| Installer spec status and statement that no installer exists | | | | | |
| README / docs entrypoint / install links | | | | | |
| Commands, package names, URLs, and CLI names exposed to users | | | | | |
| Supported/unsupported distribution and host wording | | | | | |
| P1 foundation versus later hosting/install capability claims | | | | | |

- Broken, stale, or misleading links and exact tested target:
- Any fictitious/unimplemented install or update command presented as usable:
- User-facing language that could imply an installer, successful setup, support, or production readiness without evidence:
- Corrective document owner and required retest/review:

## 4. Platform, profile, and support-claim matrix

The installer specification lists candidate validation targets, not certified support. For each row, record the exact source claim, the evidence status, and the authorized owner decision. Do not infer support from a runner, component test, checksum, or candidate lab result.

| OS/distribution and version | Architecture | Host/container assumptions and profile | Source wording at exact revision | Evidence available (image digest, clean VM, tests) | Status (candidate / tested / supported / unsupported / unknown) — reviewer to determine | Owner disposition and conditions |
|---|---|---|---|---|---|---|
| Debian 12 | | | | | | |
| Debian 13 | | | | | | |
| Ubuntu 24.04 LTS | | | | | | |
| arm64 candidate | | | | | | |
| AlmaLinux / Rocky candidates | | | | | | |
| Other or excluded target | | | | | | |

- Exact candidate test order and rationale (blank pending owner decision):
- Distinguish initial test order from supported/certified promise:
- Distro/package-manager, SELinux/firewall, kernel capability, virtualization/container exclusions:
- CPU, memory, disk, concurrency, runtime/toolchain assumptions and evidence, or explicitly unknown/out of scope:
- Required clean-image sources and immutable image digests:
- Exact evidence required before changing a candidate to supported:
- User-facing wording for untested, unsupported, or partially tested targets:

## 5. Installation and preflight design review

For each design step, record the exact specification section, expected behavior, evidence examined, failure handling, and disposition. Blank fields do not assert that the mechanism exists.

| Step / requirement | Exact spec section/revision | Required precondition and behavior | Evidence or acceptance scenario | Expected safe failure / recovery | Finding and reviewer disposition |
|---|---|---|---|---|---|
| Signed immutable release and HTTPS/hash/signing-key verification before elevation | | | | | |
| OS/architecture, privileges, clean-host, hostname/DNS, resources, ports, firewall, package sources, kernel and SSH preflight | | | | | |
| Detect blocked outbound TCP port 25 and warn before optional mail setup; record opt-in/refusal behavior and safe outcome | `docs/INSTALLER_SPEC.md` §Profile, line 25 | | | | |
| Verify externally reachable HTTPS before reporting successful HTTPS setup; pending or failed verification must not produce a success signal | `docs/INSTALLER_SPEC.md` line 28 | | | | |
| Non-destructive report and exact change preview | | | | | |
| Explicit interactive consent / documented unattended opt-in | | | | | |
| Mutation journal, staging, service users, configuration tests, TLS and admin identity | | | | | |
| Unique short-lived bootstrap secret; strong credentials/MFA; no universal password | | | | | |
| Health/permission checks before reporting completion | | | | | |
| Failure diagnostics, state preservation and refusal of destructive co-install | | | | | |

- Optional profiles/services and default-disabled behavior:
- Root/elevation boundary and least-privilege assumptions:
- Preflight false positive/negative handling and operator override rules:
- Consent and unattended-mode controls, audit, and abort behavior:
- SSH/operator access preservation and recovery channel assumptions:
- Secrets, signing keys, logs, support bundles, and diagnostic redaction assumptions:
- Exact unsupported, unsafe, or conflicting host behavior:

## 6. Upgrade, update, uninstall, rollback, backup, and recovery

| Lifecycle path | Exact documented/design revision | Preconditions and data/state affected | Failure/interruption scenarios | Rollback/recovery and operator guidance | Evidence / reviewer finding |
|---|---|---|---|---|---|
| Upgrade from previous release | | | | | |
| Update and signature/key revocation handling | | | | | |
| Resume after interrupted installation/update | | | | | |
| Uninstall and retained-data choices | | | | | |
| Backup and restore | | | | | |
| Rollback after failed health/configuration check | | | | | |
| Reinstall / partial prior state / existing panel | | | | | |
| Rerun after a successful install: explicitly decide whether idempotent behavior is required, supported, deferred, or out of scope | | | | | |

- Exact journal checkpoints and reversible/irreversible actions:
- Data and configuration preservation/deletion semantics:
- Recovery when package verification, network, package manager, disk, DNS, TLS, or backup target fails:
- Conditions under which rollback is impossible, and required user warning:
- Evidence required for each claimed recovery path and exact test environment:

## 7. Acceptance scenarios and clean-VM evidence plan

For P0-02 specification review, specify the proposed target image/profile and immutable image digest when selected, source and planned artifact revisions, command/script, expected successful or negative result, before/after state, planned evidence location, and reviewer disposition of the test contract. Leave actual result/date blank; do not record planned tests as executed evidence. At the applicable implementation/CI gate, retain an executed fresh-install scenario for each owner-selected image/profile.

| Acceptance/failure scenario | Proposed/selected image digest/profile and test contract revision | Planned command/script and source/artifact revision | Expected result, before/after state, and planned evidence artifact | Actual result/date — leave blank until applicable implementation/CI gate | Reviewer disposition of planned contract / limitation |
|---|---|---|---|---|---|
| Successful fresh install on each owner-selected image/profile | | | | | |
| Selected-profile disposable-host end-to-end integration for Nginx, PHP, database, and agent services (TEST_STRATEGY CI lane 5); record service health and cross-service success/failure or explicit reviewed defer/out-of-scope disposition | | | | | |
| Externally verify HTTPS before reporting setup success; pending or failed verification must not signal success | | | | | |
| Optional services remain disabled unless explicitly selected; record selected/not-selected scenario and state | | | | | |
| Detect blocked outbound TCP port 25 and warn before optional mail setup; record observed result or owner-reviewed out-of-scope disposition | | | | | |
| Rerun after successful install, if required by owner disposition | | | | | |
| Unsupported distro or architecture | | | | | |
| Occupied ports 80/443 or conflicting service/existing panel | | | | | |
| Disk full / insufficient CPU, memory, or storage | | | | | |
| Package repository or network failure | | | | | |
| Interrupted package installation at each mutating journal checkpoint (create one matrix row per checkpoint below) | | | | | |
| Dangerous firewall change / SSH session loss | | | | | |
| Bad hostname/DNS or pending/failed ACME challenge | | | | | |
| Revoked, wrong, missing, or substituted signing key/artifact | | | | | |
| Broken backup target or failed restore | | | | | |
| Failed health/configuration/permission check after changes | | | | | |
| Reinstall, upgrade, resume, rollback, or recovery failure | | | | | |

For P0-02 specification review, enumerate each mutating checkpoint and record its proposed failure-injection contract, expected safe state and recovery, evidence to capture, and planned implementation/CI gate. Execution results are not a P0-02 prerequisite; at the applicable implementation/CI gate, each in-scope checkpoint requires its own result or an explicitly reviewed defer/out-of-scope disposition. One generic interruption test does not establish coverage of every checkpoint.

| Exact journal checkpoint ID/name and source revision | Mutation / invariant | Injected failure point and command | Expected safe rollback/resume result | Exact image/artifact/logs and before/after state | Actual result/date — reviewer to determine | Reviewer disposition / limitation |
|---|---|---|---|---|---|---|
| (add one row per mutating checkpoint) | | | | | | |

- Clean-VM image and checksum sources:
- VM isolation and proof that no production/real host was changed:
- Test operator / participant identity, role, and authority to operate the disposable target (or state automated / no participant):
- Expected-state oracle owner and authority; result reviewer identity/role/authority verifying each observed result; record independence/conflicts when required by the applicable gate or adopted by the authorized owner:
- Test matrix, parallelism and reproducibility:
- Missing scenarios or platforms and later gate:
- Exact logs, before/after state, artifact hashes, and reviewer access controls:

## 8. Security assumptions and review boundary

| Assumption / control | Exact design/source revision | Evidence and threat/ADR linkage | Unresolved question / owner to determine | Security reviewer disposition |
|---|---|---|---|---|
| Artifact authenticity, signature trust root, key custody/rotation/revocation | | | | |
| Elevated execution, privilege separation, service identities and capabilities | | | | |
| Preflight safety, explicit consent, change preview, and refusal behavior | | | | |
| Journal integrity, crash consistency, rollback and recovery | | | | |
| TLS, bootstrap credentials, MFA, secrets and redacted diagnostics | | | | |
| Firewall/SSH survivability and emergency recovery | | | | |
| Package sources, downloaded payloads, provenance and supply-chain controls | | | | |

- Related canonical threat IDs, ADR IDs, and exact revisions:
- Independent security review scope and reviewer authority:
- Security findings, unresolved risks, exclusions, and required follow-up:
- Explicit privileged-agent/host-action scope, if applicable, and separate approval reference:
- This specification review does not substitute for threat-model approval or P0-06 review:

## 9. Findings, exclusions, and follow-up

| Finding ID / priority | Exact source/evidence reference | Required change or accepted limitation | Responsible owner / follow-up issue | Retest/review condition and disposition |
|---|---|---|---|---|
| | | | | |
| | | | | |

- Features/services explicitly planned only, disabled, unsupported, or out of scope:
- Evidence unavailable or not independently reproduced:
- Conditions requiring separate ADR, security review, licence review, or owner decision:
- Follow-up documents/issues, responsible reviewer, and trigger for reconsideration:

## 10. P0-02 installer-spec disposition

- P0-02 installer-spec disposition: [ ] Met for named specification scope  [ ] Not met  [ ] Deferred
- Exact criteria reviewed and evidence basis:
- Remaining contradictions, claims, or required documentation changes:
- Named decision owner, role/authority, and date (UTC):
- Installer/release reviewer disposition, name/authority, and date:
- Independent security reviewer disposition, if required, name/authority, and date:
- Exact final record revision / immutable commit:

### Separate issue #2 installer/OS/firewall-recovery plan disposition

This row evaluates the proposed recovery and failure-injection plan against the installer/OS/firewall-recovery Deliverables in issue #2 and any exact acceptance extension explicitly adopted by the authorized owner or added to the issue body. Issue #2's current explicit Acceptance clause addresses reviewed security design, linked tests/P1 work items, and `UPDATES.md` findings; it does not explicitly name installer recovery. Record plan readiness only here; it does not record installer execution, security-design acceptance, OS support, or overall issue #2 acceptance. Cross-reference P0-06 security-design findings separately and use P0-08 only for owner-confirmed work-item routing. Actual execution evidence belongs to the later implementation/CI gate.

- Exact issue #2 body revision and retrieval evidence; separately cite the relevant Deliverables text, explicit Acceptance clause, and any authorized acceptance extension/updated body:
- Exact planned scenario/checkpoint matrix revision and covered profiles, failure cases, expected safe state and recovery:
- P0-06 security-design record ID/revision and related threat/test-contract references (or scoped N/A rationale):
- P0-08 crosswalk revision/hash, owner-confirmed P1 work-item IDs and linked acceptance/test contracts (or explicit unassigned/unmapped status):
- Plan disposition for this installer/OS/firewall-recovery slice: [ ] Adequate as a plan for the exact named scope  [ ] Needs revision  [ ] Deferred
- Findings, limitations, missing scenarios/contracts, and required follow-up:
- Authorized owner/reviewer identities, authority, scope and date (UTC):
- UPDATES.md/checkpoint reference for the recorded plan disposition:
- Exact final record revision / immutable commit:
- Actual execution results: not recorded here; record only at the applicable implementation/CI gate against exact image/artifact/profile.

### Separate P0-02 documentation-entrypoint disposition

The P0 plan groups the docs entrypoint with UX and installer specification deliverables. UX acceptance is recorded in its companion form; this row records only the docs-entrypoint slice, so neither form silently closes the other slices.

- Docs-entrypoint disposition: [ ] Met for named entrypoint scope  [ ] Not met  [ ] Deferred
- Exact entrypoint paths/revisions, links and navigation criteria reviewed:
- Findings, broken/stale claims or accepted limitations:
- Evidence and review method:
- Authorized decision owner, reviewer authority and date (UTC):
- UPDATES.md/checkpoint reference for the recorded result:
- Exact final record revision / immutable commit:

## Boundary and limits

This record evaluates specification and documentation evidence only. It is not installer approval, OS support certification, security signoff, release/distribution permission, implementation authorization, merge/deployment authorization, or P0 acceptance. A clean-VM test result applies only to its exact image, artifact, profile, and scenarios. Every result and decision remains blank until an authorized owner and required reviewers record it against exact revisions and evidence.
