// Package review is an offline design prototype for P4 migration review binding.
// It performs no authorization, archive processing, filesystem access, or effects.
package review

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
)

var (
	ErrInvalidPlan      = errors.New("invalid migration plan")
	ErrPlanChanged      = errors.New("migration plan differs from reviewed plan")
	ErrAuthorityChanged = errors.New("execution authority differs from reviewed scope")
	ErrInventoryChanged = errors.New("destination inventory changed after review")
	ErrUnauthorized     = errors.New("current authorization check denied the reviewed action set")
	ErrPolicyChanged    = errors.New("policy revision changed after review")
)

// Plan contains opaque references and safe-to-share metadata only. Secret values,
// host paths, archive bytes, and executable commands are intentionally absent.
type Plan struct {
	ID                string   `json:"id"`
	TenantID          string   `json:"tenant_id"`
	SourceSHA256      string   `json:"source_sha256"`
	ParserVersion     string   `json:"parser_version"`
	DestinationID     string   `json:"destination_id"`
	InventoryRevision string   `json:"inventory_revision"`
	Actions           []Action `json:"actions"`
}

type Action struct {
	Ref              string   `json:"ref"`
	Kind             string   `json:"kind"`
	ObjectRef        string   `json:"object_ref"`
	ExpectedRevision string   `json:"expected_revision"`
	DependsOn        []string `json:"depends_on,omitempty"`
}

// Review is a data-only receipt for comparing a later request with what was
// reviewed. It is not a signed token, approval credential, or authorization proof.
type Review struct {
	MigrationID       string `json:"migration_id"`
	PlanSHA256        string `json:"plan_sha256"`
	ActionSetSHA256   string `json:"action_set_sha256"`
	TenantID          string `json:"tenant_id"`
	DestinationID     string `json:"destination_id"`
	InventoryRevision string `json:"inventory_revision"`
	PolicyRevision    string `json:"policy_revision"`
	PrincipalID       string `json:"principal_id"`
}

// ExecutionContext is supplied by a hypothetical caller after separately
// authenticating and authorizing that caller against current policy.
type ExecutionContext struct {
	TenantID          string
	DestinationID     string
	InventoryRevision string
	PolicyRevision    string
	PrincipalID       string
	Authorization     AuthorizationResult
}

// AuthorizationResult models a separate decision made by a current policy
// checker. This prototype only compares the result; it does not produce it.
type AuthorizationResult struct {
	TenantID        string
	DestinationID   string
	PrincipalID     string
	ActionSetSHA256 string
	PolicyRevision  string
	Allowed         bool
}

// RecordReview snapshots the exact plan and review scope for offline comparison.
// Calling it does not establish that the principal was authenticated or authorized.
// inventoryRevision remains one illustrative opaque scalar. Its scope and
// consistency semantics are not defined or approved by this prototype.
func RecordReview(plan Plan, principalID, policyRevision string) (Review, error) {
	if err := validatePlan(plan); err != nil {
		return Review{}, err
	}
	if strings.TrimSpace(principalID) == "" {
		return Review{}, fmt.Errorf("%w: principal is required", ErrInvalidPlan)
	}
	if strings.TrimSpace(policyRevision) == "" {
		return Review{}, fmt.Errorf("%w: policy revision is required", ErrInvalidPlan)
	}
	canonical := canonicalPlan(plan)
	planHash, err := digest(canonical)
	if err != nil {
		return Review{}, err
	}
	actionHash, err := digest(canonical.Actions)
	if err != nil {
		return Review{}, err
	}
	return Review{
		MigrationID:       plan.ID,
		PlanSHA256:        planHash,
		ActionSetSHA256:   actionHash,
		TenantID:          plan.TenantID,
		DestinationID:     plan.DestinationID,
		InventoryRevision: plan.InventoryRevision,
		PolicyRevision:    policyRevision,
		PrincipalID:       principalID,
	}, nil
}

// Revalidate compares a later operation with the reviewed plan and scope.
// The caller must perform real authentication/authorization and destination
// inventory reads independently; this function has no external effects.
func Revalidate(plan Plan, review Review, current ExecutionContext) error {
	if err := validatePlan(plan); err != nil {
		return err
	}
	canonical := canonicalPlan(plan)
	planHash, err := digest(canonical)
	if err != nil {
		return err
	}
	actionHash, err := digest(canonical.Actions)
	if err != nil {
		return err
	}
	if plan.ID != review.MigrationID || planHash != review.PlanSHA256 || actionHash != review.ActionSetSHA256 ||
		plan.TenantID != review.TenantID || plan.DestinationID != review.DestinationID ||
		plan.InventoryRevision != review.InventoryRevision {
		return ErrPlanChanged
	}
	if current.TenantID != review.TenantID || current.DestinationID != review.DestinationID ||
		current.PrincipalID != review.PrincipalID {
		return ErrAuthorityChanged
	}
	if current.PolicyRevision != review.PolicyRevision ||
		current.Authorization.PolicyRevision != review.PolicyRevision {
		return ErrPolicyChanged
	}
	if current.Authorization.TenantID != review.TenantID ||
		current.Authorization.DestinationID != review.DestinationID ||
		current.Authorization.PrincipalID != review.PrincipalID ||
		current.Authorization.ActionSetSHA256 != review.ActionSetSHA256 {
		return ErrAuthorityChanged
	}
	if !current.Authorization.Allowed {
		return ErrUnauthorized
	}
	if current.InventoryRevision != review.InventoryRevision {
		return ErrInventoryChanged
	}
	return nil
}

