#!/usr/bin/env bash
# Disposable fresh guest ONLY. Plaintext WordPress data never leaves this guest.
set -Eeuo pipefail
[[ "${TONG_DISPOSABLE_WORDPRESS_RESTORE:-}" == 1 ]] || {
  echo 'WordPress restore requires the explicit disposable-guest lab gate' >&2
  exit 2
}
cd /home/ci
umask 077
backup=/home/ci/tenant-a-wordpress.backup.tpbk
key=/home/ci/wrapping.key
bundle=/home/ci/wordpress-restore.tar.gz
public=/srv/tong/tenants/tenant-a/sites/site-a/public
identifiers=/home/ci/wordpress-restore-identifiers.json
backup_tool=/home/ci/tongbackup
sites_test=/home/ci/sites.test
[[ -x "$backup_tool" ]] || backup_tool=/home/ci/tong-p1-dev/bin/tongbackup
[[ -x "$sites_test" ]] || sites_test=/home/ci/tong-p1-dev/tests/sites.test
test -x "$backup_tool"
test -x "$sites_test"
test -f "$backup"
test -f "$key"
test ! -e "$bundle"
test ! -e "$identifiers"
test ! -e "$public/wordpress"
cleanup() {
  status=$?
  trap - EXIT
  set +e
  sudo nginx -s quit -p /etc/nginx/ -c /home/ci/wp-restore-nginx.conf >/dev/null 2>&1
  if [[ -s /run/tong/wp-restore-fpm.pid ]]; then
    sudo kill -TERM "$(sudo cat /run/tong/wp-restore-fpm.pid)" >/dev/null 2>&1
  fi
  cleanup_status=0
  python3 - "$identifiers" <<'PY'
import json, pathlib, re, stat, subprocess, sys
path = pathlib.Path(sys.argv[1])
if path.exists():
    info = path.lstat()
    if not stat.S_ISREG(info.st_mode) or stat.S_IMODE(info.st_mode) != 0o600 or info.st_size > 4096:
        raise SystemExit('unsafe restore cleanup identity file')
    data = json.loads(path.read_text())
    if set(data) != {'database', 'user'} or not re.fullmatch(r'tp_wp_a_[0-9a-f]{16}', data['database']) or not re.fullmatch(r'tu_wp_a_[0-9a-f]{16}', data['user']):
        raise SystemExit('restore cleanup identity is outside disposable namespace')
    statement = "DROP USER IF EXISTS '%s'@'localhost';\nDROP DATABASE IF EXISTS `%s`;\n" % (data['user'], data['database'])
    subprocess.run(['sudo', 'mariadb', '--protocol=socket', '-uroot'], input=statement.encode(), stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL, check=True, timeout=30)
PY
  cleanup_status=$?
  rm -f -- "$key" "$bundle" "$identifiers" /home/ci/wp-restore-tenant.cnf /home/ci/wp-wrong-key /home/ci/wp-wrong-key.tar.gz /home/ci/wp-wrong-tenant.tar.gz
  sudo rm -f -- /home/ci/wp-restore-nginx.conf /home/ci/wp-restore-fpm.conf
  if [[ "$status" == 0 && "$cleanup_status" != 0 ]]; then
    echo 'restored WordPress SQL fixture cleanup failed' >&2
    status=1
  fi
  exit "$status"
}
trap cleanup EXIT
chmod 0600 "$key" "$backup"
[[ "$(stat -c '%s' "$backup")" -le $((34 * 1024 * 1024)) ]]

# Authentication must finish before any site tree or database is provisioned.
if "$backup_tool" decrypt --tenant tenant-b --key-file "$key" --output /home/ci/wp-wrong-tenant.tar.gz < "$backup" >/dev/null 2>&1; then
  echo 'WordPress backup accepted the wrong tenant' >&2
  exit 1
fi
test ! -e /home/ci/wp-wrong-tenant.tar.gz
head -c 32 /dev/urandom > /home/ci/wp-wrong-key
chmod 0600 /home/ci/wp-wrong-key
if "$backup_tool" decrypt --tenant tenant-a --key-file /home/ci/wp-wrong-key --output /home/ci/wp-wrong-key.tar.gz < "$backup" >/dev/null 2>&1; then
  echo 'WordPress backup accepted the wrong wrapping key' >&2
  exit 1
fi
test ! -e /home/ci/wp-wrong-key.tar.gz
test ! -e "$public/wordpress"
"$backup_tool" decrypt --tenant tenant-a --key-file "$key" --output "$bundle" < "$backup"
[[ "$(stat -c '%a' "$bundle")" == 600 ]]

sudo apt-get update
sudo apt-get install -y --no-install-recommends nginx php-fpm php-cli php-mysql mariadb-server curl
sudo systemctl start mariadb
sudo install -d -o ci -g ci -m 0755 "$public"
TONG_WORDPRESS_RESTORE_ROOT="$public" TONG_WORDPRESS_RESTORE_BUNDLE="$bundle" \
  "$sites_test" -test.run '^TestRestoreWordPressBackupForCleanGuest$' -test.v -test.timeout=5m
test ! -e "$bundle"
test -f "$public/wordpress/wp-config.php"

# Reuse the reviewed renderer; the second tenant is a distinct static fallback.
TONG_TWO_TENANT_SITE_PREVIEW_DIR=/home/ci/wp-restore-preview \
  "$sites_test" -test.run '^TestWriteTwoTenantSitePreviewsForCleanVM$' -test.timeout=1m
