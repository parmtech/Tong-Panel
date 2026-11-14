package recovery

import (
	"encoding/json"
	"errors"
	"sync"
	"testing"
)

func restartModel(t *testing.T, before *Model) Model {
	t.Helper()
	serialized, err := json.Marshal(before)
	if err != nil {
		t.Fatalf("serialize model snapshot: %v", err)
	}
	var after Model
	if err := json.Unmarshal(serialized, &after); err != nil {
		t.Fatalf("restore model snapshot: %v", err)
	}
	return after
}

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

func TestRecoveryAfterSerializedRestartBoundaries(t *testing.T) {
	tests := []struct {
		name          string
		prepare       func(t *testing.T) *Model
		observedRoute string
		wantState     State
		wantOutcome   string
	}{
		{
			name: "switch requested but route unchanged",
			prepare: func(t *testing.T) *Model {
				return readyForSwitch(t)
			},
			observedRoute: "release-old",
			wantState:     StateFailed,
			wantOutcome:   "not_promoted",
		},
		{
			name: "route changed before observation",
			prepare: func(t *testing.T) *Model {
				m := readyForSwitch(t)
				if err := m.ApplySwitch(); err != nil {
					t.Fatal(err)
				}
				return m
			},
			observedRoute: "release-new",
			wantState:     StateActive,
			wantOutcome:   "promoted",
		},
		{
			name: "rollback requested but route unchanged",
			prepare: func(t *testing.T) *Model {
				m := readyForSwitch(t)
				if err := m.ApplySwitch(); err != nil {
					t.Fatal(err)
				}
				m.Recover("release-new")
				if err := m.RequestRollback(true, true); err != nil {
					t.Fatal(err)
				}
				return m
			},
			observedRoute: "release-new",
			wantState:     StateActive,
			wantOutcome:   "rollback_not_observed",
		},
		{
			name: "rollback route changed before observation",
			prepare: func(t *testing.T) *Model {
				m := readyForSwitch(t)
				if err := m.ApplySwitch(); err != nil {
					t.Fatal(err)
				}
				m.Recover("release-new")
				if err := m.RequestRollback(true, true); err != nil {
					t.Fatal(err)
				}
				if err := m.ApplyRollback(true, true); err != nil {
					t.Fatal(err)
				}
				return m
			},
			observedRoute: "release-old",
			wantState:     StateRolledBack,
			wantOutcome:   "rolled_back",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			before := tt.prepare(t)
			restarted := restartModel(t, before)
			restarted.Recover(tt.observedRoute)
			if restarted.State != tt.wantState || restarted.Outcome != tt.wantOutcome {
				t.Fatalf("recovery after restart = state %q outcome %q, want state %q outcome %q", restarted.State, restarted.Outcome, tt.wantState, tt.wantOutcome)
			}
		})
	}
}

func TestRepeatedPromotionRecoveryIsIdempotent(t *testing.T) {
	m := readyForSwitch(t)
	if err := m.ApplySwitch(); err != nil {
		t.Fatal(err)
	}
	m.Recover(m.CandidateDigest)
	m.Recover(m.CandidateDigest)
	if m.State != StateActive || m.Outcome != "promoted" {
		t.Fatalf("repeated promotion recovery = state %q outcome %q", m.State, m.Outcome)
	}
}

func TestCandidateRouteWithoutCompleteProofRemainsUncertainAfterSnapshotRoundTrip(t *testing.T) {
	proofLoss := []struct {
		name   string
		mutate func(*Model)
	}{
		{name: "candidate verification", mutate: func(m *Model) { m.CandidateVerified = false }},
		{name: "artifact record", mutate: func(m *Model) { m.ArtifactRecorded = false }},
		{name: "candidate started", mutate: func(m *Model) { m.CandidateStarted = false }},
		{name: "health check", mutate: func(m *Model) { m.HealthPassed = false }},
	}

	for _, tc := range proofLoss {
		t.Run(tc.name, func(t *testing.T) {
			before := readyForSwitch(t)
			if err := before.ApplySwitch(); err != nil {
				t.Fatal(err)
			}
			tc.mutate(before)

			restarted := restartModel(t, before)
			restarted.Recover(restarted.CandidateDigest)
			if restarted.State != StateUncertain || restarted.Outcome != "uncertain" {
				t.Fatalf("recovery with missing proof = state %q outcome %q, want uncertainty", restarted.State, restarted.Outcome)
			}
		})
	}
}

