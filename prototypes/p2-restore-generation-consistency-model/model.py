"""Synthetic in-memory contract for complete restore-bundle activation."""

from __future__ import annotations

from dataclasses import dataclass


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


class ActivationStore:
    """One in-memory active pointer; validation completes before its one swap."""

    def __init__(self) -> None:
        self._active: ActiveBundle | None = None

    @property
    def active(self) -> ActiveBundle | None:
        return self._active

    def activate(self, candidate: RestoreCandidate) -> ActiveBundle:
        """Validate every record, then replace the active bundle in one step."""

        complete_bundle = _validated_bundle(candidate)
        # The model exposes a single pointer. No component is made active during
        # validation, so any denial preserves the prior bundle object unchanged.
        self._active = complete_bundle
        return complete_bundle


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
