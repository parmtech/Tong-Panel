# Draft P4 compatibility-matrix validator

> **A structurally valid synthetic fixture or validator result is not product support approval.** This offline tool never approves P4, changes a gate, or marks a real profile supported. No real supported profile or matrix instance is included with this prototype.

## Purpose and scope

[P4-DESIGN-ACCEPTANCE.md](phases/P4-DESIGN-ACCEPTANCE.md) case 9 asks for a compatibility matrix tied to exact profiles, source commits, acceptance evidence, limitations, and reviewer disposition. It also asks for a fail-closed validator when required artifacts are missing, stale, unresolved, or tied to another profile/version. This script is a draft structural checker for a proposed JSON representation of that matrix. The existing Markdown seed register remains `TBD`/`unverified`; this tool does not parse it or turn it into machine-accepted evidence.

The matrix format and case identifiers below are **prototype choices**, not an approved API or support policy. The tool checks only local structure and paths. It does not contact GitHub, verify a workflow conclusion, authenticate a reviewer, inspect artifact contents, verify signatures, or decide whether evidence is sufficient for a real support claim. The caller supplies the exact source commit under review; evidence must match it. A successful result means only that the synthetic/proposed input passed these mechanical checks.

## Draft input contract

The JSON document has integer `schema_version: 1`, a 40-character lowercase `candidate_commit`, a non-empty `profiles` array, and a non-empty `rows` array. Each profile has a unique `id`, exact `os`, `architecture`, and a non-empty `components` map whose values are pinned versions. `TBD`, `unknown`, `latest`, wildcard forms, common version ranges, and blank values are rejected. Two different profile IDs with the same OS/architecture/component tuple are ambiguous and rejected.

Each row binds one `capability` to one declared `profile_id` and records:

- `status`: `supported`, `unsupported`, `experimental`, or `unverified`;
- `available`: must be consistent with the status label (`true` only for `supported`; false for all other statuses). This is a field-consistency check, not a product capability decision;
- `prerequisite_gates`: unique gate IDs with repository-relative `reference` files;
- `acceptance_case_ids`: unique case identifiers from the current numbered P4 cases 1-10;
- `tested_commit`, `evidence`, `known_limits`, `exclusions`, and `reviewer_decision`.

Evidence records bind one acceptance case to a `tested_commit`, `profile_id`, computed `profile_fingerprint`, `result`, and a non-empty repository-relative `artifact` file. For `supported`, evidence must cover every listed case exactly once, each result must be `passed`, the row and evidence commits must match both `candidate_commit` and the caller-supplied expected source commit, and the reviewer-decision fields must be present with an accepted disposition. For other statuses, partial or absent evidence is allowed, but a row cannot be marked available. These fields are self-reported metadata; their presence does not authenticate an approval.

All gate and artifact paths must resolve to non-empty files beneath `--root`. Network URLs, absolute paths, parent traversal, unknown cases or object fields, duplicate JSON members, duplicate capability/profile rows, missing profile references, stale commits, and changed profile fingerprints fail closed.

## Use

From the repository root, after downloading the referenced evidence files into a controlled local evidence directory:

```sh
python3 scripts/p4_compatibility_matrix_audit.py path/to/proposed-matrix.json \
  --root . --expected-source-commit <exact-40-character-source-commit>
```

The JSON output always includes `product_support_approval: false`. Exit status 0 means only that the input is structurally consistent with this draft format; nonzero means one or more checks failed. The tool does not add or rewrite matrix rows.

The synthetic tests run with the standard library:

```sh
python3 -m unittest discover -s scripts -p 'test_p4_compatibility_matrix_audit.py' -v
```

The [offline model and fixture workflow](../.github/workflows/offline-model-checks.yml) runs these tests on relevant changes. It does not certify P4.

## Unresolved schema choices

Owner/architecture review still needs to decide whether case numbers 1-10 become stable IDs; how optional service/provider combinations are split into profiles; the canonical exact-version grammar for each component ecosystem and whether all component versions belong in the profile fingerprint; which artifact formats and remote CI references are authoritative; what independent reviewer roles are required; and how evidence supersession is recorded. Until then, keep this tool and its test data isolated, preserve `TBD`/`unverified` dispositions, and do not use a passing fixture as phase evidence.
