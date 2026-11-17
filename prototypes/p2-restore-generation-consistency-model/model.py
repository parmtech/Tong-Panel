"""Synthetic in-memory contract for complete restore-bundle activation."""

from __future__ import annotations

from dataclasses import dataclass
from enum import Enum


REQUIRED_COMPONENTS = ("site_files", "database", "settings")


class RestoreContractError(ValueError):
    """A synthetic restore candidate is incomplete or inconsistent."""


@dataclass(frozen=True)
class BundleIdentity:
    """Caller-supplied synthetic identity shared by one restore bundle."""

    tenant: str
    backup_id: str
    generation: int
    marker: str


@dataclass(frozen=True)
class ComponentRecord:
    """Synthetic metadata for one restore component; contains no payload."""

    component: str
    tenant: str
    backup_id: str
    generation: int
    marker: str


@dataclass(frozen=True)
class RestoreCandidate:
    """All component records proposed for one atomic activation decision."""

    expected_identity: BundleIdentity
    components: tuple[ComponentRecord, ...]


@dataclass(frozen=True)
class ActiveBundle:
    """Immutable synthetic snapshot referenced by the active-bundle pointer."""

    identity: BundleIdentity
    components: tuple[ComponentRecord, ...]


class ActivationState(Enum):
    """Known outcome for the most recent candidate activation attempt."""

    ACTIVE = "active"
    INACTIVE = "inactive"
    RECONCILIATION_REQUIRED = "reconciliation_required"


class InjectedActivationFailure(Enum):
    """Synthetic failpoints for activation outcome tests."""

    BEFORE_ACTIVATION = "before_activation"
    ACK_LOST_AFTER_ACTIVATION = "ack_lost_after_activation"


class ActivationFailedBeforeActivation(RestoreContractError):
    """The candidate is known inactive; the previous active bundle is intact."""


class ActivationAcknowledgmentLost(RestoreContractError):
    """The in-memory pointer changed, but the outcome needs exact read-back."""


class ActivationOutcomeUnresolved(RestoreContractError):
    """Activation is unresolved and must be reconciled before another attempt."""


@dataclass(frozen=True)
class ActivationOutcome:
    """Explicit state for one candidate, separate from the visible active bundle."""

    candidate: ActiveBundle
    state: ActivationState


