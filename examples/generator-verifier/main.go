// Example: Generator-Verifier built on chakra.
//
// Two patterns the mid-2026 coding-agent survey singled out
// (see docs/research/coding-agents-sota.md and docs/v1/06-open-design-questions.md):
//
//  1. Best-of-N with a verifier reranker (the SWE-Gym inference-time
//     pattern): generate N candidates, score each with a verifier
//     primitive, keep the best. Verification is where agents fail, so
//     the verifier is a first-class primitive, not an afterthought.
//
//  2. The verifier as a trust boundary. Generated code is untrusted
//     content (it could be a wrong or malicious patch). A DeltaGate on
//     the commit edge is what actually enforces that only a
//     verifier-approved candidate reaches the trusted: namespace. Even
//     if the agent loop above has a bug and tries to commit a
//     low-scoring candidate, the DeltaGate blocks the merge.
//
// The critic example does iterative generate -> critique -> revise.
// This one does parallel-in-spirit generate-N -> score -> rerank ->
// gated-commit, and it exercises the DeltaGate that critic never touches.
package main

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/panyam/agentcalc"
)

// --- Generator: produces one candidate solution ---

type Generator struct{ model LLM }

func (g *Generator) Meta() chakra.Meta {
	return chakra.Meta{ID: "generator", Description: "generates a candidate patch (LLM, ~$0.02/call, nondeterministic)"}
}

func (g *Generator) Run(ctx context.Context, in chakra.Message, env chakra.Envelope) (chakra.Message, error) {
	task := in.Payload.(string)
	resp, err := g.model.Chat(ctx, []Turn{
		{Role: "system", Content: "Write a patch that solves the task. Vary your approach across attempts."},
		{Role: "user", Content: task},
	})
	if err != nil {
		return chakra.Message{}, err
	}
	return chakra.Message{Payload: resp.Answer, Schema: "candidate-patch"}, nil
}

// --- Verifier: scores a candidate. The SWE-Gym reranker. ---
//
// In a real system this is often a *separately trained* model (SWE-Gym
// trains an agent and a verifier together and uses the verifier to
// rerank candidates at inference time). Here it just runs tests and
// returns a score in [0,1]. It is deterministic and cheap, and its
// Meta.Description says so — the convention that steers a composing
// agent toward it over another LLM call.
type Verifier struct{}

func (v *Verifier) Meta() chakra.Meta {
	return chakra.Meta{ID: "verifier", Description: "scores a candidate patch by running tests (deterministic, cheap)"}
}

func (v *Verifier) Run(ctx context.Context, in chakra.Message, env chakra.Envelope) (chakra.Message, error) {
	patch := in.Payload.(string)
	passed, total := runTests(patch) // your test harness
	score := 0.0
	if total > 0 {
		score = float64(passed) / float64(total)
	}
	return chakra.Message{Payload: Verdict{Score: score, Passed: passed, Total: total}, Schema: "verdict"}, nil
}

type Verdict struct {
	Score         float64
	Passed, Total int
}

// --- Committer: writes the chosen patch to the trusted namespace ---
//
// It writes into its (scoped) store. That write does NOT reach the
// parent store until the DeltaGate on the commit edge approves it.
// The committer reports the verifier verdict as its result so the gate
// can adjudicate without re-running anything.
type Committer struct{}

func (c *Committer) Meta() chakra.Meta {
	return chakra.Meta{ID: "committer", Description: "commits an approved patch to trusted:solution", Reversible: true}
}

func (c *Committer) Run(ctx context.Context, in chakra.Message, env chakra.Envelope) (chakra.Message, error) {
	winner := in.Payload.(Winner)
	env.Store.Set("trusted:solution", winner.Patch) // captured in the scope, not yet merged
	return chakra.Message{Payload: winner.Verdict, Schema: "verdict"}, nil
}

type Winner struct {
	Patch   string
	Verdict Verdict
}

// --- The trust boundary: a DeltaGate on the commit edge ---
//
// Fires AFTER the committer returns, BEFORE its writes merge to the
// parent store. Rejects the merge unless the verifier score clears the
// threshold, and unless every write targets the trusted: namespace.
// This is defense in depth: the agent already picks the best candidate,
// but the gate guarantees nothing unverified crosses into trusted: even
// if the loop above is wrong. Content is the adversary, not the agent.
func RequireScore(min float64) chakra.DeltaGate {
	return func(result chakra.Message, delta chakra.StoreDelta, env chakra.Envelope) error {
		v, ok := result.Payload.(Verdict)
		if !ok {
			return fmt.Errorf("commit rejected: no verdict attached to result")
		}
		if v.Score < min {
			return fmt.Errorf("commit rejected: score %.2f < threshold %.2f (%d/%d tests passed)",
				v.Score, min, v.Passed, v.Total)
		}
		for k := range delta.Writes {
			if !strings.HasPrefix(k, "trusted:") {
				return fmt.Errorf("commit rejected: write to non-trusted key %q", k)
			}
		}
		return nil
	}
}

