// Package registry provides PrimitiveRegistry — the runtime catalog of all
// registered primitives with type-checked hot-swapping and bedrock protection.
package registry

import (
	"sync"

	"github.com/panyam/chakra/primitive"
	"github.com/panyam/chakra/types"
)

// RegisterOption is a functional option for configuring registration behavior.
type RegisterOption func(*registerConfig)

type registerConfig struct {
	bedrock bool
	dual    *types.DualType
}

// WithBedrock marks the primitive as immutable — it cannot be swapped.
func WithBedrock() RegisterOption {
	return func(c *registerConfig) {
		c.bedrock = true
	}
}

// WithDualType associates a DualType with the registered primitive.
func WithDualType(dt types.DualType) RegisterOption {
	return func(c *registerConfig) {
		c.dual = &dt
	}
}

// PrimitiveRegistry is the runtime catalog of all primitives.
type PrimitiveRegistry struct {
	mu         sync.RWMutex
	primitives map[primitive.PrimitiveID]primitive.Primitive
	types      map[primitive.PrimitiveID]types.DualType
	versions   map[primitive.PrimitiveID][]primitive.Primitive
	bedrock    map[primitive.PrimitiveID]bool
}

// Register adds a primitive to the registry.
func (r *PrimitiveRegistry) Register(p primitive.Primitive, opts ...RegisterOption) error {
	panic("not implemented")
}

// Get retrieves a primitive by ID.
func (r *PrimitiveRegistry) Get(id primitive.PrimitiveID) (primitive.Primitive, error) {
	panic("not implemented")
}

// Swap replaces a primitive with a type-checked replacement.
func (r *PrimitiveRegistry) Swap(oldID primitive.PrimitiveID, newP primitive.Primitive, checker types.TypeChecker) error {
	panic("not implemented")
}

// Versions returns the version history for a primitive.
func (r *PrimitiveRegistry) Versions(id primitive.PrimitiveID) []primitive.Primitive {
	panic("not implemented")
}

// MarkBedrock marks an already-registered primitive as immutable.
func (r *PrimitiveRegistry) MarkBedrock(id primitive.PrimitiveID) error {
	panic("not implemented")
}
