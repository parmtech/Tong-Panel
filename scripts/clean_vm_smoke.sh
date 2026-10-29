#!/usr/bin/env bash
set -Eeuo pipefail
: "${VM_DISTRO:?}"
: "${VM_IMAGE_URL:?}"
: "${VM_CHECKSUM_ALGORITHM:?}"
: "${VM_IMAGE_CHECKSUM:?}"
: "${VM_EXPECTED_ID:?}"
: "${VM_EXPECTED_VERSION:?}"
: "${VM_RUNNER_TEMP:?}"
work="$(mktemp -d "$VM_RUNNER_TEMP/tong-clean-vm.XXXXXX")"
port=2222
pidfile="$work/qemu.pid"
serial="$work/serial.log"
cleanup() {
  status=$?
  trap - EXIT
  if [[ -f "$pidfile" ]]; then
    pid="$(cat "$pidfile")"
    kill "$pid" 2>/dev/null || true
    for _ in {1..20}; do kill -0 "$pid" 2>/dev/null || break; sleep 0.25; done
  fi
  if [[ $status -ne 0 && -f "$serial" ]]; then
    echo "QEMU serial log ($VM_DISTRO):"
    tail -n 120 "$serial" || true
  fi
  if [[ $status -ne 0 && -f "$work/restore-serial.log" ]]; then
    echo "QEMU restore guest serial log ($VM_DISTRO):"
    tail -n 120 "$work/restore-serial.log" || true
  fi
  rm -rf "$work"
  exit "$status"
}
trap cleanup EXIT
if ! update_output="$(sudo apt-get update 2>&1)"; then
  printf '%s\n' "$update_output" >&2
  if ! grep -Fq 'Failed to fetch https://packages.microsoft.com/' <<<"$update_output" ||
     grep -E '^(Err:|E: Failed to fetch)' <<<"$update_output" | grep -v 'packages.microsoft.com' >/dev/null; then
    echo "apt update failed outside the optional Microsoft package feed" >&2
    exit 1
  fi
  mapfile -t microsoft_sources < <(sudo grep -RIlF -- 'packages.microsoft.com' /etc/apt/sources.list /etc/apt/sources.list.d 2>/dev/null || true)
  if [[ "${#microsoft_sources[@]}" -eq 0 ]]; then
    echo "Microsoft package feed failed but its source file could not be identified" >&2
    exit 1
  fi
  for source in "${microsoft_sources[@]}"; do
    sudo mv "$source" "$source.disabled-for-clean-vm"
  done
  sudo apt-get update
else
  printf '%s\n' "$update_output"
fi
sudo apt-get install -y --no-install-recommends qemu-system-x86 qemu-utils cloud-image-utils openssh-client curl ca-certificates
base="$work/base-image"
disk="$work/guest.qcow2"
seed="$work/seed.iso"
key="$work/ci_ed25519"
curl --fail --location --retry 3 --output "$base" "$VM_IMAGE_URL"
bash scripts/verify_vm_image_checksum.sh "$VM_CHECKSUM_ALGORITHM" "$VM_IMAGE_CHECKSUM" "$base"
ssh-keygen -q -t ed25519 -N "" -f "$key"
pub="$(cat "$key.pub")"
cat > "$work/user-data" <<EOF
#cloud-config
users:
  - name: ci
    gecos: Tong disposable VM check
    sudo: ALL=(ALL) NOPASSWD:ALL
    lock_passwd: true
    shell: /bin/bash
    ssh_authorized_keys:
      - $pub
