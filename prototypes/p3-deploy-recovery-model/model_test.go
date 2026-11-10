package recovery

import (
	"errors"
	"sync"
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
	if _, err := model.Admit(request); err != nil {
		t.Fatal(err)
	}
	request.PayloadDigest = "digest-b"
	if _, err := model.Admit(request); !errors.Is(err, ErrIdempotencyConflict) {
		t.Fatalf("changed payload error = %v", err)
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
