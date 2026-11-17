#!/usr/bin/env bash
set -Eeuo pipefail
helper="$(dirname "$0")/verify_vm_image_checksum.sh"
work="$(mktemp -d)"
trap 'rm -rf "$work"' EXIT
image="$work/image"
printf 'pinned image fixture\n' > "$image"

for algorithm in sha256 sha512; do
  case "$algorithm" in
    sha256) expected="$(sha256sum -- "$image" | cut -d ' ' -f 1)" ;;
    sha512) expected="$(sha512sum -- "$image" | cut -d ' ' -f 1)" ;;
  esac
  bash "$helper" "$algorithm" "$expected" "$image"
  bad_expected="0${expected:1}"
  [[ "$bad_expected" != "$expected" ]] || bad_expected="1${expected:1}"
  if bash "$helper" "$algorithm" "$bad_expected" "$image" >/dev/null 2>&1; then
    echo "$algorithm accepted a mismatched digest" >&2
    exit 1
  fi
  if bash "$helper" "$algorithm" "${expected%?}" "$image" >/dev/null 2>&1; then
    echo "$algorithm accepted a malformed digest" >&2
    exit 1
  fi
  printf 'tampered image\n' >> "$image"
  if bash "$helper" "$algorithm" "$expected" "$image" >/dev/null 2>&1; then
    echo "$algorithm accepted a tampered image" >&2
    exit 1
  fi
  printf 'pinned image fixture\n' > "$image"
done

valid_sha256="$(sha256sum -- "$image" | cut -d ' ' -f 1)"
if bash "$helper" sha256 "$valid_sha256" "$work/missing" >/dev/null 2>&1; then
  echo "accepted a missing image" >&2
  exit 1
fi
echo "VM image checksum verifier accepts valid SHA-256/SHA-512 and rejects mismatches, malformed digests, tampering and missing files"