ssh_pwauth: false
disable_root: true
EOF
cat > "$work/meta-data" <<EOF
instance-id: tong-$VM_DISTRO-$(date +%s)
local-hostname: tong-$VM_DISTRO
EOF
cloud-localds "$seed" "$work/user-data" "$work/meta-data"
qemu-img create -q -f qcow2 -F qcow2 -b "$base" "$disk" 8G
qemu-system-x86_64 -machine accel=tcg -cpu max -m 2048 -smp 2 -daemonize -pidfile "$pidfile" -serial "file:$serial" -monitor none -display none -drive "file=$disk,if=virtio,format=qcow2" -drive "file=$seed,if=virtio,format=raw,readonly=on" -netdev "user,id=net0,hostfwd=tcp:127.0.0.1:$port-:22" -device virtio-net-pci,netdev=net0
ssh_opts=(-i "$key" -p "$port" -o IdentitiesOnly=yes -o BatchMode=yes -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -o ConnectTimeout=4)
scp_opts=(-i "$key" -P "$port" -o IdentitiesOnly=yes -o BatchMode=yes -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null)
guest="ci@127.0.0.1"
ready=false
for _ in {1..60}; do
  if ssh "${ssh_opts[@]}" "$guest" true >/dev/null 2>&1; then ready=true; break; fi
  if ! kill -0 "$(cat "$pidfile")" 2>/dev/null; then echo "QEMU exited before SSH"; exit 1; fi
  sleep 5
done
[[ "$ready" == true ]] || { echo "Timed out waiting for clean VM SSH"; exit 1; }
retry_guest_ssh() {
  for attempt in {1..12}; do
    if ssh "${ssh_opts[@]}" "$guest" "$@"; then return 0; fi
    echo "SSH attempt $attempt interrupted while cloud-init may restart sshd; retrying" >&2
    sleep 5
  done
  echo "SSH command did not recover after cloud-init startup" >&2
  return 1
}
retry_guest_scp() {
  for attempt in {1..12}; do
    if scp "${scp_opts[@]}" "$@"; then return 0; fi
    echo "SCP attempt $attempt interrupted during guest startup; retrying" >&2
    sleep 5
  done
  echo "SCP transfer did not recover after guest startup" >&2
  return 1
}
# cloud-init can restart sshd after its initial key becomes reachable. Retry the
# wait command and file transfers across that expected one-time disconnect.
retry_guest_ssh "cloud-init status --wait"
retry_guest_scp "$VM_RUNNER_TEMP/tong-p1-UNSIGNED-dev-linux-amd64.tar.gz" "$VM_RUNNER_TEMP/SHA256SUMS" "$guest:/home/ci/"
retry_guest_scp "$VM_RUNNER_TEMP/tong-package-lab/tong-panel-dev-v1.deb" "$VM_RUNNER_TEMP/tong-package-lab/tong-panel-dev-v2.deb" "$VM_RUNNER_TEMP/tong-package-lab/PACKAGES.SHA256SUMS" "$guest:/home/ci/"
retry_guest_scp scripts/smoke_p1.py "$guest:/home/ci/smoke_p1.py"
retry_guest_scp scripts/p2_package_install_vm.sh "$guest:/home/ci/p2_package_install_vm.sh"
cat > "$work/guest-check.sh" <<'GUEST'
#!/usr/bin/env bash
set -Eeuo pipefail
trap 'status=$?; echo "clean guest smoke failed (status=$status) at: $BASH_COMMAND" >&2' ERR
test "$(id -u)" -ne 0
test "$(uname -m)" = x86_64
. /etc/os-release
test "$ID" = "$1"
test "$VERSION_ID" = "$2"
printf 'Clean guest: %s (%s), uid=%s\n' "$PRETTY_NAME" "$VERSION_ID" "$(id -u)"
cd /home/ci
sha256sum -c SHA256SUMS
tar -xzf tong-p1-UNSIGNED-dev-linux-amd64.tar.gz
test -x tong-p1-dev/bin/tongd
test -x tong-p1-dev/bin/tongctl
test -x tong-p1-dev/bin/tongbackup
test -x tong-p1-dev/tests/files.test
test -x tong-p1-dev/tests/api.test
test -x tong-p1-dev/tests/sites.test
grep -q 'UNSIGNED DEVELOPER PREVIEW ONLY' tong-p1-dev/DO_NOT_INSTALL.txt
./tong-p1-dev/tests/files.test -test.v -test.timeout=4m
./tong-p1-dev/tests/api.test -test.v -test.timeout=4m
./tong-p1-dev/tests/sites.test -test.v -test.timeout=4m
TONG_TEST_PINNED_WORDPRESS_RELEASE=1 ./tong-p1-dev/tests/sites.test -test.run '^TestPinnedWordPressCoreReleaseOnCleanGuest$' -test.timeout=5m
sudo apt-get update
sudo apt-get install -y --no-install-recommends nginx php-fpm mariadb-server curl
sudo systemctl start mariadb
bash ./tong-p1-dev/tests/nginx-config-rollback.sh
TONG_NGINX_PREVIEW_PATH=/home/ci/nginx-site.conf ./tong-p1-dev/tests/sites.test -test.run '^TestRenderNginxVhostRequiresVerifiedCanonicalDomains$' -test.timeout=1m
TONG_PHP_FPM_PREVIEW_PATH=/home/ci/php-fpm-pool.conf ./tong-p1-dev/tests/sites.test -test.run '^TestRenderPHPFPMPoolRequiresVerifiedIsolatedSite$' -test.timeout=1m
cat > /home/ci/php-fpm-test.conf <<'FPM'
[global]
daemonize = no
error_log = /tmp/tong-php-fpm-test.log
include = /home/ci/php-fpm-pool.conf
FPM
fpm_bin="$(command -v php-fpm || true)"
if [[ -z "$fpm_bin" ]]; then
  for candidate in /usr/sbin/php-fpm[0-9]*; do
    if [[ -x "$candidate" ]]; then fpm_bin="$candidate"; break; fi
  done
