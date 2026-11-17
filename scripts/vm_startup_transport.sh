#!/usr/bin/env bash
# Sourced by the disposable VM harness after ssh_opts and guest are set.
# Only SSH transport failures (255) may be retried across cloud-init's sshd
# restart. A guest command's nonzero result must remain a failed check.
retry_guest_ssh() {
  local attempt status
  for attempt in {1..12}; do
    if ssh "${ssh_opts[@]}" "$guest" "$@"; then
      return 0
    else
      status=$?
    fi
    if [[ "$status" -ne 255 ]]; then
      echo "Guest SSH command failed (status=$status); preserving the failure" >&2
      return "$status"
    fi
    if [[ "$attempt" -eq 12 ]]; then
      echo "SSH transport did not recover after cloud-init startup" >&2
      return "$status"
    fi
    echo "SSH transport attempt $attempt interrupted while cloud-init may restart sshd; retrying" >&2
    sleep 5
  done
}
