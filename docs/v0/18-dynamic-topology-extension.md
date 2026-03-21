# 18 — Dynamic Topology: Extension Prompt for Claude Code

## Context

This document extends the agent calculus design (files 01–17) with a significant architectural addition: **topology as a first-class runtime value**. Read files 01–17 first if you haven't — this builds directly on the primitive algebra, Governor, event bus, and class/instance model already designed there.

The motivation: the reasoning patterns we want to support (ReAct, Plan→Execute, Critic/Verifier, Tree of Thought, Reflexion) were originally designed as prompt engineering conventions. Mapping them onto a typed execution substrate exposed misalignments — chief among them that the execution graph was static. Dynamic topology resolves most of those misalignments and makes the calculus genuinely expressive rather than just well-organized.

---

## The Core Shift

**Before:** topology is a deployment artifact. The PipelineDesigner (file 09) produces a static graph at design time. Primitives run within it. The graph doesn't change.

**After:** topology is a typed value that lives in the runtime, can be read by any primitive, and can be rewritten under controlled conditions at checkpoint boundaries.

```go
type Topology struct {
    mu      sync.RWMutex
    nodes   map[PrimitiveID]*NodeState
    edges   map[EdgeID]*TypedChannel
    cycles  map[CycleID]*CycleState     // explicit loop declarations
    version atomic.Int64
    log     []TopologyMutation          // full mutation history
}
```

The moment topology is a value:
- Agents can introspect their own execution structure
- Agents can propose mutations to that structure
- Patterns become topology templates that can be instantiated, composed, and swapped
- The Crystallizer can crystallize topology preferences, not just primitive behavior

---

## Patterns as Topology Templates

Each reasoning pattern is a **partial graph** — a named template with declared nodes, edges, cycles, and termination conditions.

```go
type TopologyTemplate struct {
    Name        string
    Nodes       []NodeSpec           // required node types
    Edges       []EdgeSpec           // connections between nodes
    Cycles      []CycleSpec          // explicitly declared loops
    Termination TerminationSpec      // what breaks any cycles
    Requires    []TemplateRequirement // e.g. fork() for ToT
    CostProfile CostProfile          // expected cost shape
    Compatible  []string             // which templates can replace this one
}
```

The standard library of templates to implement:

### ReActTemplate
```
nodes:  [Reason, ActionSelector, ToolDispatch, Observer]
edges:  Reason → ActionSelector → ToolDispatch → Observer → Reason
cycles: {id: "main_loop", Observer → Reason, termination: TerminationSignal}
```
The cycle is explicit — not an implicit "keep going." The `TerminationSignal` is a `Signal` primitive (file 02) that checks deterministic conditions first, then a cheap LLM call for semantic completion.

### PlanExecuteTemplate
```
nodes:  [Planner, StepExecutor, StageGate, Replanner]
edges:  Planner → StepExecutor[0..N] → StageGate → StepExecutor[next]
        StageGate(on_fail) → Replanner → StepExecutor[current]
cycles: none (linear), but replanning inserts a conditional branch
```

### CriticTemplate
```
nodes:  [Generator, Critic, FeedbackMerger]
edges:  Generator → Critic
        Critic(pass) → Output
        Critic(fail) → FeedbackMerger → Generator
cycles: {id: "critique_loop", conditional: !critic.pass,
         termination: max_iterations OR critic.pass}
```

### TreeOfThoughtTemplate
```
nodes:  [Brancher, Evaluator[N], Selector]
edges:  Brancher → Evaluator[0..N] (parallel fork)
        Evaluator[0..N] → Selector (join)
        Selector(recurse) → Brancher
requires: fork()           // isolated envelope per branch
cycles:   {id: "exploration", optional recursion on Selector}
special:  ToTGovernor      // exponential cost awareness
```

### ReflexionTemplate
```
nodes:  [Actor, SelfCritic, RevisionGate]
edges:  Actor → SelfCritic → RevisionGate
        RevisionGate(fail) → Actor (with critique in context)
        RevisionGate(pass) → Output
metadata: evaluator_independence: false  // self-evaluation, lower trust_level
```

---

## New Primitives Required

### TopologyInspector
```go
type TopologyInspector struct{}

func (t *TopologyInspector) Execute(
    ctx context.Context,
    scope InspectionScope, // LOCAL | FULL
    env  Envelope,
) (Topology, error)

// Metadata
// cost:     very low (read-only graph traversal)
// latency:  very low
// always_on: available to any primitive
// side_effects: none
```

### TopologyProposer
```go
type TopologyProposer struct {
    model ModelSpec // constrained: single LLM call, no tools
}

func (t *TopologyProposer) Execute(
    ctx         context.Context,
    observation Observation,    // what the agent noticed
    current     Topology,       // what the graph looks like now
    env         Envelope,
) (TopologyMutation, error)

// Output is a PROPOSAL, never directly applied.
// Metadata
// cost:     medium (one LLM reasoning call)
// latency:  medium
// max_depth: 0 (cannot spawn sub-agents)
```

