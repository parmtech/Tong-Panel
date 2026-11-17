"""Synthetic binding from a registered schedule run to a backup identity.

The model owns no scheduler or transfer behavior. It composes the existing
schedule-run and backup-retry fixture contracts in memory only.
"""

from __future__ import annotations

from dataclasses import dataclass, is_dataclass
from typing import Protocol


_UNSET = object()
_GENERIC_DENIAL = "scheduled backup handoff denied"


class ScheduledBackupHandoffDenied(ValueError):
    """A run-to-backup binding is invalid, foreign, or conflicts with state."""

    def __init__(self) -> None:
        super().__init__(_GENERIC_DENIAL)


class ScheduleRunSource(Protocol):
    """The narrow fixture surface used from the existing schedule-run model."""

    @property
    def schedules(self) -> tuple[object, ...]: ...

    def deliver(
        self,
        schedule_ref: object,
        occurrence_token: object,
        *,
        snapshot_revision: object = _UNSET,
    ) -> object: ...


@dataclass(frozen=True)
class ScheduledBackupBinding:
    """One immutable registered run bound to its existing backup identity."""

    run: object
    identity: object
    expected_size_bytes: int


class RunToBackupHandoffModel:
    """Bind a trusted fixture occurrence to one immutable backup identity.

    The logical key is the existing schedule contract's registered
    ``(tenant_id, schedule_id, occurrence_token)`` tuple. The immutable run
    record also retains its ``run_id`` and revision. The backup identity is
    carried unchanged and is never rebuilt or interpreted beyond checking that
    its existing fixture tenant identity agrees with the registered schedule.
    That comparison is fixture consistency only; it is not authorization.
    """

    def __init__(
        self,
        schedule_runs: ScheduleRunSource,
        backup_identity_type: type,
    ) -> None:
        if not callable(getattr(schedule_runs, "deliver", None)):
            raise ScheduledBackupHandoffDenied()
        if not isinstance(getattr(schedule_runs, "schedules", None), tuple):
            raise ScheduledBackupHandoffDenied()
        identity_params = getattr(backup_identity_type, "__dataclass_params__", None)
        if (
            not isinstance(backup_identity_type, type)
            or not is_dataclass(backup_identity_type)
            or identity_params is None
            or not identity_params.frozen
        ):
            raise ScheduledBackupHandoffDenied()
        self._schedule_runs = schedule_runs
        self._backup_identity_type = backup_identity_type
        self._bindings: dict[tuple[str, str, str], ScheduledBackupBinding] = {}

    @property
    def bindings(self) -> tuple[ScheduledBackupBinding, ...]:
        return tuple(self._bindings.values())

    def bind(
        self,
        schedule_ref: object,
        occurrence_token: object,
        identity: object,
        expected_size_bytes: object,
        *,
        snapshot_revision: object = _UNSET,
    ) -> ScheduledBackupBinding:
        """Deliver/replay one registered occurrence and bind its backup value.

        Transfer bounds are deliberately not defaulted or selected here. The
        caller supplies those directly to the existing retry simulator.
        """

        registration = self._registration(schedule_ref)
        identity_tenant = _require_backup_identity(identity, self._backup_identity_type)
        if identity_tenant != registration.tenant_id:
            raise ScheduledBackupHandoffDenied()
        if type(occurrence_token) is not str or occurrence_token == "":
            raise ScheduledBackupHandoffDenied()
        if type(expected_size_bytes) is not int or expected_size_bytes < 0:
            raise ScheduledBackupHandoffDenied()

        if snapshot_revision is _UNSET:
            run = self._schedule_runs.deliver(schedule_ref, occurrence_token)
        else:
            run = self._schedule_runs.deliver(
                schedule_ref,
                occurrence_token,
                snapshot_revision=snapshot_revision,
            )

        key = (run.tenant_id, run.schedule_id, run.occurrence_token)
        existing = self._bindings.get(key)
        if existing is not None:
            if (
                existing.run == run
                and existing.identity == identity
                and existing.expected_size_bytes == expected_size_bytes
            ):
                return existing
            raise ScheduledBackupHandoffDenied()

        binding = ScheduledBackupBinding(
            run=run,
            identity=identity,
            expected_size_bytes=expected_size_bytes,
        )
        self._bindings[key] = binding
        return binding

    def _registration(self, schedule_ref: object) -> object:
        if type(schedule_ref) is not str or schedule_ref == "":
            raise ScheduledBackupHandoffDenied()
        for registration in self._schedule_runs.schedules:
            if getattr(registration, "schedule_ref", None) == schedule_ref:
                return registration
        raise ScheduledBackupHandoffDenied()


def _require_backup_identity(value: object, identity_type: type) -> str:
    """Require the canonical frozen retry identity and its complete value shape."""

    if type(value) is not identity_type or not is_dataclass(value):
        raise ScheduledBackupHandoffDenied()
    params = getattr(type(value), "__dataclass_params__", None)
    if params is None or not params.frozen:
        raise ScheduledBackupHandoffDenied()
    try:
        hash(value)
    except TypeError as exc:
        raise ScheduledBackupHandoffDenied() from exc

    tenant_id = getattr(value, "tenant_id", None)
    backup_id = getattr(value, "backup_id", None)
    generation = getattr(value, "generation", None)
    expected_sha256 = getattr(value, "expected_sha256", None)
    if (
        type(tenant_id) is not str
        or tenant_id == ""
        or type(backup_id) is not str
        or backup_id == ""
        or type(generation) is not int
        or generation < 1
        or type(expected_sha256) is not str
        or len(expected_sha256) != 64
        or expected_sha256 != expected_sha256.lower()
        or any(char not in "0123456789abcdef" for char in expected_sha256)
    ):
        raise ScheduledBackupHandoffDenied()
    return tenant_id
