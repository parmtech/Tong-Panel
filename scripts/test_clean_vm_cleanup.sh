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
awk '/^finish\(\) \{$/ { inside=1 } inside { print } inside && /^\}$/ { exit }' \
  "$script_dir/p2_package_install_vm.sh" > "$test_root/finish.sh"
source "$test_root/finish.sh"
declare -F finish >/dev/null

# These are synthetic secrets, including a long line and control characters.
# Reuse them for every raw-console and service-command-output test.
canaries="$test_root/canaries"
printf '%s\n' '-----BEGIN OPENSSH PRIVATE KEY-----' 'synthetic-multiline-key-canary' \
  '-----END OPENSSH PRIVATE KEY-----' 'Authorization: Bearer synthetic-token-canary' \
  'Set-Cookie: session=synthetic-cookie-canary' '::warning::synthetic-command-canary' > "$canaries"
printf '\033[31m\rsynthetic-control-canary\000\007\n' >> "$canaries"
awk 'BEGIN { for (i=0; i<1048576; i++) printf "x"; printf "\n" }' >> "$canaries"

check_case() {
  local name="$1" expected_status="$2" console_mode="$3"
  local VM_DISTRO=synthetic-test-guest
  local work="$test_root/$name" pidfile serial output="$test_root/$name.output" result
  [[ "$work" == "$test_root/"* ]] || return 90
  mkdir "$work"
  pidfile="$work/qemu.pid"
  serial="$work/serial.log"
  if [[ "$console_mode" != absent ]]; then
    # Synthetic canaries only. Include complete and truncated key-like text,
    # unrelated credentials and CI command syntax; no console line is trusted.
    printf '%s\n' '-----BEGIN OPENSSH PRIVATE KEY-----' 'synthetic-main-key-canary' \
      '-----END OPENSSH PRIVATE KEY-----' 'password=synthetic-password-canary' \
      '::warning::synthetic-console-command-canary' > "$serial"
    printf '%s\n' 'synthetic-mid-block-key-canary' '-----END RSA PRIVATE KEY-----' \
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
    [[ "$(cat "$output")" == "clean_vm failure_status=$expected_status serial_console=withheld" ]] || {
      # Never echo captured output in the failure path, either.
      echo 'FAIL: cleanup output was not fixed metadata' >&2; exit 1;
    }
  fi
  printf 'PASS: %s\n' "$name"
}

check_case source-failure 1 source
check_case restore-failure 7 restore
check_case failure-before-console 2 absent
check_case successful-cleanup 0 source

check_package_case() {
  local name="$1" expected_status="$2" query_status="$3" expected_state="$4" package_phase="$5" expected_phase="$6"
  local preflight_work="$test_root/$name" preflight_listener_pid="" result output="$test_root/$name.output"
  local query_called="$test_root/$name.query"
  [[ "$preflight_work" == "$test_root/"* ]] || return 90
  mkdir "$preflight_work"
  # Exercise the real finish function, replacing system access with noisy stubs.
  timeout() {
    [[ "$*" == '--kill-after=1s 3s sudo -n systemctl is-active --quiet tong-panel-dev.service' ]] || return 90
    printf 'called\n' > "$query_called"
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
  printf 'PASS: %s\n' "$name"
}

check_package_case package-active 9 0 active install install
check_package_case package-inactive 9 3 not_active upgrade upgrade
check_package_case package-unknown 9 4 unknown downgrade downgrade
check_package_case package-query-failed 9 1 unavailable preflight preflight
check_package_case package-query-timeout 9 124 unavailable service service
check_package_case package-untrusted-phase 9 0 active "$(tr -d '\000' < "$canaries")" unknown
check_package_case package-success 0 0 active remove remove