### TopologyRewriter
```go
type TopologyRewriter struct {
    checker  TypeChecker
    governor TopologyGovernor
}

func (r *TopologyRewriter) Execute(
    ctx      context.Context,
    mutation TopologyMutation,
    env      Envelope,
) (Topology, error)

// Validates mutation before applying:
//   1. TypeChecker: edge type compatibility at transition points
//   2. CompatibilityMatrix: is this template transition allowed?
//   3. TopologyGovernor: rate limits, justification check, cost delta
//   4. Checkpoint barrier: drain in-flight messages before applying
//
// Only applies mutation if ALL checks pass.
// Increments topology.version on success.
// Appends to topology.log regardless.
//
// Metadata
// cost:     low-to-medium
// timing:   CHECKPOINT BOUNDARIES ONLY — enforced, not advisory
// reversible: true (topology.log enables rollback)
```

---

## TopologyMutation Types

```go
type TopologyMutation interface {
    topologyMutation()
    Justification() string    // required — agent must explain the mutation
    CostDelta() float64       // estimated cost change
}

// Concrete mutations:
type AddNode          struct { Primitive Primitive; Edges []EdgeSpec }
type RemoveNode       struct { PrimitiveID PrimitiveID }
type InsertBetween    struct { NewNode Primitive; ExistingEdge EdgeID }
type SplicePipeline   struct { AtNode PrimitiveID; Template TopologyTemplate }
type AddCycle         struct { From, To PrimitiveID; Condition CycleCondition }
type BreakCycle       struct { CycleID CycleID }
type SwapTemplate     struct { OldTemplate, NewTemplate TopologyTemplate }
```

`SwapTemplate` is the most important — it replaces a matched subgraph with a different topology template. This is how pattern switching works at runtime.

---

## TopologyGovernor

Separate from the ResourceGovernor (file 07), specifically for mutation rate limiting:

```go
type TopologyGovernor struct {
    MaxMutationsPerRun      int
    MinStepsBetweenMutations int     // must run N steps before mutating again
    MutationCostWeight      float64  // added to envelope cost per mutation
    JustificationRequired   bool     // agent must provide a reason
    CompatibilityMatrix     CompatibilityMatrix
}

type CompatibilityMatrix map[string]map[string]TransitionSpec

type TransitionSpec struct {
    Allowed     bool
    Conditions  []string  // e.g. "only at plan step boundaries"
    StateCarry  StateCarryPolicy // what state survives the transition
}
```

### Standard Compatibility Matrix

```go
var DefaultCompatibilityMatrix = CompatibilityMatrix{
    "ReAct": {
        "PlanExecute": {Allowed: true,  StateCarry: CARRY_WORKING_CONTEXT},
        "Critic":      {Allowed: true,  StateCarry: WRAP_AS_GENERATOR},
        "ToT":         {Allowed: true,  StateCarry: CARRY_WORKING_CONTEXT},
        "Reflexion":   {Allowed: true,  StateCarry: CARRY_WORKING_CONTEXT},
    },
    "PlanExecute": {
        "ReAct":       {Allowed: true,  StateCarry: DISCARD_PLAN},
        "ToT":         {Allowed: false, Conditions: []string{"only at plan step boundaries"}},
        "Critic":      {Allowed: true,  StateCarry: WRAP_STEP_AS_GENERATOR},
    },
    "Critic": {
        "ReAct":       {Allowed: true,  StateCarry: DISCARD_GENERATOR_SPLIT},
        "PlanExecute": {Allowed: true,  StateCarry: CARRY_WORKING_CONTEXT},
    },
    // etc.
}
```

---

## The Observation → Mutation Flow

How an agent decides to mutate its topology:

```
1. Agent running on topology T observes something notable:
   - Repeated failures of the same type
   - N iterations with no progress
   - Problem structure becomes clearer (ReAct → PlanExecute)
   - Output quality below evaluator threshold
   - Cost pressure triggering simplification

2. Observation is typed:
   type Observation struct {
       Type      ObservationType
       Evidence  []TraceEntry     // from call_trace
       Severity  float64
   }

3. At next checkpoint:
   current  := TopologyInspector.Execute(LOCAL)
   proposal := TopologyProposer.Execute(observation, current, env)
   // proposal is a TopologyMutation with justification

4. TopologyRewriter validates and applies (or rejects):
   - CompatibilityMatrix check
   - TypeChecker on edge transitions
   - TopologyGovernor rate limit check
   - Checkpoint barrier (drain in-flight)

5. If applied: topology.version++, mutation logged
   If rejected: reason logged, agent continues on current topology
```

---

## Observability: Topology-Aware Tracing

Static topologies produce clean linear traces. Dynamic ones require version-aware tracing:

```go
type TraceEntry struct {
    PrimitiveID      PrimitiveID
    TopologyVersion  int           // which topology version was active
    Timestamp        time.Time
    CostActual       float64
    Envelope         Envelope
}

type TopologyLogEntry struct {
    Version    int
    Mutation   TopologyMutation
    Reason     string              // the justification
    AppliedAt  time.Time
    ApprovedBy string              // "TopologyGovernor" or human ID
}
```

