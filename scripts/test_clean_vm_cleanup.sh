#!/usr/bin/env bash
set -Eeuo pipefail
script_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
test_root="$(mktemp -d "$script_dir/.cleanup-test.XXXXXX")"
test_root="$(cd -- "$test_root" && pwd)"
cleanup_test() {
  # Verify the absolute target remains inside this test's workspace directory.
  case "$test_root" in
    "$script_dir"/.cleanup-test.*) rm -rf -- "$test_root" ;;
    *) echo 'FAIL: unexpected cleanup test directory' >&2 ;;
  esac
}
trap cleanup_test EXIT
# Extract the actual runner cleanup without starting QEMU or executing setup.
awk '/^cleanup\(\) \{$/ { inside=1 } inside { print } inside && /^\}$/ { exit }' \
  "$script_dir/clean_vm_smoke.sh" > "$test_root/cleanup.sh"
source "$test_root/cleanup.sh"
declare -F cleanup >/dev/null
awk '/^host_fail\(\) \{$/ { inside=1 } inside { print } inside && /^\}$/ { exit }' \
  "$script_dir/clean_vm_smoke.sh" > "$test_root/host-fail.sh"
source "$test_root/host-fail.sh"
declare -F host_fail >/dev/null
awk '/^finish\(\) \{$/ { inside=1 } inside { print } inside && /^\}$/ { exit }' \
  "$script_dir/p2_package_install_vm.sh" > "$test_root/finish.sh"
source "$test_root/finish.sh"
declare -F finish >/dev/null

# These are synthetic secrets, including a long line and control characters.
# Reuse them for every raw-console and service-command-output test.
canaries="$test_root/canaries"
printf '%s\r\n' '-----BEGIN OPENSSH PRIVATE KEY-----' 'synthetic-multiline-key-canary' \
  '-----END OPENSSH PRIVATE KEY-----' 'Authorization: Bearer synthetic-token-canary' \
  'Set-Cookie: session=synthetic-cookie-canary' '::warning::synthetic-command-canary' > "$canaries"
printf '\033[31m\rsynthetic-control-canary\000\007\r\n' >> "$canaries"
awk 'BEGIN { for (i=0; i<1048576; i++) printf "x"; printf "\r\n" }' >> "$canaries"

check_case() {
  local name="$1" expected_status="$2" console_mode="$3"
  local VM_DISTRO=synthetic-test-guest
  local work="$test_root/$name" pidfile serial output="$test_root/$name.output" result
  local vm_stage=source_guest_tests vm_failure_line=42
  [[ "$work" == "$test_root/"* ]] || return 90
  mkdir "$work"
  pidfile="$work/qemu.pid"
  serial="$work/serial.log"
  if [[ "$console_mode" != absent ]]; then
    # Synthetic canaries only. Include complete and truncated key-like text,
    # unrelated credentials and CI command syntax; no console line is trusted.
    printf '%s\r\n' '-----BEGIN OPENSSH PRIVATE KEY-----' 'synthetic-main-key-canary' \
      '-----END OPENSSH PRIVATE KEY-----' 'password=synthetic-password-canary' \
      '::warning::synthetic-console-command-canary' > "$serial"
    printf '%s\r\n' 'synthetic-mid-block-key-canary' '-----END RSA PRIVATE KEY-----' \
      'unlabelled-synthetic-secret-canary' > "$work/restore-serial.log"
    cat "$canaries" >> "$serial"
    cat "$canaries" >> "$work/restore-serial.log"
    if [[ "$console_mode" == restore ]]; then serial="$work/restore-serial.log"; fi
  fi
  set +e
  (
    # cleanup must preserve the failed command's exit status.
    (exit "$expected_status")
    cleanup
  ) > "$output" 2>&1
  result=$?
  set -e
  [[ "$(wc -c < "$output")" -le 128 ]] || { echo 'FAIL: console diagnostic byte bound' >&2; exit 1; }
  [[ "$result" -eq "$expected_status" && ! -e "$work" ]] || {
    echo 'FAIL: cleanup status or directory removal changed' >&2; exit 1;
  }
  if [[ "$expected_status" -eq 0 ]]; then
    [[ ! -s "$output" ]] || { echo 'FAIL: success emitted console data' >&2; exit 1; }
  else
    [[ "$(cat "$output")" == "::error::clean_vm failure_stage=source_guest_tests line=42 status=$expected_status serial_console=withheld" ]] || {
      # Never echo captured output in the failure path, either.
      echo 'FAIL: cleanup output was not fixed metadata' >&2; exit 1;
    }
  fi
  printf 'PASS: %s\r\n' "$name"
}

