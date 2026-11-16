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

if [[ -z "$output" || "$output" == */ ]]; then
  echo "package output must name a file without a trailing slash: $output" >&2
  exit 2
fi

output_parent="$(dirname -- "$output")"
output_name="$(basename -- "$output")"
if [[ -z "$output_name" || "$output_name" == . || "$output_name" == .. ]]; then
  echo "package output must name a file: $output" >&2
  exit 2
fi

check_output_target() {
  local target=$1
  if [[ -L "$target" ]]; then
    echo "package output must not be a symbolic link: $target" >&2
    return 2
  fi
  if [[ -e "$target" && ! -f "$target" ]]; then
    echo "package output must be a regular file or absent: $target" >&2
    return 2
  fi
}

script_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd -- "$script_dir/.." && pwd)"
work="$(mktemp -d "${TMPDIR:-/tmp}/tong-panel-deb.XXXXXX")"
staging_dir=""
cleanup() {
  rm -rf -- "$work" 2>/dev/null || true
  if [[ -n "$staging_dir" ]]; then
    rm -rf -- "$staging_dir" 2>/dev/null || true
  fi
}
trap cleanup EXIT
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

mkdir -p -- "$output_parent"
check_output_target "$output"

# Keep the staged archive on the destination filesystem so the final rename is
# atomic. The output parent must be trusted; this does not defend against a
# hostile parent-component swap or provide crash durability.
staging_dir="$(mktemp -d -- "$output_parent/.tong-panel-deb.XXXXXX")"
staged="$staging_dir/$output_name"
dpkg-deb --build --root-owner-group "$root" "$staged" >/dev/null
dpkg-deb --info "$staged" >/dev/null
# Publish a caller-owned regular file with a stable mode independent of umask.
chmod 0644 -- "$staged"

# Recheck after validation to avoid replacing a destination whose type changed
# while the package was being built.
check_output_target "$output"
mv -fT -- "$staged" "$output"
