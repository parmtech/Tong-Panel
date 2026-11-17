#!/usr/bin/env bash
set -Eeuo pipefail

if [[ $# -lt 3 || $# -gt 4 ]]; then
  echo "usage: $0 OUTPUT_DIRECTORY TONGD_BINARY TONGCTL_BINARY [--publication-only|--publication-failures-only]" >&2
  exit 2
fi

out=$1
tongd=$2
tongctl=$3
publication_mode=
if [[ $# -eq 4 ]]; then
  case "$4" in
    --publication-only|--publication-failures-only) publication_mode=$4 ;;
    *)
      echo "usage: $0 OUTPUT_DIRECTORY TONGD_BINARY TONGCTL_BINARY [--publication-only|--publication-failures-only]" >&2
      exit 2
      ;;
  esac
fi
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

mkdir -p "$work/fake-bin"
cat >"$work/fake-bin/dpkg" <<'DPKG'
#!/usr/bin/env bash
set -eu
if [[ "$#" -ne 1 || "$1" != "--print-architecture" ]]; then
  exit 93
fi
printf '%s\n' arm64
DPKG
chmod 0755 "$work/fake-bin/dpkg"
if PATH="$work/fake-bin:$PATH" bash scripts/build_deb.sh "$work/non-amd64.deb" "0.0.0~dev.1" "$tongd" "$tongctl" >"$work/non-amd64.stdout" 2>"$work/non-amd64.stderr"; then
  echo "package builder accepted a non-amd64 architecture" >&2
  exit 1
else
  architecture_status=$?
fi
[[ "$architecture_status" == 2 ]]
[[ ! -e "$work/non-amd64.deb" ]]

mkdir -p "$work/publication-bin" "$work/private-tmp" "$work/publication output"
cat >"$work/publication-bin/dpkg" <<'DPKG'
#!/usr/bin/env bash
set -eu
if [[ "$#" -ne 1 || "$1" != "--print-architecture" ]]; then
  exit 93
fi
printf '%s\n' amd64
DPKG
cat >"$work/publication-bin/dpkg-deb" <<'DPKGDEB'
#!/usr/bin/env bash
set -eu
mode=${DPKG_DEB_TEST_MODE:-success}
if [[ -n "${DPKG_DEB_CALL_LOG:-}" ]]; then
  printf '%s\n' "$1" >>"$DPKG_DEB_CALL_LOG"
fi
case "$mode:$1" in
  partial-build:--build)
    archive=${!#}
    printf '%s\n' 'partial synthetic archive' >"$archive"
    exit 41
    ;;
  info-failure:--build)
    archive=${!#}
    printf '%s\n' 'synthetic archive' >"$archive"
    ;;
  info-failure:--info)
    exit 42
    ;;
  destination-directory-during-info:--build)
    archive=${!#}
    printf '%s\n' 'synthetic archive' >"$archive"
    ;;
  destination-directory-during-info:--info)
    mkdir -p -- "$DPKG_DEB_MUTATE_TARGET"
    ;;
  destination-symlink-during-info:--build)
    archive=${!#}
    printf '%s\n' 'synthetic archive' >"$archive"
    ;;
  destination-symlink-during-info:--info)
    ln -s "$DPKG_DEB_MUTATE_REFERENT" "$DPKG_DEB_MUTATE_TARGET"
    ;;
  success:--build)
    archive=${!#}
    if [[ -n "${DPKG_DEB_EXPECT_PARENT:-}" ]]; then
      stage_parent=${archive%/*}
      [[ "${stage_parent%/*}" == "$DPKG_DEB_EXPECT_PARENT" ]] || {
        echo 'archive was not staged in the exact output parent' >&2
        exit 95
      }
      [[ "${archive##*/}" == "$DPKG_DEB_EXPECT_NAME" ]] || exit 96
    fi
    printf '%s\n' 'synthetic archive' >"$archive"
    ;;
  success:--info)
    [[ "$#" -eq 2 && -s "$2" ]]
    ;;
  *)
    echo "unexpected dpkg-deb call for test mode $mode: $*" >&2
    exit 94
    ;;
esac
DPKGDEB
chmod 0755 "$work/publication-bin/dpkg" "$work/publication-bin/dpkg-deb"

