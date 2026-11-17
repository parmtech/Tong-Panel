#!/usr/bin/env bash
set -Eeuo pipefail
script_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
# This source defines a function only. All SSH and sleeping below are stubs.
source "$script_dir/vm_startup_transport.sh"

check_case() (
  local name="$1" expected_status="$2" expected_calls="$3" expected_pauses="$4"
  shift 4
  local -a responses=("$@") ssh_opts=(-o BatchMode=yes)
  local guest=ci@127.0.0.1 calls=0 pauses=0 result=0
  ssh() {
    [[ "$#" -eq 4 && "$1" == -o && "$2" == BatchMode=yes && "$3" == "$guest" && "$4" == 'cloud-init status --wait' ]] || return 90
    local response="${responses[$calls]:-255}"
    calls=$((calls + 1))
    return "$response"
  }
  sleep() {
    [[ "$#" -eq 1 && "$1" == 5 ]] || return 91
    pauses=$((pauses + 1))
  }
  if retry_guest_ssh 'cloud-init status --wait'; then result=0; else result=$?; fi
  if [[ "$result" -ne "$expected_status" || "$calls" -ne "$expected_calls" || "$pauses" -ne "$expected_pauses" ]]; then
    printf '%s: status=%s calls=%s pauses=%s; expected %s/%s/%s\n' "$name" "$result" "$calls" "$pauses" "$expected_status" "$expected_calls" "$expected_pauses" >&2
    exit 1
  fi
  printf 'PASS: %s\n' "$name"
)

check_case 'ready guest executes once' 0 1 0 0
check_case 'sshd restart recovers on next transport attempt' 0 2 1 255 0
check_case 'permanent transport failure exhausts the bound' 255 12 11 255
# A second invocation would succeed. Retrying would hide the guest's failure.
check_case 'guest failure is not retried into success' 2 1 0 2 0
