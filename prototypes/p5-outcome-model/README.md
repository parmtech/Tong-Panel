# P5 remote outcome model (prototype)

This standalone, standard-library-only Go module explores the P5 acceptance requirement that partitions and offline nodes must not produce ghost-success or duplicate effects. It has no networking, credentials, persistent storage, host integration, or runtime dependency on Tong Panel.

The proposed classifier reports success only when an acceptance response and effect verification are both present. A lost response, unverified acceptance, or reported failure after dispatch remains `Unknown`. Unknown outcomes may only be replayed in this model when the exact same idempotency key is reused and durable worker-side deduplication is assumed.

These are candidate safety semantics for discussion, not an approved protocol or an implementation contract. In particular, this prototype does not establish what effect verification means, how durable deduplication survives worker restore, or how authorization/revocation interacts with an in-flight operation. Those decisions require P0/P1/P5 review and the upstream gates recorded in `docs/P5_PREREQUISITE_AND_ACCEPTANCE_SPEC.md`. Do not connect this module to a real node or production system.

Run locally with the repository's pinned Windows Go toolchain:

```powershell
& 'C:\Users\~MaNBiR~\Documents\Codex\work\toolchains\go\bin\go.exe' test ./...
& 'C:\Users\~MaNBiR~\Documents\Codex\work\toolchains\go\bin\go.exe' vet ./...
```
