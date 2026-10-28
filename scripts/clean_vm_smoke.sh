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
  rm -rf "$work"
  exit "$status"
}
trap cleanup EXIT
sudo apt-get update
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
retry_guest_scp scripts/smoke_p1.py "$guest:/home/ci/smoke_p1.py"
cat > "$work/guest-check.sh" <<'GUEST'
#!/usr/bin/env bash
set -Eeuo pipefail
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
test -x tong-p1-dev/tests/files.test
test -x tong-p1-dev/tests/api.test
test -x tong-p1-dev/tests/sites.test
grep -q 'UNSIGNED DEVELOPER PREVIEW ONLY' tong-p1-dev/DO_NOT_INSTALL.txt
./tong-p1-dev/tests/files.test -test.v -test.timeout=4m
./tong-p1-dev/tests/api.test -test.v -test.timeout=4m
./tong-p1-dev/tests/sites.test -test.v -test.timeout=4m
sudo apt-get update
sudo apt-get install -y --no-install-recommends nginx php-fpm
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
python3 /home/ci/smoke_p1.py /home/ci/tong-p1-dev/bin/tongd
/home/ci/tong-p1-dev/bin/tongctl doctor > doctor.json
python3 -c 'import json; d=json.load(open("doctor.json")); assert len(d["checks"]) == 4; print("clean-guest doctor JSON parsed; support remains uncertified")'
GUEST
scp "${scp_opts[@]}" "$work/guest-check.sh" "$guest:/home/ci/guest-check.sh"
ssh "${ssh_opts[@]}" "$guest" bash /home/ci/guest-check.sh "$VM_EXPECTED_ID" "$VM_EXPECTED_VERSION"
