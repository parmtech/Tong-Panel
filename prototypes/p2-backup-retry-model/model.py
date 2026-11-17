"""Non-normative in-memory model of P2 backup generation fencing.

This package stores only synthetic identities, digests, and states. It has no
storage, network, or Tong Panel product integration.
"""

from __future__ import annotations

import hashlib
import math
from collections.abc import Callable
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


class InvalidTransferPolicy(BackupModelError):
    pass


class InvalidTransferClock(BackupModelError):
    pass


class InvalidTransferProgress(BackupModelError):
    pass


class TransferConcurrencyLimit(BackupModelError):
    pass


class TransferAttemptLimit(BackupModelError):
    pass


class TransferBackoffNotElapsed(BackupModelError):
    pass


class TransferReconciliationPending(BackupModelError):
    pass


class TransferNotRetryable(BackupModelError):
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
    RECONCILIATION_REQUIRED = "reconciliation_required"
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

    def finish_failed_attempt(self, attempt_id: int, *, reconciliation_required: bool) -> None:
        """Mark a synthetic attempt failed or awaiting reconciliation."""

        if type(reconciliation_required) is not bool:
            raise BackupModelError("reconciliation_required must be a boolean")
        attempt = self._get_attempt(attempt_id)
        if attempt.state not in (AttemptState.ACTIVE, AttemptState.FAILED):
            raise AttemptTerminal("attempt cannot transition to a failed state")
        attempt.state = (
            AttemptState.RECONCILIATION_REQUIRED
            if reconciliation_required
            else AttemptState.FAILED
        )

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


@dataclass(frozen=True)
class TransferLimits:
    """Caller-supplied limits for one offline transfer simulation."""

    max_size_bytes: int
    max_duration_seconds: float
    max_attempts: int
    backoff_seconds: tuple[float, ...]
    max_concurrent_transfers: int

    def __post_init__(self) -> None:
        if type(self.max_size_bytes) is not int or self.max_size_bytes < 1:
            raise InvalidTransferPolicy("max_size_bytes must be a positive integer")
        if (
            isinstance(self.max_duration_seconds, bool)
            or not isinstance(self.max_duration_seconds, (int, float))
            or not math.isfinite(self.max_duration_seconds)
            or self.max_duration_seconds <= 0
        ):
            raise InvalidTransferPolicy("max_duration_seconds must be finite and positive")
        if type(self.max_attempts) is not int or self.max_attempts < 1:
            raise InvalidTransferPolicy("max_attempts must be a positive integer")
        if (
            type(self.max_concurrent_transfers) is not int
            or self.max_concurrent_transfers < 1
        ):
            raise InvalidTransferPolicy("max_concurrent_transfers must be a positive integer")
        if not isinstance(self.backoff_seconds, tuple):
            raise InvalidTransferPolicy("backoff_seconds must be an immutable tuple")
        if len(self.backoff_seconds) != self.max_attempts - 1:
            raise InvalidTransferPolicy("provide one backoff delay for each permitted retry")
        for delay in self.backoff_seconds:
            if (
                isinstance(delay, bool)
                or not isinstance(delay, (int, float))
                or not math.isfinite(delay)
                or delay < 0
            ):
                raise InvalidTransferPolicy("backoff delays must be finite and non-negative")


class TransferState(Enum):
    ACTIVE = "active"
    VERIFIED = "verified"
    FAILED = "failed"
    CLEANED = "partial_object_removal_verified"
    RECONCILIATION_REQUIRED = "reconciliation_required"
    STALE = "stale_reconciliation_required"


class TransferFailureReason(Enum):
    SIZE_LIMIT = "size_limit"
    DURATION_LIMIT = "duration_limit"
    INTERRUPTED = "interrupted"
    INCOMPLETE = "incomplete"
    DIGEST_MISMATCH = "digest_mismatch"
    GENERATION_CONFLICT = "generation_conflict"
    STALE_GENERATION = "stale_generation"


