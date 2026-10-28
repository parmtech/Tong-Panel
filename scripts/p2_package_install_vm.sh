#!/usr/bin/env bash
set -Eeuo pipefail

if [[ $# -ne 2 ]]; then
  echo "usage: $0 V1_PACKAGE V2_PACKAGE" >&2
  exit 2
fi
v1=$1
v2=$2

trap 'status=$?; if [[ $status -ne 0 ]]; then sudo systemctl status tong-panel-dev.service --no-pager || true; sudo journalctl -u tong-panel-dev.service -n 80 --no-pager || true; fi; exit "$status"' EXIT

[[ "$(id -u)" != 0 ]]
[[ "$(uname -m)" == x86_64 ]]
[[ -f "$v1" && -f "$v2" ]]
sha256sum -c PACKAGES.SHA256SUMS

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
curl --silent --show-error --fail http://127.0.0.1:8090/ | grep -q 'Development preview'
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
