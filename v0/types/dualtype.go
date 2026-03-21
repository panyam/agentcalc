// Package types defines the dual type system — structural (JSON Schema) and
// semantic (natural language description) — used for type-checked hot-swapping.
package types

import "encoding/json"

// TypeLevel indicates the strictness of type checking applied.
type TypeLevel int

const (
	// TypeLevelNone — no type checking.
	TypeLevelNone TypeLevel = iota
	// TypeLevelStructural — structural compatibility only.
	TypeLevelStructural
	// TypeLevelSemantic — semantic compatibility only.
	TypeLevelSemantic
	// TypeLevelFull — both structural and semantic checks.
	TypeLevelFull
)

// StructuralType is a JSON Schema describing the shape of a primitive's I/O.
type StructuralType struct {
	Schema json.RawMessage
}

// SemanticType is a natural language description of what a primitive does,
// used for semantic compatibility checking.
type SemanticType struct {
	Description string
}

// DualType pairs structural and semantic type information for a primitive.
type DualType struct {
	Structural StructuralType
	Semantic   SemanticType
	Level      TypeLevel
}

// TypeCompatibility is the result of a compatibility check between two DualTypes.
type TypeCompatibility struct {
	Compatible bool
	Confidence float64 // 0.0–1.0
	Reason     string
}
