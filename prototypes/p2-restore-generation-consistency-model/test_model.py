import unittest
from dataclasses import FrozenInstanceError, replace

from model import (
    REQUIRED_COMPONENTS,
    ActivationAcknowledgmentLost,
    ActivationFailedBeforeActivation,
    ActivationOutcomeUnresolved,
    ActivationState,
    ActivationStore,
    ActiveBundle,
    BundleIdentity,
    ComponentRecord,
    InjectedActivationFailure,
    RestoreCandidate,
    RestoreContractError,
)


INITIAL_IDENTITY = BundleIdentity("tenant-a", "backup-a", 1, "marker-one")


def records_for(identity=INITIAL_IDENTITY):
    return tuple(
        ComponentRecord(
            component=name,
            tenant=identity.tenant,
            backup_id=identity.backup_id,
            generation=identity.generation,
            marker=identity.marker,
        )
        for name in REQUIRED_COMPONENTS
    )


def candidate_for(identity=INITIAL_IDENTITY, components=None):
    if components is None:
        components = records_for(identity)
    return RestoreCandidate(identity, components)


def change_component(components, component_name, **changes):
    return tuple(
        replace(record, **changes) if record.component == component_name else record
        for record in components
    )


class RestoreGenerationConsistencyTests(unittest.TestCase):
    def assert_denied_without_changing_active(self, invalid_candidate):
        store = ActivationStore()
        original = store.activate(candidate_for())

        with self.assertRaises(RestoreContractError):
            store.activate(invalid_candidate)

        self.assertIs(store.active, original)
        self.assertEqual(store.active.identity, INITIAL_IDENTITY)
        self.assertEqual(store.active.components, records_for())

    def test_all_components_must_agree_before_first_activation(self):
        store = ActivationStore()
        identity = BundleIdentity("tenant-b", "backup-b", 7, "opaque-marker-7")

        active = store.activate(candidate_for(identity))

        self.assertIs(store.active, active)
        self.assertEqual(active.identity, identity)
        self.assertEqual(tuple(record.component for record in active.components), REQUIRED_COMPONENTS)
        self.assertTrue(all(record.tenant == "tenant-b" for record in active.components))
        self.assertTrue(all(record.backup_id == "backup-b" for record in active.components))
        self.assertTrue(all(record.generation == 7 for record in active.components))
        self.assertTrue(all(record.marker == "opaque-marker-7" for record in active.components))

    def test_complete_replacement_swaps_one_bundle_without_mixed_generations(self):
        store = ActivationStore()
        original = store.activate(candidate_for())
        replacement_identity = BundleIdentity("tenant-a", "backup-a", 2, "marker-two")

        replacement = store.activate(candidate_for(replacement_identity))

        self.assertIs(store.active, replacement)
        self.assertIsNot(store.active, original)
        self.assertEqual(original.identity, INITIAL_IDENTITY)
        self.assertEqual(replacement.identity, replacement_identity)
        self.assertEqual(
            {(record.tenant, record.backup_id, record.generation, record.marker)
             for record in store.active.components},
            {("tenant-a", "backup-a", 2, "marker-two")},
        )
        self.assertEqual(tuple(record.component for record in store.active.components), REQUIRED_COMPONENTS)

    def test_failure_before_activation_is_known_inactive_and_preserves_prior_active(self):
        store = ActivationStore()
        original = store.activate(candidate_for())
        replacement_identity = BundleIdentity("tenant-a", "backup-a", 2, "marker-two")
        replacement_candidate = candidate_for(replacement_identity)

        with self.assertRaises(ActivationFailedBeforeActivation):
            store.activate(
                replacement_candidate,
                failure=InjectedActivationFailure.BEFORE_ACTIVATION,
            )

        outcome = store.last_outcome
        self.assertEqual(outcome.state, ActivationState.INACTIVE)
        self.assertEqual(outcome.candidate.identity, replacement_identity)
        self.assertIs(store.active, original)
        self.assertEqual(store.active.identity, INITIAL_IDENTITY)

        # A known pre-activation failure permits a direct retry because the
        # candidate was not installed and no outcome is unresolved.
        replacement = store.activate(replacement_candidate)
        self.assertEqual(store.last_outcome.state, ActivationState.ACTIVE)
        self.assertIs(store.active, replacement)
        self.assertEqual(store.active.identity, replacement_identity)

    def test_failure_before_first_activation_is_known_inactive_not_unresolved(self):
        store = ActivationStore()
        candidate = candidate_for()

        with self.assertRaises(ActivationFailedBeforeActivation):
            store.activate(candidate, failure=InjectedActivationFailure.BEFORE_ACTIVATION)

        self.assertEqual(store.last_outcome.state, ActivationState.INACTIVE)
        self.assertIsNone(store.active)

        active = store.activate(candidate)
        self.assertEqual(store.last_outcome.state, ActivationState.ACTIVE)
        self.assertIs(store.active, active)

    def test_lost_acknowledgment_hides_active_until_exact_readback(self):
        store = ActivationStore()
        original = store.activate(candidate_for())
        replacement_identity = BundleIdentity("tenant-a", "backup-a", 2, "marker-two")
        replacement_candidate = candidate_for(replacement_identity)
        expected_read_back = ActiveBundle(
            replacement_identity,
            records_for(replacement_identity),
        )

        with self.assertRaises(ActivationAcknowledgmentLost):
            store.activate(
                replacement_candidate,
                failure=InjectedActivationFailure.ACK_LOST_AFTER_ACTIVATION,
            )

        unresolved = store.last_outcome
        self.assertEqual(unresolved.state, ActivationState.RECONCILIATION_REQUIRED)
        self.assertEqual(unresolved.candidate.identity, replacement_identity)
        self.assertIsNone(store.active)
        self.assertIsNot(store.active, original)

        reconciled = store.reconcile(expected_read_back)

        self.assertIs(store.active, reconciled)
        self.assertEqual(store.active.identity, replacement_identity)
        self.assertEqual(store.last_outcome.state, ActivationState.ACTIVE)

    def test_mismatched_readback_and_blind_retries_stay_unresolved_without_mutation(self):
        store = ActivationStore()
        store.activate(candidate_for())
        replacement_identity = BundleIdentity("tenant-a", "backup-a", 2, "marker-two")
        replacement_candidate = candidate_for(replacement_identity)
        with self.assertRaises(ActivationAcknowledgmentLost):
            store.activate(
                replacement_candidate,
                failure=InjectedActivationFailure.ACK_LOST_AFTER_ACTIVATION,
            )

        unresolved = store.last_outcome
        invalid_read_backs = (
            None,
            ActiveBundle(INITIAL_IDENTITY, records_for()),
            ActiveBundle(
                replacement_identity,
                change_component(
                    records_for(replacement_identity),
                    "settings",
                    marker="different-marker",
                ),
            ),
            ActiveBundle(
                replace(replacement_identity, generation=True),
                records_for(replace(replacement_identity, generation=True)),
            ),
            ActiveBundle(
                replace(replacement_identity, generation=1.0),
                records_for(replace(replacement_identity, generation=1.0)),
            ),
            ActiveBundle(
                replacement_identity,
                (
                    replace(
                        records_for(replacement_identity)[0],
                        generation=True,
                    ),
                    *records_for(replacement_identity)[1:],
                ),
            ),
            ActiveBundle(
                replacement_identity,
                (
                    replace(
                        records_for(replacement_identity)[0],
                        generation=1.0,
                    ),
                    *records_for(replacement_identity)[1:],
                ),
            ),
            object(),
        )
        for read_back in invalid_read_backs:
            with self.subTest(read_back=read_back):
                with self.assertRaises(ActivationOutcomeUnresolved):
                    store.reconcile(read_back)
                self.assertIs(store.last_outcome, unresolved)
                self.assertEqual(
                    store.last_outcome.state,
                    ActivationState.RECONCILIATION_REQUIRED,
                )
                self.assertIsNone(store.active)

        for retry_candidate in (replacement_candidate, candidate_for()):
            with self.subTest(identity=retry_candidate.expected_identity):
                with self.assertRaises(ActivationOutcomeUnresolved):
                    store.activate(retry_candidate)
                self.assertIs(store.last_outcome, unresolved)
                self.assertIsNone(store.active)

        resolved = store.reconcile(
            ActiveBundle(replacement_identity, records_for(replacement_identity))
        )
        self.assertEqual(resolved.identity, replacement_identity)
        self.assertIs(store.active, resolved)

    def test_reconcile_without_pending_outcome_is_denied_without_mutation(self):
        store = ActivationStore()
        active = store.activate(candidate_for())
        outcome = store.last_outcome

        with self.assertRaises(RestoreContractError):
            store.reconcile(active)

        self.assertIs(store.active, active)
        self.assertIs(store.last_outcome, outcome)

    def test_missing_empty_or_non_string_expected_markers_are_denied_without_mutation(self):
        for marker in (None, ""):
            with self.subTest(marker=marker):
                identity = replace(INITIAL_IDENTITY, marker=marker)
                self.assert_denied_without_changing_active(candidate_for(identity))

    def test_each_component_missing_or_empty_marker_is_denied_without_mutation(self):
        for component_name in REQUIRED_COMPONENTS:
            for marker in (None, ""):
                with self.subTest(component=component_name, marker=marker):
                    components = change_component(
                        records_for(), component_name, marker=marker
                    )
                    self.assert_denied_without_changing_active(
                        candidate_for(components=components)
                    )

    def test_missing_duplicate_and_unknown_components_are_denied_without_mutation(self):
        records = records_for()
        cases = (
            ("missing", (records[0], records[1])),
            ("duplicate", (records[0], records[0], records[2])),
            (
                "unknown",
                (records[0], records[1], replace(records[2], component="manifest")),
            ),
        )

        for name, components in cases:
            with self.subTest(case=name):
                self.assert_denied_without_changing_active(candidate_for(components=components))

    def test_non_string_component_name_is_denied_without_mutation(self):
        components = change_component(records_for(), "settings", component=None)
        self.assert_denied_without_changing_active(candidate_for(components=components))

    def test_wrong_outer_candidate_type_is_denied_without_mutation(self):
        self.assert_denied_without_changing_active(object())

    def test_each_component_identity_mismatch_is_denied_without_mutation(self):
        mismatches = (
            ("tenant", {"tenant": "tenant-b"}),
            ("backup_id", {"backup_id": "backup-b"}),
            ("generation", {"generation": 2}),
            ("marker", {"marker": "other-marker"}),
        )

        for component_name in REQUIRED_COMPONENTS:
            for field_name, changes in mismatches:
                with self.subTest(component=component_name, field=field_name):
                    components = change_component(
                        records_for(), component_name, **changes
                    )
                    self.assert_denied_without_changing_active(
                        candidate_for(components=components)
                    )

    def test_non_component_records_and_malformed_tenant_or_backup_ids_are_denied(self):
        records = records_for()
        cases = (
            ("non-record", (records[0], object(), records[2])),
            (
                "tenant-type",
                change_component(records, "site_files", tenant=None),
            ),
            (
                "backup-id-type",
                change_component(records, "settings", backup_id=42),
            ),
        )

        for name, components in cases:
            with self.subTest(case=name):
                self.assert_denied_without_changing_active(
                    candidate_for(components=components)
                )

    def test_invalid_expected_identity_and_non_tuple_components_are_denied_without_mutation(self):
        invalid_generations = (0, -1, True, 1.0)
        for generation in invalid_generations:
            with self.subTest(generation=generation):
                identity = replace(INITIAL_IDENTITY, generation=generation)
                self.assert_denied_without_changing_active(candidate_for(identity))

        self.assert_denied_without_changing_active(RestoreCandidate(None, records_for()))
        self.assert_denied_without_changing_active(RestoreCandidate(INITIAL_IDENTITY, None))
        self.assert_denied_without_changing_active(
            RestoreCandidate(INITIAL_IDENTITY, list(records_for()))
        )

    def test_identity_candidate_records_and_active_bundle_are_immutable(self):
        candidate = candidate_for()
        active = ActivationStore().activate(candidate)

        with self.assertRaises(FrozenInstanceError):
            candidate.expected_identity.generation = 8
        with self.assertRaises(FrozenInstanceError):
            candidate.components[0].marker = "changed"
        with self.assertRaises(FrozenInstanceError):
            active.identity.marker = "changed"
        with self.assertRaises(FrozenInstanceError):
            active.components = ()


if __name__ == "__main__":
    unittest.main()
