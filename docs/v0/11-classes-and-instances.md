# 11 — Agent Classes & Instances

## The Distinction

```
AgentClass    = the specification (the recipe)
AgentInstance = a running agent with state (the cook, mid-task)
```

**Class** — static definition:
```
AgentClass = {
  tool_signatures:     Tool[]
  memory_config:       MemoryConfig
  evaluators:          Evaluator[]
  governor_defaults:   BudgetConfig
  system_prompt_tmpl:  string
  crystallized_rules:  DeterministicPrimitive[]
  interrupt_policy:    InterruptPolicy
  type_level:          0 | 1 | 2 | 3
}
```

**Instance** — live execution state:
```
AgentInstance = {
  class_ref:           AgentClass
  working_context:     ContextWindow    ← this run's state
  episodic_log:        Event[]          ← this instance's history
  hot_swapped:         Map<PrimID, P>   ← divergences from class
  cost_envelope:       Envelope         ← budget tracking
  checkpoint:          State?           ← saved state if suspended
}
```

---

## Instance Divergence — This Is a Feature

In classical OOP, instances share the class definition exactly. In this system, **instances can diverge from their class** through hot-swapping and contextual specialization:

```
CodeReviewAgentClass:
  retrieval: EmbeddingRetrieval(model=ada)
  evaluators: [test_pass_rate]

Instance A (Python repo):
  hot-swapped: EmbeddingRetrieval → PythonSpecificRetriever
  added evaluator: pep8_compliance
  ← specialized beyond its class

Instance B (Rust repo):
  hot-swapped: EmbeddingRetrieval → RustSpecificRetriever
  added evaluator: borrow_checker_clean
  ← diverged differently
```

Divergence is valuable — instances self-specialize to their context. But it creates a question: what happens to what was learned?

---

## The Instance Lifecycle

```
AgentClass
    ↓ instantiate (bind to task/context)
AgentInstance
    ↓ execute (accumulate state, possibly hot-swap)
AgentInstance (evolved)
    ↓ checkpoint (save state for resume or analysis)
    ↓
    ├── TERMINATE          discard instance state
    │
    └── GRADUATE           promote useful divergences back to class
            ↓
        Pattern analysis: did this instance consistently
                          diverge in a useful direction?
            ↓
        If yes: create subclass or update class definition
```

---

## Graduation: How Classes Evolve

If an instance **consistently** diverges from its class in a useful way, that pattern can be promoted back:

```
Observation:
  All Python instances swap in PythonSpecificRetriever
  All Python instances add pep8_compliance evaluator

Graduate to subclass:
  PythonCodeReviewAgent extends CodeReviewAgentClass:
    + PythonSpecificRetriever
    + pep8_compliance evaluator
```

Graduation is how the **class hierarchy evolves from observed instance behavior**, rather than being designed entirely top-down.

The Crystallizer (file 10) and Graduation work together: Crystallizer extracts rules from instance behavior; Graduation promotes the rules and structural changes to the class definition. The feedback loop:

```
Instances run → Profiler watches divergences → Crystallizer extracts rules
    ↓
Rules + divergences reviewed by human → Graduation emits new/updated class
    ↓
New instances instantiated from richer class
```

---

## Shared vs Private Memory

```
Class-level (shared, read-mostly):
  Crystallized rules      ← learned globally
  Shared vector store     ← codebase RAG, shared knowledge
  Class evaluators        ← goals every instance shares
  Type definitions        ← primitive contracts

Instance-level (private, read-write):
  Working context         ← this run's conversation
  Episodic log            ← this instance's history
  Hot-swapped primitives  ← this instance's specializations
  Cost envelope           ← this instance's budget
```

**Rule:** Instances read from shared memory freely. Instances may only **propose** writes to shared memory. Proposals go through Consolidation (file 04) or Graduation review — never directly from an instance. This prevents one instance from corrupting the shared state all instances depend on.

---

## Parallelism and Instance Isolation

Multiple instances of the same class can run concurrently. They:
- Share class-level memory (read-only access)
- Have completely isolated instance-level state
- Cannot directly communicate (must go via EventBus)
- Have independent cost envelopes (one runaway instance doesn't drain others)

The Go VM (file 14) models each instance as a goroutine — lightweight, isolated, communicating through channels.

---

## Class Versioning

When a class is updated (via graduation or manual editing), existing running instances must be handled:

```
ClassUpdate policy options:
  DRAIN:    let running instances finish, then swap
  MIGRATE:  update running instances at their next checkpoint
  FORK:     run old and new class versions in parallel, compare outputs
```

`FORK` is expensive but safe for significant class changes — you can validate the new class behavior against the old before committing.
