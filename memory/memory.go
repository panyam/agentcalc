// Package memory defines the multi-level memory hierarchy (L1-L5) and the
// MemoryStore interface that all memory backends implement.
package memory

import (
	"context"

	"github.com/panyam/chakra/primitive"
)

// MemoryLevel represents a tier in the memory hierarchy.
type MemoryLevel int

const (
	// L1ContextWindow — in-memory working context for the current session.
	L1ContextWindow MemoryLevel = iota + 1
	// L2EpisodicLog — recent event history.
	L2EpisodicLog
	// L3ClassMemory — shared knowledge across instances of a class.
	L3ClassMemory
	// L4CrossAgentMemory — knowledge shared across agent classes.
	L4CrossAgentMemory
	// L5ArchivalMemory — long-term, persistent storage.
	L5ArchivalMemory
)

// MemoryConfig specifies the memory levels and parameters for an agent.
type MemoryConfig struct {
	Levels         []MemoryLevel
	MaxContextSize int
	EpisodicLimit  int
	ConsolidateAt  int // trigger consolidation after this many events
}

// MemoryStore is the interface for any memory backend. Memory stores are
// themselves primitives in the calculus.
type MemoryStore interface {
	primitive.Primitive

	// Get retrieves a value by key from the specified level.
	Get(ctx context.Context, level MemoryLevel, key string) (any, error)

	// Put stores a value at the specified level.
	Put(ctx context.Context, level MemoryLevel, key string, value any) error

	// Query performs a similarity/relevance search at the specified level.
	Query(ctx context.Context, level MemoryLevel, query string, limit int) ([]RetrievalResult, error)
}