fi
test -n "$fpm_bin"
pool_user="$(awk -F ' = ' '/^user = / { print $2; exit }' /home/ci/php-fpm-pool.conf)"
pool_group="$(awk -F ' = ' '/^group = / { print $2; exit }' /home/ci/php-fpm-pool.conf)"
test -n "$pool_user"
test "$pool_user" = "$pool_group"
# This account exists only in the disposable guest so FPM can load its pool.
sudo useradd --system --user-group --no-create-home --shell /usr/sbin/nologin "$pool_user"
sudo "$fpm_bin" -t -y /home/ci/php-fpm-test.conf
cat > /home/ci/nginx-test.conf <<'NGINX'
worker_processes 1;
pid /tmp/tong-nginx-test.pid;
error_log stderr;
events { worker_connections 64; }
http {
    include /etc/nginx/mime.types;
    default_type application/octet-stream;
    include /home/ci/nginx-site.conf;
}
NGINX
sudo nginx -t -p /etc/nginx/ -c /home/ci/nginx-test.conf
TONG_TWO_TENANT_SITE_PREVIEW_DIR=/home/ci/site-e2e ./tong-p1-dev/tests/sites.test -test.run '^TestWriteTwoTenantSitePreviewsForCleanVM$' -test.timeout=1m
awk -F ' = ' '/^user = / { print $2 }' /home/ci/site-e2e/php-fpm-pools.conf > /home/ci/site-e2e/pool-users.txt
test "$(wc -l < /home/ci/site-e2e/pool-users.txt)" -eq 2
pool_user_a="$(sed -n '1p' /home/ci/site-e2e/pool-users.txt)"
pool_user_b="$(sed -n '2p' /home/ci/site-e2e/pool-users.txt)"
test -n "$pool_user_a"
test -n "$pool_user_b"
test "$pool_user_a" != "$pool_user_b"
while IFS= read -r pool_user; do
  if ! id -u "$pool_user" >/dev/null 2>&1; then
    sudo useradd --system --user-group --no-create-home --shell /usr/sbin/nologin "$pool_user"
  fi
