#!/usr/bin/env python3
"""Draft-only, offline structural audit for a proposed P4 profile matrix.

This tool never grants product support approval. A structurally valid synthetic
fixture is not evidence that any real Tong Panel profile is supported.
"""

from __future__ import annotations

import argparse
import hashlib
import json
import re
from pathlib import Path, PurePosixPath
from typing import Any


SCHEMA_VERSION = 1
MAX_MATRIX_BYTES = 1_048_576
SHA256_RE = re.compile(r"^[0-9a-f]{40}$")
STATUSES = {"supported", "unsupported", "experimental", "unverified"}
EVIDENCE_RESULTS = {"passed", "failed", "blocked", "unresolved"}
REVIEW_DISPOSITIONS = {"accepted", "rejected", "pending"}
# P4-DESIGN-ACCEPTANCE.md currently numbers its acceptance cases 1 through 10.
# This draft validator treats those numbers as identifiers; that schema choice
# remains subject to owner/architecture review.
P4_CASE_IDS = {str(number) for number in range(1, 11)}
PLACEHOLDERS = {"", "*", "any", "latest", "tbd", "unknown", "unresolved"}
ALTERNATIVE_RE = re.compile(r"\b(?:and/or|or|and)\b|\||\s/\s|,", re.IGNORECASE)


class DuplicateJSONKey(ValueError):
    """Raised when a JSON object contains an ambiguous duplicate key."""


def _unique_object(pairs: list[tuple[str, Any]]) -> dict[str, Any]:
    result: dict[str, Any] = {}
    for key, value in pairs:
        if key in result:
            raise DuplicateJSONKey(f"duplicate JSON key: {key}")
        result[key] = value
    return result


def profile_fingerprint(profile: dict[str, Any]) -> str:
    """Return a stable digest for the exact profile fields used by this draft."""
    components = profile.get("components")
    canonical = {
        "id": profile.get("id"),
        "os": profile.get("os"),
        "architecture": profile.get("architecture"),
        "components": dict(sorted(components.items())) if isinstance(components, dict) else components,
    }
    encoded = json.dumps(canonical, sort_keys=True, separators=(",", ":"), ensure_ascii=True)
    return hashlib.sha256(encoded.encode("ascii")).hexdigest()


def _text(value: Any) -> bool:
    return isinstance(value, str) and bool(value.strip())


def _pinned_text(value: Any) -> bool:
    if not _text(value):
        return False
    normalized = value.strip().casefold()
    return (
        normalized not in PLACEHOLDERS
        and not re.search(r"\b(?:any|latest|tbd|unknown|unresolved)\b", normalized)
        and not any(character in value for character in "*?")
        and not ALTERNATIVE_RE.search(normalized)
    )


def _pinned_version(value: Any) -> bool:
    if not _pinned_text(value):
        return False
    normalized = value.strip()
    # Reject common range syntax while allowing exact package versions such as
    # Debian's ``1.2~rc1``. The final schema still needs an owner-approved
    # version grammar for each component ecosystem.
    if re.search(r"(^|\s)(?:>=|<=|>|<|~=|\^|~)\s*\S", normalized):
        return False
    if re.search(r"\s+-\s+|\|\|", normalized):
        return False
    return True


def _string_list(value: Any) -> bool:
    return isinstance(value, list) and all(_text(item) for item in value)


def _unknown_fields(value: dict[str, Any], allowed: set[str], location: str, errors: list[str]) -> None:
    for key in sorted(str(key) for key in value if key not in allowed):
        errors.append(f"field_unknown:{location}.{key}")


def _local_file(root: Path, reference: Any, location: str, errors: list[str]) -> Path | None:
    if not _text(reference):
        errors.append(f"reference_missing:{location}")
        return None
    value = reference.strip()
    # References are deliberately repository-relative and offline. Reject URL,
    # drive, backslash, absolute, and traversal forms before resolving them.
    path = PurePosixPath(value)
    parts = value.split("/")
    if (
        path.is_absolute()
        or "\\" in value
        or ":" in parts[0]
        or any(part in {"", ".", ".."} for part in parts)
        or "://" in value
    ):
        errors.append(f"reference_outside_root:{location}")
        return None
    try:
        root_real = root.resolve(strict=True)
        resolved = root_real.joinpath(*parts).resolve(strict=True)
        resolved.relative_to(root_real)
    except (OSError, RuntimeError, ValueError):
        errors.append(f"reference_unresolved:{location}")
        return None
    if not resolved.is_file():
        errors.append(f"reference_not_file:{location}")
        return None
    try:
        if resolved.stat().st_size == 0:
            errors.append(f"reference_empty:{location}")
            return None
    except OSError:
        errors.append(f"reference_unresolved:{location}")
        return None
    return resolved


