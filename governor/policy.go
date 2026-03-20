package governor

// DegradationPolicy determines what happens when a resource limit is exceeded.
type DegradationPolicy int

const (
	// HALT — stop execution and return an error.
	HALT DegradationPolicy = iota
	// SUMMARIZE — produce a summarized/reduced-quality output.
	SUMMARIZE
	// ESCALATE — escalate to a human or higher authority.
	ESCALATE
	// FALLBACK — delegate to the fallback primitive.
	FALLBACK
)
