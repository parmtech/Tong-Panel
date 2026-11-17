import importlib.util
import sys
import unittest
from dataclasses import FrozenInstanceError, dataclass, replace
from pathlib import Path

from model import RunToBackupHandoffModel, ScheduledBackupHandoffDenied


REPOSITORY_ROOT = Path(__file__).resolve().parents[2]


def load_fixture_module(name, relative_path):
    path = REPOSITORY_ROOT / relative_path
    spec = importlib.util.spec_from_file_location(name, path)
    if spec is None or spec.loader is None:
        raise RuntimeError(f"cannot load fixture module at {path}")
    module = importlib.util.module_from_spec(spec)
    sys.modules[name] = module
    spec.loader.exec_module(module)
    return module


schedule_contract = load_fixture_module(
    "p2_schedule_run_contract",
    Path("prototypes/p2-backup-schedule-run-model/model.py"),
)
retry_contract = load_fixture_module(
    "p2_backup_retry_contract",
    Path("prototypes/p2-backup-retry-model/model.py"),
)


class FakeClock:
    def __init__(self, value=0.0):
        self.value = value

    def __call__(self):
        return self.value

    def advance(self, seconds):
        self.value += seconds


def make_identity(tenant_id, backup_id, generation, content):
    return retry_contract.identity_for_content(tenant_id, backup_id, generation, content)


