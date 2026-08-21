// Example: Interrupt and event handling in chakra.
//
// The kernel has no event bus. Instead:
//   - Store.Watch() gives you a reactive channel for any key
//   - env.Done gives you cancellation
//   - Go's select statement gives you priority
//
// This is enough. An agent that wants to be interruptible
// checks an "inbox" in the Store at each loop iteration.
// External systems (humans, other agents, webhooks) write
// to that inbox via Store.Set.
//
// This example shows:
//   1. Human-in-the-loop: a human can send "btw" messages mid-run
//   2. Agent-to-agent signals: one agent can nudge another
//   3. Priority handling: urgent interrupts vs background hints
//   4. Graceful vs hard cancellation
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/panyam/agentcalc"
)

// ============================================================
// An interruptible agent
// ============================================================

type InterruptibleAgent struct {
	registry *chakra.Registry
	model    LLM
	maxIter  int
}

func (a *InterruptibleAgent) Meta() chakra.Meta {
	return chakra.Meta{ID: "interruptible-agent", Description: "agent that checks for external events each iteration"}
}

func (a *InterruptibleAgent) Run(ctx context.Context, in chakra.Message, env chakra.Envelope) (chakra.Message, error) {
	task := in.Payload.(string)
	history := []Turn{{Role: "user", Content: task}}
	myID := a.Meta().ID

	// Watch for incoming events. Anyone can write to this key.
	// The channel is non-blocking — if nothing's there, we move on.
	inbox := env.Store.Watch("inbox:" + myID)

	// Watch for priority interrupts separately — these demand immediate attention
	urgent := env.Store.Watch("urgent:" + myID)

	for i := 0; i < a.maxIter; i++ {

		// --- Priority 1: hard cancellation ---
		select {
		case <-env.Done:
			return chakra.Message{Payload: "cancelled"}, chakra.ErrCancelled
		default:
		}

		// --- Priority 2: urgent interrupts (human says "stop" or "change direction") ---
		select {
		case event := <-urgent:
			interrupt := event.(Event)
			switch interrupt.Type {
			case "redirect":
				// human said "btw, try a different approach"
				// inject into history as a new user message
				history = append(history, Turn{
					Role:    "user",
					Content: fmt.Sprintf("[INTERRUPT] New direction from %s: %s", interrupt.From, interrupt.Content),
				})
			case "stop":
				// graceful stop — finish current thought, then return
				return chakra.Message{
					Payload: fmt.Sprintf("stopped by %s: %s", interrupt.From, interrupt.Content),
				}, nil
			}
		default:
			// no urgent interrupt — continue normally
		}

		// --- Priority 3: background hints (low-priority, non-blocking) ---
		select {
		case event := <-inbox:
			hint := event.(Event)
			// background info — append as context but don't change direction
			history = append(history, Turn{
				Role:    "system",
				Content: fmt.Sprintf("[hint from %s] %s", hint.From, hint.Content),
			})
		default:
			// no hints — that's fine
		}

		// --- The actual work (same as any ReAct loop) ---
		if env.Budget.Exhausted() {
			return chakra.Message{Payload: "budget exhausted"}, nil
		}

		resp, err := a.model.Chat(ctx, history)
		if err != nil {
			return chakra.Message{}, err
		}
		if resp.Done {
			return chakra.Message{Payload: resp.Answer}, nil
		}

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
		env.Budget.IterSoFar++
	}

	return chakra.Message{Payload: "max iterations"}, nil
}

// Event is user-space — not a kernel type. You define what events look like.
type Event struct {
	Type    string // "redirect", "stop", "hint", etc.
	From    string // who sent it
	Content string
}

// ============================================================
// Human-in-the-loop: external goroutine that writes to Store
// ============================================================

// SimulateHuman shows how a human (or a UI, or a webhook) sends
// events to a running agent. The mechanism is just Store.Set.
func SimulateHuman(store chakra.Store) {
	// Imagine this is reading from stdin, a websocket, or a Slack bot

	// After 5 seconds, human sends a hint
	time.Sleep(5 * time.Second)
	store.Set("inbox:interruptible-agent", Event{
		Type:    "hint",
		From:    "human",
		Content: "btw, the bug might be in the auth middleware, not the handler",
	})

	// After 10 seconds, human redirects
	time.Sleep(5 * time.Second)
	store.Set("urgent:interruptible-agent", Event{
		Type:    "redirect",
		From:    "human",
		Content: "actually, focus on the token validation — I just found the real issue is there",
	})
}

