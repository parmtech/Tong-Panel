package recovery

import (
	"errors"
	"testing"
)

func readyForSwitch(t *testing.T) *Model {
	t.Helper()
	m := New("release-old", "release-new", true)
	if err := m.RecordArtifact("release-new"); err != nil {
		t.Fatal(err)
	}
	if err := m.StartCandidate(); err != nil {
		t.Fatal(err)
	}
	if err := m.MarkHealthy(); err != nil {
		t.Fatal(err)
	}
	if err := m.RequestSwitch(); err != nil {
		t.Fatal(err)
	}
	return &m
}

func TestCannotRecordUnverifiedOrSubstitutedArtifact(t *testing.T) {
	m := New("release-old", "release-new", false)
	if err := m.RecordArtifact("release-new"); !errors.Is(err, ErrArtifactNotVerified) {
		t.Fatalf("unverified artifact error = %v", err)
	}
	m = New("release-old", "release-new", true)
	if err := m.RecordArtifact("different-digest"); !errors.Is(err, ErrArtifactNotVerified) {
		t.Fatalf("substituted artifact error = %v", err)
	}
}

func TestSwitchEffectIsNotSuccessUntilCandidateRouteObserved(t *testing.T) {
	m := readyForSwitch(t)
	if err := m.ApplySwitch(); err != nil {
		t.Fatal(err)
	}
	if m.State != StateSwitching || m.Outcome != "" {
		t.Fatalf("effect alone claimed success: state=%q outcome=%q", m.State, m.Outcome)
	}
	m.Recover(m.ActualRoute)
	if m.State != StateActive || m.Outcome != "promoted" {
		t.Fatalf("observed promotion = state %q outcome %q", m.State, m.Outcome)
	}
}

func TestOldRouteAfterInterruptedSwitchIsNotPromotion(t *testing.T) {
	m := readyForSwitch(t)
	m.Recover(m.PriorDigest)
	if m.State != StateFailed || m.Outcome != "not_promoted" {
		t.Fatalf("old route recovery = state %q outcome %q", m.State, m.Outcome)
	}
}

func TestUnknownRouteRemainsUncertain(t *testing.T) {
	m := readyForSwitch(t)
	m.Recover("unrecognized-release")
	if m.State != StateUncertain || m.Outcome != "uncertain" {
		t.Fatalf("unknown route recovery = state %q outcome %q", m.State, m.Outcome)
	}
}

func TestRollbackIsSuccessOnlyAfterPriorRouteObserved(t *testing.T) {
	m := readyForSwitch(t)
	if err := m.ApplySwitch(); err != nil {
		t.Fatal(err)
	}
	m.Recover(m.ActualRoute)
	if err := m.RequestRollback(true, true); err != nil {
		t.Fatal(err)
	}
	if err := m.ApplyRollback(true, true); err != nil {
		t.Fatal(err)
	}
	if m.State != StateRollingBack || m.Outcome != "" {
		t.Fatalf("rollback effect claimed completion: state=%q outcome=%q", m.State, m.Outcome)
	}
	m.Recover(m.ActualRoute)
	if m.State != StateRolledBack || m.Outcome != "rolled_back" {
		t.Fatalf("observed rollback = state %q outcome %q", m.State, m.Outcome)
	}
}

func TestUnavailablePriorReleaseCannotBeSelectedForRollback(t *testing.T) {
	m := readyForSwitch(t)
	if err := m.ApplySwitch(); err != nil {
		t.Fatal(err)
	}
	m.Recover(m.ActualRoute)
	if err := m.RequestRollback(false, true); !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("unavailable prior release rollback error = %v", err)
	}
}
