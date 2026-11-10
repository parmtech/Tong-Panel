#!/usr/bin/env python3
"""Single-attempt HTTPS marker diagnostics for the disposable two-tenant guest."""

from dataclasses import dataclass
import os
from pathlib import Path
import re
import stat
import subprocess
import sys
import tempfile


EXPECTED = {
    "tenant-a.example.test": b"tenant-a-isolated",
    "tenant-b.example.test": b"tenant-b-isolated",
}
BODY_LIMIT = 4096


@dataclass(frozen=True)
class ProbeResult:
    host: str
    status: str
    transport_exit: int
    body_bytes: int
    classification: str

    @property
    def passed(self):
        return self.classification == "pass"

    def diagnostic(self):
        return (f"https_isolation host={self.host} status={self.status} "
                f"transport_exit={self.transport_exit} bytes={self.body_bytes} "
                f"classification={self.classification}")


def probe(host, runner=subprocess.run):
    """Probe a fixed fixture host once; retain neither body nor raw diagnostics."""
    if host not in EXPECTED:
        raise ValueError("unsupported disposable probe host")
    try:
        return _probe_once(host, runner)
    except OSError:
        # Temporary-file creation or cleanup can fail too; never expose an
        # underlying path/message through a Python traceback in the CI log.
        return ProbeResult(host, "000", -1, 0, "probe_unavailable")


def _probe_once(host, runner):
    status, transport, failure = "000", -1, None
    with tempfile.TemporaryDirectory(prefix="tong-https-isolation-") as directory:
        response = Path(directory) / "response"
        descriptor = os.open(response, os.O_CREAT | os.O_EXCL | os.O_WRONLY, 0o600)
        os.close(descriptor)
        command = [
            "curl", "--disable", "--noproxy", "*", "--silent", "--show-error",
            "--insecure", "--resolve", host + ":443:127.0.0.1",
            "--connect-timeout", "5", "--max-time", "15",
            "--max-filesize", str(BODY_LIMIT), "--output", str(response),
            "--write-out", "%{http_code}", "https://" + host + "/",
        ]
        try:
            result = runner(command, check=False, stdout=subprocess.PIPE,
                            stderr=subprocess.DEVNULL, text=True, timeout=20,
                            env={"PATH": os.defpath, "LC_ALL": "C"})
            if type(result.returncode) is int and -128 <= result.returncode <= 255:
                transport = result.returncode
            raw_status = result.stdout.strip() if isinstance(result.stdout, str) else ""
            if re.fullmatch(r"[1-5][0-9]{2}", raw_status):
                status = raw_status
            elif raw_status != "000":
                failure = "malformed_status"
        except subprocess.TimeoutExpired:
            failure = "process_timeout"
        except (OSError, subprocess.SubprocessError):
            failure = "transport_unavailable"
        try:
            if not stat.S_ISREG(response.lstat().st_mode):
                raise OSError
            with response.open("rb") as stream:
                body = stream.read(BODY_LIMIT + 1)
        except OSError:
            return ProbeResult(host, status, transport, 0, "response_unavailable")
        size = len(body)
        if size > BODY_LIMIT or transport == 63:
            classification = "response_limit"
        elif failure is not None:
            classification = failure
        elif transport == 28:
            classification = "transport_timeout"
        elif transport != 0:
            classification = "transport_failure"
        elif status != "200":
            classification = "unexpected_http_status"
        elif body != EXPECTED[host]:
            classification = "unexpected_body"
        else:
            classification = "pass"
        return ProbeResult(host, status, transport, size, classification)


def main(arguments=None, runner=subprocess.run, output=None):
    if arguments is None:
        arguments = sys.argv[1:]
    if arguments:
        print("usage: probe_https_isolation.py", file=sys.stderr)
        return 2
    if output is None:
        output = sys.stdout
    for host in EXPECTED:
        result = probe(host, runner)
        try:
            print(result.diagnostic(), file=output)
        except OSError:
            return 1
        if not result.passed:
            return 1
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
