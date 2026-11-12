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
		{name: "whitespace-only job id", job: Job{ID: " \t", Environment: "staging"}, node: node, now: now},
		{name: "empty job environment", job: Job{ID: "job-1"}, node: node, now: now},
		{name: "whitespace-only job environment", job: Job{ID: "job-1", Environment: " \t"}, node: node, now: now},
		{name: "empty node id", job: job, node: Node{Environment: "staging", Healthy: true, InventoryFresh: true}, now: now},
		{name: "whitespace-only node id", job: job, node: Node{ID: " \t", Environment: "staging", Healthy: true, InventoryFresh: true}, now: now},
		{name: "empty node environment", job: job, node: Node{ID: "node-1", Healthy: true, InventoryFresh: true}, now: now},
		{name: "whitespace-only node environment", job: job, node: Node{ID: "node-1", Environment: " \t", Healthy: true, InventoryFresh: true}, now: now},
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

func TestSelectNodeIsDeterministicAcrossInputOrder(t *testing.T) {
	now := time.Date(2026, 11, 10, 12, 0, 0, 0, time.UTC)
	request := PlacementRequest{
		Job:            Job{ID: "job-1", Environment: "production"},
		RequiredLabels: map[string]string{"region": "east", "class": "web"},
		CapacityUnits:  2,
	}
	nodes := []Node{
		{ID: "node-z", Environment: "production", Healthy: true, InventoryFresh: true, Labels: map[string]string{"region": "east", "class": "web"}, CapacityKnown: true, AvailableUnits: 8},
		{ID: "node-a", Environment: "production", Healthy: true, InventoryFresh: true, Labels: map[string]string{"region": "east", "class": "web"}, CapacityKnown: true, AvailableUnits: 3},
		{ID: "node-b", Environment: "production", Healthy: true, InventoryFresh: true, Labels: map[string]string{"region": "west", "class": "web"}, CapacityKnown: true, AvailableUnits: 8},
	}
	for _, candidates := range [][]Node{nodes, {nodes[2], nodes[0], nodes[1]}, {nodes[1], nodes[2], nodes[0]}} {
		got, ok := SelectNode(request, candidates, now)
		if !ok || got.ID != "node-a" {
			t.Fatalf("SelectNode() = (%q, %t), want (node-a, true)", got.ID, ok)
		}
	}
}

func TestSelectNodeFailsClosedForUnknownOrInsufficientFacts(t *testing.T) {
	now := time.Date(2026, 11, 10, 12, 0, 0, 0, time.UTC)
	request := PlacementRequest{
		Job:            Job{ID: "job-1", Environment: "staging"},
		RequiredLabels: map[string]string{"pool": "isolated"},
		CapacityUnits:  2,
	}
	valid := Node{
		ID:             "node-1",
		Environment:    "staging",
		Healthy:        true,
		InventoryFresh: true,
		Labels:         map[string]string{"pool": "isolated"},
		CapacityKnown:  true,
		AvailableUnits: 2,
	}
	tests := []struct {
		name string
		edit func(*Node)
	}{
		{name: "unknown capacity", edit: func(n *Node) { n.CapacityKnown = false }},
		{name: "insufficient capacity", edit: func(n *Node) { n.AvailableUnits = 1 }},
		{name: "missing label", edit: func(n *Node) { n.Labels = map[string]string{} }},
		{name: "wrong environment", edit: func(n *Node) { n.Environment = "production" }},
		{name: "stale inventory", edit: func(n *Node) { n.InventoryFresh = false }},
		{name: "unhealthy", edit: func(n *Node) { n.Healthy = false }},
		{name: "maintenance active", edit: func(n *Node) {
			n.Maintenance = []MaintenanceWindow{{StartsAt: now.Add(-time.Minute), EndsAt: now.Add(time.Minute)}}
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node := valid
			tt.edit(&node)
			if got, ok := SelectNode(request, []Node{node}, now); ok {
				t.Fatalf("SelectNode() = (%q, true), want no eligible node", got.ID)
			}
		})
	}
}

func TestSelectNodeRejectsMalformedRequestsAndInventoryIDs(t *testing.T) {
	now := time.Date(2026, 11, 10, 12, 0, 0, 0, time.UTC)
	validJob := Job{ID: "job-1", Environment: "staging"}
	validNode := Node{
		ID: "node-1", Environment: "staging", Healthy: true, InventoryFresh: true,
		CapacityKnown: true, AvailableUnits: 4,
	}
	tests := []struct {
		name    string
		request PlacementRequest
		nodes   []Node
	}{
		{name: "zero capacity request", request: PlacementRequest{Job: validJob, CapacityUnits: 0}, nodes: []Node{validNode}},
		{name: "negative capacity request", request: PlacementRequest{Job: validJob, CapacityUnits: -1}, nodes: []Node{validNode}},
		{name: "empty constraint key", request: PlacementRequest{Job: validJob, RequiredLabels: map[string]string{"": "east"}, CapacityUnits: 1}, nodes: []Node{validNode}},
		{name: "whitespace-only constraint key", request: PlacementRequest{Job: validJob, RequiredLabels: map[string]string{" \t": "east"}, CapacityUnits: 1}, nodes: []Node{validNode}},
		{name: "whitespace-only constraint value", request: PlacementRequest{Job: validJob, RequiredLabels: map[string]string{"region": " \t"}, CapacityUnits: 1}, nodes: []Node{validNode}},
		{name: "whitespace-only job id", request: PlacementRequest{Job: Job{ID: " \t", Environment: "staging"}, CapacityUnits: 1}, nodes: []Node{validNode}},
		{name: "whitespace-only job environment", request: PlacementRequest{Job: Job{ID: "job-1", Environment: " \t"}, CapacityUnits: 1}, nodes: []Node{validNode}},
		{name: "whitespace-only node environment", request: PlacementRequest{Job: validJob, CapacityUnits: 1}, nodes: []Node{{ID: "node-1", Environment: " \t", Healthy: true, InventoryFresh: true, CapacityKnown: true, AvailableUnits: 4}}},
		{name: "wildcard constraint", request: PlacementRequest{Job: validJob, RequiredLabels: map[string]string{"pool": "*"}, CapacityUnits: 1}, nodes: []Node{validNode}},
		{name: "duplicate node IDs", request: PlacementRequest{Job: validJob, CapacityUnits: 1}, nodes: []Node{validNode, validNode}},
		{name: "blank node ID in otherwise valid inventory", request: PlacementRequest{Job: validJob, CapacityUnits: 1}, nodes: []Node{validNode, {Environment: "staging", Healthy: true, InventoryFresh: true, CapacityKnown: true, AvailableUnits: 4}}},
		{name: "whitespace-only node ID in otherwise valid inventory", request: PlacementRequest{Job: validJob, CapacityUnits: 1}, nodes: []Node{validNode, {ID: " \t", Environment: "staging", Healthy: true, InventoryFresh: true, CapacityKnown: true, AvailableUnits: 4}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got, ok := SelectNode(tt.request, tt.nodes, now); ok {
				t.Fatalf("SelectNode() = (%q, true), want fail closed", got.ID)
			}
		})
	}
}
