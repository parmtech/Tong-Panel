package review

import (
	"errors"
	"strings"
	"testing"
)

func samplePlan() Plan {
	return Plan{
		ID:                "migration-opaque-1",
		TenantID:          "tenant-opaque-1",
		SourceSHA256:      "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		ParserVersion:     "cpanel-fixture-v1",
		DestinationID:     "host-opaque-1",
		InventoryRevision: "revision-7",
		Actions: []Action{
			{Ref: "step-1", Kind: "create_site", ObjectRef: "site-opaque-1", ExpectedRevision: "fixture-revision-4"},
			{Ref: "step-2", Kind: "reset_mailbox_credential", ObjectRef: "mailbox-opaque-1", DependsOn: []string{"step-1"}},
		},
	}
}

func currentFor(plan Plan, review Review) ExecutionContext {
	return ExecutionContext{
		TenantID:          plan.TenantID,
		DestinationID:     plan.DestinationID,
		InventoryRevision: plan.InventoryRevision,
		PolicyRevision:    review.PolicyRevision,
		PrincipalID:       review.PrincipalID,
		Authorization: AuthorizationResult{
			TenantID:        plan.TenantID,
			DestinationID:   plan.DestinationID,
			PrincipalID:     review.PrincipalID,
			ActionSetSHA256: review.ActionSetSHA256,
			PolicyRevision:  review.PolicyRevision,
			Allowed:         true,
		},
	}
}

func TestRevalidateAcceptsUnchangedReviewedScope(t *testing.T) {
	plan := samplePlan()
	review, err := RecordReview(plan, "operator-opaque-1", "policy-3")
	if err != nil {
		t.Fatal(err)
	}
	if err := Revalidate(plan, review, currentFor(plan, review)); err != nil {
		t.Fatalf("Revalidate() error = %v", err)
	}
}

func TestRevalidateRejectsPlanOrActionMutation(t *testing.T) {
	tests := map[string]func(*Plan){
		"source changed":            func(p *Plan) { p.SourceSHA256 = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb" },
		"plan id changed":           func(p *Plan) { p.ID = "other-migration" },
		"tenant changed":            func(p *Plan) { p.TenantID = "other-tenant" },
		"parser version changed":    func(p *Plan) { p.ParserVersion = "different-parser" },
		"destination changed":       func(p *Plan) { p.DestinationID = "other-host" },
		"expected revision changed": func(p *Plan) { p.Actions[0].ExpectedRevision = "revision-99" },
		"dependency changed":        func(p *Plan) { p.Actions[1].DependsOn = nil },
		"action removed":            func(p *Plan) { p.Actions = p.Actions[:1] },
		"action target changed":     func(p *Plan) { p.Actions[0].ObjectRef = "other-object" },
		"action added": func(p *Plan) {
			p.Actions = append(p.Actions, Action{Ref: "step-3", Kind: "create_db", ObjectRef: "db-opaque-1"})
		},
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			plan := samplePlan()
			review, err := RecordReview(plan, "operator-opaque-1", "policy-3")
			if err != nil {
				t.Fatal(err)
			}
			mutate(&plan)
			if err := Revalidate(plan, review, currentFor(plan, review)); !errors.Is(err, ErrPlanChanged) {
				t.Fatalf("Revalidate() error = %v, want ErrPlanChanged", err)
			}
		})
	}
}