check_case source-failure 1 source
check_case restore-failure 7 restore
check_case failure-before-console 2 absent
check_case successful-cleanup 0 source

host_fail_test="$test_root/host-fail-test.output"
host_fail_work="$test_root/host-fail-test"
mkdir "$host_fail_work"
set +e
(
  work="$host_fail_work"
  pidfile="$work/qemu.pid"
  serial="$work/serial.log"
  vm_stage=restore_guest_restore
  vm_failure_line=0
  trap cleanup EXIT
  host_fail 7 42
) > "$host_fail_test" 2>&1
host_fail_status=$?
set -e
[[ "$host_fail_status" -eq 7 && ! -e "$host_fail_work" ]] || {
  echo 'FAIL: host failure reporter changed exit status or skipped cleanup' >&2; exit 1;
}
[[ "$(cat "$host_fail_test")" == '::error::clean_vm failure_stage=restore_guest_restore line=42 status=7 serial_console=withheld' ]] || {
  echo 'FAIL: host failure reporter output was not fixed metadata' >&2; exit 1;
}
printf 'PASS: host failure stage/status reporting\r\n'

host_err_trap="$test_root/host-err-trap.sh"
awk '/^trap .*vm_failure_line=.* ERR$/ { print; found=1; exit } END { if (!found) exit 1 }' \
  "$script_dir/clean_vm_smoke.sh" > "$host_err_trap"
host_err_work="$test_root/host-err-trap"
host_err_output="$test_root/host-err-trap.output"
mkdir "$host_err_work"
set +e
(
  set -Eeuo pipefail
  work="$host_err_work"
  pidfile="$work/qemu.pid"
  serial="$work/serial.log"
  vm_stage=source_guest_tests
  vm_failure_line=0
  trap cleanup EXIT
  source "$host_err_trap"
  bash -c 'exit 23'
) > "$host_err_output" 2>&1
host_err_status=$?
set -e
[[ "$host_err_status" -eq 23 && ! -e "$host_err_work" ]] || {
  echo 'FAIL: host ERR trap or cleanup changed the original status' >&2; exit 1;
}
[[ "$(cat "$host_err_output")" =~ ^::error::clean_vm\ failure_stage=source_guest_tests\ line=[0-9]+\ status=23\ serial_console=withheld$ ]] || {
  echo 'FAIL: host ERR trap did not emit fixed failure metadata' >&2; exit 1;
}
printf 'PASS: host ERR trap preserves status and emits bounded metadata\r\n'

cleanup_failure_work="$test_root/cleanup-failure"
cleanup_failure_output="$test_root/cleanup-failure.output"
mkdir "$cleanup_failure_work"
set +e
(
  work="$cleanup_failure_work"
  pidfile="$work/qemu.pid"
  serial="$work/serial.log"
  vm_stage=source_guest_tests
  vm_failure_line=0
  rm() { return 92; }
  trap cleanup EXIT
  exit 7
) > "$cleanup_failure_output" 2>&1
cleanup_failure_status=$?
set -e
[[ "$cleanup_failure_status" -eq 7 && -d "$cleanup_failure_work" ]] || {
  echo 'FAIL: cleanup failure changed the original host status' >&2; exit 1;
}
[[ "$(cat "$cleanup_failure_output")" == '::error::clean_vm failure_stage=source_guest_tests line=0 status=7 serial_console=withheld' ]] || {
  echo 'FAIL: cleanup failure altered fixed diagnostic metadata' >&2; exit 1;
}
command rm -rf -- "$cleanup_failure_work"
printf 'PASS: original status survives cleanup failure\r\n'