func validatePlan(plan Plan) error {
	if strings.TrimSpace(plan.ID) == "" || strings.TrimSpace(plan.TenantID) == "" ||
		strings.TrimSpace(plan.ParserVersion) == "" || strings.TrimSpace(plan.DestinationID) == "" ||
		strings.TrimSpace(plan.InventoryRevision) == "" {
		return fmt.Errorf("%w: required plan field is empty", ErrInvalidPlan)
	}
	digest, err := hex.DecodeString(plan.SourceSHA256)
	if err != nil || len(digest) != sha256.Size || strings.ToLower(plan.SourceSHA256) != plan.SourceSHA256 {
		return fmt.Errorf("%w: source digest must be 64 lowercase hexadecimal characters", ErrInvalidPlan)
	}
	if len(plan.Actions) == 0 {
		return fmt.Errorf("%w: at least one action is required", ErrInvalidPlan)
	}
	refs := make(map[string]struct{}, len(plan.Actions))
	graph := make(map[string][]string, len(plan.Actions))
	indegree := make(map[string]int, len(plan.Actions))
	for _, action := range plan.Actions {
		if strings.TrimSpace(action.Ref) == "" || strings.TrimSpace(action.Kind) == "" ||
			strings.TrimSpace(action.ObjectRef) == "" {
			return fmt.Errorf("%w: each action needs a ref, kind, and opaque object ref", ErrInvalidPlan)
		}
		if _, exists := refs[action.Ref]; exists {
			return fmt.Errorf("%w: duplicate action ref %q", ErrInvalidPlan, action.Ref)
		}
		refs[action.Ref] = struct{}{}
		indegree[action.Ref] = 0
	}
	for _, action := range plan.Actions {
		seenDependencies := make(map[string]struct{}, len(action.DependsOn))
		for _, dependency := range action.DependsOn {
			if _, exists := refs[dependency]; !exists {
				return fmt.Errorf("%w: action %q has unknown dependency %q", ErrInvalidPlan, action.Ref, dependency)
			}
			if dependency == action.Ref {
				return fmt.Errorf("%w: action %q depends on itself", ErrInvalidPlan, action.Ref)
			}
			if _, exists := seenDependencies[dependency]; exists {
				return fmt.Errorf("%w: action %q repeats dependency %q", ErrInvalidPlan, action.Ref, dependency)
			}
			seenDependencies[dependency] = struct{}{}
			graph[dependency] = append(graph[dependency], action.Ref)
			indegree[action.Ref]++
		}
	}
	ready := make([]string, 0, len(plan.Actions))
	for ref, degree := range indegree {
		if degree == 0 {
			ready = append(ready, ref)
		}
	}
	visited := 0
	for len(ready) > 0 {
		ref := ready[len(ready)-1]
		ready = ready[:len(ready)-1]
		visited++
		for _, dependent := range graph[ref] {
			indegree[dependent]--
			if indegree[dependent] == 0 {
				ready = append(ready, dependent)
			}
		}
	}
	if visited != len(plan.Actions) {
		return fmt.Errorf("%w: action dependencies contain a cycle", ErrInvalidPlan)
	}
	return nil
}

// canonicalPlan sorts a copy so input slice order does not imply execution
// order. DependsOn expresses required ordering; independent actions may run in
// any topological order. Strings are ordered bytewise by Go's string ordering.
func canonicalPlan(plan Plan) Plan {
	canonical := plan
	canonical.Actions = make([]Action, len(plan.Actions))
	for i, action := range plan.Actions {
		canonical.Actions[i] = action
		canonical.Actions[i].DependsOn = append([]string(nil), action.DependsOn...)
		sort.Strings(canonical.Actions[i].DependsOn)
	}
	sort.Slice(canonical.Actions, func(i, j int) bool {
		return canonical.Actions[i].Ref < canonical.Actions[j].Ref
	})
	return canonical
}

func digest(value any) (string, error) {
	encoded, err := json.Marshal(value)
	if err != nil {
		return "", fmt.Errorf("encode review value: %w", err)
	}
	sum := sha256.Sum256(encoded)
	return hex.EncodeToString(sum[:]), nil
}
