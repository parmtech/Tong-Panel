# Read-only structured support export

`tongctl support-bundle` writes one versioned JSON object to stdout. It accepts
no flags, attachment paths or output-file arguments. Callers can redirect stdout
to a file using their shell and select appropriate file permissions. The command
does not create files, bind ports, contact a network service or change the host.

This first diagnostic export is an explicit allowlist, not a text scrubber.
Every emitted string is a fixed enum or notice. It contains only:

- schema version and the fixed development-only notice;
- complete/partial collection status and a static collection code;
- normalised OS/runtime architecture and recognised Ubuntu 24.04 or Debian 12
  lab pairs, with other/unknown markers for unsupported or unavailable values;
- coarse adequate/low/unknown RAM and root-disk classifications;
- whether an inspected snapshot saw a TCP 80/443 listener, or collection was unknown.

The thresholds match existing preliminary developer doctor guidance: 1 GiB
available RAM and 8 GiB available root-filesystem space. A lab-candidate label is
not a supported-installation claim. A `not_seen` web-port result does not prove
that a service can bind, that external ports are reachable, or that installation
is safe. Resource values are not exported exactly.

The collector reads the same fixed OS/kernel/root-filesystem facts as doctor.
It reads no application database, credentials, environment variables, logs,
application configuration, tenant data, hostname or user-selected attachments. Raw OS-release
strings, listening addresses/arbitrary ports, filesystem paths and underlying
collector/writer error messages are never copied into the export. Invalid or
oversized OS-release data becomes unknown. Duplicate ID/version fields fail
closed. Known partial readings are retained when OS metadata is unavailable.

Exit 0 means the JSON export was written, including an explicitly partial export.
It does not mean host health, installation readiness, or P1 acceptance. Extra
arguments fail with exit 2 and generic usage; an output failure fails with exit 1
and a generic error, without echoing its underlying message. A failed stdout
write can leave a partial stream, so consumers must require exit 0 and valid JSON.

Tests inject credential canaries into OS metadata, runtime identifiers,
environment, collector errors and invalid arguments; check that none are exported;
and cover missing/unsupported data, coarse thresholds, deterministic bounded
output, writer errors and short writes. This command is a facts-only foundation.
Log redaction/attachments, API/UI diagnostic export, complete support workflows,
external security review and other P1/product acceptance gates remain open.
