"""Structural checks for the draft P0 threat crosswalk, not threat acceptance."""
import os
import sys
import tempfile
import unittest
from pathlib import Path

from p0_threat_crosswalk_audit import InvalidCrosswalk, audit_crosswalk


class P0ThreatCrosswalkAuditTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.parent = Path(self.temp.name)
        self.root = self.parent / "repo"
        self.docs = self.root / "docs"
        self.internal = self.root / "internal"
        self.docs.mkdir(parents=True)
        self.internal.mkdir()
        (self.internal / "source.go").write_text("package fixture\n", encoding="utf-8")
        (self.internal / "source_test.go").write_text(
            "package fixture\nfunc TestPresent(t interface{ Helper() }) {}\n",
            encoding="utf-8",
        )
        (self.docs / "guide.md").write_text("structural reference\n", encoding="utf-8")
        self.crosswalk = self.docs / "P0_THREAT_TEST_CROSSWALK.md"

    def row(self, number, evidence=None):
        evidence = evidence or (
            "[source.go](../internal/source.go); "
            "[source_test.go](../internal/source_test.go): `TestPresent`; "
            "[guide](guide.md#section)"
        )
        return f"| **T{number:02d} fixture threat** | {evidence} | source-present | no mitigation claim | [issue](https://example.test) |"

    def write_crosswalk(self, rows=None):
        rows = rows if rows is not None else [self.row(number) for number in range(1, 16)]
        content = "\n".join([
            "# Draft crosswalk",
            "## T01-T15 inventory",
            "| Threat | Source/test references | Evidence | Limits | Tracker |",
            "| --- | --- | --- | --- | --- |",
            *rows,
            "## Commands for later evidence collection",
            "No tests are run by this inventory.",
        ])
        self.crosswalk.write_text(content, encoding="utf-8")

    def test_valid_references_report_structure_only(self):
        self.write_crosswalk()
        report = audit_crosswalk(self.root, self.crosswalk)
        self.assertEqual(report["result"], "structural_only")
        self.assertEqual(report["threat_rows"], 15)
        self.assertEqual(report["local_file_references"], 45)
        self.assertEqual(report["test_symbols_present"], 15)
        self.assertEqual(report["symbol_check"], "structural_text_signature_only")
        self.assertIn("not a language parser", report["notice"])
        self.assertIn("not executed", report["notice"])
        self.assertIn("no mitigation or approval", report["notice"])

    def test_invalid_root_or_crosswalk_path_value_error_is_normalized(self):
        self.write_crosswalk()
        invalid_root = Path(f"{self.root}\x00")
        invalid_crosswalk = Path(f"{self.crosswalk}\x00")
        with self.assertRaises(InvalidCrosswalk):
            audit_crosswalk(invalid_root, self.crosswalk)
        with self.assertRaises(InvalidCrosswalk):
            audit_crosswalk(self.root, invalid_crosswalk)

    def test_missing_or_outside_local_reference_fails(self):
        self.write_crosswalk()
        self.crosswalk.write_text(
            self.crosswalk.read_text(encoding="utf-8").replace(
                "../internal/source.go", "../internal/missing.go", 1
            ),
            encoding="utf-8",
        )
        with self.assertRaises(InvalidCrosswalk):
            audit_crosswalk(self.root, self.crosswalk)

        outside = self.parent / "outside.go"
        outside.write_text("package fixture\n", encoding="utf-8")
        self.write_crosswalk()
        self.crosswalk.write_text(
            self.crosswalk.read_text(encoding="utf-8").replace(
                "../internal/source.go", "../../outside.go", 1
            ),
            encoding="utf-8",
        )
        with self.assertRaises(InvalidCrosswalk):
            audit_crosswalk(self.root, self.crosswalk)

    def test_symlinked_local_reference_cannot_escape_repository(self):
        self.write_crosswalk()
        outside = self.parent / "outside.go"
        outside.write_text("package outside\n", encoding="utf-8")
        escaped_link = self.docs / "escape.go"

        if not hasattr(os, "symlink"):
            if sys.platform.startswith("linux"):
                self.fail("Linux CI must support symlink creation")
            self.skipTest("symlink creation is unsupported on this platform")

        try:
            os.symlink(outside, escaped_link)
        except NotImplementedError as exc:
            if sys.platform.startswith("linux"):
                raise
            self.skipTest(f"symlink creation is unsupported on this platform: {exc}")
        except OSError as exc:
            if os.name == "nt" and getattr(exc, "winerror", None) == 1314:
                self.skipTest(f"Windows symlink privilege is unavailable: {exc}")
            raise

        self.assertTrue(escaped_link.is_symlink())
        self.assertFalse(escaped_link.resolve().is_relative_to(self.root.resolve()))
        self.crosswalk.write_text(
            self.crosswalk.read_text(encoding="utf-8").replace(
                "../internal/source.go", "escape.go", 1
            ),
            encoding="utf-8",
        )
        with self.assertRaises(InvalidCrosswalk):
            audit_crosswalk(self.root, self.crosswalk)

    def test_embedded_nul_local_path_is_reported_as_invalid_crosswalk(self):
        self.write_crosswalk()
        self.crosswalk.write_text(
            self.crosswalk.read_text(encoding="utf-8").replace(
                "../internal/source.go", "../internal/\x00source.go", 1
            ),
            encoding="utf-8",
        )
        with self.assertRaises(InvalidCrosswalk):
            audit_crosswalk(self.root, self.crosswalk)

    def test_common_spec_files_are_recognized_as_mapped_test_files(self):
        for extension in (".spec.js", ".spec.mjs", ".spec.ts", ".spec.tsx"):
            with self.subTest(extension=extension):
                self.write_crosswalk()
                test_path = self.internal / f"source{extension}"
                test_path.write_text('test("TestPresent", () => {});\n', encoding="utf-8")
                self.crosswalk.write_text(
                    self.crosswalk.read_text(encoding="utf-8").replace(
                        "[source_test.go](../internal/source_test.go): `TestPresent`;",
                        f"[source{extension}](../internal/source{extension}): `TestPresent`;",
                        1,
                    ),
                    encoding="utf-8",
                )
                report = audit_crosswalk(self.root, self.crosswalk)
                self.assertEqual(report["test_symbols_present"], 15)

    def test_uppercase_spec_mjs_is_recognized_and_symbol_checked(self):
        self.write_crosswalk()
        test_path = self.internal / "source.SPEC.MJS"
        test_path.write_text('test("TestPresent", () => {});\n', encoding="utf-8")
        self.crosswalk.write_text(
            self.crosswalk.read_text(encoding="utf-8").replace(
                "[source_test.go](../internal/source_test.go): `TestPresent`;",
                "[source.SPEC.MJS](../internal/source.SPEC.MJS): `TestPresent`;",
                1,
            ),
            encoding="utf-8",
        )
        report = audit_crosswalk(self.root, self.crosswalk)
        self.assertEqual(report["test_symbols_present"], 15)

    def test_missing_symbol_and_missing_test_file_mapping_fail(self):
        self.write_crosswalk()
        self.crosswalk.write_text(
            self.crosswalk.read_text(encoding="utf-8").replace("TestPresent", "TestMissing", 1),
            encoding="utf-8",
        )
        with self.assertRaises(InvalidCrosswalk):
            audit_crosswalk(self.root, self.crosswalk)

        self.write_crosswalk()
        self.crosswalk.write_text(
            self.crosswalk.read_text(encoding="utf-8").replace(
                "[source_test.go](../internal/source_test.go): ", "[guide](guide.md): ", 1
            ),
            encoding="utf-8",
        )
        with self.assertRaises(InvalidCrosswalk):
            audit_crosswalk(self.root, self.crosswalk)

    def test_unbackticked_test_symbol_is_checked(self):
        self.write_crosswalk()
        self.crosswalk.write_text(
            self.crosswalk.read_text(encoding="utf-8").replace("`TestPresent`", "TestPresent", 1),
            encoding="utf-8",
        )
        report = audit_crosswalk(self.root, self.crosswalk)
        self.assertEqual(report["test_symbols_present"], 15)

    def test_threat_rows_must_be_unique_and_complete(self):
        duplicate = [self.row(number) for number in range(1, 16)]
        duplicate[1] = self.row(1)
        self.write_crosswalk(duplicate)
        with self.assertRaises(InvalidCrosswalk):
            audit_crosswalk(self.root, self.crosswalk)

        self.write_crosswalk([self.row(number) for number in range(1, 15)])
        with self.assertRaises(InvalidCrosswalk):
            audit_crosswalk(self.root, self.crosswalk)


if __name__ == "__main__":
    unittest.main()
