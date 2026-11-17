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


class InvalidCheckpoint(BackupModelError):
    pass


@dataclass(frozen=True)
class Identity:
    """Immutable tenant-scoped backup generation bound to expected content."""

    tenant_id: str
    backup_id: str
    generation: int
    expected_sha256: str

    def __post_init__(self) -> None:
        if (
            not isinstance(self.tenant_id, str)
            or not self.tenant_id
            or not isinstance(self.backup_id, str)
            or not self.backup_id
            or type(self.generation) is not int
            or self.generation < 1
        ):
            raise InvalidIdentity(
                "tenant ID, backup ID, and positive integer generation are required"
            )
        if (
            not isinstance(self.expected_sha256, str)
            or len(self.expected_sha256) != 64
            or self.expected_sha256 != self.expected_sha256.lower()
            or any(char not in "0123456789abcdef" for char in self.expected_sha256)
        ):
            raise InvalidIdentity("expected SHA-256 must be 64 lowercase hexadecimal characters")


def identity_for_content(
    tenant_id: str,
    backup_id: str,
    generation: int,
    content: bytes,
) -> Identity:
    """Create a synthetic identity from fixture bytes."""

    return Identity(tenant_id, backup_id, generation, hashlib.sha256(content).hexdigest())


def _artifact_key(identity: Identity) -> tuple[str, str]:
    return (identity.tenant_id, identity.backup_id)


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
        self._current: dict[tuple[str, str], _Artifact] = {}

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

        artifact_key = _artifact_key(attempt.identity)
        current = self._current.get(artifact_key)
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

        self._current[artifact_key] = _Artifact(
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

        current = self._current.get(_artifact_key(attempt.identity))
        if current is not None and current.identity.generation > attempt.identity.generation:
            attempt.state = AttemptState.STALE
            raise StaleAttempt("a newer generation cannot be deleted by this attempt")
        raise DeleteOutOfScope("non-stale deletion policy is outside this model")

    def attempt(self, attempt_id: int) -> AttemptSnapshot:
        value = self._get_attempt(attempt_id)
        return AttemptSnapshot(attempt_id, value.identity, value.state)

    def current(self, tenant_id: str, backup_id: str) -> ArtifactSnapshot | None:
        if (
            not isinstance(tenant_id, str)
            or not tenant_id
            or not isinstance(backup_id, str)
            or not backup_id
        ):
            raise InvalidIdentity("tenant ID and backup ID are required for lookup")
        value = self._current.get((tenant_id, backup_id))
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


@dataclass(frozen=True)
class _CheckpointOperation:
    operation_id: int
    identity: Identity
    expected_size_bytes: int
    attempt_ids: tuple[int, ...]


@dataclass(frozen=True)
class _CheckpointTransferAttempt:
    attempt_id: int
    operation_id: int
    attempt_number: int
    started_at: float
    updated_at: float
    transferred_bytes: int
    state: TransferState
    failure_reason: TransferFailureReason | None
    cleanup_verified: bool | None
    finished_at: float | None


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
        *,
        clock_epoch: str | None = None,
    ) -> None:
        if (
            not isinstance(model, Model)
            or not isinstance(limits, TransferLimits)
            or not callable(clock)
        ):
            raise InvalidTransferPolicy("a Model, explicit TransferLimits, and clock are required")
        if clock_epoch is not None and (type(clock_epoch) is not str or not clock_epoch):
            raise InvalidTransferClock("clock_epoch must be a non-empty synthetic epoch label")
        self._model = model
        self._limits = limits
        self._clock = clock
        self._clock_epoch = clock_epoch
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
        elif hashlib.sha256(content).hexdigest() != operation.identity.expected_sha256:
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

    def checkpoint(self) -> SyntheticCheckpoint:
        """Capture terminal coupled state as an in-memory synthetic value.

        This is not a persistence or serialization format. An explicit epoch
        label and an already-observed clock anchor are required so a caller
        cannot silently carry elapsed downtime into this model.
        """

        if self._clock_epoch is None:
            raise InvalidCheckpoint("checkpoint requires an explicit synthetic clock epoch")
        if self._active_attempts:
            raise InvalidCheckpoint("active transfer attempts cannot be checkpointed")
        if self._last_clock_value is None:
            raise InvalidCheckpoint("checkpoint requires an observed synthetic clock anchor")

        checkpoint = SyntheticCheckpoint(
            clock_epoch=self._clock_epoch,
            clock_anchor=self._last_clock_value,
            limits=self._limits,
            next_attempt_id=self._model._next_attempt,
            model_attempts=tuple(
                AttemptSnapshot(attempt_id, value.identity, value.state)
                for attempt_id, value in sorted(self._model._attempts.items())
            ),
            current_artifacts=tuple(
                ArtifactSnapshot(value.identity, value.published_by)
                for _, value in sorted(self._model._current.items())
            ),
            next_operation_id=self._next_operation_id,
            operations=tuple(
                _CheckpointOperation(
                    operation_id,
                    value.identity,
                    value.expected_size_bytes,
                    tuple(value.attempt_ids),
                )
                for operation_id, value in sorted(self._operations.items())
            ),
            transfer_attempts=tuple(
                _CheckpointTransferAttempt(
                    attempt_id=value.attempt_id,
                    operation_id=value.operation_id,
                    attempt_number=value.attempt_number,
                    started_at=value.started_at,
                    updated_at=value.updated_at,
                    transferred_bytes=value.transferred_bytes,
                    state=value.state,
                    failure_reason=value.failure_reason,
                    cleanup_verified=value.cleanup_verified,
                    finished_at=value.finished_at,
                )
                for attempt_id, value in sorted(self._attempts.items())
            ),
        )
        checkpoint._validate()
        return checkpoint

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
        now = self._read_clock(self._clock)
        if self._last_clock_value is not None and now < self._last_clock_value:
            raise InvalidTransferClock("injected clock must be monotonic")
        self._last_clock_value = now
        return now

    @staticmethod
    def _read_clock(clock: Callable[[], float]) -> float:
        value = clock()
        if isinstance(value, bool) or not isinstance(value, (int, float)):
            raise InvalidTransferClock("injected clock must return a finite number")
        try:
            finite = math.isfinite(value)
        except OverflowError:
            finite = False
        if not finite:
            raise InvalidTransferClock("injected clock must return a finite number")
        return float(value)


