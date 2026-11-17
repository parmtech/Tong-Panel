"""Structural regression checks for immutable external workflow actions."""

import re
from pathlib import Path
import unittest


USES_LINE = re.compile(r"^\s*(?:-\s*)?uses\s*:\s*(.*?)\s*$")
EXTERNAL_COMMIT_REF = re.compile(
    r"^(?:[A-Za-z0-9_.-]+/)+[A-Za-z0-9_.-]+@[0-9a-fA-F]{40}$"
)


def workflow_uses(text):
    """Return (line number, reference) pairs for YAML uses keys."""
    references = []
    for line_number, line in enumerate(text.splitlines(), start=1):
        match = USES_LINE.match(line)
        if match is None:
            continue

        value = match.group(1).strip()
        if value.startswith(("'", '"')):
            quote = value[0]
            closing = value.find(quote, 1)
            reference = value[1:closing] if closing > 0 else value
        else:
            reference = value.split("#", maxsplit=1)[0].strip()
        references.append((line_number, reference))
    return references


def is_pinned_reference(reference):
    if reference.startswith("./"):
        return len(reference) > 2 and not any(char.isspace() for char in reference)
    return EXTERNAL_COMMIT_REF.fullmatch(reference) is not None


def find_unpinned_references(workflow_root):
    problems = []
    workflow_files = sorted(
        path for path in workflow_root.rglob("*")
        if path.is_file() and path.suffix.lower() in {".yml", ".yaml"}
    )
    if not workflow_files:
        return [f"{workflow_root}: no workflow YAML files found"]

    for path in workflow_files:
        for line_number, reference in workflow_uses(path.read_text(encoding="utf-8")):
            if not is_pinned_reference(reference):
                problems.append(f"{path}:{line_number}: unpinned action reference {reference!r}")
    return problems


class WorkflowActionPinTests(unittest.TestCase):
    def test_requires_full_commit_sha_for_external_references(self):
        sha = "a" * 40
        self.assertTrue(is_pinned_reference(f"actions/checkout@{sha}"))
        self.assertTrue(is_pinned_reference("./.github/actions/local-check"))
        for reference in (
            "actions/checkout@v4",
            "actions/checkout@abc123",
            "actions/checkout@" + "a" * 39,
            "actions/checkout@" + "g" * 40,
            "../.github/actions/local-check",
            "docker://example/action:latest",
        ):
            with self.subTest(reference=reference):
                self.assertFalse(is_pinned_reference(reference))

    def test_extracts_uses_values_and_ignores_run_text(self):
        text = """jobs:
  build:
    steps:
      - uses: actions/checkout@aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa # v4
      - uses: './.github/actions/local-check'
        run: echo uses: actions/example@v1
"""
        self.assertEqual(
            workflow_uses(text),
            [
                (4, "actions/checkout@aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"),
                (5, "./.github/actions/local-check"),
            ],
        )

    def test_repository_workflow_external_references_are_pinned(self):
        repo_root = Path(__file__).resolve().parent.parent
        problems = find_unpinned_references(repo_root / ".github" / "workflows")
        self.assertEqual(problems, [], "\n".join(problems))


if __name__ == "__main__":
    unittest.main()
