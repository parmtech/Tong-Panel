#!/usr/bin/env python3
"""Check structural source/test references in the draft P0 threat crosswalk.

This checks that cited repository files and explicitly named test functions
have structural source-text signatures. It does not use a language parser or
run tests, assess mitigations, or approve any project gate.
"""
from __future__ import annotations

import json
import re
import sys
from pathlib import Path
from urllib.parse import unquote, urlsplit


MAX_CROSSWALK_BYTES = 262144
INVENTORY_HEADING = "## T01-T15 inventory"
END_HEADING = "## Commands for later evidence collection"
EXPECTED_THREATS = {f"T{number:02d}" for number in range(1, 16)}

MARKDOWN_LINK = re.compile(r"\[[^\]]*\]\(([^)]+)\)")
CODE_SPAN = re.compile(r"`([^`]+)`")
TEST_SYMBOL = re.compile(r"(?:Test|Fuzz|Benchmark|Example)[A-Z][A-Za-z0-9_]*|test_[A-Za-z0-9_]+")
PLAIN_GO_TEST_SYMBOL = re.compile(r"(?<![A-Za-z0-9_])(?:Test|Fuzz|Benchmark|Example)[A-Z][A-Za-z0-9_]*")
THREAT_ROW = re.compile(r"^\|\s*\*\*(T\d{2})\b")


class InvalidCrosswalk(ValueError):
    """The source/test reference inventory is incomplete or invalid."""


def _is_test_file(path: Path) -> bool:
    name = path.name.lower()
    return (
        name.endswith((
            "_test.go", "_test.py", "_test.js", "_test.mjs", "_test.ts", "_test.tsx",
            ".spec.js", ".spec.mjs", ".spec.ts", ".spec.tsx",
        ))
        or name.startswith("test_")
    )


def _resolve_local_link(root: Path, document: Path, raw_target: str) -> Path | None:
    try:
        parsed = urlsplit(raw_target.strip())
    except ValueError as exc:
        raise InvalidCrosswalk("local reference is malformed") from exc
    if parsed.scheme or parsed.netloc:
        return None
    relative_target = unquote(parsed.path)
    if not relative_target:
        return None

    try:
        candidate = (document.parent / Path(relative_target)).resolve()
        if not candidate.is_relative_to(root):
            raise InvalidCrosswalk("local reference resolves outside the repository")
        if not candidate.is_file():
            raise InvalidCrosswalk("local reference does not resolve to a file")
    except InvalidCrosswalk:
        raise
    except (OSError, RuntimeError, ValueError) as exc:
        raise InvalidCrosswalk("local reference cannot be resolved") from exc
    return candidate


def _definition_exists(path: Path, symbol: str) -> bool:
    try:
        source = path.read_text(encoding="utf-8", errors="replace")
    except OSError as exc:
        raise InvalidCrosswalk("test reference is unreadable") from exc

    escaped = re.escape(symbol)
    suffix = path.suffix.lower()
    if suffix == ".go":
        pattern = rf"(?m)^\s*func\s+{escaped}\s*\("
    elif suffix == ".py":
        pattern = rf"(?m)^\s*def\s+{escaped}\s*\("
    elif suffix in {".js", ".mjs", ".ts", ".tsx"}:
        pattern = (
            rf"^\s*(?:(?:export\s+)?(?:async\s+)?function\s+{escaped}\s*\(|"
            rf"(?:export\s+)?(?:const|let|var)\s+{escaped}\s*=)|"
            rf"\btest\s*\(\s*['\"]{escaped}['\"]"
        )
    elif suffix == ".sh":
        pattern = rf"(?m)^\s*(?:function\s+)?{escaped}\s*(?:\(\s*\))"
    else:
        return False
    return re.search(pattern, source, re.MULTILINE) is not None


def _table_cells(line: str) -> list[str]:
    if not line.rstrip().endswith("|"):
        raise InvalidCrosswalk("threat row is not a complete Markdown table row")
    cells = [cell.strip() for cell in line.strip().split("|")[1:-1]]
    if len(cells) != 5:
        raise InvalidCrosswalk("threat row has an unexpected number of columns")
    return cells