invalid_path_parent="$work/must-not-be-created"
invalid_trailing_target="$invalid_path_parent/trailing.deb/"
: >"$work/dpkg-deb.calls"
if PATH="$work/publication-bin:$PATH" DPKG_DEB_TEST_MODE=success \
  DPKG_DEB_CALL_LOG="$work/dpkg-deb.calls" \
  bash scripts/build_deb.sh "$invalid_trailing_target" "0.0.0~dev.1" "$tongd" "$tongctl" \
  >"$work/publication.stdout" 2>"$work/publication.stderr"; then
  echo "package builder accepted an output path with a trailing slash" >&2
  exit 1
else
  invalid_path_status=$?
fi
[[ "$invalid_path_status" == 2 ]]
[[ ! -e "$invalid_path_parent" && ! -L "$invalid_path_parent" ]]
[[ ! -s "$work/dpkg-deb.calls" ]]

assert_no_package_staging() {
  local parent=$1
  local temp_root=$2
  if find "$parent" -mindepth 1 -maxdepth 1 -type d -name '.tong-panel-deb.*' -print -quit | grep -q .; then
    echo "package builder left a destination staging directory in $parent" >&2
    exit 1
  fi
  if find "$temp_root" -mindepth 1 -maxdepth 1 -type d -name 'tong-panel-deb.*' -print -quit | grep -q .; then
    echo "package builder left a temporary work directory in $temp_root" >&2
    exit 1
  fi
}

run_publication_failure() {
  local mode=$1
  local target=$2
  local expected_status=$3
  local status
  if PATH="$work/publication-bin:$PATH" TMPDIR="$work/private-tmp" \
    DPKG_DEB_TEST_MODE="$mode" \
    bash scripts/build_deb.sh "$target" "0.0.0~dev.1" "$tongd" "$tongctl" \
    >"$work/publication.stdout" 2>"$work/publication.stderr"; then
    echo "package builder unexpectedly succeeded in $mode mode" >&2
    exit 1
  else
    status=$?
  fi
  [[ "$status" == "$expected_status" ]]
  assert_no_package_staging "$(dirname -- "$target")" "$work/private-tmp"
}

test_platform="$(uname -s)"
mode_probe="$work/mode-probe"
printf 'mode probe\n' >"$mode_probe"
chmod 0600 "$mode_probe"
mode_0600_result="$(stat -c '%a' "$mode_probe")"
chmod 0644 "$mode_probe"
mode_0644_result="$(stat -c '%a' "$mode_probe")"
if [[ "$test_platform" == Linux ]]; then
  if [[ "$mode_0600_result" != 600 || "$mode_0644_result" != 644 ]]; then
    echo "Linux did not preserve chmod 0600 and 0644 for the package publication test" >&2
    exit 1
  fi
  check_mode_assertions=true
  mode_check_summary='0600 sentinel and 0644 output mode checks passed'
elif [[ "$mode_0600_result" == 600 && "$mode_0644_result" == 644 ]]; then
  check_mode_assertions=true
  mode_check_summary='0600 sentinel and 0644 output mode checks passed'
else
  check_mode_assertions=false
  mode_check_summary='0600 sentinel and 0644 output mode checks skipped'
  printf 'SKIP: permission-mode checks; %s mount reports %s after chmod 0600 and %s after chmod 0644.\n' \
    "$test_platform" "$mode_0600_result" "$mode_0644_result" >&2
fi

symlink_probe_target="$work/symlink-probe-target"
symlink_probe="$work/symlink-probe"
printf 'symlink probe\n' >"$symlink_probe_target"
symlink_checks_supported=false
if ln -s "$symlink_probe_target" "$symlink_probe" >/dev/null 2>&1 && [[ -L "$symlink_probe" ]]; then
  symlink_checks_supported=true
fi
rm -f -- "$symlink_probe"
if [[ "$test_platform" == Linux && "$symlink_checks_supported" != true ]]; then
  echo "Linux failed the symbolic-link capability probe" >&2
  exit 1
elif [[ "$symlink_checks_supported" != true ]]; then
  printf 'SKIP: symbolic-link destination checks; %s mount does not expose created links as symlinks.\n' \
    "$test_platform" >&2
fi

