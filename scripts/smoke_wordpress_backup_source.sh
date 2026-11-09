#!/usr/bin/env bash
# Disposable source-guest lab: bundle one installed WordPress site's database,
# private configuration, and user content, then encrypt the bounded bundle.
set -Eeuo pipefail
[[ "${TONG_DISPOSABLE_WORDPRESS_RESTORE:-}" == 1 ]] || { echo 'source bundle helper is restricted to the disposable restore VM' >&2; exit 2; }
db_name="${1:?database name required}"
table_prefix="${2:?table prefix required}"
admin_user="${3:?administrator login required}"
db_client_config="/home/ci/wp-tenant-a.cnf"
site="/srv/tong/tenants/tenant-a/sites/site-a/public/wordpress"
artifact="/home/ci/tenant-a-wordpress.backup.tpbk"
work="$(mktemp -d /home/ci/tong-wordpress-backup.XXXXXX)"
payload="$work/payload"
cleanup() {
  status=$?
  trap - EXIT
  rm -rf -- "$work"
  exit "$status"
}
trap cleanup EXIT
umask 077
[[ "$db_name" =~ ^[a-z0-9_]{1,64}$ ]]
[[ "$table_prefix" =~ ^[A-Za-z_][A-Za-z0-9_]{0,15}$ ]]
[[ "$admin_user" =~ ^[a-z0-9_]{1,64}$ ]]
mkdir -m 0700 "$payload"

# This disposable fixture has no live users or background writers. Stop web
# and FPM service before creating the coordinated database/filesystem snapshot.
for nginx_config in /home/ci/nginx-e2e.conf /home/ci/nginx-https-e2e.conf; do
  pidfile="$(sed -n 's/^pid //p' "$nginx_config" | tr -d ';')"
  if [[ -n "$pidfile" && -s "$pidfile" ]]; then
    nginx_pid="$(sudo cat "$pidfile")"
    sudo nginx -s quit -p /etc/nginx/ -c "$nginx_config"
    for _ in {1..30}; do
      sudo kill -0 "$nginx_pid" 2>/dev/null || break
      sleep 1
    done
    if sudo kill -0 "$nginx_pid" 2>/dev/null; then
      echo 'Nginx did not quiesce before the WordPress backup snapshot' >&2
      exit 1
    fi
  fi
done
if [[ -s /run/tong/php-fpm-e2e.pid ]]; then
  fpm_pid="$(sudo cat /run/tong/php-fpm-e2e.pid)"
  sudo kill -QUIT "$fpm_pid"
  for _ in {1..30}; do
    sudo kill -0 "$fpm_pid" 2>/dev/null || break
    sleep 1
  done
  if sudo kill -0 "$fpm_pid" 2>/dev/null; then
    echo 'PHP-FPM did not quiesce before the WordPress backup snapshot' >&2
    exit 1
  fi
fi

# This database marker and upload marker prove the application payload survives
# independently of WordPress's default installation rows.
mariadb --defaults-extra-file="$db_client_config" "$db_name" -e \
  "INSERT INTO ${table_prefix}options (option_name, option_value, autoload) VALUES ('tong_restore_marker', 'private-wp-restore-db-v1', 'no') ON DUPLICATE KEY UPDATE option_value=VALUES(option_value)"
mariadb-dump --defaults-extra-file="$db_client_config" --single-transaction "$db_name" > "$payload/database.sql"
install -m 0600 "$site/wp-config.php" "$payload/wp-config.php"
cp -a "$site/wp-content" "$payload/wp-content"
rm -f "$payload/wp-content/mu-plugins/tong-lab-isolation.php"
install -d -m 0700 "$payload/wp-content/uploads"
printf '%s\n' 'private-wp-restore-upload-v1' > "$payload/wp-content/uploads/tong-restore-marker.txt"

python3 - "$payload" "$db_name" "$table_prefix" "$admin_user" <<'PY'
import configparser
import hashlib
import json
import os
import pathlib
import re
import stat
import sys

root = pathlib.Path(sys.argv[1])
database, prefix, admin_user = sys.argv[2:]
credentials = configparser.ConfigParser()
if not credentials.read('/home/ci/wp-tenant-a.cnf') or 'client' not in credentials:
    raise SystemExit('private tenant client file is unavailable')
