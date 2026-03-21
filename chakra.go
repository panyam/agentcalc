// Package chakra implements the agent calculus kernel.
//
// Four things: Primitive, Envelope, Store, Gate.
// Everything else is user-space.
package chakra

import "context"

// Primitive is the universal type. Agents, tools, memory, evaluators —
// everything implements Primitive.
type Primitive interface {
	Meta() Meta
	Run(ctx context.Context, in Message, env Envelope) (Message, error)
}

// Meta is the minimal metadata the kernel needs. Three fields.
type Meta struct {
	ID          string
	Description string // NL description — the only "type" info at kernel level
	Reversible  bool   // the one safety flag the kernel needs
}

// Message is the universal input/output type.
type Message struct {
	Payload any
	Schema  string // a label, not enforced by kernel
}

// Envelope carries execution context through the call graph.
// Not immutable — the kernel mutates Depth and Trace during Invoke.
type Envelope struct {
	Depth  int
	Budget Budget
	Trace  []string       // ordered primitive IDs — the audit log
	Done   <-chan struct{} // cancellation — this IS the governor
	Store  Store          // shared state, opaque to kernel
}

// Budget limits execution. The kernel checks Exhausted() before
// every Invoke. Policy beyond that is user-space.
type Budget struct {
	MaxDepth   int
	MaxCost    float64
	CostSoFar  float64
	MaxIter    int
	IterSoFar  int
}

// Exhausted returns true if any budget limit has been reached.
func (b Budget) Exhausted() bool {
	if b.MaxDepth > 0 && b.MaxIter > 0 && b.IterSoFar >= b.MaxIter {
		return true
	}
	if b.MaxCost > 0 && b.CostSoFar >= b.MaxCost {
		return true
	}
	return false
}

// Gate is a validation function on an edge between two primitives.
// It sees the outgoing message, the next primitive, and the envelope.
// Return non-nil to block the invocation.
// Gates fire BEFORE invocation.
type Gate func(out Message, next Primitive, env Envelope) error

// DeltaGate fires AFTER a primitive returns, before its Store writes
// are merged to the parent. It sees the result, the delta (what the
// primitive read and wrote), and the parent envelope.
// Return non-nil to reject the delta — no writes merge.
type DeltaGate func(result Message, delta StoreDelta, env Envelope) error

// StoreDelta captures what a primitive did with the Store during
// its execution. Reads and Writes are recorded by ScopedStore.
type StoreDelta struct {
	Reads  []string       // keys the primitive read from the parent
	Writes map[string]any // keys the primitive wrote (uncommitted)
}
