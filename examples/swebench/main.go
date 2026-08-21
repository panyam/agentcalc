// Example: SWE-bench coding agent built on chakra.
//
// This is the target use case. A ReAct agent with 4 tools,
// a budget gate, and a test-driven loop:
//   1. Read the issue
//   2. Search for relevant code
//   3. Read files to understand context
//   4. Edit files to fix the bug
//   5. Run tests to verify
//   6. If tests fail, loop back to step 2
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/panyam/agentcalc"
)

// ============================================================
// Tools — leaf primitives, no registry needed
// ============================================================

type ReadFile struct{}

func (t *ReadFile) Meta() chakra.Meta {
	return chakra.Meta{ID: "read-file", Description: "read file contents by path"}
}

func (t *ReadFile) Run(ctx context.Context, in chakra.Message, env chakra.Envelope) (chakra.Message, error) {
	path := in.Payload.(string)
	data, err := os.ReadFile(path)
	if err != nil {
		return chakra.Message{}, err
	}
	return chakra.Message{Payload: string(data), Schema: "file-content"}, nil
}

type SearchCode struct{}

func (t *SearchCode) Meta() chakra.Meta {
	return chakra.Meta{ID: "search-code", Description: "grep codebase for a pattern"}
}

func (t *SearchCode) Run(ctx context.Context, in chakra.Message, env chakra.Envelope) (chakra.Message, error) {
	pattern := in.Payload.(string)
	out, err := exec.CommandContext(ctx, "grep", "-rn", pattern, ".").Output()
	if err != nil {
		// grep returns exit 1 for no matches — not an error
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			return chakra.Message{Payload: "no matches", Schema: "grep-results"}, nil
		}
		return chakra.Message{}, err
	}
	return chakra.Message{Payload: string(out), Schema: "grep-results"}, nil
}

type EditFile struct{}

func (t *EditFile) Meta() chakra.Meta {
	return chakra.Meta{ID: "edit-file", Description: "apply a patch to a file", Reversible: true}
}

func (t *EditFile) Run(ctx context.Context, in chakra.Message, env chakra.Envelope) (chakra.Message, error) {
	edit := in.Payload.(EditRequest)

	data, err := os.ReadFile(edit.Path)
	if err != nil {
		return chakra.Message{}, err
	}

	content := string(data)
	if !strings.Contains(content, edit.Old) {
		return chakra.Message{Payload: "old string not found in file", Schema: "error"}, nil
	}

	newContent := strings.Replace(content, edit.Old, edit.New, 1)
	if err := os.WriteFile(edit.Path, []byte(newContent), 0644); err != nil {
		return chakra.Message{}, err
	}

	return chakra.Message{Payload: "ok", Schema: "status"}, nil
}

type EditRequest struct {
	Path string
	Old  string
	New  string
}

type RunTests struct{}

func (t *RunTests) Meta() chakra.Meta {
	return chakra.Meta{ID: "run-tests", Description: "run test suite and return results"}
}

func (t *RunTests) Run(ctx context.Context, in chakra.Message, env chakra.Envelope) (chakra.Message, error) {
	testCmd := in.Payload.(string) // e.g. "go test ./auth/..."
	parts := strings.Fields(testCmd)
	out, err := exec.CommandContext(ctx, parts[0], parts[1:]...).CombinedOutput()

	result := TestResult{Output: string(out), Pass: err == nil}
	return chakra.Message{Payload: result, Schema: "test-result"}, nil
}

type TestResult struct {
	Output string
	Pass   bool
}

// ============================================================
// The SWE-bench agent — a for loop with tools
// ============================================================

type SWEAgent struct {
	registry *chakra.Registry
	model    LLM
	maxIter  int
}

func (a *SWEAgent) Meta() chakra.Meta {
	return chakra.Meta{ID: "swe-agent", Description: "SWE-bench coding agent: read, search, edit, test"}
}

