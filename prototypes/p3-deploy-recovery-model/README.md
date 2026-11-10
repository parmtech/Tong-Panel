# P3 deploy recovery model (non-normative)

This is a tiny, in-memory executable example for exploring the crash/restart assertions in `docs/P3_ENTRY_AND_ACCEPTANCE_TEST_PLAN.md` (P3-04/P3-07). It performs no filesystem, process, network, database, proxy, or host operations. It is not Tong Panel product code, a production contract, or evidence that any deployment/security property works.

The illustrative model allows a deployment to be called active only after the candidate artifact is verified, recorded, started, health-checked, and its route is observed. An uncertain route observation remains uncertain. A second tiny model checks that repeating the same `(tenant, app, key, request digest)` returns one simulated job ID and changed payloads conflict. It serializes concurrent duplicate calls within one process, but does not model multiple processes or durable transactions. That key scope is only a reviewable example, not an approved policy. The models are intentionally incomplete: they do not model tenant authorization, durable storage, jobs, quotas, audit persistence, or process supervision. Production design requires approved P1/P2 contracts and the reviewed decisions listed in the acceptance plan.

From this directory, with the supplied Windows Go toolchain:

```powershell
$env:GO111MODULE = 'off'
& 'C:\Users\~MaNBiR~\Documents\Codex\work\toolchains\go\bin\go.exe' test -count=1
```
