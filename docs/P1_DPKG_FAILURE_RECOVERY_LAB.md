# P1 — unprivileged dpkg upgrade-failure and recovery lab

**Status:** real `dpkg` execution against *inert synthetic fixture packages*, with **separate temporary package database and installation root**. It is not a Tong Panel installation, root-host smoke test, signed package, verified release, or full interruption/disaster-recovery guarantee.

## What this tests

`scripts/test_dpkg_failure_recovery_lab.py` builds an explicitly synthetic `tong-dpkg-failure-fixture` package at versions 1 and 2 with tiny maintainer hooks that do nothing except `exit 0` or intentionally `exit 73`. A unique private `TemporaryDirectory` holds all package input, `dpkg` administrative records, private installation root, private log, and an unrelated state marker.

It **refuses to run as root**, and requires explicit `TONG_DISPOSABLE_DPKG_LAB=1` opt-in. A private package database and installation root do **not** necessarily suppress site-configured `dpkg` pre/post-invoke hooks: hosted CI showed an unrelated `needrestart` hook attempt to touch `/run`, denied by the unprivileged runner. Thus **only run this lab on disposable CI hosts/VM guests**; never run it against untrusted packages or on a real server. The opt-in and non-root checks are deliberate safety barriers.

It also preserves the existing non-root execution rule. Every `dpkg` invocation explicitly sets `--admindir`, `--instdir`, `--log` to the temporary sandbox, and uses `--force-not-root --force-script-chrootless` to execute **only the locally constructed inert hooks**. There is no real system package install, shell command from a client, network call, service restart, real site data, or host configuration change. Do not repurpose the test runner for untrusted `.deb` files: in chrootless mode a malicious maintainer hook could act with the executing user's permissions.

The four scenarios demonstrate **actual dpkg semantics**, not an invented recovery contract:

1. **New `preinst` fails:** the prior installed version and its payload remain version 1; unrelated private test state survives.
2. **New `postinst` fails after unpack:** version 2's payload exists, but `dpkg-query` reports `install ok half-configured|2`—**not a successful upgrade**. An explicitly requested version-1 reinstall restores installed status and version-1 payload; private test state survives. There is **no automatic rollback** in the test.
3. **Corrupt archive:** invalid package bytes are rejected before unpack; prior installed payload/state survive.
4. **Old `prerm` fails:** dpkg may **fall back to the new package's `prerm` and complete upgrade successfully**. This is a crucial limitation: a successful dpkg exit status **does not prove** all original script/service effects were safe or that an upgrade was healthy. The test observes and records the actual fallback, instead of falsely expecting a fail-closed abort.

Each case starts from a clean private package DB and executes as a nonprivileged process. Full daemon/process supervision, signed provenance, independent trusted key custody, package-manager kill/SIGKILL, filesystem power-loss semantics, fsync/atomic rename, malicious maintainer scripts, cross-tenant restore, systemd state and rollback safety on actual host directories are **not** tested.

## Running and CI evidence

On a Linux runner with `dpkg`, `dpkg-deb` and Python 3, **as a non-root account**:

```sh
TONG_DISPOSABLE_DPKG_LAB=1 python3 -m unittest discover -s scripts -p 'test_dpkg_failure_recovery_lab.py' -v
```

The clean-VM workflow runs the four-case suite once on its unprivileged Ubuntu runner and again as `ci` in **both disposable Ubuntu 24.04 and Debian 12 guests**, before their existing extracted preview/WordPress/backup/package lifecycle checks. Each guest still uses the existing pristine image, checksum checks, non-production warning and safety gates. Failure of the lab fails the workflow; it does not touch the guest's actual `dpkg` database.

Local development evidence on 13 Nov 2026: 4/4 tests passed both in a temporary-root fixture run and a second run under Unix user `nobody` (the latter checks the non-root behavior). A deliberate initial assumption that a failing old `prerm` always aborts was **falsified** by actual dpkg; the final regression correctly checks dpkg's fallback instead. The `postinst` case asserts the exact half-configured status observed locally. **Hosted evidence applies only after the new exact commit finishes CI**; do not inherit the prior PR #171 head's 13/13 result.

This lab complements [control/data archive verification](P1_DEV_PACKAGE_STRUCTURE.md) and the [read-only signed-release plus recovery-advice models](../prototypes/p1-release-manifest-model/README.md). None can approve a signed installer or host execution. ADR-007/008/009, independent security review, publisher key distribution/compromise response, rollback journaling and supported production recovery remain blockers in [P1 acceptance](P1_SECURITY_ACCEPTANCE_2026-11-13.md) and [master tracker #172](https://github.com/parmtech/Tong-Panel/issues/172).


## Hosted discrepancy and correction (13 November 2026)

Initial [exact-head VM run #43340739617](https://github.com/parmtech/Tong-Panel/actions/runs/43340739617) failed in both runner jobs **before booting guests** because the test asserted one exact English `dpkg` warning string. The GitHub runner uses `dpkg: trying prerm maintainer script from the new ...`, whereas the local version says `dpkg: trying script from the new package ...`. The test now recognizes both precise fallback forms **and still requires actual version-2 installed state, payload and unchanged private marker**. Logs also exposed a configured `needrestart` hook attempting `/run` access (denied); explicit disposable-environment opt-in and prominent warnings were added instead of pretending the private admin directory suppresses all system configuration. Local regression with an unprivileged account passed 4/4 with opt-in and rejected execution without opt-in. New exact-head hosted verification is required.
