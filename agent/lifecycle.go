package agent

import (
	"context"

	"github.com/panyam/chakra/primitive"
)

// Checkpoint captures the agent's full state for later resumption.
func (a *AgentInstance) Checkpoint() *primitive.State {
	panic("not implemented")
}

// Resume restores an agent instance from a previously captured state.
func (a *AgentInstance) Resume(state *primitive.State) {
	panic("not implemented")
}

// Terminate gracefully shuts down the agent instance.
func (a *AgentInstance) Terminate(ctx context.Context) error {
	panic("not implemented")
}
