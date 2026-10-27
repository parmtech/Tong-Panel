# Tong Panel — UX and UI Blueprint
Status: proposed; draft PR #14 contains a static preview and draft PR #19 proposes a panel-process status card. The status card reads only the local panel liveness and version endpoints; it is not host monitoring. Neither preview provides hosting operations or establishes full accessibility acceptance.

## Design language
An original premium responsive experience combining a smartphone/iPad-like home icon grid, Windows/macOS Finder-style file explorer, discoverable contextual controls, global search, helpful first-run tours, accessible interfaces, clear feedback and light/dark/auto themes. Familiar patterns WITHOUT copying proprietary cPanel designs, trademarks or source assets.

## Information architecture
- Home: Websites, Domains, Applications, File Explorer, Databases, Email, SSL, DNS, Backups, Monitoring, Server, Security, Accounts, Marketplace, Settings, Help.
- Desktop sidebar for Home, Activity, Sites/Apps, Files, Admin and help. Mobile bottom/drawer navigation.
- Global search with synonyms ("make email", "new WordPress", "Node app", "PHP version", "SSL redirect").
- Customer roles never expose inoperative administrator-only controls. Permissions are enforced server-side.

## File Explorer: beginner-friendly yet powerful
- Folder tree, breadcrumbs, list/grid, details/preview pane, recent locations, bookmarks, search, sort and filters.
- Single select, multi-select, long-press/right-click context menu, drag/drop, copy/cut/paste, duplicate, move, rename, create folder/file.
- Chunked resumable uploads, downloads, file editor with line numbers, safe image/document preview, archive extract/compress, trash/restore, explicit permanent delete, permission display and storage usage.
- Desktop-style keyboard navigation and supported shortcuts, touch-friendly controls, accessible labels and focus state.
- **Safety:** per-tenant filesystem root, validation of canonical paths and symlinks, archive path traversal defense, no privileged execution on uploaded content.

## Guided wizards
- New site: choose domain -> runtime/template -> optional database -> Git/import -> HTTPS -> review -> deploy -> health checks.
- Node app: Node LTS, source/Git, startup and build templates, environment variables, health path, resources.
- Python app: Python version, venv, framework, WSGI/ASGI, secrets and reverse proxy.
- Mail: mailbox/forward/relay, SPF/DKIM/DMARC diagnostics, clear warning for port and reputation restrictions.
- SSL: existing cert detection, ACME challenge selection, DNS validation, renewal, HTTPS redirect external verification.
- Backup/restore: select data -> preview/dry-run -> verify archive and compatibility -> apply -> validate.
- Every long task displays queued/running/verified/pending/failed status, progress and repair guidance.

## Accessibility and performance
- WCAG 2.2 AA target, usable from keyboard and screenreader, strong contrast and motion reduction.
- Responsive on phone/tablet/desktop, optional compact/expert density, clear touch targets and localization/RTL-ready layout.
- Clear Save/Cancel/Undo, no surprise data loss on refresh, nonblocking notifications, redacted diagnostic export.
- Dynamic state rather than decorative fake controls. Animations only where meaningful and inexpensive.

## Screens (design acceptance)
1. Login/MFA/recovery and first-run tour.
2. Grid home and global search.
3. Websites, domain settings, SSL and DNS.
4. File Explorer (tree/list/grid/editor/multiselect).
5. Application marketplace, install flow, deployment logs and rollback.
6. Node/Python/PHP runtime manager.
7. Database, email and backup workflows.
8. Admin users/resellers/packages, quotas and audit.
9. Server and service health, firewall, updates and alerts.
10. Help centre, guided walkthrough and troubleshooting.

## Usability validation
Test representative beginners and admins. Common action should be discoverable from grid/search within three navigation steps (apart from form completion). Test mobile, keyboard, low-bandwidth, error/retry, large file lists and long operation workflows. Any function lacking a backend must be marked planned, not clickable pretending to work.
