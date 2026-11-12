#!/usr/bin/env python3
"""Fail-closed *blocked-state* audit; never an approval or release signer.

This only proves the source-controlled P1 blockers have not been silently lost.
A real production go/no-go requires separate externally reviewed evidence.
"""
from __future__ import annotations

import argparse
import json
import sys
from pathlib import Path

REQUIRED = {
    "p0_adr_review": ("docs/ADR_REGISTER.md", 1),
    "independent_security_review": ("docs/THREAT_MODEL.md", 2),
    "production_authentication": ("docs/ADR-009-AUTHENTICATION.md", 6),
    "privileged_agent_boundary": ("docs/SECURITY.md", 8),
    "certificate_key_custody": ("docs/P1_CERTIFICATE_OPERATION_CONTRACT.md", 1),
    "browser_terminal_t0": ("docs/P1_BROWSER_TERMINAL_PROPOSAL.md", 1),
    "signed_installer_recovery": ("docs/INSTALLER_SPEC.md", 11),
    "two_tenant_restore_acceptance": ("docs/P2_TWO_TENANT_END_TO_END_ACCEPTANCE.md", 157),
    "accessibility_acceptance": ("docs/UX_UI_SPEC.md", 4),
}


class InvalidGateManifest(ValueError):
    """Gate data is ambiguous, incomplete, or falsely implies approval."""


def unique_object(pairs: list[tuple[str, object]]) -> dict[str, object]:
    result: dict[str, object] = {}
    for key, value in pairs:
        if key in result:
            raise InvalidGateManifest(f"duplicate JSON field: {key}")
        result[key] = value
    return result


def audit(repo_root: Path, manifest: Path) -> dict[str, object]:
    root = repo_root.resolve()
    target = manifest.resolve()
    if not target.is_relative_to(root) or not target.is_file():
        raise InvalidGateManifest("manifest must be a regular repository file")
    if target.stat().st_size > 65536:
        raise InvalidGateManifest("manifest exceeds size limit")
    try:
        data = json.loads(target.read_text(encoding="utf-8"), object_pairs_hook=unique_object)
    except (OSError, UnicodeError, json.JSONDecodeError, InvalidGateManifest) as exc:
        raise InvalidGateManifest("invalid or unreadable gate manifest") from exc
    if not isinstance(data, dict) or set(data) != {"schema", "release_candidate", "gates"}:
        raise InvalidGateManifest("unexpected top-level manifest fields")
    if type(data["schema"]) is not int or data["schema"] != 1:
        raise InvalidGateManifest("unsupported gate schema")
    if data["release_candidate"] is not False:
        raise InvalidGateManifest("a source-controlled checklist cannot authorize a release")
    gates = data["gates"]
    if not isinstance(gates, list) or len(gates) != len(REQUIRED):
        raise InvalidGateManifest("missing or extra required gates")
    seen: set[str] = set()
    for gate in gates:
        if not isinstance(gate, dict) or set(gate) != {"id", "status", "evidence", "issue"}:
            raise InvalidGateManifest("invalid gate fields")
        gate_id = gate["id"]
        if not isinstance(gate_id, str) or gate_id not in REQUIRED or gate_id in seen:
            raise InvalidGateManifest("unknown or duplicate gate")
        seen.add(gate_id)
        source, issue = REQUIRED[gate_id]
        if gate["status"] != "blocked":
            raise InvalidGateManifest("unverifiable approval claim: external review is required")
        if gate["evidence"] != source or type(gate["issue"]) is not int or gate["issue"] != issue:
            raise InvalidGateManifest("gate evidence or issue binding changed")
        path = (root / source).resolve()
        if not path.is_relative_to(root) or not path.is_file():
            raise InvalidGateManifest("required gate reference missing or outside repository")
    if seen != set(REQUIRED):
        raise InvalidGateManifest("required gate IDs incomplete")
    return {
        "result": "blocked",
        "release_candidate": False,
        "blocked_count": len(gates),
        "blocked_gates": sorted(seen),
        "notice": "Development blocker inventory only; no owner/security/release approval is inferred.",
    }


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--expect-blocked", action="store_true", help="CI asserts blockers remain explicit; not a release-pass signal")
    args = parser.parse_args(argv)
    root = Path(__file__).resolve().parent.parent
    try:
        report = audit(root, root / "docs/P1_GATE_SNAPSHOT.json")
    except InvalidGateManifest as exc:
        print(json.dumps({"result": "invalid", "error": str(exc)}), file=sys.stderr)
        return 3
    print(json.dumps(report, sort_keys=True))
    return 0 if args.expect_blocked else 2


if __name__ == "__main__":
    sys.exit(main())