@dataclass(frozen=True)
class SyntheticCheckpoint:
    """Immutable, in-memory value for deterministic synthetic reconstruction.

    This deliberately has no byte encoding, file format, schema version, or
    durable recovery semantics. It captures one coupled Model and
    TransferSimulator fixture only.
    """

    clock_epoch: str
    clock_anchor: float
    limits: TransferLimits
    next_attempt_id: int
    model_attempts: tuple[AttemptSnapshot, ...]
    current_artifacts: tuple[ArtifactSnapshot, ...]
    next_operation_id: int
    operations: tuple[_CheckpointOperation, ...]
    transfer_attempts: tuple[_CheckpointTransferAttempt, ...]

    def reconstruct(
        self,
        clock: Callable[[], float],
        *,
        clock_epoch: str,
    ) -> tuple[Model, TransferSimulator]:
        """Rebuild the coupled fixture at its exact saved synthetic clock anchor."""

        self._validate()
        if not callable(clock):
            raise InvalidCheckpoint("reconstruction requires a supplied synthetic clock")
        if type(clock_epoch) is not str or clock_epoch != self.clock_epoch:
            raise InvalidCheckpoint("supplied clock epoch does not match the checkpoint")
        if TransferSimulator._read_clock(clock) != float(self.clock_anchor):
            raise InvalidCheckpoint("supplied clock must begin at the saved synthetic anchor")

        model = Model()
        model._next_attempt = self.next_attempt_id
        model._attempts = {
            record.attempt_id: _Attempt(record.identity, record.state)
            for record in self.model_attempts
        }
        model._current = {
            _artifact_key(artifact.identity): _Artifact(artifact.identity, artifact.published_by)
            for artifact in self.current_artifacts
        }

        simulator = TransferSimulator(
            model,
            self.limits,
            clock,
            clock_epoch=self.clock_epoch,
        )
        simulator._last_clock_value = float(self.clock_anchor)
        simulator._next_operation_id = self.next_operation_id
        simulator._operations = {
            operation.operation_id: _TransferOperation(
                operation.identity,
                operation.expected_size_bytes,
                list(operation.attempt_ids),
            )
            for operation in self.operations
        }
        simulator._attempts = {
            record.attempt_id: _TransferAttempt(
                attempt_id=record.attempt_id,
                operation_id=record.operation_id,
                attempt_number=record.attempt_number,
                started_at=record.started_at,
                updated_at=record.updated_at,
                transferred_bytes=record.transferred_bytes,
                state=record.state,
                failure_reason=record.failure_reason,
                cleanup_verified=record.cleanup_verified,
                finished_at=record.finished_at,
            )
            for record in self.transfer_attempts
        }
        # Active attempts are rejected by validation, so the reconstructed
        # simulator begins with no occupied concurrency slots.
        simulator._active_attempts = set()
        return model, simulator

    def _validate(self) -> None:
        """Reject active or inconsistent coupled fixture state before copying."""

        def invalid(message: str) -> None:
            raise InvalidCheckpoint(message)

        def finite_number(value: object) -> bool:
            if isinstance(value, bool) or not isinstance(value, (int, float)):
                return False
            try:
                return math.isfinite(value)
            except OverflowError:
                return False

        if type(self.clock_epoch) is not str or not self.clock_epoch:
            invalid("checkpoint has no synthetic clock epoch")
        if not finite_number(self.clock_anchor):
            invalid("checkpoint clock anchor must be finite")
        if not isinstance(self.limits, TransferLimits):
            invalid("checkpoint is missing explicit transfer limits")
        for name, value in (
            ("next_attempt_id", self.next_attempt_id),
            ("next_operation_id", self.next_operation_id),
        ):
            if type(value) is not int or value < 0:
                invalid(f"checkpoint {name} must be a non-negative integer")
        for name, value in (
            ("model_attempts", self.model_attempts),
            ("current_artifacts", self.current_artifacts),
            ("operations", self.operations),
            ("transfer_attempts", self.transfer_attempts),
        ):
            if type(value) is not tuple:
                invalid(f"checkpoint {name} must be an immutable tuple")

        model_records: dict[int, AttemptSnapshot] = {}
        for record in self.model_attempts:
            if not isinstance(record, AttemptSnapshot):
                invalid("checkpoint contains an invalid model attempt record")
            if (
                type(record.attempt_id) is not int
                or record.attempt_id < 1
                or record.attempt_id > self.next_attempt_id
                or not isinstance(record.identity, Identity)
                or not isinstance(record.state, AttemptState)
                or record.state is AttemptState.ACTIVE
            ):
                invalid("checkpoint contains an active or invalid model attempt")
            if record.attempt_id in model_records:
                invalid("checkpoint contains duplicate model attempt IDs")
            model_records[record.attempt_id] = record
        if len(model_records) != self.next_attempt_id or (
            model_records and max(model_records) != self.next_attempt_id
        ):
            invalid("model attempt counter does not match its complete history")

        artifacts: dict[tuple[str, str], ArtifactSnapshot] = {}
        for artifact in self.current_artifacts:
            if not isinstance(artifact, ArtifactSnapshot) or not isinstance(
                artifact.identity, Identity
            ):
                invalid("checkpoint contains an invalid current artifact")
            artifact_key = _artifact_key(artifact.identity)
            if artifact_key in artifacts:
                invalid("checkpoint contains duplicate tenant/backup identities")
            publisher = model_records.get(artifact.published_by)
            if (
                type(artifact.published_by) is not int
                or publisher is None
                or publisher.identity != artifact.identity
                or publisher.state is not AttemptState.VERIFIED
            ):
                invalid("current artifact does not match a verified publisher")
            artifacts[artifact_key] = artifact

        operations: dict[int, _CheckpointOperation] = {}
        for operation in self.operations:
            if not isinstance(operation, _CheckpointOperation):
                invalid("checkpoint contains an invalid transfer operation")
            if (
                type(operation.operation_id) is not int
                or operation.operation_id < 1
                or operation.operation_id > self.next_operation_id
                or not isinstance(operation.identity, Identity)
                or type(operation.expected_size_bytes) is not int
                or operation.expected_size_bytes < 0
                or type(operation.attempt_ids) is not tuple
                or not operation.attempt_ids
            ):
                invalid("checkpoint contains an invalid transfer operation")
            if operation.operation_id in operations:
                invalid("checkpoint contains duplicate operation IDs")
            operations[operation.operation_id] = operation
        if len(operations) != self.next_operation_id or (
            operations and max(operations) != self.next_operation_id
        ):
            invalid("operation counter does not match its complete history")

        previous_initial_attempt_id: int | None = None
        for operation_id in sorted(operations):
            operation = operations[operation_id]
            if any(type(attempt_id) is not int or attempt_id < 1 for attempt_id in operation.attempt_ids):
                invalid("operation contains an invalid attempt ID")
            if any(
                earlier >= later
                for earlier, later in zip(operation.attempt_ids, operation.attempt_ids[1:])
            ):
                invalid("attempt IDs must increase within each operation")
            initial_attempt_id = operation.attempt_ids[0]
            if (
                previous_initial_attempt_id is not None
                and initial_attempt_id <= previous_initial_attempt_id
            ):
                invalid("initial attempt IDs must increase with operation IDs")
            previous_initial_attempt_id = initial_attempt_id

        transfer_records: dict[int, _CheckpointTransferAttempt] = {}
        for record in self.transfer_attempts:
            if not isinstance(record, _CheckpointTransferAttempt):
                invalid("checkpoint contains an invalid transfer attempt record")
            if type(record.attempt_id) is not int or record.attempt_id in transfer_records:
                invalid("checkpoint contains duplicate or invalid transfer attempt IDs")
            transfer_records[record.attempt_id] = record
        if set(transfer_records) != set(model_records):
            invalid("model and transfer attempt histories do not match")

        observed_operation_attempts: set[int] = set()
        for operation_id, operation in operations.items():
            if len(set(operation.attempt_ids)) != len(operation.attempt_ids):
                invalid("operation contains duplicate attempt IDs")
            if len(operation.attempt_ids) > self.limits.max_attempts:
                invalid("operation history exceeds its configured attempt cap")
            for index, attempt_id in enumerate(operation.attempt_ids, start=1):
                if type(attempt_id) is not int or attempt_id in observed_operation_attempts:
                    invalid("operation histories contain duplicate or invalid attempt IDs")
                observed_operation_attempts.add(attempt_id)
                record = transfer_records.get(attempt_id)
                model_record = model_records.get(attempt_id)
                if record is None or model_record is None:
                    invalid("operation references an unknown attempt")
                if (
                    type(record.operation_id) is not int
                    or record.operation_id < 1
                    or type(record.attempt_number) is not int
                    or record.attempt_number < 1
                    or record.attempt_number > self.limits.max_attempts
                    or record.operation_id != operation_id
                    or record.attempt_number != index
                    or model_record.identity != operation.identity
                ):
                    invalid("operation and attempt histories disagree")
                if (
                    type(record.transferred_bytes) is not int
                    or not 0 <= record.transferred_bytes <= operation.expected_size_bytes
                    or not finite_number(record.started_at)
                    or not finite_number(record.updated_at)
                    or not finite_number(record.finished_at)
                    or record.started_at > record.updated_at
                    or record.updated_at != record.finished_at
                    or record.finished_at > self.clock_anchor
                    or record.state is TransferState.ACTIVE
                    or not isinstance(record.state, TransferState)
                    or (
                        record.failure_reason is not None
                        and not isinstance(record.failure_reason, TransferFailureReason)
                    )
                    or (
                        record.cleanup_verified is not None
                        and type(record.cleanup_verified) is not bool
                    )
                ):
                    invalid("checkpoint contains an active or invalid transfer attempt")

                if (
                    record.failure_reason is TransferFailureReason.STALE_GENERATION
                    and record.state is not TransferState.STALE
                ):
                    invalid("stale-generation reason requires a stale transfer state")
                if record.failure_reason in (
                    TransferFailureReason.DIGEST_MISMATCH,
                    TransferFailureReason.GENERATION_CONFLICT,
                ) and (
                    record.transferred_bytes != operation.expected_size_bytes
                    or operation.expected_size_bytes > self.limits.max_size_bytes
                ):
                    invalid("digest or generation conflicts require a complete in-limit transfer")

                model_state = model_record.state
                elapsed = record.finished_at - record.started_at
                operation_key = _artifact_key(operation.identity)
                current = artifacts.get(operation_key)
                if model_state is AttemptState.STALE and (
                    current is None
                    or current.identity.generation <= operation.identity.generation
                ):
                    invalid("stale model attempt has no newer current generation")
                if operation.expected_size_bytes > self.limits.max_size_bytes and (
                    index != 1
                    or len(operation.attempt_ids) != 1
                    or record.state is not TransferState.FAILED
                    or model_state is not AttemptState.FAILED
                    or record.failure_reason is not TransferFailureReason.SIZE_LIMIT
                    or record.cleanup_verified is not None
                    or record.transferred_bytes != 0
                    or record.finished_at != record.started_at
                ):
                    invalid("over-limit declared size must be an immediate unretryable size failure")
                if record.state is TransferState.VERIFIED:
                    valid_pair = model_state in (AttemptState.VERIFIED, AttemptState.STALE)
                    if (
                        not valid_pair
                        or record.failure_reason is not None
                        or record.cleanup_verified is not None
                        or record.transferred_bytes != operation.expected_size_bytes
                        or operation.expected_size_bytes > self.limits.max_size_bytes
                        or elapsed > self.limits.max_duration_seconds
                        or (
                            model_state is AttemptState.STALE
                            and (
                                operation_key not in artifacts
                                or artifacts[operation_key].identity.generation
                                <= operation.identity.generation
                            )
                        )
                    ):
                        invalid("verified transfer state is inconsistent")
                elif record.state is TransferState.STALE:
                    if (
                        model_state is not AttemptState.STALE
                        or record.failure_reason is not TransferFailureReason.STALE_GENERATION
                        or record.cleanup_verified is not None
                        or record.transferred_bytes != operation.expected_size_bytes
                        or operation.expected_size_bytes > self.limits.max_size_bytes
                        or current is None
                        or current.identity.generation <= operation.identity.generation
                    ):
                        invalid("stale transfer state is inconsistent")
                elif record.state is TransferState.CLEANED:
                    if (
                        model_state is not AttemptState.FAILED
                        or record.failure_reason is None
                        or record.cleanup_verified is not True
                        or record.transferred_bytes == 0
                    ):
                        invalid("cleaned transfer state is inconsistent")
                elif record.state is TransferState.RECONCILIATION_REQUIRED:
                    if (
                        model_state is not AttemptState.RECONCILIATION_REQUIRED
                        or record.failure_reason is None
                        or record.cleanup_verified is True
                        or record.transferred_bytes == 0
                    ):
                        invalid("unresolved reconciliation state is inconsistent")
                elif record.state is TransferState.FAILED:
                    if (
                        model_state is not AttemptState.FAILED
                        or record.failure_reason is None
                        or record.cleanup_verified is not None
                        or record.transferred_bytes != 0
                    ):
                        invalid("failed transfer state is inconsistent")
                else:
                    invalid("checkpoint contains an unsupported transfer state")

                if elapsed > self.limits.max_duration_seconds:
                    if record.failure_reason is not TransferFailureReason.DURATION_LIMIT:
                        invalid("over-duration attempt has an inconsistent failure reason")
                elif record.failure_reason is TransferFailureReason.DURATION_LIMIT:
                    invalid("duration-limit failure did not exceed its configured bound")

                if index > 1:
                    previous = transfer_records.get(operation.attempt_ids[index - 2])
                    if previous is None:
                        invalid("retry history is missing its previous attempt")
                    if (
                        previous.state not in (TransferState.FAILED, TransferState.CLEANED)
                        or previous.failure_reason is TransferFailureReason.SIZE_LIMIT
                        or record.started_at
                        < previous.finished_at
                        + self.limits.backoff_seconds[previous.attempt_number - 1]
                    ):
                        invalid("retry history violates its prior outcome or backoff")

        if observed_operation_attempts != set(transfer_records):
            invalid("transfer attempts are not completely assigned to operations")

        previous_started_at: float | None = None
        for attempt_id in sorted(transfer_records):
            started_at = transfer_records[attempt_id].started_at
            if previous_started_at is not None and started_at < previous_started_at:
                invalid("attempt start times must be nondecreasing by attempt ID")
            previous_started_at = started_at

        for attempt in model_records.values():
            if (
                attempt.state is AttemptState.VERIFIED
                and _artifact_key(attempt.identity) not in artifacts
            ):
                invalid("verified attempt has no current artifact")

        successful_identities: dict[tuple[str, str, int], Identity] = {}
        for transfer in transfer_records.values():
            if transfer.state is not TransferState.VERIFIED:
                continue
            identity = operations[transfer.operation_id].identity
            generation_key = (*_artifact_key(identity), identity.generation)
            prior_identity = successful_identities.setdefault(generation_key, identity)
            if prior_identity != identity:
                invalid("successful transfers conflict on a verified generation digest")

        for artifact_key, artifact in artifacts.items():
            publisher_transfer = transfer_records.get(artifact.published_by)
            if publisher_transfer is None or publisher_transfer.state is not TransferState.VERIFIED:
                invalid("current artifact has no verified transfer publisher")
            for transfer in transfer_records.values():
                if transfer.state is not TransferState.VERIFIED:
                    continue
                transfer_identity = operations[transfer.operation_id].identity
                # Duplicate verification preserves the original publisher. Equal-time
                # finishes have no saved completion order, so cannot prove a forgery.
                if (
                    transfer_identity == artifact.identity
                    and transfer.finished_at < publisher_transfer.finished_at
                ):
                    invalid("current publisher has an earlier verified publication")
                if (
                    _artifact_key(transfer_identity) == artifact_key
                    and transfer_identity.generation < artifact.identity.generation
                    and transfer.finished_at > publisher_transfer.finished_at
                ):
                    invalid("older verified transfer finishes after the current publication")
            for attempt in model_records.values():
                if (
                    _artifact_key(attempt.identity) != artifact_key
                    or attempt.state is not AttemptState.VERIFIED
                ):
                    continue
                if attempt.identity.generation > artifact.identity.generation:
                    invalid("current artifact is older than a verified generation")
                if (
                    attempt.identity.generation == artifact.identity.generation
                    and attempt.identity != artifact.identity
                ):
                    invalid("current artifact conflicts with a verified generation digest")

        for stale_id, stale in transfer_records.items():
            if stale.state is not TransferState.STALE:
                continue
            stale_identity = operations[stale.operation_id].identity
            has_prior_higher_publication = any(
                witness_id != stale_id
                and witness.state is TransferState.VERIFIED
                and model_records[witness_id].state
                in (AttemptState.VERIFIED, AttemptState.STALE)
                and _artifact_key(operations[witness.operation_id].identity)
                == _artifact_key(stale_identity)
                and operations[witness.operation_id].identity.generation
                > stale_identity.generation
                and witness.finished_at <= stale.finished_at
                for witness_id, witness in transfer_records.items()
            )
            if not has_prior_higher_publication:
                invalid("stale transfer has no prior higher-generation verified publication")

        for attempt_id, conflict in transfer_records.items():
            if conflict.failure_reason is not TransferFailureReason.GENERATION_CONFLICT:
                continue
            conflict_identity = operations[conflict.operation_id].identity
            has_higher_generation_published_before_conflict = any(
                publication.state is TransferState.VERIFIED
                and model_records[publication_id].state
                in (AttemptState.VERIFIED, AttemptState.STALE)
                and _artifact_key(operations[publication.operation_id].identity)
                == _artifact_key(conflict_identity)
                and operations[publication.operation_id].identity.generation
                > conflict_identity.generation
                and publication.finished_at < conflict.finished_at
                for publication_id, publication in transfer_records.items()
            )
            has_prior_verified_witness = any(
                witness_id != attempt_id
                and witness.state is TransferState.VERIFIED
                and model_records[witness_id].state
                in (AttemptState.VERIFIED, AttemptState.STALE)
                and _artifact_key(operations[witness.operation_id].identity)
                == _artifact_key(conflict_identity)
                and operations[witness.operation_id].identity.generation
                == conflict_identity.generation
                and operations[witness.operation_id].identity != conflict_identity
                and witness.finished_at <= conflict.finished_at
                for witness_id, witness in transfer_records.items()
            )
            if not has_prior_verified_witness:
                invalid("generation conflict has no prior verified identity witness")
            if has_higher_generation_published_before_conflict:
                invalid("generation conflict follows a newer verified publication")

        concurrency_events: list[tuple[float, int]] = []
        for record in transfer_records.values():
            operation = operations[record.operation_id]
            if (
                operation.expected_size_bytes > self.limits.max_size_bytes
                or record.finished_at == record.started_at
            ):
                continue
            concurrency_events.append((record.started_at, 1))
            concurrency_events.append((record.finished_at, -1))
        active_intervals = 0
        for _, change in sorted(concurrency_events):
            # A finish (-1) sorts before a start (+1) at the same timestamp.
            active_intervals += change
            if active_intervals < 0 or active_intervals > self.limits.max_concurrent_transfers:
                invalid("historical admitted transfers exceed the concurrency cap")
        if active_intervals != 0:
            invalid("historical transfer intervals are inconsistent")