// --- The agent: best-of-N generate, verify, rerank, gated commit ---

type GenVerifyAgent struct {
	registry *chakra.Registry
	n        int // candidates per task
}

func (a *GenVerifyAgent) Meta() chakra.Meta {
	return chakra.Meta{ID: "genverify-agent", Description: "best-of-N generate + verifier rerank + gated commit"}
}

func (a *GenVerifyAgent) Run(ctx context.Context, in chakra.Message, env chakra.Envelope) (chakra.Message, error) {
	task := in.Payload.(string)

	best := Winner{Verdict: Verdict{Score: -1}}

	// --- best-of-N: the for loop IS the reranker ---
	for i := 0; i < a.n; i++ {
		select {
		case <-env.Done:
			return chakra.Message{}, chakra.ErrCancelled
		default:
		}

		cand, err := a.registry.Invoke(ctx, "genverify-agent", "generator",
			chakra.Message{Payload: task, Schema: "text"}, env)
		if err != nil {
			return chakra.Message{}, fmt.Errorf("generator: %w", err)
		}
		patch := cand.Payload.(string)

		vres, err := a.registry.Invoke(ctx, "genverify-agent", "verifier",
			chakra.Message{Payload: patch, Schema: "candidate-patch"}, env)
		if err != nil {
			return chakra.Message{}, fmt.Errorf("verifier: %w", err)
		}
		verdict := vres.Payload.(Verdict)

		// rerank: keep the best-scoring candidate so far
		if verdict.Score > best.Verdict.Score {
			best = Winner{Patch: patch, Verdict: verdict}
		}

		env.Budget.IterSoFar++
	}

	// --- commit through the gated edge ---
	// If best.Verdict.Score is below threshold, the DeltaGate rejects the
	// merge and Invoke returns an error. Nothing reaches trusted:solution.
	if _, err := a.registry.Invoke(ctx, "genverify-agent", "committer",
		chakra.Message{Payload: best, Schema: "winner"}, env); err != nil {
		return chakra.Message{Payload: fmt.Sprintf("no candidate cleared verification: %v", err), Schema: "text"}, nil
	}

	return chakra.Message{Payload: best.Patch, Schema: "candidate-patch"}, nil
}

// --- Wiring ---

func main() {
	reg := chakra.NewRegistry()
	store := chakra.NewMemStore()

	reg.Register(&Generator{model: newLLM()})
	reg.Register(&Verifier{})
	reg.Register(&Committer{})

	agent := &GenVerifyAgent{registry: reg, n: 5}
	reg.Register(agent)

	// The agent may call generator and verifier freely.
	reg.Connect("genverify-agent", "generator")
	reg.Connect("genverify-agent", "verifier")

	// The commit edge carries the trust boundary: a DeltaGate that only
	// merges writes if the verifier score clears 0.8 (80% of tests pass).
	reg.ConnectWithDelta("genverify-agent", "committer", RequireScore(0.8))

	cancel := make(chan struct{})
	env := chakra.Envelope{
		Budget: chakra.Budget{MaxIter: 20, MaxCost: 5.0, MaxDepth: 5},
		Done:   cancel,
		Store:  store,
	}

	result, err := agent.Run(context.Background(), chakra.Message{
		Payload: "Fix the failing test in auth/login_test.go",
		Schema:  "text",
	}, env)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(result.Payload)

	// trusted:solution is set only if the gate approved the commit.
	if sol, ok := store.Get("trusted:solution"); ok {
		fmt.Println("committed:", sol)
	} else {
		fmt.Println("nothing committed: no candidate cleared verification")
	}
}

// --- Stubs (not kernel concerns) ---

type LLM interface {
	Chat(ctx context.Context, history []Turn) (LLMResponse, error)
}
type Turn struct{ Role, Content string }
type LLMResponse struct {
	Answer string
	Cost   float64
}

func newLLM() LLM                      { panic("plug in your LLM client") }
func runTests(patch string) (int, int) { panic("plug in your test harness") }
