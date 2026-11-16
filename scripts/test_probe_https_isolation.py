import contextlib
import io
import os
import shutil
import stat
from types import SimpleNamespace
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
                "proc/123/status": "State:\tS (sleeping)\nPPid:\t1\nsecret-process-canary\n",
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
                "https_fixture snapshot=racy_not_readiness",
                "https_fixture nginx_master=state_S nginx_children_R_S_D_Z_T_t_X_I=0,0,0,0,0,0,0,0",
                "https_fixture fpm_master=unavailable fpm_children_R_S_D_Z_T_t_X_I=unavailable",
                "https_fixture fpm_socket_a=missing",
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


    def test_snapshot_process_states_bounds_and_redaction(self):
        script = Path(__file__).with_name("clean_vm_smoke.sh").read_text()
        diagnostic = script.split("<<'PY_HTTPS_DIAGNOSTICS' || true\n", 1)[1].split("\nPY_HTTPS_DIAGNOSTICS\n", 1)[0]
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            def write(name, value):
                path = root / name
                path.parent.mkdir(parents=True, exist_ok=True)
                path.write_text(value)
            def run():
                def fixture_path(*parts):
                    return root / Path(*parts).as_posix().lstrip("/")
                output = io.StringIO()
                with patch("pathlib.Path", fixture_path), contextlib.redirect_stdout(output):
                    namespace = {}
                    exec(compile(diagnostic, "<guest https diagnostics>", "exec"), namespace)
                text = output.getvalue()
                self.assertLessEqual(len(text.encode()), 2048)
                self.assertNotIn("secret", text)
                self.assertNotIn(str(root), text)
                return text, namespace
            write("run/tong/nginx-https-e2e.pid", "123\n")
            write("run/tong/php-fpm-e2e.pid", "124\n")
            write("proc/123/comm", "nginx\n")
            write("proc/124/comm", "php-fpm8.2\n")
            write("proc/123/status", "State: S (sleeping)\nPPid: 1\n")
            write("proc/124/status", "State: S (sleeping)\nPPid: 1\n")
            write("proc/net/tcp", "header\n")
            write("proc/net/tcp6", "header\n")
            for number, state in enumerate("RSDZTtXI", 200):
                write(f"proc/{number}/status", f"State: {state} (fixture)\nPPid: 123\nsecret-canary\n")
            text, ns = run()
            self.assertIn("nginx_children_R_S_D_Z_T_t_X_I=1,1,1,1,1,1,1,1", text)
            self.assertIn("fpm_master=state_S fpm_children_R_S_D_Z_T_t_X_I=0,0,0,0,0,0,0,0", text)
            for state in "RSDZTtXI":
                write("proc/123/status", f"State: {state} (fixture)\nPPid: 1\n")
                text, _ = run()
                self.assertIn(f"nginx_master=state_{state}", text)
            write("proc/123/status", "State: S (sleeping)\nPPid: 1\n")
            write("run/tong/nginx-https-e2e.pid", "999\n")
            text, _ = run()
            self.assertIn("nginx_master=unavailable", text)
            write("run/tong/nginx-https-e2e.pid", "123\n")
            write("proc/124/comm", "secret-canary\n")
            text, _ = run()
            self.assertIn("fpm_master=unavailable", text)
            write("proc/124/comm", "php-fpm8.3\n")
            write("proc/200/status", "State: R\nPPid: secret-canary\n")
            text, _ = run()
            self.assertIn("nginx_children_R_S_D_Z_T_t_X_I=unavailable", text)
            write("proc/200/status", "X" * 65537)
            text, _ = run()
            self.assertIn("nginx_children_R_S_D_Z_T_t_X_I=unavailable", text)
            write("run/tong/nginx-https-e2e.pid", "1" * 33)
            text, _ = run()
            self.assertIn("nginx_master=unavailable", text)
            write("run/tong/nginx-https-e2e.pid", "123\n")
            write("proc/200/status", "State: R (fixture)\nPPid: 123\n")
            for number in range(300, 325):
                write(f"proc/{number}/status", "State: S (fixture)\nPPid: 123\n")
            text, ns = run()
            self.assertIn("nginx_children_R_S_D_Z_T_t_X_I=unavailable", text)
            with patch("pathlib.Path.iterdir", return_value=iter([root / "x"] * 4097)):
                self.assertEqual(ns["child_counts"]("123"), "unavailable")
            with patch("pathlib.Path.open", side_effect=PermissionError("secret-canary")):
                self.assertEqual(ns["master_info"]("/private", {"nginx"}), (None, "unavailable"))
            write("run/tong/php/tenant-a/site-a.sock", "secret-canary")
            text, _ = run()
            self.assertIn("fpm_socket_a=other", text)
            with patch("pathlib.Path.lstat", return_value=SimpleNamespace(st_mode=stat.S_IFSOCK)):
                text, _ = run()
                self.assertIn("fpm_socket_a=socket", text)
            with patch("pathlib.Path.lstat", return_value=SimpleNamespace(st_mode=stat.S_IFLNK)):
                text, _ = run()
                self.assertIn("fpm_socket_a=other", text)
            with patch("pathlib.Path.lstat", side_effect=PermissionError("secret-canary")):
                text, _ = run()
                self.assertIn("fpm_socket_a=unavailable", text)

    def test_failure_branch_keeps_probe_status_and_has_no_retry(self):
        script = Path(__file__).with_name("clean_vm_smoke.sh").read_text()
        branch = script.split("if python3 /home/ci/probe_https_isolation.py; then", 1)[1].split("\nfi\n", 1)[0]
        self.assertIn("probe_status=$?", branch)
        self.assertIn("timeout --kill-after=1s 3s python3", branch)
        self.assertIn("exit \"$probe_status\"", branch)
        self.assertNotIn("curl", branch)
        self.assertNotIn("probe_https_isolation.py", branch)
        self.assertNotIn("tenant-b", branch)
        bash = shutil.which("bash")
        if bash:
            command = ("python3() { printf 'probe-call\\n'; return 17; }\n"
                       "timeout() { printf 'diagnostic-call\\n'; cat >/dev/null; return 124; }\n"
                       "if python3 /home/ci/probe_https_isolation.py; then" + branch + "\nfi\n")
            result = subprocess.run([bash, "-c", command], capture_output=True, text=True, timeout=10)
            self.assertEqual(result.returncode, 17)
            self.assertEqual(result.stdout.splitlines(), ["probe-call", "diagnostic-call"])


if __name__ == "__main__":
    unittest.main()
