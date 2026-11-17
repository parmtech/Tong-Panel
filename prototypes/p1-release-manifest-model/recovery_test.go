package releasemodel

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"
)

const priorCommit = "fedcba9876543210fedcba9876543210fedcba98"

func recoveryFixture(t *testing.T) (fixture, RecoveryJournal, RecoveryObservation) {
	t.Helper()
	f := newFixture(t)
	hash := sha256.Sum256(f.signed.Manifest)
	j := RecoveryJournal{AttemptID: strings.Repeat("a", 32), PreviousCommit: priorCommit,
		CandidateCommit: f.expected.SourceCommit, CandidateVersion: f.expected.Release,
		ManifestSHA256: hex.EncodeToString(hash[:]), Phase: PhaseSwitchRequested}
	obs := RecoveryObservation{ActiveCommit: j.CandidateCommit, Health: HealthHealthy,
		EffectsFenced: true, PreviousRetained: true}
	return f, j, obs
}

func testPlan(t *testing.T, f fixture, j RecoveryJournal, obs RecoveryObservation, want RecoveryAction) {
	t.Helper()
	plan, err := PlanUpgradeRecovery(f.expected, f.signed, f.payloads, j, obs)
	if err != nil || plan.Action != want || plan.Reason == "" {
		t.Fatalf("unexpected recovery plan %v, error %v, want %s", plan, err, want)
	}
	again, err := PlanUpgradeRecovery(f.expected, f.signed, f.payloads, j, obs)
	if err != nil || !reflect.DeepEqual(again, plan) {
		t.Fatalf("repeated recovery changed the advisory plan: %v %v", again, err)
	}
}

func TestInterruptedUpgradeRequiresFreshProofAndObservedHealth(t *testing.T) {
	f, journal, obs := recoveryFixture(t)
	for _, tc := range []struct {
		name   string
		phase  RecoveryPhase
		active string
		health RecoveryHealth
		fenced bool
		prior  bool
		want   RecoveryAction
	}{
		{"before switch", PhasePrepared, priorCommit, HealthUnknown, true, true, ActionKeepPrevious},
		{"before switch effects not fenced", PhasePrepared, priorCommit, HealthUnknown, false, true, ActionManualReview},
		{"switch requested but prior still active", PhaseSwitchRequested, priorCommit, HealthUnknown, true, true, ActionKeepPrevious},
		{"switched and healthy", PhaseSwitchRequested, journal.CandidateCommit, HealthHealthy, true, true, ActionCommitCandidate},
		{"switched but status unknown", PhaseSwitchRequested, journal.CandidateCommit, HealthUnknown, true, true, ActionManualReview},
		{"switched but prior missing", PhaseSwitchRequested, journal.CandidateCommit, HealthHealthy, true, false, ActionManualReview},
		{"switched and unhealthy", PhaseSwitchRequested, journal.CandidateCommit, HealthUnhealthy, true, true, ActionRequestRollback},
		{"unhealthy with prior missing", PhaseSwitchRequested, journal.CandidateCommit, HealthUnhealthy, true, false, ActionManualReview},
		{"unfenced candidate", PhaseSwitchRequested, journal.CandidateCommit, HealthHealthy, false, true, ActionManualReview},
		{"rollback requested, candidate still active", PhaseRollbackRequested, journal.CandidateCommit, HealthUnknown, true, true, ActionRequestRollback},
		{"rollback pending with prior missing", PhaseRollbackRequested, journal.CandidateCommit, HealthUnhealthy, true, false, ActionManualReview},
		{"rollback requested, prior observed", PhaseRollbackRequested, priorCommit, HealthUnknown, true, true, ActionConfirmRollback},
		{"rollback recorded and prior observed", PhaseRolledBack, priorCommit, HealthUnknown, true, false, ActionConfirmRollback},
		{"rollback recorded but candidate observed", PhaseRolledBack, journal.CandidateCommit, HealthHealthy, true, true, ActionManualReview},
		{"committed and healthy", PhaseCommitted, journal.CandidateCommit, HealthHealthy, true, false, ActionConsistent},
		{"committed but unhealthy", PhaseCommitted, journal.CandidateCommit, HealthUnhealthy, true, true, ActionManualReview},
		{"committed but active prior", PhaseCommitted, priorCommit, HealthHealthy, true, true, ActionManualReview},
		{"unobserved external switch", PhaseSwitchRequested, strings.Repeat("b", 40), HealthUnknown, true, true, ActionManualReview},
	} {
		t.Run(tc.name, func(t *testing.T) {
			journal.Phase = tc.phase
			obs.ActiveCommit, obs.Health, obs.EffectsFenced, obs.PreviousRetained = tc.active, tc.health, tc.fenced, tc.prior
			testPlan(t, f, journal, obs, tc.want)
		})
	}
}

