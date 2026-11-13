#!/usr/bin/env bash
# Disposable guest check: encrypted tenant dump, authenticated restore, and
# cross-tenant/key denial. The clean VM harness copies only the sealed artifact
# and externally held test key to a second fresh guest for the restore drill.
set -Eeuo pipefail
work="$(mktemp -d /home/ci/tong-backup-source.XXXXXX)"
db_source="tp_backup_source"
db_restore="tp_backup_restore"
db_user="tu_backup_source"
password="$(od -An -N32 -tx1 /dev/urandom | tr -d ' \n')"
marker="private-tenant-restore-fixture-v1"
artifact="/home/ci/tenant-a.backup.tpbk"
key_copy="/home/ci/wrapping.key"
cleanup() {
  status=$?
  trap - EXIT
  sudo mariadb --protocol=socket -uroot >/dev/null 2>&1 <<SQL || true
DROP USER IF EXISTS '$db_user'@'localhost';
DROP DATABASE IF EXISTS \`$db_source\`;
DROP DATABASE IF EXISTS \`$db_restore\`;
SQL
  rm -rf -- "$work"
  exit "$status"
}
trap cleanup EXIT

sudo mariadb --protocol=socket -uroot <<SQL
CREATE DATABASE \`$db_source\`;
CREATE DATABASE \`$db_restore\`;
CREATE USER '$db_user'@'localhost' IDENTIFIED BY '$password';
GRANT ALL PRIVILEGES ON \`$db_source\`.* TO '$db_user'@'localhost';
GRANT ALL PRIVILEGES ON \`$db_restore\`.* TO '$db_user'@'localhost';
CREATE TABLE \`$db_source\`.restore_marker (value VARCHAR(80) NOT NULL);
INSERT INTO \`$db_source\`.restore_marker VALUES ('$marker');
CREATE TABLE \`$db_restore\`.restore_guard (value VARCHAR(80) NOT NULL);
INSERT INTO \`$db_restore\`.restore_guard VALUES ('untouched-before-restore');
SQL

umask 077
cat > "$work/tenant.cnf" <<EOF
[client]
user=$db_user
password=$password
host=127.0.0.1
protocol=tcp
EOF
head -c 32 /dev/urandom > "$work/wrapping.key"
head -c 32 /dev/urandom > "$work/wrong.key"
chmod 600 "$work/tenant.cnf" "$work/wrapping.key" "$work/wrong.key"

mariadb-dump --defaults-extra-file="$work/tenant.cnf" --single-transaction "$db_source" |
  /home/ci/tong-p1-dev/bin/tongbackup encrypt --tenant tenant-a --key-file "$work/wrapping.key" --output "$artifact"
[[ "$(stat -c '%a' "$artifact")" == 600 ]]
if grep -aFq "$marker" "$artifact"; then
  echo "encrypted backup contains the plaintext restore marker" >&2
  exit 1
fi

# Verify the actual encrypted dump before the existing restore. This is a
# separate read of the artifact, not an atomic verification/restore promise.
before_verify="$(sha256sum "$artifact" | cut -d ' ' -f 1)"
/home/ci/tong-p1-dev/bin/tongbackup verify --tenant tenant-a --key-file "$work/wrapping.key" < "$artifact" > "$work/verify.out" 2> "$work/verify.err"
[[ ! -s "$work/verify.out" && ! -s "$work/verify.err" ]]
[[ "$(sha256sum "$artifact" | cut -d ' ' -f 1)" == "$before_verify" ]]
python3 - "$artifact" "$work/tampered.tpbk" <<'PY_TAMPER'
from pathlib import Path
import sys
data = bytearray(Path(sys.argv[1]).read_bytes())
if not data:
    raise SystemExit("empty synthetic backup")
data[-1] ^= 1
Path(sys.argv[2]).write_bytes(data)
PY_TAMPER
if /home/ci/tong-p1-dev/bin/tongbackup verify --tenant tenant-a --key-file "$work/wrapping.key" < "$work/tampered.tpbk" > "$work/verify-bad.out" 2> "$work/verify-bad.err"; then
  echo "tampered backup unexpectedly verified" >&2
  exit 1
fi
[[ ! -s "$work/verify-bad.out" ]]
[[ "$(sha256sum "$artifact" | cut -d ' ' -f 1)" == "$before_verify" ]]
guard="$(mariadb --defaults-extra-file="$work/tenant.cnf" --batch --skip-column-names "$db_restore" -e 'SELECT value FROM restore_guard')"
[[ "$guard" == 'untouched-before-restore' ]]

if /home/ci/tong-p1-dev/bin/tongbackup decrypt --tenant tenant-b --key-file "$work/wrapping.key" --output "$work/wrong-tenant.sql" < "$artifact" >/dev/null 2>&1; then
  echo "backup decrypted under the wrong tenant" >&2
  exit 1
fi
[[ ! -e "$work/wrong-tenant.sql" ]]
if /home/ci/tong-p1-dev/bin/tongbackup decrypt --tenant tenant-a --key-file "$work/wrong.key" --output "$work/wrong-key.sql" < "$artifact" >/dev/null 2>&1; then
  echo "backup decrypted under the wrong key" >&2
  exit 1
fi
[[ ! -e "$work/wrong-key.sql" ]]

/home/ci/tong-p1-dev/bin/tongbackup decrypt --tenant tenant-a --key-file "$work/wrapping.key" --output "$work/restore.sql" < "$artifact"
mariadb --defaults-extra-file="$work/tenant.cnf" "$db_restore" < "$work/restore.sql"
restored="$(mariadb --defaults-extra-file="$work/tenant.cnf" --batch --skip-column-names "$db_restore" -e 'SELECT value FROM restore_marker')"
[[ "$restored" == "$marker" ]]
guard="$(mariadb --defaults-extra-file="$work/tenant.cnf" --batch --skip-column-names "$db_restore" -e 'SELECT value FROM restore_guard')"
[[ "$guard" == 'untouched-before-restore' ]]
install -m 0600 "$work/wrapping.key" "$key_copy"
printf '%s\n' 'PASS: encrypted tenant dump hides plaintext; wrong tenant/key are denied; authenticated restore recovers the source marker'
