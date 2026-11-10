// Package placementmodel explores two fail-closed placement predicates.
// It is an offline design prototype and is not used by Tong Panel runtime code.
package placementmodel

import "time"

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
	Maintenance    []MaintenanceWindow
}

// Job is a synthetic request with a required environment binding.
type Job struct {
	ID          string
	Environment string
}

// CanPlace permits a candidate only with concrete exact environment bindings,
// fresh healthy node facts, and a valid time outside every maintenance window.
// This does not select a node or establish authorization or isolation.
func CanPlace(job Job, node Node, now time.Time) bool {
	if job.ID == "" || job.Environment == "" || node.ID == "" || node.Environment == "" || now.IsZero() {
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
