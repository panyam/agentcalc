// Example: Critic/Verifier agent built on chakra.
//
// Pattern: generate → critique → revise, until the critic passes.
// Both generator and critic are primitives. The verifier agent
// orchestrates them through the registry.
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/panyam/agentcalc"
)

// --- The critic primitive ---

type Critic struct {
	model LLM
}

func (c *Critic) Meta() chakra.Meta {
	return chakra.Meta{ID: "critic", Description: "reviews code for correctness and style"}
}

func (c *Critic) Run(ctx context.Context, in chakra.Message, env chakra.Envelope) (chakra.Message, error) {
	code := in.Payload.(string)
	resp, err := c.model.Chat(ctx, []Turn{
		{Role: "system", Content: "Review this code. Respond with JSON: {\"pass\": bool, \"issues\": [string]}"},
		{Role: "user", Content: code},
	})
	if err != nil {
		return chakra.Message{}, err
	}
	return chakra.Message{Payload: resp.Parsed, Schema: "critic-review"}, nil
}

// --- The generator primitive ---

type Generator struct {
	model LLM
}

func (g *Generator) Meta() chakra.Meta {
	return chakra.Meta{ID: "generator", Description: "generates or revises code based on feedback"}
}

func (g *Generator) Run(ctx context.Context, in chakra.Message, env chakra.Envelope) (chakra.Message, error) {
	req := in.Payload.(GenerateRequest)
	history := []Turn{
		{Role: "system", Content: "You are a code generator. Write clean, correct code."},
		{Role: "user", Content: req.Task},
	}
	if req.Feedback != "" {
		history = append(history,
			Turn{Role: "assistant", Content: req.PreviousCode},
			Turn{Role: "user", Content: "Feedback: " + req.Feedback + "\nPlease revise."},
		)
	}
	resp, err := g.model.Chat(ctx, history)
	if err != nil {
		return chakra.Message{}, err
	}
	return chakra.Message{Payload: resp.Answer, Schema: "code"}, nil
}

type GenerateRequest struct {
	Task         string
	PreviousCode string
	Feedback     string
}

// --- The verifier agent: the orchestrating for loop ---

type VerifierAgent struct {
	registry *chakra.Registry
	maxRounds int
}

func (v *VerifierAgent) Meta() chakra.Meta {
	return chakra.Meta{ID: "verifier-agent", Description: "generate-critique loop until critic passes"}
}

func (v *VerifierAgent) Run(ctx context.Context, in chakra.Message, env chakra.Envelope) (chakra.Message, error) {
	task := in.Payload.(string)
	var code string
	var feedback string

	for round := 0; round < v.maxRounds; round++ {
		select {
		case <-env.Done:
			return chakra.Message{}, chakra.ErrCancelled
		default:
		}

		// --- generate (or revise) ---
		genMsg := chakra.Message{
			Payload: GenerateRequest{Task: task, PreviousCode: code, Feedback: feedback},
			Schema:  "generate-request",
		}
		genResult, err := v.registry.Invoke(ctx, "verifier-agent", "generator", genMsg, env)
		if err != nil {
			return chakra.Message{}, fmt.Errorf("generator failed: %w", err)
		}
		code = genResult.Payload.(string)

		// --- critique ---
		critiqueResult, err := v.registry.Invoke(ctx, "verifier-agent", "critic",
			chakra.Message{Payload: code, Schema: "code"}, env)
		if err != nil {
			return chakra.Message{}, fmt.Errorf("critic failed: %w", err)
		}

		review := critiqueResult.Payload.(CriticReview)
		if review.Pass {
			return chakra.Message{Payload: code, Schema: "code"}, nil
		}

		// feed issues back for next round
		feedback = ""
		for _, issue := range review.Issues {
			feedback += "- " + issue + "\n"
		}

		env.Budget.IterSoFar++
	}

	// max rounds — return best effort
	return chakra.Message{Payload: code, Schema: "code"}, nil
}

type CriticReview struct {
	Pass   bool
	Issues []string
}

// --- Wiring ---

func main() {
	reg := chakra.NewRegistry()
	store := chakra.NewMemStore()
	model := newLLM()

	reg.Register(&Generator{model: model})
	reg.Register(&Critic{model: model})

	agent := &VerifierAgent{registry: reg, maxRounds: 5}
	reg.Register(agent)

	// verifier-agent can call generator and critic
	reg.Connect("verifier-agent", "generator")
	reg.Connect("verifier-agent", "critic")

	cancel := make(chan struct{})
	env := chakra.Envelope{
		Budget: chakra.Budget{MaxIter: 10, MaxCost: 10.0, MaxDepth: 5},
		Done:   cancel,
		Store:  store,
	}

	result, err := agent.Run(context.Background(), chakra.Message{
		Payload: "Write a Go function that merges two sorted slices",
		Schema:  "text",
	}, env)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(result.Payload)
}

// --- Stubs ---

type LLM interface {
	Chat(ctx context.Context, history []Turn) (LLMResponse, error)
}
type Turn struct{ Role, Content string }
type LLMResponse struct {
	Answer string
	Parsed any
	Cost   float64
}

func newLLM() LLM { panic("plug in your LLM client") }