func TestActiveRecoveryWithoutCompleteProofBecomesUncertain(t *testing.T) {
	proofLoss := []struct {
		name   string
		mutate func(*Model)
	}{
		{name: "candidate verification", mutate: func(m *Model) { m.CandidateVerified = false }},
		{name: "artifact record", mutate: func(m *Model) { m.ArtifactRecorded = false }},
		{name: "candidate started", mutate: func(m *Model) { m.CandidateStarted = false }},
		{name: "health check", mutate: func(m *Model) { m.HealthPassed = false }},
	}

	for _, tc := range proofLoss {
		t.Run(tc.name, func(t *testing.T) {
			m := readyForSwitch(t)
			if err := m.ApplySwitch(); err != nil {
				t.Fatal(err)
			}
			m.Recover(m.CandidateDigest)
			routeBefore := m.ActualRoute
			tc.mutate(m)
			m.Recover(m.CandidateDigest)

			if m.State != StateUncertain || m.Outcome != "uncertain" || m.ActualRoute != routeBefore || m.ObservedRoute != m.CandidateDigest {
				t.Fatalf("active recovery with lost proof = state %q outcome %q actual=%q observed=%q", m.State, m.Outcome, m.ActualRoute, m.ObservedRoute)
			}
		})
	}
}

func TestApplySwitchRechecksPromotionProof(t *testing.T) {
	proofLoss := []struct {
		name   string
		mutate func(*Model)
	}{
		{name: "candidate verification", mutate: func(m *Model) { m.CandidateVerified = false }},
		{name: "artifact record", mutate: func(m *Model) { m.ArtifactRecorded = false }},
		{name: "candidate started", mutate: func(m *Model) { m.CandidateStarted = false }},
		{name: "health check", mutate: func(m *Model) { m.HealthPassed = false }},
	}

	for _, tc := range proofLoss {
		t.Run(tc.name, func(t *testing.T) {
			m := readyForSwitch(t)
			tc.mutate(m)
			routeBefore := m.ActualRoute
			if err := m.ApplySwitch(); !errors.Is(err, ErrInvalidTransition) {
				t.Fatalf("promotion effect without complete proof = %v, want invalid transition", err)
			}
			if m.State != StateSwitching || m.ActualRoute != routeBefore || m.Outcome != "" {
				t.Fatalf("rejected promotion effect changed model: state=%q route=%q outcome=%q", m.State, m.ActualRoute, m.Outcome)
			}
		})
	}
}

func TestContradictoryObservationAfterPromotionBecomesUncertain(t *testing.T) {
	m := readyForSwitch(t)
	if err := m.ApplySwitch(); err != nil {
		t.Fatal(err)
	}
	m.Recover(m.CandidateDigest)
	m.Recover(m.PriorDigest)
	if m.State != StateUncertain || m.Outcome != "uncertain" {
		t.Fatalf("contradictory promotion observation = state %q outcome %q", m.State, m.Outcome)
	}
}

func TestOldRouteAfterInterruptedSwitchIsNotPromotion(t *testing.T) {
	m := readyForSwitch(t)
	m.Recover(m.PriorDigest)
	if m.State != StateFailed || m.Outcome != "not_promoted" {
		t.Fatalf("old route recovery = state %q outcome %q", m.State, m.Outcome)
	}
}

func TestRepeatedNotPromotedRecoveryIsIdempotent(t *testing.T) {
	m := readyForSwitch(t)
	m.Recover(m.PriorDigest)
	m.Recover(m.PriorDigest)
	if m.State != StateFailed || m.Outcome != "not_promoted" {
		t.Fatalf("repeated not-promoted recovery = state %q outcome %q", m.State, m.Outcome)
	}
}