done < /home/ci/site-e2e/pool-users.txt
sudo install -d -m 0755 /run/tong/php/tenant-a /run/tong/php/tenant-b
sudo install -d -o "$pool_user_a" -g "$pool_user_a" -m 0755 /srv/tong/tenants/tenant-a/sites/site-a/public
sudo install -d -o "$pool_user_b" -g "$pool_user_b" -m 0755 /srv/tong/tenants/tenant-b/sites/site-b/public
sudo install -d -o "$pool_user_a" -g "$pool_user_a" -m 0700 /srv/tong/tenants/tenant-a/run/site-a/tmp /srv/tong/tenants/tenant-a/run/site-a/sessions
sudo install -d -o "$pool_user_b" -g "$pool_user_b" -m 0700 /srv/tong/tenants/tenant-b/run/site-b/tmp /srv/tong/tenants/tenant-b/run/site-b/sessions
sudo tee /srv/tong/tenants/tenant-a/sites/site-a/public/index.php >/dev/null <<'PHP'
<?php
$other = @file_get_contents('/srv/tong/tenants/tenant-b/run/site-b/secret.txt');
echo $other === false ? 'tenant-a-isolated' : 'tenant-a-leaked';
PHP
sudo tee /srv/tong/tenants/tenant-b/sites/site-b/public/index.php >/dev/null <<'PHP'
<?php
$other = @file_get_contents('/srv/tong/tenants/tenant-a/run/site-a/secret.txt');
echo $other === false ? 'tenant-b-isolated' : 'tenant-b-leaked';
PHP
printf '%s\n' 'tenant-a-secret' | sudo tee /srv/tong/tenants/tenant-a/run/site-a/secret.txt >/dev/null
printf '%s\n' 'tenant-b-secret' | sudo tee /srv/tong/tenants/tenant-b/run/site-b/secret.txt >/dev/null
sudo chown "$pool_user_a:$pool_user_a" /srv/tong/tenants/tenant-a/sites/site-a/public/index.php /srv/tong/tenants/tenant-a/run/site-a/secret.txt
sudo chown "$pool_user_b:$pool_user_b" /srv/tong/tenants/tenant-b/sites/site-b/public/index.php /srv/tong/tenants/tenant-b/run/site-b/secret.txt
sudo chmod 0644 /srv/tong/tenants/tenant-a/sites/site-a/public/index.php /srv/tong/tenants/tenant-b/sites/site-b/public/index.php
sudo chmod 0600 /srv/tong/tenants/tenant-a/run/site-a/secret.txt /srv/tong/tenants/tenant-b/run/site-b/secret.txt
cat > /home/ci/php-fpm-e2e.conf <<'FPM'
[global]
daemonize = yes
pid = /run/tong/php-fpm-e2e.pid
error_log = /tmp/tong-php-fpm-e2e.log
include = /home/ci/site-e2e/php-fpm-pools.conf
FPM
sudo "$fpm_bin" -D -y /home/ci/php-fpm-e2e.conf
for _ in {1..30}; do
  [[ -S /run/tong/php/tenant-a/site-a.sock && -S /run/tong/php/tenant-b/site-b.sock ]] && break
  sleep 1
done
test -S /run/tong/php/tenant-a/site-a.sock
test -S /run/tong/php/tenant-b/site-b.sock
sudo systemctl stop nginx || true
cat > /home/ci/nginx-e2e.conf <<'NGINX'
user www-data;
worker_processes 1;
pid /run/tong/nginx-e2e.pid;
error_log /tmp/tong-nginx-e2e.log;
events { worker_connections 64; }
http {
    include /etc/nginx/mime.types;
    default_type application/octet-stream;
    include /home/ci/site-e2e/nginx-sites.conf;
}
NGINX
sudo nginx -t -p /etc/nginx/ -c /home/ci/nginx-e2e.conf
sudo nginx -p /etc/nginx/ -c /home/ci/nginx-e2e.conf
cleanup_site_services() {
  sudo systemctl stop mariadb >/dev/null 2>&1 || true
  sudo nginx -s quit -p /etc/nginx/ -c /home/ci/nginx-e2e.conf >/dev/null 2>&1 || true
  sudo nginx -s quit -p /etc/nginx/ -c /home/ci/nginx-https-e2e.conf >/dev/null 2>&1 || true
  if [[ -f /run/tong/php-fpm-e2e.pid ]]; then
    sudo kill -TERM "$(cat /run/tong/php-fpm-e2e.pid)" 2>/dev/null || true
  fi
}
trap cleanup_site_services EXIT
python3 - <<'PY'
from urllib.error import HTTPError
from urllib.request import Request, urlopen