def _test_symbol_events(cell: str) -> list[tuple[int, str]]:
    events: list[tuple[int, str]] = []
    masked = list(cell)

    def blank(start: int, end: int) -> None:
        for index in range(start, end):
            if masked[index] not in "\r\n":
                masked[index] = " "

    for match in CODE_SPAN.finditer(cell):
        symbol = match.group(1).strip()
        if TEST_SYMBOL.fullmatch(symbol):
            events.append((match.start(), symbol))
        blank(match.start(), match.end())

    for match in MARKDOWN_LINK.finditer(cell):
        blank(match.start(), match.end())

    for match in PLAIN_GO_TEST_SYMBOL.finditer("".join(masked)):
        events.append((match.start(), match.group(0)))
    return sorted(events)


def audit_crosswalk(repo_root: Path, crosswalk_path: Path) -> dict[str, object]:
    """Validate local paths and explicitly named test definitions only."""
    try:
        root = repo_root.resolve(strict=True)
        document = crosswalk_path.resolve(strict=True)
    except (OSError, RuntimeError, ValueError) as exc:
        raise InvalidCrosswalk("repository root or crosswalk is unavailable") from exc
    if not root.is_dir() or not document.is_relative_to(root) or not document.is_file():
        raise InvalidCrosswalk("crosswalk must be a regular file inside the repository")
    try:
        if document.stat().st_size > MAX_CROSSWALK_BYTES:
            raise InvalidCrosswalk("crosswalk exceeds the structural audit size limit")
        # The validated paths and symbols are ASCII; replacement keeps unrelated
        # legacy prose encoding from changing the structural checks.
        lines = document.read_text(encoding="utf-8", errors="replace").splitlines()
    except InvalidCrosswalk:
        raise
    except OSError as exc:
        raise InvalidCrosswalk("crosswalk is unreadable") from exc

    try:
        start = lines.index(INVENTORY_HEADING) + 1
        end = lines.index(END_HEADING, start)
    except ValueError as exc:
        raise InvalidCrosswalk("crosswalk inventory section is missing or incomplete") from exc

    seen: list[str] = []
    local_reference_count = 0
    symbol_count = 0
    for line in lines[start:end]:
        row_match = THREAT_ROW.match(line)
        if not row_match:
            continue
        threat_id = row_match.group(1)
        if threat_id not in EXPECTED_THREATS or threat_id in seen:
            raise InvalidCrosswalk("unknown or duplicate threat row")
        seen.append(threat_id)
        cells = _table_cells(line)
        evidence_cell = cells[1]

        test_paths: list[tuple[int, Path]] = []
        row_local_reference_count = 0
        local_links = list(MARKDOWN_LINK.finditer(evidence_cell))
        for link in local_links:
            path = _resolve_local_link(root, document, link.group(1))
            if path is None:
                continue
            local_reference_count += 1
            row_local_reference_count += 1
            if _is_test_file(path):
                test_paths.append((link.start(), path))
        if not local_links or row_local_reference_count == 0:
            raise InvalidCrosswalk("threat row has no local source/test references")

        for position, symbol in _test_symbol_events(evidence_cell):
            preceding = [entry for entry in test_paths if entry[0] < position]
            if not preceding:
                raise InvalidCrosswalk("named test symbol has no preceding linked test file")
            test_path = max(preceding, key=lambda entry: entry[0])[1]
            if not _definition_exists(test_path, symbol):
                raise InvalidCrosswalk("named test symbol is not defined in its mapped file")
            symbol_count += 1

    if len(seen) != len(EXPECTED_THREATS) or set(seen) != EXPECTED_THREATS:
        raise InvalidCrosswalk("crosswalk must contain each threat row T01 through T15 exactly once")

    return {
        "result": "structural_only",
        "threat_rows": len(seen),
        "local_file_references": local_reference_count,
        "test_symbols_present": symbol_count,
        "symbol_check": "structural_text_signature_only",
        "notice": (
            "Named test symbols use structural source-text signatures, not a language parser or test execution. "
            "Reference presence only; cited tests were not executed, and no mitigation or approval is inferred."
        ),
    }


def main() -> int:
    root = Path(__file__).resolve().parent.parent
    try:
        report = audit_crosswalk(root, root / "docs/P0_THREAT_TEST_CROSSWALK.md")
    except InvalidCrosswalk as exc:
        print(json.dumps({"result": "invalid", "error": str(exc)}, sort_keys=True), file=sys.stderr)
        return 3
    print(json.dumps(report, sort_keys=True))
    return 0


if __name__ == "__main__":
    sys.exit(main())
