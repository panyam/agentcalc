// Example: Adaptive agent with dynamic goals, dynamic interrupt policy.
//
// This models agents like Lovable or Codex CLI where:
//   - The end goal evolves as the agent learns more about the task
//   - Which interrupts matter changes based on current phase
//   - "Done" is not a fixed condition — it's a judgment call that shifts
//   - The human can redefine what success looks like mid-run
//
// The key insight: the interrupt policy, the done condition, and the
// goal itself are all STATE — and state lives in the Store. The agent
// reads its own policy from the Store each iteration. Anyone (human,
// watchdog, the agent itself) can update that policy at any time.
//
// The kernel doesn't need to know about any of this. Store.Watch is
// the only mechanism needed.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/panyam/agentcalc"
)

// ============================================================
// The agent's policy — lives in the Store, not in code
// ============================================================

// Policy describes what the agent is doing, what it's listening for,
// and what "done" means right now. This is mutable state.
type Policy struct {
	// What are we trying to achieve? Updated as understanding deepens.
	Goal string

	// What phase are we in? Different phases care about different signals.
	Phase string // "exploring", "implementing", "testing", "polishing"

	// Which store keys should we watch for events?
	// The agent subscribes/unsubscribes dynamically.
	WatchKeys []string

	// What constitutes "done"? A list of conditions — all must be true.
	// These are labels the agent checks against its own state.
	DoneConditions []string // e.g. ["tests:pass", "lint:clean", "human:approved"]

	// Which event types should interrupt the current iteration?
	// vs which should be queued as hints for next iteration?
	UrgentTypes []string // e.g. ["human:redirect", "human:stop", "test:failed"]
	HintTypes   []string // e.g. ["lint:warning", "coverage:low"]
}

// ============================================================
// The adaptive agent
// ============================================================

type AdaptiveAgent struct {
	registry *chakra.Registry
	model    LLM
	maxIter  int
}

func (a *AdaptiveAgent) Meta() chakra.Meta {
	return chakra.Meta{ID: "adaptive-agent", Description: "agent with evolving goals and dynamic interrupt policy"}
}