def _profile_identity(profile: dict[str, Any]) -> tuple[Any, ...]:
    components = profile.get("components")
    normalized_components = ()
    if isinstance(components, dict):
        normalized_components = tuple(
            sorted((str(name).strip().casefold(), str(version).strip().casefold()) for name, version in components.items())
        )
    return (
        str(profile.get("os", "")).strip().casefold(),
        str(profile.get("architecture", "")).strip().casefold(),
        normalized_components,
    )


def audit_matrix(matrix: Any, root: Path, expected_source_commit: str) -> list[str]:
    """Return deterministic structural errors; never returns an approval."""
    errors: list[str] = []
    if not isinstance(matrix, dict):
        return ["matrix_not_object:root"]
    root = Path(root)
    _unknown_fields(matrix, {"schema_version", "candidate_commit", "profiles", "rows"}, "root", errors)

    schema_version = matrix.get("schema_version")
    if type(schema_version) is not int or schema_version != SCHEMA_VERSION:
        errors.append("schema_version_unsupported:root")

    candidate_commit = matrix.get("candidate_commit")
    if not isinstance(candidate_commit, str) or not SHA256_RE.fullmatch(candidate_commit):
        errors.append("candidate_commit_invalid:root")
    if not isinstance(expected_source_commit, str) or not SHA256_RE.fullmatch(expected_source_commit):
        errors.append("expected_source_commit_invalid:root")
    elif candidate_commit != expected_source_commit:
        errors.append("candidate_commit_stale:root")

    profiles_value = matrix.get("profiles")
    profiles: dict[str, dict[str, Any]] = {}
    identities: dict[tuple[Any, ...], str] = {}
    if not isinstance(profiles_value, list) or not profiles_value:
        errors.append("profiles_missing:root")
        profiles_value = []
    for index, profile in enumerate(profiles_value):
        location = f"profiles[{index}]"
        if not isinstance(profile, dict):
            errors.append(f"profile_not_object:{location}")
            continue
        _unknown_fields(profile, {"id", "os", "architecture", "components"}, location, errors)
        profile_id = profile.get("id")
        if not _pinned_text(profile_id):
            errors.append(f"profile_id_missing:{location}")
            continue
        profile_id = profile_id.strip()
        if profile_id in profiles:
            errors.append(f"profile_id_ambiguous:{profile_id}")
            continue
        os_name = profile.get("os")
        architecture = profile.get("architecture")
        components = profile.get("components")
        profile_valid = True
        if not _pinned_text(os_name):
            errors.append(f"profile_os_unpinned:{profile_id}")
            profile_valid = False
        if not _pinned_text(architecture):
            errors.append(f"profile_architecture_unpinned:{profile_id}")
            profile_valid = False
        if not isinstance(components, dict) or not components:
            errors.append(f"profile_components_missing:{profile_id}")
            profile_valid = False
        else:
            seen_component_names: set[str] = set()
            for name, version in components.items():
                if not _pinned_text(name) or not _pinned_version(version):
                    errors.append(f"profile_component_unpinned:{profile_id}")
                    profile_valid = False
                    continue
                normalized_name = name.strip().casefold()
                if normalized_name in seen_component_names:
                    errors.append(f"profile_component_ambiguous:{profile_id}")
                    profile_valid = False
                seen_component_names.add(normalized_name)
        profiles[profile_id] = profile
        if profile_valid:
            identity = _profile_identity(profile)
            previous = identities.get(identity)
            if previous is not None:
                errors.append(f"profile_identity_ambiguous:{previous},{profile_id}")
            else:
                identities[identity] = profile_id

    rows_value = matrix.get("rows")
    if not isinstance(rows_value, list) or not rows_value:
        errors.append("matrix_rows_missing:root")
        rows_value = []
    row_keys: set[tuple[str, str]] = set()
    for index, row in enumerate(rows_value):
        location = f"rows[{index}]"
        if not isinstance(row, dict):
            errors.append(f"row_not_object:{location}")
            continue
        _unknown_fields(
            row,
            {
                "capability",
                "profile_id",
                "status",
                "available",
                "prerequisite_gates",
                "acceptance_case_ids",
                "tested_commit",
                "evidence",
                "known_limits",
                "exclusions",
                "reviewer_decision",
            },
            location,
            errors,
        )

        capability = row.get("capability")
        profile_id = row.get("profile_id")
        if not _text(capability):
            errors.append(f"capability_missing:{location}")
            capability = ""
        if not _text(profile_id):
            errors.append(f"profile_reference_missing:{location}")
            profile_id = ""
        elif profile_id not in profiles:
            errors.append(f"profile_reference_unresolved:{location}")
        row_key = (capability.strip(), profile_id.strip())
        if row_key in row_keys:
            errors.append(f"capability_profile_ambiguous:{location}")
        row_keys.add(row_key)

        status = row.get("status")
        if not isinstance(status, str) or status not in STATUSES:
            errors.append(f"status_invalid:{location}")
        available = row.get("available")
        if not isinstance(available, bool):
            errors.append(f"availability_missing:{location}")
        elif status == "supported" and not available:
            errors.append(f"supported_not_available:{location}")
        elif isinstance(status, str) and status in {"unsupported", "experimental", "unverified"} and available:
            errors.append(f"non_supported_marked_available:{location}")

        gates = row.get("prerequisite_gates")
        if not isinstance(gates, list) or not gates:
            errors.append(f"prerequisite_gates_missing:{location}")
            gates = []
        gate_ids: set[str] = set()
        for gate_index, gate in enumerate(gates):
            gate_location = f"{location}.prerequisite_gates[{gate_index}]"
            if not isinstance(gate, dict) or not _text(gate.get("id")):
                errors.append(f"prerequisite_gate_invalid:{gate_location}")
                continue
            _unknown_fields(gate, {"id", "reference"}, gate_location, errors)
            gate_id = gate["id"].strip()
            if gate_id in gate_ids:
                errors.append(f"prerequisite_gate_ambiguous:{gate_location}")
            gate_ids.add(gate_id)
            _local_file(root, gate.get("reference"), f"{gate_location}.reference", errors)

        case_ids = row.get("acceptance_case_ids")
        if not isinstance(case_ids, list) or not case_ids:
            errors.append(f"acceptance_cases_missing:{location}")
            case_ids = []
        normalized_cases: list[str] = []
        for case_index, case_id in enumerate(case_ids):
            case_location = f"{location}.acceptance_case_ids[{case_index}]"
            if not isinstance(case_id, str) or case_id not in P4_CASE_IDS:
                errors.append(f"acceptance_case_unresolved:{case_location}")
                continue
            if case_id in normalized_cases:
                errors.append(f"acceptance_case_ambiguous:{case_location}")
            normalized_cases.append(case_id)

        for field in ("known_limits", "exclusions"):
            if not _string_list(row.get(field)):
                errors.append(f"{field}_missing:{location}")

        review = row.get("reviewer_decision")
        if (
            not isinstance(review, dict)
            or not isinstance(review.get("disposition"), str)
            or review.get("disposition") not in REVIEW_DISPOSITIONS
        ):
            errors.append(f"reviewer_decision_missing:{location}")
            review = {}
        else:
            _unknown_fields(review, {"disposition", "reviewer"}, f"{location}.reviewer_decision", errors)
        if status == "supported":
            if review.get("disposition") != "accepted" or not _text(review.get("reviewer")):
                errors.append(f"supported_reviewer_disposition_missing:{location}")

        row_commit = row.get("tested_commit")
        if row_commit is not None and (not isinstance(row_commit, str) or not SHA256_RE.fullmatch(row_commit)):
            errors.append(f"tested_commit_invalid:{location}")
        elif isinstance(row_commit, str) and row_commit != expected_source_commit:
            errors.append(f"row_tested_commit_stale:{location}")
        evidence_value = row.get("evidence")
        if not isinstance(evidence_value, list):
            errors.append(f"evidence_missing:{location}")
            evidence_value = []
        if evidence_value and row_commit is None:
            errors.append(f"tested_commit_missing:{location}")
        evidence_cases: set[str] = set()
        selected_profile = profiles.get(profile_id)
        expected_fingerprint = profile_fingerprint(selected_profile) if selected_profile is not None else None
        for evidence_index, evidence in enumerate(evidence_value):
            evidence_location = f"{location}.evidence[{evidence_index}]"
            if not isinstance(evidence, dict):
                errors.append(f"evidence_not_object:{evidence_location}")
                continue
            _unknown_fields(
                evidence,
                {"case_id", "tested_commit", "profile_id", "profile_fingerprint", "result", "artifact"},
                evidence_location,
                errors,
            )
            evidence_case = evidence.get("case_id")
            if not isinstance(evidence_case, str) or evidence_case not in P4_CASE_IDS:
                errors.append(f"evidence_case_unresolved:{evidence_location}")
            else:
                if evidence_case not in normalized_cases:
                    errors.append(f"evidence_case_not_required:{evidence_location}")
                if evidence_case in evidence_cases:
                    errors.append(f"evidence_case_ambiguous:{evidence_location}")
                evidence_cases.add(evidence_case)
            evidence_commit = evidence.get("tested_commit")
            if not isinstance(evidence_commit, str) or not SHA256_RE.fullmatch(evidence_commit):
                errors.append(f"evidence_commit_invalid:{evidence_location}")
            else:
                if evidence_commit != candidate_commit or evidence_commit != expected_source_commit:
                    errors.append(f"evidence_stale:{evidence_location}")
                if row_commit is not None and evidence_commit != row_commit:
                    errors.append(f"row_evidence_commit_mismatch:{evidence_location}")
            if evidence.get("profile_id") != profile_id:
                errors.append(f"evidence_profile_mismatch:{evidence_location}")
            if evidence.get("profile_fingerprint") != expected_fingerprint:
                errors.append(f"evidence_profile_stale:{evidence_location}")
            evidence_result = evidence.get("result")
            if not isinstance(evidence_result, str) or evidence_result not in EVIDENCE_RESULTS:
                errors.append(f"evidence_result_invalid:{evidence_location}")
            _local_file(root, evidence.get("artifact"), f"{evidence_location}.artifact", errors)

        if status == "supported":
            if not isinstance(row_commit, str) or row_commit != candidate_commit or row_commit != expected_source_commit:
                errors.append(f"supported_tested_commit_stale:{location}")
            if evidence_cases != set(normalized_cases):
                errors.append(f"supported_evidence_incomplete:{location}")
            for evidence_index, evidence in enumerate(evidence_value):
                if isinstance(evidence, dict) and evidence.get("result") != "passed":
                    errors.append(f"supported_evidence_not_passed:{location}.evidence[{evidence_index}]")

    return sorted(set(errors))


