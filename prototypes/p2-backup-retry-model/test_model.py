import unittest
from dataclasses import FrozenInstanceError

from model import (
    AttemptState,
    AttemptTerminal,
    DeleteOutOfScope,
    DigestMismatch,
    GenerationConflict,
    Identity,
    InvalidIdentity,
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
    def __init__(self):
        self.value = 0.0

    def __call__(self):
        return self.value

    def advance(self, seconds):
        self.value += seconds


def transfer_simulator(
    *, max_size=64, max_duration=5, max_attempts=2, backoff=(2,), concurrency=1
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
    return model, clock, TransferSimulator(model, limits, clock)


class BackupRetryModelTests(unittest.TestCase):
    def test_newer_generation_fences_stale_publish_and_delete(self):
        model = Model()
        older_bytes = b"synthetic generation one"
        newer_bytes = b"synthetic generation two"
        older = identity_for_content("tenant-a-nightly", 1, older_bytes)
        newer = identity_for_content("tenant-a-nightly", 2, newer_bytes)

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

        current = model.current(older.backup_id)
        self.assertIsNotNone(current)
        self.assertEqual(current.identity, newer)
        self.assertEqual(current.published_by, newer_publisher)
        with self.assertRaises(AttemptTerminal):
            model.verify_and_publish(stale_resume, older_bytes)
        with self.assertRaises(AttemptTerminal):
            model.reject_stale_delete(older_publisher)
        self.assertEqual(model.current(older.backup_id), current)

    def test_non_stale_delete_guard_is_out_of_scope_without_mutation(self):
        model = Model()
        content = b"synthetic verified artifact"
        identity = identity_for_content("tenant-a-nightly", 1, content)
        publisher = model.begin_attempt(identity)
        model.verify_and_publish(publisher, content)

        state_before = model.attempt(publisher).state
        current_before = model.current(identity.backup_id)
        with self.assertRaises(DeleteOutOfScope):
            model.reject_stale_delete(publisher)

        self.assertEqual(model.attempt(publisher).state, state_before)
        self.assertEqual(model.current(identity.backup_id), current_before)

    def test_digest_mismatch_fails_without_changing_current(self):
        model = Model()
        first_bytes = b"synthetic verified artifact"
        first = identity_for_content("tenant-b-nightly", 7, first_bytes)
        first_attempt = model.begin_attempt(first)
        model.verify_and_publish(first_attempt, first_bytes)

        expected_bytes = b"next expected artifact"
        next_identity = identity_for_content("tenant-b-nightly", 8, expected_bytes)
        next_attempt = model.begin_attempt(next_identity)
        with self.assertRaises(DigestMismatch):
            model.verify_and_publish(next_attempt, b"corrupted synthetic artifact")
        self.assertEqual(model.attempt(next_attempt).state, AttemptState.FAILED)
        self.assertEqual(model.current(first.backup_id).identity, first)
        self.assertEqual(model.current(first.backup_id).published_by, first_attempt)

    def test_attempt_identity_is_immutable_after_begin(self):
        model = Model()
        content = b"identity-bound synthetic bytes"
        identity = identity_for_content("tenant-a-weekly", 4, content)
        attempt_id = model.begin_attempt(identity)

        with self.assertRaises(FrozenInstanceError):
            identity.generation = 99
        model.verify_and_publish(attempt_id, content)

        attempt = model.attempt(attempt_id)
        self.assertEqual(attempt.identity, identity)
        self.assertEqual(attempt.state, AttemptState.VERIFIED)
        self.assertEqual(model.current(identity.backup_id).identity, identity)

    def test_boolean_attempt_id_cannot_alias_integer_attempt(self):
        model = Model()
        content = b"strict synthetic attempt id"
        identity = identity_for_content("tenant-a-weekly", 4, content)
        attempt_id = model.begin_attempt(identity)
        self.assertEqual(attempt_id, 1)

        with self.assertRaises(UnknownAttempt):
            model.attempt(True)
        with self.assertRaises(UnknownAttempt):
            model.verify_and_publish(True, content)
        with self.assertRaises(UnknownAttempt):
            model.reject_stale_delete(True)

        self.assertEqual(model.attempt(attempt_id).state, AttemptState.ACTIVE)
        self.assertIsNone(model.current(identity.backup_id))

    def test_same_generation_cannot_change_digest(self):
        model = Model()
        first_bytes = b"generation content"
        first = identity_for_content("tenant-a-monthly", 3, first_bytes)
        first_attempt = model.begin_attempt(first)
        model.verify_and_publish(first_attempt, first_bytes)

        conflicting_bytes = b"different content"
        conflicting = identity_for_content(first.backup_id, first.generation, conflicting_bytes)
        conflict_attempt = model.begin_attempt(conflicting)
        with self.assertRaises(GenerationConflict):
            model.verify_and_publish(conflict_attempt, conflicting_bytes)
        self.assertEqual(model.attempt(conflict_attempt).state, AttemptState.FAILED)
        self.assertEqual(model.current(first.backup_id).identity, first)
        self.assertEqual(model.current(first.backup_id).published_by, first_attempt)

    def test_identity_rejects_invalid_generation_and_digest(self):
        with self.assertRaises(InvalidIdentity):
            identity_for_content(42, 1, b"synthetic")
        with self.assertRaises(InvalidIdentity):
            identity_for_content("tenant-a", True, b"synthetic")
        with self.assertRaises(InvalidIdentity):
            Identity("tenant-a", 1, "not-a-digest")
        for digest in (None, bytes(64)):
            with self.subTest(digest_type=type(digest).__name__):
                with self.assertRaises(InvalidIdentity):
                    Identity("tenant-a", 1, digest)

    def test_transfer_limits_require_explicit_bounds_and_backoff_entries(self):
        with self.assertRaises(InvalidTransferPolicy):
            TransferLimits(64, 5, 2, (), 1)
        with self.assertRaises(InvalidTransferPolicy):
            TransferLimits(64, 5, True, (), 1)

    def test_oversize_transfer_is_terminal_before_any_upload(self):
        model, _, simulator = transfer_simulator(max_size=3)
        content = b"four"
        identity = identity_for_content("tenant-a-nightly", 1, content)

        attempt = simulator.start_transfer(identity, len(content))
        result = simulator.transfer(attempt)

        self.assertEqual(result.state, TransferState.FAILED)
        self.assertEqual(result.failure_reason, TransferFailureReason.SIZE_LIMIT)
        self.assertEqual(result.transferred_bytes, 0)
        self.assertEqual(model.attempt(attempt).state, AttemptState.FAILED)
        self.assertIsNone(model.current(identity.backup_id))
        with self.assertRaises(TransferNotRetryable):
            simulator.retry_transfer(attempt)

    def test_transfer_publishes_only_after_full_digest_verification(self):
        model, clock, simulator = transfer_simulator(max_attempts=1, backoff=())
        content = b"synthetic verified bytes"
        identity = identity_for_content("tenant-a-nightly", 4, content)

        attempt = simulator.start_transfer(identity, len(content))
        with self.assertRaises(TransferConcurrencyLimit):
            simulator.start_transfer(
                identity_for_content("tenant-b-nightly", 1, b"other"), len(b"other")
            )
        simulator.report_progress(attempt, len(content))
        clock.advance(1)
        result = simulator.finish_transfer(attempt, content)

        self.assertEqual(result.state, TransferState.VERIFIED)
        self.assertIsNone(result.failure_reason)
        self.assertEqual(result.duration_seconds, 1)
        self.assertEqual(model.attempt(attempt).state, AttemptState.VERIFIED)
        self.assertEqual(model.current(identity.backup_id).identity, identity)

    def test_transfer_at_inclusive_size_and_duration_limits_can_verify(self):
        content = b"exact-bound synthetic payload"
        identity = identity_for_content("tenant-a-nightly", 4, content)
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
        self.assertEqual(model.current(identity.backup_id).identity, identity)

    def test_identity_handoff_to_fresh_simulator_only(self):
        content = b"synthetic identity handoff bytes"
        identity = identity_for_content("tenant-a-nightly", 6, content)
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
        self.assertEqual(fresh_model.current(identity.backup_id).identity, identity)

    def test_partial_upload_requires_verified_cleanup_or_reconciliation(self):
        content = b"synthetic partial object"
        identity = identity_for_content("tenant-a-nightly", 1, content)

        model, _, simulator = transfer_simulator()
        cleaned_attempt = simulator.start_transfer(identity, len(content))
        simulator.report_progress(cleaned_attempt, 5)
        cleaned = simulator.interrupt_transfer(cleaned_attempt, cleanup_verified=True)
        self.assertEqual(cleaned.state, TransferState.CLEANED)
        self.assertTrue(cleaned.cleanup_verified)
        self.assertEqual(model.attempt(cleaned_attempt).state, AttemptState.FAILED)
        self.assertIsNone(model.current(identity.backup_id))

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
        self.assertIsNone(unresolved_model.current(identity.backup_id))
        with self.assertRaises(TransferReconciliationPending):
            unresolved_simulator.retry_transfer(unresolved_attempt)

    def test_retry_uses_same_identity_and_enforces_backoff_and_attempt_cap(self):
        model, clock, simulator = transfer_simulator(max_attempts=2, backoff=(2,))
        content = b"synthetic retry bytes"
        identity = identity_for_content("tenant-a-nightly", 3, content)
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
        self.assertIsNone(model.current(identity.backup_id))

    def test_duration_limit_with_partial_bytes_requires_reconciliation(self):
        model, clock, simulator = transfer_simulator(max_duration=2)
        content = b"synthetic timeout payload"
        identity = identity_for_content("tenant-a-nightly", 2, content)
        attempt = simulator.start_transfer(identity, len(content))
        clock.advance(3)

        result = simulator.report_progress(attempt, 4)

        self.assertEqual(result.state, TransferState.RECONCILIATION_REQUIRED)
        self.assertEqual(result.failure_reason, TransferFailureReason.DURATION_LIMIT)
        self.assertEqual(result.duration_seconds, 3)
        self.assertEqual(model.attempt(attempt).state, AttemptState.RECONCILIATION_REQUIRED)
        self.assertIsNone(model.current(identity.backup_id))

    def test_digest_mismatch_cannot_be_reported_as_verified(self):
        model, _, simulator = transfer_simulator()
        expected = b"same-size expected bytes"
        corrupted = b"same-size mismatch bytes"
        identity = identity_for_content("tenant-a-nightly", 5, expected)
        attempt = simulator.start_transfer(identity, len(expected))
        simulator.report_progress(attempt, len(expected))

        result = simulator.finish_transfer(attempt, corrupted, cleanup_verified=True)

        self.assertEqual(result.state, TransferState.CLEANED)
        self.assertEqual(result.failure_reason, TransferFailureReason.DIGEST_MISMATCH)
        self.assertTrue(result.cleanup_verified)
        self.assertEqual(model.attempt(attempt).state, AttemptState.FAILED)
        self.assertIsNone(model.current(identity.backup_id))

    def test_stale_transfer_cannot_replace_a_newer_verified_generation(self):
        model, _, simulator = transfer_simulator(max_attempts=1, backoff=(), concurrency=2)
        older_bytes = b"synthetic transfer generation one"
        newer_bytes = b"synthetic transfer generation two"
        older = identity_for_content("tenant-a-nightly", 1, older_bytes)
        newer = identity_for_content("tenant-a-nightly", 2, newer_bytes)

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
        self.assertEqual(model.current(older.backup_id).identity, newer)


if __name__ == "__main__":
    unittest.main()
