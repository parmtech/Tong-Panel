# Phase 3 disposable app fixtures

Synthetic fixtures for isolated acceptance-test design only. They contain no credentials, use no third-party packages, and do not represent the production app contract. Run only in a disposable test environment after the P3 entry gates are proven. The Node HTTP fixture binds only to loopback; Python is an in-process ASGI callable.

## Local fixture checks

From the repository root, run:

```text
node --test testdata/p3-fixtures/node/server.test.cjs
python -m unittest discover -s testdata/p3-fixtures/python -v
```

Each suite covers the synthetic readiness response (including the unhealthy case), release identity, unknown paths, and non-GET method denial. These checks do not cover any Phase 3 acceptance scenario end to end. In particular, they do not test tenant separation, package/runtime pinning, Git authentication or webhook signatures, sandboxing, quotas, secrets, backup/restore, custom-domain/TLS routing, deployment history, rollback, marketplace trust, or host health under load. Do not use these fixtures as evidence that any such property works.
