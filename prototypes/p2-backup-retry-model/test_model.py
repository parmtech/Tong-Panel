import unittest
from dataclasses import FrozenInstanceError, replace

from model import (
    AttemptState,
    AttemptTerminal,
    DeleteOutOfScope,
    DigestMismatch,
    GenerationConflict,
    Identity,
    InvalidIdentity,
    InvalidCheckpoint,
    InvalidTransferClock,
    InvalidTransferPolicy,
    Model,
    StaleAttempt,
    TransferAttemptLimit,
    TransferBackoffNotElapsed,
    TransferConcurrencyLimit,
    TransferFailureReason,
    TransferLimits,
    TransferNotRetryable,
    TransferReconciliationPending,
    TransferSimulator,
    TransferState,
    UnknownAttempt,
    identity_for_content,
)


class FakeClock:
    def __init__(self, value=0.0):
        self.value = value

    def __call__(self):
        return self.value

    def advance(self, seconds):
        self.value += seconds


def transfer_simulator(
    *,
    max_size=64,
    max_duration=5,
    max_attempts=2,
    backoff=(2,),
    concurrency=1,
    clock_epoch=None,
):
    clock = FakeClock()
    limits = TransferLimits(
        max_size_bytes=max_size,
        max_duration_seconds=max_duration,
        max_attempts=max_attempts,
        backoff_seconds=backoff,
        max_concurrent_transfers=concurrency,
    )
    model = Model()
    return model, clock, TransferSimulator(model, limits, clock, clock_epoch=clock_epoch)


def fixture_identity(backup_id, generation, content):
    """Keep legacy fixture labels while supplying an explicit tenant identity."""

    if isinstance(backup_id, str):
        parts = backup_id.split("-", 2)
        tenant_id = (
            "-".join(parts[:2])
            if len(parts) >= 2 and parts[0] == "tenant"
            else "fixture-tenant"
        )
    else:
        tenant_id = "fixture-tenant"
    return identity_for_content(tenant_id, backup_id, generation, content)


def replace_transfer_record(checkpoint, attempt_id, **changes):
    return replace(
        checkpoint,
        transfer_attempts=tuple(
            replace(record, **changes) if record.attempt_id == attempt_id else record
            for record in checkpoint.transfer_attempts
        ),
    )


