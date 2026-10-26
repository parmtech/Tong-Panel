# P5 — Multi-Server, Plugins & Ecosystem
**Status:** PLANNED, NOT IMPLEMENTED.  
**Depends on:** P4 production-like single-host workflows, resilient job model, secret rotation and incident procedures.

## Scope
- Remote agents with one-time enrollment, identity per server, mutual TLS, revocation, rotation and policy-scoped actions.
- Multi-node console: resource inventory, placement rules, health, machine labels, scheduled maintenance, staging/production separation.
- Safe inter-node workload migration, DNS/SSL cutover, app/DB backup handoff, rollback.
- Optional redundant DNS, mail and database architectures with documented failure boundaries, without pretending distributed state magically offers HA.
- Marketplace/public app templates and third-party plugin framework, manifest schema, signature/version requirements, capability permissions, moderated review, dependency update alerts and deprecation.
- Resource and tenant analytics, external billing/provisioning integrations, notifications and audit export.
- Standard APIs for ecosystem, app packages, community contributions, recipes and permissions.
- Additional Linux distribution / arm64 matrices **only after** separate automated verification.

## Acceptance
- Network partitions, offline nodes and rotated/revoked credentials don't cause ghost-success or privilege persistence.
- Wrong node/tenant job denied; no arbitrary remote shell.
- Migration rollback staged; data consistency validated.
- Plugin disable removes capability access, leaves recoverable user data, doesn't break panel boot.
- Registry supply-chain audit, staged review and signed/pinned release checks.

## Release gate
Clearly separate “multi-server management available” from “high availability supported”; document topology, supported recovery point/time objectives and failure handling based on measured tests.