def get(host, path="/"):
    req = Request("http://127.0.0.1" + path, headers={"Host": host})
    try:
        with urlopen(req, timeout=10) as response:
            return response.status, response.read().decode()
    except HTTPError as error:
        return error.code, error.read().decode()

for host, marker in [
    ("tenant-a.example.test", "tenant-a-isolated"),
    ("tenant-b.example.test", "tenant-b-isolated"),
]:
    status, body = get(host)
    assert status == 200 and marker in body, (host, status, body)
    hidden_status, _ = get(host, "/.private.php")
    assert hidden_status == 403, (host, hidden_status)
print("two-tenant Nginx/PHP-FPM smoke passed; cross-tenant reads denied and hidden files blocked")
PY
sudo nginx -s quit -p /etc/nginx/ -c /home/ci/nginx-e2e.conf
sudo install -d -m 0755 /etc/tong/certs/tenant-a/site-a /etc/tong/certs/tenant-b/site-b
sudo openssl req -x509 -nodes -days 1 -newkey rsa:2048 -subj "/CN=tenant-a.example.test" -addext "subjectAltName=DNS:tenant-a.example.test,DNS:www.tenant-a.example.test" -keyout /etc/tong/certs/tenant-a/site-a/privkey.pem -out /etc/tong/certs/tenant-a/site-a/fullchain.pem >/dev/null 2>&1
sudo openssl req -x509 -nodes -days 1 -newkey rsa:2048 -subj "/CN=tenant-b.example.test" -addext "subjectAltName=DNS:tenant-b.example.test" -keyout /etc/tong/certs/tenant-b/site-b/privkey.pem -out /etc/tong/certs/tenant-b/site-b/fullchain.pem >/dev/null 2>&1
sudo chmod 0600 /etc/tong/certs/tenant-a/site-a/privkey.pem /etc/tong/certs/tenant-b/site-b/privkey.pem
TONG_NGINX_HTTPS_PREVIEW_PATH=/home/ci/site-e2e/nginx-https-sites.conf ./tong-p1-dev/tests/sites.test -test.run '^TestRenderNginxHTTPSVhostRequiresVerifiedDomainsAndCanonicalRedirect$' -test.timeout=1m
cat > /home/ci/nginx-https-e2e.conf <<'NGINX'
user www-data;
worker_processes 1;
pid /run/tong/nginx-https-e2e.pid;
error_log /tmp/tong-nginx-https-e2e.log;
events { worker_connections 64; }
http {
    include /etc/nginx/mime.types;
    default_type application/octet-stream;
    include /home/ci/site-e2e/nginx-https-sites.conf;
}
NGINX
sudo nginx -t -p /etc/nginx/ -c /home/ci/nginx-https-e2e.conf
sudo nginx -p /etc/nginx/ -c /home/ci/nginx-https-e2e.conf
sudo install -d -o root -g root -m 0755 /srv/tong/tenants/tenant-a/sites/site-a/public/.well-known/acme-challenge
sudo install -d -o root -g root -m 0755 /srv/tong/tenants/tenant-b/sites/site-b/public/.well-known/acme-challenge
printf '%s\n' 'tenant-a-challenge' | sudo tee /srv/tong/tenants/tenant-a/sites/site-a/public/.well-known/acme-challenge/token >/dev/null
printf '%s\n' 'tenant-b-challenge' | sudo tee /srv/tong/tenants/tenant-b/sites/site-b/public/.well-known/acme-challenge/token >/dev/null
for host in tenant-a.example.test tenant-b.example.test; do
  headers="$(curl --silent --show-error --resolve "$host:80:127.0.0.1" -D - -o /dev/null "http://$host/path?from=http")"
  grep -qE '^HTTP/[^ ]+ 308' <<<"$headers"
  grep -qi "^[Ll]ocation: https://$host/path?from=http" <<<"$headers"