cleanup_only_work="$test_root/cleanup-only-failure"
cleanup_only_output="$test_root/cleanup-only-failure.output"
mkdir "$cleanup_only_work"
set +e
(
  work="$cleanup_only_work"
  pidfile="$work/qemu.pid"
  serial="$work/serial.log"
  vm_stage=source_guest_tests
  vm_failure_line=0
  rm() { printf 'sensitive-path=%s\r\n' "$*" >&2; return 92; }
  trap cleanup EXIT
  exit 0
) > "$cleanup_only_output" 2>&1
cleanup_only_status=$?
set -e
[[ "$cleanup_only_status" -eq 92 && -d "$cleanup_only_work" ]] || {
  echo 'FAIL: cleanup-only failure status was not preserved' >&2; exit 1;
}
if grep -Fq -- "$cleanup_only_work" "$cleanup_only_output"; then
  echo 'FAIL: cleanup-only failure exposed its temporary path' >&2; exit 1;
fi
[[ "$(cat "$cleanup_only_output")" =~ ^::error::clean_vm\ failure_stage=cleanup\ line=[0-9]+\ status=92\ serial_console=withheld$ ]] || {
  echo 'FAIL: cleanup-only failure lacked fixed metadata' >&2; exit 1;
}
command rm -rf -- "$cleanup_only_work"
printf 'PASS: cleanup-only failure is classified without a raw path\r\n'


# Exercise cleanup with errexit enabled, as in the runner. Noisy native-command
# failures must not bypass removal, expose paths, or replace an earlier status.
check_cleanup_command_failure() {
  local command_name="$1" initial_status="$2" expected_status="$3"
  local work="$test_root/cleanup-$command_name-$initial_status" output="$test_root/cleanup-$command_name-$initial_status.output" result
  local pidfile="$work/qemu.pid" vm_stage=source_guest_tests vm_failure_line=42
  mkdir "$work"
  printf '12345\n' > "$pidfile"
  set +e
  (
    set -Eeuo pipefail
    if [[ "$command_name" == cat ]]; then
      cat() { printf 'private pid path: %s\r\n' "$*" >&2; return 61; }
    else
      kill() { return 0; }
      sleep() { printf 'private sleep path: %s\r\n' "$work" >&2; return 62; }
    fi
    trap cleanup EXIT
    exit "$initial_status"
  ) > "$output" 2>&1
  result=$?
  set -e
  [[ "$result" -eq "$expected_status" && ! -e "$work" ]] || {
    echo 'FAIL: cleanup command failure changed status or skipped removal' >&2; exit 1;
  }
  if [[ "$initial_status" -eq 0 ]]; then
    [[ "$(cat "$output")" =~ ^::error::clean_vm\ failure_stage=cleanup\ line=[0-9]+\ status=$expected_status\ serial_console=withheld$ ]] || {
      echo 'FAIL: cleanup command failure exposed native diagnostics' >&2; exit 1;
    }
  else
    [[ "$(cat "$output")" == "::error::clean_vm failure_stage=source_guest_tests line=42 status=$initial_status serial_console=withheld" ]] || {
      echo 'FAIL: cleanup command failure replaced original metadata' >&2; exit 1;
    }
  fi
  printf 'PASS: cleanup %s failure with original status %d\r\n' "$command_name" "$initial_status"
}
check_cleanup_command_failure cat 0 61
check_cleanup_command_failure cat 7 7
check_cleanup_command_failure sleep 0 62
check_cleanup_command_failure sleep 7 7

for helper in require_no_plaintext require_ciphertext_size; do
  awk -v helper="$helper" '$0 == helper "() {" { inside=1 } inside { print } inside && /^\}$/ { exit }' \
    "$script_dir/clean_vm_smoke.sh" > "$test_root/$helper.sh"
  source "$test_root/$helper.sh"
  declare -F "$helper" >/dev/null
