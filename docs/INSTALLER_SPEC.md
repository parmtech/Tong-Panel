# Tong Panel — Safe One-Command Installer Specification
Status: proposed. THERE IS NO INSTALLER YET. Do not run fictitious commands.

## Goal
A documented single-entry installation on **verified supported fresh Linux VPS or dedicated servers**. Tong Panel cannot be installed inside third-party shared hosting to replace provider-level cPanel without root access.

## Initial compatibility plan (subject to tests)
Validate Debian 12/13 and Ubuntu 24.04 LTS on x86_64; add arm64 only after independent tests. AlmaLinux/Rocky later after separate package/SELinux/firewall checks. Do not market “any Linux” or promise unsupported VPS/container hosts.

## Installer chain
1. Official signed immutable release, HTTPS, verified hashes and signing key before elevated execution.
2. Preflight checks OS/architecture, root, clean host, hostname and DNS, CPU/memory/storage, critical ports, firewalls, package sources, kernel capabilities, SSH survivability.
3. Non-destructive preflight report and exact changes preview.
4. Explicit interactive consent, or carefully documented unattended opt-in.
5. Journal each mutation; staging, safe package/service setup, limited system users, config tests, TLS bootstrap and admin identity.
6. Bootstrap with unique short-lived secret, forced strong credentials and MFA; never embed a universal password.
7. Health check panel, web stack and permissions; only then signal installed.
8. Failed steps recover safely, preserve state, provide clear diagnostics, refuse destructive co-install.
9. Dedicated upgrade, resume, uninstall, backup/restore and rollback paths.

## Profiles
- Minimal web-only: panel + Nginx/PHP + DB choice.
- Standard hosting: optional mail/DNS and backups.
- App developer: Node/Python native, optional isolated containers.
- All optional services disabled until explicitly selected; detect blocked outbound port 25 and warn before mail setup.

## Must-pass failure cases
Unsupported distro, occupied 80/443 ports, disk full, apt/network failure, interrupted package install, existing panels, dangerous firewall change, bad DNS and pending ACME challenge, revoked signing key, broken backup target and failed restore. Do not report successful HTTPS setup until verified.

## Future CLI interface candidates
tongctl doctor, tongctl status, tongctl update, tongctl backup, tongctl restore, tongctl logs. These names are design ideas only.

## CI install validation
Test clean OS VM matrices and resource profiles, upgrades from previous release, failure at each journal checkpoint, reinstalls, recovery, partial network outages, permission confinement, no accidental data deletion and operator SSH access preservation.
