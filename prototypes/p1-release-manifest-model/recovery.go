package releasemodel

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"regexp"
)

// RecoveryPhase describes only the last simulated durable journal boundary.
// A journal state never proves that the external switch or rollback occurred.
type RecoveryPhase string

const (
	PhasePrepared          RecoveryPhase = "prepared"
	PhaseSwitchRequested   RecoveryPhase = "switch_requested"
	PhaseRollbackRequested RecoveryPhase = "rollback_requested"
	PhaseCommitted         RecoveryPhase = "committed"
	PhaseRolledBack        RecoveryPhase = "rolled_back"
)

// RecoveryJournal is a candidate record for an interrupted *offline* upgrade.
// All identity fields are immutable for one attempt; no filesystem is touched.
type RecoveryJournal struct {
	AttemptID        string        `json:"attempt_id"`
	PreviousCommit   string        `json:"previous_commit"`
	CandidateCommit  string        `json:"candidate_commit"`
	CandidateVersion string        `json:"candidate_version"`
	ManifestSHA256   string        `json:"manifest_sha256"`
	Phase            RecoveryPhase `json:"phase"`
}

type RecoveryHealth string

const (
	HealthUnknown   RecoveryHealth = "unknown"
	HealthHealthy   RecoveryHealth = "healthy"
	HealthUnhealthy RecoveryHealth = "unhealthy"
)

// RecoveryObservation models an external *claimed* observation after all
// previously dispatched effects have been fenced. The caller must establish
// fencing/health/identity authenticity in a separate reviewed implementation.
type RecoveryObservation struct {
	ActiveCommit     string
	Health           RecoveryHealth
	EffectsFenced    bool
	PreviousRetained bool
}

// These are advisory decisions, NEVER an executable action or release success.
type RecoveryAction string

const (
	ActionManualReview    RecoveryAction = "manual_review"
	ActionKeepPrevious    RecoveryAction = "keep_previous"
	ActionCommitCandidate RecoveryAction = "candidate_ready_for_reviewed_commit"
	ActionRequestRollback RecoveryAction = "request_reviewed_rollback"
	ActionConfirmRollback RecoveryAction = "rollback_consistent"
	ActionConsistent      RecoveryAction = "record_consistent"
)

type RecoveryPlan struct {
	Action RecoveryAction
	Reason string
}

const maxJournalBytes = 4096

var attemptIDPattern = regexp.MustCompile(`^[0-9a-f]{32}$`)

func validRecoveryJournal(j RecoveryJournal) bool {
	if !attemptIDPattern.MatchString(j.AttemptID) ||
		!commitPattern.MatchString(j.PreviousCommit) ||
		!commitPattern.MatchString(j.CandidateCommit) ||
		j.CandidateCommit == j.PreviousCommit ||
		!versionPattern.MatchString(j.CandidateVersion) ||
		!shaPattern.MatchString(j.ManifestSHA256) {
		return false
	}
	switch j.Phase {
	case PhasePrepared, PhaseSwitchRequested, PhaseRollbackRequested, PhaseCommitted, PhaseRolledBack:
		return true
	default:
		return false
	}
}

// ReadRecoveryJournal accepts exactly json.Marshal's canonical representation
// and rejects duplicates/unknown keys/alternate encodings. It does not verify
// durable storage integrity, because it has no independent journal signature.
func ReadRecoveryJournal(raw []byte) (RecoveryJournal, error) {
	if len(raw) == 0 || len(raw) > maxJournalBytes {
		return RecoveryJournal{}, rejected("invalid recovery journal length")
	}
	var j RecoveryJournal
	if err := json.Unmarshal(raw, &j); err != nil || !validRecoveryJournal(j) {
		return RecoveryJournal{}, rejected("invalid recovery journal fields")
	}
	canonical, err := json.Marshal(j)
	if err != nil || !bytes.Equal(canonical, raw) {
		return RecoveryJournal{}, rejected("noncanonical recovery journal")
	}
	return j, nil
}

// PlanUpgradeRecovery validates the candidate AGAIN against the independent
// trust pins and current in-memory bytes. It returns a *recommendation* only;
// no effect, journal write, signing, installation or audit is performed.
func PlanUpgradeRecovery(expected Expected, signed SignedFixture, payloads map[string][]byte, j RecoveryJournal, observation RecoveryObservation) (RecoveryPlan, error) {
	manual := RecoveryPlan{Action: ActionManualReview, Reason: "insufficient independent evidence; do not mutate the host"}
	if !validRecoveryJournal(j) || !commitPattern.MatchString(observation.ActiveCommit) {
		return manual, rejected("invalid journal or active identity")
	}
	if observation.Health != HealthHealthy && observation.Health != HealthUnhealthy && observation.Health != HealthUnknown {
		return manual, rejected("invalid health observation")
	}
	if err := Verify(expected, signed, payloads); err != nil {
		return manual, err
	}
	manifestDigest := sha256.Sum256(signed.Manifest)
	if j.CandidateCommit != expected.SourceCommit || j.CandidateVersion != expected.Release ||
		j.ManifestSHA256 != hex.EncodeToString(manifestDigest[:]) {
		return manual, rejected("journal changed candidate identity or validated bytes")
	}
	if !observation.EffectsFenced {
		return manual, nil
	}

	if observation.ActiveCommit == j.PreviousCommit {
		switch j.Phase {
		case PhasePrepared, PhaseSwitchRequested:
			return RecoveryPlan{Action: ActionKeepPrevious, Reason: "prior version observed; no candidate completion inferred"}, nil
		case PhaseRollbackRequested, PhaseRolledBack:
			return RecoveryPlan{Action: ActionConfirmRollback, Reason: "previous version observed after rollback"}, nil
		default:
			return manual, nil // committed record disagrees with active effect
		}
	}
	if observation.ActiveCommit != j.CandidateCommit {
		return manual, nil
	}
	switch j.Phase {
	case PhaseSwitchRequested:
		if !observation.PreviousRetained {
			return manual, nil // do not destroy the only rollback copy
		}
		if observation.Health == HealthHealthy {
			return RecoveryPlan{Action: ActionCommitCandidate, Reason: "candidate observed healthy; external audit/commit still required"}, nil
		}
		if observation.Health == HealthUnhealthy {
			return RecoveryPlan{Action: ActionRequestRollback, Reason: "candidate unhealthy; prior copy reportedly retained"}, nil
		}
	case PhaseRollbackRequested:
		if observation.PreviousRetained {
			return RecoveryPlan{Action: ActionRequestRollback, Reason: "rollback not yet observed complete"}, nil
		}
	case PhaseCommitted:
		if observation.Health == HealthHealthy {
			return RecoveryPlan{Action: ActionConsistent, Reason: "record and observation agree; no release approval inferred"}, nil
		}
	}
	return manual, nil
}
