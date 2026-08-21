package agent

import (
	"context"

	"github.com/panyam/agentcalc/envelope"
	"github.com/panyam/agentcalc/event"
	"github.com/panyam/agentcalc/memory"
	"github.com/panyam/agentcalc/primitive"
)

// Task represents a unit of work assigned to an agent instance.
type Task struct {
	ID          string
	Description string
	Input       primitive.Input
	Metadata    map[string]any
}

// AgentInstance is the runtime representation of an agent — each instance runs
// as its own goroutine. Communication with other agents happens only through
// the EventBus.
type AgentInstance struct {
	class      *AgentClass
	workingCtx memory.ContextWindow
	episodic   []event.Event
	hotSwapped map[primitive.PrimitiveID]primitive.Primitive
	envelope   envelope.Envelope
	checkpoint *primitive.State
	eventBus   *event.EventBus
}

// ID returns the instance's primitive ID.
func (a *AgentInstance) ID() primitive.PrimitiveID {
	panic("not implemented")
}

// Metadata returns the instance's metadata, derived from its class.
func (a *AgentInstance) Metadata() primitive.PrimitiveMetadata {
	panic("not implemented")
}

// Execute runs the agent on the given input within the envelope's constraints.
func (a *AgentInstance) Execute(ctx context.Context, input primitive.Input, env primitive.Envelope) (primitive.Output, error) {
	panic("not implemented")
}

// Interrupt requests the agent to stop per the given policy.
func (a *AgentInstance) Interrupt(policy primitive.InterruptPolicy) (primitive.State, error) {
	panic("not implemented")
}

// Rollback restores the agent to a previously checkpointed state.
func (a *AgentInstance) Rollback(state primitive.State) error {
	panic("not implemented")
}

// Run is the agent's main goroutine loop. It uses a two-phase select pattern:
// first drain the critical channel (to guarantee priority), then multi-select
// across all channels. Go's select does not guarantee priority ordering, so the
// two-phase pattern is necessary.
func (a *AgentInstance) Run(ctx context.Context, task Task) error {
	panic("not implemented")
}