func (a *AdaptiveAgent) Run(ctx context.Context, in chakra.Message, env chakra.Envelope) (chakra.Message, error) {
	task := in.Payload.(string)
	myID := a.Meta().ID

	// --- Bootstrap: set initial policy in the Store ---
	initialPolicy := Policy{
		Goal:  task,
		Phase: "exploring",
		WatchKeys: []string{
			"events:" + myID, // generic event channel
		},
		DoneConditions: []string{"task:complete"},
		UrgentTypes:    []string{"human:redirect", "human:stop"},
		HintTypes:      []string{"human:hint"},
	}
	env.Store.Set("policy:"+myID, initialPolicy)

	history := []Turn{
		{Role: "system", Content: adaptiveSystemPrompt},
		{Role: "user", Content: task},
	}

	// Dynamic watch channels — we rebuild these when policy changes
	watches := a.subscribe(env.Store, initialPolicy.WatchKeys)
	policyCh := env.Store.Watch("policy:" + myID)

	for i := 0; i < a.maxIter; i++ {

		// --- 1. Read current policy (may have changed) ---
		policy := a.currentPolicy(env.Store)

		// --- 2. Check hard cancellation ---
		select {
		case <-env.Done:
			return chakra.Message{}, chakra.ErrCancelled
		default:
		}

		// --- 3. Check if policy was updated (by human, watchdog, or self) ---
		select {
		case <-policyCh:
			newPolicy := a.currentPolicy(env.Store)
			// policy changed — re-subscribe to new keys
			watches = a.subscribe(env.Store, newPolicy.WatchKeys)
			// inject the goal change into context
			if newPolicy.Goal != policy.Goal {
				history = append(history, Turn{
					Role:    "system",
					Content: fmt.Sprintf("[GOAL CHANGED] was: %s → now: %s", policy.Goal, newPolicy.Goal),
				})
			}
			if newPolicy.Phase != policy.Phase {
				history = append(history, Turn{
					Role:    "system",
					Content: fmt.Sprintf("[PHASE CHANGED] %s → %s", policy.Phase, newPolicy.Phase),
				})
			}
			policy = newPolicy
		default:
		}

		// --- 4. Drain events from all watched channels ---
		for key, ch := range watches {
			a.drainEvents(ch, key, policy, &history)
		}

		// --- 5. Check if done conditions are met ---
		if a.isDone(env.Store, policy) {
			return chakra.Message{
				Payload: fmt.Sprintf("completed in phase '%s': %s", policy.Phase, lastContent(history)),
				Schema:  "result",
			}, nil
		}

		// --- 6. Budget ---
		if env.Budget.Exhausted() {
			return chakra.Message{Payload: "budget exhausted"}, nil
		}

		// --- 7. Think + Act (standard loop) ---
		resp, err := a.model.Chat(ctx, history)
		if err != nil {
			return chakra.Message{}, err
		}
		env.Budget.CostSoFar += resp.Cost

		if resp.Done {
			return chakra.Message{Payload: resp.Answer}, nil
		}

		// --- 8. Handle meta-actions: the model can update its own policy ---
		if resp.ToolName == "update-policy" {
			a.handlePolicyUpdate(env.Store, resp.ToolArgs, policy)
			history = append(history,
				Turn{Role: "assistant", Content: "Action: update-policy"},
				Turn{Role: "tool", Content: "policy updated"},
			)
			continue
		}

		// --- 9. Regular tool invocation ---
		toolMsg := chakra.Message{Payload: resp.ToolArgs, Schema: resp.ToolName}
		result, err := a.registry.Invoke(ctx, myID, resp.ToolName, toolMsg, env)

		var observation string
		if err != nil {
			observation = fmt.Sprintf("Error: %v", err)
		} else {
			observation = fmt.Sprintf("%v", result.Payload)
		}

		history = append(history,
			Turn{Role: "assistant", Content: fmt.Sprintf("Action: %s", resp.ToolName)},
			Turn{Role: "tool", Content: observation},
		)

		// --- 10. Let the model decide if phase should change ---
		// The model sees the observation and may call update-policy next turn.
		// We don't force phase transitions — the agent decides.

		env.Budget.IterSoFar++
	}

	return chakra.Message{Payload: "max iterations"}, nil
}

// subscribe creates Watch channels for a set of store keys.
func (a *AdaptiveAgent) subscribe(store chakra.Store, keys []string) map[string]<-chan any {
	watches := make(map[string]<-chan any, len(keys))
	for _, key := range keys {
		watches[key] = store.Watch(key)
	}
	return watches
}

// currentPolicy reads the agent's policy from the store.
func (a *AdaptiveAgent) currentPolicy(store chakra.Store) Policy {
	v, ok := store.Get("policy:" + a.Meta().ID)
	if !ok {
		return Policy{} // shouldn't happen — we set it at bootstrap
	}
	return v.(Policy)
}

// drainEvents reads all pending events from a channel, classifying
// them as urgent or hints based on current policy.
func (a *AdaptiveAgent) drainEvents(ch <-chan any, key string, policy Policy, history *[]Turn) {
	for {
		select {
		case v := <-ch:
			event := v.(Event)
			if isUrgent(event.Type, policy.UrgentTypes) {
				*history = append(*history, Turn{
					Role:    "user",
					Content: fmt.Sprintf("[URGENT from %s via %s] %s", event.From, key, event.Content),
				})
			} else if isHint(event.Type, policy.HintTypes) {
				*history = append(*history, Turn{
					Role:    "system",
					Content: fmt.Sprintf("[hint from %s via %s] %s", event.From, key, event.Content),
				})
			}
			// events not in urgent or hint lists are silently ignored
		default:
			return
		}
	}
}

// isDone checks all done conditions against the store.
// Each condition is a key that must be truthy in the store.
func (a *AdaptiveAgent) isDone(store chakra.Store, policy Policy) bool {
	for _, cond := range policy.DoneConditions {
		v, ok := store.Get("done:" + a.Meta().ID + ":" + cond)
		if !ok {
			return false
		}
		if b, isBool := v.(bool); isBool && !b {
			return false
		}
	}
	return len(policy.DoneConditions) > 0
}

