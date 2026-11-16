"""Non-normative in-memory model of P2 backup generation fencing.

This package stores only synthetic identities, digests, and states. It has no
storage, network, or Tong Panel product integration.
"""

from __future__ import annotations

import hashlib
from dataclasses import dataclass
from enum import Enum


class BackupModelError(Exception):
    """Base class for modeled state-transition denials."""


class InvalidIdentity(BackupModelError):
    pass


class UnknownAttempt(BackupModelError):
    pass


class AttemptTerminal(BackupModelError):
    pass


class DigestMismatch(BackupModelError):
    pass


class StaleAttempt(BackupModelError):
    pass


class GenerationConflict(BackupModelError):
    pass


class DeleteOutOfScope(BackupModelError):
    pass


@dataclass(frozen=True)
class Identity:
    """Immutable logical backup stream generation bound to expected content."""

    backup_id: str
    generation: int
    expected_sha256: str

    def __post_init__(self) -> None:
        if (
            not isinstance(self.backup_id, str)
            or not self.backup_id
            or type(self.generation) is not int
            or self.generation < 1
        ):
            raise InvalidIdentity("backup ID and positive integer generation are required")
        if (
            not isinstance(self.expected_sha256, str)
            or len(self.expected_sha256) != 64
            or self.expected_sha256 != self.expected_sha256.lower()
            or any(char not in "0123456789abcdef" for char in self.expected_sha256)
        ):
            raise InvalidIdentity("expected SHA-256 must be 64 lowercase hexadecimal characters")


def identity_for_content(backup_id: str, generation: int, content: bytes) -> Identity:
    """Create a synthetic identity from fixture bytes."""

    return Identity(backup_id, generation, hashlib.sha256(content).hexdigest())


class AttemptState(Enum):
    ACTIVE = "active"
    VERIFIED = "verified"
    # A stale attempt is terminal. The caller must reconcile current state and
    # start a new attempt if further work is appropriate.
    STALE = "stale_reconciliation_required"
    FAILED = "failed"


@dataclass
class _Attempt:
    identity: Identity
    state: AttemptState


@dataclass
class _Artifact:
    identity: Identity
    published_by: int


@dataclass(frozen=True)
class AttemptSnapshot:
    attempt_id: int
    identity: Identity
    state: AttemptState


@dataclass(frozen=True)
class ArtifactSnapshot:
    identity: Identity
    published_by: int


class Model:
    """Synthetic attempt and current-artifact state held only in memory."""

    def __init__(self) -> None:
        self._next_attempt = 0
        self._attempts: dict[int, _Attempt] = {}
        self._current: dict[str, _Artifact] = {}

    def begin_attempt(self, identity: Identity) -> int:
        if not isinstance(identity, Identity):
            raise InvalidIdentity("an immutable Identity value is required")
        self._next_attempt += 1
        attempt_id = self._next_attempt
        self._attempts[attempt_id] = _Attempt(identity=identity, state=AttemptState.ACTIVE)
        return attempt_id

    def verify_and_publish(self, attempt_id: int, content: bytes) -> None:
        """Verify bound bytes and generation before changing the current artifact."""

        attempt = self._get_attempt(attempt_id)
        self._require_active(attempt)
        if hashlib.sha256(content).hexdigest() != attempt.identity.expected_sha256:
            attempt.state = AttemptState.FAILED
            raise DigestMismatch("content does not match the attempt's bound digest")

        current = self._current.get(attempt.identity.backup_id)
        if current is not None:
            if current.identity.generation > attempt.identity.generation:
                attempt.state = AttemptState.STALE
                raise StaleAttempt("a newer generation is already current")
            if current.identity.generation == attempt.identity.generation:
                if current.identity == attempt.identity:
                    attempt.state = AttemptState.VERIFIED
                    return
                attempt.state = AttemptState.FAILED
                raise GenerationConflict("generation is bound to another identity digest")

        self._current[attempt.identity.backup_id] = _Artifact(
            identity=attempt.identity,
            published_by=attempt_id,
        )
        attempt.state = AttemptState.VERIFIED

    def reject_stale_delete(self, attempt_id: int) -> None:
        """Fence a stale delete request without modeling deletion policy.

        If a newer generation is current, this attempt becomes stale and leaves
        that artifact untouched. Otherwise, deletion remains outside this
        model and the method raises without changing attempt or artifact state.
        """

        attempt = self._get_attempt(attempt_id)
        if attempt.state not in (AttemptState.ACTIVE, AttemptState.VERIFIED):
            raise AttemptTerminal("attempt is already terminal")

        current = self._current.get(attempt.identity.backup_id)
        if current is not None and current.identity.generation > attempt.identity.generation:
            attempt.state = AttemptState.STALE
            raise StaleAttempt("a newer generation cannot be deleted by this attempt")
        raise DeleteOutOfScope("non-stale deletion policy is outside this model")

    def attempt(self, attempt_id: int) -> AttemptSnapshot:
        value = self._get_attempt(attempt_id)
        return AttemptSnapshot(attempt_id, value.identity, value.state)

    def current(self, backup_id: str) -> ArtifactSnapshot | None:
        value = self._current.get(backup_id)
        if value is None:
            return None
        return ArtifactSnapshot(value.identity, value.published_by)

    def _get_attempt(self, attempt_id: int) -> _Attempt:
        if type(attempt_id) is not int:
            raise UnknownAttempt(f"unknown attempt {attempt_id!r}")
        try:
            return self._attempts[attempt_id]
        except KeyError as exc:
            raise UnknownAttempt(f"unknown attempt {attempt_id}") from exc

    @staticmethod
    def _require_active(attempt: _Attempt) -> None:
        if attempt.state is not AttemptState.ACTIVE:
            raise AttemptTerminal("attempt is already terminal")
