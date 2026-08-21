// Package envelope defines the Envelope — an immutable context threaded through
// every primitive call. With*() methods return copies, preventing child calls
// from corrupting parent state.
package envelope

import (
	"time"

	"github.com/panyam/agentcalc/primitive"
)

// Envelope is threaded through every Primitive.Execute call. It is immutable:
// all With*() methods return a new copy.
type Envelope struct {
	Depth     int
	CostSpent float64
	TimeSpent time.Duration
	CallTrace []primitive.PrimitiveID
	Budget    Budget
	TraceID   string
}

// WithDepth returns a copy of the Envelope with the given depth.
func (e Envelope) WithDepth(depth int) Envelope {
	e.Depth = depth
	e.CallTrace = append(append([]primitive.PrimitiveID{}, e.CallTrace...), "")
	return e
}

// WithCostSpent returns a copy with updated cost.
func (e Envelope) WithCostSpent(cost float64) Envelope {
	e.CostSpent = cost
	return e
}

// WithTimeSpent returns a copy with updated time spent.
func (e Envelope) WithTimeSpent(d time.Duration) Envelope {
	e.TimeSpent = d
	return e
}

// WithTrace returns a copy with the given primitive appended to the call trace.
func (e Envelope) WithTrace(id primitive.PrimitiveID) Envelope {
	trace := make([]primitive.PrimitiveID, len(e.CallTrace)+1)
	copy(trace, e.CallTrace)
	trace[len(e.CallTrace)] = id
	e.CallTrace = trace
	return e
}

// WithBudget returns a copy with the given budget.
func (e Envelope) WithBudget(b Budget) Envelope {
	e.Budget = b
	return e
}