for failure_mode in partial-build info-failure; do
  if [[ "$failure_mode" == partial-build ]]; then
    expected_status=41
  else
    expected_status=42
  fi

  absent_target="$work/publication output/new-$failure_mode.deb"
  run_publication_failure "$failure_mode" "$absent_target" "$expected_status"
  [[ ! -e "$absent_target" && ! -L "$absent_target" ]]

  sentinel_target="$work/publication output/sentinel-$failure_mode.deb"
  sentinel_expected="$work/sentinel-$failure_mode.expected"
  printf 'sentinel for %s\n' "$failure_mode" >"$sentinel_expected"
  cp "$sentinel_expected" "$sentinel_target"
  chmod 0600 "$sentinel_target"
  sentinel_mode_before="$(stat -c '%a' "$sentinel_target")"
  if [[ "$check_mode_assertions" == true ]]; then
    [[ "$sentinel_mode_before" == 600 ]]
  fi
  run_publication_failure "$failure_mode" "$sentinel_target" "$expected_status"
  cmp "$sentinel_expected" "$sentinel_target"
  if [[ "$check_mode_assertions" == true ]]; then
    [[ "$(stat -c '%a' "$sentinel_target")" == 600 ]]
  fi
done

special_target="$work/publication output/-synthetic package.deb"
special_expected="$work/synthetic-archive.expected"
printf '%s\n' 'synthetic archive' >"$special_expected"
printf '%s\n' 'existing regular-file sentinel' >"$special_target"
chmod 0600 "$special_target"
(
  umask 077
  PATH="$work/publication-bin:$PATH" TMPDIR="$work/private-tmp" \
    DPKG_DEB_TEST_MODE=success \
    bash scripts/build_deb.sh "$special_target" "0.0.0~dev.1" "$tongd" "$tongctl"
)
[[ -f "$special_target" ]]
cmp "$special_expected" "$special_target"
if [[ "$check_mode_assertions" == true ]]; then
  [[ "$(stat -c '%a' "$special_target")" == 644 ]]
fi
assert_no_package_staging "$(dirname -- "$special_target")" "$work/private-tmp"

# Both the intended parent and its newline-truncated sibling exist: success at
# the requested target alone would not detect staging in the wrong directory.
newline_sibling="$work/publication newline"
newline_parent="$newline_sibling"$'\n\n'
if mkdir -p -- "$newline_parent" 2>"$work/newline-path.stderr" && [[ -d "$newline_parent" ]]; then
  mkdir -p -- "$newline_sibling"
  newline_name=$'-synthetic package.deb\n\n'
  newline_target="$newline_parent/$newline_name"
  PATH="$work/publication-bin:$PATH" TMPDIR="$work/private-tmp" \
    DPKG_DEB_TEST_MODE=success DPKG_DEB_EXPECT_PARENT="$newline_parent" \
    DPKG_DEB_EXPECT_NAME="$newline_name" \
    bash scripts/build_deb.sh "$newline_target" "0.0.0~dev.1" "$tongd" "$tongctl"
  cmp "$special_expected" "$newline_target"
  assert_no_package_staging "$newline_parent" "$work/private-tmp"
  assert_no_package_staging "$newline_sibling" "$work/private-tmp"
elif [[ "$test_platform" == Linux ]]; then
  echo 'Linux failed to create the newline-ending package output parent' >&2
  exit 1
else
  printf 'SKIP: newline-ending output parent; %s mount does not support the fixture.\n' "$test_platform" >&2
fi

if [[ "$publication_mode" == --publication-failures-only ]]; then
  printf 'Package publication failure, cleanup, and path checks passed; %s.\n' "$mode_check_summary"
  exit 0
fi

reject_target_before_build() {
  local target=$1
  local status
  : >"$work/dpkg-deb.calls"
  if PATH="$work/publication-bin:$PATH" TMPDIR="$work/private-tmp" \
    DPKG_DEB_TEST_MODE=success DPKG_DEB_CALL_LOG="$work/dpkg-deb.calls" \
    bash scripts/build_deb.sh "$target" "0.0.0~dev.1" "$tongd" "$tongctl" \
    >"$work/publication.stdout" 2>"$work/publication.stderr"; then
    echo "package builder accepted a non-regular output target: $target" >&2
    exit 1
  else
    status=$?
  fi
  [[ "$status" == 2 ]]
  [[ ! -s "$work/dpkg-deb.calls" ]]
  assert_no_package_staging "$(dirname -- "$target")" "$work/private-tmp"
}

if [[ "$symlink_checks_supported" == true ]]; then
  symlink_referent="$work/publication output/symlink-referent.deb"
  symlink_expected="$work/symlink-referent.expected"
  printf 'symlink referent sentinel\n' >"$symlink_expected"
  cp "$symlink_expected" "$symlink_referent"
  symlink_target="$work/publication output/symlink-target.deb"
  ln -s "$symlink_referent" "$symlink_target"
  reject_target_before_build "$symlink_target"
  cmp "$symlink_expected" "$symlink_referent"
  [[ -L "$symlink_target" ]]

  dangling_target="$work/publication output/dangling-link.deb"
  ln -s "$work/publication output/missing-referent.deb" "$dangling_target"
  reject_target_before_build "$dangling_target"
  [[ -L "$dangling_target" ]]