func (a *SWEAgent) Run(ctx context.Context, in chakra.Message, env chakra.Envelope) (chakra.Message, error) {
	issue := in.Payload.(string)
	history := []Turn{
		{Role: "system", Content: sweSystemPrompt},
		{Role: "user", Content: issue},
	}

	for i := 0; i < a.maxIter; i++ {
		// --- cancellation ---
		select {
		case <-env.Done:
			return chakra.Message{}, chakra.ErrCancelled
		default:
		}
		if env.Budget.Exhausted() {
			return chakra.Message{Payload: "budget exhausted"}, nil
		}

		// --- think ---
		resp, err := a.model.Chat(ctx, history)
		if err != nil {
			return chakra.Message{}, err
		}
		env.Budget.CostSoFar += resp.Cost

		if resp.Done {
			return chakra.Message{Payload: resp.Answer, Schema: "patch"}, nil
		}

		// --- act ---
		toolMsg := chakra.Message{Payload: resp.ToolArgs, Schema: resp.ToolName}
		result, err := a.registry.Invoke(ctx, "swe-agent", resp.ToolName, toolMsg, env)

		var observation string
		if err != nil {
			observation = fmt.Sprintf("Error: %v", err)
		} else {
			observation = fmt.Sprintf("%v", result.Payload)
		}

		// --- observe ---
		history = append(history,
			Turn{Role: "assistant", Content: fmt.Sprintf("Action: %s\nArgs: %v", resp.ToolName, resp.ToolArgs)},
			Turn{Role: "tool", Content: observation},
		)

		// --- persist observation to store (other agents or post-mortem can read it) ---
		env.Store.Set(fmt.Sprintf("trace:%s:step:%d", a.Meta().ID, i), Turn{
			Role:    "observation",
			Content: observation,
		})

		env.Budget.IterSoFar++
	}

	return chakra.Message{Payload: "max iterations reached"}, nil
}

const sweSystemPrompt = `You are a software engineer fixing a bug.

Available tools:
- read-file: read a file. Args: file path (string)
- search-code: grep for a pattern. Args: pattern (string)
- edit-file: replace text in a file. Args: {"Path": "...", "Old": "...", "New": "..."}
- run-tests: run tests. Args: test command (string)

Strategy:
1. Read the issue carefully
2. Search for relevant code
3. Read the files to understand context
4. Make minimal edits to fix the bug
5. Run tests to verify
6. If tests fail, read the output and try again

When you are done, respond with done=true and your summary.`

// ============================================================
// A budget gate — attach to every edge
// ============================================================

func BudgetGate() chakra.Gate {
	return func(out chakra.Message, next chakra.Primitive, env chakra.Envelope) error {
		if env.Budget.Exhausted() {
			return chakra.ErrBudgetExceeded
		}
		return nil
	}
}

// A safety gate — prevent edits outside the repo
func RepoSandboxGate(repoRoot string) chakra.Gate {
	return func(out chakra.Message, next chakra.Primitive, env chakra.Envelope) error {
		if next.Meta().ID != "edit-file" {
			return nil
		}
		edit, ok := out.Payload.(EditRequest)
		if !ok {
			return nil
		}
		if !strings.HasPrefix(edit.Path, repoRoot) {
			return fmt.Errorf("chakra: edit blocked — %s is outside repo root %s", edit.Path, repoRoot)
		}
		return nil
	}
}

// ============================================================
// Wiring
// ============================================================

func main() {
	reg := chakra.NewRegistry()
	store := chakra.NewMemStore()
	model := newLLM()

	// register tools
	reg.Register(&ReadFile{})
	reg.Register(&SearchCode{})
	reg.Register(&EditFile{})
	reg.Register(&RunTests{})

	// register agent
	agent := &SWEAgent{registry: reg, model: model, maxIter: 30}
	reg.Register(agent)

	// connect with gates
	budget := BudgetGate()
	sandbox := RepoSandboxGate("/tmp/swebench/repo")

	reg.Connect("swe-agent", "read-file", budget)
	reg.Connect("swe-agent", "search-code", budget)
	reg.Connect("swe-agent", "edit-file", budget, sandbox) // both gates on this edge
	reg.Connect("swe-agent", "run-tests", budget)

	// create envelope
	cancel := make(chan struct{})
	env := chakra.Envelope{
		Budget: chakra.Budget{MaxIter: 30, MaxCost: 10.0, MaxDepth: 5},
		Done:   cancel,
		Store:  store,
	}

	result, err := agent.Run(context.Background(), chakra.Message{
		Payload: "Tests in auth/login_test.go are failing. The error is: expected token to contain user_id field but got nil. See issue #1234.",
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