done
body_a="$(curl --silent --show-error --insecure --resolve tenant-a.example.test:443:127.0.0.1 https://tenant-a.example.test/)"
body_b="$(curl --silent --show-error --insecure --resolve tenant-b.example.test:443:127.0.0.1 https://tenant-b.example.test/)"
[[ "$body_a" == *tenant-a-isolated* && "$body_b" == *tenant-b-isolated* ]]
[[ "$(curl --silent --show-error --resolve tenant-a.example.test:80:127.0.0.1 http://tenant-a.example.test/.well-known/acme-challenge/token)" == tenant-a-challenge ]]
[[ "$(curl --silent --show-error --resolve tenant-b.example.test:80:127.0.0.1 http://tenant-b.example.test/.well-known/acme-challenge/token)" == tenant-b-challenge ]]
hidden_status="$(curl --silent --show-error --insecure --resolve tenant-a.example.test:443:127.0.0.1 -o /dev/null -w '%{http_code}' https://tenant-a.example.test/.private.php)"
[[ "$hidden_status" == 403 ]]
printf '%s\n' 'two-tenant HTTPS smoke passed; HTTP redirects, ACME challenge path stays on HTTP, TLS sites remain isolated'
# The preflight is read-only: this intentionally reports the ports occupied by
# the disposable Nginx fixture as blockers and exits 3 rather than authorizing
# any installation step. The command has no mutation implementation.
if /home/ci/tong-p1-dev/bin/tongctl preflight --min-disk-bytes 1 --required-tcp-ports 80,443 > install-preflight.json; then
  preflight_status=0
else
  preflight_status=$?
fi
[[ "$preflight_status" == 3 ]]
python3 -c 'import json; d=json.load(open("install-preflight.json")); assert d["gate"]["allowed"] is False; assert "tcp_port_in_use:80" in d["gate"]["blockers"]; assert "tcp_port_in_use:443" in d["gate"]["blockers"]; assert "not installation authorization" in d["notice"]; print("clean-guest install preflight correctly blocked occupied web ports without host mutation")'
bash ./tong-p1-dev/tests/mariadb-isolation.sh
bash ./tong-p1-dev/tests/encrypted-backup-restore.sh
python3 /home/ci/smoke_p1.py /home/ci/tong-p1-dev/bin/tongd
/home/ci/tong-p1-dev/bin/tongctl doctor > doctor.json
python3 -c 'import json; d=json.load(open("doctor.json")); assert len(d["checks"]) == 4; print("clean-guest doctor JSON parsed; support remains uncertified")'
bash ./p2_package_install_vm.sh ./tong-panel-dev-v1.deb ./tong-panel-dev-v2.deb
GUEST
scp "${scp_opts[@]}" "$work/guest-check.sh" "$guest:/home/ci/guest-check.sh"
ssh "${ssh_opts[@]}" "$guest" bash /home/ci/guest-check.sh "$VM_EXPECTED_ID" "$VM_EXPECTED_VERSION"

# Copy only the ciphertext and its externally held, restrictive-permission
# test key to the runner, destroy the source VM disk, and restore on a new
# overlay from the pristine cloud image. The runner temp directory is transient
# test staging; this is not a real offsite provider or durable backup service.
retry_guest_scp "$guest:/home/ci/tenant-a.backup.tpbk" "$guest:/home/ci/wrapping.key" "$work/"
chmod 600 "$work/tenant-a.backup.tpbk" "$work/wrapping.key"
if grep -aFq 'private-tenant-restore-fixture-v1' "$work/tenant-a.backup.tpbk"; then
  echo "runner staging received plaintext instead of an encrypted backup" >&2
  exit 1