func TestRecoveryFromNonRecoveryStateBecomesUncertain(t *testing.T) {
	m := readyForSwitch(t)
	m.State = StateReady
	routeBefore := m.ActualRoute
	m.Recover(m.CandidateDigest)
	if m.State != StateUncertain || m.Outcome != "uncertain" || m.ActualRoute != routeBefore || m.ObservedRoute != m.CandidateDigest {
		t.Fatalf("recovery from ready state = state %q outcome %q actual=%q observed=%q", m.State, m.Outcome, m.ActualRoute, m.ObservedRoute)
	}
}

func TestContradictoryObservationAfterFailedPromotionBecomesUncertain(t *testing.T) {
	m := readyForSwitch(t)
	m.Recover(m.PriorDigest)
	if m.State != StateFailed || m.Outcome != "not_promoted" {
		t.Fatalf("initial failed promotion = state %q outcome %q", m.State, m.Outcome)
	}
	routeBefore := m.ActualRoute
	m.Recover(m.CandidateDigest)
	if m.State != StateUncertain || m.Outcome != "uncertain" || m.ActualRoute != routeBefore || m.ObservedRoute != m.CandidateDigest {
		t.Fatalf("contradictory failed-promotion observation = state %q outcome %q actual=%q observed=%q", m.State, m.Outcome, m.ActualRoute, m.ObservedRoute)
	}
}