mapfile -t pool_users < <(awk -F ' = ' '/^user = / { print $2 }' /home/ci/wp-restore-preview/php-fpm-pools.conf)
[[ "${#pool_users[@]}" == 2 && "${pool_users[0]}" != "${pool_users[1]}" ]]
for index in 0 1; do
  pool_user="${pool_users[$index]}"
  [[ "$pool_user" =~ ^[a-z_][a-z0-9_]*$ ]]
  if ! id -u "$pool_user" >/dev/null 2>&1; then
    sudo useradd --system --user-group --no-create-home --shell /usr/sbin/nologin "$pool_user"
  fi
  if [[ "$index" == 0 ]]; then tenant=a; else tenant=b; fi
  sudo install -d -m 0755 "/run/tong/php/tenant-$tenant"
  sudo install -d -o "$pool_user" -g "$pool_user" -m 0700 "/srv/tong/tenants/tenant-$tenant/run/site-$tenant/tmp" "/srv/tong/tenants/tenant-$tenant/run/site-$tenant/sessions"
done
sudo chown -R "${pool_users[0]}:${pool_users[0]}" "$public/wordpress"
sudo find "$public/wordpress" -type d -exec chmod 0755 {} +
sudo find "$public/wordpress" -type f ! -name wp-config.php -exec chmod 0644 {} +
sudo chmod 0600 "$public/wordpress/wp-config.php"
sudo -u www-data test -r "$public/wordpress/readme.html"
sudo -u www-data test ! -r "$public/wordpress/wp-config.php"
sudo -u "${pool_users[1]}" test ! -r "$public/wordpress/wp-config.php"
sudo install -d -o "${pool_users[1]}" -g "${pool_users[1]}" -m 0755 /srv/tong/tenants/tenant-b/sites/site-b/public
printf '%s\n' 'fresh-restore-tenant-b-static' | sudo tee /srv/tong/tenants/tenant-b/sites/site-b/public/index.html >/dev/null
sudo chmod 0644 /srv/tong/tenants/tenant-b/sites/site-b/public/index.html

# Lab-only controls are deliberately reintroduced, not copied as app backup data.
sudo install -d -o "${pool_users[0]}" -g "${pool_users[0]}" -m 0700 "$public/wordpress/wp-content/mu-plugins"
sudo tee "$public/wordpress/wp-content/mu-plugins/tong-lab-isolation.php" >/dev/null <<'PHP'
<?php
if (!defined('DISABLE_WP_CRON')) define('DISABLE_WP_CRON', true);
if (!defined('WP_HTTP_BLOCK_EXTERNAL')) define('WP_HTTP_BLOCK_EXTERNAL', true);
add_filter('pre_wp_mail', static function () { return true; });
PHP
sudo chown "${pool_users[0]}:${pool_users[0]}" "$public/wordpress/wp-content/mu-plugins/tong-lab-isolation.php"
sudo chmod 0644 "$public/wordpress/wp-content/mu-plugins/tong-lab-isolation.php"
cat > /home/ci/wp-restore-fpm.conf <<'FPM'
[global]
daemonize = yes
pid = /run/tong/wp-restore-fpm.pid
error_log = /tmp/tong-wp-restore-fpm.log
include = /home/ci/wp-restore-preview/php-fpm-pools.conf
FPM
fpm_bin="$(command -v php-fpm || true)"
if [[ -z "$fpm_bin" ]]; then
  for candidate in /usr/sbin/php-fpm[0-9]*; do
    if [[ -x "$candidate" ]]; then fpm_bin="$candidate"; break; fi
  done
fi
test -n "$fpm_bin"
sudo "$fpm_bin" -t -y /home/ci/wp-restore-fpm.conf
sudo "$fpm_bin" -D -y /home/ci/wp-restore-fpm.conf
for _ in {1..30}; do
  [[ -S /run/tong/php/tenant-a/site-a.sock && -S /run/tong/php/tenant-b/site-b.sock ]] && break
  sleep 1
done
test -S /run/tong/php/tenant-a/site-a.sock
test -S /run/tong/php/tenant-b/site-b.sock
sudo systemctl stop nginx
cat > /home/ci/wp-restore-nginx.conf <<'NGINX'
user www-data;
worker_processes 1;
pid /run/tong/wp-restore-nginx.pid;
error_log /tmp/tong-wp-restore-nginx.log;
events { worker_connections 64; }
http {
    include /etc/nginx/mime.types;
    default_type application/octet-stream;
    include /home/ci/wp-restore-preview/nginx-sites.conf;
}
NGINX
sudo nginx -t -p /etc/nginx/ -c /home/ci/wp-restore-nginx.conf
sudo nginx -p /etc/nginx/ -c /home/ci/wp-restore-nginx.conf
homepage="$(curl --silent --show-error --fail --max-time 30 --resolve tenant-a.example.test:80:127.0.0.1 http://tenant-a.example.test/wordpress/)"
[[ "$homepage" == *'Tong Tenant Alpha Lab'* && "$homepage" != *'Tong Tenant Beta Lab'* ]]
upload="$(curl --silent --show-error --fail --max-time 30 --resolve tenant-a.example.test:80:127.0.0.1 http://tenant-a.example.test/wordpress/wp-content/uploads/tong-restore-marker.txt)"
[[ "$upload" == private-wp-restore-upload-v1 ]]
fallback="$(curl --silent --show-error --fail --max-time 30 --resolve tenant-b.example.test:80:127.0.0.1 http://tenant-b.example.test/)"
[[ "$fallback" == fresh-restore-tenant-b-static ]]
if mariadb --defaults-extra-file=/home/ci/wp-restore-tenant.cnf mysql --batch --skip-column-names -e 'SELECT User FROM user LIMIT 1' >/dev/null 2>&1; then
  echo 'restored tenant database user unexpectedly read privileged database data' >&2
  exit 1
fi
printf '%s\n' 'PASS: encrypted WordPress SQL/config/content restored on a fresh guest; real homepage/upload verified, private config and privileged DB access denied'