func TestRevalidateRejectsChangedAuthority(t *testing.T) {
	plan := samplePlan()
	review, err := RecordReview(plan, "operator-opaque-1", "policy-3")
	if err != nil {
		t.Fatal(err)
	}
	tests := map[string]func(*ExecutionContext){
		"tenant":                 func(c *ExecutionContext) { c.TenantID = "other-tenant" },
		"missing tenant":         func(c *ExecutionContext) { c.TenantID = " " },
		"destination":            func(c *ExecutionContext) { c.DestinationID = "other-host" },
		"missing destination":    func(c *ExecutionContext) { c.DestinationID = " " },
		"principal":              func(c *ExecutionContext) { c.PrincipalID = "other-operator" },
		"missing principal":      func(c *ExecutionContext) { c.PrincipalID = " " },
		"authorized tenant":      func(c *ExecutionContext) { c.Authorization.TenantID = "other-tenant" },
		"authorized destination": func(c *ExecutionContext) { c.Authorization.DestinationID = "other-host" },
		"authorized principal":   func(c *ExecutionContext) { c.Authorization.PrincipalID = "other-operator" },
		"authorized action set":  func(c *ExecutionContext) { c.Authorization.ActionSetSHA256 = "different" },
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			current := currentFor(plan, review)
			mutate(&current)
			if err := Revalidate(plan, review, current); !errors.Is(err, ErrAuthorityChanged) {
				t.Fatalf("Revalidate() error = %v, want ErrAuthorityChanged", err)
			}
		})
	}
}

func TestRevalidateRejectsStaleDestinationInventory(t *testing.T) {
	plan := samplePlan()
	review, err := RecordReview(plan, "operator-opaque-1", "policy-3")
	if err != nil {
		t.Fatal(err)
	}
	current := currentFor(plan, review)
	current.InventoryRevision = "revision-8"
	if err := Revalidate(plan, review, current); !errors.Is(err, ErrInventoryChanged) {
		t.Fatalf("Revalidate() error = %v, want ErrInventoryChanged", err)
	}
}

func TestRevalidateRejectsStalePlanInventoryRevision(t *testing.T) {
	plan := samplePlan()
	review, err := RecordReview(plan, "operator-opaque-1", "policy-3")
	if err != nil {
		t.Fatal(err)
	}
	plan.InventoryRevision = "revision-8"
	if err := Revalidate(plan, review, currentFor(plan, review)); !errors.Is(err, ErrPlanChanged) {
		t.Fatalf("Revalidate() error = %v, want ErrPlanChanged", err)
	}
}

func TestRevalidateRejectsStalePolicyRevision(t *testing.T) {
	plan := samplePlan()
	review, err := RecordReview(plan, "operator-opaque-1", "policy-3")
	if err != nil {
		t.Fatal(err)
	}
	tests := map[string]func(*ExecutionContext){
		"context revision is stale": func(current *ExecutionContext) {
			current.PolicyRevision = "policy-4"
		},
		"authorization revision is stale": func(current *ExecutionContext) {
			current.Authorization.PolicyRevision = "policy-4"
		},
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			current := currentFor(plan, review)
			mutate(&current)
			if err := Revalidate(plan, review, current); !errors.Is(err, ErrPolicyChanged) {
				t.Fatalf("Revalidate() error = %v, want ErrPolicyChanged", err)
			}
		})
	}
}

func TestRevalidateRequiresCurrentAuthorization(t *testing.T) {
	plan := samplePlan()
	review, err := RecordReview(plan, "operator-opaque-1", "policy-3")
	if err != nil {
		t.Fatal(err)
	}
	current := currentFor(plan, review)
	current.Authorization.Allowed = false
	if err := Revalidate(plan, review, current); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("Revalidate() error = %v, want ErrUnauthorized", err)
	}
}

func TestCanonicalHashesIgnoreActionAndDependencyPermutation(t *testing.T) {
	plan := samplePlan()
	plan.Actions = append(plan.Actions, Action{Ref: "step-3", Kind: "create_db", ObjectRef: "db-opaque-1", DependsOn: []string{"step-1", "step-2"}})
	review, err := RecordReview(plan, "operator-opaque-1", "policy-3")
	if err != nil {
		t.Fatal(err)
	}
	plan.Actions[0], plan.Actions[2] = plan.Actions[2], plan.Actions[0]
	plan.Actions[0].DependsOn[0], plan.Actions[0].DependsOn[1] = plan.Actions[0].DependsOn[1], plan.Actions[0].DependsOn[0]
	if err := Revalidate(plan, review, currentFor(plan, review)); err != nil {
		t.Fatalf("Revalidate() rejected reordered equivalent plan: %v", err)
	}
}

