// Package outcomemodel explores conservative reporting for remote operations.
// It is a local design prototype and is not used by Tong Panel runtime code.
package outcomemodel

// Outcome is the coordinator's evidence-based view of one remote attempt.
type Outcome uint8

const (
	NotDispatched Outcome = iota
	Unknown
	RejectedBeforeEffect
	Succeeded
)

// ResponseKind records the strongest response evidence received from a remote
// worker. A reported failure does not establish that no side effect occurred.
type ResponseKind uint8

const (
	NoResponse ResponseKind = iota
	Accepted
	RejectedWithoutEffect
	ReportedFailure
)

// Evidence contains observations only; callers must not infer missing facts.
type Evidence struct {
	DispatchAttempted bool
	Response          ResponseKind
	EffectVerified    bool
}

// Classify maps incomplete or contradictory observations to Unknown. Success
// requires both an acceptance response and independent effect verification.
func Classify(e Evidence) Outcome {
	if !e.DispatchAttempted {
		if e.Response == NoResponse && !e.EffectVerified {
			return NotDispatched
		}
		return Unknown
	}

	switch e.Response {
	case RejectedWithoutEffect:
		if !e.EffectVerified {
			return RejectedBeforeEffect
		}
	case Accepted:
		if e.EffectVerified {
			return Succeeded
		}
	}
	return Unknown
}

// MayReplay reports whether another dispatch is safe under the evidence and
// explicitly supplied deduplication facts. An Unknown attempt may only be
// replayed with the same idempotency key against a worker with durable dedupe.
func MayReplay(outcome Outcome, sameKey, durableWorkerDedupe bool) bool {
	switch outcome {
	case NotDispatched:
		return true
	case Unknown:
		return sameKey && durableWorkerDedupe
	default:
		return false
	}
}