// handlePolicyUpdate lets the model modify its own policy.
// The model outputs a partial policy update as JSON.
func (a *AdaptiveAgent) handlePolicyUpdate(store chakra.Store, args any, current Policy) {
	// The model can update any subset of the policy.
	// In practice this would be a JSON merge.
	update, ok := args.(map[string]any)
	if !ok {
		return
	}
	if goal, ok := update["goal"].(string); ok {
		current.Goal = goal
	}
	if phase, ok := update["phase"].(string); ok {
		current.Phase = phase
	}
	if conds, ok := update["done_conditions"].([]string); ok {
		current.DoneConditions = conds
	}
	if urgent, ok := update["urgent_types"].([]string); ok {
		current.UrgentTypes = urgent
	}
	if hints, ok := update["hint_types"].([]string); ok {
		current.HintTypes = hints
	}
	if keys, ok := update["watch_keys"].([]string); ok {
		current.WatchKeys = keys
	}
	store.Set("policy:"+a.Meta().ID, current)
}

// ============================================================
// Event types
// ============================================================

type Event struct {
	Type    string // e.g. "human:redirect", "test:passed", "judge:approved"
	From    string
	Content string
	Data    json.RawMessage // arbitrary structured data
}

func isUrgent(t string, urgentTypes []string) bool {
	for _, u := range urgentTypes {
		if t == u {
			return true
		}
	}
	return false
}

func isHint(t string, hintTypes []string) bool {
	for _, h := range hintTypes {
		if t == h {
			return true
		}
	}
	return false
}

// ============================================================
// How the outside world interacts with this agent
// ============================================================

// --- A human changes the goal mid-run ---
func humanRedirectsGoal(store chakra.Store, agentID string) {
	// Read current policy
	v, _ := store.Get("policy:" + agentID)
	policy := v.(Policy)

	// Change the goal
	policy.Goal = "actually, forget the login bug — the payment flow is broken and that's P0"
	policy.Phase = "exploring" // reset phase since goal changed

	// Write it back — the agent will see it via policyCh next iteration
	store.Set("policy:"+agentID, policy)
}

// --- A CI system signals test results ---
func ciReportsTestResults(store chakra.Store, agentID string, passed bool) {
	store.Set("events:"+agentID, Event{
		Type:    "test:passed",
		From:    "ci",
		Content: fmt.Sprintf("tests passed: %v", passed),
	})
	// Also update the done condition
	store.Set("done:"+agentID+":tests:pass", passed)
}

// --- A code review judge approves ---
func judgeApproves(store chakra.Store, agentID string) {
	store.Set("events:"+agentID, Event{
		Type:    "judge:approved",
		From:    "code-review-judge",
		Content: "LGTM",
	})
	store.Set("done:"+agentID+":judge:approved", true)
}

// --- A human says "btw" mid-run ---
func humanSaysBtw(store chakra.Store, agentID string, msg string) {
	store.Set("events:"+agentID, Event{
		Type:    "human:hint",
		From:    "human",
		Content: msg,
	})
}

// --- A human says "STOP, new priority" ---
func humanEscalates(store chakra.Store, agentID string, msg string) {
	store.Set("events:"+agentID, Event{
		Type:    "human:redirect",
		From:    "human",
		Content: msg,
	})
}

// ============================================================
// Example: phase-driven policy evolution
//
// As the agent moves through phases, it changes what it listens to:
//
//   exploring:     watch for human:hint, human:redirect
//                  done = never (exploring doesn't end itself)
//
//   implementing:  watch for test:failed, human:redirect, lint:warning
//                  done = tests:pass
//
//   testing:       watch for test:passed, test:failed, human:redirect
//                  done = tests:pass AND coverage > 80%
//
//   polishing:     watch for judge:approved, human:redirect
//                  done = judge:approved AND lint:clean
//
// The agent calls update-policy to transition between phases.
// The model sees the current phase in its system prompt and decides
// when to transition based on what it observes.
//
// Example model output:
//   "I've found the bug in auth/middleware.go. Transitioning to implementing."
//   Action: update-policy
//   Args: {"phase": "implementing", "done_conditions": ["tests:pass"],
//          "watch_keys": ["events:adaptive-agent", "ci:adaptive-agent"],
//          "urgent_types": ["human:redirect", "test:failed"]}
// ============================================================