func TestRecordReviewDoesNotReorderCallerSlices(t *testing.T) {
	plan := samplePlan()
	plan.Actions[0], plan.Actions[1] = plan.Actions[1], plan.Actions[0]
	plan.Actions = append(plan.Actions, Action{
		Ref: "step-3", Kind: "create_db", ObjectRef: "db-opaque-1", DependsOn: []string{"step-2", "step-1"},
	})
	before := append([]Action(nil), plan.Actions...)
	for i := range before {
		before[i].DependsOn = append([]string(nil), plan.Actions[i].DependsOn...)
	}

	if _, err := RecordReview(plan, "operator-opaque-1", "policy-3"); err != nil {
		t.Fatal(err)
	}
	if len(plan.Actions) != len(before) {
		t.Fatalf("RecordReview() changed action count from %d to %d", len(before), len(plan.Actions))
	}
	for i := range before {
		if plan.Actions[i].Ref != before[i].Ref {
			t.Errorf("RecordReview() reordered action %d: got %q, want %q", i, plan.Actions[i].Ref, before[i].Ref)
		}
		if len(plan.Actions[i].DependsOn) != len(before[i].DependsOn) {
			t.Errorf("RecordReview() changed dependencies for action %q", before[i].Ref)
			continue
		}
		for j := range before[i].DependsOn {
			if plan.Actions[i].DependsOn[j] != before[i].DependsOn[j] {
				t.Errorf("RecordReview() reordered dependencies for action %q", before[i].Ref)
				break
			}
		}
	}
}

func TestRecordReviewRejectsAmbiguousPlan(t *testing.T) {
	tests := map[string]func(*Plan){
		"malformed digest":             func(p *Plan) { p.SourceSHA256 = "not-a-digest" },
		"uppercase digest":             func(p *Plan) { p.SourceSHA256 = strings.ToUpper(p.SourceSHA256) },
		"short valid hex digest":       func(p *Plan) { p.SourceSHA256 = p.SourceSHA256[:62] },
		"no actions":                   func(p *Plan) { p.Actions = nil },
		"blank action ref":             func(p *Plan) { p.Actions[0].Ref = " " },
		"blank action kind":            func(p *Plan) { p.Actions[0].Kind = "\t" },
		"blank action object ref":      func(p *Plan) { p.Actions[0].ObjectRef = " " },
		"duplicate action ref":         func(p *Plan) { p.Actions[1].Ref = p.Actions[0].Ref },
		"unknown dependency":           func(p *Plan) { p.Actions[1].DependsOn = []string{"missing"} },
		"self dependency":              func(p *Plan) { p.Actions[0].DependsOn = []string{"step-1"} },
		"duplicate dependency":         func(p *Plan) { p.Actions[1].DependsOn = []string{"step-1", "step-1"} },
		"dependency cycle":             func(p *Plan) { p.Actions[0].DependsOn = []string{"step-2"} },
		"missing destination revision": func(p *Plan) { p.InventoryRevision = " " },
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			plan := samplePlan()
			mutate(&plan)
			if _, err := RecordReview(plan, "operator-opaque-1", "policy-3"); !errors.Is(err, ErrInvalidPlan) {
				t.Fatalf("RecordReview() error = %v, want ErrInvalidPlan", err)
			}
		})
	}
}

