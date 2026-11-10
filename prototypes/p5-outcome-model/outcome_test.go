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

func TestMayReplayRequiresCompleteBindingAndDurableDedupeForUnknown(t *testing.T) {
	tests := []struct {
		name     string
		outcome  Outcome
		evidence ReplayEvidence
		want     bool
	}{
		{name: "not dispatched", outcome: NotDispatched, want: true},
		{name: "unknown with complete binding", outcome: Unknown, evidence: ReplayEvidence{SameIdempotencyKey: true, SameTargetNode: true, SameRequest: true, DurableWorkerDedupe: true}, want: true},
		{name: "unknown with new key", outcome: Unknown, evidence: ReplayEvidence{SameTargetNode: true, SameRequest: true, DurableWorkerDedupe: true}},
		{name: "unknown with changed target", outcome: Unknown, evidence: ReplayEvidence{SameIdempotencyKey: true, SameRequest: true, DurableWorkerDedupe: true}},
		{name: "unknown with changed request", outcome: Unknown, evidence: ReplayEvidence{SameIdempotencyKey: true, SameTargetNode: true, DurableWorkerDedupe: true}},
		{name: "unknown without durable dedupe", outcome: Unknown, evidence: ReplayEvidence{SameIdempotencyKey: true, SameTargetNode: true, SameRequest: true}},
		{name: "rejected", outcome: RejectedBeforeEffect, evidence: ReplayEvidence{SameIdempotencyKey: true, SameTargetNode: true, SameRequest: true, DurableWorkerDedupe: true}},
		{name: "succeeded", outcome: Succeeded, evidence: ReplayEvidence{SameIdempotencyKey: true, SameTargetNode: true, SameRequest: true, DurableWorkerDedupe: true}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := MayReplay(tt.outcome, tt.evidence); got != tt.want {
				t.Fatalf("MayReplay(%v, evidence=%+v) = %t, want %t", tt.outcome, tt.evidence, got, tt.want)
			}
		})
	}
}
