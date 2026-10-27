# P2 Safe File Access Prototype

**Status:** isolated Linux-only prototype in draft PR #35. It is not an API, file manager, tenant authorization layer, or production filesystem service.

## Contract

- `files.OpenRoot` pins a configured directory descriptor and refuses a symlink at the final root component. The configured root and its parent path must be administrator-controlled.
- `Root.ReadFile` accepts only a normalized relative path, opens beneath the pinned root with Linux `openat2`, and forbids symlinks, magic links, and crossing nested mount points. No fallback is attempted if the kernel does not support the required syscall/resolution flags.
- `Root.WriteNew` creates a new regular file inside a random mode-0700 staging directory at the pinned root, then uses `renameat2(RENAME_NOREPLACE)` into the destination directory. This prevents a tenant-writable destination directory from swapping the temporary source path. It uses mode 0600, caps content at 16 MiB, and never overwrites an existing file or symlink. A parent-directory sync error can occur after the rename, in which case the file may be present while durability remains unknown.
- A process crash can leave a private staging directory; automated orphan cleanup is not yet implemented.
- `TenantRoots` copies a fixed server-owned tenant-to-root mapping and selects only the assigned root. The caller must supply a tenant identifier derived from a verified server principal; this helper does not authenticate or authorize that principal.
- Reads are limited to 16 MiB maximum, and only regular files are returned. FIFOs and devices are not treated as readable file content.
- `Close` is synchronized with reads; reads after close fail.
- The package is not wired to HTTP, the privileged agent, site identities, quotas, or ACLs. It must not be represented as cross-tenant file access control.

## Tests

Linux tests cover parent traversal, absolute/non-normalized paths, symlink and root-symlink rejection, directory/FIFO inputs, byte limits, close behavior, tenant-root selection, unknown-tenant denial, atomic create-only writes, concurrent single-winner creation, and concurrent reads/writes while an attacker repeatedly swaps paths for symlinks to an outside secret. Successful operations must never expose or change outside-root fixture data; kernel errors are expected to fail closed.

## Remaining P2 work

This primitive does not satisfy the P2 Files UI/API or end-to-end two-tenant isolation gate. `WriteNew` supports creation only; update/delete, upload quotas, archive extraction, file metadata policy, orphan staging cleanup, and concurrent-edit semantics are not implemented. The tenant mapping is not an authorization boundary until integrated with reviewed identity policy. The phase still depends on reviewed P1 identity/job/host-agent boundaries, installer and supported-OS lab, service adapters, scoped ACLs/quotas, and end-to-end restore/security evidence. No host mutation or deployment is included.
