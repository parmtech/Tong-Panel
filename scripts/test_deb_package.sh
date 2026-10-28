#!/usr/bin/env bash
set -Eeuo pipefail

if [[ $# -ne 3 ]]; then
  echo "usage: $0 OUTPUT_DIRECTORY TONGD_BINARY TONGCTL_BINARY" >&2
  exit 2
fi

out=$1
tongd=$2
tongctl=$3
work="$(mktemp -d "${TMPDIR:-/tmp}/tong-package-test.XXXXXX")"
trap 'rm -rf -- "$work"' EXIT

if bash scripts/build_deb.sh "$work/invalid.deb" "invalid version" "$tongd" "$tongctl" >/dev/null 2>&1; then
  echo "package builder accepted a malformed version" >&2
  exit 1
fi
if bash scripts/build_deb.sh "$work/missing.deb" "0.0.0~dev.1" "$work/absent-tongd" "$tongctl" >/dev/null 2>&1; then
  echo "package builder accepted a missing binary" >&2
  exit 1
fi

mkdir -p "$out"
bash scripts/build_deb.sh "$out/tong-panel-dev-v1.deb" "0.0.0~dev.1" "$tongd" "$tongctl"
bash scripts/build_deb.sh "$out/tong-panel-dev-v2.deb" "0.0.0~dev.2" "$tongd" "$tongctl"
for package in "$out/tong-panel-dev-v1.deb" "$out/tong-panel-dev-v2.deb"; do
  [[ "$(dpkg-deb --field "$package" Package)" == tong-panel-dev ]]
  [[ "$(dpkg-deb --field "$package" Architecture)" == amd64 ]]
  [[ "$(dpkg-deb --field "$package" Depends)" == *systemd* ]]
  contents="$(dpkg-deb --contents "$package")"
  grep -q './usr/libexec/tong-panel-dev/tongd' <<<"$contents"
  grep -q './usr/bin/tongctl' <<<"$contents"
  grep -q './usr/lib/systemd/system/tong-panel-dev.service' <<<"$contents"
  grep -q './usr/share/doc/tong-panel-dev/DO_NOT_INSTALL.txt' <<<"$contents"
  if grep -qE '^\./DEBIAN/(preinst|config)$' <<<"$contents"; then
    echo "unexpected pre-install package script found" >&2
    exit 1
  fi
done
[[ "$(dpkg-deb --field "$out/tong-panel-dev-v1.deb" Version)" == 0.0.0~dev.1 ]]
[[ "$(dpkg-deb --field "$out/tong-panel-dev-v2.deb" Version)" == 0.0.0~dev.2 ]]
(
  cd "$out"
  sha256sum tong-panel-dev-v1.deb tong-panel-dev-v2.deb > PACKAGES.SHA256SUMS
)
printf '%s\n' 'Debian package structure, version guards, and disclaimer checks passed.'