func TestRevalidateRejectsInconsistentReviewSnapshot(t *testing.T) {
	plan := samplePlan()
	review, err := RecordReview(plan, "operator-opaque-1", "policy-3")
	if err != nil {
		t.Fatal(err)
	}
	tests := map[string]struct {
		mutate func(*Review)
		want   error
	}{
		"migration id":       {mutate: func(r *Review) { r.MigrationID = "other-migration" }, want: ErrPlanChanged},
		"plan digest":        {mutate: func(r *Review) { r.PlanSHA256 = "different" }, want: ErrPlanChanged},
		"action set digest":  {mutate: func(r *Review) { r.ActionSetSHA256 = "different" }, want: ErrPlanChanged},
		"tenant":             {mutate: func(r *Review) { r.TenantID = "other-tenant" }, want: ErrPlanChanged},
		"destination":        {mutate: func(r *Review) { r.DestinationID = "other-host" }, want: ErrPlanChanged},
		"inventory revision": {mutate: func(r *Review) { r.InventoryRevision = "revision-8" }, want: ErrPlanChanged},
		"policy revision":    {mutate: func(r *Review) { r.PolicyRevision = "policy-4" }, want: ErrPolicyChanged},
		"principal":          {mutate: func(r *Review) { r.PrincipalID = "other-operator" }, want: ErrAuthorityChanged},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			inconsistent := review
			test.mutate(&inconsistent)
			if err := Revalidate(plan, inconsistent, currentFor(plan, review)); !errors.Is(err, test.want) {
				t.Fatalf("Revalidate() error = %v, want %v", err, test.want)
			}
		})
	}
}

func TestRecordReviewRejectsBlankScope(t *testing.T) {
	plan := samplePlan()
	if _, err := RecordReview(plan, " ", "policy-3"); !errors.Is(err, ErrInvalidPlan) {
		t.Fatalf("RecordReview() with blank principal error = %v, want ErrInvalidPlan", err)
	}
	if _, err := RecordReview(plan, "operator-opaque-1", "\t"); !errors.Is(err, ErrInvalidPlan) {
		t.Fatalf("RecordReview() with blank policy revision error = %v, want ErrInvalidPlan", err)
	}
}

func TestRecordReviewRejectsMissingRequiredPlanFields(t *testing.T) {
	tests := map[string]func(*Plan){
		"migration id":       func(p *Plan) { p.ID = " " },
		"tenant id":          func(p *Plan) { p.TenantID = "\t" },
		"parser version":     func(p *Plan) { p.ParserVersion = " " },
		"destination id":     func(p *Plan) { p.DestinationID = "\t" },
		"inventory revision": func(p *Plan) { p.InventoryRevision = " " },
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			plan := samplePlan()
			mutate(&plan)
			if _, err := RecordReview(plan, "operator-opaque-1", "policy-3"); !errors.Is(err, ErrInvalidPlan) {
				t.Fatalf("RecordReview() error = %v, want ErrInvalidPlan", err)
			}
		})
	}
}

func TestRevalidateRejectsIncompleteAuthorizationResult(t *testing.T) {
	plan := samplePlan()
	review, err := RecordReview(plan, "operator-opaque-1", "policy-3")
	if err != nil {
		t.Fatal(err)
	}
	tests := map[string]struct {
		mutate func(*ExecutionContext)
		want   error
	}{
		"missing principal": {
			mutate: func(c *ExecutionContext) { c.Authorization.PrincipalID = " " },
			want:   ErrAuthorityChanged,
		},
		"missing tenant binding": {
			mutate: func(c *ExecutionContext) { c.Authorization.TenantID = " " },
			want:   ErrAuthorityChanged,
		},
		"missing destination binding": {
			mutate: func(c *ExecutionContext) { c.Authorization.DestinationID = " " },
			want:   ErrAuthorityChanged,
		},
		"missing action binding": {
			mutate: func(c *ExecutionContext) { c.Authorization.ActionSetSHA256 = "" },
			want:   ErrAuthorityChanged,
		},
		"missing policy revision": {
			mutate: func(c *ExecutionContext) { c.Authorization.PolicyRevision = "" },
			want:   ErrPolicyChanged,
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			current := currentFor(plan, review)
			test.mutate(&current)
			if err := Revalidate(plan, review, current); !errors.Is(err, test.want) {
				t.Fatalf("Revalidate() error = %v, want %v", err, test.want)
			}
		})
	}
}