@dataclass(frozen=True)
class TransferSnapshot:
    attempt_id: int
    attempt_number: int
    identity: Identity
    expected_size_bytes: int
    transferred_bytes: int
    duration_seconds: float
    state: TransferState
    failure_reason: TransferFailureReason | None
    cleanup_verified: bool | None


@dataclass
class _TransferOperation:
    identity: Identity
    expected_size_bytes: int
    attempt_ids: list[int]


@dataclass
class _TransferAttempt:
    attempt_id: int
    operation_id: int
    attempt_number: int
    started_at: float
    updated_at: float
    transferred_bytes: int = 0
    state: TransferState = TransferState.ACTIVE
    failure_reason: TransferFailureReason | None = None
    cleanup_verified: bool | None = None
    finished_at: float | None = None


class TransferSimulator:
    """Provider-neutral transfer outcomes driven by caller-supplied limits and clock.

    No payload is stored and no remote object is contacted. ``cleanup_verified``
    is an injected test observation, not a cleanup operation.
    """

    def __init__(
        self,
        model: Model,
        limits: TransferLimits,
        clock: Callable[[], float],
    ) -> None:
        if (
            not isinstance(model, Model)
            or not isinstance(limits, TransferLimits)
            or not callable(clock)
        ):
            raise InvalidTransferPolicy("a Model, explicit TransferLimits, and clock are required")
        self._model = model
        self._limits = limits
        self._clock = clock
        self._last_clock_value: float | None = None
        self._next_operation_id = 0
        self._operations: dict[int, _TransferOperation] = {}
        self._attempts: dict[int, _TransferAttempt] = {}
        self._active_attempts: set[int] = set()

    def start_transfer(self, identity: Identity, expected_size_bytes: int) -> int:
        """Start a synthetic transfer, returning its model attempt ID.

        Oversize requests are recorded as terminal failures before any transfer
        activity, so they consume no concurrency slot and cannot be verified.
        """

        if not isinstance(identity, Identity):
            raise InvalidIdentity("an immutable Identity value is required")
        if type(expected_size_bytes) is not int or expected_size_bytes < 0:
            raise InvalidTransferProgress("expected_size_bytes must be a non-negative integer")
        if (
            expected_size_bytes <= self._limits.max_size_bytes
            and len(self._active_attempts) >= self._limits.max_concurrent_transfers
        ):
            raise TransferConcurrencyLimit("simultaneous transfer cap reached")

        now = self._now()
        self._next_operation_id += 1
        operation_id = self._next_operation_id
        operation = _TransferOperation(identity, expected_size_bytes, [])
        self._operations[operation_id] = operation
        attempt_id = self._new_attempt(operation_id, operation, now)
        if expected_size_bytes > self._limits.max_size_bytes:
            self._settle_failure(
                self._attempts[attempt_id], TransferFailureReason.SIZE_LIMIT, None, now
            )
        else:
            self._active_attempts.add(attempt_id)
        return attempt_id

    def retry_transfer(self, previous_attempt_id: int) -> int:
        """Retry a resolved attempt using its original immutable identity and size."""

        previous = self._get_attempt(previous_attempt_id)
        operation = self._operations[previous.operation_id]
        if operation.attempt_ids[-1] != previous_attempt_id:
            raise TransferNotRetryable("only the latest attempt in an operation can be retried")
        if previous.state is TransferState.RECONCILIATION_REQUIRED:
            raise TransferReconciliationPending("resolve the partial object before retrying")
        if previous.state not in (TransferState.FAILED, TransferState.CLEANED):
            raise TransferNotRetryable("only a failed, resolved attempt can be retried")
        if previous.failure_reason is TransferFailureReason.SIZE_LIMIT:
            raise TransferNotRetryable("an oversize transfer cannot be retried unchanged")
        if len(operation.attempt_ids) >= self._limits.max_attempts:
            raise TransferAttemptLimit("maximum transfer attempts reached")

        now = self._now()
        required_at = previous.finished_at + self._limits.backoff_seconds[previous.attempt_number - 1]
        if now < required_at:
            raise TransferBackoffNotElapsed("configured retry backoff has not elapsed")
        if len(self._active_attempts) >= self._limits.max_concurrent_transfers:
            raise TransferConcurrencyLimit("simultaneous transfer cap reached")
        attempt_id = self._new_attempt(previous.operation_id, operation, now)
        self._active_attempts.add(attempt_id)
        return attempt_id

    def report_progress(
        self,
        attempt_id: int,
        transferred_bytes: int,
        *,
        cleanup_verified: bool | None = None,
    ) -> TransferSnapshot:
        """Record synthetic cumulative progress and enforce the duration bound."""

        attempt = self._require_active(attempt_id)
        if (
            type(transferred_bytes) is not int
            or not attempt.transferred_bytes
            <= transferred_bytes
            <= self._operations[attempt.operation_id].expected_size_bytes
        ):
            raise InvalidTransferProgress(
                "progress must be an integer within the declared transfer size"
            )
        self._validate_cleanup_observation(cleanup_verified)
        now = self._now()
        attempt.transferred_bytes = transferred_bytes
        attempt.updated_at = now
        if now - attempt.started_at > self._limits.max_duration_seconds:
            self._settle_failure(attempt, TransferFailureReason.DURATION_LIMIT, cleanup_verified, now)
        return self._snapshot(attempt)

    def finish_transfer(
        self,
        attempt_id: int,
        content: bytes,
        *,
        cleanup_verified: bool | None = None,
    ) -> TransferSnapshot:
        """Verify a complete synthetic transfer before publishing its identity."""

        attempt = self._require_active(attempt_id)
        if not isinstance(content, bytes):
            raise InvalidTransferProgress("content must be synthetic bytes")
        self._validate_cleanup_observation(cleanup_verified)
        now = self._now()
        attempt.updated_at = now
        operation = self._operations[attempt.operation_id]

        if now - attempt.started_at > self._limits.max_duration_seconds:
            self._settle_failure(attempt, TransferFailureReason.DURATION_LIMIT, cleanup_verified, now)
        elif len(content) > self._limits.max_size_bytes:
            self._settle_failure(attempt, TransferFailureReason.SIZE_LIMIT, cleanup_verified, now)
        elif (
            len(content) != operation.expected_size_bytes
            or attempt.transferred_bytes != operation.expected_size_bytes
        ):
            self._settle_failure(attempt, TransferFailureReason.INCOMPLETE, cleanup_verified, now)
        elif hashlib.sha256(content).hexdigest() != attempt.identity.expected_sha256:
            self._settle_failure(
                attempt, TransferFailureReason.DIGEST_MISMATCH, cleanup_verified, now
            )
        else:
            try:
                self._model.verify_and_publish(attempt_id, content)
            except StaleAttempt:
                attempt.state = TransferState.STALE
                attempt.failure_reason = TransferFailureReason.STALE_GENERATION
                attempt.finished_at = now
                self._active_attempts.discard(attempt_id)
            except GenerationConflict:
                self._settle_failure(
                    attempt, TransferFailureReason.GENERATION_CONFLICT, cleanup_verified, now
                )
            except DigestMismatch:
                self._settle_failure(
                    attempt, TransferFailureReason.DIGEST_MISMATCH, cleanup_verified, now
                )
            else:
                attempt.state = TransferState.VERIFIED
                attempt.finished_at = now
                self._active_attempts.discard(attempt_id)
        return self._snapshot(attempt)

    def interrupt_transfer(
        self,
        attempt_id: int,
        *,
        cleanup_verified: bool | None = None,
    ) -> TransferSnapshot:
        """End a transfer after interruption, with explicit partial-object handling."""

        attempt = self._require_active(attempt_id)
        self._validate_cleanup_observation(cleanup_verified)
        now = self._now()
        attempt.updated_at = now
        reason = (
            TransferFailureReason.DURATION_LIMIT
            if now - attempt.started_at > self._limits.max_duration_seconds
            else TransferFailureReason.INTERRUPTED
        )
        self._settle_failure(attempt, reason, cleanup_verified, now)
        return self._snapshot(attempt)

    def transfer(self, attempt_id: int) -> TransferSnapshot:
        return self._snapshot(self._get_attempt(attempt_id))

    def _new_attempt(
        self,
        operation_id: int,
        operation: _TransferOperation,
        now: float,
    ) -> int:
        attempt_id = self._model.begin_attempt(operation.identity)
        operation.attempt_ids.append(attempt_id)
        self._attempts[attempt_id] = _TransferAttempt(
            attempt_id=attempt_id,
            operation_id=operation_id,
            attempt_number=len(operation.attempt_ids),
            started_at=now,
            updated_at=now,
        )
        return attempt_id

    def _settle_failure(
        self,
        attempt: _TransferAttempt,
        reason: TransferFailureReason,
        cleanup_verified: bool | None,
        now: float,
    ) -> None:
        has_remote_bytes = attempt.transferred_bytes > 0
        if has_remote_bytes and cleanup_verified is True:
            attempt.state = TransferState.CLEANED
            attempt.cleanup_verified = True
            reconciliation_required = False
        elif has_remote_bytes:
            attempt.state = TransferState.RECONCILIATION_REQUIRED
            attempt.cleanup_verified = cleanup_verified
            reconciliation_required = True
        else:
            attempt.state = TransferState.FAILED
            attempt.cleanup_verified = None
            reconciliation_required = False
        attempt.failure_reason = reason
        attempt.finished_at = now
        model_state = self._model.attempt(attempt.attempt_id).state
        if model_state is AttemptState.ACTIVE or (
            reconciliation_required and model_state is AttemptState.FAILED
        ):
            self._model.finish_failed_attempt(
                attempt.attempt_id,
                reconciliation_required=reconciliation_required,
            )
        self._active_attempts.discard(attempt.attempt_id)

    def _snapshot(self, attempt: _TransferAttempt) -> TransferSnapshot:
        return TransferSnapshot(
            attempt_id=attempt.attempt_id,
            attempt_number=attempt.attempt_number,
            identity=self._operations[attempt.operation_id].identity,
            expected_size_bytes=self._operations[attempt.operation_id].expected_size_bytes,
            transferred_bytes=attempt.transferred_bytes,
            duration_seconds=attempt.updated_at - attempt.started_at,
            state=attempt.state,
            failure_reason=attempt.failure_reason,
            cleanup_verified=attempt.cleanup_verified,
        )

    def _require_active(self, attempt_id: int) -> _TransferAttempt:
        attempt = self._get_attempt(attempt_id)
        if attempt.state is not TransferState.ACTIVE:
            raise AttemptTerminal("transfer attempt is already terminal")
        return attempt

    def _get_attempt(self, attempt_id: int) -> _TransferAttempt:
        if type(attempt_id) is not int:
            raise UnknownAttempt(f"unknown attempt {attempt_id!r}")
        try:
            return self._attempts[attempt_id]
        except KeyError as exc:
            raise UnknownAttempt(f"unknown attempt {attempt_id}") from exc

    @staticmethod
    def _validate_cleanup_observation(cleanup_verified: bool | None) -> None:
        if cleanup_verified is not None and type(cleanup_verified) is not bool:
            raise InvalidTransferProgress("cleanup_verified must be true, false, or unspecified")

    def _now(self) -> float:
        value = self._clock()
        if (
            isinstance(value, bool)
            or not isinstance(value, (int, float))
            or not math.isfinite(value)
        ):
            raise InvalidTransferClock("injected clock must return a finite number")
        now = float(value)
        if self._last_clock_value is not None and now < self._last_clock_value:
            raise InvalidTransferClock("injected clock must be monotonic")
        self._last_clock_value = now
        return now
