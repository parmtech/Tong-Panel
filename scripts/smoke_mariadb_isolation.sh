#!/usr/bin/env bash
# Disposable guest-only check that MariaDB grants isolate two tenant databases.
set -Eeuo pipefail
work="$(mktemp -d)"
db_a="tp_aaaaaaaaaaaaaaaaaaaaaaaaaa"
db_b="tp_bbbbbbbbbbbbbbbbbbbbbbbbbb"
user_a="tu_aaaaaaaaaaaaaaaaaaaaaaaaaa"
user_b="tu_bbbbbbbbbbbbbbbbbbbbbbbbbb"
password_a="$(od -An -N32 -tx1 /dev/urandom | tr -d ' \n')"
password_b="$(od -An -N32 -tx1 /dev/urandom | tr -d ' \n')"
cleanup() {
  status=$?
  trap - EXIT
  sudo mariadb --protocol=socket -uroot >/dev/null 2>&1 <<SQL || true
DROP USER IF EXISTS '$user_a'@'localhost';
DROP USER IF EXISTS '$user_b'@'localhost';
DROP DATABASE IF EXISTS \`$db_a\`;
DROP DATABASE IF EXISTS \`$db_b\`;
SQL
  rm -rf "$work"
  exit "$status"
}
trap cleanup EXIT

sudo mariadb --protocol=socket -uroot <<SQL
CREATE DATABASE \`$db_a\`;
CREATE DATABASE \`$db_b\`;
CREATE USER '$user_a'@'localhost' IDENTIFIED BY '$password_a';
CREATE USER '$user_b'@'localhost' IDENTIFIED BY '$password_b';
GRANT ALL PRIVILEGES ON \`$db_a\`.* TO '$user_a'@'localhost';
GRANT ALL PRIVILEGES ON \`$db_b\`.* TO '$user_b'@'localhost';
CREATE TABLE \`$db_a\`.tenant_marker (owner_name VARCHAR(32) NOT NULL);
CREATE TABLE \`$db_b\`.tenant_marker (owner_name VARCHAR(32) NOT NULL);
INSERT INTO \`$db_a\`.tenant_marker VALUES ('tenant-a');
INSERT INTO \`$db_b\`.tenant_marker VALUES ('tenant-b');
SQL

write_client_config() {
  local path="$1" username="$2" password="$3"
  umask 077
  cat > "$path" <<EOF
[client]
user=$username
password=$password
host=127.0.0.1
protocol=tcp
EOF
  chmod 600 "$path"
}
write_client_config "$work/tenant-a.cnf" "$user_a" "$password_a"
write_client_config "$work/tenant-b.cnf" "$user_b" "$password_b"

own_a="$(mariadb --defaults-extra-file="$work/tenant-a.cnf" --batch --skip-column-names "$db_a" -e 'SELECT owner_name FROM tenant_marker')"
own_b="$(mariadb --defaults-extra-file="$work/tenant-b.cnf" --batch --skip-column-names "$db_b" -e 'SELECT owner_name FROM tenant_marker')"
[[ "$own_a" == tenant-a && "$own_b" == tenant-b ]]

if mariadb --defaults-extra-file="$work/tenant-a.cnf" --batch --skip-column-names "$db_b" -e 'SELECT owner_name FROM tenant_marker' >/dev/null 2>&1; then
  echo "tenant A unexpectedly read tenant B database" >&2
  exit 1
fi
if mariadb --defaults-extra-file="$work/tenant-b.cnf" --batch --skip-column-names "$db_a" -e 'SELECT owner_name FROM tenant_marker' >/dev/null 2>&1; then
  echo "tenant B unexpectedly read tenant A database" >&2
  exit 1
fi

printf '%s\n' "PASS: each temporary MariaDB user reads its own tenant database and is denied the other tenant's database"