func TestRecoveryAfterJSONRestartRetainsUncertainty(t *testing.T) {
	f, journal, obs := recoveryFixture(t)
	for _, phase := range []RecoveryPhase{PhasePrepared, PhaseSwitchRequested, PhaseRollbackRequested, PhaseCommitted, PhaseRolledBack} {
		t.Run(string(phase), func(t *testing.T) {
			journal.Phase = phase
			raw, err := json.Marshal(journal)
			if err != nil {
				t.Fatal(err)
			}
			copyAfterRestart, err := ReadRecoveryJournal(raw)
			if err != nil || !reflect.DeepEqual(copyAfterRestart, journal) {
				t.Fatalf("journal round-trip: %v", err)
			}
			// Simulate a crash where a side effect is still unconfirmed.
			obs.EffectsFenced = false
			testPlan(t, f, copyAfterRestart, obs, ActionManualReview)
		})
	}
}

func TestRecoveryRechecksCurrentCandidateAndOutOfBandKey(t *testing.T) {
	for _, mode := range []string{"tampered manifest", "tampered payload", "wrong trusted public key", "changed source pin", "changed journal digest", "changed journal candidate", "changed journal version"} {
		t.Run(mode, func(t *testing.T) {
			f, j, obs := recoveryFixture(t)
			switch mode {
			case "tampered manifest":
				f.signed.Manifest[2] ^= 1
			case "tampered payload":
				f.payloads["tong-fixture.amd64.deb"][0] ^= 1
			case "wrong trusted public key":
				k, _, _ := ed25519.GenerateKey(rand.Reader)
				f.expected.PublicKey = k
			case "changed source pin":
				f.expected.SourceCommit = strings.Repeat("c", 40)
			case "changed journal digest":
				j.ManifestSHA256 = strings.Repeat("a", 64)
			case "changed journal candidate":
				j.CandidateCommit = strings.Repeat("b", 40)
			case "changed journal version":
				j.CandidateVersion = "9.1.0"
			}
			plan, err := PlanUpgradeRecovery(f.expected, f.signed, f.payloads, j, obs)
			if !errors.Is(err, ErrRejected) || plan.Action != ActionManualReview {
				t.Fatalf("untrusted candidate gave plan %v err %v", plan, err)
			}
		})
	}
}

func TestRecoveryRejectsJournalAndObservationTamper(t *testing.T) {
	for _, mode := range []string{"empty attempt", "bad attempt", "same previous and candidate", "invalid previous", "unrecognized phase", "invalid digest", "unknown health", "invalid observation ID"} {
		t.Run(mode, func(t *testing.T) {
			f, j, obs := recoveryFixture(t)
			switch mode {
			case "empty attempt":
				j.AttemptID = ""
			case "bad attempt":
				j.AttemptID = strings.Repeat("g", 32)
			case "same previous and candidate":
				j.PreviousCommit = j.CandidateCommit
			case "invalid previous":
				j.PreviousCommit = "unknown"
			case "unrecognized phase":
				j.Phase = "skip-review"
			case "invalid digest":
				j.ManifestSHA256 = "not-a-digest"
			case "unknown health":
				obs.Health = "greenish"
			case "invalid observation ID":
				obs.ActiveCommit = "unknown"
			}
			plan, err := PlanUpgradeRecovery(f.expected, f.signed, f.payloads, j, obs)
			if !errors.Is(err, ErrRejected) || plan.Action != ActionManualReview {
				t.Fatalf("invalid journal/observation was accepted: %v %v", plan, err)
			}
		})
	}
}

func TestRecoveryJournalRejectsAmbiguousAndMalformedJSON(t *testing.T) {
	_, j, _ := recoveryFixture(t)
	raw, err := json.Marshal(j)
	if err != nil {
		t.Fatal(err)
	}
	for _, candidate := range [][]byte{
		append([]byte(" "), raw...),
		append(append([]byte(nil), raw...), '\n'),
		append(append([]byte(nil), raw[:len(raw)-1]...), []byte(`,"unknown":true}`)...),
		[]byte(`{"attempt_id":"first","attempt_id":"second"}`),
		[]byte(`{}`),
		bytesRepeat('a', maxJournalBytes+1),
	} {
		if _, err := ReadRecoveryJournal(candidate); !errors.Is(err, ErrRejected) {
			t.Fatalf("invalid journal accepted: %q", candidate[:min(30, len(candidate))])
		}
	}
}

func bytesRepeat(char byte, length int) []byte {
	b := make([]byte, length)
	for i := range b {
		b[i] = char
	}
	return b
}
