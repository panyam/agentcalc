# 04 — Memory as a Primitive

## Memory Is Not Just an Agent

Memory is a **layered hierarchy of storage with an escalating retrieval stack on top**. The storage is explicit and typed. The retrieval layer can range from deterministic to agentic — but you only escalate when cheaper retrieval fails.

> The "agent that surfaces pertinent points" is real — but it is the *top* of the retrieval stack, not the whole thing. Most retrievals should never reach it.

---

## The Storage Hierarchy

```
L1: In-context        microseconds   tiny (token window)    free        ephemeral
L2: Working/scratch   fast           session-scoped         cheap       structured
L3: Episodic          medium         run-scoped             moderate    logged events
L4: Semantic/Vector   slow           persistent             expensive   fuzzy/approximate
L5: Crystallized      instant        permanent              free        deterministic rules
```

**L5 is the most important optimization target.** Memory that has been reduced to a deterministic rule via the Crystallizer (see file 10) is the fastest possible retrieval — it doesn't retrieve at all, it just executes.

Each level is a `Memory` primitive with its own metadata:

```
Memory(
  level:        L1 | L2 | L3 | L4 | L5
  scope:        session | run | persistent
  retrieval_fn: RetrievalPrimitive
  cost:         PrimitiveMetadata.base_cost
  latency:      PrimitiveMetadata.base_latency
)
```

---

## The Retrieval Stack

Retrieval escalates through a stack of strategies, each more expensive than the last:

```
Attempt 1: Exact key lookup           deterministic   free       always try first
Attempt 2: Recency + keyword filter   deterministic   cheap      structured queries
Attempt 3: Vector similarity search   computed        moderate   semantic proximity
Attempt 4: LLM relevance judgment     agentic         expensive  only when others fail
```

This is a **primitive chain** — each attempt fires only if the previous one returns no confident result. The escalation threshold is configurable per agent class.

The agentic retrieval at L4 is a small, constrained agent whose only job is:
> "Given this context, which of these memory candidates are actually relevant?"

It is not a general agent. It has a narrow scope, a tight cost budget, and cannot itself recurse into memory retrieval (that would be infinite regress).

---

## Recency × Relevance Scoring

When multiple memory candidates are returned, they are ranked by a scoring primitive:

```
Score(memory, context) =
  α × recency_weight(age)              +
  β × semantic_similarity(memory, ctx) +
  γ × importance_tag(memory)
```

`α`, `β`, `γ` are tunable per agent class:
- Coding agent: high α (recent edits matter most)
- Research agent: high β (semantic relevance over recency)
- Planning agent: high γ (important decisions regardless of age)

The scorer is a deterministic primitive — cheap, fast, always runs. No LLM needed.

---

## Memory Primitive Metadata

Memory primitives follow the same metadata contract as all primitives:

```
EpisodicMemory:
  base_cost:     low (append only)
  base_latency:  low
  reversibility: false (logs are immutable)
  preemptible:   true (reads can be interrupted)

VectorMemory:
  base_cost:     moderate (embedding + search)
  base_latency:  moderate
  reversibility: true (reads) / false (writes)
  preemptible:   true (reads)
```

---

## Consolidation — The Background Meta-Primitive

Consolidation compresses episodic → semantic memory between sessions. Analogous to memory consolidation during sleep.

```
Consolidator (runs offline, never in hot path):
  input:   episodic log from last N runs
  output:  compressed semantic memories
           + crystallization candidates (for Crystallizer, see file 10)
           + importance-tagged key episodes to retain
  cost:    high
  timing:  between sessions, never during execution
```

Consolidation prevents episodic memory from growing unboundedly. Low-importance episodes are discarded. High-importance ones are compressed to semantic form. Patterns that repeated across episodes become crystallization candidates.

---

## Shared vs Private Memory Across Agent Instances

This is critical for multi-instance safety (see file 11 for classes/instances):

```
Class-level memory (shared, read-mostly):
  Crystallized rules     ← learned globally, read by all instances
  Shared vector store    ← the codebase RAG, shared knowledge base
  Class evaluators       ← goals every instance shares

Instance-level memory (private, read-write):
  Working context        ← this run's conversation and scratch state
  Episodic log           ← this instance's history
  Hot-swapped primitives ← this instance's specializations
```

**Instances read from shared memory but only propose writes to it.**

Writes to shared memory go through a `Consolidator` or `Graduation` process (see file 11) — never directly from an instance. This prevents one instance from corrupting state that all other instances depend on.

---

## Memory as an Optimization Target

The hierarchy gives a clear optimization path:

```
If a retrieval always returns the same result for a class of inputs
  → Crystallize it (L4 → L5)
  
If an episodic pattern repeats across runs
  → Consolidate it (L3 → L4 semantic)
  
If a semantic memory is always retrieved together with another
  → Pre-join them into a compound memory (reduce retrieval calls)
```

This mirrors how human expertise works: a novice retrieves knowledge; an expert has crystallized it into fast pattern recognition.
