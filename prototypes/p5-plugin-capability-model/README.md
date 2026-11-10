# P5 plugin capability model (prototype)

This standalone standard-library-only Go module models one narrow acceptance property: a disabled plugin grant and any mismatched plugin/install/tenant/resource/capability binding are denied. It has no network, persistent storage, host integration, secrets, sandbox, or Tong Panel runtime dependency.

`Grant` and `Request` must be created by a trusted policy/identity layer in any future implementation. The plugin must not provide either as authoritative input. This prototype does not define grant storage, signature verification, global plugins, capability schemas, cache invalidation, data preservation, or process isolation. Exact string matching is only a candidate decision rule for review; it is not a plugin security boundary or an approved architecture.

Run locally with the repository's pinned Windows Go toolchain:

```powershell
& 'C:\Users\~MaNBiR~\Documents\Codex\work\toolchains\go\bin\go.exe' test ./...
& 'C:\Users\~MaNBiR~\Documents\Codex\work\toolchains\go\bin\go.exe' vet ./...
```
