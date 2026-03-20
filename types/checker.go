package types

import "github.com/panyam/chakra/primitive"

// TypeChecker validates compatibility between primitives for hot-swapping.
type TypeChecker interface {
	// StructurallyCompatible checks if two types have compatible I/O schemas.
	StructurallyCompatible(old, new DualType) TypeCompatibility

	// SemanticCompatible checks if two types are semantically interchangeable.
	SemanticCompatible(old, new DualType) TypeCompatibility

	// Infer derives a DualType from a primitive's metadata and interface.
	Infer(p primitive.Primitive) DualType
}
