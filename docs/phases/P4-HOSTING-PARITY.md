# P4 — Traditional Hosting Feature Parity
**Status:** PLANNED, NOT IMPLEMENTED.  
**Depends on:** P2 stable hosting lifecycle, P3 job isolation/secure app foundation.

## Objective
Cover major cPanel/WHM-style hosting workflows in Tong Panel's original interface, with published explicit compatibility limits. “Parity” is the workstream, not a certification that every proprietary feature can be replicated.

## Feature slices
1. **Email service**: opt-in Postfix and Dovecot, mailboxes, aliases/forwarders, auto replies, IMAP/SMTP config wizard, queue/rate/abuse controls, spam and optional malware, external-relay support, mailbox migration and webmail integration.
2. **Email authentication**: DKIM key rotation, SPF/DMARC DNS validation, MX records and PTR checks, TLS certificates, deliverability diagnostics, blacklisting guidance, cloud/VPS port-25 detection. No guaranteed deliverability.
3. **DNS service**: optional PowerDNS/BIND authoritative zone management with DNSSEC; externally hosted zones via scoped provider API credentials, TTL propagation and import/export.
4. **Advanced databases**: PostgreSQL and optional Redis/Valkey, user privilege scoping, database backup/restore and guarded operations.
5. **Account management**: resellers, hosting packages, team members, quotas, suspend/restore, usage accounting, ownership, SSH/SFTP keys, FTP adapter only where justified.
6. **Migration**: cPanel/WHM archive import with archive validation, list unsupported elements, dry run, server mapping, credential reset, data staging, test and cutover/rollback. Later Plesk/DirectAdmin import as independently tested adapters.
7. **Site operations**: Apache compatibility option where safe, granular PHP extensions and config, cron, rewrite rules, redirect checking, web statistics, app troubleshooting, log search and notifications.
8. **Operator interfaces**: scoped OpenAPI, token rotation, CLI, events/webhooks, backup and audit export.

## Key acceptance
- Email send and receive, queue troubleshooting and authentication record correctness on an eligible VPS; explain blocked SMTP ports.
- Split DNS deployment and external provider fail/timeout modes tested.
- Quota limits enforced at API AND privileged host adapter; reseller cannot escape packages.
- Migration safely rejects malicious or incompatible archives; dry run contains complete list of pending writes, destination conflict handling and test report.
- Database/email/backup restorations tested on isolated lab host, not assumed.
- Feature matrix mapped to tested examples and clear not-supported labels.

## Release gate
Demonstrate a tested full-hosting reference deployment, usable docs, known limitations, secure restore and tenant permissions; no guaranteed 1:1 parity with cPanel/WHM.
