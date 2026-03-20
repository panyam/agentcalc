package primitive

import (
	"encoding/json"
	"time"
)

// State is a json-serializable checkpoint. Each primitive owns its checkpoint
// format — the opaque fields allow arbitrary serialization without coupling
// primitives to each other's internal representations.
type State struct {
	// PrimitiveID identifies which primitive produced this checkpoint.
	PrimitiveID PrimitiveID

	// Timestamp records when this checkpoint was created.
	Timestamp time.Time

	// Data holds the opaque, primitive-specific checkpoint payload.
	Data json.RawMessage

	// Version allows primitives to evolve their checkpoint format.
	Version int
}