func TestUnknownRouteDuringRollbackBecomesUncertain(t *testing.T) {
	m := readyForSwitch(t)
	if err := m.ApplySwitch(); err != nil {
		t.Fatal(err)
	}
	m.Recover(m.CandidateDigest)
	if err := m.RequestRollback(true, true); err != nil {
		t.Fatal(err)
	}
	routeBefore := m.ActualRoute
	m.Recover("unrecognized-release")
	if m.State != StateUncertain || m.Outcome != "uncertain" || m.ActualRoute != routeBefore || m.ObservedRoute != "unrecognized-release" {
		t.Fatalf("unknown rollback observation = state %q outcome %q actual=%q observed=%q", m.State, m.Outcome, m.ActualRoute, m.ObservedRoute)
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

func TestRepeatedRollbackRecoveryIsIdempotent(t *testing.T) {
	m := readyForSwitch(t)
	if err := m.ApplySwitch(); err != nil {
		t.Fatal(err)
	}
	m.Recover(m.CandidateDigest)
	if err := m.RequestRollback(true, true); err != nil {
		t.Fatal(err)
	}
	if err := m.ApplyRollback(true, true); err != nil {
		t.Fatal(err)
	}
	m.Recover(m.PriorDigest)
	m.Recover(m.PriorDigest)
	if m.State != StateRolledBack || m.Outcome != "rolled_back" {
		t.Fatalf("repeated rollback recovery = state %q outcome %q", m.State, m.Outcome)
	}
}

func TestContradictoryObservationAfterRollbackBecomesUncertain(t *testing.T) {
	m := readyForSwitch(t)
	if err := m.ApplySwitch(); err != nil {
		t.Fatal(err)
	}
	m.Recover(m.CandidateDigest)
	if err := m.RequestRollback(true, true); err != nil {
		t.Fatal(err)
	}
	if err := m.ApplyRollback(true, true); err != nil {
		t.Fatal(err)
	}
	m.Recover(m.PriorDigest)
	m.Recover(m.CandidateDigest)
	if m.State != StateUncertain || m.Outcome != "uncertain" {
		t.Fatalf("contradictory rollback observation = state %q outcome %q", m.State, m.Outcome)
	}
}

func TestPriorReleaseMustBeAvailableAndVerifiedForRollback(t *testing.T) {
	tests := []struct {
		name           string
		priorAvailable bool
		priorVerified  bool
	}{
		{name: "unavailable", priorAvailable: false, priorVerified: true},
		{name: "unverified", priorAvailable: true, priorVerified: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := readyForSwitch(t)
			if err := m.ApplySwitch(); err != nil {
				t.Fatal(err)
			}
			m.Recover(m.CandidateDigest)

			if err := m.RequestRollback(tc.priorAvailable, tc.priorVerified); !errors.Is(err, ErrInvalidTransition) {
				t.Fatalf("rollback request error = %v, want invalid transition", err)
			}
			if m.State != StateActive || m.ActualRoute != m.CandidateDigest || m.Outcome != "promoted" {
				t.Fatalf("rejected rollback request changed state: state=%q route=%q outcome=%q", m.State, m.ActualRoute, m.Outcome)
			}
		})
	}
}

func TestApplyRollbackRechecksPriorReleaseProof(t *testing.T) {
	proofLoss := []struct {
		name           string
		priorAvailable bool
		priorVerified  bool
	}{
		{name: "prior release unavailable", priorAvailable: false, priorVerified: true},
		{name: "prior release unverified", priorAvailable: true, priorVerified: false},
	}

	for _, tc := range proofLoss {
		t.Run(tc.name, func(t *testing.T) {
			m := readyForSwitch(t)
			if err := m.ApplySwitch(); err != nil {
				t.Fatal(err)
			}
			m.Recover(m.CandidateDigest)
			if err := m.RequestRollback(true, true); err != nil {
				t.Fatal(err)
			}
			if err := m.ApplyRollback(tc.priorAvailable, tc.priorVerified); !errors.Is(err, ErrInvalidTransition) {
				t.Fatalf("rollback effect without complete prior-release proof = %v, want invalid transition", err)
			}
			if m.State != StateRollingBack || m.ActualRoute != m.CandidateDigest || m.Outcome != "" {
				t.Fatalf("rejected rollback effect changed state: state=%q route=%q outcome=%q", m.State, m.ActualRoute, m.Outcome)
			}
		})
	}
}

func TestTransitionGuardsRejectCallsOutsideTheirState(t *testing.T) {
	tests := []struct {
		name string
		run  func(*Model) error
	}{
		{name: "request switch outside healthy state", run: func(m *Model) error { return m.RequestSwitch() }},
		{name: "apply switch outside switching state", run: func(m *Model) error { return m.ApplySwitch() }},
		{name: "request rollback outside active state", run: func(m *Model) error { return m.RequestRollback(true, true) }},
		{name: "apply rollback outside rolling-back state", run: func(m *Model) error { return m.ApplyRollback(true, true) }},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := New("release-old", "release-new", true)
			if err := tc.run(&m); !errors.Is(err, ErrInvalidTransition) {
				t.Fatalf("out-of-state transition error = %v, want invalid transition", err)
			}
			if m.ActualRoute != m.PriorDigest || m.State != StateReady || m.Outcome != "" {
				t.Fatalf("rejected transition changed model: state=%q route=%q outcome=%q", m.State, m.ActualRoute, m.Outcome)
			}
		})
	}
}

// This test constructs inconsistent in-memory records solely to exercise the guard predicates;
// these states are not reachable through the legal model transition sequence.
func TestRequestSwitchRequiresHealthAndArtifactProof(t *testing.T) {
	tests := []struct {
		name             string
		artifactRecorded bool
		healthPassed     bool
	}{
		{name: "health proof missing", artifactRecorded: true, healthPassed: false},
		{name: "artifact proof missing", artifactRecorded: false, healthPassed: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := New("release-old", "release-new", true)
			m.State = StateHealthy
			m.CandidateStarted = true
			m.ArtifactRecorded = tc.artifactRecorded
			m.HealthPassed = tc.healthPassed
			if err := m.RequestSwitch(); !errors.Is(err, ErrInvalidTransition) {
				t.Fatalf("switch without required proof = %v, want invalid transition", err)
			}
			if m.State != StateHealthy || m.ActualRoute != m.PriorDigest || m.Outcome != "" {
				t.Fatalf("rejected switch changed model: state=%q route=%q outcome=%q", m.State, m.ActualRoute, m.Outcome)
			}
		})
	}
}
func TestAdmissionRetryWithSameKeyAndPayloadReturnsOneJob(t *testing.T) {
	model := NewAdmissionModel()
	request := AdmissionRequest{TenantID: "tenant-a", AppID: "app-a", Key: "request-1", PayloadDigest: "digest-a"}
	first, err := model.Admit(request)
	if err != nil {
		t.Fatal(err)
	}
	second, err := model.Admit(request)
	if err != nil {
		t.Fatal(err)
	}
	if first != second || model.JobCount() != 1 {
		t.Fatalf("retry created or returned a different job: first=%q second=%q count=%d", first, second, model.JobCount())
	}
}

