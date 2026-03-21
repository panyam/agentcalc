package envelope

import "time"

// Budget defines resource limits for a primitive execution tree.
type Budget struct {
	// MaxDepth is the maximum call depth allowed.
	MaxDepth int

	// CostCeiling is the maximum cumulative cost allowed.
	CostCeiling float64

	// TimeBudget is the maximum wall-clock time allowed.
	TimeBudget time.Duration
}

// TimeRemaining returns the remaining time budget given current spend in the envelope.
func (b Budget) TimeRemaining(env Envelope) time.Duration {
	remaining := b.TimeBudget - env.TimeSpent
	if remaining < 0 {
		return 0
	}
	return remaining
}

// CostRemaining returns the remaining cost budget given current spend.
func (b Budget) CostRemaining(env Envelope) float64 {
	remaining := b.CostCeiling - env.CostSpent
	if remaining < 0 {
		return 0
	}
	return remaining
}
