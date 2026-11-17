"""Synthetic in-memory identity model for scheduled backup run delivery.

Schedule ownership is derived from fixture registration. This module does not
implement a scheduler, job queue, persistence, authorization, or backup work.
"""

from __future__ import annotations

from dataclasses import dataclass


_UNSET = object()
_GENERIC_DENIAL = "scheduled occurrence denied"


class ScheduleRunDenied(ValueError):
    """A schedule occurrence input is unknown, invalid, or conflicts with state."""

    def __init__(self) -> None:
        # Keep unknown, foreign, malformed, and conflicting inputs alike so the
        # fixture does not encode an existence-disclosure behavior.
        super().__init__(_GENERIC_DENIAL)


@dataclass(frozen=True)
class ScheduleRegistration:
    """Trusted fixture registration; tenant ownership is stored here."""

    schedule_ref: str
    tenant_id: str
    schedule_id: str
    revision: str


@dataclass(frozen=True)
class RunRecord:
    """Immutable synthetic record for one logical scheduled occurrence."""

    run_id: str
    tenant_id: str
    schedule_id: str
    occurrence_token: str
    revision: str


class ScheduleRunModel:
    """Materialize each valid fixture occurrence into one immutable run record."""

    def __init__(self) -> None:
        self._schedules_by_ref: dict[str, ScheduleRegistration] = {}
        self._schedule_keys: set[tuple[str, str]] = set()
        self._runs_by_key: dict[tuple[str, str, str], RunRecord] = {}
        self._next_schedule_number = 1
        self._next_run_number = 1

    @property
    def schedules(self) -> tuple[ScheduleRegistration, ...]:
        return tuple(self._schedules_by_ref.values())

    @property
    def runs(self) -> tuple[RunRecord, ...]:
        return tuple(self._runs_by_key.values())

    def register_schedule(
        self, tenant_id: object, schedule_id: object, revision: object
    ) -> ScheduleRegistration:
        """Register an immutable test schedule and return its opaque fixture ref."""

        tenant = _require_identifier(tenant_id)
        schedule = _require_identifier(schedule_id)
        version = _require_identifier(revision)
        schedule_key = (tenant, schedule)
        if schedule_key in self._schedule_keys:
            raise ScheduleRunDenied()

        schedule_ref = f"schedule-{self._next_schedule_number:06d}"
        registration = ScheduleRegistration(schedule_ref, tenant, schedule, version)
        self._schedules_by_ref[schedule_ref] = registration
        self._schedule_keys.add(schedule_key)
        self._next_schedule_number += 1
        return registration

    def deliver(
        self,
        schedule_ref: object,
        occurrence_token: object,
        *,
        snapshot_revision: object = _UNSET,
        tenant_override: object = _UNSET,
    ) -> RunRecord:
        """Record one occurrence, returning the existing record on exact replay.

        `tenant_override` is an adversarial test input and any supplied value is
        rejected. The optional revision stands for an already captured opaque
        fixture snapshot, not a computed schedule or an approved schedule-edit
        policy.
        """

        # The event contract carries no tenant field. Reject even a matching
        # override so event data can never participate in tenant selection.
        if tenant_override is not _UNSET:
            raise ScheduleRunDenied()

        reference = _require_identifier(schedule_ref)
        occurrence = _require_identifier(occurrence_token)
        registration = self._schedules_by_ref.get(reference)
        if registration is None:
            raise ScheduleRunDenied()

        if snapshot_revision is _UNSET:
            revision = registration.revision
        else:
            revision = _require_identifier(snapshot_revision)

        logical_key = (registration.tenant_id, registration.schedule_id, occurrence)
        existing = self._runs_by_key.get(logical_key)
        if existing is not None:
            if existing.revision != revision:
                raise ScheduleRunDenied()
            return existing

        if revision != registration.revision:
            raise ScheduleRunDenied()

        run_id = f"run-{self._next_run_number:06d}"
        record = RunRecord(
            run_id=run_id,
            tenant_id=registration.tenant_id,
            schedule_id=registration.schedule_id,
            occurrence_token=occurrence,
            revision=revision,
        )
        self._runs_by_key[logical_key] = record
        self._next_run_number += 1
        return record


def _require_identifier(value: object) -> str:
    if type(value) is not str or value == "":
        raise ScheduleRunDenied()
    return value
