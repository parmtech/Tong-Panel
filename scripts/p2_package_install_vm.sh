#!/usr/bin/env bash
set -Eeuo pipefail

if [[ $# -ne 2 ]]; then
  echo "usage: $0 V1_PACKAGE V2_PACKAGE" >&2
  exit 2
fi
v1=$1
v2=$2

preflight_work="$(mktemp -d)"
preflight_listener_pid=""
finish() {
  status=$?
  if [[ -n "$preflight_listener_pid" ]]; then
    kill "$preflight_listener_pid" 2>/dev/null || true
    wait "$preflight_listener_pid" 2>/dev/null || true
  fi
  rm -rf -- "$preflight_work"
  if [[ $status -ne 0 ]]; then
    sudo systemctl status tong-panel-dev.service --no-pager || true
    sudo journalctl -u tong-panel-dev.service -n 80 --no-pager || true
  fi
  exit "$status"
}
trap finish EXIT

[[ "$(id -u)" != 0 ]]
[[ "$(uname -m)" == x86_64 ]]
[[ -f "$v1" && -f "$v2" ]]
sha256sum -c PACKAGES.SHA256SUMS

dpkg-deb --extract "$v1" "$preflight_work/extracted"
preflightctl="$preflight_work/extracted/usr/bin/tongctl"
[[ -x "$preflightctl" ]]

assert_package_absent() {
  if dpkg-query -W -f='${db:Status-Abbrev}' tong-panel-dev 2>/dev/null | grep -q '^ii'; then
    echo "package was installed before preflight authorization" >&2
    exit 1
  fi
}
assert_preflight_blocker() {
  python3 - "$1" "$2" <<'PY'
import json, sys
document = json.load(open(sys.argv[1], encoding="utf-8"))
assert document["gate"]["allowed"] is False, document
assert sys.argv[2] in document["gate"]["blockers"], document
assert "not installation authorization" in document["notice"], document
PY
}
assert_package_absent

# A policy threshold above any possible disk size must block before dpkg mutates the guest.
if "$preflightctl" preflight --min-disk-bytes 18446744073709551615 --required-tcp-ports 8090 > "$preflight_work/low-disk.json"; then
  low_disk_status=0
else
  low_disk_status=$?
fi
[[ "$low_disk_status" == 3 ]]
assert_preflight_blocker "$preflight_work/low-disk.json" "disk_below_minimum"
assert_package_absent

# Occupy the actual developer service port on loopback and require preflight to block before installation.
python3 - "$preflight_work/listener-ready" <<'PY' &
import socket, sys, time
listener = socket.socket()
listener.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
listener.bind(("127.0.0.1", 8090))
listener.listen()
open(sys.argv[1], "w", encoding="ascii").close()
while True:
    time.sleep(1)
PY
preflight_listener_pid=$!
for _ in {1..50}; do
  [[ -f "$preflight_work/listener-ready" ]] && break
  kill -0 "$preflight_listener_pid" 2>/dev/null || break
  sleep 0.1
done
if [[ ! -f "$preflight_work/listener-ready" ]]; then
  echo "could not start the isolated listener on 127.0.0.1:8090" >&2
  exit 1
fi
if "$preflightctl" preflight --min-disk-bytes 1 --required-tcp-ports 8090 > "$preflight_work/occupied-port.json"; then
  occupied_status=0
else
  occupied_status=$?
fi
[[ "$occupied_status" == 3 ]]
assert_preflight_blocker "$preflight_work/occupied-port.json" "tcp_port_in_use:8090"
assert_package_absent
kill "$preflight_listener_pid"
wait "$preflight_listener_pid" 2>/dev/null || true
preflight_listener_pid=""

# Installation proceeds only after the real guest facts meet the lab policy.
"$preflightctl" preflight --min-disk-bytes 1 --required-tcp-ports 8090 > "$preflight_work/allowed.json"
python3 - "$preflight_work/allowed.json" <<'PY'
import json, sys
document = json.load(open(sys.argv[1], encoding="utf-8"))
assert document["gate"]["allowed"] is True, document
PY
assert_package_absent

sudo dpkg --install "$v1"
[[ "$(dpkg-query -W -f='${Version}' tong-panel-dev)" == 0.0.0~dev.1 ]]
if sudo systemctl is-active --quiet tong-panel-dev.service; then
  echo "package unexpectedly started its service during installation" >&2
  exit 1
fi
[[ "$(sudo systemctl is-enabled tong-panel-dev.service 2>/dev/null || true)" == disabled ]]

wait_for_service() {
  for _ in {1..20}; do
    if curl --silent --show-error --fail http://127.0.0.1:8090/healthz >/dev/null; then
      return 0
    fi
    sleep 1
  done
  echo "loopback service did not become healthy" >&2
  return 1
}
assert_default_deny() {
  python3 - <<'PY'
from urllib.error import HTTPError
from urllib.request import urlopen
import json
try:
    urlopen("http://127.0.0.1:8090/api/v1/me", timeout=5)
except HTTPError as error:
    body = json.loads(error.read())
    assert error.code == 503, (error.code, body)
    assert body.get("error", {}).get("code") == "auth_unconfigured", body
else:
    raise AssertionError("protected API did not fail closed without configured authentication")
PY
}
assert_loopback_only() {
  python3 - <<'PY'
entries = []
for line in open("/proc/net/tcp", encoding="ascii").read().splitlines()[1:]:
    fields = line.split()
    if fields[3] != "0A":  # only the listening socket; accepted connections repeat its local port
        continue
    local = fields[1]
    address, port = local.split(":")
    if int(port, 16) == 8090:
        entries.append(local)
assert entries == ["0100007F:1F9A"], entries
assert not any(
    line.split()[3] == "0A" and int(line.split()[1].split(":")[1], 16) == 8090
    for line in open("/proc/net/tcp6", encoding="ascii").read().splitlines()[1:]
), "IPv6 listener present"
print("Tong Panel service listens only on 127.0.0.1:8090")
PY
}
assert_non_root_service() {
  pid="$(sudo systemctl show --property=MainPID --value tong-panel-dev.service)"
  [[ "$pid" =~ ^[1-9][0-9]*$ ]]
  service_uid="$(ps -o uid= -p "$pid" | xargs)"
  [[ -n "$service_uid" && "$service_uid" != 0 ]]
  printf 'service runs as uid %s\n' "$service_uid"
}

sudo systemctl start tong-panel-dev.service
wait_for_service
assert_default_deny
assert_loopback_only
assert_non_root_service
page="$(curl --silent --show-error --fail http://127.0.0.1:8090/)"
[[ "$page" == *'Development preview'* ]]
sudo sh -c 'printf "%s\n" "keep-through-package-rollback" > /var/lib/tong-panel-dev/package-lab-marker'

sudo dpkg --install "$v2"
[[ "$(dpkg-query -W -f='${Version}' tong-panel-dev)" == 0.0.0~dev.2 ]]
if sudo systemctl is-active --quiet tong-panel-dev.service; then
  echo "upgrade did not stop the old service before replacing its files" >&2
  exit 1
fi
sudo systemctl start tong-panel-dev.service
wait_for_service
assert_default_deny
assert_loopback_only
assert_non_root_service
[[ "$(sudo cat /var/lib/tong-panel-dev/package-lab-marker)" == keep-through-package-rollback ]]

sudo dpkg --install "$v1"
[[ "$(dpkg-query -W -f='${Version}' tong-panel-dev)" == 0.0.0~dev.1 ]]
sudo systemctl start tong-panel-dev.service
wait_for_service
assert_default_deny
assert_loopback_only
assert_non_root_service

sudo dpkg --remove tong-panel-dev
if dpkg-query -W -f='${Status}' tong-panel-dev 2>/dev/null | grep -q 'installed$'; then
  echo "package removal left tong-panel-dev installed" >&2
  exit 1
fi
[[ ! -e /usr/lib/systemd/system/tong-panel-dev.service ]]
[[ ! -e /usr/libexec/tong-panel-dev/tongd ]]
[[ "$(sudo cat /var/lib/tong-panel-dev/package-lab-marker)" == keep-through-package-rollback ]]
printf '%s\n' 'Disposable Debian package install, loopback/default-deny, upgrade, downgrade, and non-destructive removal checks passed.'
