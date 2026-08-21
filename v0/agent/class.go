// Package agent defines AgentClass (the static specification) and AgentInstance
// (the runtime goroutine) for agent primitives in the calculus.
package agent

import (
	"github.com/panyam/agentcalc/governor"
	"github.com/panyam/agentcalc/memory"
	"github.com/panyam/agentcalc/primitive"
	"github.com/panyam/agentcalc/types"
)

// AgentClass is the static specification of an agent — its capabilities,
// constraints, and defaults. Multiple AgentInstances can share one AgentClass.
type AgentClass struct {
	// ID uniquely identifies this agent class.
	ID primitive.PrimitiveID

	// ToolSignatures lists the primitives this agent can invoke as tools.
	ToolSignatures []primitive.PrimitiveID

	// MemoryConfig specifies the memory levels and configuration.
	MemoryConfig memory.MemoryConfig

	// Evaluators are primitives used to assess output quality.
	Evaluators []primitive.PrimitiveID

	// GovernorDefaults provides default budget and degradation settings.
	GovernorDefaults governor.ResourceGovernor

	// SystemPromptTemplate is the base prompt template for this agent class.
	SystemPromptTemplate string

	// CrystallizedRules are JIT-compiled behavioral rules (from doc 10).
	CrystallizedRules []string

	// InterruptPolicy determines how instances of this class handle interrupts.
	InterruptPolicy primitive.InterruptPolicy

	// TypeLevel sets the type checking strictness for this agent's I/O.
	TypeLevel types.TypeLevel
}
