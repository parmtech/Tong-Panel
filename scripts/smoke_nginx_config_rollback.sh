#!/usr/bin/env bash
# Disposable-VM check for staged Nginx validation and config rollback behavior.
set -Eeuo pipefail

work="$(mktemp -d /tmp/tong-nginx-rollback.XXXXXX)"
port="$(python3 -c 'import socket; s=socket.socket(); s.bind(("127.0.0.1", 0)); print(s.getsockname()[1]); s.close()')"
active="$work/active.conf"
candidate="$work/candidate.conf"
backup="$work/active.backup"
main="$work/nginx.conf"
validator="$work/validator.conf"

cleanup() {
  sudo nginx -s quit -p "$work/" -c "$main" >/dev/null 2>&1 || true
  sudo rm -rf -- "$work"
}
trap cleanup EXIT

cat > "$main" <<EOF
worker_processes 1;
pid $work/nginx.pid;
error_log $work/error.log;
events { worker_connections 32; }
http { include $active; }
EOF
cat > "$validator" <<EOF
worker_processes 1;
pid $work/validator.pid;
error_log $work/validator-error.log;
events { worker_connections 32; }
http { include $candidate; }
EOF

write_config() {
  local target="$1" marker="$2"
  cat > "$target" <<EOF
server {
    listen 127.0.0.1:$port;
    location / { return 200 "$marker"; }
}
EOF
}

read_marker() {
  curl --silent --show-error --fail "http://127.0.0.1:$port/"
}

wait_for_marker() {
  local expected="$1"
  for _ in {1..20}; do
    if [[ "$(read_marker 2>/dev/null || true)" == "$expected" ]]; then return 0; fi
    sleep 0.25
  done
  echo "Nginx did not serve expected marker: $expected" >&2
  return 1
}

restore_active() {
  mv -f -- "$backup" "$active"
  sudo nginx -t -p "$work/" -c "$main" >/dev/null
}

apply_candidate() {
  local reload_mode="$1"
  # Validate the staged file with an isolated Nginx config before publication.
  if ! sudo nginx -t -p "$work/" -c "$validator" >/dev/null 2>&1; then
    return 1
  fi
  cp --preserve=mode -- "$active" "$backup"
  mv -f -- "$candidate" "$active"
  if ! sudo nginx -t -p "$work/" -c "$main" >/dev/null 2>&1; then
    restore_active
    return 1
  fi
  if [[ "$reload_mode" == fail ]]; then
    restore_active
    return 1
  fi
  if ! sudo nginx -s reload -p "$work/" -c "$main" >/dev/null 2>&1; then
    restore_active
    sudo nginx -s reload -p "$work/" -c "$main" >/dev/null 2>&1 || true
    return 1
  fi
  rm -f -- "$backup"
}

write_config "$active" old-config
sudo nginx -t -p "$work/" -c "$main" >/dev/null
sudo nginx -p "$work/" -c "$main"
wait_for_marker old-config

# Invalid candidate must fail before replacing the live file.
printf '%s\n' 'server { this is invalid nginx syntax; }' > "$candidate"
if apply_candidate success; then
  echo "invalid staged Nginx configuration was accepted" >&2
  exit 1
fi
grep -q 'old-config' "$active"
wait_for_marker old-config

# A valid stage followed by a failed reload must restore the old config.
write_config "$candidate" failed-reload-config
if apply_candidate fail; then
  echo "simulated reload failure unexpectedly succeeded" >&2
  exit 1
fi
grep -q 'old-config' "$active"
wait_for_marker old-config

# A valid candidate must atomically replace the config and reload successfully.
write_config "$candidate" new-config
apply_candidate success
wait_for_marker new-config
grep -q 'new-config' "$active"

printf '%s\n' 'PASS: Nginx staged syntax failure preserves live config; reload failure restores it; valid config applies and reloads'