done
check_validation_failure() {
  local mode="$1" command_status="$2" expected_status="$3" stat_size="${4:-33554433}"
  local work="$test_root/validation-$mode-$command_status" output="$test_root/validation-$mode-$command_status.output" result
  local expected_failure_line=""
  local pidfile="$work/qemu.pid" vm_stage=ciphertext_staging vm_failure_line=0
  if [[ "$mode" == stat ]]; then
    if [[ "$command_status" -ne 0 ]]; then
      expected_failure_line="$(awk '/^  if size=/ { print NR; exit }' "$test_root/require_ciphertext_size.sh")"
    elif [[ "$stat_size" == invalid ]]; then
      expected_failure_line="$(awk '/^    \[\[ "\$size" =~/ { print NR; exit }' "$test_root/require_ciphertext_size.sh")"
    elif [[ "$expected_status" -ne 0 ]]; then
      expected_failure_line="$(awk '/^    \[\[ "\$size" -le/ { print NR; exit }' "$test_root/require_ciphertext_size.sh")"
    fi
  fi
  mkdir "$work"
  set +e
  (
    set -Eeuo pipefail
    trap cleanup EXIT
    if [[ "$mode" == grep ]]; then
      grep() { printf 'private grep path: %s\r\n' "$*" >&2; return "$command_status"; }
      require_no_plaintext 'private fixture' "$work/backup.tpbk"
    else
      stat() {
        printf 'private stat path: %s\r\n' "$*" >&2
        if [[ "$command_status" -eq 0 ]]; then printf '%s\n' "$stat_size"; fi
        return "$command_status"
      }
      require_ciphertext_size "$work/backup.tpbk"
    fi
  ) > "$output" 2>&1
  result=$?
  set -e
  [[ "$result" -eq "$expected_status" && ! -e "$work" ]] || {
    echo 'FAIL: validation command status or removal changed' >&2; exit 1;
  }
  if [[ "$expected_status" -eq 0 ]]; then
    [[ ! -s "$output" ]] || { echo 'FAIL: successful validation emitted native output' >&2; exit 1; }
  elif [[ -n "$expected_failure_line" ]]; then
    [[ "$(cat "$output")" == "::error::clean_vm failure_stage=ciphertext_staging line=$expected_failure_line status=$expected_status serial_console=withheld" ]] || {
      echo 'FAIL: size validation diagnostic identified the wrong source line' >&2; exit 1;
    }
  else
    [[ "$(cat "$output")" =~ ^::error::clean_vm\ failure_stage=ciphertext_staging\ line=[0-9]+\ status=$expected_status\ serial_console=withheld$ ]] || {
      echo 'FAIL: validation error lacked fixed metadata' >&2; exit 1;
    }
  fi
  printf 'PASS: %s status %d validates fail-closed (size=%s)\r\n' "$mode" "$command_status" "$stat_size"
}
check_validation_failure grep 0 1
check_validation_failure grep 1 0
check_validation_failure grep 2 2
check_validation_failure grep 73 73
check_validation_failure stat 71 71
check_validation_failure stat 0 1
check_validation_failure stat 0 0 33554432
check_validation_failure stat 0 0 1
check_validation_failure stat 0 1 invalid

# Run the actual pre-trap prefix with a failing mktemp; its native output is
# untrusted even before a workspace and EXIT handler exist.
awk '/^port=2222$/ { exit } { print }' "$script_dir/clean_vm_smoke.sh" > "$test_root/pretrap.sh"
set +e
(
  mktemp() { printf 'private temp path: %s\r\n' "$*" >&2; return 73; }
  export VM_DISTRO=test VM_IMAGE_URL=test VM_CHECKSUM_ALGORITHM=test VM_IMAGE_CHECKSUM=test
  export VM_EXPECTED_ID=test VM_EXPECTED_VERSION=test VM_RUNNER_TEMP="$test_root/private-runner-path"
  source "$test_root/pretrap.sh"
) > "$test_root/pretrap.output" 2>&1
pretrap_status=$?
set -e
[[ "$pretrap_status" -eq 73 && "$(cat "$test_root/pretrap.output")" =~ ^::error::clean_vm\ failure_stage=runner_preflight\ line=[0-9]+\ status=73\ serial_console=withheld$ ]] || {
  echo 'FAIL: pre-trap mktemp diagnostic or status changed' >&2; exit 1;
}
printf 'PASS: pre-trap mktemp failure emits bounded metadata\r\n'

