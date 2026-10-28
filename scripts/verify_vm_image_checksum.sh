#!/usr/bin/env bash
set -Eeuo pipefail

if [[ $# -ne 3 ]]; then
  echo "usage: $0 <sha256|sha512> <expected-hex-digest> <image-file>" >&2
  exit 2
fi

algorithm="$1"
expected="$2"
image="$3"
case "$algorithm" in
  sha256) digest_length=64; checksum_command=sha256sum ;;
  sha512) digest_length=128; checksum_command=sha512sum ;;
  *) echo "unsupported image checksum algorithm" >&2; exit 2 ;;
esac
if [[ ! "$expected" =~ ^[0-9a-f]+$ || ${#expected} -ne $digest_length ]]; then
  echo "invalid pinned image checksum" >&2
  exit 2
fi
if [[ ! -f "$image" || -L "$image" ]]; then
  echo "image must be a regular, non-symlink file" >&2
  exit 2
fi

actual="$("$checksum_command" -- "$image" | cut -d ' ' -f 1)"
if [[ "$actual" != "$expected" ]]; then
  echo "downloaded VM image checksum mismatch" >&2
  exit 1
fi
