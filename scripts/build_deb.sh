#!/usr/bin/env bash
set -Eeuo pipefail

if [[ $# -ne 4 ]]; then
  echo "usage: $0 OUTPUT.deb VERSION TONGD_BINARY TONGCTL_BINARY" >&2
  exit 2
fi

output=$1
version=$2
tongd=$3
tongctl=$4

if [[ ! "$version" =~ ^[0-9][A-Za-z0-9.+:~]*$ ]]; then
  echo "invalid Debian package version" >&2
  exit 2
fi
if [[ "$(dpkg --print-architecture)" != amd64 ]]; then
  echo "only the explicitly tested amd64 package is supported" >&2
  exit 2
fi
for binary in "$tongd" "$tongctl"; do
  if [[ ! -f "$binary" || ! -x "$binary" ]]; then
    echo "required executable is missing: $binary" >&2
    exit 2
  fi
done

script_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd -- "$script_dir/.." && pwd)"
work="$(mktemp -d "${TMPDIR:-/tmp}/tong-panel-deb.XXXXXX")"
trap 'rm -rf -- "$work"' EXIT
root="$work/root"

install -D -m 0755 "$tongd" "$root/usr/libexec/tong-panel-dev/tongd"
install -D -m 0755 "$tongctl" "$root/usr/bin/tongctl"
install -D -m 0644 "$repo_root/packaging/debian/tong-panel-dev.service" "$root/usr/lib/systemd/system/tong-panel-dev.service"
install -D -m 0644 "$repo_root/packaging/debian/DO_NOT_INSTALL.txt" "$root/usr/share/doc/tong-panel-dev/DO_NOT_INSTALL.txt"
install -D -m 0644 "$repo_root/LICENSE" "$root/usr/share/doc/tong-panel-dev/copyright"
install -D -m 0755 "$repo_root/packaging/debian/postinst" "$root/DEBIAN/postinst"
install -D -m 0755 "$repo_root/packaging/debian/prerm" "$root/DEBIAN/prerm"
install -D -m 0755 "$repo_root/packaging/debian/postrm" "$root/DEBIAN/postrm"
sed "s/^Version: @VERSION@$/Version: $version/" "$repo_root/packaging/debian/tong-panel-dev.control" > "$root/DEBIAN/control"

mkdir -p "$(dirname -- "$output")"
dpkg-deb --build --root-owner-group "$root" "$output" >/dev/null
dpkg-deb --info "$output" >/dev/null
