// Package primitive defines the core Primitive interface — the single algebraic
// type from which agents, tools, memory, evaluators, type checkers, and governors
// are all derived.
package primitive

import (
	"context"
	"time"
)

// PrimitiveID is a unique identifier for any primitive in the system.
type PrimitiveID string

// InterruptPolicy controls how a primitive responds to preemption signals.
type InterruptPolicy int

const (
	// InterruptNever — primitive cannot be interrupted.
	InterruptNever InterruptPolicy = iota
	// InterruptCheckpoint — interrupt after checkpointing current state.
	InterruptCheckpoint
	// InterruptImmediate — interrupt immediately, best-effort state save.
	InterruptImmediate
)

// PrimitiveMetadata carries factual and contextual attributes of a primitive.
type PrimitiveMetadata struct {
	BaseCost    float64
	BaseLatency time.Duration
	Reversible  bool
	Preemptible bool
	Criticality float64 // 0.0–1.0
	TrustLevel  float64 // 0.0–1.0
	TypeLevel   int     // 0-3
}

// Primitive is the universal interface. Every component in the agent calculus
// — agents, tools, memory stores, evaluators, governors — implements Primitive.
type Primitive interface {
	// ID returns the unique identifier for this primitive.
	ID() PrimitiveID

	// Metadata returns the factual and contextual attributes.
	Metadata() PrimitiveMetadata

	// Execute runs the primitive's core logic.
	Execute(ctx context.Context, input Input, env Envelope) (Output, error)

	// Interrupt requests the primitive to stop, governed by the given policy.
	Interrupt(policy InterruptPolicy) (State, error)

	// Rollback restores the primitive to a previously checkpointed state.
	Rollback(state State) error
}

// Envelope is a forward-declared type alias so the primitive package can
// reference it in signatures. The concrete definition lives in the envelope package.
// Implementations will use envelope.Envelope directly.
type Envelope = any
