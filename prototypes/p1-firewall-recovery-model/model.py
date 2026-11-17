"""In-memory model for synthetic firewall-change recovery.

This module models symbolic rules and supplied probe results only. It does not
invoke a firewall, SSH client, socket, shell, subprocess, or service manager.
Its transitions are not a production firewall or installer policy.
"""

from dataclasses import dataclass
from enum import Enum
from typing import Any


class TransactionStatus(str, Enum):
    PENDING = "pending_verification"
    COMMITTED = "committed"
    ROLLED_BACK = "rolled_back"


@dataclass(frozen=True)
class ProbeResult:
    """Results from a test-provided fake probe; no connection is made here."""

    established_session_survives: bool
    fresh_connection_succeeds: bool

    def __post_init__(self) -> None:
        if type(self.established_session_survives) is not bool:
            raise ValueError("established-session result must be boolean")
        if type(self.fresh_connection_succeeds) is not bool:
            raise ValueError("fresh-connection result must be boolean")


@dataclass(frozen=True)
class JournalEntry:
    transaction_id: str
    previous_rules: tuple[str, ...]
    candidate_rules: tuple[str, ...]
    status: TransactionStatus
    verification_complete: bool = False


class RecoveryModel:
    """Single-transaction recovery state machine over symbolic rule names."""

    SCHEMA = 2

    def __init__(self, initial_rules: tuple[str, ...] | list[str]) -> None:
        self._active_rules = self._validate_rules(initial_rules, "initial rules")
        self._journal: JournalEntry | None = None

    @staticmethod
    def _validate_rules(value: Any, field: str) -> tuple[str, ...]:
        if not isinstance(value, (list, tuple)):
            raise ValueError(f"{field} must be a list of symbolic rules")
        if any(not isinstance(rule, str) or not rule for rule in value):
            raise ValueError(f"{field} contains an invalid symbolic rule")
        if len(set(value)) != len(value):
            raise ValueError(f"{field} contains duplicate symbolic rules")
        return tuple(value)

    @property
    def active_rules(self) -> tuple[str, ...]:
        return self._active_rules

    @property
    def journal(self) -> JournalEntry | None:
        return self._journal

    def apply_candidate(
        self, transaction_id: str, candidate_rules: tuple[str, ...] | list[str]
    ) -> None:
        """Apply a candidate to model state and retain its exact prior snapshot."""

        if not isinstance(transaction_id, str) or not transaction_id:
            raise ValueError("transaction ID must be a non-empty string")
        if self._journal is not None and self._journal.status is TransactionStatus.PENDING:
            raise ValueError("a transaction is already awaiting verification")

        candidate = self._validate_rules(candidate_rules, "candidate rules")
        self._journal = JournalEntry(
            transaction_id=transaction_id,
            previous_rules=self._active_rules,
            candidate_rules=candidate,
            status=TransactionStatus.PENDING,
        )
        self._active_rules = candidate

    def verify(
        self,
        transaction_id: str,
        probe: ProbeResult,
        *,
        operator_confirmed: bool,
    ) -> TransactionStatus:
        """Commit only on explicit confirmation and both synthetic probe checks."""

        entry = self._require_pending(transaction_id)
        if not isinstance(probe, ProbeResult):
            raise ValueError("verification requires a synthetic ProbeResult")
        if type(operator_confirmed) is not bool:
            raise ValueError("operator confirmation must be boolean")

        if (
            operator_confirmed
            and probe.established_session_survives
            and probe.fresh_connection_succeeds
        ):
            self._journal = JournalEntry(
                entry.transaction_id,
                entry.previous_rules,
                entry.candidate_rules,
                TransactionStatus.COMMITTED,
                verification_complete=True,
            )
            return TransactionStatus.COMMITTED

        self._active_rules = entry.previous_rules
        self._journal = JournalEntry(
            entry.transaction_id,
            entry.previous_rules,
            entry.candidate_rules,
            TransactionStatus.ROLLED_BACK,
        )
        return TransactionStatus.ROLLED_BACK

    def recover_pending(self) -> TransactionStatus:
        """Restore the in-memory snapshot for an interrupted pending transaction.

        This is a conservative rule for this synthetic model, not a production
        recovery policy. Replaying it after rollback leaves state unchanged.
        """

        if self._journal is None:
            raise ValueError("there is no transaction to recover")
        entry = self._journal
        if entry.status is TransactionStatus.PENDING:
            self._active_rules = entry.previous_rules
            self._journal = JournalEntry(
                entry.transaction_id,
                entry.previous_rules,
                entry.candidate_rules,
                TransactionStatus.ROLLED_BACK,
            )
        return self._journal.status

    def to_state(self) -> dict[str, Any]:
        """Return detached, JSON-compatible model state for restart simulation."""

        entry = self._journal
        journal: dict[str, Any] | None = None
        if entry is not None:
            journal = {
                "transaction_id": entry.transaction_id,
                "previous_rules": list(entry.previous_rules),
                "candidate_rules": list(entry.candidate_rules),
                "status": entry.status.value,
                "verification_complete": entry.verification_complete,
            }
        return {
            "schema": self.SCHEMA,
            "active_rules": list(self._active_rules),
            "journal": journal,
        }

    @classmethod
    def from_state(cls, state: Any) -> "RecoveryModel":
        """Restore model state after validating its synthetic journal invariants."""

        if not isinstance(state, dict) or set(state) != {"schema", "active_rules", "journal"}:
            raise ValueError("invalid model state")
        if type(state["schema"]) is not int or state["schema"] != cls.SCHEMA:
            raise ValueError("unsupported model-state schema")

        model = cls(cls._validate_rules(state["active_rules"], "active rules"))
        raw_entry = state["journal"]
        if raw_entry is None:
            return model
        if not isinstance(raw_entry, dict) or set(raw_entry) != {
            "transaction_id",
            "previous_rules",
            "candidate_rules",
            "status",
            "verification_complete",
        }:
            raise ValueError("invalid model journal")
        transaction_id = raw_entry["transaction_id"]
        if not isinstance(transaction_id, str) or not transaction_id:
            raise ValueError("invalid journal transaction ID")
        try:
            status = TransactionStatus(raw_entry["status"])
        except (TypeError, ValueError) as exc:
            raise ValueError("invalid journal status") from exc

        # Internal consistency only: this marker does not authenticate saved state.
        verification_complete = raw_entry["verification_complete"]
        if type(verification_complete) is not bool or verification_complete != (
            status is TransactionStatus.COMMITTED
        ):
            raise ValueError("verification marker does not match journal status")

        previous = cls._validate_rules(raw_entry["previous_rules"], "journal snapshot")
        candidate = cls._validate_rules(raw_entry["candidate_rules"], "journal candidate")
        expected_active = {
            TransactionStatus.PENDING: candidate,
            TransactionStatus.COMMITTED: candidate,
            TransactionStatus.ROLLED_BACK: previous,
        }[status]
        if model.active_rules != expected_active:
            raise ValueError("active rules do not match journal status")

        model._journal = JournalEntry(
            transaction_id, previous, candidate, status, verification_complete
        )
        return model

    def _require_pending(self, transaction_id: str) -> JournalEntry:
        entry = self._journal
        if (
            entry is None
            or entry.transaction_id != transaction_id
            or entry.status is not TransactionStatus.PENDING
        ):
            raise ValueError("transaction is not pending verification")
        return entry
