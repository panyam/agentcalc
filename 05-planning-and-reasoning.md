# 05 — Planning & Reasoning

## Planning Is a Primitive

Planning is not a special property of agents — it is a higher-order primitive that takes a goal and a set of primitives and returns a sequence of actions. Different planning strategies have different cost/reliability tradeoffs.

---

## Core Reasoning Patterns

### ReAct (Reason → Act → Observe)

The baseline. The agent alternates between reasoning and acting in a loop.

```
Thought:      "I need to find which file contains the auth logic"
Action:       search_codebase("authentication")
Observation:  [auth.go, middleware/jwt.go, ...]
Thought:      "jwt.go is most likely, let me read it"
Action:       read_file("middleware/jwt.go")
...
```

**Strengths:** Simple, robust, handles novel situations well.  
**Weaknesses:** Can loop, slow for well-structured tasks, expensive per step.

### Chain-of-Thought

Force explicit step-by-step reasoning *before* acting. Reduces reasoning errors at the cost of token overhead.

```
"Let me think through this step by step before taking any action..."
```

Almost always worth doing for non-trivial decisions. Combine with ReAct: reason in the Thought step before each Action.

### Plan → Execute

Generate a complete plan upfront, then execute each step.

```
Plan:
  1. Read current auth implementation
  2. Identify the token validation function
  3. Write tests for edge cases
  4. Implement the fix
  5. Run tests

Execute: step through plan
```

**Strengths:** Predictable, auditable, cheap per step (execution is mechanical).  
**Weaknesses:** Brittle when reality diverges from plan. Needs a replanning trigger.

### Reflexion / Self-Critique

After each action, the agent evaluates its own output before proceeding.

```
Act → Self-Critique → Revise if needed → Proceed
```

**Strengths:** Catches errors early, before they propagate.  
**Weaknesses:** Doubles cost per step. Use when correctness is critical.

### Tree of Thought

Explore multiple possible next actions, evaluate each branch, pick the best.

```
Current state → Branch A → evaluate
             → Branch B → evaluate  ← best score
             → Branch C → evaluate

Proceed with Branch B
```

**Strengths:** Highest reliability on ambiguous decisions.  
**Weaknesses:** Expensive (multiple LLM calls per decision). Reserve for high-stakes branch points.

---

## Planning Patterns as Primitives

Each pattern is a `PlanningPrimitive` with explicit cost metadata:

```
ReActPlanner:
  cost_per_step:  1 LLM call
  reliability:    medium
  use_when:       novel situations, unclear path

PlanExecutePlanner:
  cost_per_step:  ~0 (execution is mechanical after planning)
  plan_cost:      1 LLM call upfront
  reliability:    high (for well-defined tasks)
  use_when:       structured, predictable tasks

ReflexionPlanner:
  cost_per_step:  2 LLM calls
  reliability:    high
  use_when:       correctness-critical steps

TreeOfThoughtPlanner:
  cost_per_step:  N LLM calls (N = branching factor)
  reliability:    highest
  use_when:       critical decision points only
```

The ResourceGovernor (see file 07) selects or constrains which planner runs based on budget.

---

## Replanning

Plan → Execute breaks when reality diverges. The system needs a `ReplanningTrigger`:

```
ReplanningTrigger fires when:
  - An action returns an unexpected result
  - A Validator returns false
  - A Signal fires mid-execution
  - Cost has exceeded X% of budget with Y% of plan complete

On trigger:
  - Checkpoint current state
  - Re-invoke planner with: original_goal + current_state + what_failed
  - Continue from checkpoint with new plan
```

Replanning is expensive — treat it as an interrupt, not a normal loop. The goal is to minimize replanning through good upfront planning, not to make replanning cheap.

---

## The Planner Selection Policy

The orchestrator picks a planner based on task characteristics:

```
if task.is_novel and task.goal_clarity == low:
    use ReActPlanner

if task.is_structured and task.steps_known:
    use PlanExecutePlanner

if task.correctness_critical:
    wrap chosen planner with ReflexionPlanner

if task.has_irreversible_steps:
    use TreeOfThought at those specific decision points
```

This is itself a classifier primitive — cheap, deterministic, runs once at task start.