class ActivationStore:
    """One in-memory bundle pointer with explicit uncertain-ack handling.

    The synthetic pointer, known active bundle, and last attempt outcome are
    separate so a lost acknowledgment cannot be reported as active before an
    exact read-back is reconciled.
    """

    def __init__(self) -> None:
        self._pointer: ActiveBundle | None = None
        self._active: ActiveBundle | None = None
        self._pending: ActiveBundle | None = None
        self._last_outcome: ActivationOutcome | None = None

    @property
    def active(self) -> ActiveBundle | None:
        """Return a known active bundle, or ``None`` while an outcome is unknown."""

        if self._pending is not None:
            return None
        return self._active

    @property
    def last_outcome(self) -> ActivationOutcome | None:
        return self._last_outcome

    def activate(
        self,
        candidate: RestoreCandidate,
        *,
        failure: InjectedActivationFailure | None = None,
    ) -> ActiveBundle:
        """Validate a candidate, then activate or inject one synthetic failure.

        ``BEFORE_ACTIVATION`` is a known inactive result for this candidate and
        leaves the prior pointer and known active bundle unchanged. An
        acknowledgment lost after the pointer swap sets a reconciliation state;
        callers cannot observe an active bundle or retry until exact read-back.
        """

        if self._pending is not None:
            raise ActivationOutcomeUnresolved(
                "reconcile the unresolved activation before retrying"
            )
        if failure is not None and type(failure) is not InjectedActivationFailure:
            raise RestoreContractError("failure must be an injected activation failpoint")

        complete_bundle = _validated_bundle(candidate)

        if failure is InjectedActivationFailure.BEFORE_ACTIVATION:
            self._last_outcome = ActivationOutcome(
                candidate=complete_bundle,
                state=ActivationState.INACTIVE,
            )
            raise ActivationFailedBeforeActivation(
                "candidate failed before the in-memory pointer changed"
            )

        if failure is InjectedActivationFailure.ACK_LOST_AFTER_ACTIVATION:
            self._pointer = complete_bundle
            self._pending = complete_bundle
            self._last_outcome = ActivationOutcome(
                candidate=complete_bundle,
                state=ActivationState.RECONCILIATION_REQUIRED,
            )
            raise ActivationAcknowledgmentLost(
                "pointer changed; exact read-back is required to resolve the outcome"
            )

        # Validate every record before replacing the synthetic pointer.
        # No component becomes active during validation.
        self._pointer = complete_bundle
        self._active = complete_bundle
        self._last_outcome = ActivationOutcome(
            candidate=complete_bundle,
            state=ActivationState.ACTIVE,
        )
        return complete_bundle

    def reconcile(self, read_back: object) -> ActiveBundle:
        """Resolve a lost acknowledgment only from the exact pointer value."""

        if self._pending is None:
            raise RestoreContractError("there is no unresolved activation to reconcile")
        if type(read_back) is not ActiveBundle:
            raise ActivationOutcomeUnresolved(
                "read-back does not exactly match the unresolved pointer"
            )
        try:
            validated_read_back = _validated_bundle(
                RestoreCandidate(read_back.identity, read_back.components)
            )
        except RestoreContractError as exc:
            raise ActivationOutcomeUnresolved(
                "read-back contains an invalid bundle identity or component"
            ) from exc
        if validated_read_back != self._pending or validated_read_back != self._pointer:
            raise ActivationOutcomeUnresolved(
                "read-back does not exactly match the unresolved pointer"
            )

        self._active = self._pending
        self._pending = None
        self._last_outcome = ActivationOutcome(
            candidate=self._active,
            state=ActivationState.ACTIVE,
        )
        return self._active


def _validate_identity(identity: object, *, label: str) -> BundleIdentity:
    if type(identity) is not BundleIdentity:
        raise RestoreContractError(f"{label} must be a BundleIdentity")

    for field_name in ("tenant", "backup_id", "marker"):
        value = getattr(identity, field_name)
        if not isinstance(value, str) or value == "":
            raise RestoreContractError(f"{label} {field_name} must be a non-empty string")

    if type(identity.generation) is not int or identity.generation < 1:
        raise RestoreContractError(f"{label} generation must be a positive integer")
    return identity


def _validated_bundle(candidate: object) -> ActiveBundle:
    if type(candidate) is not RestoreCandidate:
        raise RestoreContractError("candidate must be a RestoreCandidate")

    expected = _validate_identity(candidate.expected_identity, label="expected identity")
    if not isinstance(candidate.components, tuple):
        raise RestoreContractError("components must be an immutable tuple")

    records = candidate.components
    if any(type(record) is not ComponentRecord for record in records):
        raise RestoreContractError("each component must be a ComponentRecord")

    names = tuple(record.component for record in records)
    if any(not isinstance(name, str) or name not in REQUIRED_COMPONENTS for name in names):
        raise RestoreContractError("candidate contains an unknown component")
    if len(set(names)) != len(names):
        raise RestoreContractError("candidate contains a duplicate component")
    if set(names) != set(REQUIRED_COMPONENTS):
        raise RestoreContractError("candidate is missing a required component")

    by_name = {record.component: record for record in records}
    for component_name in REQUIRED_COMPONENTS:
        record = by_name[component_name]
        record_identity = BundleIdentity(
            tenant=record.tenant,
            backup_id=record.backup_id,
            generation=record.generation,
            marker=record.marker,
        )
        _validate_identity(record_identity, label=f"{component_name} identity")
        if record_identity != expected:
            raise RestoreContractError(
                f"{component_name} identity does not match the expected bundle identity"
            )

    # Return canonical component order. The caller swaps the active pointer only
    # after this fully validated immutable value has been constructed.
    return ActiveBundle(
        identity=expected,
        components=tuple(by_name[name] for name in REQUIRED_COMPONENTS),
    )
