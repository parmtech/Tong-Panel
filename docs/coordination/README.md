# Tong Panel shared phase communication hub

Canonical branch: `docs/phase-coordination`. This is a documentation-only communication branch, not an integration or release branch. Do not merge it into phase work or main just to read messages.

## One place to read, separate places to write

All five Windows phase chats read this hub and **all six event folders**, not only their own. Each writer publishes a **new immutable Markdown file** in its own folder. Never have five workers rewrite one shared status file. Unique event files prevent line collisions and lost updates; Git still serializes branch-head updates, so writers must handle rejected pushes safely.

| Writer | Event log | Windows chat |
| --- | --- | --- |
| P1 | [P1 events](events/p1/) | Advance Tong-Panel Phase 1 |
| P2 | [P2 events](events/p2/) | Start Tong Panel Phase 2 smoke test |
| P3 | [P3 events](events/p3/) | Complete Tong Panel Phase 3 |
| P4 | [P4 events](events/p4/) | Continue Tong Panel Phase 4 |
| P5 | [P5 events](events/p5/) | Verify Tong Panel Phase 5 gates |
| Coordinator | [Coordinator events](events/coordinator/) | Coordination and cross-phase decisions |

The table is a stable navigation index, **not a live status dashboard**. Latest timestamped events are authoritative for communication; verify code and test evidence separately. Only the coordinator may edit this index/protocol. All other writers append their own events. A relay must name the real reporting phase and mark itself as a relay, never impersonate a worker acknowledgment.

## Required read/write cycle

1. At each session start/resume, fetch this branch or read it through GitHub. Read every new event in every folder since your last recorded read, including questions addressed to you and coordinator decisions.
2. Before changing a shared interface, claiming a gate passed, or starting a dependent slice, read again. While actively working, check at meaningful checkpoints and at least once per roughly 15 minutes of active work, not in a busy loop.
3. Publish your own status at start, meaningful milestone, changed plan, dependency question, blocker, and before pausing/handoff. Avoid empty heartbeat commits. Include the exact event IDs you read and any action or reply.
4. Answer a question with a new event in **your** folder, referencing the original question ID. The asker closes it with another event referencing the answer. Silence is not approval.
5. Read the remote copy after publication and record its commit. If write access is unavailable, report the exact blocker and the complete pending event in the chat for relay. Do not say you published it.
6. Idle/completed chats do not wake because GitHub changed. These are asynchronous checkpoints, not live chat or automatic scheduling. A coordinator/parent must invoke an idle chat when a timely response is needed. No unattended polling has been installed by this document.

## Event naming and contents

Path: `docs/coordination/events/<p1|p2|p3|p4|p5|coordinator>/<UTC-YYYYMMDDTHHMMSSZ>-<writer>-<random-UUID>.md`.

Use a fresh UUID and never overwrite/delete a published event. Correct mistakes with a new event referencing the superseded ID. Do not rely on timestamp ordering alone: include explicit reply/dependency IDs.

Each event must contain:

- Event ID, UTC time, writer/phase, kind (`status`, `question`, `answer`, `notice`, `blocker`, `decision`, `ack`), recipients.
- Current state and concrete completed work; current task/issue.
- Code branch, exact commit, PR, isolated worktree (redact personal home details).
- Tests actually executed with command/result/evidence; distinguish passed, failed, and not run.
- Gates/dependencies and their evidence; unverified assumptions.
- Question/decision needed, intended recipient, priority, next action.
- Read receipt: event IDs and remote coordination commit read; reply-to/supersedes IDs where relevant.
- On failure: sanitized command/error, attempt count, denial/pending/uncertain outcome, what completed, proposed next step.

This repository is **public**. Publish only project coordination information. Never include credentials, secrets, customer data, private chat transcripts, personal paths, or private security exploit details.

## Concurrency-safe publication

Preferred API path: create a **new** unique file on `docs/phase-coordination` using GitHub's contents API. Never use update-file for events. On a branch-head conflict, reread the head and check whether the exact event path already exists. If it exists with your exact content, publication succeeded; do not duplicate it. If absent and outcome is known, retry against fresh state at most twice. If content differs, stop and report. Do not blindly retry uncertain writes.

Git path: use a dedicated coordination checkout/worktree for **each writer**, separate from its code worktree, on a unique local branch such as `coord/p2-<session>` based on `origin/docs/phase-coordination`. Never check out the shared branch in multiple worktrees. Stage only the new event, commit, and push `HEAD:refs/heads/docs/phase-coordination` normally, never force-push. On non-fast-forward, fetch, first check whether the event landed, then rebase only your unpublished event commit onto the updated remote and retry once. Unique paths should make this clean. If conflict, unknown result, dirty unrelated files, or repeated rejection occurs, stop and escalate. Never reset, clean, stash, or switch another worker's checkout.

Only authorized repository paths and network routes may be used. API access is not permission to bypass a denied operation. Keep code commits on phase-specific branches/worktrees. Existing AGENTS.md and UPDATES.md requirements continue for phase code handoffs; this hub is an additional communication channel, not a replacement.

## Escalation and decisions

After **two materially identical failures**, stop blind retries and publish a blocker immediately; a third attempt needs materially new evidence. Escalate access, capability, security, or dependency blockers immediately. Coordinator may arrange a bounded stronger-model diagnosis through supported controls, then return to the requested routine model after targeted verification. Do not claim a model change without verifying it, and never treat a stronger model as permission to bypass access controls.

Cross-phase interface changes require an explicit proposal/answer and coordinator decision with affected owners and evidence before dependent implementation. A coordination decision cannot replace user/security approval. Conflicting claims remain blocked until reconciled.

## Phase gates and production limits

Use the latest repository plans and exact tested commits, not this branch's inherited code snapshot:
- P1: P0 architecture/threat decisions and complete P1 exit evidence.
- P2: verified P1 identity/jobs/host-agent, installer and secret-handling prerequisites; P2 exit includes two-tenant lifecycle/isolation and fresh-VM restore evidence.
- P3: verified P2 contracts/gate; deploy/rollback and hostile-app isolation.
- P4: verified P2/P3 prerequisites; mail/DNS, quota and migration acceptance.
- P5: verified P4 plus reliable single-server recovery; partition/revocation/recovery tests.

Until gates pass, stay within existing authorized design/test-specification/non-privileged prototype scope. No phase flip based on drafts, green unit tests alone, or chat claims. No production changes, merges, deployments, force pushes, privileged server mutations, or Phase 6 expansion are authorized by this hub.

## Initial verification state

Hub publication and per-worker adoption/read-write checks must be reported separately. Creation of this document does **not** prove all five workers can read/write. Coordinator events record the actual round-trip results and remaining blockers.
