import contextlib
import io
import os
from pathlib import Path
import subprocess
import tempfile
import unittest
from unittest.mock import patch

from probe_https_isolation import BODY_LIMIT, EXPECTED, TIMING_NAMES, WRITE_OUT, main, probe


class HTTPSIsolationProbeTests(unittest.TestCase):
    def runner(self, body=b"tenant-a-isolated", status="200", code=0, error=None, timings="0.001000|0.020000|0.021000|0.022000"):
        self.calls = []
        self.responses = []

        def run(command, **options):
            self.calls.append((command, options))
            path = Path(command[command.index("--output") + 1])
            self.responses.append(path)
            self.assertTrue(path.exists())
            if os.name != "nt":
                self.assertEqual(path.stat().st_mode & 0o777, 0o600)
                self.assertEqual(path.parent.stat().st_mode & 0o777, 0o700)
            path.write_bytes(body)
            if error is not None:
                raise error
            return subprocess.CompletedProcess(command, code, stdout=status + "|" + timings,
                                               stderr="secret-error-canary")

        return run

    def test_success_has_fixed_bounded_curl_arguments_and_private_cleanup(self):
        result = probe("tenant-a.example.test", self.runner())
        self.assertTrue(result.passed)
        self.assertEqual(len(self.calls), 1)
        command, options = self.calls[0]
        self.assertEqual(command[:4], ["curl", "--disable", "--noproxy", "*"])
        self.assertEqual(command[-1], "https://tenant-a.example.test/")
        self.assertEqual(command[command.index("--resolve") + 1], "tenant-a.example.test:443:127.0.0.1")
        for flag, value in [("--connect-timeout", "5"), ("--max-time", "15"), ("--max-filesize", "4096")]:
            self.assertEqual(command[command.index(flag) + 1], value)
        self.assertIn("--insecure", command)
        self.assertNotIn("--location", command)
        self.assertNotIn("-L", command)
        self.assertFalse(any("retry" in argument for argument in command))
        self.assertEqual(options["timeout"], 20)
        self.assertEqual(options["stderr"], subprocess.DEVNULL)
        self.assertEqual(options["env"], {"PATH": os.defpath, "LC_ALL": "C"})
        self.assertFalse(options["check"])
        self.assertFalse(self.responses[0].exists())
        self.assertFalse(self.responses[0].parent.exists())
        self.assertEqual(command[command.index("--write-out") + 1], WRITE_OUT)
        for name, value in zip(TIMING_NAMES, result.timings):
            self.assertIn(f"{name}={value}", result.diagnostic())

    def test_timeout_preserves_sanitized_timing_phases(self):
        result = probe("tenant-a.example.test", self.runner(
            status="000", code=28, body=b"", timings="0.001000|0.000000|0.000000|5.002000"))
        self.assertEqual(result.classification, "transport_timeout")
        self.assertIn("time_connect=0.001000", result.diagnostic())
        self.assertIn("time_appconnect=0.000000", result.diagnostic())
        self.assertIn("time_total=5.002000", result.diagnostic())
        self.assertEqual(len(self.calls), 1)

    def test_malformed_timing_metadata_is_redacted_and_fails_closed(self):
        for timings in ["secret-canary", "nan|0.000000|0.000000|1.000000",
                        "0.000000|0.000000|0.000000|1000.000000",
                        "0.000000|0.000000|0.000000|1.000000\nsecret-canary"]:
            with self.subTest(timings=timings):
                result = probe("tenant-a.example.test", self.runner(timings=timings))
                self.assertFalse(result.passed)
                self.assertEqual(result.classification, "malformed_metadata")
                self.assertEqual(result.timings, ())
                self.assertNotIn("secret", result.diagnostic())
                self.assertEqual(len(self.calls), 1)

    def test_http_error_and_redirect_are_single_attempt_failures(self):
        for status in ["502", "503", "504", "302", "403"]:
            with self.subTest(status=status):
                result = probe("tenant-a.example.test", self.runner(body=b"error-body-canary", status=status))
                self.assertEqual(result.classification, "unexpected_http_status")
                self.assertEqual(result.status, status)
                self.assertEqual(len(self.calls), 1)
                self.assertNotIn("error-body-canary", result.diagnostic())

    def test_wrong_200_body_never_passes_or_retries(self):
        for body in [b"tenant-a-leaked", b"tenant-b-isolated", b"prefix tenant-a-isolated", b"tenant-a-isolated\n", b"secret-body-canary"]:
            with self.subTest(body=body):
                result = probe("tenant-a.example.test", self.runner(body=body))
                self.assertEqual(result.classification, "unexpected_body")
                self.assertEqual(len(self.calls), 1)
                self.assertNotIn(body.decode(), result.diagnostic())

    def test_both_timeout_kinds_and_launch_error_are_redacted_and_cleaned(self):
        for error in [subprocess.TimeoutExpired("secret-command-canary", 20, output="secret-output-canary"), OSError("secret-path-canary")]:
            with self.subTest(error=type(error).__name__):
                result = probe("tenant-a.example.test", self.runner(error=error))
                self.assertFalse(result.passed)
                self.assertNotIn("secret", result.diagnostic())
                self.assertEqual(len(self.calls), 1)
                self.assertFalse(self.responses[0].parent.exists())
        result = probe("tenant-a.example.test", self.runner(status="000", code=28, body=b""))
        self.assertEqual(result.classification, "transport_timeout")
        self.assertEqual(result.transport_exit, 28)

    def test_oversize_is_bounded_even_when_injected_runner_ignores_curl_limit(self):
        result = probe("tenant-a.example.test", self.runner(body=b"x" * (BODY_LIMIT + 1000)))
        self.assertEqual(result.classification, "response_limit")
        self.assertEqual(result.body_bytes, BODY_LIMIT + 1)
        result = probe("tenant-a.example.test", self.runner(body=b"", code=63))
        self.assertEqual(result.classification, "response_limit")
        self.assertFalse(self.responses[0].parent.exists())

    def test_status_and_transport_diagnostics_cannot_inject_text(self):
        for status in ["200\nsecret-status-canary", "999", "20", "", "secret-status-canary"]:
            with self.subTest(status=status):
                result = probe("tenant-a.example.test", self.runner(status=status))
                self.assertEqual(result.status, "000")
                self.assertEqual(result.classification, "malformed_status")
                self.assertNotIn("secret", result.diagnostic())
        result = probe("tenant-a.example.test", self.runner(code="secret-exit-canary"))
        self.assertEqual(result.transport_exit, -1)
        self.assertFalse(result.passed)

    def test_main_probes_each_fixed_host_once_and_stops_on_first_failure(self):
        calls = []

        def valid_runner(command, **options):
            calls.append(command)
            host = command[-1].split("/")[2]
            Path(command[command.index("--output") + 1]).write_bytes(EXPECTED[host])
            return subprocess.CompletedProcess(command, 0, stdout="200|0.001000|0.020000|0.021000|0.022000")

        output = io.StringIO()
        self.assertEqual(main([], valid_runner, output), 0)
        self.assertEqual(len(calls), 2)
        self.assertEqual(len(output.getvalue().splitlines()), 2)
        output = io.StringIO()
        self.assertEqual(main([], self.runner(body=b"secret-canary"), output), 1)
        self.assertEqual(len(self.calls), 1)
        self.assertNotIn("secret-canary", output.getvalue())

    def test_arguments_and_unknown_hosts_are_rejected_without_network(self):
        def never_run(*args, **options):
            self.fail("unexpected network attempt")

        error = io.StringIO()
        with contextlib.redirect_stderr(error):
            self.assertEqual(main(["secret-argument-canary"], never_run), 2)
        self.assertNotIn("secret-argument-canary", error.getvalue())
        with self.assertRaises(ValueError):
            probe("arbitrary.example.test", never_run)

    def test_temporary_file_failure_is_static_without_exception_message(self):
        with patch("probe_https_isolation.tempfile.TemporaryDirectory", side_effect=OSError("secret-directory-canary")):
            result = probe("tenant-a.example.test", self.runner())
        self.assertEqual(result.classification, "probe_unavailable")
        self.assertEqual(len(self.calls), 0)
        self.assertNotIn("secret-directory-canary", result.diagnostic())

    def test_guest_failure_snapshot_prints_only_fixed_metadata(self):
        script = Path(__file__).with_name("clean_vm_smoke.sh").read_text()
        diagnostic = script.split("<<'PY_HTTPS_DIAGNOSTICS' || true\n", 1)[1].split("\nPY_HTTPS_DIAGNOSTICS\n", 1)[0]
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            fixtures = {
                "run/tong/nginx-https-e2e.pid": "123\n",
                "proc/123/comm": "nginx\n",
                "proc/123/status": "State:\tS (sleeping)\nsecret-process-canary\n",
                "proc/net/tcp": "header\n0: 0100007F:01BB 00000000:0000 0A secret-socket-canary\n",
                "proc/net/tcp6": "header\n",
            }
            for name, value in fixtures.items():
                path = root / name
                path.parent.mkdir(parents=True, exist_ok=True)
                path.write_text(value)

            def fixture_path(*parts):
                return root / Path(*parts).as_posix().lstrip("/")

            output = io.StringIO()
            with patch("pathlib.Path", fixture_path), contextlib.redirect_stdout(output):
                exec(compile(diagnostic, "<guest https diagnostics>", "exec"), {})
            self.assertEqual(output.getvalue().splitlines(), [
                "https_fixture master=state_S",
                "https_fixture family=tcp listen_80=0 listen_443=1",
                "https_fixture family=tcp6 listen_80=0 listen_443=0",
            ])
            # An unavailable proc snapshot remains static; no path or exception text.
            (root / "proc/net/tcp").unlink()
            output = io.StringIO()
            with patch("pathlib.Path", fixture_path), contextlib.redirect_stdout(output):
                exec(compile(diagnostic, "<guest https diagnostics>", "exec"), {})
            self.assertIn("family=tcp listeners=unavailable", output.getvalue())
            self.assertNotIn("secret", output.getvalue())


if __name__ == "__main__":
    unittest.main()