fi

directory_target="$work/publication output/existing-directory.deb"
mkdir -p "$directory_target"
reject_target_before_build "$directory_target"

fifo_target="$work/publication output/existing-fifo.deb"
mkfifo "$fifo_target"
reject_target_before_build "$fifo_target"

changed_target="$work/publication output/created-during-info.deb"
: >"$work/dpkg-deb.calls"
if PATH="$work/publication-bin:$PATH" TMPDIR="$work/private-tmp" \
  DPKG_DEB_TEST_MODE=destination-directory-during-info \
  DPKG_DEB_CALL_LOG="$work/dpkg-deb.calls" \
  DPKG_DEB_MUTATE_TARGET="$changed_target" \
  bash scripts/build_deb.sh "$changed_target" "0.0.0~dev.1" "$tongd" "$tongctl" \
  >"$work/publication.stdout" 2>"$work/publication.stderr"; then
  echo "package builder published after destination changed to a directory" >&2
  exit 1
else
  changed_status=$?
fi
[[ "$changed_status" == 2 ]]
[[ -d "$changed_target" ]]
[[ "$(wc -l <"$work/dpkg-deb.calls")" -eq 2 ]]
grep -Fxq -- --build "$work/dpkg-deb.calls"
grep -Fxq -- --info "$work/dpkg-deb.calls"
assert_no_package_staging "$(dirname -- "$changed_target")" "$work/private-tmp"

if [[ "$symlink_checks_supported" == true ]]; then
  changed_symlink_target="$work/publication output/created-link-during-info.deb"
  changed_symlink_referent="$work/publication output/link-mutation-sentinel.deb"
  changed_symlink_expected="$work/link-mutation-sentinel.expected"
  printf 'destination symlink sentinel\n' >"$changed_symlink_expected"
  cp "$changed_symlink_expected" "$changed_symlink_referent"
  : >"$work/dpkg-deb.calls"
  if PATH="$work/publication-bin:$PATH" TMPDIR="$work/private-tmp" \
    DPKG_DEB_TEST_MODE=destination-symlink-during-info \
    DPKG_DEB_CALL_LOG="$work/dpkg-deb.calls" \
    DPKG_DEB_MUTATE_TARGET="$changed_symlink_target" \
    DPKG_DEB_MUTATE_REFERENT="$changed_symlink_referent" \
    bash scripts/build_deb.sh "$changed_symlink_target" "0.0.0~dev.1" "$tongd" "$tongctl" \
    >"$work/publication.stdout" 2>"$work/publication.stderr"; then
    echo "package builder published after destination changed to a symbolic link" >&2
    exit 1
  else
    changed_symlink_status=$?
  fi
  [[ "$changed_symlink_status" == 2 ]]
  [[ -L "$changed_symlink_target" ]]
  cmp "$changed_symlink_expected" "$changed_symlink_referent"
  [[ "$(wc -l <"$work/dpkg-deb.calls")" -eq 2 ]]
  grep -Fxq -- --build "$work/dpkg-deb.calls"
  grep -Fxq -- --info "$work/dpkg-deb.calls"
  assert_no_package_staging "$(dirname -- "$changed_symlink_target")" "$work/private-tmp"
fi

if [[ "$publication_mode" == --publication-only ]]; then
  printf 'Package publication failure, destination-type, and cleanup checks passed; %s.\n' "$mode_check_summary"
  exit 0
fi

python3 -m unittest discover -s scripts -p 'test_verify_dev_deb.py' -v

mkdir -p "$out"
bash scripts/build_deb.sh "$out/tong-panel-dev-v1.deb" "0.0.0~dev.1" "$tongd" "$tongctl"
bash scripts/build_deb.sh "$out/tong-panel-dev-v2.deb" "0.0.0~dev.2" "$tongd" "$tongctl"
python3 scripts/verify_dev_deb.py "$out/tong-panel-dev-v1.deb" "0.0.0~dev.1" "$tongd" "$tongctl"
python3 scripts/verify_dev_deb.py "$out/tong-panel-dev-v2.deb" "0.0.0~dev.2" "$tongd" "$tongctl"

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
