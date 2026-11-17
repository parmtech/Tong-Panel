# P2 restore generation consistency model (synthetic only)

This standalone Python-standard-library model checks one proposed acceptance
condition: site/files, database, and settings records must agree on the same
synthetic tenant, backup ID, generation, and opaque marker before activation.
The model validates the complete candidate first, then replaces one in-memory
active-bundle pointer. A denied candidate preserves the prior active bundle.

Tests cover all-agree activation, complete replacement without mixed
generations, missing/duplicate/unknown components, missing or empty markers,
and invalid identity values. Each of `site_files`, `database`, and `settings`
is independently tested for tenant, backup ID, generation, and marker
mismatches, plus missing and empty markers. Non-record components and malformed
tenant/backup ID types are also denied without replacing a prior active bundle.
Identity integers and marker strings are disposable test inputs only; this
model does not select a production identifier, marker format, generation
policy, or consistency policy.

The model performs no filesystem, database, network, encryption, key, provider,
scheduler, authorization, or host operation. It stores no payload and models no
snapshot/quiescence behavior, durable state, actual restore, cleanup, or product
activation. The pointer assignment is a synthetic state transition, not proof
of an atomic filesystem or database transaction. These tests are acceptance
preparation only, not P2 acceptance or security approval. P0/P1, owner/security,
production, and integrated-restore gates remain open.

From this directory, run:

```sh
PYTHONDONTWRITEBYTECODE=1 python3 -m unittest discover -s . -p 'test_*.py' -v
```

The offline workflow runs the same standard-library test discovery on Ubuntu
24.04 with a pinned checkout action and no third-party Python installation.
Passing the synthetic check does not close issue #152 or change any gate.
