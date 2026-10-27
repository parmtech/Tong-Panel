# P2 Safe File Read Prototype

**Status:** isolated Linux-only prototype in draft PR #35. It is not an API, file manager, tenant authorization layer, or production filesystem service.

## Contract

- `files.OpenRoot` pins a configured directory descriptor and refuses a symlink at the final root component. The configured root and its parent path must be administrator-controlled.
- `Root.ReadFile` accepts only a normalized relative path, opens beneath the pinned root with Linux `openat2`, and forbids symlinks, magic links, and crossing nested mount points. No fallback is attempted if the kernel does not support the required syscall/resolution flags.
- Reads are limited to 16 MiB maximum, and only regular files are returned. FIFOs and devices are not treated as readable file content.
- `Close` is synchronized with reads; reads after close fail.
- The package is not wired to HTTP, the privileged agent, site identities, quotas, or ACLs. It must not be represented as cross-tenant file access control.

## Tests

Linux tests cover parent traversal, absolute/non-normalized paths, symlink and root-symlink rejection, directory/non-regular inputs, byte limits, close behavior, and concurrent reads while an attacker repeatedly swaps a regular file for a symlink to an outside secret. A successful read during the swap may return only the in-root fixture content; kernel errors are expected to fail closed.

## Verification

Code head `15dd7c412bbdfe34fa884d935c49cadf950750b7` passed [Go formatting, full race tests, and vet #40601259497](https://github.com/parmtech/Tong-Panel/actions/runs/40601259497), [daemon smoke #40601259446](https://github.com/parmtech/Tong-Panel/actions/runs/40601259446), and [unsigned archive lab #40601259524](https://github.com/parmtech/Tong-Panel/actions/runs/40601259524). Local Go tests were unavailable.

## Remaining P2 work

This primitive does not satisfy the P2 Files UI/API or two-tenant isolation gate. The phase still depends on reviewed P1 identity/job/host-agent boundaries, installer and supported-OS lab, service adapters, scoped ACLs/quotas, safe writes/archive handling, and end-to-end restore/security evidence. No host mutation or deployment is included.
