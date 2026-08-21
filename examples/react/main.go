// Example: ReAct agent built on chakra.
//
// The agent IS a for loop. Think, act, observe.
// It captures a Registry to invoke tools.
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/panyam/agentcalc"
)

// --- The agent itself ---

type ReActAgent struct {
	registry *chakra.Registry
	model    LLM // your LLM client — not a kernel concern
	maxIter  int
}

func (a *ReActAgent) Meta() chakra.Meta {
	return chakra.Meta{
		ID:          "react-agent",
		Description: "ReAct loop: think, act, observe",
		Reversible:  false,
	}
}

func (a *ReActAgent) Run(ctx context.Context, in chakra.Message, env chakra.Envelope) (chakra.Message, error) {
	prompt := in.Payload.(string)
	history := []Turn{{Role: "user", Content: prompt}}

	for i := 0; i < a.maxIter; i++ {
		// --- check cancellation + budget ---
		select {
		case <-env.Done:
			return chakra.Message{}, chakra.ErrCancelled
		default:
		}
		if env.Budget.Exhausted() {
			return chakra.Message{Payload: "budget exhausted, best effort: " + lastContent(history)}, nil
		}

		// --- think: ask the model what to do ---
		resp, err := a.model.Chat(ctx, history)
		if err != nil {
			return chakra.Message{}, err
		}

		// --- done? ---
		if resp.Done {
			return chakra.Message{Payload: resp.Answer, Schema: "text"}, nil
		}

		// --- act: invoke the tool through the registry ---
		toolMsg := chakra.Message{Payload: resp.ToolArgs, Schema: resp.ToolName}
		result, err := a.registry.Invoke(ctx, "react-agent", resp.ToolName, toolMsg, env)
		if err != nil {
			// tool failed — feed the error back to the model, don't crash
			history = append(history,
				Turn{Role: "assistant", Content: fmt.Sprintf("Action: %s(%v)", resp.ToolName, resp.ToolArgs)},
				Turn{Role: "tool", Content: fmt.Sprintf("Error: %v", err)},
			)
			continue
		}

		// --- observe: feed result back into history ---
		history = append(history,
			Turn{Role: "assistant", Content: fmt.Sprintf("Action: %s(%v)", resp.ToolName, resp.ToolArgs)},
			Turn{Role: "tool", Content: fmt.Sprintf("%v", result.Payload)},
		)

		// track cost
		env.Budget.CostSoFar += resp.Cost
		env.Budget.IterSoFar++
	}

	return chakra.Message{Payload: "max iterations reached"}, nil
}

// --- A simple tool primitive ---

type GrepTool struct{}

func (t *GrepTool) Meta() chakra.Meta {
	return chakra.Meta{ID: "grep", Description: "search code for a pattern", Reversible: true}
}

func (t *GrepTool) Run(ctx context.Context, in chakra.Message, env chakra.Envelope) (chakra.Message, error) {
	pattern := in.Payload.(string)
	// ... actual grep implementation ...
	results := grep(pattern)
	return chakra.Message{Payload: results, Schema: "grep-results"}, nil
}

// --- Wiring it up ---

func main() {
	reg := chakra.NewRegistry()
	store := chakra.NewMemStore()

	// register tools
	reg.Register(&GrepTool{})
	reg.Register(&ReadFileTool{})
	reg.Register(&EditFileTool{})
	reg.Register(&RunTestsTool{})

	// register the agent
	agent := &ReActAgent{registry: reg, model: newLLM(), maxIter: 20}
	reg.Register(agent)

	// connect: agent can call any tool
	reg.Connect("react-agent", "grep")
	reg.Connect("react-agent", "read-file")
	reg.Connect("react-agent", "edit-file")
	reg.Connect("react-agent", "run-tests")

	// create envelope with budget
	cancel := make(chan struct{})
	env := chakra.Envelope{
		Budget: chakra.Budget{MaxIter: 20, MaxCost: 5.0, MaxDepth: 10},
		Done:   cancel,
		Store:  store,
	}

	// run
	result, err := agent.Run(context.Background(), chakra.Message{
		Payload: "Fix the failing test in auth/login_test.go",
		Schema:  "text",
	}, env)

	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(result.Payload)
}

// --- Stubs for things that aren't kernel concerns ---

type LLM interface {
	Chat(ctx context.Context, history []Turn) (LLMResponse, error)
}
type Turn struct {
	Role    string
	Content string
}
type LLMResponse struct {
	Done     bool
	Answer   string
	ToolName string
	ToolArgs any
	Cost     float64
}

func newLLM() LLM                    { panic("plug in your LLM client") }
func grep(pattern string) []string   { panic("plug in your grep") }
func lastContent(h []Turn) string    { return h[len(h)-1].Content }

type ReadFileTool struct{}
func (t *ReadFileTool) Meta() chakra.Meta { return chakra.Meta{ID: "read-file", Description: "read a file"} }
func (t *ReadFileTool) Run(ctx context.Context, in chakra.Message, env chakra.Envelope) (chakra.Message, error) { panic("impl") }

type EditFileTool struct{}
func (t *EditFileTool) Meta() chakra.Meta { return chakra.Meta{ID: "edit-file", Description: "edit a file", Reversible: true} }
func (t *EditFileTool) Run(ctx context.Context, in chakra.Message, env chakra.Envelope) (chakra.Message, error) { panic("impl") }

type RunTestsTool struct{}
func (t *RunTestsTool) Meta() chakra.Meta { return chakra.Meta{ID: "run-tests", Description: "run tests"} }
func (t *RunTestsTool) Run(ctx context.Context, in chakra.Message, env chakra.Envelope) (chakra.Message, error) { panic("impl") }
