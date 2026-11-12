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
  control_dir="$work/control-$(basename "$package" .deb)"
  data_dir="$work/data-$(basename "$package" .deb)"
  mkdir -p "$control_dir" "$data_dir"
  dpkg-deb --control "$package" "$control_dir"
  dpkg-deb --extract "$package" "$data_dir"
  for script in postinst prerm postrm; do
    [[ -x "$control_dir/$script" ]]
    cmp "$control_dir/$script" "packaging/debian/$script"
  done
  cmp "$data_dir/usr/lib/systemd/system/tong-panel-dev.service" "packaging/debian/tong-panel-dev.service"
done
[[ "$(dpkg-deb --field "$out/tong-panel-dev-v1.deb" Version)" == 0.0.0~dev.1 ]]
[[ "$(dpkg-deb --field "$out/tong-panel-dev-v2.deb" Version)" == 0.0.0~dev.2 ]]

# Keep package installation inert. Only the disposable-VM lifecycle test may
# start the service, and package removal must preserve its state directory.
if grep -Eq 'systemctl[[:space:]]+(enable|start|restart)' packaging/debian/postinst; then
  echo "postinst must not enable or start the development service" >&2
  exit 1
fi
grep -Fq 'systemctl daemon-reload' packaging/debian/postinst
grep -Fq 'upgrade|remove|deconfigure)' packaging/debian/prerm
grep -Fq 'systemctl stop tong-panel-dev.service' packaging/debian/prerm
if grep -Eq 'rm[[:space:]].*(/var/lib/tong-panel-dev|TONG_DATA_DIR)' packaging/debian/postrm; then
  echo "postrm must preserve development service state" >&2
  exit 1
fi
grep -Fxq 'DynamicUser=yes' packaging/debian/tong-panel-dev.service
grep -Fxq 'StateDirectoryMode=0700' packaging/debian/tong-panel-dev.service
grep -Fxq 'UMask=0077' packaging/debian/tong-panel-dev.service
grep -Fxq 'NoNewPrivileges=yes' packaging/debian/tong-panel-dev.service
grep -Fxq 'ProtectSystem=strict' packaging/debian/tong-panel-dev.service
grep -Fxq 'CapabilityBoundingSet=' packaging/debian/tong-panel-dev.service
grep -Fxq 'AmbientCapabilities=' packaging/debian/tong-panel-dev.service
(
  cd "$out"
  sha256sum tong-panel-dev-v1.deb tong-panel-dev-v2.deb > PACKAGES.SHA256SUMS
)
printf '%s\n' 'Debian package structure, embedded lifecycle files, safe defaults, version guards, and disclaimer checks passed.'
