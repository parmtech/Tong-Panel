"""Fail-closed checks for source-controlled development gate evidence."""
import json
import tempfile
import unittest
from pathlib import Path

from p1_gate_audit import REQUIRED, InvalidGateManifest, audit, unique_object


class P1GateAuditTests(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self.tmp.cleanup)
        self.root = Path(self.tmp.name)
        self.entries = [
            {"id": gate, "status": "blocked", "evidence": evidence, "issue": issue}
            for gate, (evidence, issue) in REQUIRED.items()
        ]
        for path, _ in REQUIRED.values():
            location = self.root / path
            location.parent.mkdir(parents=True, exist_ok=True)
            location.write_text("review proposal, not approved", encoding="utf-8")
        self.manifest = self.root / "docs/P1_GATE_SNAPSHOT.json"

    def write(self, **changes):
        payload = {"schema": 1, "release_candidate": False, "gates": self.entries}
        payload.update(changes)
        self.manifest.write_text(json.dumps(payload), encoding="utf-8")

    def test_known_blockers_are_audited_not_approved(self):
        self.write()
        report = audit(self.root, self.manifest)
        self.assertEqual(report["result"], "blocked")
        self.assertFalse(report["release_candidate"])
        self.assertEqual(report["blocked_count"], len(REQUIRED))

    def test_changed_or_missing_gate_never_passes(self):
        for change in [
            lambda x: x.pop(),
            lambda x: x.append(dict(x[0])),
            lambda x: x[0].update(id="invented"),
            lambda x: x[0].update(status="approved"),
            lambda x: x[0].update(status="complete"),
            lambda x: x[0].update(evidence="../outside"),
            lambda x: x[0].update(issue=True),
        ]:
            with self.subTest(change=change):
                self.entries = [
                    {"id": gate, "status": "blocked", "evidence": source, "issue": issue}
                    for gate, (source, issue) in REQUIRED.items()
                ]
                change(self.entries)
                self.write()
                with self.assertRaises(InvalidGateManifest):
                    audit(self.root, self.manifest)

    def test_release_claim_and_wrong_schema_fail_closed(self):
        for data in [{"release_candidate": True}, {"schema": True}, {"schema": 2}]:
            with self.subTest(data=data):
                self.write(**data)
                with self.assertRaises(InvalidGateManifest):
                    audit(self.root, self.manifest)

    def test_duplicate_json_key_and_malformed_input(self):
        with self.assertRaises(InvalidGateManifest):
            unique_object([("status", "blocked"), ("status", "approved")])
        self.manifest.write_text('{"schema":1,"schema":1}', encoding="utf-8")
        with self.assertRaises(InvalidGateManifest):
            audit(self.root, self.manifest)

    def test_missing_or_external_reference_fails(self):
        self.write()
        (self.root / REQUIRED["p0_adr_review"][0]).unlink()
        with self.assertRaises(InvalidGateManifest):
            audit(self.root, self.manifest)
        with self.assertRaises(InvalidGateManifest):
            audit(self.root, self.root.parent / "foreign.json")

    def test_oversized_manifest_fails(self):
        self.manifest.write_bytes(b" " * 65537)
        with self.assertRaises(InvalidGateManifest):
            audit(self.root, self.manifest)


if __name__ == "__main__":
    unittest.main()
