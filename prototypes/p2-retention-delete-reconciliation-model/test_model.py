import unittest
from dataclasses import FrozenInstanceError

from model import (
    DeleteIntentDenied,
    DeleteObservation,
    IntentState,
    ObservedPresence,
    RetentionDeleteModel,
    VerifiedArtifact,
)


class RetentionDeleteReconciliationTests(unittest.TestCase):
    def setUp(self):
        self.model = RetentionDeleteModel()
        self.artifact = VerifiedArtifact("tenant-a", "artifact-a")
        self.intent = self.model.request_delete("operation-a", self.artifact)

    def assert_state_unchanged_after_denial(self, operation):
        before = self.model.intents
        with self.assertRaises(DeleteIntentDenied) as denied:
            operation()
        self.assertEqual(str(denied.exception), "delete reconciliation denied")
        self.assertEqual(self.model.intents, before)

    def test_request_creates_intent_bound_to_one_immutable_artifact(self):
        self.assertEqual(self.intent.operation_id, "operation-a")
        self.assertIs(self.intent.artifact, self.artifact)
        self.assertEqual(self.intent.state, IntentState.REQUESTED)
        self.assertIsNone(self.intent.last_observation)

    def test_exact_request_replay_returns_same_intent_without_resetting_state(self):
        replay = self.model.request_delete("operation-a", self.artifact)
        self.assertIs(replay, self.intent)
        self.assertEqual(self.model.intents, (self.intent,))

        unresolved = self.model.observe(
            "operation-a",
            self.artifact,
            DeleteObservation(self.artifact, ObservedPresence.UNKNOWN),
        )
        replay_after_uncertainty = self.model.request_delete("operation-a", self.artifact)

        self.assertEqual(unresolved.state, IntentState.RECONCILIATION_REQUIRED)
        self.assertIs(replay_after_uncertainty, unresolved)
        self.assertEqual(replay_after_uncertainty.state, IntentState.RECONCILIATION_REQUIRED)

    def test_operation_id_reuse_for_other_artifact_or_tenant_is_denied(self):
        other_artifacts = (
            VerifiedArtifact("tenant-a", "artifact-b"),
            VerifiedArtifact("tenant-b", "artifact-a"),
        )
        for other in other_artifacts:
            with self.subTest(other=other):
                self.assert_state_unchanged_after_denial(
                    lambda other=other: self.model.request_delete("operation-a", other)
                )

    def test_wrong_tenant_or_artifact_observation_is_denied_without_retargeting(self):
        unresolved = self.model.observe(
            "operation-a",
            self.artifact,
            DeleteObservation(self.artifact, ObservedPresence.UNKNOWN),
        )
        self.assertEqual(unresolved.state, IntentState.RECONCILIATION_REQUIRED)
        wrong_targets = (
            VerifiedArtifact("tenant-b", "artifact-a"),
            VerifiedArtifact("tenant-a", "artifact-b"),
        )
        for wrong_target in wrong_targets:
            with self.subTest(target=wrong_target):
                evidence = DeleteObservation(wrong_target, ObservedPresence.VERIFIED_ABSENT)
                self.assert_state_unchanged_after_denial(
                    lambda evidence=evidence: self.model.observe(
                        "operation-a", self.artifact, evidence
                    )
                )
        current = self.model.intents[0]
        self.assertIs(current.artifact, self.artifact)
        self.assertIs(current, unresolved)
        self.assertEqual(current.state, IntentState.RECONCILIATION_REQUIRED)

    def test_present_unknown_and_unavailable_observations_require_reconciliation(self):
        for presence in (
            ObservedPresence.PRESENT,
            ObservedPresence.UNKNOWN,
            ObservedPresence.UNAVAILABLE,
        ):
            with self.subTest(presence=presence):
                model = RetentionDeleteModel()
                artifact = VerifiedArtifact("tenant-a", "artifact-a")
                model.request_delete("operation-a", artifact)

                result = model.observe(
                    "operation-a", artifact, DeleteObservation(artifact, presence)
                )

                self.assertEqual(result.state, IntentState.RECONCILIATION_REQUIRED)
                self.assertIs(result.artifact, artifact)
                self.assertEqual(result.last_observation.presence, presence)

    def test_unresolved_state_survives_replay_and_another_uncertain_observation(self):
        unresolved = self.model.observe(
            "operation-a",
            self.artifact,
            DeleteObservation(self.artifact, ObservedPresence.UNAVAILABLE),
        )
        replay = self.model.request_delete("operation-a", self.artifact)
        self.assertIs(replay, unresolved)

        still_unresolved = self.model.observe(
            "operation-a",
            self.artifact,
            DeleteObservation(self.artifact, ObservedPresence.PRESENT),
        )

        self.assertEqual(still_unresolved.state, IntentState.RECONCILIATION_REQUIRED)
        self.assertIs(still_unresolved.artifact, self.artifact)
        self.assertEqual(still_unresolved.last_observation.presence, ObservedPresence.PRESENT)

    def test_exact_absence_resolves_each_prior_uncertain_observation(self):
        for prior_presence in (
            ObservedPresence.PRESENT,
            ObservedPresence.UNKNOWN,
            ObservedPresence.UNAVAILABLE,
        ):
            with self.subTest(prior_presence=prior_presence):
                model = RetentionDeleteModel()
                artifact = VerifiedArtifact("tenant-a", "artifact-a")
                requested = model.request_delete("operation-a", artifact)
                unresolved = model.observe(
                    "operation-a",
                    artifact,
                    DeleteObservation(artifact, prior_presence),
                )
                self.assertEqual(unresolved.state, IntentState.RECONCILIATION_REQUIRED)

                completed = model.observe(
                    "operation-a",
                    artifact,
                    DeleteObservation(artifact, ObservedPresence.VERIFIED_ABSENT),
                )

                self.assertEqual(completed.state, IntentState.VERIFIED_ABSENT)
                self.assertEqual(completed.operation_id, requested.operation_id)
                self.assertIs(completed.artifact, artifact)
                self.assertEqual(completed.last_observation.presence, ObservedPresence.VERIFIED_ABSENT)
                self.assertIs(model.intents[0], completed)

    def test_only_exact_verified_absence_marks_intent_complete(self):
        for presence in (
            ObservedPresence.PRESENT,
            ObservedPresence.UNKNOWN,
            ObservedPresence.UNAVAILABLE,
        ):
            with self.subTest(presence=presence):
                model = RetentionDeleteModel()
                artifact = VerifiedArtifact("tenant-a", "artifact-a")
                model.request_delete("operation-a", artifact)
                result = model.observe(
                    "operation-a", artifact, DeleteObservation(artifact, presence)
                )
                self.assertNotEqual(result.state, IntentState.VERIFIED_ABSENT)

        completed = self.model.observe(
            "operation-a",
            self.artifact,
            DeleteObservation(self.artifact, ObservedPresence.VERIFIED_ABSENT),
        )
        self.assertEqual(completed.state, IntentState.VERIFIED_ABSENT)
        self.assertIs(completed.artifact, self.artifact)
        self.assertEqual(completed.last_observation.presence, ObservedPresence.VERIFIED_ABSENT)
        self.assertIs(
            self.model.observe(
                "operation-a",
                self.artifact,
                DeleteObservation(self.artifact, ObservedPresence.VERIFIED_ABSENT),
            ),
            completed,
        )

    def test_completed_intent_cannot_be_reopened_by_contradictory_observation(self):
        completed = self.model.observe(
            "operation-a",
            self.artifact,
            DeleteObservation(self.artifact, ObservedPresence.VERIFIED_ABSENT),
        )

        self.assert_state_unchanged_after_denial(
            lambda: self.model.observe(
                "operation-a",
                self.artifact,
                DeleteObservation(self.artifact, ObservedPresence.PRESENT),
            )
        )
        self.assertIs(self.model.intents[0], completed)

    def test_intent_target_and_observation_identity_are_immutable(self):
        observation = DeleteObservation(self.artifact, ObservedPresence.UNKNOWN)

        for record, fields in (
            (self.artifact, ("tenant_id", "artifact_id")),
            (self.intent, ("operation_id", "artifact", "state")),
            (observation, ("artifact", "presence")),
        ):
            for field in fields:
                with self.subTest(record=type(record).__name__, field=field):
                    with self.assertRaises(FrozenInstanceError):
                        setattr(record, field, "replacement")

        self.assertEqual(self.model.intents, (self.intent,))

    def test_invalid_identity_and_unknown_intent_denials_do_not_mutate_state(self):
        bad_values = (None, "", 17, True, b"id")
        for value in bad_values:
            with self.subTest(value=value):
                self.assert_state_unchanged_after_denial(
                    lambda value=value: self.model.request_delete(value, self.artifact)
                )
                self.assert_state_unchanged_after_denial(
                    lambda value=value: self.model.request_delete("operation-b", value)
                )
                self.assert_state_unchanged_after_denial(
                    lambda value=value: self.model.observe(
                        value,
                        self.artifact,
                        DeleteObservation(self.artifact, ObservedPresence.VERIFIED_ABSENT),
                    )
                )

        self.assert_state_unchanged_after_denial(
            lambda: self.model.observe(
                "operation-missing",
                self.artifact,
                DeleteObservation(self.artifact, ObservedPresence.VERIFIED_ABSENT),
            )
        )

    def test_malformed_or_wrong_type_observations_are_denied_without_mutation(self):
        invalid_observations = (
            None,
            object(),
            DeleteObservation(object(), ObservedPresence.VERIFIED_ABSENT),
            DeleteObservation(VerifiedArtifact("", "artifact-a"), ObservedPresence.VERIFIED_ABSENT),
            DeleteObservation(VerifiedArtifact("tenant-a", ""), ObservedPresence.VERIFIED_ABSENT),
            DeleteObservation(self.artifact, "verified_absent"),
            DeleteObservation(self.artifact, None),
            DeleteObservation(self.artifact, True),
        )
        for observation in invalid_observations:
            with self.subTest(observation=observation):
                self.assert_state_unchanged_after_denial(
                    lambda observation=observation: self.model.observe(
                        "operation-a", self.artifact, observation
                    )
                )


