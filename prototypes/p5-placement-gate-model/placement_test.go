package placementmodel

import (
	"testing"
	"time"
)

func TestCanPlaceRequiresConcreteFreshHealthyExactEnvironment(t *testing.T) {
	now := time.Date(2026, 11, 10, 12, 0, 0, 0, time.UTC)
	job := Job{ID: "job-1", Environment: "staging"}
	node := Node{ID: "node-1", Environment: "staging", Healthy: true, InventoryFresh: true}
	if !CanPlace(job, node, now) {
		t.Fatal("exact environment with fresh healthy inventory should be eligible")
	}

	tests := []struct {
		name string
		job  Job
		node Node
		now  time.Time
	}{
		{name: "empty job id", job: Job{Environment: "staging"}, node: node, now: now},
		{name: "empty job environment", job: Job{ID: "job-1"}, node: node, now: now},
		{name: "empty node id", job: job, node: Node{Environment: "staging", Healthy: true, InventoryFresh: true}, now: now},
		{name: "empty node environment", job: job, node: Node{ID: "node-1", Healthy: true, InventoryFresh: true}, now: now},
		{name: "cross environment", job: job, node: Node{ID: "node-1", Environment: "production", Healthy: true, InventoryFresh: true}, now: now},
		{name: "unhealthy", job: job, node: Node{ID: "node-1", Environment: "staging", InventoryFresh: true}, now: now},
		{name: "stale inventory", job: job, node: Node{ID: "node-1", Environment: "staging", Healthy: true}, now: now},
		{name: "unknown time", job: job, node: node},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if CanPlace(tt.job, tt.node, tt.now) {
				t.Fatal("incomplete or mismatched placement facts were accepted")
			}
		})
	}
}

func TestCanPlaceUsesHalfOpenMaintenanceWindow(t *testing.T) {
	start := time.Date(2026, 11, 10, 12, 0, 0, 0, time.UTC)
	end := start.Add(time.Hour)
	job := Job{ID: "job-1", Environment: "production"}
	node := Node{
		ID:             "node-1",
		Environment:    "production",
		Healthy:        true,
		InventoryFresh: true,
		Maintenance:    []MaintenanceWindow{{StartsAt: start, EndsAt: end}},
	}
	for _, tt := range []struct {
		name string
		now  time.Time
		want bool
	}{
		{name: "before start", now: start.Add(-time.Nanosecond), want: true},
		{name: "at start", now: start},
		{name: "during window", now: start.Add(time.Minute)},
		{name: "at end", now: end, want: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := CanPlace(job, node, tt.now); got != tt.want {
				t.Fatalf("CanPlace at %s = %t, want %t", tt.now, got, tt.want)
			}
		})
	}
}

func TestCanPlaceFailsClosedForMalformedMaintenanceWindow(t *testing.T) {
	now := time.Date(2026, 11, 10, 12, 0, 0, 0, time.UTC)
	job := Job{ID: "job-1", Environment: "production"}
	node := Node{ID: "node-1", Environment: "production", Healthy: true, InventoryFresh: true}
	for _, window := range []MaintenanceWindow{
		{},
		{StartsAt: now, EndsAt: now},
		{StartsAt: now.Add(time.Hour), EndsAt: now},
	} {
		node.Maintenance = []MaintenanceWindow{window}
		if CanPlace(job, node, now) {
			t.Fatalf("malformed window %+v should fail closed", window)
		}
	}
}