def _read_matrix(path: Path) -> Any:
    with path.open("rb") as source:
        raw = source.read(MAX_MATRIX_BYTES + 1)
    if len(raw) > MAX_MATRIX_BYTES:
        raise ValueError("matrix exceeds size limit")
    return json.loads(raw.decode("utf-8"), object_pairs_hook=_unique_object)


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(
        description="Draft-only structural audit; never grants product support approval."
    )
    parser.add_argument("matrix", type=Path, help="repository-relative JSON matrix")
    parser.add_argument("--root", type=Path, default=Path.cwd(), help="repository root (default: cwd)")
    parser.add_argument(
        "--expected-source-commit",
        required=True,
        help="exact source commit being audited; must match the matrix and each evidence record",
    )
    args = parser.parse_args(argv)
    root = args.root.resolve()
    matrix_path = args.matrix if args.matrix.is_absolute() else root / args.matrix
    errors: list[str] = []
    _local_file(root, matrix_path.relative_to(root).as_posix() if matrix_path.is_relative_to(root) else None, "matrix", errors)
    if not errors:
        try:
            matrix = _read_matrix(matrix_path)
            errors = audit_matrix(matrix, root, args.expected_source_commit)
        except (OSError, UnicodeError, json.JSONDecodeError, DuplicateJSONKey, ValueError, RecursionError) as exc:
            errors = [f"matrix_unreadable:{type(exc).__name__}"]
    result = {
        "tool_scope": "draft structural consistency only",
        "structurally_valid": not errors,
        "product_support_approval": False,
        "errors": errors,
    }
    print(json.dumps(result, sort_keys=True))
    return 0 if not errors else 1


if __name__ == "__main__":
    raise SystemExit(main())
