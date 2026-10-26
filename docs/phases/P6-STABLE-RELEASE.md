# P6 — Stability, Security Audit & 1.0 Release
**Status:** PLANNED, NOT IMPLEMENTED.  
**Depends on:** validated prior milestone scope; supported features from feature matrix audited.

## Release engineering
1. Freeze target scope and compatibility matrix; review incomplete/backward-incompatible functions.
2. Independent security review and penetration tests against root agent, tenant isolation, file manager, app/build execution, marketplace, recovery and API.
3. SAST/dependency/scanner CVEs, SBOM, reproducible build/signing and verified installation/update chain.
4. Clean install, in-place upgrade, failed upgrade/revert, interrupted install and uninstall across **all claimed OS/architectures**.
5. Staging disaster recovery: encrypted offsite backup, lost VPS restore, DNS/mail/DB/app/files reconciliation.
6. Accessibility WCAG 2.2 AA validation, beginner usability, screenreader/mobile/keyboard workflow tests.
7. Load/concurrency, fail/retry/idempotency, resource saturation and observability; publish measured limits and performance targets.
8. Documented help centre, onboarding walkthroughs, admin and developer docs, API/versioning compatibility, user support and incident disclosure.
9. Licences, copyrights, third-party NOTICE, trademarks, user-facing source offer and contributor/legal review.
10. Release notes, known limitations, signed artifacts/checksums, rollback docs, security support and deprecation policy.

## Hard release blockers
- Known unmitigated critical/high exploitable vulnerability.
- Root access via untrusted API/plugin or cross-tenant data exposure.
- Failed backup/restoration on supported stack.
- Destructive or unbounded package upgrades, no recovery path.
- Missing licence compliance for included components or exposed credentials.
- OS listed as supported without clean VM test evidence.
- Security/legal claims exceeding independent evidence.

## Go/no-go
Maintainer must record review, tests, external audit status, supported profiles and approved release artifacts. A release candidate is not automatically stable. “No known critical/high defects” is not a guarantee of 100% bug-free software.

## Follow-up
Maintain semver, vulnerability response, release channels, published compatibility support and continuous regression/security testing. Tong Panel remains open source; users supply their own infrastructure.
