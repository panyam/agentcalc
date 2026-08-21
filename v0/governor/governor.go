// Package governor implements the ResourceGovernor — a Primitive wrapper that
// enforces budget constraints and applies degradation policies when limits are hit.
package governor

import (
	"context"

	"github.com/panyam/agentcalc/envelope"
	"github.com/panyam/agentcalc/primitive"
)

// ResourceGovernor wraps a target Primitive with budget enforcement and
// degradation behavior. It implements Primitive itself.
type ResourceGovernor struct {
	target   primitive.Primitive
	budget   envelope.Budget
	policy   DegradationPolicy
	fallback primitive.Primitive // used when policy is FALLBACK
	scope    TransactionScope
}

// TransactionScope defines the atomicity boundary for governor-managed operations.
type TransactionScope struct {
	Atomic  bool
	GroupID string
}

// ID returns the governor's primitive ID.
func (g *ResourceGovernor) ID() primitive.PrimitiveID {
	panic("not implemented")
}

// Metadata returns the governor's metadata.
func (g *ResourceGovernor) Metadata() primitive.PrimitiveMetadata {
	panic("not implemented")
}

// Execute runs the target primitive within budget constraints. If a limit is
// exceeded, the configured DegradationPolicy determines behavior.
func (g *ResourceGovernor) Execute(ctx context.Context, input primitive.Input, env primitive.Envelope) (primitive.Output, error) {
	panic("not implemented")
}

// Interrupt requests the governor (and its target) to stop.
func (g *ResourceGovernor) Interrupt(policy primitive.InterruptPolicy) (primitive.State, error) {
	panic("not implemented")
}

// Rollback restores the governor's target to a previous state.
func (g *ResourceGovernor) Rollback(state primitive.State) error {
	panic("not implemented")
}
