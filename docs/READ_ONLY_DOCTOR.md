# Tong Panel P1 — Read-only Host Doctor (Developer Preview)

**Status:** experimental; on `feature/p1-core-foundation`, draft PR #14. Not an installer; no distro certified.  
**Security:** the doctor opens a read-only `/etc/os-release`, `/proc/meminfo`, `/proc/net/tcp` and `/proc/net/tcp6`, and uses `statfs("/")`. It does **not** bind 80/443, scan public IPs, change packages, edit firewall rules, install or restart services, or need root. Running it cannot tell whether ports are reachable from the Internet.

## Developer command
From the checked-out development branch on Linux with Go 1.23 or later:

```bash
go run ./cmd/tongctl doctor
```

JSON contains:
- `OS`, `Version`, `Arch` and `Candidate`: an initial research matrix only, not installer compatibility certification
- `Warnings`: operator-readable unknowns and preliminary resource concerns
- `checks`: `os`, `memory`, `disk`, `ports` with `pass`, `warn` or `block` status

A `Candidate: true` says only that Ubuntu 24.04 or Debian 12 on amd64 fits the **proposed lab target**. It is not authority to run server setup. `memory` reports MemAvailable (not MemFree), `disk` uses root filesystem free blocks. Preliminary thresholds are 1 GiB *available RAM* and 8 GiB free disk for development guidance and are not measured minimum production requirements. If readings cannot be collected, report `warn` instead of a misleading pass. `ports` snapshots LISTEN states in `/proc`; cannot identify race conditions, cloud firewalls, ingress, domain DNS or TLS challenges.

## Tests performed
Local container: `go test -race ./...`, `go vet ./...`, `go build ./...` all passed after this change. Unit tests cover supported and unsupported OS identification, a simulated occupied HTTPS port, low/unknown resource states, malformed `/proc` records and a fixture that must remain unchanged. Check PR #14 GitHub Actions for separate remote CI evidence.

## Explicitly out of scope
- Network-facing installer, package mutation, sudo or automated firewall changes.
- Domain, DNS, SMTP, SSL, version probing, port-80/443 reserve/rebind.
- Signed releases, resource guarantees, virtual memory/cgroup headroom tests, process identification and distro lab certification.
- Root-service security review and complete installer rollback.

**Next preflight improvements:** kernel/cgroup/container detection, FQDN + DNS checks (with explicit consent), package conflict reporting, non-destructive SSH/firewall survival analysis, and resource profiles measured in disposable Ubuntu/Debian VMs. Do not mark issue #11 done until those are covered and reviewed.