class BackupRetryModelTests(unittest.TestCase):
    def test_newer_generation_fences_stale_publish_and_delete(self):
        model = Model()
        older_bytes = b"synthetic generation one"
        newer_bytes = b"synthetic generation two"
        older = fixture_identity("tenant-a-nightly", 1, older_bytes)
        newer = fixture_identity("tenant-a-nightly", 2, newer_bytes)

        older_publisher = model.begin_attempt(older)
        stale_resume = model.begin_attempt(older)
        newer_publisher = model.begin_attempt(newer)
        model.verify_and_publish(older_publisher, older_bytes)
        model.verify_and_publish(newer_publisher, newer_bytes)

        with self.assertRaises(StaleAttempt):
            model.verify_and_publish(stale_resume, older_bytes)
        self.assertEqual(model.attempt(stale_resume).state, AttemptState.STALE)

        with self.assertRaises(StaleAttempt):
            model.reject_stale_delete(older_publisher)
        self.assertEqual(model.attempt(older_publisher).state, AttemptState.STALE)

        current = model.current(older.tenant_id, older.backup_id)
        self.assertIsNotNone(current)
        self.assertEqual(current.identity, newer)
        self.assertEqual(current.published_by, newer_publisher)
        with self.assertRaises(AttemptTerminal):
            model.verify_and_publish(stale_resume, older_bytes)
        with self.assertRaises(AttemptTerminal):
            model.reject_stale_delete(older_publisher)
        self.assertEqual(model.current(older.tenant_id, older.backup_id), current)

    def test_non_stale_delete_guard_is_out_of_scope_without_mutation(self):
        model = Model()
        content = b"synthetic verified artifact"
        identity = fixture_identity("tenant-a-nightly", 1, content)
        publisher = model.begin_attempt(identity)
        model.verify_and_publish(publisher, content)

        state_before = model.attempt(publisher).state
        current_before = model.current(identity.tenant_id, identity.backup_id)
        with self.assertRaises(DeleteOutOfScope):
            model.reject_stale_delete(publisher)

        self.assertEqual(model.attempt(publisher).state, state_before)
        self.assertEqual(model.current(identity.tenant_id, identity.backup_id), current_before)

    def test_digest_mismatch_fails_without_changing_current(self):
        model = Model()
        first_bytes = b"synthetic verified artifact"
        first = fixture_identity("tenant-b-nightly", 7, first_bytes)
        first_attempt = model.begin_attempt(first)
        model.verify_and_publish(first_attempt, first_bytes)

        expected_bytes = b"next expected artifact"
        next_identity = fixture_identity("tenant-b-nightly", 8, expected_bytes)
        next_attempt = model.begin_attempt(next_identity)
        with self.assertRaises(DigestMismatch):
            model.verify_and_publish(next_attempt, b"corrupted synthetic artifact")
        self.assertEqual(model.attempt(next_attempt).state, AttemptState.FAILED)
        self.assertEqual(model.current(first.tenant_id, first.backup_id).identity, first)
        self.assertEqual(model.current(first.tenant_id, first.backup_id).published_by, first_attempt)

    def test_attempt_identity_is_immutable_after_begin(self):
        model = Model()
        content = b"identity-bound synthetic bytes"
        identity = fixture_identity("tenant-a-weekly", 4, content)
        attempt_id = model.begin_attempt(identity)

        with self.assertRaises(FrozenInstanceError):
            identity.generation = 99
        with self.assertRaises(FrozenInstanceError):
            identity.tenant_id = "tenant-b"
        model.verify_and_publish(attempt_id, content)

        attempt = model.attempt(attempt_id)
        self.assertEqual(attempt.identity, identity)
        self.assertEqual(attempt.state, AttemptState.VERIFIED)
        self.assertEqual(model.current(identity.tenant_id, identity.backup_id).identity, identity)

    def test_boolean_attempt_id_cannot_alias_integer_attempt(self):
        model = Model()
        content = b"strict synthetic attempt id"
        identity = fixture_identity("tenant-a-weekly", 4, content)
        attempt_id = model.begin_attempt(identity)
        self.assertEqual(attempt_id, 1)

        with self.assertRaises(UnknownAttempt):
            model.attempt(True)
        with self.assertRaises(UnknownAttempt):
            model.verify_and_publish(True, content)
        with self.assertRaises(UnknownAttempt):
            model.reject_stale_delete(True)

        self.assertEqual(model.attempt(attempt_id).state, AttemptState.ACTIVE)
        self.assertIsNone(model.current(identity.tenant_id, identity.backup_id))

    def test_same_generation_cannot_change_digest(self):
        model = Model()
        first_bytes = b"generation content"
        first = fixture_identity("tenant-a-monthly", 3, first_bytes)
        first_attempt = model.begin_attempt(first)
        model.verify_and_publish(first_attempt, first_bytes)

        conflicting_bytes = b"different content"
        conflicting = fixture_identity(first.backup_id, first.generation, conflicting_bytes)
        conflict_attempt = model.begin_attempt(conflicting)
        with self.assertRaises(GenerationConflict):
            model.verify_and_publish(conflict_attempt, conflicting_bytes)
        self.assertEqual(model.attempt(conflict_attempt).state, AttemptState.FAILED)
        self.assertEqual(model.current(first.tenant_id, first.backup_id).identity, first)
        self.assertEqual(model.current(first.tenant_id, first.backup_id).published_by, first_attempt)

    def test_identity_rejects_invalid_generation_and_digest(self):
        with self.assertRaises(InvalidIdentity):
            fixture_identity(42, 1, b"synthetic")
        with self.assertRaises(InvalidIdentity):
            fixture_identity("tenant-a", True, b"synthetic")
        with self.assertRaises(InvalidIdentity):
            Identity("tenant-a", "nightly", 1, "not-a-digest")
        for digest in (None, bytes(64)):
            with self.subTest(digest_type=type(digest).__name__):
                with self.assertRaises(InvalidIdentity):
                    Identity("tenant-a", "nightly", 1, digest)

    def test_identity_requires_non_empty_string_tenant(self):
        model = Model()
        content = b"tenant identity baseline"
        baseline = identity_for_content("tenant-a", "nightly", 1, content)
        publisher = model.begin_attempt(baseline)
        model.verify_and_publish(publisher, content)
        current_before = model.current("tenant-a", "nightly")
        attempt_before = model.attempt(publisher)

        for tenant_id in ("", None, 17, True, b"tenant-a"):
            with self.subTest(tenant_id=tenant_id):
                with self.assertRaises(InvalidIdentity):
                    identity_for_content(tenant_id, "nightly", 1, content)
                with self.assertRaises(InvalidIdentity):
                    model.current(tenant_id, "nightly")

        self.assertEqual(model.attempt(publisher), attempt_before)
        self.assertEqual(model.current("tenant-a", "nightly"), current_before)

    def test_same_backup_id_is_independent_across_tenants(self):
        model = Model()
        content_a = b"tenant A shared-name backup"
        content_b = b"tenant B shared-name backup"
        identity_a = identity_for_content("tenant-a", "nightly", 1, content_a)
        identity_b = identity_for_content("tenant-b", "nightly", 1, content_b)
        attempt_a = model.begin_attempt(identity_a)
        attempt_b = model.begin_attempt(identity_b)

        model.verify_and_publish(attempt_a, content_a)
        model.verify_and_publish(attempt_b, content_b)

        current_a = model.current("tenant-a", "nightly")
        current_b = model.current("tenant-b", "nightly")
        self.assertEqual(current_a.identity, identity_a)
        self.assertEqual(current_a.published_by, attempt_a)
        self.assertEqual(current_b.identity, identity_b)
        self.assertEqual(current_b.published_by, attempt_b)
        self.assertIsNone(model.current("tenant-c", "nightly"))

    def test_generation_conflict_and_stale_fence_are_tenant_local(self):
        model = Model()
        content_a1 = b"tenant A generation one"
        content_a2 = b"tenant A generation two"
        content_a2_conflict = b"tenant A conflicting generation two"
        content_b99 = b"tenant B generation ninety nine"
        identity_a1 = identity_for_content("tenant-a", "nightly", 1, content_a1)
        identity_a2 = identity_for_content("tenant-a", "nightly", 2, content_a2)
        conflict_a2 = identity_for_content(
            "tenant-a", "nightly", 2, content_a2_conflict
        )
        identity_b99 = identity_for_content("tenant-b", "nightly", 99, content_b99)

        attempt_a1 = model.begin_attempt(identity_a1)
        stale_resume_a1 = model.begin_attempt(identity_a1)
        attempt_b99 = model.begin_attempt(identity_b99)
        model.verify_and_publish(attempt_b99, content_b99)
        # Tenant B's higher generation must not fence tenant A's same-named stream.
        model.verify_and_publish(attempt_a1, content_a1)
        with self.assertRaises(DeleteOutOfScope):
            model.reject_stale_delete(attempt_a1)
        self.assertEqual(model.attempt(attempt_a1).state, AttemptState.VERIFIED)

        attempt_a2 = model.begin_attempt(identity_a2)
        model.verify_and_publish(attempt_a2, content_a2)

        with self.assertRaises(StaleAttempt):
            model.verify_and_publish(stale_resume_a1, content_a1)
        self.assertEqual(model.current("tenant-a", "nightly").identity, identity_a2)
        self.assertEqual(model.current("tenant-b", "nightly").identity, identity_b99)

        conflict_attempt = model.begin_attempt(conflict_a2)
        with self.assertRaises(GenerationConflict):
            model.verify_and_publish(conflict_attempt, content_a2_conflict)
        self.assertEqual(model.attempt(conflict_attempt).state, AttemptState.FAILED)
        self.assertEqual(model.current("tenant-a", "nightly").identity, identity_a2)
        self.assertEqual(model.current("tenant-b", "nightly").identity, identity_b99)

    def test_checkpoint_round_trip_preserves_same_backup_id_per_tenant(self):
        content_a = b"checkpoint tenant A"
        content_b = b"checkpoint tenant B"
        identity_a = identity_for_content("tenant-a", "nightly", 1, content_a)
        identity_b = identity_for_content("tenant-b", "nightly", 1, content_b)
        model, _, simulator = transfer_simulator(
            max_attempts=1,
            backoff=(),
            concurrency=2,
            clock_epoch="fixture-tenant-checkpoint",
        )

        attempt_a = simulator.start_transfer(identity_a, len(content_a))
        simulator.report_progress(attempt_a, len(content_a))
        simulator.finish_transfer(attempt_a, content_a)
        attempt_b = simulator.start_transfer(identity_b, len(content_b))
        simulator.report_progress(attempt_b, len(content_b))
        simulator.finish_transfer(attempt_b, content_b)

        checkpoint = simulator.checkpoint()
        rebuilt_model, rebuilt = checkpoint.reconstruct(
            FakeClock(checkpoint.clock_anchor),
            clock_epoch="fixture-tenant-checkpoint",
        )

        self.assertEqual(rebuilt_model.current("tenant-a", "nightly").identity, identity_a)
        self.assertEqual(rebuilt_model.current("tenant-b", "nightly").identity, identity_b)
        self.assertEqual(rebuilt.transfer(attempt_a).identity, identity_a)
        self.assertEqual(rebuilt.transfer(attempt_b).identity, identity_b)

        duplicate_current = replace(
            checkpoint,
            current_artifacts=checkpoint.current_artifacts
            + (checkpoint.current_artifacts[0],),
        )
        with self.assertRaises(InvalidCheckpoint):
            duplicate_current.reconstruct(
                FakeClock(checkpoint.clock_anchor),
                clock_epoch="fixture-tenant-checkpoint",
            )

    def test_transfer_limits_require_explicit_bounds_and_backoff_entries(self):
        with self.assertRaises(InvalidTransferPolicy):
            TransferLimits(64, 5, 2, (), 1)
        with self.assertRaises(InvalidTransferPolicy):
            TransferLimits(64, 5, True, (), 1)

    def test_oversize_transfer_is_terminal_before_any_upload(self):
        model, _, simulator = transfer_simulator(max_size=3)
        content = b"four"
        identity = fixture_identity("tenant-a-nightly", 1, content)

        attempt = simulator.start_transfer(identity, len(content))
        result = simulator.transfer(attempt)

        self.assertEqual(result.state, TransferState.FAILED)
        self.assertEqual(result.failure_reason, TransferFailureReason.SIZE_LIMIT)
        self.assertEqual(result.transferred_bytes, 0)
        self.assertEqual(model.attempt(attempt).state, AttemptState.FAILED)
        self.assertIsNone(model.current(identity.tenant_id, identity.backup_id))
        with self.assertRaises(TransferNotRetryable):
            simulator.retry_transfer(attempt)

    def test_transfer_publishes_only_after_full_digest_verification(self):
        model, clock, simulator = transfer_simulator(max_attempts=1, backoff=())
        content = b"synthetic verified bytes"
        identity = fixture_identity("tenant-a-nightly", 4, content)

        attempt = simulator.start_transfer(identity, len(content))
        with self.assertRaises(TransferConcurrencyLimit):
            simulator.start_transfer(
                fixture_identity("tenant-b-nightly", 1, b"other"), len(b"other")
            )
        simulator.report_progress(attempt, len(content))
        clock.advance(1)
        result = simulator.finish_transfer(attempt, content)

        self.assertEqual(result.state, TransferState.VERIFIED)
        self.assertIsNone(result.failure_reason)
        self.assertEqual(result.duration_seconds, 1)
        self.assertEqual(model.attempt(attempt).state, AttemptState.VERIFIED)
        self.assertEqual(model.current(identity.tenant_id, identity.backup_id).identity, identity)

    def test_transfer_at_inclusive_size_and_duration_limits_can_verify(self):
        content = b"exact-bound synthetic payload"
        identity = fixture_identity("tenant-a-nightly", 4, content)
        max_duration = 3
        model, clock, simulator = transfer_simulator(
            max_size=len(content),
            max_duration=max_duration,
            max_attempts=1,
            backoff=(),
        )

        attempt = simulator.start_transfer(identity, len(content))
        simulator.report_progress(attempt, len(content))
        clock.advance(max_duration)
        result = simulator.finish_transfer(attempt, content)

        self.assertEqual(result.state, TransferState.VERIFIED)
        self.assertEqual(result.expected_size_bytes, len(content))
        self.assertEqual(result.transferred_bytes, len(content))
        self.assertEqual(result.duration_seconds, max_duration)
        self.assertEqual(model.current(identity.tenant_id, identity.backup_id).identity, identity)

    def test_identity_handoff_to_fresh_simulator_only(self):
        content = b"synthetic identity handoff bytes"
        identity = fixture_identity("tenant-a-nightly", 6, content)
        limits = TransferLimits(
            max_size_bytes=len(content),
            max_duration_seconds=5,
            max_attempts=1,
            backoff_seconds=(),
            max_concurrent_transfers=1,
        )
        first_clock = FakeClock()
        first_model = Model()
        first_simulator = TransferSimulator(first_model, limits, first_clock)
        first_attempt = first_simulator.start_transfer(identity, len(content))
        first_simulator.interrupt_transfer(first_attempt)
        snapshot = first_simulator.transfer(first_attempt)
        self.assertEqual(snapshot.state, TransferState.FAILED)
        self.assertEqual(snapshot.transferred_bytes, 0)

        # Carry only identity and size; do not infer operation/attempt/backoff recovery,
        # durable persistence, or process-restart behavior from this instance handoff.
        fresh_clock = FakeClock()
        fresh_model = Model()
        fresh_simulator = TransferSimulator(fresh_model, limits, fresh_clock)
        fresh_attempt = fresh_simulator.start_transfer(
            snapshot.identity, snapshot.expected_size_bytes
        )
        self.assertEqual(fresh_simulator.transfer(fresh_attempt).identity, snapshot.identity)

        fresh_simulator.report_progress(fresh_attempt, len(content))
        result = fresh_simulator.finish_transfer(fresh_attempt, content)

        self.assertEqual(result.state, TransferState.VERIFIED)
        self.assertEqual(result.identity, snapshot.identity)
        self.assertEqual(result.transferred_bytes, snapshot.expected_size_bytes)
        self.assertEqual(fresh_model.current(identity.tenant_id, identity.backup_id).identity, identity)

    def test_partial_upload_requires_verified_cleanup_or_reconciliation(self):
        content = b"synthetic partial object"
        identity = fixture_identity("tenant-a-nightly", 1, content)

        model, _, simulator = transfer_simulator()
        cleaned_attempt = simulator.start_transfer(identity, len(content))
        simulator.report_progress(cleaned_attempt, 5)
        cleaned = simulator.interrupt_transfer(cleaned_attempt, cleanup_verified=True)
        self.assertEqual(cleaned.state, TransferState.CLEANED)
        self.assertTrue(cleaned.cleanup_verified)
        self.assertEqual(model.attempt(cleaned_attempt).state, AttemptState.FAILED)
        self.assertIsNone(model.current(identity.tenant_id, identity.backup_id))

        unresolved_model, _, unresolved_simulator = transfer_simulator()
        unresolved_attempt = unresolved_simulator.start_transfer(identity, len(content))
        unresolved_simulator.report_progress(unresolved_attempt, 5)
        unresolved = unresolved_simulator.interrupt_transfer(unresolved_attempt)
        self.assertEqual(unresolved.state, TransferState.RECONCILIATION_REQUIRED)
        self.assertIsNone(unresolved.cleanup_verified)
        self.assertEqual(
            unresolved_model.attempt(unresolved_attempt).state,
            AttemptState.RECONCILIATION_REQUIRED,
        )
        self.assertIsNone(unresolved_model.current(identity.tenant_id, identity.backup_id))
        with self.assertRaises(TransferReconciliationPending):
            unresolved_simulator.retry_transfer(unresolved_attempt)

    def test_retry_uses_same_identity_and_enforces_backoff_and_attempt_cap(self):
        model, clock, simulator = transfer_simulator(max_attempts=2, backoff=(2,))
        content = b"synthetic retry bytes"
        identity = fixture_identity("tenant-a-nightly", 3, content)
        first = simulator.start_transfer(identity, len(content))
        simulator.interrupt_transfer(first)

        with self.assertRaises(TransferBackoffNotElapsed):
            simulator.retry_transfer(first)
        clock.advance(2)
        second = simulator.retry_transfer(first)
        self.assertEqual(simulator.transfer(second).identity, identity)
        self.assertEqual(simulator.transfer(second).attempt_number, 2)
        simulator.interrupt_transfer(second)
        with self.assertRaises(TransferAttemptLimit):
            simulator.retry_transfer(second)
        self.assertIsNone(model.current(identity.tenant_id, identity.backup_id))

    def test_duration_limit_with_partial_bytes_requires_reconciliation(self):
        model, clock, simulator = transfer_simulator(max_duration=2)
        content = b"synthetic timeout payload"
        identity = fixture_identity("tenant-a-nightly", 2, content)
        attempt = simulator.start_transfer(identity, len(content))
        clock.advance(3)

        result = simulator.report_progress(attempt, 4)

        self.assertEqual(result.state, TransferState.RECONCILIATION_REQUIRED)
        self.assertEqual(result.failure_reason, TransferFailureReason.DURATION_LIMIT)
        self.assertEqual(result.duration_seconds, 3)
        self.assertEqual(model.attempt(attempt).state, AttemptState.RECONCILIATION_REQUIRED)
        self.assertIsNone(model.current(identity.tenant_id, identity.backup_id))

    def test_digest_mismatch_cannot_be_reported_as_verified(self):
        model, _, simulator = transfer_simulator()
        expected = b"same-size expected bytes"
        corrupted = b"same-size mismatch bytes"
        identity = fixture_identity("tenant-a-nightly", 5, expected)
        attempt = simulator.start_transfer(identity, len(expected))
        simulator.report_progress(attempt, len(expected))

        result = simulator.finish_transfer(attempt, corrupted, cleanup_verified=True)

        self.assertEqual(result.state, TransferState.CLEANED)
        self.assertEqual(result.failure_reason, TransferFailureReason.DIGEST_MISMATCH)
        self.assertTrue(result.cleanup_verified)
        self.assertEqual(model.attempt(attempt).state, AttemptState.FAILED)
        self.assertIsNone(model.current(identity.tenant_id, identity.backup_id))

    def test_stale_transfer_cannot_replace_a_newer_verified_generation(self):
        model, _, simulator = transfer_simulator(max_attempts=1, backoff=(), concurrency=2)
        older_bytes = b"synthetic transfer generation one"
        newer_bytes = b"synthetic transfer generation two"
        older = fixture_identity("tenant-a-nightly", 1, older_bytes)
        newer = fixture_identity("tenant-a-nightly", 2, newer_bytes)

        old_attempt = simulator.start_transfer(older, len(older_bytes))
        simulator.report_progress(old_attempt, len(older_bytes))
        new_attempt = simulator.start_transfer(newer, len(newer_bytes))
        simulator.report_progress(new_attempt, len(newer_bytes))
        self.assertEqual(
            simulator.finish_transfer(new_attempt, newer_bytes).state,
            TransferState.VERIFIED,
        )
        stale = simulator.finish_transfer(old_attempt, older_bytes)

        self.assertEqual(stale.state, TransferState.STALE)
        self.assertEqual(stale.failure_reason, TransferFailureReason.STALE_GENERATION)
        self.assertEqual(model.attempt(old_attempt).state, AttemptState.STALE)
        self.assertEqual(model.current(older.tenant_id, older.backup_id).identity, newer)

    def test_checkpoint_preserves_unresolved_reconciliation_and_keeps_retry_blocked(self):
        content = b"synthetic unresolved checkpoint payload"
        identity = fixture_identity("tenant-a-nightly", 1, content)
        model, clock, simulator = transfer_simulator(clock_epoch="fixture-epoch-a")
        attempt = simulator.start_transfer(identity, len(content))
        simulator.report_progress(attempt, 7)
        simulator.interrupt_transfer(attempt)

        checkpoint = simulator.checkpoint()
        rebuilt_model, rebuilt = checkpoint.reconstruct(
            FakeClock(checkpoint.clock_anchor), clock_epoch="fixture-epoch-a"
        )

        self.assertEqual(
            rebuilt.transfer(attempt).state,
            TransferState.RECONCILIATION_REQUIRED,
        )
        self.assertEqual(
            rebuilt_model.attempt(attempt).state,
            AttemptState.RECONCILIATION_REQUIRED,
        )
        with self.assertRaises(TransferReconciliationPending):
            rebuilt.retry_transfer(attempt)
        self.assertEqual(model.attempt(attempt).state, AttemptState.RECONCILIATION_REQUIRED)

    def test_checkpoint_retains_retry_backoff_attempt_and_concurrency_caps(self):
        content = b"synthetic retry checkpoint payload"
        identity = fixture_identity("tenant-a-weekly", 4, content)
        model, clock, simulator = transfer_simulator(
            max_size=len(content),
            max_duration=5,
            max_attempts=2,
            backoff=(3,),
            concurrency=1,
            clock_epoch="fixture-epoch-retry",
        )
        first = simulator.start_transfer(identity, len(content))
        simulator.report_progress(first, 1)
        clock.advance(1)
        simulator.interrupt_transfer(first, cleanup_verified=True)

        checkpoint = simulator.checkpoint()
        rebuilt_clock = FakeClock(checkpoint.clock_anchor)
        _, rebuilt = checkpoint.reconstruct(
            rebuilt_clock, clock_epoch="fixture-epoch-retry"
        )

        self.assertEqual(rebuilt._limits, simulator._limits)
        with self.assertRaises(TransferBackoffNotElapsed):
            rebuilt.retry_transfer(first)
        rebuilt_clock.advance(3)
        second = rebuilt.retry_transfer(first)
        self.assertEqual(rebuilt.transfer(second).attempt_number, 2)
        self.assertEqual(rebuilt.transfer(second).identity, identity)
        with self.assertRaises(TransferConcurrencyLimit):
            rebuilt.start_transfer(
                fixture_identity("tenant-b-weekly", 1, b"other"), 5
            )

        rebuilt.interrupt_transfer(second)
        terminal_checkpoint = rebuilt.checkpoint()
        terminal_model, terminal_rebuilt = terminal_checkpoint.reconstruct(
            FakeClock(terminal_checkpoint.clock_anchor),
            clock_epoch="fixture-epoch-retry",
        )
        self.assertEqual(terminal_rebuilt.transfer(second).attempt_number, 2)
        self.assertEqual(terminal_rebuilt.transfer(second).state, TransferState.FAILED)
        with self.assertRaises(TransferAttemptLimit):
            terminal_rebuilt.retry_transfer(second)
        self.assertEqual(terminal_model.attempt(second).state, AttemptState.FAILED)

    def test_reconstructed_generation_digest_and_stale_delete_fences_hold(self):
        older_bytes = b"checkpoint generation one"
        newer_bytes = b"checkpoint generation two"
        conflict_bytes = b"conflicting generation two"
        older = fixture_identity("tenant-a-nightly", 1, older_bytes)
        newer = fixture_identity("tenant-a-nightly", 2, newer_bytes)
        conflict = fixture_identity("tenant-a-nightly", 2, conflict_bytes)
        model, clock, simulator = transfer_simulator(
            max_size=128,
            max_attempts=1,
            backoff=(),
            concurrency=2,
            clock_epoch="fixture-epoch-fences",
        )

        older_publisher = simulator.start_transfer(older, len(older_bytes))
        simulator.report_progress(older_publisher, len(older_bytes))
        simulator.finish_transfer(older_publisher, older_bytes)
        newer_publisher = simulator.start_transfer(newer, len(newer_bytes))
        simulator.report_progress(newer_publisher, len(newer_bytes))
        simulator.finish_transfer(newer_publisher, newer_bytes)

        checkpoint = simulator.checkpoint()
        rebuilt_model, rebuilt = checkpoint.reconstruct(
            FakeClock(checkpoint.clock_anchor), clock_epoch="fixture-epoch-fences"
        )
        stale = rebuilt.start_transfer(older, len(older_bytes))
        rebuilt.report_progress(stale, len(older_bytes))
        stale_result = rebuilt.finish_transfer(stale, older_bytes)
        self.assertEqual(stale_result.state, TransferState.STALE)
        self.assertEqual(stale_result.failure_reason, TransferFailureReason.STALE_GENERATION)

        conflict_attempt = rebuilt.start_transfer(conflict, len(conflict_bytes))
        rebuilt.report_progress(conflict_attempt, len(conflict_bytes))
        conflict_result = rebuilt.finish_transfer(
            conflict_attempt, conflict_bytes, cleanup_verified=True
        )
        self.assertEqual(conflict_result.failure_reason, TransferFailureReason.GENERATION_CONFLICT)
        self.assertEqual(conflict_result.state, TransferState.CLEANED)

        with self.assertRaises(StaleAttempt):
            rebuilt_model.reject_stale_delete(older_publisher)
        current = rebuilt_model.current(older.tenant_id, older.backup_id)
        self.assertEqual(current.identity, newer)
        self.assertEqual(current.published_by, newer_publisher)
        self.assertEqual(model.current(older.tenant_id, older.backup_id).identity, newer)
        self.assertNotEqual(current.identity.expected_sha256, conflict.expected_sha256)
        after_stale_delete = rebuilt.checkpoint()
        self.assertEqual(
            after_stale_delete.model_attempts[older_publisher - 1].state,
            AttemptState.STALE,
        )

    def test_checkpoint_validation_rejects_stale_state_without_newer_current_or_size_bound(self):
        short_bytes = b"a"
        newer_bytes = b"b"
        stale_bytes = b"cc"
        older = fixture_identity("tenant-a-nightly", 1, short_bytes)
        newer = fixture_identity("tenant-a-nightly", 2, newer_bytes)
        stale_identity = fixture_identity("tenant-a-nightly", 1, stale_bytes)
        _, _, simulator = transfer_simulator(
            max_size=2,
            max_attempts=1,
            backoff=(),
            clock_epoch="fixture-epoch-stale-validation",
        )

        older_publisher = simulator.start_transfer(older, len(short_bytes))
        simulator.report_progress(older_publisher, len(short_bytes))
        simulator.finish_transfer(older_publisher, short_bytes)
        newer_publisher = simulator.start_transfer(newer, len(newer_bytes))
        simulator.report_progress(newer_publisher, len(newer_bytes))
        simulator.finish_transfer(newer_publisher, newer_bytes)
        stale = simulator.start_transfer(stale_identity, len(stale_bytes))
        simulator.report_progress(stale, len(stale_bytes))
        self.assertEqual(simulator.finish_transfer(stale, stale_bytes).state, TransferState.STALE)
        checkpoint = simulator.checkpoint()

        current_at_old_generation = replace(
            checkpoint.current_artifacts[0], identity=older, published_by=older_publisher
        )
        no_newer_current = replace(
            checkpoint, current_artifacts=(current_at_old_generation,)
        )
        with self.assertRaises(InvalidCheckpoint):
            no_newer_current.reconstruct(
                FakeClock(checkpoint.clock_anchor),
                clock_epoch="fixture-epoch-stale-validation",
            )

        no_current = replace(checkpoint, current_artifacts=())
        with self.assertRaises(InvalidCheckpoint):
            no_current.reconstruct(
                FakeClock(checkpoint.clock_anchor),
                clock_epoch="fixture-epoch-stale-validation",
            )

        smaller_cap = replace(checkpoint.limits, max_size_bytes=1)
        stale_over_size = replace(checkpoint, limits=smaller_cap)
        with self.assertRaises(InvalidCheckpoint):
            stale_over_size.reconstruct(
                FakeClock(checkpoint.clock_anchor),
                clock_epoch="fixture-epoch-stale-validation",
            )

    def test_checkpoint_requires_historical_higher_publication_before_stale_finish(self):
        newer_bytes = b"generation two"
        stale_bytes = b"generation one stale"
        newer_identity = fixture_identity("tenant-a-stale-history", 2, newer_bytes)
        stale_identity = fixture_identity("tenant-a-stale-history", 1, stale_bytes)
        _, clock, simulator = transfer_simulator(
            max_duration=5,
            max_attempts=1,
            backoff=(),
            concurrency=2,
            clock_epoch="fixture-epoch-stale-history-order",
        )

        newer = simulator.start_transfer(newer_identity, len(newer_bytes))
        simulator.report_progress(newer, len(newer_bytes))
        stale = simulator.start_transfer(stale_identity, len(stale_bytes))
        simulator.report_progress(stale, len(stale_bytes))
        clock.advance(2)
        simulator.finish_transfer(newer, newer_bytes)
        clock.advance(1)
        self.assertEqual(
            simulator.finish_transfer(stale, stale_bytes).state, TransferState.STALE
        )

        checkpoint = simulator.checkpoint()
        checkpoint.reconstruct(
            FakeClock(checkpoint.clock_anchor),
            clock_epoch="fixture-epoch-stale-history-order",
        )
        forged_early_stale = replace_transfer_record(
            checkpoint, stale, updated_at=1, finished_at=1
        )
        with self.assertRaisesRegex(
            InvalidCheckpoint,
            "stale transfer has no prior higher-generation verified publication",
        ):
            forged_early_stale.reconstruct(
                FakeClock(checkpoint.clock_anchor),
                clock_epoch="fixture-epoch-stale-history-order",
            )

    def test_checkpoint_accepts_stale_finish_at_publication_tie_before_later_generation(self):
        gen2_bytes = b"generation two"
        gen1_bytes = b"generation one"
        gen3_bytes = b"generation three"
        backup_id = "tenant-a-stale-tie"
        gen2_identity = fixture_identity(backup_id, 2, gen2_bytes)
        gen1_identity = fixture_identity(backup_id, 1, gen1_bytes)
        gen3_identity = fixture_identity(backup_id, 3, gen3_bytes)
        _, clock, simulator = transfer_simulator(
            max_duration=5,
            max_attempts=1,
            backoff=(),
            concurrency=1,
            clock_epoch="fixture-epoch-stale-history-tie",
        )

        gen2 = simulator.start_transfer(gen2_identity, len(gen2_bytes))
        simulator.report_progress(gen2, len(gen2_bytes))
        clock.advance(1)
        simulator.finish_transfer(gen2, gen2_bytes)
        stale = simulator.start_transfer(gen1_identity, len(gen1_bytes))
        simulator.report_progress(stale, len(gen1_bytes))
        self.assertEqual(simulator.finish_transfer(stale, gen1_bytes).state, TransferState.STALE)
        gen3 = simulator.start_transfer(gen3_identity, len(gen3_bytes))
        simulator.report_progress(gen3, len(gen3_bytes))
        clock.advance(1)
        simulator.finish_transfer(gen3, gen3_bytes)

        checkpoint = simulator.checkpoint()
        transfer_records = {
            record.attempt_id: record for record in checkpoint.transfer_attempts
        }
        self.assertEqual(
            transfer_records[gen2].finished_at, transfer_records[stale].finished_at
        )
        self.assertEqual(checkpoint.current_artifacts[0].identity, gen3_identity)
        _, rebuilt = checkpoint.reconstruct(
            FakeClock(checkpoint.clock_anchor), clock_epoch="fixture-epoch-stale-history-tie"
        )
        self.assertEqual(rebuilt.transfer(stale).state, TransferState.STALE)

    def test_checkpoint_validation_rejects_current_older_or_conflicting_with_verified_attempt(self):
        older_bytes = b"generation one"
        newer_bytes = b"generation two"
        older = fixture_identity("tenant-a-monthly", 1, older_bytes)
        newer = fixture_identity("tenant-a-monthly", 2, newer_bytes)
        _, _, simulator = transfer_simulator(
            max_attempts=1,
            backoff=(),
            clock_epoch="fixture-epoch-current-validation",
        )
        older_publisher = simulator.start_transfer(older, len(older_bytes))
        simulator.report_progress(older_publisher, len(older_bytes))
        simulator.finish_transfer(older_publisher, older_bytes)
        newer_publisher = simulator.start_transfer(newer, len(newer_bytes))
        simulator.report_progress(newer_publisher, len(newer_bytes))
        simulator.finish_transfer(newer_publisher, newer_bytes)
        checkpoint = simulator.checkpoint()

        missing_current = replace(checkpoint, current_artifacts=())
        with self.assertRaises(InvalidCheckpoint):
            missing_current.reconstruct(
                FakeClock(checkpoint.clock_anchor),
                clock_epoch="fixture-epoch-current-validation",
            )

        current_at_old_generation = replace(
            checkpoint.current_artifacts[0], identity=older, published_by=older_publisher
        )
        older_current = replace(
            checkpoint, current_artifacts=(current_at_old_generation,)
        )
        with self.assertRaises(InvalidCheckpoint):
            older_current.reconstruct(
                FakeClock(checkpoint.clock_anchor),
                clock_epoch="fixture-epoch-current-validation",
            )

        current_bytes = b"same generation original"
        conflict_bytes = b"same generation conflicting bytes"
        current_identity = fixture_identity("tenant-b-monthly", 7, current_bytes)
        conflicting_identity = fixture_identity(
            current_identity.backup_id, current_identity.generation, conflict_bytes
        )
        _, _, conflict_simulator = transfer_simulator(
            max_attempts=1,
            backoff=(),
            clock_epoch="fixture-epoch-digest-validation",
        )
        current_publisher = conflict_simulator.start_transfer(
            current_identity, len(current_bytes)
        )
        conflict_simulator.report_progress(current_publisher, len(current_bytes))
        conflict_simulator.finish_transfer(current_publisher, current_bytes)
        conflicting_attempt = conflict_simulator.start_transfer(
            conflicting_identity, len(conflict_bytes)
        )
        conflict_simulator.report_progress(conflicting_attempt, len(conflict_bytes))
        conflict_simulator.finish_transfer(
            conflicting_attempt, conflict_bytes, cleanup_verified=True
        )
        conflict_checkpoint = conflict_simulator.checkpoint()

        verified_conflict_attempts = tuple(
            replace(record, state=AttemptState.VERIFIED)
            if record.attempt_id == conflicting_attempt
            else record
            for record in conflict_checkpoint.model_attempts
        )
        verified_conflict_transfers = tuple(
            replace(
                record,
                state=TransferState.VERIFIED,
                failure_reason=None,
                cleanup_verified=None,
            )
            if record.attempt_id == conflicting_attempt
            else record
            for record in conflict_checkpoint.transfer_attempts
        )
        conflicting_verified_history = replace(
            conflict_checkpoint,
            model_attempts=verified_conflict_attempts,
            transfer_attempts=verified_conflict_transfers,
        )
        with self.assertRaises(InvalidCheckpoint):
            conflicting_verified_history.reconstruct(
                FakeClock(conflict_checkpoint.clock_anchor),
                clock_epoch="fixture-epoch-digest-validation",
            )

    def test_checkpoint_rejects_forged_oversize_interruption_and_keeps_legitimate_size_failures(self):
        declared_bytes = b"synthetic declared size"
        declared_identity = fixture_identity("tenant-a-sized", 1, declared_bytes)
        _, _, interrupted_simulator = transfer_simulator(
            max_size=64,
            max_attempts=1,
            backoff=(),
            clock_epoch="fixture-epoch-overlimit-forgery",
        )
        interrupted_attempt = interrupted_simulator.start_transfer(
            declared_identity, len(declared_bytes)
        )
        interrupted_simulator.interrupt_transfer(interrupted_attempt)
        interrupted_checkpoint = interrupted_simulator.checkpoint()
        forged_operation = replace(
            interrupted_checkpoint.operations[0],
            expected_size_bytes=interrupted_checkpoint.limits.max_size_bytes + 1,
        )
        forged_overlimit_interruption = replace(
            interrupted_checkpoint, operations=(forged_operation,)
        )
        with self.assertRaises(InvalidCheckpoint):
            forged_overlimit_interruption.reconstruct(
                FakeClock(interrupted_checkpoint.clock_anchor),
                clock_epoch="fixture-epoch-overlimit-forgery",
            )

        oversize_payload = b"over limit"
        oversize_identity = fixture_identity("tenant-a-sized", 2, oversize_payload)
        _, _, oversize_simulator = transfer_simulator(
            max_size=1,
            max_attempts=2,
            backoff=(2,),
            clock_epoch="fixture-epoch-oversize-start",
        )
        oversize_attempt = oversize_simulator.start_transfer(
            oversize_identity, len(oversize_payload)
        )
        oversize_checkpoint = oversize_simulator.checkpoint()
        oversize_model, rebuilt_oversize = oversize_checkpoint.reconstruct(
            FakeClock(oversize_checkpoint.clock_anchor),
            clock_epoch="fixture-epoch-oversize-start",
        )
        self.assertEqual(rebuilt_oversize.transfer(oversize_attempt).state, TransferState.FAILED)
        self.assertEqual(
            rebuilt_oversize.transfer(oversize_attempt).failure_reason,
            TransferFailureReason.SIZE_LIMIT,
        )
        self.assertEqual(oversize_model.attempt(oversize_attempt).state, AttemptState.FAILED)
        with self.assertRaises(TransferNotRetryable):
            rebuilt_oversize.retry_transfer(oversize_attempt)

        size_limited_content = b"actual bytes exceed the cap"
        size_limited_identity = fixture_identity(
            "tenant-a-sized", 3, size_limited_content
        )
        _, _, size_limited_simulator = transfer_simulator(
            max_size=2,
            max_attempts=1,
            backoff=(),
            clock_epoch="fixture-epoch-actual-size",
        )
        size_limited_attempt = size_limited_simulator.start_transfer(
            size_limited_identity, 2
        )
        size_limited_simulator.report_progress(size_limited_attempt, 2)
        size_limited_simulator.finish_transfer(
            size_limited_attempt, size_limited_content, cleanup_verified=True
        )
        size_limited_checkpoint = size_limited_simulator.checkpoint()
        _, rebuilt_size_limited = size_limited_checkpoint.reconstruct(
            FakeClock(size_limited_checkpoint.clock_anchor),
            clock_epoch="fixture-epoch-actual-size",
        )
        result = rebuilt_size_limited.transfer(size_limited_attempt)
        self.assertEqual(result.state, TransferState.CLEANED)
        self.assertEqual(result.failure_reason, TransferFailureReason.SIZE_LIMIT)

    def test_checkpoint_preserves_interleaved_ids_and_rejects_impossible_allocation_order(self):
        first_content = b"first operation"
        second_content = b"second operation"
        first_identity = fixture_identity("tenant-a-order", 1, first_content)
        second_identity = fixture_identity("tenant-b-order", 1, second_content)
        _, _, simulator = transfer_simulator(
            max_attempts=2,
            backoff=(0,),
            concurrency=2,
            clock_epoch="fixture-epoch-attempt-order",
        )

        first_initial = simulator.start_transfer(first_identity, len(first_content))
        second_initial = simulator.start_transfer(second_identity, len(second_content))
        simulator.report_progress(first_initial, 1)
        simulator.interrupt_transfer(first_initial, cleanup_verified=True)
        simulator.report_progress(second_initial, len(second_content))
        simulator.finish_transfer(second_initial, second_content)
        first_retry = simulator.retry_transfer(first_initial)
        simulator.interrupt_transfer(first_retry)

        checkpoint = simulator.checkpoint()
        self.assertEqual(checkpoint.operations[0].attempt_ids, (first_initial, first_retry))
        self.assertEqual(checkpoint.operations[1].attempt_ids, (second_initial,))
        checkpoint.reconstruct(
            FakeClock(checkpoint.clock_anchor), clock_epoch="fixture-epoch-attempt-order"
        )

        single_identity = fixture_identity(
            "tenant-a-within-order", 1, b"single operation payload"
        )
        _, _, single_simulator = transfer_simulator(
            max_attempts=2,
            backoff=(0,),
            clock_epoch="fixture-epoch-within-operation-order",
        )
        first_attempt = single_simulator.start_transfer(
            single_identity, len(b"single operation payload")
        )
        single_simulator.interrupt_transfer(first_attempt)
        second_attempt = single_simulator.retry_transfer(first_attempt)
        single_simulator.interrupt_transfer(second_attempt)
        single_checkpoint = single_simulator.checkpoint()
        reversed_operation = replace(
            single_checkpoint.operations[0],
            attempt_ids=(second_attempt, first_attempt),
        )
        reversed_attempt_numbers = {
            first_attempt: 2,
            second_attempt: 1,
        }
        reversed_attempt_history = replace(
            single_checkpoint,
            operations=(reversed_operation,),
            transfer_attempts=tuple(
                replace(
                    record,
                    attempt_number=reversed_attempt_numbers[record.attempt_id],
                )
                for record in single_checkpoint.transfer_attempts
            ),
        )
        with self.assertRaisesRegex(
            InvalidCheckpoint, "attempt IDs must increase within each operation"
        ):
            reversed_attempt_history.reconstruct(
                FakeClock(single_checkpoint.clock_anchor),
                clock_epoch="fixture-epoch-within-operation-order",
            )

        first_only_identity = fixture_identity("tenant-a-initial-order", 1, b"first")
        second_only_identity = fixture_identity("tenant-b-initial-order", 1, b"other")
        _, _, allocation_simulator = transfer_simulator(
            max_attempts=1,
            backoff=(),
            concurrency=2,
            clock_epoch="fixture-epoch-initial-order-forgery",
        )
        first_only = allocation_simulator.start_transfer(first_only_identity, len(b"first"))
        allocation_simulator.interrupt_transfer(first_only)
        second_only = allocation_simulator.start_transfer(second_only_identity, len(b"other"))
        allocation_simulator.interrupt_transfer(second_only)
        allocation_checkpoint = allocation_simulator.checkpoint()
        reversed_initial_operations = (
            replace(
                allocation_checkpoint.operations[0],
                identity=second_only_identity,
                attempt_ids=(second_only,),
            ),
            replace(
                allocation_checkpoint.operations[1],
                identity=first_only_identity,
                attempt_ids=(first_only,),
            ),
        )
        reversed_initial_transfers = tuple(
            replace(
                record,
                operation_id=2 if record.attempt_id == first_only else 1,
            )
            for record in allocation_checkpoint.transfer_attempts
        )
        reversed_operation_allocation = replace(
            allocation_checkpoint,
            operations=reversed_initial_operations,
            transfer_attempts=reversed_initial_transfers,
        )
        with self.assertRaisesRegex(InvalidCheckpoint, "initial attempt IDs"):
            reversed_operation_allocation.reconstruct(
                FakeClock(allocation_checkpoint.clock_anchor),
                clock_epoch="fixture-epoch-initial-order-forgery",
            )

    def test_checkpoint_allows_reverse_completion_but_rejects_reversed_start_order(self):
        first_content = b"first concurrent transfer"
        second_content = b"second concurrent transfer"
        first_identity = fixture_identity("tenant-a-concurrent", 1, first_content)
        second_identity = fixture_identity("tenant-b-concurrent", 1, second_content)
        clock = FakeClock(1)
        limits = TransferLimits(
            max_size_bytes=64,
            max_duration_seconds=5,
            max_attempts=1,
            backoff_seconds=(),
            max_concurrent_transfers=2,
        )
        model = Model()
        simulator = TransferSimulator(
            model, limits, clock, clock_epoch="fixture-epoch-start-order"
        )

        first = simulator.start_transfer(first_identity, len(first_content))
        clock.advance(1)
        second = simulator.start_transfer(second_identity, len(second_content))
        simulator.report_progress(second, len(second_content))
        clock.advance(1)
        simulator.finish_transfer(second, second_content)
        clock.advance(1)
        simulator.report_progress(first, len(first_content))
        simulator.finish_transfer(first, first_content)

        checkpoint = simulator.checkpoint()
        self.assertEqual(checkpoint.transfer_attempts[1].finished_at, 3)
        self.assertEqual(checkpoint.transfer_attempts[0].finished_at, 4)
        checkpoint.reconstruct(
            FakeClock(checkpoint.clock_anchor), clock_epoch="fixture-epoch-start-order"
        )

        reversed_start = replace_transfer_record(checkpoint, second, started_at=0.5)
        with self.assertRaises(InvalidCheckpoint):
            reversed_start.reconstruct(
                FakeClock(checkpoint.clock_anchor),
                clock_epoch="fixture-epoch-start-order",
            )

    def test_checkpoint_concurrency_sweep_allows_endpoint_reuse_and_rejects_overlap(self):
        first_content = b"a"
        final_content = b"b"
        first_identity = fixture_identity("tenant-a-endpoint", 1, first_content)
        final_identity = fixture_identity("tenant-b-endpoint", 1, final_content)
        _, clock, simulator = transfer_simulator(
            max_size=1,
            max_duration=5,
            max_attempts=1,
            backoff=(),
            concurrency=1,
            clock_epoch="fixture-epoch-concurrency-endpoint",
        )

        first = simulator.start_transfer(first_identity, len(first_content))
        clock.advance(1)
        oversized = simulator.start_transfer(
            fixture_identity("tenant-c-endpoint", 1, b"too big"), len(b"too big")
        )
        clock.advance(1)
        simulator.interrupt_transfer(first)
        final = simulator.start_transfer(final_identity, len(final_content))
        simulator.report_progress(final, len(final_content))
        clock.advance(2)
        simulator.finish_transfer(final, final_content)

        checkpoint = simulator.checkpoint()
        self.assertEqual(simulator.transfer(oversized).failure_reason, TransferFailureReason.SIZE_LIMIT)
        checkpoint.reconstruct(
            FakeClock(checkpoint.clock_anchor),
            clock_epoch="fixture-epoch-concurrency-endpoint",
        )

        overlapping = replace_transfer_record(checkpoint, final, started_at=1.5)
        with self.assertRaises(InvalidCheckpoint):
            overlapping.reconstruct(
                FakeClock(checkpoint.clock_anchor),
                clock_epoch="fixture-epoch-concurrency-endpoint",
            )

    def test_checkpoint_lifecycle_reasons_require_complete_size_and_digest_state(self):
        expected = b"expected"
        mismatch = b"mismatch"
        expected_identity = fixture_identity("tenant-a-lifecycle", 1, expected)
        _, _, simulator = transfer_simulator(
            max_size=16,
            max_attempts=1,
            backoff=(),
            clock_epoch="fixture-epoch-lifecycle",
        )
        complete_mismatch = simulator.start_transfer(expected_identity, len(expected))
        simulator.report_progress(complete_mismatch, len(expected))
        simulator.finish_transfer(complete_mismatch, mismatch, cleanup_verified=True)

        partial_identity = fixture_identity("tenant-b-lifecycle", 1, b"partial data")
        partial = simulator.start_transfer(partial_identity, len(b"partial data"))
        simulator.report_progress(partial, 2)
        simulator.interrupt_transfer(partial, cleanup_verified=True)
        checkpoint = simulator.checkpoint()
        checkpoint.reconstruct(
            FakeClock(checkpoint.clock_anchor), clock_epoch="fixture-epoch-lifecycle"
        )
        over_cap = replace(
            checkpoint,
            limits=replace(checkpoint.limits, max_size_bytes=len(expected) - 1),
        )
        with self.assertRaises(InvalidCheckpoint):
            over_cap.reconstruct(
                FakeClock(checkpoint.clock_anchor), clock_epoch="fixture-epoch-lifecycle"
            )

        self.assertEqual(
            simulator.transfer(complete_mismatch).failure_reason,
            TransferFailureReason.DIGEST_MISMATCH,
        )
        for reason in (
            TransferFailureReason.STALE_GENERATION,
            TransferFailureReason.DIGEST_MISMATCH,
            TransferFailureReason.GENERATION_CONFLICT,
        ):
            malformed = replace_transfer_record(checkpoint, partial, failure_reason=reason)
            with self.subTest(reason=reason):
                with self.assertRaises(InvalidCheckpoint):
                    malformed.reconstruct(
                        FakeClock(checkpoint.clock_anchor),
                        clock_epoch="fixture-epoch-lifecycle",
                    )

        no_witness_identity = fixture_identity("tenant-c-lifecycle", 3, expected)
        _, _, no_witness_simulator = transfer_simulator(
            max_size=8,
            max_attempts=1,
            backoff=(),
            clock_epoch="fixture-epoch-conflict-no-witness",
        )
        no_witness = no_witness_simulator.start_transfer(
            no_witness_identity, len(expected)
        )
        no_witness_simulator.report_progress(no_witness, len(expected))
        no_witness_simulator.finish_transfer(no_witness, mismatch, cleanup_verified=True)
        no_witness_checkpoint = no_witness_simulator.checkpoint()
        forged_conflict = replace_transfer_record(
            no_witness_checkpoint,
            no_witness,
            failure_reason=TransferFailureReason.GENERATION_CONFLICT,
        )
        with self.assertRaises(InvalidCheckpoint):
            forged_conflict.reconstruct(
                FakeClock(no_witness_checkpoint.clock_anchor),
                clock_epoch="fixture-epoch-conflict-no-witness",
            )

        newer_identity = fixture_identity("tenant-d-lifecycle", 2, expected)
        older_identity = fixture_identity("tenant-d-lifecycle", 1, expected)
        _, _, wrong_generation_simulator = transfer_simulator(
            max_size=8,
            max_attempts=1,
            backoff=(),
            clock_epoch="fixture-epoch-conflict-wrong-generation",
        )
        newer = wrong_generation_simulator.start_transfer(newer_identity, len(expected))
        wrong_generation_simulator.report_progress(newer, len(expected))
        wrong_generation_simulator.finish_transfer(newer, expected)
        candidate = wrong_generation_simulator.start_transfer(
            older_identity, len(expected)
        )
        wrong_generation_simulator.report_progress(candidate, len(expected))
        wrong_generation_simulator.finish_transfer(
            candidate, mismatch, cleanup_verified=True
        )
        wrong_generation_checkpoint = wrong_generation_simulator.checkpoint()
        wrong_generation_conflict = replace_transfer_record(
            wrong_generation_checkpoint,
            candidate,
            failure_reason=TransferFailureReason.GENERATION_CONFLICT,
        )
        with self.assertRaises(InvalidCheckpoint):
            wrong_generation_conflict.reconstruct(
                FakeClock(wrong_generation_checkpoint.clock_anchor),
                clock_epoch="fixture-epoch-conflict-wrong-generation",
            )

    def test_checkpoint_accepts_conflict_witness_after_stale_delete_and_advanced_current(self):
        witness_bytes = b"original generation"
        conflict_bytes = b"conflict generation"
        newer_bytes = b"newer generation"
        witness_identity = fixture_identity("tenant-a-history", 1, witness_bytes)
        conflict_identity = fixture_identity(
            witness_identity.backup_id, witness_identity.generation, conflict_bytes
        )
        newer_identity = fixture_identity(witness_identity.backup_id, 2, newer_bytes)
        model, _, simulator = transfer_simulator(
            max_size=64,
            max_attempts=1,
            backoff=(),
            concurrency=2,
            clock_epoch="fixture-epoch-conflict-history",
        )

        witness = simulator.start_transfer(witness_identity, len(witness_bytes))
        simulator.report_progress(witness, len(witness_bytes))
        simulator.finish_transfer(witness, witness_bytes)
        conflict = simulator.start_transfer(conflict_identity, len(conflict_bytes))
        simulator.report_progress(conflict, len(conflict_bytes))
        simulator.finish_transfer(conflict, conflict_bytes, cleanup_verified=True)
        newer = simulator.start_transfer(newer_identity, len(newer_bytes))
        simulator.report_progress(newer, len(newer_bytes))
        simulator.finish_transfer(newer, newer_bytes)
        with self.assertRaises(StaleAttempt):
            model.reject_stale_delete(witness)

        checkpoint = simulator.checkpoint()
        self.assertEqual(model.attempt(witness).state, AttemptState.STALE)
        self.assertEqual(simulator.transfer(witness).state, TransferState.VERIFIED)
        self.assertEqual(model.current(witness_identity.tenant_id, witness_identity.backup_id).identity, newer_identity)
        checkpoint.reconstruct(
            FakeClock(checkpoint.clock_anchor), clock_epoch="fixture-epoch-conflict-history"
        )

        witness_record = next(
            record for record in checkpoint.transfer_attempts if record.attempt_id == witness
        )
        conflict_record = next(
            record for record in checkpoint.transfer_attempts if record.attempt_id == conflict
        )
        newer_record = next(
            record for record in checkpoint.transfer_attempts if record.attempt_id == newer
        )
        self.assertEqual(witness_record.finished_at, conflict_record.finished_at)
        self.assertEqual(conflict_record.finished_at, newer_record.finished_at)

    def test_checkpoint_rejects_conflict_witness_published_after_conflict(self):
        witness_bytes = b"original"
        conflicting_bytes = b"conflict"
        backup_id = "tenant-a-late-witness"
        witness_identity = fixture_identity(backup_id, 1, witness_bytes)
        conflicting_identity = fixture_identity(backup_id, 1, conflicting_bytes)
        _, _, simulator = transfer_simulator(
            max_size=64,
            max_attempts=1,
            backoff=(),
            clock_epoch="fixture-epoch-late-witness",
        )

        witness = simulator.start_transfer(witness_identity, len(witness_bytes))
        simulator.report_progress(witness, len(witness_bytes))
        simulator.finish_transfer(witness, witness_bytes)
        conflict = simulator.start_transfer(conflicting_identity, len(conflicting_bytes))
        simulator.report_progress(conflict, len(conflicting_bytes))
        simulator.finish_transfer(conflict, conflicting_bytes, cleanup_verified=True)

        checkpoint = simulator.checkpoint()
        self.assertEqual(
            simulator.transfer(conflict).failure_reason,
            TransferFailureReason.GENERATION_CONFLICT,
        )
        checkpoint.reconstruct(
            FakeClock(checkpoint.clock_anchor), clock_epoch="fixture-epoch-late-witness"
        )
        conflict_record = next(
            record for record in checkpoint.transfer_attempts if record.attempt_id == conflict
        )
        later_anchor = checkpoint.clock_anchor + 1
        delayed_witness = replace_transfer_record(
            replace(checkpoint, clock_anchor=later_anchor),
            witness,
            updated_at=later_anchor,
            finished_at=later_anchor,
        )
        witness_record = next(
            record for record in delayed_witness.transfer_attempts if record.attempt_id == witness
        )
        self.assertGreater(witness_record.finished_at, conflict_record.finished_at)
        with self.assertRaisesRegex(
            InvalidCheckpoint, "generation conflict has no prior verified identity witness"
        ):
            delayed_witness.reconstruct(
                FakeClock(later_anchor), clock_epoch="fixture-epoch-late-witness"
            )

    def test_checkpoint_rejects_older_verified_transfer_finishing_after_current_publication(self):
        older_bytes = b"older generation"
        newer_bytes = b"newer generation"
        older_identity = fixture_identity("tenant-a-publication-order", 1, older_bytes)
        newer_identity = fixture_identity("tenant-a-publication-order", 2, newer_bytes)
        _, _, simulator = transfer_simulator(
            max_attempts=1,
            backoff=(),
            concurrency=2,
            clock_epoch="fixture-epoch-publication-order",
        )

        older = simulator.start_transfer(older_identity, len(older_bytes))
        simulator.report_progress(older, len(older_bytes))
        simulator.finish_transfer(older, older_bytes)
        newer = simulator.start_transfer(newer_identity, len(newer_bytes))
        simulator.report_progress(newer, len(newer_bytes))
        simulator.finish_transfer(newer, newer_bytes)
        checkpoint = simulator.checkpoint()
        checkpoint.reconstruct(
            FakeClock(checkpoint.clock_anchor),
            clock_epoch="fixture-epoch-publication-order",
        )

        later_anchor = checkpoint.clock_anchor + 1
        delayed_older_publication = replace_transfer_record(
            replace(checkpoint, clock_anchor=later_anchor),
            older,
            updated_at=later_anchor,
            finished_at=later_anchor,
        )
        with self.assertRaisesRegex(InvalidCheckpoint, "older verified transfer finishes"):
            delayed_older_publication.reconstruct(
                FakeClock(later_anchor), clock_epoch="fixture-epoch-publication-order"
            )

    def test_checkpoint_rejects_conflict_after_higher_generation_published_earlier(self):
        witness_bytes = b"original"
        conflict_bytes = b"conflict"
        mismatch_bytes = b"mismatch"
        newer_bytes = b"new generation"
        backup_id = "tenant-a-conflict-time"
        witness_identity = fixture_identity(backup_id, 1, witness_bytes)
        conflict_identity = fixture_identity(backup_id, 1, conflict_bytes)
        newer_identity = fixture_identity(backup_id, 2, newer_bytes)
        _, clock, simulator = transfer_simulator(
            max_size=64,
            max_duration=5,
            max_attempts=1,
            backoff=(),
            concurrency=3,
            clock_epoch="fixture-epoch-conflict-time",
        )

        witness = simulator.start_transfer(witness_identity, len(witness_bytes))
        simulator.report_progress(witness, len(witness_bytes))
        simulator.finish_transfer(witness, witness_bytes)
        conflict = simulator.start_transfer(conflict_identity, len(conflict_bytes))
        newer = simulator.start_transfer(newer_identity, len(newer_bytes))
        simulator.report_progress(newer, len(newer_bytes))
        clock.advance(1)
        simulator.finish_transfer(newer, newer_bytes)
        clock.advance(1)
        simulator.report_progress(conflict, len(conflict_bytes))
        simulator.finish_transfer(conflict, mismatch_bytes, cleanup_verified=True)

        checkpoint = simulator.checkpoint()
        checkpoint.reconstruct(
            FakeClock(checkpoint.clock_anchor), clock_epoch="fixture-epoch-conflict-time"
        )
        forged_conflict = replace_transfer_record(
            checkpoint,
            conflict,
            failure_reason=TransferFailureReason.GENERATION_CONFLICT,
        )
        with self.assertRaises(InvalidCheckpoint):
            forged_conflict.reconstruct(
                FakeClock(checkpoint.clock_anchor),
                clock_epoch="fixture-epoch-conflict-time",
            )

    def test_checkpoint_rejects_conflicting_successes_after_both_models_become_stale(self):
        original_bytes = b"original digest"
        conflicting_bytes = b"conflicting digest"
        newer_bytes = b"newer generation"
        backup_id = "tenant-a-stale-digests"
        original_identity = fixture_identity(backup_id, 1, original_bytes)
        conflicting_identity = fixture_identity(backup_id, 1, conflicting_bytes)
        newer_identity = fixture_identity(backup_id, 2, newer_bytes)
        model, clock, simulator = transfer_simulator(
            max_attempts=1, backoff=(), clock_epoch="fixture-epoch-stale-digests"
        )
        original = simulator.start_transfer(original_identity, len(original_bytes))
        simulator.report_progress(original, len(original_bytes))
        simulator.finish_transfer(original, original_bytes)
        conflicting = simulator.start_transfer(conflicting_identity, len(conflicting_bytes))
        simulator.report_progress(conflicting, len(conflicting_bytes))
        clock.advance(1)
        simulator.finish_transfer(conflicting, conflicting_bytes, cleanup_verified=True)
        newer = simulator.start_transfer(newer_identity, len(newer_bytes))
        simulator.report_progress(newer, len(newer_bytes))
        clock.advance(1)
        simulator.finish_transfer(newer, newer_bytes)
        with self.assertRaises(StaleAttempt):
            model.reject_stale_delete(original)
        checkpoint = simulator.checkpoint()
        checkpoint.reconstruct(
            FakeClock(checkpoint.clock_anchor), clock_epoch="fixture-epoch-stale-digests"
        )

        forged = replace_transfer_record(
            checkpoint, conflicting, state=TransferState.VERIFIED,
            failure_reason=None, cleanup_verified=None,
        )
        forged = replace(
            forged,
            model_attempts=tuple(
                replace(record, state=AttemptState.STALE)
                if record.attempt_id == conflicting else record
                for record in forged.model_attempts
            ),
        )
        with self.assertRaisesRegex(InvalidCheckpoint, "verified generation digest"):
            forged.reconstruct(
                FakeClock(checkpoint.clock_anchor), clock_epoch="fixture-epoch-stale-digests"
            )

    def test_checkpoint_rejects_later_duplicate_success_as_original_publisher(self):
        content = b"duplicate identity"
        identity = fixture_identity("tenant-a-publisher", 1, content)
        model, clock, simulator = transfer_simulator(
            max_attempts=1, backoff=(), clock_epoch="fixture-epoch-publisher"
        )
        first = simulator.start_transfer(identity, len(content))
        simulator.report_progress(first, len(content))
        clock.advance(1)
        simulator.finish_transfer(first, content)
        second = simulator.start_transfer(identity, len(content))
        simulator.report_progress(second, len(content))
        clock.advance(1)
        simulator.finish_transfer(second, content)
        self.assertEqual(model.current(identity.tenant_id, identity.backup_id).published_by, first)
        checkpoint = simulator.checkpoint()
        checkpoint.reconstruct(
            FakeClock(checkpoint.clock_anchor), clock_epoch="fixture-epoch-publisher"
        )
        forged = replace(
            checkpoint,
            current_artifacts=(replace(checkpoint.current_artifacts[0], published_by=second),),
        )
        with self.assertRaisesRegex(InvalidCheckpoint, "earlier verified publication"):
            forged.reconstruct(
                FakeClock(checkpoint.clock_anchor), clock_epoch="fixture-epoch-publisher"
            )

    def test_checkpoint_accepts_equal_time_publishers_in_either_completion_order(self):
        content = b"equal time identity"
        identity = fixture_identity("tenant-a-publisher-tie", 1, content)
        for reverse in (False, True):
            with self.subTest(reverse=reverse):
                model, clock, simulator = transfer_simulator(
                    max_attempts=1, backoff=(), concurrency=2,
                    clock_epoch="fixture-epoch-publisher-tie",
                )
                first = simulator.start_transfer(identity, len(content))
                second = simulator.start_transfer(identity, len(content))
                order = (second, first) if reverse else (first, second)
                clock.advance(1)
                for attempt_id in order:
                    simulator.report_progress(attempt_id, len(content))
                    simulator.finish_transfer(attempt_id, content)
                self.assertEqual(model.current(identity.tenant_id, identity.backup_id).published_by, order[0])
                checkpoint = simulator.checkpoint()
                reconstructed, _ = checkpoint.reconstruct(
                    FakeClock(checkpoint.clock_anchor), clock_epoch="fixture-epoch-publisher-tie"
                )
                self.assertEqual(reconstructed.current(identity.tenant_id, identity.backup_id).published_by, order[0])
                # Completion timestamps and start IDs cannot distinguish these histories.
                ambiguous = replace(
                    checkpoint,
                    current_artifacts=(
                        replace(checkpoint.current_artifacts[0], published_by=order[1]),
                    ),
                )
                ambiguous.reconstruct(
                    FakeClock(checkpoint.clock_anchor), clock_epoch="fixture-epoch-publisher-tie"
                )

    def test_checkpoint_is_independent_and_uses_saved_counter_and_clock_epoch(self):
        content = b"synthetic checkpoint independence"
        identity = fixture_identity("tenant-a-nightly", 5, content)
        model, _, simulator = transfer_simulator(
            max_attempts=1,
            backoff=(),
            clock_epoch="fixture-epoch-independent",
        )
        first = simulator.start_transfer(identity, len(content))
        simulator.report_progress(first, len(content))
        simulator.finish_transfer(first, content)

        checkpoint = simulator.checkpoint()
        original_current = model.current(identity.tenant_id, identity.backup_id)
        with self.assertRaises(InvalidCheckpoint):
            checkpoint.reconstruct(
                FakeClock(checkpoint.clock_anchor), clock_epoch="another-epoch"
            )
        with self.assertRaises(InvalidCheckpoint):
            checkpoint.reconstruct(
                FakeClock(checkpoint.clock_anchor + 1),
                clock_epoch="fixture-epoch-independent",
            )

        rebuilt_model, rebuilt = checkpoint.reconstruct(
            FakeClock(checkpoint.clock_anchor),
            clock_epoch="fixture-epoch-independent",
        )
        next_attempt = rebuilt.start_transfer(
            fixture_identity("tenant-b-nightly", 1, b"new synthetic operation"),
            len(b"new synthetic operation"),
        )
        self.assertGreater(next_attempt, first)
        self.assertEqual(model.current(identity.tenant_id, identity.backup_id), original_current)
        with self.assertRaises(UnknownAttempt):
            model.attempt(next_attempt)
        self.assertEqual(rebuilt_model.current(identity.tenant_id, identity.backup_id), original_current)

    def test_reconstructed_checkpoint_clock_failures_leave_terminal_state_unchanged(self):
        content = b"synthetic terminal checkpoint clock failures"
        identity = fixture_identity("tenant-a-clock-anchor", 1, content)
        _, _, simulator = transfer_simulator(
            max_attempts=1,
            backoff=(),
            clock_epoch="fixture-epoch-clock-failures",
        )
        attempt = simulator.start_transfer(identity, len(content))
        simulator.report_progress(attempt, len(content))
        simulator.finish_transfer(attempt, content)

        checkpoint = simulator.checkpoint()
        rebuilt_clock = FakeClock(checkpoint.clock_anchor)
        _, rebuilt = checkpoint.reconstruct(
            rebuilt_clock, clock_epoch="fixture-epoch-clock-failures"
        )
        next_identity = fixture_identity(
            "tenant-b-clock-anchor", 1, b"new synthetic operation"
        )

        rebuilt_clock.value = checkpoint.clock_anchor - 1
        with self.assertRaises(InvalidTransferClock):
            rebuilt.start_transfer(next_identity, len(b"new synthetic operation"))

        rebuilt_clock.value = float("inf")
        with self.assertRaises(InvalidTransferClock):
            rebuilt.start_transfer(next_identity, len(b"new synthetic operation"))

        rebuilt_clock.value = checkpoint.clock_anchor
        self.assertEqual(rebuilt.checkpoint(), checkpoint)

    def test_checkpoint_rejects_active_attempts_and_missing_clock_epoch(self):
        content = b"synthetic active checkpoint rejection"
        identity = fixture_identity("tenant-a-nightly", 2, content)
        _, _, active_simulator = transfer_simulator(clock_epoch="fixture-epoch-active")
        active_simulator.start_transfer(identity, len(content))
        with self.assertRaises(InvalidCheckpoint):
            active_simulator.checkpoint()

        _, _, untagged_simulator = transfer_simulator()
        untagged_simulator.start_transfer(identity, len(content))
        untagged_simulator.interrupt_transfer(1)
        with self.assertRaises(InvalidCheckpoint):
            untagged_simulator.checkpoint()

        _, _, consistent_simulator = transfer_simulator(
            max_attempts=1,
            backoff=(),
            clock_epoch="fixture-epoch-consistency",
        )
        consistent_attempt = consistent_simulator.start_transfer(identity, len(content))
        consistent_simulator.interrupt_transfer(consistent_attempt)
        checkpoint = consistent_simulator.checkpoint()
        inconsistent = replace(checkpoint, next_attempt_id=checkpoint.next_attempt_id + 1)
        with self.assertRaises(InvalidCheckpoint):
            inconsistent.reconstruct(
                FakeClock(checkpoint.clock_anchor),
                clock_epoch="fixture-epoch-consistency",
            )

        unrepresentable_anchor = replace(checkpoint, clock_anchor=10**1000)
        with self.assertRaises(InvalidCheckpoint):
            unrepresentable_anchor.reconstruct(
                FakeClock(checkpoint.clock_anchor),
                clock_epoch="fixture-epoch-consistency",
            )


if __name__ == "__main__":
    unittest.main()