fi
source_vm_pid="$(cat "$pidfile")"
ssh "${ssh_opts[@]}" "$guest" sudo poweroff >/dev/null 2>&1 || true
for _ in {1..30}; do
  kill -0 "$source_vm_pid" 2>/dev/null || break
  sleep 1
done
if kill -0 "$source_vm_pid" 2>/dev/null; then
  kill "$source_vm_pid" 2>/dev/null || true
fi
rm -f -- "$pidfile" "$disk"

cat > "$work/restore-user-data" <<EOF
#cloud-config
users:
  - name: ci
    gecos: Tong clean restore guest
    sudo: ALL=(ALL) NOPASSWD:ALL
    lock_passwd: true
    shell: /bin/bash
    ssh_authorized_keys:
      - $pub
ssh_pwauth: false
disable_root: true
EOF
cat > "$work/restore-meta-data" <<EOF
instance-id: tong-restore-$VM_DISTRO-$(date +%s)
local-hostname: tong-restore-$VM_DISTRO
EOF
cloud-localds "$seed" "$work/restore-user-data" "$work/restore-meta-data"
qemu-img create -q -f qcow2 -F qcow2 -b "$base" "$disk" 8G
serial="$work/restore-serial.log"
qemu-system-x86_64 -machine accel=tcg -cpu max -m 2048 -smp 2 -daemonize -pidfile "$pidfile" -serial "file:$serial" -monitor none -display none -drive "file=$disk,if=virtio,format=qcow2" -drive "file=$seed,if=virtio,format=raw,readonly=on" -netdev "user,id=net0,hostfwd=tcp:127.0.0.1:$port-:22" -device virtio-net-pci,netdev=net0
ready=false
for _ in {1..60}; do
  if ssh "${ssh_opts[@]}" "$guest" true >/dev/null 2>&1; then ready=true; break; fi
  if ! kill -0 "$(cat "$pidfile")" 2>/dev/null; then echo "restore QEMU exited before SSH"; exit 1; fi
  sleep 5
done
[[ "$ready" == true ]] || { echo "Timed out waiting for fresh restore VM SSH"; exit 1; }
retry_guest_ssh "cloud-init status --wait"
retry_guest_scp "$work/tenant-a.backup.tpbk" "$work/wrapping.key" "$guest:/home/ci/"
retry_guest_scp "$VM_RUNNER_TEMP/tong-p1-dev/bin/tongbackup" "$guest:/home/ci/tongbackup"
cat > "$work/restore-check.sh" <<'RESTORE'
#!/usr/bin/env bash
set -Eeuo pipefail
sudo apt-get update
sudo apt-get install -y --no-install-recommends mariadb-server
sudo systemctl start mariadb
sudo mariadb --protocol=socket -uroot -e 'CREATE DATABASE tp_backup_source'
./tongbackup decrypt --tenant tenant-a --key-file /home/ci/wrapping.key --output /home/ci/restore.sql < /home/ci/tenant-a.backup.tpbk
sudo mariadb --protocol=socket -uroot tp_backup_source < /home/ci/restore.sql
restored="$(sudo mariadb --protocol=socket -uroot --batch --skip-column-names tp_backup_source -e 'SELECT value FROM restore_marker')"
[[ "$restored" == private-tenant-restore-fixture-v1 ]]
if ./tongbackup decrypt --tenant tenant-b --key-file /home/ci/wrapping.key --output /home/ci/should-not-exist.sql < /home/ci/tenant-a.backup.tpbk >/dev/null 2>&1; then
  echo 'fresh-host restore accepted the wrong tenant' >&2
  exit 1
fi
[[ ! -e /home/ci/should-not-exist.sql ]]
rm -f -- /home/ci/wrapping.key /home/ci/restore.sql
printf '%s\n' 'PASS: encrypted MariaDB backup restored on a fresh VM from a pristine distro image'
RESTORE
retry_guest_scp "$work/restore-check.sh" "$guest:/home/ci/restore-check.sh"
ssh "${ssh_opts[@]}" "$guest" bash /home/ci/restore-check.sh
