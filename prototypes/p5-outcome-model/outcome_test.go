package outcomemodel

import "testing"

func TestClassifyRequiresSufficientEvidence(t *testing.T) {
	tests := []struct {
		name     string
		evidence Evidence
		want     Outcome
	}{
		{name: "never dispatched", evidence: Evidence{}, want: NotDispatched},
		{name: "response lost after dispatch", evidence: Evidence{DispatchAttempted: true}, want: Unknown},
		{name: "accepted but effect not verified", evidence: Evidence{DispatchAttempted: true, Response: Accepted}, want: Unknown},
		{name: "reported failure may be partial", evidence: Evidence{DispatchAttempted: true, Response: ReportedFailure}, want: Unknown},
		{name: "explicit rejection before effect", evidence: Evidence{DispatchAttempted: true, Response: RejectedWithoutEffect}, want: RejectedBeforeEffect},
		{name: "verified success", evidence: Evidence{DispatchAttempted: true, Response: Accepted, EffectVerified: true}, want: Succeeded},
		{name: "contradictory response without dispatch", evidence: Evidence{Response: Accepted}, want: Unknown},
		{name: "effect evidence without dispatch", evidence: Evidence{EffectVerified: true}, want: Unknown},
		{name: "rejection contradicts effect evidence", evidence: Evidence{DispatchAttempted: true, Response: RejectedWithoutEffect, EffectVerified: true}, want: Unknown},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Classify(tt.evidence); got != tt.want {
				t.Fatalf("Classify(%+v) = %v, want %v", tt.evidence, got, tt.want)
			}
		})
	}
}

func TestClassifySafetyInvariantsExhaustive(t *testing.T) {
	for response := 0; response <= 255; response++ {
		for dispatch := 0; dispatch <= 1; dispatch++ {
			for verified := 0; verified <= 1; verified++ {
				evidence := Evidence{
					DispatchAttempted: dispatch == 1,
					Response:          ResponseKind(response),
					EffectVerified:    verified == 1,
				}
				got := Classify(evidence)
				if got == Succeeded && !(evidence.DispatchAttempted && evidence.Response == Accepted && evidence.EffectVerified) {
					t.Fatalf("unsafe success classification for %+v", evidence)
				}
				if got == RejectedBeforeEffect && !(evidence.DispatchAttempted && evidence.Response == RejectedWithoutEffect && !evidence.EffectVerified) {
					t.Fatalf("unsafe pre-effect rejection classification for %+v", evidence)
				}
				if !evidence.DispatchAttempted && evidence.Response != NoResponse && got != Unknown {
					t.Fatalf("contradictory undispatched evidence classified as %v: %+v", got, evidence)
				}
			}
		}
	}
}

func TestMayReplayRequiresCompleteBindingAndDurableDedupeForUnknown(t *testing.T) {
	tests := []struct {
		name     string
		outcome  Outcome
		evidence ReplayEvidence
		want     bool
	}{
		{name: "not dispatched but revoked", outcome: NotDispatched},
		{name: "not dispatched and authorized", outcome: NotDispatched, evidence: ReplayEvidence{CurrentlyAuthorized: true}, want: true},
		{name: "unknown with complete binding", outcome: Unknown, evidence: ReplayEvidence{CurrentlyAuthorized: true, SameIdempotencyKey: true, SameTargetNode: true, SameRequest: true, DurableWorkerDedupe: true}, want: true},
		{name: "unknown with revoked authorization", outcome: Unknown, evidence: ReplayEvidence{SameIdempotencyKey: true, SameTargetNode: true, SameRequest: true, DurableWorkerDedupe: true}},
		{name: "unknown with new key", outcome: Unknown, evidence: ReplayEvidence{CurrentlyAuthorized: true, SameTargetNode: true, SameRequest: true, DurableWorkerDedupe: true}},
		{name: "unknown with changed target", outcome: Unknown, evidence: ReplayEvidence{CurrentlyAuthorized: true, SameIdempotencyKey: true, SameRequest: true, DurableWorkerDedupe: true}},
		{name: "unknown with changed request", outcome: Unknown, evidence: ReplayEvidence{CurrentlyAuthorized: true, SameIdempotencyKey: true, SameTargetNode: true, DurableWorkerDedupe: true}},
		{name: "unknown without durable dedupe", outcome: Unknown, evidence: ReplayEvidence{CurrentlyAuthorized: true, SameIdempotencyKey: true, SameTargetNode: true, SameRequest: true}},
		{name: "rejected", outcome: RejectedBeforeEffect, evidence: ReplayEvidence{CurrentlyAuthorized: true, SameIdempotencyKey: true, SameTargetNode: true, SameRequest: true, DurableWorkerDedupe: true}},
		{name: "succeeded", outcome: Succeeded, evidence: ReplayEvidence{CurrentlyAuthorized: true, SameIdempotencyKey: true, SameTargetNode: true, SameRequest: true, DurableWorkerDedupe: true}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := MayReplay(tt.outcome, tt.evidence); got != tt.want {
				t.Fatalf("MayReplay(%v, evidence=%+v) = %t, want %t", tt.outcome, tt.evidence, got, tt.want)
			}
		})
	}
}

func TestMayReplaySafetyInvariantExhaustive(t *testing.T) {
	for outcome := 0; outcome <= 255; outcome++ {
		for authorization := 0; authorization <= 1; authorization++ {
			for key := 0; key <= 1; key++ {
				for target := 0; target <= 1; target++ {
					for request := 0; request <= 1; request++ {
						for dedupe := 0; dedupe <= 1; dedupe++ {
							evidence := ReplayEvidence{
								CurrentlyAuthorized: authorization == 1,
								SameIdempotencyKey:  key == 1,
								SameTargetNode:      target == 1,
								SameRequest:         request == 1,
								DurableWorkerDedupe: dedupe == 1,
							}
							got := MayReplay(Outcome(outcome), evidence)
							allowed := evidence.CurrentlyAuthorized && Outcome(outcome) == NotDispatched ||
								evidence.CurrentlyAuthorized && Outcome(outcome) == Unknown && evidence.SameIdempotencyKey && evidence.SameTargetNode && evidence.SameRequest && evidence.DurableWorkerDedupe
							if got != allowed {
								t.Fatalf("MayReplay(%d, %+v) = %t, want %t", outcome, evidence, got, allowed)
							}
						}
					}
				}
			}
		}
	}
}