// ============================================================
// Agent-to-agent signaling: one agent nudges another
// ============================================================

// WatchdogAgent monitors another agent's progress and intervenes.
// This is a primitive — it captures a store reference and watches
// another agent's trace.
type WatchdogAgent struct {
	targetID string
}

func (w *WatchdogAgent) Meta() chakra.Meta {
	return chakra.Meta{ID: "watchdog", Description: "monitors another agent and intervenes if stuck"}
}

func (w *WatchdogAgent) Run(ctx context.Context, in chakra.Message, env chakra.Envelope) (chakra.Message, error) {
	// watch the target agent's iteration count
	iterCh := env.Store.Watch(fmt.Sprintf("trace:%s:iter", w.targetID))

	lastIter := 0
	stuckCount := 0

	for {
		select {
		case <-env.Done:
			return chakra.Message{}, chakra.ErrCancelled

		case v := <-iterCh:
			iter := v.(int)
			if iter == lastIter {
				stuckCount++
			} else {
				stuckCount = 0
				lastIter = iter
			}

			// if the target seems stuck, send a hint
			if stuckCount > 3 {
				env.Store.Set("inbox:"+w.targetID, Event{
					Type:    "hint",
					From:    "watchdog",
					Content: "you seem stuck — consider trying a different search query or reading a different file",
				})
				stuckCount = 0
			}
		}
	}
}

// ============================================================
// Graceful cancellation pattern
// ============================================================

// RunWithTimeout shows the cancellation pattern.
// cancel channel = env.Done. Close it to stop the agent.
func RunWithTimeout(agent chakra.Primitive, msg chakra.Message, store chakra.Store, timeout time.Duration) (chakra.Message, error) {
	cancel := make(chan struct{})
	env := chakra.Envelope{
		Budget: chakra.Budget{MaxIter: 50, MaxCost: 20.0, MaxDepth: 10},
		Done:   cancel,
		Store:  store,
	}

	// timeout → close Done channel → agent sees it next iteration
	go func() {
		time.Sleep(timeout)
		close(cancel)
	}()

	return agent.Run(context.Background(), msg, env)
}

// RunWithHumanKillSwitch shows how a human can cancel from outside.
func RunWithHumanKillSwitch(agent chakra.Primitive, msg chakra.Message, store chakra.Store) (chakra.Message, error) {
	cancel := make(chan struct{})
	env := chakra.Envelope{
		Budget: chakra.Budget{MaxIter: 50, MaxCost: 20.0, MaxDepth: 10},
		Done:   cancel,
		Store:  store,
	}

	// human types "kill" → we close the Done channel
	go func() {
		var input string
		fmt.Scanln(&input)
		if input == "kill" {
			close(cancel)
		}
	}()

	return agent.Run(context.Background(), msg, env)
}

// ============================================================
// Wiring
// ============================================================

func main() {
	reg := chakra.NewRegistry()
	store := chakra.NewMemStore()
	model := newLLM()

	// register tools (same as swebench example)
	reg.Register(&GrepTool{})

	// register agent
	agent := &InterruptibleAgent{registry: reg, model: model, maxIter: 30}
	reg.Register(agent)
	reg.Connect("interruptible-agent", "grep")

	// start human simulation in background
	go SimulateHuman(store)

	cancel := make(chan struct{})
	env := chakra.Envelope{
		Budget: chakra.Budget{MaxIter: 30, MaxCost: 10.0, MaxDepth: 5},
		Done:   cancel,
		Store:  store,
	}

	result, err := agent.Run(context.Background(), chakra.Message{
		Payload: "Fix the login bug",
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

func newLLM() LLM { panic("plug in your LLM client") }

type GrepTool struct{}
func (t *GrepTool) Meta() chakra.Meta { return chakra.Meta{ID: "grep", Description: "search code"} }
func (t *GrepTool) Run(ctx context.Context, in chakra.Message, env chakra.Envelope) (chakra.Message, error) { panic("impl") }
