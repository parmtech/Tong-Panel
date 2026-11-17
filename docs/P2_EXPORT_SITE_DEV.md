# Experimental site-file export command

`tongbackup export-site` is a development-only, unprivileged operator utility. It is not wired to the panel API, privileged agent, scheduler, production key management, database backup, or restore authorization. No owner/security or production deployment gate is changed.

The caller must stop all writers and prevent source-directory moves while the source is scanned. `--source-quiesced` is an acknowledgement of that responsibility; it does not verify that the source is quiescent. The caller must also ensure that the source root and all ancestor paths, the private output directory and its ancestors, and the external key path have stable, trusted ownership and cannot be changed by untrusted users during the command. The output directory and wrapping-key file must be outside the entire source root. The output directory must be owned by the current user and mode 0700 (no group or other permissions). The containment checks resolve symlinks and compare paths; they do not detect bind-mount or other filesystem aliases, and cannot prevent path replacement. The caller must ensure source, output and key are not aliased and that their ancestor paths remain stable. The exporter rejects its own ciphertext-stage names if they occur inside the scanned source, but that is not a general alias detector. These prerequisites are not a substitute for a filesystem snapshot, current tenant authorization, quota policy or production approval.

```sh
tongbackup export-site --tenant tenant-a \
  --key-file /run/private/backup.key \
  --root /srv/private-tenant-root --site sites/site-a/public \
  --output /srv/private-backups/site-a.tpbk --source-quiesced
```

The command creates a bounded tenant-bound encrypted archive using the existing pinned-root exporter. It rejects symlinks, hardlinks, mount crossings, special files and internal staging names; modes and timestamps are not preserved. Plaintext archive data is held in memory, not written to a staging file; Go may retain memory copies. It privately stages ciphertext as mode 0600 in the output directory and links it to a new destination without replacement only after the exporter succeeds. Successful completion is silent. Key material and plaintext are not printed.

An ordinary pre-existing output is refused. A link error is treated as **publication unknown**: a filesystem may have applied the link despite reporting an error, so the private ciphertext stage is retained and both names require supervised reconciliation. Failure to remove the stage or sync the directory after a successful link is reported as **published but durability/cleanup uncertain**. Do not blindly retry or automatically delete an ambiguous stage. Earlier errors may leave a private incomplete stage if cleanup fails. A successful export does not prove the application, SQL database, external keys or service can be restored; perform a supervised restore drill with the same tenant/key and preserve the existing release gates.
