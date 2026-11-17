import ast
import contextlib
import io
import os
import re
import shutil
import stat
from types import SimpleNamespace
from pathlib import Path
import subprocess
import tempfile
import unittest
from unittest.mock import patch

from probe_https_isolation import BODY_LIMIT, EXPECTED, PHP_FPM_COMM_NAMES, PHP_FPM_PID_FILE, TIMING_NAMES, WRITE_OUT, main, probe


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

    def test_fpm_master_check_is_bounded_allowlisted_and_redacted(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            pid_file = root / "run/tong/php-fpm-e2e.pid"
            proc_root = root / "proc"
            pid_file.parent.mkdir(parents=True)
            process = proc_root / "124"
            process.mkdir(parents=True)
            comm_file = process / "comm"

            with patch("probe_https_isolation.PHP_FPM_PID_FILE", str(pid_file)), \
                 patch("probe_https_isolation.PROC_ROOT", str(proc_root)):
                for comm in sorted(PHP_FPM_COMM_NAMES):
                    with self.subTest(comm=comm):
                        pid_file.write_bytes(b"124\n")
                        comm_file.write_bytes((comm + "\n").encode("ascii"))
                        output = io.StringIO()
                        self.assertEqual(main(["--check-fpm-master"], output=output), 0)
                        self.assertEqual(output.getvalue(), f"https_fixture fpm_master_comm={comm} classification=pass\n")

                for pid_contents, comm_contents, classification in [
                    (b"124\n", b"php-fpm-secret-canary\n", "comm_not_allowed"),
                    (b"124\nsecret-pid-canary", b"php-fpm8.2\n", "pid_invalid"),
                    (b"1" * 33, b"php-fpm8.2\n", "pid_invalid"),
                    (b"124\n", b"x" * 65, "comm_invalid"),
                    (b"\xff", b"php-fpm8.2\n", "pid_invalid"),
                    (b"124\n", b"\xff", "comm_invalid"),
                ]:
                    with self.subTest(pid_contents=pid_contents[:12], comm_contents=comm_contents[:12]):
                        pid_file.write_bytes(pid_contents)
                        comm_file.write_bytes(comm_contents)
                        output = io.StringIO()
                        self.assertEqual(main(["--check-fpm-master"], output=output), 1)
                        self.assertEqual(output.getvalue(), "https_fixture fpm_master_comm=unavailable classification=" + classification + "\n")
                        self.assertNotIn("secret", output.getvalue())

    def test_fpm_read_failures_are_classified_without_exception_text(self):
        for error, reason in [(PermissionError("secret-path-canary"), "permission_denied"),
                              (FileNotFoundError("secret-path-canary"), "unavailable")]:
            for stage in ("pid", "comm"):
                with self.subTest(error=type(error).__name__, stage=stage):
                    pid_stream = io.BytesIO(b"124\n")
                    effects = [error] if stage == "pid" else [pid_stream, error]
                    output = io.StringIO()
                    with patch("pathlib.Path.open", side_effect=effects):
                        self.assertEqual(main(["--check-fpm-master"], output=output), 1)
                    self.assertEqual(output.getvalue(),
                                     "https_fixture fpm_master_comm=unavailable classification=" + stage + "_" + reason + "\n")
                    self.assertNotIn("secret", output.getvalue())

    def test_fpm_success_check_is_wired_before_https_probe_and_shares_allowlist(self):
        script = Path(__file__).with_name("clean_vm_smoke.sh").read_text()
        command = "sudo python3 -I /home/ci/probe_https_isolation.py --check-fpm-master"
        probe_command = "if python3 /home/ci/probe_https_isolation.py; then"
        self.assertEqual(script.count(command), 1)
        self.assertLess(script.index("test -S /run/tong/php/tenant-b/site-b.sock"), script.index(command))
        self.assertLess(script.index(command), script.index(probe_command))

        diagnostic = script.split("<<'PY_HTTPS_DIAGNOSTICS' || true\n", 1)[1].split("\nPY_HTTPS_DIAGNOSTICS\n", 1)[0]
        match = re.search(r'fpm_pid, fpm_state = master_info\("([^"]+)", (\{[^}]+\})\)', diagnostic)
        self.assertIsNotNone(match, "failure snapshot must keep an explicit FPM comm allowlist")
        self.assertEqual(match.group(1), PHP_FPM_PID_FILE)
        self.assertEqual(ast.literal_eval(match.group(2)), set(PHP_FPM_COMM_NAMES))

    def test_guest_err_trap_withholds_heredoc_source_and_preserves_status(self):
        bash = shutil.which("bash")
        if not bash:
            self.skipTest("Bash is required for the guest ERR trap regression")
        script = Path(__file__).with_name("clean_vm_smoke.sh").read_text()
        guest = script.split("<<'GUEST'\n", 1)[1].split("\nGUEST\n", 1)[0]
        self.assertNotIn("$BASH_COMMAND", guest)
        reporter = re.search(r"(?ms)^report_guest_failure\(\) \{\n.*?^\}\n", guest)
        self.assertIsNotNone(reporter)
        trap = next(line for line in guest.splitlines() if line.startswith("trap ") and line.endswith(" ERR"))
        for failure in ["(exit 23) <<'EOF'\nliteral-secret-canary\nEOF\n",
                        "printf '%s' 'literal-secret-canary' | (exit 23)\n"]:
            with self.subTest(failure=failure):
                command = "set -Eeuo pipefail\nsmoke_stage=fpm_identity\n" + reporter.group(0) + trap + "\n" + failure
                result = subprocess.run([bash, "-c", command], capture_output=True, text=True, timeout=10)
                self.assertEqual(result.returncode, 23)
                self.assertEqual(result.stdout, "")
                self.assertRegex(result.stderr, r"\A::error::clean_guest failure_stage=fpm_identity line=[0-9]+ status=23\n\Z")
                self.assertNotIn("secret", result.stderr)

    def test_source_and_restore_guest_heredocs_pass_bash_syntax_check(self):
        bash = shutil.which("bash")
        if not bash:
            self.skipTest("Bash is required for nested guest syntax checks")
        script = Path(__file__).with_name("clean_vm_smoke.sh").read_text()
        guests = {
            "source": script.split("<<'GUEST'\n", 1)[1].split("\nGUEST\n", 1)[0],
            "restore": script.split("<<'RESTORE'\n", 1)[1].split("\nRESTORE\n", 1)[0],
        }
        for name, guest in guests.items():
            with self.subTest(guest=name):
                result = subprocess.run([bash, "-n"], input=guest, capture_output=True, text=True, timeout=10)
                self.assertEqual(result.returncode, 0, result.stderr)

    def test_restore_guest_err_trap_and_expected_denial_report_safely(self):
        bash = shutil.which("bash")
        if not bash:
            self.skipTest("Bash is required for the restore guest ERR trap regression")
        script = Path(__file__).with_name("clean_vm_smoke.sh").read_text()
        restore = script.split("<<'RESTORE'\n", 1)[1].split("\nRESTORE\n", 1)[0]
        reporter = re.search(r"(?ms)^report_restore_failure\(\) \{\n.*?^\}\n", restore)
        self.assertIsNotNone(reporter)
        failer = re.search(r"(?ms)^fail_restore\(\) \{\n.*?^\}\n", restore)
        self.assertIsNotNone(failer)
        trap = next(line for line in restore.splitlines() if line.startswith("trap ") and line.endswith(" ERR"))
        command = ("set -Eeuo pipefail\nrestore_stage=restore_tenant_denial\n" + reporter.group(0) + failer.group(0) +
                   trap + "\n(exit 23)\n")
        result = subprocess.run([bash, "-c", command], capture_output=True, text=True, timeout=10)
        self.assertEqual(result.returncode, 23)
        self.assertEqual(result.stdout, "")
        self.assertRegex(result.stderr, r"\A::error::clean_restore_guest failure_stage=restore_tenant_denial line=[0-9]+ status=23\n\Z")
        self.assertNotIn("secret", result.stderr)

        denial = re.search(r"(?ms)^if \./tongbackup decrypt --tenant tenant-b.*?^fi\n", restore)
        self.assertIsNotNone(denial)
        self.assertEqual(denial.group(0).count('fail_restore 1 "$LINENO"'), 1)
        artifact_assertion = re.search(r"(?m)^\[\[ ! -e /home/ci/should-not-exist\.sql \]\]$", restore)
        self.assertIsNotNone(artifact_assertion)
        with tempfile.TemporaryDirectory() as directory:
            stub = Path(directory) / "tongbackup"
            output_file = Path(directory) / "should-not-exist.sql"
            (Path(directory) / "tenant-a.backup.tpbk").write_text("synthetic ciphertext fixture\n")
            denial_command = denial.group(0).replace("/home/ci/tenant-a.backup.tpbk", "tenant-a.backup.tpbk")
            artifact_check = artifact_assertion.group(0).replace("/home/ci/should-not-exist.sql", "should-not-exist.sql")
            cases = [
                (23, False, 0, None),  # Expected tenant denial, with no output file.
                (0, False, 1, "accepted the wrong tenant"),  # Unexpected acceptance.
                (23, True, 1, "failure_stage=restore_tenant_denial"),  # Rejection left an artifact.
            ]
            for decrypt_status, leaves_output, expected_status, expected_report in cases:
                output_file.unlink(missing_ok=True)
                stub_source = "#!/usr/bin/env bash\n"
                if leaves_output:
                    stub_source += "printf '%s\\n' 'synthetic-secret-canary' > should-not-exist.sql\n"
                stub_source += f"exit {decrypt_status}\n"
                stub.write_text(stub_source)
                stub.chmod(0o700)
                command = ("set -Eeuo pipefail\nrestore_stage=restore_tenant_denial\n" + reporter.group(0) +
                           failer.group(0) + trap + "\n" + denial_command + "\n" + artifact_check +
                           "\nprintf 'branch-completed\\n'\n")
                result = subprocess.run([bash, "-c", command], cwd=directory, capture_output=True, text=True, timeout=10)
                self.assertEqual(result.returncode, expected_status)
                self.assertNotIn("secret", result.stderr)
                self.assertNotIn("should-not-exist.sql", result.stderr)
                if expected_report is None:
                    self.assertEqual(result.stdout, "branch-completed\n")
                    self.assertEqual(result.stderr, "")
                    self.assertFalse(output_file.exists())
                elif decrypt_status == 0:
                    self.assertEqual(result.stdout, "")
                    self.assertIn(expected_report, result.stderr)
                    self.assertRegex(result.stderr, r"\Afresh-host restore accepted the wrong tenant\n::error::clean_restore_guest failure_stage=restore_tenant_denial line=[0-9]+ status=1\n\Z")
                    self.assertFalse(output_file.exists())
                else:
                    self.assertEqual(result.stdout, "")
                    self.assertRegex(result.stderr, r"\A::error::clean_restore_guest failure_stage=restore_tenant_denial line=[0-9]+ status=1\n\Z")
                    self.assertTrue(output_file.exists())

    def test_guest_phase_stages_precede_operations_in_order(self):
        script = Path(__file__).with_name("clean_vm_smoke.sh").read_text()
        guest = script.split("<<'GUEST'\n", 1)[1].split("\nGUEST\n", 1)[0]
        self.assertLess(script.index("vm_stage=source_guest_setup"), script.index("vm_stage=source_guest_tests"))
        self.assertLess(script.index("vm_stage=source_guest_setup"), script.index('retry_guest_ssh "cloud-init status --wait"'))
        self.assertLess(script.index("vm_stage=restore_guest_boot"), script.index("vm_stage=restore_guest_setup"))
        self.assertLess(script.index("vm_stage=restore_guest_setup"), script.index("vm_stage=restore_guest_restore"))
        restore_setup = script.index("vm_stage=restore_guest_setup")
        restore_execution = script.index("vm_stage=restore_guest_restore")
        self.assertIn('retry_guest_ssh "cloud-init status --wait"', script[restore_setup:restore_execution])
        self.assertIn("printf '%s\\n' 'tenant-a-challenge'", guest)
        self.assertIn("printf '%s\\n' 'tenant-b-challenge'", guest)
        self.assertNotIn("printf '%s\\r\\n' 'tenant-a-challenge'", guest)
        self.assertNotIn("printf '%s\\r\\n' 'tenant-b-challenge'", guest)
        phases = [
            ("https_isolation", "if python3 /home/ci/probe_https_isolation.py; then"),
            ("acme_challenge_runtime", '[[ "$(curl --silent --show-error --resolve tenant-a.example.test:80:127.0.0.1'),
            ("hidden_path", 'hidden_status="$(curl --silent --show-error --insecure --resolve tenant-a.example.test:443:127.0.0.1'),
            ("install_preflight", "if /home/ci/tong-p1-dev/bin/tongctl preflight "),
            ("mariadb_isolation", "bash ./tong-p1-dev/tests/mariadb-isolation.sh"),
            ("encrypted_backup_restore", "bash ./tong-p1-dev/tests/encrypted-backup-restore.sh"),
            ("wordpress_backup_restore", "sudo env TONG_DISPOSABLE_WORDPRESS_RESTORE=1 bash ./smoke_wordpress_backup_source.sh"),
            ("p1_smoke", "python3 /home/ci/smoke_p1.py"),
            ("doctor", "/home/ci/tong-p1-dev/bin/tongctl doctor > doctor.json"),
            ("package_lifecycle", "bash ./p2_package_install_vm.sh"),
        ]
        marker_positions = []
        for stage, operation in phases:
            with self.subTest(stage=stage):
                marker = f"smoke_stage={stage}\n"
                self.assertEqual(guest.count(marker), 1)
                marker_position = guest.index(marker)
                self.assertTrue(guest.startswith(operation, marker_position + len(marker)))
                marker_positions.append(marker_position)
        self.assertEqual(marker_positions, sorted(marker_positions))

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
        self.assertIn("fail_guest \"$probe_status\" \"$LINENO\"", branch)
        self.assertNotIn("curl", branch)
        self.assertNotIn("probe_https_isolation.py", branch)
        self.assertNotIn("tenant-b", branch)
        bash = shutil.which("bash")
        if bash:
            guest = script.split("<<'GUEST'\n", 1)[1].split("\nGUEST\n", 1)[0]
            reporter = re.search(r"(?ms)^report_guest_failure\(\) \{\n.*?^\}\n", guest)
            failer = re.search(r"(?ms)^fail_guest\(\) \{\n.*?^\}\n", guest)
            self.assertIsNotNone(reporter)
            self.assertIsNotNone(failer)
            command = ("set -Eeuo pipefail\nsmoke_stage=https_isolation\n" + reporter.group(0) + failer.group(0) +
                       "python3() { printf 'probe-call\\n'; return 17; }\n"
                       "timeout() { printf 'diagnostic-call\\n'; cat >/dev/null; return 124; }\n"
                       "if python3 /home/ci/probe_https_isolation.py; then" + branch + "\nfi\n")
            result = subprocess.run([bash, "-c", command], capture_output=True, text=True, timeout=10)
            self.assertEqual(result.returncode, 17)
            self.assertEqual(result.stdout.splitlines(), ["probe-call", "diagnostic-call"])
            self.assertRegex(result.stderr, r"\AHTTPS isolation probe failed; fixture diagnostics are metadata only\n::error::clean_guest failure_stage=https_isolation line=[0-9]+ status=17\n\Z")


if __name__ == "__main__":
    unittest.main()
