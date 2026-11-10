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

func TestMayReplayRequiresSameKeyAndDurableDedupeForUnknown(t *testing.T) {
	tests := []struct {
		name    string
		outcome Outcome
		sameKey bool
		durable bool
		want    bool
	}{
		{name: "not dispatched", outcome: NotDispatched, want: true},
		{name: "unknown with same key and durable dedupe", outcome: Unknown, sameKey: true, durable: true, want: true},
		{name: "unknown with new key", outcome: Unknown, durable: true},
		{name: "unknown without durable dedupe", outcome: Unknown, sameKey: true},
		{name: "rejected", outcome: RejectedBeforeEffect, sameKey: true, durable: true},
		{name: "succeeded", outcome: Succeeded, sameKey: true, durable: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := MayReplay(tt.outcome, tt.sameKey, tt.durable); got != tt.want {
				t.Fatalf("MayReplay(%v, sameKey=%t, durable=%t) = %t, want %t", tt.outcome, tt.sameKey, tt.durable, got, tt.want)
			}
		})
	}
}