class ScheduledBackupHandoffTests(unittest.TestCase):
    def setUp(self):
        self.schedule_runs = schedule_contract.ScheduleRunModel()
        self.registration = self.schedule_runs.register_schedule(
            "tenant-a", "nightly", "revision-1"
        )
        self.handoff = RunToBackupHandoffModel(
            self.schedule_runs, retry_contract.Identity
        )
        self.content = b"synthetic scheduled backup bytes"
        self.identity = make_identity(
            "tenant-a", "backup-a", 1, self.content
        )

    def bind(self, occurrence="occurrence-1", identity=None, size=None, **kwargs):
        return self.handoff.bind(
            self.registration.schedule_ref,
            occurrence,
            self.identity if identity is None else identity,
            len(self.content) if size is None else size,
            **kwargs,
        )

    def test_first_run_binds_registered_run_and_retry_identity_unchanged(self):
        binding = self.bind()

        self.assertEqual(binding.run.tenant_id, "tenant-a")
        self.assertEqual(binding.run.schedule_id, "nightly")
        self.assertEqual(binding.run.occurrence_token, "occurrence-1")
        self.assertEqual(binding.run.revision, "revision-1")
        self.assertIs(binding.identity, self.identity)
        self.assertEqual(binding.expected_size_bytes, len(self.content))
        self.assertEqual(self.handoff.bindings, (binding,))

        for field in ("run", "identity", "expected_size_bytes"):
            with self.subTest(field=field):
                with self.assertRaises(FrozenInstanceError):
                    setattr(binding, field, "replacement")

    def test_exact_occurrence_replay_returns_same_binding_and_identity(self):
        first = self.bind()
        replay_run = self.schedule_runs.deliver(
            self.registration.schedule_ref,
            "occurrence-1",
            snapshot_revision="revision-1",
        )
        equivalent_identity = make_identity(
            "tenant-a", "backup-a", 1, self.content
        )

        replay = self.handoff.bind(
            self.registration.schedule_ref,
            "occurrence-1",
            equivalent_identity,
            len(self.content),
            snapshot_revision=replay_run.revision,
        )

        self.assertIs(replay_run, first.run)
        self.assertIs(replay, first)
        self.assertIs(replay.identity, self.identity)
        self.assertEqual(self.handoff.bindings, (first,))

    def test_same_occurrence_token_in_two_registered_scopes_has_two_bindings(self):
        registration_b = self.schedule_runs.register_schedule(
            "tenant-b", "nightly", "revision-1"
        )
        identity_b = make_identity(
            "tenant-b", "backup-b", 1, self.content
        )

        binding_a = self.bind(occurrence="same-occurrence")
        binding_b = self.handoff.bind(
            registration_b.schedule_ref,
            "same-occurrence",
            identity_b,
            len(self.content),
        )

        self.assertEqual(binding_a.run.tenant_id, "tenant-a")
        self.assertEqual(binding_b.run.tenant_id, "tenant-b")
        self.assertEqual(binding_a.run.schedule_id, binding_b.run.schedule_id)
        self.assertEqual(binding_a.run.occurrence_token, binding_b.run.occurrence_token)
        self.assertNotEqual(binding_a.run.run_id, binding_b.run.run_id)
        self.assertIs(binding_a.identity, self.identity)
        self.assertIs(binding_b.identity, identity_b)
        self.assertEqual(self.handoff.bindings, (binding_a, binding_b))

    def test_distinct_occurrence_creates_an_independent_binding(self):
        first = self.bind("occurrence-1")
        second_identity = make_identity(
            "tenant-a", "backup-a-next", 2, self.content
        )

        second = self.bind("occurrence-2", identity=second_identity)

        self.assertNotEqual(first.run.run_id, second.run.run_id)
        self.assertNotEqual(first.run.occurrence_token, second.run.occurrence_token)
        self.assertIs(first.identity, self.identity)
        self.assertIs(second.identity, second_identity)
        self.assertEqual(self.handoff.bindings, (first, second))

    def test_changed_identity_or_size_replay_denies_without_mutation(self):
        first = self.bind()
        before_bindings = self.handoff.bindings
        before_runs = self.schedule_runs.runs
        changed_identities = (
            make_identity("tenant-a", "backup-other", 1, self.content),
            make_identity("tenant-a", "backup-a", 2, self.content),
            make_identity("tenant-a", "backup-a", 1, b"different digest"),
        )

        for identity in changed_identities:
            with self.subTest(identity=identity):
                with self.assertRaises(ScheduledBackupHandoffDenied):
                    self.bind(identity=identity)
                self.assertEqual(self.handoff.bindings, before_bindings)
                self.assertEqual(self.schedule_runs.runs, before_runs)
                self.assertIs(self.handoff.bindings[0], first)

        with self.assertRaises(ScheduledBackupHandoffDenied):
            self.bind(size=len(self.content) + 1)
        self.assertEqual(self.handoff.bindings, before_bindings)
        self.assertEqual(self.schedule_runs.runs, before_runs)

    def test_changed_revision_denies_without_changing_run_or_binding(self):
        first = self.bind()
        before_bindings = self.handoff.bindings
        before_runs = self.schedule_runs.runs

        with self.assertRaises(schedule_contract.ScheduleRunDenied):
            self.bind(snapshot_revision="revision-2")

        self.assertEqual(self.handoff.bindings, before_bindings)
        self.assertEqual(self.schedule_runs.runs, before_runs)
        self.assertIs(self.handoff.bindings[0], first)
        self.assertIs(self.schedule_runs.runs[0], first.run)

    def test_foreign_malformed_and_tenant_mismatched_inputs_are_atomic(self):
        before_bindings = self.handoff.bindings
        before_runs = self.schedule_runs.runs

        with self.assertRaises(ScheduledBackupHandoffDenied):
            self.handoff.bind(
                "schedule-foreign",
                "occurrence-1",
                self.identity,
                len(self.content),
            )
        with self.assertRaises(schedule_contract.ScheduleRunDenied):
            self.bind("occurrence-1", snapshot_revision="")

        registration_b = self.schedule_runs.register_schedule(
            "tenant-b", "nightly", "revision-1"
        )
        schedules_after_registration = self.schedule_runs.schedules
        identity_b = make_identity("tenant-b", "backup-b", 1, self.content)
        with self.assertRaises(ScheduledBackupHandoffDenied):
            self.handoff.bind(
                registration_b.schedule_ref,
                "occurrence-1",
                self.identity,
                len(self.content),
            )
        self.assertEqual(self.handoff.bindings, before_bindings)
        self.assertEqual(self.schedule_runs.runs, before_runs)
        self.assertEqual(self.schedule_runs.schedules, schedules_after_registration)

        binding_b = self.handoff.bind(
            registration_b.schedule_ref,
            "occurrence-1",
            identity_b,
            len(self.content),
        )
        self.assertEqual(binding_b.run.tenant_id, "tenant-b")

        for schedule_ref, occurrence, identity, size in (
            (None, "occurrence-x", self.identity, len(self.content)),
            (self.registration.schedule_ref, None, self.identity, len(self.content)),
            (self.registration.schedule_ref, "occurrence-x", object(), len(self.content)),
            (self.registration.schedule_ref, "occurrence-x", self.identity, True),
            (self.registration.schedule_ref, "occurrence-x", self.identity, -1),
        ):
            with self.subTest(schedule_ref=schedule_ref, occurrence=occurrence, size=size):
                state_before = self.handoff.bindings
                runs_before = self.schedule_runs.runs
                with self.assertRaises((ScheduledBackupHandoffDenied, schedule_contract.ScheduleRunDenied)):
                    self.handoff.bind(schedule_ref, occurrence, identity, size)
                self.assertEqual(self.handoff.bindings, state_before)
                self.assertEqual(self.schedule_runs.runs, runs_before)

        with self.assertRaises(TypeError):
            self.handoff.bind(
                self.registration.schedule_ref,
                "occurrence-x",
                self.identity,
                len(self.content),
                tenant_override="tenant-a",
            )
        self.assertEqual(self.handoff.bindings, (binding_b,))

    def test_mutable_identity_is_denied_without_state_change(self):
        @dataclass
        class MutableIdentity:
            tenant_id: str

        before_bindings = self.handoff.bindings
        before_runs = self.schedule_runs.runs
        with self.assertRaises(ScheduledBackupHandoffDenied):
            self.bind(identity=MutableIdentity("tenant-a"))
        self.assertEqual(self.handoff.bindings, before_bindings)
        self.assertEqual(self.schedule_runs.runs, before_runs)

    def test_malformed_frozen_identity_is_denied_without_state_change(self):
        @dataclass(frozen=True)
        class MalformedIdentity:
            tenant_id: str
            backup_id: str
            generation: object
            expected_sha256: str

        malformed = (
            MalformedIdentity("tenant-a", "", 1, "0" * 64),
            MalformedIdentity("tenant-a", "backup-a", True, "0" * 64),
            MalformedIdentity("tenant-a", "backup-a", 1, "not-a-digest"),
        )
        before_bindings = self.handoff.bindings
        before_runs = self.schedule_runs.runs

        for identity in malformed:
            with self.subTest(identity=identity):
                with self.assertRaises(ScheduledBackupHandoffDenied):
                    self.bind(identity=identity)
                self.assertEqual(self.handoff.bindings, before_bindings)
                self.assertEqual(self.schedule_runs.runs, before_runs)

    def test_well_shaped_foreign_frozen_identity_is_denied_without_state_change(self):
        @dataclass(frozen=True)
        class LookalikeIdentity:
            tenant_id: str
            backup_id: str
            generation: int
            expected_sha256: str

        lookalike = LookalikeIdentity(
            self.identity.tenant_id,
            self.identity.backup_id,
            self.identity.generation,
            self.identity.expected_sha256,
        )
        before_bindings = self.handoff.bindings
        before_runs = self.schedule_runs.runs

        with self.assertRaises(ScheduledBackupHandoffDenied):
            self.bind(identity=lookalike)

        self.assertEqual(self.handoff.bindings, before_bindings)
        self.assertEqual(self.schedule_runs.runs, before_runs)

    def test_retry_reuses_binding_identity_size_and_injected_retry_bounds(self):
        binding = self.bind()
        limits = retry_contract.TransferLimits(
            max_size_bytes=len(self.content),
            max_duration_seconds=5,
            max_attempts=2,
            backoff_seconds=(2,),
            max_concurrent_transfers=1,
        )
        clock = FakeClock()
        transfer_model = retry_contract.Model()
        simulator = retry_contract.TransferSimulator(transfer_model, limits, clock)

        first_attempt = simulator.start_transfer(
            binding.identity, binding.expected_size_bytes
        )
        simulator.report_progress(first_attempt, 1)
        first_result = simulator.interrupt_transfer(
            first_attempt, cleanup_verified=True
        )
        self.assertEqual(first_result.state, retry_contract.TransferState.CLEANED)
        self.assertIs(first_result.identity, binding.identity)
        self.assertEqual(first_result.expected_size_bytes, binding.expected_size_bytes)

        replay = self.bind()
        self.assertIs(replay, binding)
        with self.assertRaises(retry_contract.TransferBackoffNotElapsed):
            simulator.retry_transfer(first_attempt)

        clock.advance(2)
        second_attempt = simulator.retry_transfer(first_attempt)
        second_result = simulator.transfer(second_attempt)
        self.assertEqual(second_result.attempt_number, 2)
        self.assertIs(second_result.identity, binding.identity)
        self.assertEqual(second_result.expected_size_bytes, binding.expected_size_bytes)

        with self.assertRaises(retry_contract.TransferConcurrencyLimit):
            simulator.start_transfer(binding.identity, binding.expected_size_bytes)

        clock.advance(6)
        second_failure = simulator.interrupt_transfer(
            second_attempt, cleanup_verified=True
        )
        self.assertEqual(
            second_failure.failure_reason,
            retry_contract.TransferFailureReason.DURATION_LIMIT,
        )
        clock.advance(2)
        with self.assertRaises(retry_contract.TransferAttemptLimit):
            simulator.retry_transfer(second_attempt)
        self.assertIsNone(
            transfer_model.current(
                binding.identity.tenant_id, binding.identity.backup_id
            )
        )

    def test_retry_simulator_enforces_injected_size_bound_for_handoff(self):
        binding = self.bind()
        limits = retry_contract.TransferLimits(
            max_size_bytes=len(self.content) - 1,
            max_duration_seconds=5,
            max_attempts=1,
            backoff_seconds=(),
            max_concurrent_transfers=1,
        )
        clock = FakeClock()
        transfer_model = retry_contract.Model()
        simulator = retry_contract.TransferSimulator(transfer_model, limits, clock)

        attempt = simulator.start_transfer(
            binding.identity, binding.expected_size_bytes
        )
        result = simulator.transfer(attempt)

        self.assertEqual(result.state, retry_contract.TransferState.FAILED)
        self.assertEqual(
            result.failure_reason, retry_contract.TransferFailureReason.SIZE_LIMIT
        )
        self.assertEqual(result.transferred_bytes, 0)
        self.assertIs(result.identity, binding.identity)
        self.assertEqual(result.expected_size_bytes, binding.expected_size_bytes)
        self.assertIsNone(
            transfer_model.current(
                binding.identity.tenant_id, binding.identity.backup_id
            )
        )


if __name__ == "__main__":
    unittest.main()