class RetentionDeleteBatchReconciliationTests(unittest.TestCase):
    def setUp(self):
        self.model = RetentionDeleteModel()
        self.artifacts = (
            VerifiedArtifact("tenant-a", "artifact-a"),
            VerifiedArtifact("tenant-a", "artifact-b"),
            VerifiedArtifact("tenant-b", "artifact-a"),
        )
        self.intent = self.model.request_delete_batch("operation-batch", self.artifacts)

    def observations(self, *presences):
        return tuple(
            DeleteObservation(artifact, presence)
            for artifact, presence in zip(self.artifacts, presences)
        )

    def assert_batch_state_unchanged_after_denial(self, operation):
        before = self.model.batch_intents
        with self.assertRaises(DeleteIntentDenied) as denied:
            operation()
        self.assertEqual(str(denied.exception), "delete reconciliation denied")
        self.assertEqual(self.model.batch_intents, before)

    def test_batch_request_requires_nonempty_unique_immutable_preselected_tuple(self):
        invalid_sets = (
            (),
            [self.artifacts[0]],
            (self.artifacts[0], self.artifacts[0]),
            (VerifiedArtifact("tenant-a", ""),),
            (object(),),
        )
        model = RetentionDeleteModel()
        for artifacts in invalid_sets:
            with self.subTest(artifacts=artifacts):
                before = model.batch_intents
                with self.assertRaises(DeleteIntentDenied):
                    model.request_delete_batch("operation-batch", artifacts)
                self.assertEqual(model.batch_intents, before)

    def test_batch_request_replay_is_order_independent_and_cannot_retarget(self):
        replay = self.model.request_delete_batch(
            "operation-batch", tuple(reversed(self.artifacts))
        )
        self.assertIs(replay, self.intent)
        self.assertEqual(self.intent.artifacts, tuple(sorted(
            self.artifacts, key=lambda item: (item.tenant_id, item.artifact_id)
        )))

        for changed_set in (
            self.artifacts[:-1],
            self.artifacts + (VerifiedArtifact("tenant-c", "artifact-c"),),
            (
                self.artifacts[0],
                self.artifacts[1],
                VerifiedArtifact("tenant-b", "artifact-other"),
            ),
        ):
            with self.subTest(changed_set=changed_set):
                self.assert_batch_state_unchanged_after_denial(
                    lambda changed_set=changed_set: self.model.request_delete_batch(
                        "operation-batch", changed_set
                    )
                )

    def test_partial_extra_duplicate_foreign_and_malformed_observations_deny_atomically(self):
        valid = self.observations(
            ObservedPresence.UNKNOWN,
            ObservedPresence.PRESENT,
            ObservedPresence.UNAVAILABLE,
        )
        invalid_observations = (
            valid[:-1],
            valid + (DeleteObservation(VerifiedArtifact("tenant-c", "extra"), ObservedPresence.UNKNOWN),),
            (valid[0], valid[0], valid[2]),
            (
                valid[0],
                valid[1],
                DeleteObservation(VerifiedArtifact("tenant-b", "foreign"), ObservedPresence.UNKNOWN),
            ),
            list(valid),
            (valid[0], object(), valid[2]),
            (
                valid[0],
                valid[1],
                DeleteObservation(self.artifacts[2], "verified_absent"),
            ),
        )
        for observations in invalid_observations:
            with self.subTest(observations=observations):
                self.assert_batch_state_unchanged_after_denial(
                    lambda observations=observations: self.model.observe_batch(
                        "operation-batch", observations
                    )
                )

        self.assert_batch_state_unchanged_after_denial(
            lambda: self.model.observe_batch("operation-missing", valid)
        )

    def test_full_mixed_outcomes_keep_batch_unresolved_until_all_are_absent(self):
        mixed = self.observations(
            ObservedPresence.VERIFIED_ABSENT,
            ObservedPresence.PRESENT,
            ObservedPresence.UNKNOWN,
        )
        unresolved = self.model.observe_batch("operation-batch", mixed)

        self.assertEqual(unresolved.state, IntentState.RECONCILIATION_REQUIRED)
        self.assertEqual(unresolved.last_observations, mixed)
        self.assertEqual(unresolved.artifacts, self.intent.artifacts)

        replay = self.model.request_delete_batch(
            "operation-batch", tuple(reversed(self.artifacts))
        )
        self.assertIs(replay, unresolved)

        still_unresolved = self.model.observe_batch(
            "operation-batch",
            self.observations(
                ObservedPresence.VERIFIED_ABSENT,
                ObservedPresence.UNAVAILABLE,
                ObservedPresence.PRESENT,
            ),
        )
        self.assertEqual(still_unresolved.state, IntentState.RECONCILIATION_REQUIRED)

        completed = self.model.observe_batch(
            "operation-batch",
            self.observations(*([ObservedPresence.VERIFIED_ABSENT] * 3)),
        )
        self.assertEqual(completed.state, IntentState.VERIFIED_ABSENT)
        self.assertEqual(completed.artifacts, self.intent.artifacts)
        self.assertEqual(
            {item.presence for item in completed.last_observations},
            {ObservedPresence.VERIFIED_ABSENT},
        )

    def test_full_absence_replay_is_idempotent_and_terminal_contradiction_denies(self):
        exact_absence = self.observations(
            ObservedPresence.VERIFIED_ABSENT,
            ObservedPresence.VERIFIED_ABSENT,
            ObservedPresence.VERIFIED_ABSENT,
        )
        completed = self.model.observe_batch("operation-batch", exact_absence)
        self.assertEqual(completed.state, IntentState.VERIFIED_ABSENT)
        self.assertIs(
            self.model.observe_batch("operation-batch", tuple(reversed(exact_absence))),
            completed,
        )

        for contradictory_presence in (
            ObservedPresence.PRESENT,
            ObservedPresence.UNKNOWN,
            ObservedPresence.UNAVAILABLE,
        ):
            contradiction = self.observations(
                ObservedPresence.VERIFIED_ABSENT,
                contradictory_presence,
                ObservedPresence.VERIFIED_ABSENT,
            )
            with self.subTest(presence=contradictory_presence):
                self.assert_batch_state_unchanged_after_denial(
                    lambda contradiction=contradiction: self.model.observe_batch(
                        "operation-batch", contradiction
                    )
                )

        self.assertIs(self.model.batch_intents[0], completed)

    def test_batch_and_single_intent_state_remain_separate(self):
        single_model = RetentionDeleteModel()
        single = single_model.request_delete("operation-single", self.artifacts[0])
        batch = single_model.request_delete_batch(
            "operation-batch", self.artifacts
        )

        self.assertEqual(single_model.intents, (single,))
        self.assertEqual(single_model.batch_intents, (batch,))
        self.assertEqual(single.state, IntentState.REQUESTED)
        self.assertEqual(batch.state, IntentState.REQUESTED)

    def test_operation_id_cannot_be_reused_across_single_and_batch_apis(self):
        single_model = RetentionDeleteModel()
        single = single_model.request_delete("shared-operation", self.artifacts[0])
        single_state = single_model.intents
        batch_state = single_model.batch_intents

        with self.assertRaises(DeleteIntentDenied):
            single_model.request_delete_batch("shared-operation", self.artifacts)

        self.assertEqual(single_model.intents, single_state)
        self.assertEqual(single_model.batch_intents, batch_state)

        batch_model = RetentionDeleteModel()
        batch = batch_model.request_delete_batch("shared-operation", self.artifacts)
        single_state = batch_model.intents
        batch_state = batch_model.batch_intents

        with self.assertRaises(DeleteIntentDenied):
            batch_model.request_delete("shared-operation", self.artifacts[0])

        self.assertEqual(batch_model.intents, single_state)
        self.assertEqual(batch_model.batch_intents, batch_state)
        self.assertIs(batch_model.batch_intents[0], batch)
        self.assertIs(single_model.intents[0], single)


if __name__ == "__main__":
    unittest.main()
