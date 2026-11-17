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
TIMING_NAMES = ("time_connect", "time_appconnect", "time_starttransfer", "time_total")
WRITE_OUT = "%{http_code}" + "".join("|%{" + name + "}" for name in TIMING_NAMES)
PHP_FPM_COMM_NAMES = frozenset({"php-fpm", "php-fpm8.2", "php-fpm8.3"})
PHP_FPM_PID_FILE = "/run/tong/php-fpm-e2e.pid"
PROC_ROOT = "/proc"


@dataclass(frozen=True)
class ProbeResult:
    host: str
    status: str
    transport_exit: int
    body_bytes: int
    classification: str
    timings: tuple = ()

    @property
    def passed(self):
        return self.classification == "pass"

    def diagnostic(self):
        phases = "".join(f" {name}={value}" for name, value in zip(TIMING_NAMES, self.timings))
        return (f"https_isolation host={self.host} status={self.status} "
                f"transport_exit={self.transport_exit} bytes={self.body_bytes} "
                f"classification={self.classification}{phases}")


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


def php_fpm_master_identity(pid_file=None, proc_root=None):
    """Return only an allowlisted comm name and a fixed failure classification."""
    if pid_file is None:
        pid_file = PHP_FPM_PID_FILE
    if proc_root is None:
        proc_root = PROC_ROOT
    try:
        with Path(pid_file).open("rb") as source:
            pid_bytes = source.read(33)
        if len(pid_bytes) > 32:
            return None, "pid_invalid"
        pid = pid_bytes.decode("ascii", "strict").strip()
        if not re.fullmatch(r"[1-9][0-9]{0,9}", pid):
            return None, "pid_invalid"
    except PermissionError:
        return None, "pid_permission_denied"
    except OSError:
        return None, "pid_unavailable"
    except (ValueError, UnicodeError):
        return None, "pid_invalid"
    try:
        with (Path(proc_root) / pid / "comm").open("rb") as source:
            comm_bytes = source.read(65)
        if len(comm_bytes) > 64:
            return None, "comm_invalid"
        comm = comm_bytes.decode("ascii", "strict").removesuffix("\n")
        return (comm, "pass") if comm in PHP_FPM_COMM_NAMES else (None, "comm_not_allowed")
    except PermissionError:
        return None, "comm_permission_denied"
    except OSError:
        return None, "comm_unavailable"
    except (ValueError, UnicodeError):
        return None, "comm_invalid"


def _probe_once(host, runner):
    status, transport, failure = "000", -1, None
    timings = ()
    with tempfile.TemporaryDirectory(prefix="tong-https-isolation-") as directory:
        response = Path(directory) / "response"
        descriptor = os.open(response, os.O_CREAT | os.O_EXCL | os.O_WRONLY, 0o600)
        os.close(descriptor)
        command = [
            "curl", "--disable", "--noproxy", "*", "--silent", "--show-error",
            "--insecure", "--resolve", host + ":443:127.0.0.1",
            "--connect-timeout", "5", "--max-time", "15",
            "--max-filesize", str(BODY_LIMIT), "--output", str(response),
            "--write-out", WRITE_OUT, "https://" + host + "/",
        ]
        try:
            result = runner(command, check=False, stdout=subprocess.PIPE,
                            stderr=subprocess.DEVNULL, text=True, timeout=20,
                            env={"PATH": os.defpath, "LC_ALL": "C"})
            if type(result.returncode) is int and -128 <= result.returncode <= 255:
                transport = result.returncode
            raw_output = result.stdout.strip() if isinstance(result.stdout, str) else ""
            fields = raw_output.split("|")
            raw_status = fields[0]
            # Only fixed-width numeric curl timings may enter the public log.
            # Malformed timing metadata must never turn a failed probe into a pass.
            if len(fields) == 5 and all(re.fullmatch(r"[0-9]{1,3}\.[0-9]{6}", value) for value in fields[1:]):
                timings = tuple(fields[1:])
            else:
                failure = "malformed_metadata"
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
            return ProbeResult(host, status, transport, 0, "response_unavailable", timings)
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
        return ProbeResult(host, status, transport, size, classification, timings)


def main(arguments=None, runner=subprocess.run, output=None):
    if arguments is None:
        arguments = sys.argv[1:]
    if arguments == ["--check-fpm-master"]:
        if output is None:
            output = sys.stdout
        comm, classification = php_fpm_master_identity()
        try:
            print("https_fixture fpm_master_comm=" + (comm or "unavailable") +
                  " classification=" + classification, file=output)
        except OSError:
            return 1
        return 0 if comm is not None else 1
    if arguments:
        print("usage: probe_https_isolation.py [--check-fpm-master]", file=sys.stderr)
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
