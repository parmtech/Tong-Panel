// Package recovery is a non-normative, in-memory P3 deployment recovery model.
package recovery

import "errors"

type State string

const (
	StateReady       State = "ready"
	StateStarting    State = "starting"
	StateHealthy     State = "healthy"
	StateSwitching   State = "switching"
	StateActive      State = "active"
	StateRollingBack State = "rolling_back"
	StateRolledBack  State = "rolled_back"
	StateFailed      State = "failed"
	StateUncertain   State = "outcome_uncertain"
)

var ErrArtifactNotVerified = errors.New("candidate artifact is not verified")
var ErrInvalidTransition = errors.New("invalid recovery-model transition")

// Model keeps the simulated control-plane record and external route separate.
// Tests can stop between mutations to represent process/host interruption.
type Model struct {
	State State

	PriorDigest       string
	CandidateDigest   string
	ArtifactRecorded  bool
	CandidateVerified bool
	CandidateStarted  bool
	HealthPassed      bool

	ActualRoute   string
	ObservedRoute string
	Outcome       string
}

func New(priorDigest, candidateDigest string, candidateVerified bool) Model {
	return Model{
		State:             StateReady,
		PriorDigest:       priorDigest,
		CandidateDigest:   candidateDigest,
		CandidateVerified: candidateVerified,
		ActualRoute:       priorDigest,
		ObservedRoute:     priorDigest,
	}
}

func (m *Model) RecordArtifact(digest string) error {
	if m.State != StateReady || !m.CandidateVerified || digest == "" || digest != m.CandidateDigest {
		return ErrArtifactNotVerified
	}
	m.ArtifactRecorded = true
	return nil
}

func (m *Model) StartCandidate() error {
	if m.State != StateReady || !m.ArtifactRecorded || !m.CandidateVerified {
		return ErrInvalidTransition
	}
	m.CandidateStarted = true
	m.State = StateStarting
	return nil
}

func (m *Model) MarkHealthy() error {
	if m.State != StateStarting || !m.CandidateStarted {
		return ErrInvalidTransition
	}
	m.HealthPassed = true
	m.State = StateHealthy
	return nil
}

func (m *Model) RequestSwitch() error {
	if m.State != StateHealthy || !m.HealthPassed || !m.ArtifactRecorded {
		return ErrInvalidTransition
	}
	m.State = StateSwitching
	return nil
}

// ApplySwitch models an external route effect. It does not mean the control
// plane has observed or durably recorded that effect.
func (m *Model) ApplySwitch() error {
	if m.State != StateSwitching || !m.CandidateVerified || !m.ArtifactRecorded || !m.CandidateStarted || !m.HealthPassed {
		return ErrInvalidTransition
	}
	m.ActualRoute = m.CandidateDigest
	return nil
}

func (m *Model) RequestRollback(priorAvailable, priorVerified bool) error {
	if m.State != StateActive || !priorAvailable || !priorVerified {
		return ErrInvalidTransition
	}
	m.State = StateRollingBack
	m.Outcome = ""
	return nil
}

// ApplyRollback models an external effect, separated from its observation.
func (m *Model) ApplyRollback(priorAvailable, priorVerified bool) error {
	if m.State != StateRollingBack || !priorAvailable || !priorVerified {
		return ErrInvalidTransition
	}
	m.ActualRoute = m.PriorDigest
	return nil
}

// Recover consumes an independently observed route after interruption.
// Unknown or inconsistent observations never become a success outcome.
func (m *Model) Recover(observedRoute string) {
	m.ObservedRoute = observedRoute
	switch m.State {
	case StateSwitching:
		switch {
		case observedRoute == m.CandidateDigest && m.CandidateVerified && m.ArtifactRecorded && m.CandidateStarted && m.HealthPassed:
			m.State = StateActive
			m.Outcome = "promoted"
		case observedRoute == m.PriorDigest:
			m.State = StateFailed
			m.Outcome = "not_promoted"
		default:
			m.markUncertain()
		}
	case StateRollingBack:
		switch observedRoute {
		case m.PriorDigest:
			m.State = StateRolledBack
			m.Outcome = "rolled_back"
		case m.CandidateDigest:
			m.State = StateActive
			m.Outcome = "rollback_not_observed"
		default:
			m.markUncertain()
		}
	case StateActive:
		if observedRoute == m.CandidateDigest && m.CandidateVerified && m.ArtifactRecorded && m.CandidateStarted && m.HealthPassed {
			m.Outcome = "promoted"
		} else {
			m.markUncertain()
		}
	case StateRolledBack:
		if observedRoute == m.PriorDigest {
			m.Outcome = "rolled_back"
		} else {
			m.markUncertain()
		}
	case StateFailed:
		if observedRoute == m.PriorDigest {
			m.Outcome = "not_promoted"
		} else {
			m.markUncertain()
		}
	default:
		m.markUncertain()
	}
}

func (m *Model) markUncertain() {
	m.State = StateUncertain
	m.Outcome = "uncertain"
}
