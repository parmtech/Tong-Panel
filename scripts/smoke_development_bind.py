#!/usr/bin/env python3
"""Exercise a real tongd binary's development-only loopback bind configuration.

Usage: python3 scripts/smoke_development_bind.py /path/to/tongd
Uses disposable local state, no credentials from the environment, and no proxy.
"""

import json
import os
from pathlib import Path
import socket
import subprocess
import sys
import tempfile
import time
from urllib.error import HTTPError, URLError
from urllib.request import ProxyHandler, Request, build_opener


HTTP = build_opener(ProxyHandler({}))
TOKEN = "development-bind-smoke-fixture-token-0123456789"


def require(condition, message):
    if not condition:
        raise RuntimeError(message)


def environment(root, address, sqlite=False):
    result = {key: value for key, value in os.environ.items()
              if not key.upper().startswith("TONG_")}
    result.update(TONG_DATA_DIR=str(root / "journal"), TONG_DEV_LISTEN_ADDR=address)
    if sqlite:
        result.update(TONG_DEV_SQLITE_DB=str(root / "sqlite" / "api.db"),
                      TONG_DEV_TOKEN=TOKEN)
    return result


def stop(process):
    if process.poll() is None:
        process.terminate()
        try:
            process.wait(timeout=7)
        except subprocess.TimeoutExpired:
            process.kill()
            process.wait(timeout=5)
            raise RuntimeError("development daemon failed to stop within bound")


def expect_failure(binary, env):
    process = subprocess.Popen([binary], env=env, stdout=subprocess.DEVNULL,
                               stderr=subprocess.DEVNULL)
    try:
        try:
            result = process.wait(timeout=5)
        except subprocess.TimeoutExpired:
            raise RuntimeError("rejected configuration unexpectedly kept daemon running")
        require(result != 0, "rejected configuration unexpectedly exited successfully")
    finally:
        stop(process)


def request(port, path):
    req = Request(f"http://127.0.0.1:{port}" + path,
                  headers={"Accept": "application/json"})
    try:
        response = HTTP.open(req, timeout=1)
    except HTTPError as error:
        response = error
    with response:
        raw = response.read(65537)
        require(len(raw) <= 65536, "development HTTP response exceeded bound")
        return response.code, json.loads(raw)


def healthy_custom_port(binary, root, port):
    process = subprocess.Popen([binary], env=environment(root, f"127.0.0.1:{port}"),
                               stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
    try:
        deadline = time.monotonic() + 10
        while True:
            require(process.poll() is None, "custom-port daemon exited before readiness")
            try:
                status, body = request(port, "/healthz")
            except (OSError, URLError):
                require(time.monotonic() < deadline, "custom-port readiness timed out")
                time.sleep(0.05)
                continue
            require(status == 200 and body == {"status": "ok"},
                    "custom-port health response mismatch")
            break
        status, body = request(port, "/api/v1/me")
        require(status == 503 and body == {"error": {
            "code": "auth_unconfigured", "message": "Service Unavailable"}},
            "custom-port daemon did not fail closed without authentication")
        require(process.poll() is None, "custom-port daemon exited during checks")
    finally:
        stop(process)
    require(process.returncode == 0, "custom-port daemon failed graceful shutdown")


def main(arguments):
    require(len(arguments) == 1, "usage: smoke_development_bind.py /path/to/tongd")
    binary = str(Path(arguments[0]).resolve(strict=True))
    with tempfile.TemporaryDirectory(prefix="tong-bind-smoke-") as temporary:
        root = Path(temporary)
        # Hold an OS-selected port until launch preparation is complete. Release
        # once; a racing listener makes this run fail, never retry on another port.
        with socket.socket() as reserved:
            reserved.bind(("127.0.0.1", 0))
            port = reserved.getsockname()[1]
            require(port != 8090, "OS selected the default port; refusing ambiguous smoke")
        healthy_custom_port(binary, root / "custom", port)

        unsafe = ["0.0.0.0:8090", ":8090", "localhost:8090", "192.0.2.1:8090",
                  "[::1]:8090", "127.0.0.1:http",
                  "127.0.0.1:0", "127.0.0.1:65536", "127.0.0.1:-1",
                  " 127.0.0.1:8090", "127.0.0.1:8090 "]
        for index, address in enumerate(unsafe):
            for sqlite in (False, True):
                state = root / f"invalid-{index}-{int(sqlite)}"
                state.mkdir()
                expect_failure(binary, environment(state, address, sqlite))
                require(not list(state.rglob("*")),
                        "invalid bind configuration created persistent state")

        # An actual listener keeps the custom port occupied throughout launch.
        # No HTTP probes are sent to that socket or any unrelated daemon.
        with socket.socket() as occupied:
            occupied.bind(("127.0.0.1", 0))
            occupied.listen(1)
            port = occupied.getsockname()[1]
            require(port != 8090, "OS selected default port for collision check")
            expect_failure(binary, environment(root / "occupied", f"127.0.0.1:{port}"))
    print("Development custom loopback bind, unsafe-input and occupied-port smoke passed")


if __name__ == "__main__":
    try:
        main(sys.argv[1:])
    except (RuntimeError, OSError, ValueError) as error:
        print("Development bind smoke failed: " + str(error), file=sys.stderr)
        sys.exit(1)
