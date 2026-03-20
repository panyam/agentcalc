# 02 — Primitive Algebra

## The Primitive Type

Every component in the system — tools, memory, evaluators, agents — is an instance of `Primitive`:

```
Primitive = {
  signature:      Input → Output
  metadata:       PrimitiveMetadata
  lifecycle:      static | generatable | hot-swappable
}

PrimitiveMetadata = {
  // Factual — owned by the primitive author, never overridden
  base_cost:        float        // compute/API cost per invocation
  base_latency:     float        // expected execution time
  reversibility:    bool         // can this be undone?
  
  // Contextual — set by orchestrator at composition time
  criticality:      float        // 0.0 = background, 1.0 = must fire now
  trust_level:      float        // how much autonomy is this granted?
  
  // Interrupt contract
  preemptible:      bool
  checkpoint_fn:    () → State
  resume_fn:        State → ()
  rollback_fn:      () → ()
}
```

---

## Primitive Categories

**Leaf Primitives** — no sub-primitives inside
```
Action(schema, cost, latency, reversibility)  → Result
Memory(type, retrieval_fn, scope)             → State
Signal(condition, threshold)                  → Bool     ← subsumption lives here
```

**Higher-Order Primitives** — take/return other primitives
```
Evaluator(fn: Result → Score)                 → Score
Validator(fn: Result → Bool)                  → Bool
Classifier(fn: Context → Label)               → Label
Composer(P[], strategy)                       → P        ← builds new primitives
Agent(P[], policy, memory)                    → P        ← IS a primitive
```

**Meta-Primitives** — operate on the primitive space itself
```
Generator(spec)           → P        ← creates new primitives at runtime
Swapper(P_old, P_new)     → void     ← hot-swaps a primitive
Inspector(P)              → Metadata ← introspection
Crystallizer(logs)        → P        ← reduces agent behavior to deterministic code
```

---

## Metadata Authority

Metadata is assigned by three parties, with clear authority boundaries:

| Metadata Field | Authority | Rationale |
|---|---|---|
| `base_cost`, `base_latency` | Primitive author | Factual — they know their own mechanics |
| `reversibility` | Primitive author | Factual — cannot be overridden |
| `criticality` | Orchestrator at composition time | Contextual — depends on the system, not the primitive |
| `trust_level` | Orchestrator | Contextual |
| Runtime overrides | LLM can **propose**, Governor **approves** | LLM is advisor, not authority |

> **Rule:** Factual metadata is immutable after authoring. Contextual metadata is set at composition. The LLM may propose changes but never unilaterally apply them.

---

## The Closure Property in Practice

Because `Agent` is a `Primitive`, all of these are valid:

```
// An agent used as a tool
search_agent = Agent([web_search, read_page], ...)
orchestrator = Agent([search_agent, write_tool], ...)  // agent as tool ✓

// An agent used as an evaluator
critic_agent = Agent([read_diff, assess_quality], ...)
pipeline.set_evaluator(critic_agent)  // agent as evaluator ✓

// An agent used as a type checker
semantic_checker = Agent([compare_signatures, assess_compat], ...)
type_system.set_semantic_layer(semantic_checker)  // agent as type checker ✓

// An agent generating primitives
generator_agent = Agent([inspect_codebase, emit_tools], ...)
primitive_graph.register_generator(generator_agent)  // agent as generator ✓
```

No special casing at any layer. The algebra is closed.

---

## The Reactive vs Deliberative Axis

Rather than a binary (subsumption vs LLM reasoning), every primitive has a position on a continuous axis expressed through its metadata:

```
criticality → 1.0  + preemptible=false  →  behaves like subsumption base layer
criticality → 0.0  + high cost          →  deliberative, only fires when needed
```

The scheduling policy is a function over these values — not a hardcoded architectural choice. This means subsumption-style reflexes and LLM-style reasoning emerge from the same primitive type, differentiated only by metadata.
