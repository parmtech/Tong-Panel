# Metadata failure and retry tests

These disposable tests exercise failure paths without changing migration scripts,
adding production test hooks or touching an installed panel.

`migration_failure_test.go` constructs schema 11 from the embedded migrations,
then seeds representative identity, membership, session, job and audit metadata.
A deliberately conflicting fixture occupies the second table name created by
migration 12. Opening through the real migration runner fails after the first
DDL statement. A raw connection verifies that the version remains 11, the first
new table was rolled back, and the conflict fixture and baseline metadata remain
unchanged. Removing only the deliberate conflict allows a fresh `Open` to retry
successfully, with current schema, integrity and foreign keys verified.

This verifies SQL-error transaction rollback and repair/retry for migration 12.
It does not prove interruption by SIGKILL, every historical migration, downgrade,
disk failure or power loss. Removing the known synthetic conflict in a test is
not a general production schema-repair procedure.

`snapshot_validation_test.go` checks that non-SQLite and truncated image fixtures
are rejected by the read-only snapshot validator without changing their bytes.
Rejection does not repair corrupted data or authenticate the provenance of a
backup. It complements the snapshot creation tests for foreign-key/version
validation, cancellation cleanup and successful fresh-copy restoration.

See `UPDATES.md` for exact authored versus executed evidence. Production recovery
still needs protected backup/key custody, credential and queue rollback policy,
external-effect reconciliation and supported OS/storage failure drills.
