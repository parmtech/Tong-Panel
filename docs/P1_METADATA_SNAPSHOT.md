# Control-plane metadata snapshot primitive

`database.CreateSnapshot(ctx, db, parent)` creates a consistent standalone SQLite
image from a trusted control-plane database handle. It returns the absolute path
to `metadata.db` inside a newly owned temporary directory beneath an existing,
trusted parent. It does not choose an existing destination file or overwrite a
parent artifact. No daemon endpoint, CLI command or live restore is added.

The function uses parameterized `VACUUM INTO`, rather than copying a live main
file without its WAL. It validates the result through a separate read-only
connection, requiring the current schema-version marker, exactly one successful
integrity result and no foreign-key violations. Validation does not migrate the
image. Cancellation and failures return no artifact path and attempt to clean
only the newly owned directory, after closing its handles. Cleanup failure is
reported; callers must inspect their protected parent if cleanup is incomplete.

The directory/file request Unix modes 0700/0600. These Go modes do not establish
Windows ACL privacy; native Windows tests verify function and data consistency.
The caller must supply protected local storage, retain or dispose of successful
artifacts securely, and manage storage space. The parent is not created or
chmodded; hostile-directory/symlink confinement and network-storage certification
are outside this primitive.

Snapshots contain sensitive control-plane state, including credential verifiers,
session/recovery hashes, encrypted MFA material, jobs and audit history. They do
not contain externally held encryption/pepper keys. A coherent image is not a
safe production restore policy: older state can resurrect sessions or recovery
capabilities and repeat externally completed work. Production restore must fence
execution, address credential invalidation and key availability, and reconcile
external effects before traffic resumes. None of those decisions is implemented
by returning a snapshot filename.

The disposable rehearsal holds an independent uncommitted writer while taking a
snapshot, checks committed-state isolation, and restores only the completed main
image to a fresh path. Subsequent source changes must not appear in that copy.
Tests also check unique destinations, no clobbering, validation failures and
cancellation cleanup. See `UPDATES.md` for actually executed evidence.

[SQLite documents VACUUM INTO](https://www.sqlite.org/lang_vacuum.html) as a
consistent logical snapshot and permits an existing empty destination. An
interrupted operation may leave incomplete output. The implementation validates
before returning success; tests do not simulate disk failure or prove power-loss
durability. Integrity and foreign keys are distinct checks, as described in the
[SQLite pragma reference](https://www.sqlite.org/pragma.html).

This increment advances P1-07 metadata recovery foundations. It is not encrypted
offsite backup, authenticated archive verification, a schema-fingerprint audit,
a downgrade path, a production restore ceremony or Phase 1 completion.
