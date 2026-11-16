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
    Model,
    StaleAttempt,
    identity_for_content,
)


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


if __name__ == "__main__":
    unittest.main()