func TestAdmissionRejectsKeyReuseWithDifferentPayload(t *testing.T) {
	model := NewAdmissionModel()
	request := AdmissionRequest{TenantID: "tenant-a", AppID: "app-a", Key: "request-1", PayloadDigest: "digest-a"}
	originalID, err := model.Admit(request)
	if err != nil {
		t.Fatal(err)
	}
	changed := request
	changed.PayloadDigest = "digest-b"
	if _, err := model.Admit(changed); !errors.Is(err, ErrIdempotencyConflict) {
		t.Fatalf("changed payload error = %v", err)
	}
	if retryID, err := model.Admit(request); err != nil || retryID != originalID {
		t.Fatalf("original retry after conflict = id %q, error %v; want original id %q", retryID, err, originalID)
	}
	if _, err := model.Admit(changed); !errors.Is(err, ErrIdempotencyConflict) {
		t.Fatalf("repeated changed payload error = %v, want idempotency conflict", err)
	}
	if model.JobCount() != 1 {
		t.Fatalf("conflict changed job count to %d", model.JobCount())
	}
}

func TestAdmissionKeysAreScopedByTenantAndAppInThisModel(t *testing.T) {
	model := NewAdmissionModel()
	base := AdmissionRequest{TenantID: "tenant-a", AppID: "app-a", Key: "request-1", PayloadDigest: "digest-a"}
	first, err := model.Admit(base)
	if err != nil {
		t.Fatal(err)
	}
	otherTenant := base
	otherTenant.TenantID = "tenant-b"
	second, err := model.Admit(otherTenant)
	if err != nil {
		t.Fatal(err)
	}
	otherApp := base
	otherApp.AppID = "app-b"
	third, err := model.Admit(otherApp)
	if err != nil {
		t.Fatal(err)
	}
	if first == second || first == third || second == third || model.JobCount() != 3 {
		t.Fatalf("scope separation failed: ids=(%q,%q,%q) count=%d", first, second, third, model.JobCount())
	}
}

func TestAdmissionSameKeyDifferentPayloadAcrossScopesIsIndependent(t *testing.T) {
	model := NewAdmissionModel()
	requests := []AdmissionRequest{
		{TenantID: "tenant-a", AppID: "app-a", Key: "request-1", PayloadDigest: "digest-a"},
		{TenantID: "tenant-b", AppID: "app-a", Key: "request-1", PayloadDigest: "digest-b"},
		{TenantID: "tenant-a", AppID: "app-b", Key: "request-1", PayloadDigest: "digest-c"},
	}

	jobIDs := make([]string, len(requests))
	for i, request := range requests {
		jobID, err := model.Admit(request)
		if err != nil {
			t.Fatalf("admit request %d in another modeled scope: %v", i, err)
		}
		jobIDs[i] = jobID
	}

	for i, request := range requests {
		retryID, err := model.Admit(request)
		if err != nil {
			t.Fatalf("retry request %d: %v", i, err)
		}
		if retryID != jobIDs[i] {
			t.Errorf("retry %d returned job %q, want original %q", i, retryID, jobIDs[i])
		}
	}

	if jobIDs[0] == jobIDs[1] || jobIDs[0] == jobIDs[2] || jobIDs[1] == jobIDs[2] {
		t.Errorf("different modeled scopes aliased job IDs: %q", jobIDs)
	}
	if got := model.JobCount(); got != len(requests) {
		t.Errorf("job count = %d, want %d", got, len(requests))
	}
}

