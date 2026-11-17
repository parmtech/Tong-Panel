import unittest

from model import ProbeResult, RecoveryModel, TransactionStatus


ADMIN_PATH = "symbolic:operator-control-path"
SITE_PATH = "symbolic:site-service-path"
BLOCKED_ADMIN_PATH = "symbolic:blocked-operator-path"


class FakeProbe:
    """A deterministic fixture; it never opens an SSH or network connection."""

    def __init__(self, established_survives: bool, fresh_path_available: bool) -> None:
        self.established_survives = established_survives
        self.fresh_path_available = fresh_path_available
        self.observed_rules: tuple[str, ...] | None = None
        self.calls = 0

    def observe(self, symbolic_rules: tuple[str, ...]) -> ProbeResult:
        self.calls += 1
        self.observed_rules = symbolic_rules
        fresh_connection_succeeds = (
            self.fresh_path_available and ADMIN_PATH in symbolic_rules
        )
        return ProbeResult(
            established_session_survives=self.established_survives,
            fresh_connection_succeeds=fresh_connection_succeeds,
        )


class FirewallRecoveryModelTests(unittest.TestCase):
    def setUp(self) -> None:
        self.initial = (ADMIN_PATH, SITE_PATH)

    def test_established_session_does_not_mask_synthetic_lockout(self) -> None:
        model = RecoveryModel(self.initial)
        candidate = (SITE_PATH, BLOCKED_ADMIN_PATH)
        model.apply_candidate("change-1", candidate)
        probe = FakeProbe(established_survives=True, fresh_path_available=True)

        status = model.verify(
            "change-1", probe.observe(model.active_rules), operator_confirmed=True
        )

        self.assertEqual(probe.calls, 1)
        self.assertEqual(probe.observed_rules, candidate)
        self.assertFalse(ADMIN_PATH in probe.observed_rules)
        self.assertEqual(status, TransactionStatus.ROLLED_BACK)
        self.assertEqual(model.active_rules, self.initial)
        self.assertEqual(model.journal.previous_rules, self.initial)
        self.assertEqual(model.journal.status, TransactionStatus.ROLLED_BACK)

    def test_commit_requires_both_session_checks_and_confirmation(self) -> None:
        model = RecoveryModel(self.initial)
        candidate = (ADMIN_PATH, SITE_PATH, "symbolic:extra-service-path")
        model.apply_candidate("change-2", candidate)
        probe = FakeProbe(established_survives=True, fresh_path_available=True)

        status = model.verify(
            "change-2", probe.observe(model.active_rules), operator_confirmed=True
        )

        self.assertEqual(status, TransactionStatus.COMMITTED)
        self.assertEqual(model.active_rules, candidate)
        self.assertEqual(model.journal.status, TransactionStatus.COMMITTED)

        committed_state = model.to_state()
        restarted = RecoveryModel.from_state(committed_state)
        self.assertEqual(restarted.recover_pending(), TransactionStatus.COMMITTED)
        self.assertEqual(restarted.active_rules, candidate)
        self.assertEqual(restarted.to_state(), committed_state)

    def test_lost_established_session_rolls_back_even_if_fresh_path_works(self) -> None:
        model = RecoveryModel(self.initial)
        candidate = (ADMIN_PATH, SITE_PATH, "symbolic:extra-service-path")
        model.apply_candidate("change-3", candidate)
        probe = FakeProbe(established_survives=False, fresh_path_available=True)

        status = model.verify(
            "change-3", probe.observe(model.active_rules), operator_confirmed=True
        )

        self.assertEqual(status, TransactionStatus.ROLLED_BACK)
        self.assertEqual(model.active_rules, self.initial)

    def test_missing_operator_confirmation_rolls_back(self) -> None:
        model = RecoveryModel(self.initial)
        candidate = (ADMIN_PATH, SITE_PATH, "symbolic:extra-service-path")
        model.apply_candidate("change-4", candidate)
        probe = FakeProbe(established_survives=True, fresh_path_available=True)

        status = model.verify(
            "change-4", probe.observe(model.active_rules), operator_confirmed=False
        )

        self.assertEqual(status, TransactionStatus.ROLLED_BACK)
        self.assertEqual(model.active_rules, self.initial)

    def test_restart_with_pending_transaction_restores_exact_snapshot(self) -> None:
        model = RecoveryModel(self.initial)
        model.apply_candidate("change-5", (SITE_PATH, BLOCKED_ADMIN_PATH))

        restarted = RecoveryModel.from_state(model.to_state())
        status = restarted.recover_pending()

        self.assertEqual(status, TransactionStatus.ROLLED_BACK)
        self.assertEqual(restarted.active_rules, self.initial)
        self.assertEqual(restarted.journal.previous_rules, self.initial)

    def test_replaying_recovery_is_idempotent(self) -> None:
        model = RecoveryModel(self.initial)
        model.apply_candidate("change-6", (SITE_PATH, BLOCKED_ADMIN_PATH))
        restarted = RecoveryModel.from_state(model.to_state())

        first = restarted.recover_pending()
        first_state = restarted.to_state()
        second = restarted.recover_pending()

        self.assertEqual(first, TransactionStatus.ROLLED_BACK)
        self.assertEqual(second, TransactionStatus.ROLLED_BACK)
        self.assertEqual(restarted.to_state(), first_state)
        self.assertEqual(RecoveryModel.from_state(first_state).to_state(), first_state)

    def test_inconsistent_restart_state_is_rejected(self) -> None:
        model = RecoveryModel(self.initial)
        model.apply_candidate("change-7", (SITE_PATH, BLOCKED_ADMIN_PATH))
        corrupted = model.to_state()
        corrupted["active_rules"] = list(self.initial)

        with self.assertRaisesRegex(ValueError, "do not match journal status"):
            RecoveryModel.from_state(corrupted)

    def test_second_change_cannot_replace_pending_journal(self) -> None:
        model = RecoveryModel(self.initial)
        model.apply_candidate("change-8", (SITE_PATH, BLOCKED_ADMIN_PATH))
        pending_state = model.to_state()

        with self.assertRaisesRegex(ValueError, "already awaiting verification"):
            model.apply_candidate("change-9", (ADMIN_PATH,))

        self.assertEqual(model.to_state(), pending_state)

    def test_status_only_change_cannot_commit_pending_snapshot(self) -> None:
        model = RecoveryModel(self.initial)
        model.apply_candidate("unverified-change", (SITE_PATH, BLOCKED_ADMIN_PATH))
        corrupted = model.to_state()
        corrupted["journal"]["status"] = TransactionStatus.COMMITTED.value

        with self.assertRaisesRegex(ValueError, "verification marker"):
            RecoveryModel.from_state(corrupted)

    def test_verification_marker_must_match_status_and_be_boolean(self) -> None:
        model = RecoveryModel(self.initial)
        model.apply_candidate("marker-change", (SITE_PATH, BLOCKED_ADMIN_PATH))
        for status, marker in (
            (TransactionStatus.PENDING, True),
            (TransactionStatus.ROLLED_BACK, True),
            (TransactionStatus.COMMITTED, False),
            (TransactionStatus.PENDING, 0),
            (TransactionStatus.COMMITTED, 1),
        ):
            with self.subTest(status=status, marker=marker):
                corrupted = model.to_state()
                corrupted["journal"]["status"] = status.value
                corrupted["journal"]["verification_complete"] = marker
                with self.assertRaisesRegex(ValueError, "verification marker"):
                    RecoveryModel.from_state(corrupted)

    def test_old_schema_without_verification_marker_is_rejected(self) -> None:
        model = RecoveryModel(self.initial)
        model.apply_candidate("old-change", (SITE_PATH, BLOCKED_ADMIN_PATH))
        old_state = model.to_state()
        old_state["schema"] = 1
        del old_state["journal"]["verification_complete"]

        with self.assertRaisesRegex(ValueError, "unsupported model-state schema"):
            RecoveryModel.from_state(old_state)


if __name__ == "__main__":
    unittest.main()
