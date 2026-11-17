"""Synthetic reconciliation model for a preselected backup deletion intent.

This module stores immutable fixture identities and injected observations only.
It never deletes an artifact or contacts a provider.
"""

from __future__ import annotations

from dataclasses import dataclass
from enum import Enum


_GENERIC_DENIAL = "delete reconciliation denied"


class DeleteIntentDenied(ValueError):
    """A delete intent or observation is invalid, unknown, or conflicts with state."""

    def __init__(self) -> None:
        super().__init__(_GENERIC_DENIAL)


class IntentState(Enum):
    REQUESTED = "requested"
    RECONCILIATION_REQUIRED = "reconciliation_required"
    VERIFIED_ABSENT = "verified_absent"


class ObservedPresence(Enum):
    VERIFIED_ABSENT = "verified_absent"
    PRESENT = "present"
    UNKNOWN = "unknown"
    UNAVAILABLE = "unavailable"


@dataclass(frozen=True)
class VerifiedArtifact:
    """Opaque fixture identity declared to represent a previously verified artifact."""

    tenant_id: str
    artifact_id: str


@dataclass(frozen=True)
class DeleteObservation:
    """Injected presence observation bound to the exact synthetic artifact identity."""

    artifact: VerifiedArtifact
    presence: ObservedPresence


@dataclass(frozen=True)
class DeleteIntent:
    """Immutable operation binding and current synthetic reconciliation state."""

    operation_id: str
    artifact: VerifiedArtifact
    state: IntentState
    last_observation: DeleteObservation | None = None


@dataclass(frozen=True)
class DeleteBatchIntent:
    """Immutable reconciliation state for one fixed preselected artifact set."""

    operation_id: str
    artifacts: tuple[VerifiedArtifact, ...]
    state: IntentState
    last_observations: tuple[DeleteObservation, ...] | None = None


class RetentionDeleteModel:
    """Track single and fixed-set delete intents without performing deletion."""

    def __init__(self) -> None:
        self._intents: dict[str, DeleteIntent] = {}
        self._batch_intents: dict[str, DeleteBatchIntent] = {}

    @property
    def intents(self) -> tuple[DeleteIntent, ...]:
        return tuple(self._intents.values())

    @property
    def batch_intents(self) -> tuple[DeleteBatchIntent, ...]:
        return tuple(self._batch_intents.values())

    def request_delete(self, operation_id: object, artifact: object) -> DeleteIntent:
        """Create or replay an intent without resetting any unresolved state."""

        operation = _require_identifier(operation_id)
        target = _require_artifact(artifact)
        if operation in self._batch_intents:
            raise DeleteIntentDenied()
        existing = self._intents.get(operation)
        if existing is not None:
            if existing.artifact != target:
                raise DeleteIntentDenied()
            return existing

        intent = DeleteIntent(
            operation_id=operation,
            artifact=target,
            state=IntentState.REQUESTED,
        )
        self._intents[operation] = intent
        return intent

    def observe(
        self,
        operation_id: object,
        artifact: object,
        observation: object,
    ) -> DeleteIntent:
        """Apply injected evidence; only exact verified absence completes an intent."""

        operation = _require_identifier(operation_id)
        target = _require_artifact(artifact)
        evidence = _require_observation(observation)
        intent = self._intents.get(operation)
        if intent is None or intent.artifact != target or evidence.artifact != target:
            raise DeleteIntentDenied()

        if intent.state is IntentState.VERIFIED_ABSENT:
            if evidence.presence is ObservedPresence.VERIFIED_ABSENT:
                return intent
            # A completed intent is terminal; contradictory later observations
            # cannot reopen it or replace its exact target.
            raise DeleteIntentDenied()

        if evidence.presence is ObservedPresence.VERIFIED_ABSENT:
            state = IntentState.VERIFIED_ABSENT
        else:
            state = IntentState.RECONCILIATION_REQUIRED

        updated = DeleteIntent(
            operation_id=intent.operation_id,
            artifact=intent.artifact,
            state=state,
            last_observation=evidence,
        )
        self._intents[operation] = updated
        return updated

    def request_delete_batch(
        self,
        operation_id: object,
        artifacts: object,
    ) -> DeleteBatchIntent:
        """Create or replay an immutable, caller-preselected artifact set."""

        operation = _require_identifier(operation_id)
        targets = _require_artifact_set(artifacts)
        if operation in self._intents:
            raise DeleteIntentDenied()
        existing = self._batch_intents.get(operation)
        if existing is not None:
            if existing.artifacts != targets:
                raise DeleteIntentDenied()
            return existing

        intent = DeleteBatchIntent(
            operation_id=operation,
            artifacts=targets,
            state=IntentState.REQUESTED,
        )
        self._batch_intents[operation] = intent
        return intent

    def observe_batch(
        self,
        operation_id: object,
        observations: object,
    ) -> DeleteBatchIntent:
        """Apply one exact full-set observation without changing its target set."""

        operation = _require_identifier(operation_id)
        evidence = _require_observation_set(observations)
        intent = self._batch_intents.get(operation)
        if intent is None:
            raise DeleteIntentDenied()

        expected = set(intent.artifacts)
        observed = {item.artifact for item in evidence}
        if observed != expected:
            raise DeleteIntentDenied()

        all_absent = all(
            item.presence is ObservedPresence.VERIFIED_ABSENT for item in evidence
        )
        if intent.state is IntentState.VERIFIED_ABSENT:
            if all_absent:
                return intent
            # A completed fixed set is terminal; later contradictions cannot
            # reopen or retarget it.
            raise DeleteIntentDenied()

        state = (
            IntentState.VERIFIED_ABSENT
            if all_absent
            else IntentState.RECONCILIATION_REQUIRED
        )
        updated = DeleteBatchIntent(
            operation_id=intent.operation_id,
            artifacts=intent.artifacts,
            state=state,
            last_observations=evidence,
        )
        self._batch_intents[operation] = updated
        return updated


def _require_identifier(value: object) -> str:
    if type(value) is not str or value == "":
        raise DeleteIntentDenied()
    return value


def _require_artifact(value: object) -> VerifiedArtifact:
    if type(value) is not VerifiedArtifact:
        raise DeleteIntentDenied()
    _require_identifier(value.tenant_id)
    _require_identifier(value.artifact_id)
    return value


def _require_artifact_set(value: object) -> tuple[VerifiedArtifact, ...]:
    """Validate and canonically order a nonempty immutable unique artifact set."""

    if type(value) is not tuple or not value:
        raise DeleteIntentDenied()
    artifacts = tuple(_require_artifact(item) for item in value)
    if len(set(artifacts)) != len(artifacts):
        raise DeleteIntentDenied()
    return tuple(sorted(artifacts, key=lambda item: (item.tenant_id, item.artifact_id)))


def _require_observation(value: object) -> DeleteObservation:
    if type(value) is not DeleteObservation:
        raise DeleteIntentDenied()
    _require_artifact(value.artifact)
    if type(value.presence) is not ObservedPresence:
        raise DeleteIntentDenied()
    return value


def _require_observation_set(value: object) -> tuple[DeleteObservation, ...]:
    """Validate and canonically order an immutable unique observation tuple."""

    if type(value) is not tuple or not value:
        raise DeleteIntentDenied()
    observations = tuple(_require_observation(item) for item in value)
    artifacts = tuple(item.artifact for item in observations)
    if len(set(artifacts)) != len(artifacts):
        raise DeleteIntentDenied()
    return tuple(
        sorted(
            observations,
            key=lambda item: (item.artifact.tenant_id, item.artifact.artifact_id),
        )
    )