To debug a run: join `TraceEntry.TopologyVersion` with `TopologyLogEntry.Version` and you can reconstruct exactly what the graph looked like when any primitive fired.

---

## Connection to Crystallization

The Crystallizer (file 10) can now operate at the topology level, not just the primitive level:

**Topology preference crystallization:**
```
Profiler observes across 50 runs of CodingAgent:
  "When task.type == REFACTOR, agent switches ReAct → PlanExecute within 3 steps
   in 47/50 cases (94% confidence)"

Crystallizer proposes:
  New rule in CodingAgentClass:
    if task_classifier(task) == REFACTOR:
        start_topology = PlanExecuteTemplate  // not ReActTemplate

Graduate to class:
  CodingAgentClass.topology_selector updated
  Future instances start on the right topology immediately
```

The topology switch that was a runtime reasoning decision becomes a startup decision. Faster, cheaper, more predictable — the same progression as primitive crystallization but one level higher.

---

## Go Implementation Notes

The key insight is that Go's goroutine + channel model IS already a dynamic topology. The `Topology` struct makes that management explicit and governed:

```go
// Adding a node = spawning a goroutine
func (t *Topology) AddNode(p Primitive, edges []EdgeSpec) error {
    t.mu.Lock()
    defer t.mu.Unlock()

    in  := make(chan Message, bufferSize)
    out := make(chan Message, bufferSize)
    ctx, cancel := context.WithCancel(t.rootCtx)

    go p.Run(ctx, in, out)  // goroutine IS the node

    t.nodes[p.ID()] = &NodeState{cancel: cancel, in: in, out: out}
    t.connectEdges(p.ID(), edges)
    t.version.Add(1)
    return nil
}

// Removing a node = drain + cancel
func (t *Topology) RemoveNode(id PrimitiveID) error {
    state := t.nodes[id]
    // 1. Signal no more input
    close(state.in)
    // 2. Drain output (checkpoint barrier)
    for range state.out {}
    // 3. Cancel context
    state.cancel()
    // 4. Remove from map
    delete(t.nodes, id)
    t.version.Add(1)
    return nil
}

// SwapTemplate = coordinated remove/add within a lock + barrier
func (t *Topology) SwapTemplate(old, new TopologyTemplate) error {
    // Must be at checkpoint — enforced by caller (TopologyRewriter)
    t.mu.Lock()
    defer t.mu.Unlock()
    // drain all edges in old template subgraph
    // remove old nodes
    // instantiate new nodes with carried state
    // connect new edges
    t.version.Add(1)
    return nil
}
```

Cycles are explicit `CycleState` values with their own termination goroutine watching the condition.

---

## What to Implement (Suggested Order)

1. **`TopologyTemplate` struct and the 5 standard templates** (ReAct, PlanExecute, Critic, ToT, Reflexion) as concrete Go values in a `templates` package

2. **`TopologyMutation` interface and all concrete mutation types** — keep them simple, just data

3. **`CompatibilityMatrix`** — the static table of allowed transitions with `StateCarryPolicy`

4. **`TopologyGovernor`** — rate limiting and justification enforcement

5. **`TopologyInspector`** — simple read-only traversal, implement `Primitive` interface

6. **`TopologyProposer`** — single LLM call, constrained, implements `Primitive`

7. **`TopologyRewriter`** — the critical path: validation chain + checkpoint barrier + mutation application

8. **Topology-aware `TraceEntry`** — extend existing trace with `TopologyVersion`

9. **End-to-end test:** agent running on `ReActTemplate`, hits N iterations without progress, switches to `PlanExecuteTemplate`, completes task. Verify trace shows version transition, verify TopologyLog has justified mutation entry.

---

## Key Design Constraints to Maintain

- **Mutations only at checkpoint boundaries** — enforced in `TopologyRewriter`, not advisory
- **Proposals never auto-applied** — `TopologyProposer` returns a mutation value, `TopologyRewriter` decides
- **Full mutation log always** — even rejected mutations are logged with reason
- **CompatibilityMatrix is static** — it lives in bedrock, cannot itself be mutated at runtime
- **TopologyGovernor is non-swappable** — it's a bedrock primitive
- **State carry policy is explicit** — every template transition declares what state survives

---

## Open Questions to Explore During Implementation

1. How does `StateCarryPolicy` work concretely when transitioning from `CriticTemplate` back to `ReActTemplate`? What happens to the Generator/Critic split state?

2. Should `TopologyProposer` have access to the full `TopologyLog` (history of past mutations) or just the current topology? History could enable learning ("this switch didn't help last time") but adds cost.

3. How does the `fork()` requirement for `ToTTemplate` interact with the `Envelope`? Does each branch get a deep copy of the full envelope or just the cost tracking fields?

4. Can the `Crystallizer` propose changes to the `CompatibilityMatrix` itself based on observed safe transitions? (Probably not — the matrix is bedrock — but worth discussing.)

5. What's the right buffer size for `TypedChannel` in the dynamic case? Static topologies can be tuned; dynamic ones need adaptive buffering or the goroutines will block unexpectedly during transitions.