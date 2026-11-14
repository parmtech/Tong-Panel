import contextlib
import io
import os
from pathlib import Path
import subprocess
import unittest
from unittest.mock import patch

from probe_https_isolation import BODY_LIMIT, EXPECTED, main, probe


class HTTPSIsolationProbeTests(unittest.TestCase):
    def runner(self, body=b"tenant-a-isolated", status="200", code=0, error=None):
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
            return subprocess.CompletedProcess(command, code, stdout=status,
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
            return subprocess.CompletedProcess(command, 0, stdout="200")

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


if __name__ == "__main__":
    unittest.main()