func TestAdmissionRejectsMissingScopeOrIdempotencyFields(t *testing.T) {
	model := NewAdmissionModel()
	request := AdmissionRequest{TenantID: "tenant-a", AppID: "app-a", Key: "request-1", PayloadDigest: "digest-a"}
	for _, mutate := range []func(*AdmissionRequest){
		func(r *AdmissionRequest) { r.TenantID = "" },
		func(r *AdmissionRequest) { r.AppID = "" },
		func(r *AdmissionRequest) { r.Key = "" },
		func(r *AdmissionRequest) { r.PayloadDigest = "" },
	} {
		bad := request
		mutate(&bad)
		if _, err := model.Admit(bad); !errors.Is(err, ErrInvalidAdmission) {
			t.Fatalf("invalid request error = %v", err)
		}
	}
	if model.JobCount() != 0 {
		t.Fatalf("invalid requests created %d jobs", model.JobCount())
	}
}

func TestConcurrentIdenticalAdmissionRetriesReturnOneJob(t *testing.T) {
	model := NewAdmissionModel()
	request := AdmissionRequest{TenantID: "tenant-a", AppID: "app-a", Key: "request-1", PayloadDigest: "digest-a"}
	const callers = 64
	ids := make(chan string, callers)
	errs := make(chan error, callers)
	var group sync.WaitGroup
	group.Add(callers)
	for range callers {
		go func() {
			defer group.Done()
			id, err := model.Admit(request)
			if err != nil {
				errs <- err
				return
			}
			ids <- id
		}()
	}
	group.Wait()
	close(ids)
	close(errs)
	for err := range errs {
		t.Errorf("concurrent retry failed: %v", err)
	}
	var first string
	count := 0
	for id := range ids {
		if first == "" {
			first = id
		} else if id != first {
			t.Errorf("concurrent retries returned multiple jobs: %q and %q", first, id)
		}
		count++
	}
	if count != callers || model.JobCount() != 1 {
		t.Fatalf("got %d successful callers and %d jobs; want %d and 1", count, model.JobCount(), callers)
	}
}

func TestConcurrentChangedPayloadsUnderOneKeyHaveSingleWinner(t *testing.T) {
	model := NewAdmissionModel()
	base := AdmissionRequest{TenantID: "tenant-a", AppID: "app-a", Key: "request-1", PayloadDigest: "digest-a"}
	const callers = 64
	results := make(chan struct {
		digest string
		id     string
		err    error
	}, callers)
	var group sync.WaitGroup
	group.Add(callers)
	for i := range callers {
		request := base
		if i%2 == 1 {
			request.PayloadDigest = "digest-b"
		}
		go func() {
			defer group.Done()
			id, err := model.Admit(request)
			results <- struct {
				digest string
				id     string
				err    error
			}{digest: request.PayloadDigest, id: id, err: err}
		}()
	}
	group.Wait()
	close(results)
	var winnerDigest, winnerID string
	accepted, conflicts := 0, 0
	for result := range results {
		switch {
		case result.err == nil:
			accepted++
			if winnerDigest == "" {
				winnerDigest, winnerID = result.digest, result.id
			} else if winnerDigest != result.digest || winnerID != result.id {
				t.Errorf("multiple payloads/jobs accepted: (%q,%q) and (%q,%q)", winnerDigest, winnerID, result.digest, result.id)
			}
		case errors.Is(result.err, ErrIdempotencyConflict):
			conflicts++
		default:
			t.Errorf("unexpected admission error: %v", result.err)
		}
	}
	if accepted == 0 || conflicts != callers-accepted || model.JobCount() != 1 {
		t.Fatalf("accepted=%d conflicts=%d jobs=%d; want one winner and one job", accepted, conflicts, model.JobCount())
	}
}
