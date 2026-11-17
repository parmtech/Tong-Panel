// Package placementmodel explores fail-closed placement predicates.
// It is an offline design prototype and is not used by Tong Panel runtime code.
package placementmodel

import (
	"strings"
	"time"
)

// MaintenanceWindow is a half-open interval: StartsAt is included and EndsAt
// is excluded. Timezone interpretation and schedule ownership are out of scope.
type MaintenanceWindow struct {
	StartsAt time.Time
	EndsAt   time.Time
}

// Node is a synthetic placement candidate. Environment and health facts must
// come from a trusted inventory source in any future implementation.
type Node struct {
	ID             string
	Environment    string
	Healthy        bool
	InventoryFresh bool
	Labels         map[string]string
	CapacityKnown  bool
	AvailableUnits int
	Maintenance    []MaintenanceWindow
}

// Job is a synthetic request with a required environment binding.
type Job struct {
	ID          string
	Environment string
}

// PlacementRequest contains explicit scheduling constraints for the offline
// candidate selector. CapacityUnits are abstract; no resource model is chosen.
type PlacementRequest struct {
	Job            Job
	RequiredLabels map[string]string
	CapacityUnits  int
}

// CanPlace permits a candidate only with concrete exact environment bindings,
// fresh healthy node facts, and a valid time outside every maintenance window.
// This does not select a node or establish authorization or isolation.
func CanPlace(job Job, node Node, now time.Time) bool {
	if strings.TrimSpace(job.ID) == "" || strings.TrimSpace(job.Environment) == "" ||
		strings.TrimSpace(node.ID) == "" || strings.TrimSpace(node.Environment) == "" || now.IsZero() {
		return false
	}
	if !node.Healthy || !node.InventoryFresh || job.Environment != node.Environment {
		return false
	}
	for _, window := range node.Maintenance {
		if window.StartsAt.IsZero() || window.EndsAt.IsZero() || !window.StartsAt.Before(window.EndsAt) {
			return false
		}
		if !now.Before(window.StartsAt) && now.Before(window.EndsAt) {
			return false
		}
	}
	return true
}

// SelectNode returns the lexicographically smallest eligible node ID. It
// requires fresh inventory, exact environment and label matches, known
// sufficient abstract capacity, and maintenance eligibility. Invalid or
// wildcard constraints, non-positive requests, blank/duplicate node IDs fail
// closed. This is a candidate deterministic rule only: it does not reserve
// capacity, provide fairness, authorize a job, or establish isolation.
func SelectNode(request PlacementRequest, nodes []Node, now time.Time) (Node, bool) {
	if request.CapacityUnits <= 0 || !validConstraints(request.RequiredLabels) {
		return Node{}, false
	}

	seenIDs := make(map[string]struct{}, len(nodes))
	for _, node := range nodes {
		if strings.TrimSpace(node.ID) == "" {
			return Node{}, false
		}
		if _, exists := seenIDs[node.ID]; exists {
			return Node{}, false
		}
		seenIDs[node.ID] = struct{}{}
	}

	var selected Node
	found := false
	for _, node := range nodes {
		if !CanPlace(request.Job, node, now) || !node.CapacityKnown ||
			node.AvailableUnits < request.CapacityUnits ||
			!matchesLabels(node.Labels, request.RequiredLabels) {
			continue
		}
		if !found || node.ID < selected.ID {
			selected = node
			found = true
		}
	}
	return selected, found
}

func validConstraints(constraints map[string]string) bool {
	for key, value := range constraints {
		if strings.TrimSpace(key) == "" || strings.TrimSpace(value) == "" || key == "*" || value == "*" {
			return false
		}
	}
	return true
}

func matchesLabels(labels, constraints map[string]string) bool {
	for key, required := range constraints {
		if labels[key] != required {
			return false
		}
	}
	return true
}
