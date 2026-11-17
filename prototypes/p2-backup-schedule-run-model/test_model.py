import unittest
from dataclasses import FrozenInstanceError

from model import RunRecord, ScheduleRunDenied, ScheduleRunModel


class ScheduleRunModelTests(unittest.TestCase):
    def make_model(self):
        model = ScheduleRunModel()
        registration = model.register_schedule("tenant-a", "nightly", "revision-1")
        return model, registration

    def test_first_delivery_records_one_run_with_registered_owner(self):
        model, registration = self.make_model()

        run = model.deliver(registration.schedule_ref, "occurrence-1")

        self.assertEqual(
            run,
            RunRecord(
                run_id="run-000001",
                tenant_id="tenant-a",
                schedule_id="nightly",
                occurrence_token="occurrence-1",
                revision="revision-1",
            ),
        )
        self.assertEqual(model.runs, (run,))

    def test_exact_duplicate_delivery_returns_the_same_immutable_run(self):
        model, registration = self.make_model()
        first = model.deliver(registration.schedule_ref, "occurrence-1")

        duplicate = model.deliver(
            registration.schedule_ref,
            "occurrence-1",
            snapshot_revision="revision-1",
        )

        self.assertIs(duplicate, first)
        self.assertEqual(model.runs, (first,))

    def test_same_occurrence_token_is_scoped_to_tenant_and_schedule(self):
        model = ScheduleRunModel()
        tenant_a = model.register_schedule("tenant-a", "nightly", "revision-1")
        tenant_b = model.register_schedule("tenant-b", "nightly", "revision-1")

        run_a = model.deliver(tenant_a.schedule_ref, "same-occurrence")
        run_b = model.deliver(tenant_b.schedule_ref, "same-occurrence")

        self.assertNotEqual(run_a.run_id, run_b.run_id)
        self.assertEqual(run_a.tenant_id, "tenant-a")
        self.assertEqual(run_b.tenant_id, "tenant-b")
        self.assertEqual(run_a.schedule_id, run_b.schedule_id)
        self.assertEqual(run_a.occurrence_token, run_b.occurrence_token)
        self.assertEqual(model.runs, (run_a, run_b))

    def test_unknown_schedule_and_tenant_override_share_generic_denial(self):
        model = ScheduleRunModel()
        tenant_a = model.register_schedule("tenant-a", "nightly", "revision-1")
        tenant_b = model.register_schedule("tenant-b", "nightly", "revision-1")
        before = model.runs

        with self.assertRaises(ScheduleRunDenied) as overridden:
            model.deliver(tenant_b.schedule_ref, "occurrence-1", tenant_override="tenant-a")
        with self.assertRaises(ScheduleRunDenied) as unknown:
            model.deliver("schedule-unknown", "occurrence-1")

        self.assertEqual(str(overridden.exception), str(unknown.exception))
        self.assertEqual(model.runs, before)
        # A valid event still uses the registration owner, without an override.
        self.assertEqual(model.deliver(tenant_a.schedule_ref, "occurrence-1").tenant_id, "tenant-a")

    def test_tenant_override_cannot_change_registered_owner(self):
        model, registration = self.make_model()
        schedules_before = model.schedules
        runs_before = model.runs

        for tenant_override in ("tenant-a", "tenant-b"):
            with self.subTest(tenant_override=tenant_override):
                with self.assertRaises(ScheduleRunDenied) as denied:
                    model.deliver(
                        registration.schedule_ref,
                        "occurrence-1",
                        tenant_override=tenant_override,
                    )
                self.assertEqual(str(denied.exception), "scheduled occurrence denied")
                self.assertEqual(model.schedules, schedules_before)
                self.assertEqual(model.runs, runs_before)

        self.assertEqual(model.deliver(registration.schedule_ref, "occurrence-1").tenant_id, "tenant-a")

    def test_conflicting_snapshot_revision_replay_cannot_replace_run(self):
        model, registration = self.make_model()
        original = model.deliver(registration.schedule_ref, "occurrence-1")
        before = model.runs

        with self.assertRaises(ScheduleRunDenied):
            model.deliver(
                registration.schedule_ref,
                "occurrence-1",
                snapshot_revision="revision-2",
            )

        self.assertEqual(model.runs, before)
        self.assertIs(model.deliver(registration.schedule_ref, "occurrence-1"), original)

    def test_run_identity_fields_and_schedule_registration_are_immutable(self):
        model, registration = self.make_model()
        run = model.deliver(registration.schedule_ref, "occurrence-1")

        for field in ("run_id", "tenant_id", "schedule_id", "occurrence_token", "revision"):
            with self.subTest(record="run", field=field):
                with self.assertRaises(FrozenInstanceError):
                    setattr(run, field, "replacement")

        for field in ("schedule_ref", "tenant_id", "schedule_id", "revision"):
            with self.subTest(record="schedule", field=field):
                with self.assertRaises(FrozenInstanceError):
                    setattr(registration, field, "replacement")

        self.assertEqual(model.runs, (run,))
        self.assertEqual(model.schedules, (registration,))

    def test_distinct_occurrence_creates_a_new_run_without_replacing_prior_run(self):
        model, registration = self.make_model()
        first = model.deliver(registration.schedule_ref, "occurrence-1")

        second = model.deliver(registration.schedule_ref, "occurrence-2")

        self.assertNotEqual(first.run_id, second.run_id)
        self.assertEqual(first.occurrence_token, "occurrence-1")
        self.assertEqual(second.occurrence_token, "occurrence-2")
        self.assertEqual(model.runs, (first, second))

    def test_malformed_schedule_registration_ids_are_denied_without_mutation(self):
        malformed = (None, "", 7, True, b"bytes")
        for field in ("tenant", "schedule", "revision"):
            for value in malformed:
                with self.subTest(field=field, value=value):
                    model = ScheduleRunModel()
                    values = {
                        "tenant": "tenant-a",
                        "schedule": "nightly",
                        "revision": "revision-1",
                    }
                    values[field] = value
                    with self.assertRaises(ScheduleRunDenied):
                        model.register_schedule(
                            values["tenant"], values["schedule"], values["revision"]
                        )
                    self.assertEqual(model.schedules, ())
                    self.assertEqual(model.runs, ())
                    valid = model.register_schedule("tenant-a", "nightly", "revision-1")
                    self.assertEqual(valid.schedule_ref, "schedule-000001")

    def test_malformed_delivery_ids_are_denied_without_mutation(self):
        malformed = (None, "", 7, True, b"bytes")
        for field in ("schedule_ref", "occurrence", "revision"):
            for value in malformed:
                with self.subTest(field=field, value=value):
                    model, registration = self.make_model()
                    kwargs = {}
                    schedule_ref = registration.schedule_ref
                    occurrence = "occurrence-1"
                    if field == "schedule_ref":
                        schedule_ref = value
                    elif field == "occurrence":
                        occurrence = value
                    elif field == "revision":
                        kwargs["snapshot_revision"] = value
                    with self.assertRaises(ScheduleRunDenied):
                        model.deliver(schedule_ref, occurrence, **kwargs)

                    self.assertEqual(model.runs, ())
                    next_run = model.deliver(registration.schedule_ref, "occurrence-1")
                    self.assertEqual(next_run.run_id, "run-000001")

    def test_duplicate_schedule_registration_cannot_replace_fixture_snapshot(self):
        model, registration = self.make_model()
        before = model.schedules

        with self.assertRaises(ScheduleRunDenied):
            model.register_schedule("tenant-a", "nightly", "revision-2")

        self.assertEqual(model.schedules, before)
        self.assertIs(model.schedules[0], registration)


if __name__ == "__main__":
    unittest.main()