const adaptiveSystemPrompt = `You are an adaptive coding agent.

Your current goal and phase are maintained as state — you can update them.

Available tools:
- read-file, search-code, edit-file, run-tests (standard coding tools)
- update-policy: change your goal, phase, done conditions, or what events you listen to.
  Args: JSON object with any subset of {goal, phase, done_conditions, urgent_types, hint_types, watch_keys}

Phases:
- exploring: understand the problem. Search, read, build mental model.
- implementing: write the fix. Edit files, run tests.
- testing: verify thoroughly. Run tests, check edge cases.
- polishing: final review. Lint, format, check for leftover debug code.

Transition between phases when you're confident the current phase is complete.
Your done conditions determine when you stop — update them as your understanding deepens.

If you receive an [URGENT] message, address it immediately.
If you receive a [hint], factor it into your next decision.
If your [GOAL CHANGED], reassess your plan from the new goal.`

// ============================================================
// Wiring
// ============================================================

func main() {
	reg := chakra.NewRegistry()
	store := chakra.NewMemStore()
	model := newLLM()

	reg.Register(&ReadFileTool{})
	reg.Register(&SearchCodeTool{})
	reg.Register(&EditFileTool{})
	reg.Register(&RunTestsTool{})

	agent := &AdaptiveAgent{registry: reg, model: model, maxIter: 50}
	reg.Register(agent)

	reg.Connect("adaptive-agent", "read-file")
	reg.Connect("adaptive-agent", "search-code")
	reg.Connect("adaptive-agent", "edit-file")
	reg.Connect("adaptive-agent", "run-tests")

	// Simulate external events
	go func() {
		// ... human, CI, judge interactions happen here
		// They write to the store. The agent picks them up.
	}()

	cancel := make(chan struct{})
	env := chakra.Envelope{
		Budget: chakra.Budget{MaxIter: 50, MaxCost: 20.0, MaxDepth: 10},
		Done:   cancel,
		Store:  store,
	}

	result, err := agent.Run(context.Background(), chakra.Message{
		Payload: "Fix the failing test in auth/login_test.go — but stay alert, priorities may shift.",
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
	Done     bool
	Answer   string
	ToolName string
	ToolArgs any
	Cost     float64
}

func newLLM() LLM              { panic("plug in your LLM client") }
func lastContent(h []Turn) string { return h[len(h)-1].Content }

type ReadFileTool struct{}
func (t *ReadFileTool) Meta() chakra.Meta { return chakra.Meta{ID: "read-file", Description: "read a file"} }
func (t *ReadFileTool) Run(ctx context.Context, in chakra.Message, env chakra.Envelope) (chakra.Message, error) { panic("impl") }

type SearchCodeTool struct{}
func (t *SearchCodeTool) Meta() chakra.Meta { return chakra.Meta{ID: "search-code", Description: "search code"} }
func (t *SearchCodeTool) Run(ctx context.Context, in chakra.Message, env chakra.Envelope) (chakra.Message, error) { panic("impl") }

type EditFileTool struct{}
func (t *EditFileTool) Meta() chakra.Meta { return chakra.Meta{ID: "edit-file", Description: "edit a file", Reversible: true} }
func (t *EditFileTool) Run(ctx context.Context, in chakra.Message, env chakra.Envelope) (chakra.Message, error) { panic("impl") }

type RunTestsTool struct{}
func (t *RunTestsTool) Meta() chakra.Meta { return chakra.Meta{ID: "run-tests", Description: "run tests"} }
func (t *RunTestsTool) Run(ctx context.Context, in chakra.Message, env chakra.Envelope) (chakra.Message, error) { panic("impl") }
