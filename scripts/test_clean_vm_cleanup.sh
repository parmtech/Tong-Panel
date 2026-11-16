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