# Run the actual syntax preflight against malformed generated source, retaining
# bash -n validation while hiding its filename-bearing native diagnostic.
awk '/^extract_guest_scripts "\$0" / || /^bash -n "\$work\/guest-scripts.syntax.sh"/ { print; found++ } END { if (found != 2) exit 1 }' \
  "$script_dir/clean_vm_smoke.sh" > "$test_root/syntax-preflight.sh"
syntax_work="$test_root/private-syntax-path"
mkdir "$syntax_work"
set +e
(
  set -Eeuo pipefail
  work="$syntax_work" pidfile="$work/qemu.pid" vm_stage=runner_preflight vm_failure_line=0
  extract_guest_scripts() { printf 'if then\n'; }
  trap cleanup EXIT
  source "$host_err_trap"
  source "$test_root/syntax-preflight.sh"
) > "$test_root/syntax-preflight.output" 2>&1
syntax_status=$?
set -e
[[ "$syntax_status" -eq 2 && ! -e "$syntax_work" && "$(cat "$test_root/syntax-preflight.output")" =~ ^::error::clean_vm\ failure_stage=runner_preflight\ line=[0-9]+\ status=2\ serial_console=withheld$ ]] || {
  echo 'FAIL: syntax preflight skipped validation or exposed native output' >&2; exit 1;
}
printf 'PASS: generated syntax failure emits bounded metadata\r\n'

check_package_case() {
  local name="$1" expected_status="$2" query_status="$3" expected_state="$4" package_phase="$5" expected_phase="$6"
  local preflight_work="$test_root/$name" preflight_listener_pid="" result output="$test_root/$name.output"
  local query_called="$test_root/$name.query"
  [[ "$preflight_work" == "$test_root/"* ]] || return 90
  mkdir "$preflight_work"
  # Exercise the real finish function, replacing system access with noisy stubs.
  timeout() {
    [[ "$*" == '--kill-after=1s 3s sudo -n systemctl is-active --quiet tong-panel-dev.service' ]] || return 90
    printf 'called\r\n' > "$query_called"
    cat "$canaries"
    cat "$canaries" >&2
    return "$query_status"
  }
  sudo() { cat "$canaries"; return 90; }
  journalctl() { cat "$canaries"; return 90; }
  set +e
  (
    (exit "$expected_status")
    finish
  ) > "$output" 2>&1
  result=$?
  set -e
  unset -f timeout sudo journalctl
  [[ "$result" -eq "$expected_status" && ! -e "$preflight_work" ]] || {
    echo 'FAIL: package cleanup status or directory removal changed' >&2; exit 1;
  }
  [[ "$(wc -c < "$output")" -le 128 ]] || { echo 'FAIL: package diagnostic byte bound' >&2; exit 1; }
  if [[ "$expected_status" -eq 0 ]]; then
    [[ ! -s "$output" && ! -e "$query_called" ]] || {
      echo 'FAIL: successful package cleanup queried service or printed output' >&2; exit 1;
    }
  else
    [[ -e "$query_called" && "$(cat "$output")" == "package_vm failure_status=$expected_status phase=$expected_phase service=$expected_state" ]] || {
      echo 'FAIL: package cleanup output was not fixed metadata' >&2; exit 1;
    }
  fi
  printf 'PASS: %s\r\n' "$name"
}

check_package_case package-active 9 0 active install install
check_package_case package-inactive 9 3 not_active upgrade upgrade
check_package_case package-unknown 9 4 unknown downgrade downgrade
check_package_case package-query-failed 9 1 unavailable preflight preflight
check_package_case package-query-timeout 9 124 unavailable service service
check_package_case package-untrusted-phase 9 0 active "$(tr -d '\000' < "$canaries")" unknown
check_package_case package-success 0 0 active remove remove
