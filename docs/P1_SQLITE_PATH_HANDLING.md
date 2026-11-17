# SQLite metadata filenames

`database.Open` accepts a filesystem path, not a SQLite connection URI. It resolves
that path to an absolute filename and builds an escaped `file:` URL with its own
pragma query. Literal query/fragment/percent characters in a filename must remain
filename data. Windows drive paths use the file-URI leading slash convention.

The old string concatenation could select a different file because SQLite ignores
URI fragments and decodes percent escapes, while the Go driver splits the first
question mark to parse connection options. See the primary references:
[SQLite URI filenames](https://www.sqlite.org/uri.html) and
[Go URL serialization](https://pkg.go.dev/net/url#URL.String).

Regression tests use the real driver, inspect the opened main database path, check
that the intended file exists, reopen persisted data and assert the required
foreign-key, WAL, FULL synchronous and busy-timeout settings. POSIX-only filename
characters are tested on Linux; Windows-compatible fragment/percent names also
run natively on Windows.

This change does not implement symlink confinement, permissions policy, network
filesystem support, metadata backup/restore or production database approval.
Callers must still select trusted local storage and protect the directory. It
does not migrate or delete any accidentally created filename from older code;
operators must inspect such files before choosing the intended metadata store.

See `UPDATES.md` for failing-before/passing-after evidence and hosted results.