user = credentials['client'].get('user', '')
password = credentials['client'].get('password', '')
if not re.fullmatch(r'[a-z0-9_]{1,64}', database) or not re.fullmatch(r'[a-z0-9_]{1,64}', user):
    raise SystemExit('invalid disposable database identity')
if not re.fullmatch(r'[0-9a-f]{64}', password) or not re.fullmatch(r'[A-Za-z_][A-Za-z0-9_]{0,15}', prefix) or not re.fullmatch(r'[a-z0-9_]{1,64}', admin_user):
    raise SystemExit('invalid disposable database secret or table prefix')
allowed = {'database.sql', 'wp-config.php', 'wp-content'}
seen = set()
total = 0
count = 0
for path in root.rglob('*'):
    relative = path.relative_to(root).as_posix()
    info = path.lstat()
    if relative.split('/')[0] not in allowed or not (stat.S_ISREG(info.st_mode) or stat.S_ISDIR(info.st_mode)):
        raise SystemExit('unsafe WordPress backup member')
    if stat.S_ISREG(info.st_mode):
        count += 1
        total += info.st_size
        if info.st_size > 16 * 1024 * 1024 or count > 2048 or total > 64 * 1024 * 1024:
            raise SystemExit('WordPress backup exceeds file or expanded-size cap')
        seen.add(relative)
if not {'database.sql', 'wp-config.php', 'wp-content/uploads/tong-restore-marker.txt'} <= seen:
    raise SystemExit('required WordPress backup members are missing')
manifest = {
    'format_version': 1,
    'tenant_id': 'tenant-a',
    'core_version': '7.1.4',
    'database_name': database,
    'database_user': user,
    'database_password': password,
    'table_prefix': prefix,
    'blog_title': 'Tong Tenant Alpha Lab',
    'admin_user': admin_user,
    'database_sha256': hashlib.sha256((root / 'database.sql').read_bytes()).hexdigest(),
    'config_sha256': hashlib.sha256((root / 'wp-config.php').read_bytes()).hexdigest(),
    'upload_marker_sha256': hashlib.sha256((root / 'wp-content/uploads/tong-restore-marker.txt').read_bytes()).hexdigest(),
}
(root / 'manifest.json').write_text(json.dumps(manifest, separators=(',', ':')) + '\n', encoding='utf-8')
os.chmod(root / 'manifest.json', 0o600)
PY

python3 - "$payload" <<'PY'
import pathlib, sys
root = pathlib.Path(sys.argv[1])
total = sum(p.stat().st_size for p in root.rglob('*') if p.is_file())
if total > 64 * 1024 * 1024:
    raise SystemExit('WordPress bundle exceeds the file archive expanded-size cap')
PY
tar -C "$payload" -czf "$work/tenant-a-wordpress.tar.gz" manifest.json database.sql wp-config.php wp-content
[[ "$(stat -c '%s' "$work/tenant-a-wordpress.tar.gz")" -le $((30 * 1024 * 1024)) ]]
/home/ci/tong-p1-dev/bin/tongbackup encrypt --tenant tenant-a --key-file /home/ci/wrapping.key \
  --output "$artifact" < "$work/tenant-a-wordpress.tar.gz"
[[ "$(stat -c '%a' "$artifact")" == 600 ]]
chown ci:ci "$artifact"
if grep -aFq 'private-wp-restore-db-v1' "$artifact" ||
   grep -aFq 'private-wp-restore-upload-v1' "$artifact" ||
   grep -aFq 'Tong Tenant Alpha Lab' "$artifact"; then
  echo 'encrypted WordPress backup contains a plaintext fixture marker' >&2
  exit 1
fi
db_password="$(python3 -c 'import configparser; c=configparser.ConfigParser(); c.read("/home/ci/wp-tenant-a.cnf"); print(c["client"]["password"])')"
if grep -aFq "$db_password" "$artifact"; then
  echo 'encrypted WordPress backup contains a plaintext database credential' >&2
  exit 1
fi
printf '%s\n' 'PASS: bounded WordPress SQL/config/wp-content bundle encrypted after source web/FPM quiescence; plaintext stays on source guest'
