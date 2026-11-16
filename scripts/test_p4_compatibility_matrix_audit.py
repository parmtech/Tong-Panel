"""Synthetic-only tests for the draft P4 compatibility matrix auditor."""

from __future__ import annotations

import contextlib
import io
import json
import tempfile
import unittest
from pathlib import Path

from p4_compatibility_matrix_audit import (
    DuplicateJSONKey,
    MAX_MATRIX_BYTES,
    _read_matrix,
    _unique_object,
    audit_matrix,
    main,
    profile_fingerprint,
)


SYNTHETIC_COMMIT = "a" * 40


class P4CompatibilityMatrixAuditTests(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self.tmp.cleanup)
        self.root = Path(self.tmp.name)
        self.write_file("docs/phases/P0-PLANNING.md", "synthetic gate reference\n")
        self.write_file("docs/phases/P4-DESIGN-ACCEPTANCE.md", "synthetic case reference\n")
        self.write_file("artifacts/case-1.txt", "synthetic evidence one\n")
        self.write_file("artifacts/case-10.txt", "synthetic evidence ten\n")
        self.matrix = self.valid_matrix()

    def write_file(self, relative_path: str, contents: str):
        path = self.root / relative_path
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_text(contents, encoding="utf-8")
        return path

    def valid_matrix(self):
        profile = {
            "id": "synthetic-web-ubuntu-fixture",
            "os": "Synthetic Ubuntu 24.04",
            "architecture": "synthetic-amd64",
            "components": {"fixture-web": "fixture-version-1"},
        }
        fingerprint = profile_fingerprint(profile)
        return {
            "schema_version": 1,
            "candidate_commit": SYNTHETIC_COMMIT,
            "profiles": [profile],
            "rows": [
                {
                    "capability": "synthetic-web-capability",
                    "profile_id": profile["id"],
                    "status": "supported",
                    "available": True,
                    "prerequisite_gates": [
                        {"id": "synthetic-p0", "reference": "docs/phases/P0-PLANNING.md"}
                    ],
                    "acceptance_case_ids": ["1", "10"],
                    "tested_commit": SYNTHETIC_COMMIT,
                    "evidence": [
                        {
                            "case_id": "1",
                            "tested_commit": SYNTHETIC_COMMIT,
                            "profile_id": profile["id"],
                            "profile_fingerprint": fingerprint,
                            "result": "passed",
                            "artifact": "artifacts/case-1.txt",
                        },
                        {
                            "case_id": "10",
                            "tested_commit": SYNTHETIC_COMMIT,
                            "profile_id": profile["id"],
                            "profile_fingerprint": fingerprint,
                            "result": "passed",
                            "artifact": "artifacts/case-10.txt",
                        },
                    ],
                    "known_limits": ["synthetic fixture only"],
                    "exclusions": ["synthetic fixture only"],
                    "reviewer_decision": {
                        "disposition": "accepted",
                        "reviewer": "synthetic-fixture-reviewer",
                    },
                }
            ],
        }

    def assert_error(self, expected: str, matrix=None, expected_commit=SYNTHETIC_COMMIT):
        errors = audit_matrix(self.matrix if matrix is None else matrix, self.root, expected_commit)
        self.assertIn(expected, errors, errors)

    def test_valid_synthetic_supported_fixture_is_only_structurally_valid(self):
        self.assertEqual([], audit_matrix(self.matrix, self.root, SYNTHETIC_COMMIT))

    def test_unverified_profile_can_be_explicitly_unavailable_without_evidence(self):
        row = self.matrix["rows"][0]
        row.update(
            status="unverified",
            available=False,
            tested_commit=None,
            evidence=[],
            reviewer_decision={"disposition": "pending", "reviewer": ""},
        )
        self.assertEqual([], audit_matrix(self.matrix, self.root, SYNTHETIC_COMMIT))

    def test_missing_or_unpinned_profiles_fail_closed(self):
        self.matrix["profiles"] = []
        self.assert_error("profiles_missing:root")

        self.matrix = self.valid_matrix()
        self.matrix["profiles"][0]["components"]["fixture-web"] = "TBD"
        self.assert_error("profile_component_unpinned:synthetic-web-ubuntu-fixture")

        for unpinned in ("10.*", ">=10.5", "10.5 - 10.9", "10.5 || 10.9", "MariaDB latest"):
            with self.subTest(version=unpinned):
                self.matrix = self.valid_matrix()
                self.matrix["profiles"][0]["components"]["fixture-web"] = unpinned
                self.assert_error("profile_component_unpinned:synthetic-web-ubuntu-fixture")

    def test_alternative_os_and_component_versions_fail_closed(self):
        for os_value in (
            "Ubuntu 24.04 or Debian 12",
            "Ubuntu 24.04 and Debian 12",
            "Ubuntu 24.04, Debian 12",
            "Ubuntu 24.04 | Debian 12",
            "Ubuntu 24.04 / Debian 12",
        ):
            with self.subTest(os=os_value):
                self.matrix = self.valid_matrix()
                self.matrix["profiles"][0]["os"] = os_value
                self.assert_error("profile_os_unpinned:synthetic-web-ubuntu-fixture")

        for version in (
            "nginx 1.26 or 1.27",
            "nginx 1.26 and 1.27",
            "nginx 1.26, 1.27",
            "1.26 || 1.27",
            "1.26 / 1.27",
        ):
            with self.subTest(version=version):
                self.matrix = self.valid_matrix()
                self.matrix["profiles"][0]["components"]["fixture-web"] = version
                self.assert_error("profile_component_unpinned:synthetic-web-ubuntu-fixture")

    def test_legitimate_single_version_formats_remain_pinned(self):
        for version in ("1:1.26.3-1ubuntu1.2", "1.2~rc1", "v1.2.3+build.7"):
            with self.subTest(version=version):
                self.matrix = self.valid_matrix()
                profile = self.matrix["profiles"][0]
                profile["components"]["fixture-web"] = version
                fingerprint = profile_fingerprint(profile)
                for evidence in self.matrix["rows"][0]["evidence"]:
                    evidence["profile_fingerprint"] = fingerprint
                self.assertEqual([], audit_matrix(self.matrix, self.root, SYNTHETIC_COMMIT))

    def test_duplicate_profile_id_and_duplicate_profile_identity_are_ambiguous(self):
        self.matrix["profiles"].append(dict(self.matrix["profiles"][0]))
        self.assert_error("profile_id_ambiguous:synthetic-web-ubuntu-fixture")

        self.matrix = self.valid_matrix()
        duplicate = dict(self.matrix["profiles"][0])
        duplicate["id"] = "other-synthetic-profile-id"
        self.matrix["profiles"].append(duplicate)
        self.assert_error(
            "profile_identity_ambiguous:synthetic-web-ubuntu-fixture,other-synthetic-profile-id"
        )

    def test_duplicate_capability_profile_row_is_ambiguous(self):
        self.matrix["rows"].append(dict(self.matrix["rows"][0]))
        self.assert_error("capability_profile_ambiguous:rows[1]")

    def test_unresolved_profile_and_gate_references_fail(self):
        self.matrix["rows"][0]["profile_id"] = "missing-profile"
        self.assert_error("profile_reference_unresolved:rows[0]")

        self.matrix = self.valid_matrix()
        self.matrix["rows"][0]["prerequisite_gates"][0]["reference"] = "docs/missing.md"
        self.assert_error("reference_unresolved:rows[0].prerequisite_gates[0].reference")

    def test_stale_source_and_case_evidence_fail(self):
        self.assert_error("candidate_commit_stale:root", expected_commit="b" * 40)

        evidence = self.matrix["rows"][0]["evidence"][0]
        evidence["tested_commit"] = "b" * 40
        self.assert_error("evidence_stale:rows[0].evidence[0]")

    def test_profile_change_invalidates_bound_evidence(self):
        self.matrix["profiles"][0]["components"]["fixture-web"] = "fixture-version-2"
        self.assert_error("evidence_profile_stale:rows[0].evidence[0]")

    def test_wrong_profile_reference_in_evidence_is_rejected(self):
        self.matrix["rows"][0]["evidence"][0]["profile_id"] = "other-profile"
        self.assert_error("evidence_profile_mismatch:rows[0].evidence[0]")

    def test_unresolved_artifact_and_path_escape_fail(self):
        self.matrix["rows"][0]["evidence"][0]["artifact"] = "artifacts/missing.txt"
        self.assert_error("reference_unresolved:rows[0].evidence[0].artifact")

        self.matrix = self.valid_matrix()
        self.matrix["rows"][0]["evidence"][0]["artifact"] = "../outside.txt"
        self.assert_error("reference_outside_root:rows[0].evidence[0].artifact")

    def test_unknown_duplicate_and_uncovered_acceptance_cases_fail(self):
        self.matrix["rows"][0]["acceptance_case_ids"].append("11")
        self.assert_error("acceptance_case_unresolved:rows[0].acceptance_case_ids[2]")

        self.matrix = self.valid_matrix()
        self.matrix["rows"][0]["acceptance_case_ids"].append("1")
        self.assert_error("acceptance_case_ambiguous:rows[0].acceptance_case_ids[2]")

        self.matrix = self.valid_matrix()
        self.matrix["rows"][0]["evidence"].pop()
        self.assert_error("supported_evidence_incomplete:rows[0]")

    def test_supported_row_rejects_failed_or_stale_row_commit(self):
        self.matrix["rows"][0]["evidence"][0]["result"] = "blocked"
        self.assert_error("supported_evidence_not_passed:rows[0].evidence[0]")

        self.matrix = self.valid_matrix()
        self.matrix["rows"][0]["tested_commit"] = "b" * 40
        self.assert_error("supported_tested_commit_stale:rows[0]")

    def test_non_supported_statuses_cannot_be_available(self):
        for status in ("unsupported", "experimental", "unverified"):
            with self.subTest(status=status):
                self.matrix = self.valid_matrix()
                self.matrix["rows"][0]["status"] = status
                self.matrix["rows"][0]["available"] = True
                self.assert_error("non_supported_marked_available:rows[0]")

    def test_duplicate_json_keys_are_rejected(self):
        with self.assertRaises(DuplicateJSONKey):
            json.loads('{"schema_version":1,"schema_version":1}', object_pairs_hook=_unique_object)

    def test_oversized_matrix_is_bounded(self):
        oversized = self.root / "oversized.json"
        oversized.write_bytes(b" " * (MAX_MATRIX_BYTES + 1))
        with self.assertRaisesRegex(ValueError, "size limit"):
            _read_matrix(oversized)

    def test_malformed_types_and_unknown_fields_fail_closed(self):
        self.matrix["schema_version"] = 1.0
        self.matrix["rows"][0]["status"] = []
        self.matrix["rows"][0]["unexpected"] = "ignored-by-no-one"
        self.matrix["rows"][0]["reviewer_decision"]["disposition"] = []
        self.matrix["rows"][0]["evidence"][0]["result"] = []
        errors = audit_matrix(self.matrix, self.root, SYNTHETIC_COMMIT)
        self.assertIn("status_invalid:rows[0]", errors)
        self.assertIn("schema_version_unsupported:root", errors)
        self.assertIn("field_unknown:rows[0].unexpected", errors)
        self.assertIn("reviewer_decision_missing:rows[0]", errors)
        self.assertIn("evidence_result_invalid:rows[0].evidence[0]", errors)

    def test_cli_always_states_that_product_support_is_not_approved(self):
        matrix_path = self.write_file("synthetic-matrix.json", json.dumps(self.matrix))
        output = io.StringIO()
        with contextlib.redirect_stdout(output):
            exit_code = main(
                [
                    matrix_path.name,
                    "--root",
                    str(self.root),
                    "--expected-source-commit",
                    SYNTHETIC_COMMIT,
                ]
            )
        self.assertEqual(0, exit_code)
        result = json.loads(output.getvalue())
        self.assertTrue(result["structurally_valid"])
        self.assertFalse(result["product_support_approval"])
        self.assertEqual("draft structural consistency only", result["tool_scope"])


if __name__ == "__main__":
    unittest.main()
