# 17 — Model Selection

## The Wrong Default

Most agent systems make an implicit assumption: one model for everything. This is almost always wrong, for the same reason you don't use a sledgehammer for everything just because it's the most powerful tool in the shed.

The right framing: **model selection is a primitive**, subject to the same metadata, hot-swap, and fallback mechanics as everything else in the system.

---

## Model as Primitive Metadata

The model isn't a global config. It's a property of each primitive:

```go
type PrimitiveMetadata struct {
    // ... existing fields ...
    Model   ModelSpec   // which model to use for this primitive's LLM calls
}

type ModelSpec struct {
    ID              string      // "claude-opus-4", "gpt-4o-mini", etc.
    ContextWindow   int         // max tokens
    CostPerToken    float64     // input + output
    Latency         Latency     // p50, p95
    Capabilities    []Capability // CODE, REASONING, LONG_CONTEXT, FAST, etc.
    FallbackChain   []string    // ordered list of fallback model IDs
}
```

This means different primitives in the same agent can use different models. The SemanticTypeChecker (file 06) might use a reasoning-optimized model. The HumanEventClassifier (file 08) uses the cheapest fast model. The code generation tool uses a code-specialized model.

---

## The Selection Axes

Four axes govern model selection. They often pull in different directions:

```
Capability ←────────────────────────────→ Cost
  "most capable"                          "cheapest that works"

Latency ←───────────────────────────────→ Quality
  "fastest response"                      "most accurate output"

Specialization ←────────────────────────→ Generality
  "best at code"                          "best across all tasks"

Context ←───────────────────────────────→ Speed
  "200k token window"                     "8k but 10x faster"
```

No model wins on all four simultaneously. Selection is always a tradeoff expressed against the current context.

---

## The ModelSelector Primitive

Model selection is itself a primitive — a classifier that maps task context to a model ID:

```
ModelSelector:
  input:   {task_type, envelope, capabilities_needed}
  output:  ModelSpec
  cost:    very low (rule-based or tiny model)
  latency: very low
  always_on: false   ← called at primitive instantiation, not every invocation
```

Selection logic:

```
if envelope.cost_spent / envelope.budget.ceiling > 0.8:
    → cheapest capable model (budget pressure)

if task.criticality > 0.9:
    → most capable model (correctness over cost)

if task.requires_long_context:
    → model with context_window > required_tokens

if task.type == CODE_GENERATION:
    → code-specialized model

if envelope.time_spent / envelope.budget.time > 0.8:
    → fastest available model (time pressure)

else:
    → class default model
```

The ModelSelector can itself be an agent for complex selection scenarios — but it should be constrained to a single LLM call with no tools.

---

## Model Per Agent Class vs Model Per Primitive

Two natural granularities:

**Model per agent class** — simpler, more predictable
```yaml
AgentClass: CodeReviewAgent
model: claude-opus-4
```
Every primitive in this agent uses the same model. Easy to reason about, easy to bill. Wasteful — a routing step within the agent doesn't need Opus.

**Model per primitive** — efficient, complex
```yaml
AgentClass: CodeReviewAgent
primitives:
  route_to_tool:       model: haiku    # cheap, just routing
  read_and_reason:     model: opus     # expensive, needs reasoning
  format_output:       model: sonnet   # medium, formatting
  semantic_type_check: model: sonnet   # medium, type checking
```
More efficient. Harder to reason about billing and behavior. The right choice for mature, optimized systems.

**Recommended progression:** start with class-level models, profile which primitives actually need capability vs which are doing cheap routing work, then split to primitive-level models for hot paths.

---

## Dynamic Model Selection at Runtime

Model selection can change during a run based on the current envelope state. This is the most powerful and most dangerous option.

**Safe dynamic selection:** at primitive instantiation, before execution begins. The ModelSelector reads the envelope and picks a model once per primitive call. This is safe.

**Unsafe dynamic selection:** switching models mid-execution (mid-generation, mid-reasoning chain). The new model inherits a context window it didn't produce. It may reinterpret prior reasoning with different "priors." This can cause subtle coherence failures.

**Safe switching points:**
```
Safe:    at checkpoint boundaries
Safe:    at pipeline stage gates
Safe:    at the start of a new ReAct loop iteration
Unsafe:  mid-generation
Unsafe:  mid-chain-of-thought
Unsafe:  mid-transaction
```

The ResourceGovernor's degradation policy `FALLBACK` can trigger a model downgrade — but only at safe switching points (checkpoints and stage gates). The Governor tracks whether a primitive is currently at a safe switching point.

---

## The Fallback Chain

Models should have fallback chains, just like primitives:

```
Primary:   claude-opus-4       (most capable, expensive)
Fallback1: claude-sonnet-4     (good capability, half cost)
Fallback2: claude-haiku-4      (fast, cheap, lower capability)
Fallback3: local-model-7b      (free, high latency, lowest capability)
```

The Governor triggers fallback when:
- Cost ceiling approaching → downgrade
- Model API unavailable → next in chain
- Latency budget exceeded → faster model
- Capability not needed for this specific call → opportunistically downgrade

The type system applies here too: a downgraded model must still satisfy the semantic type contract of the primitive it's executing. The SemanticChecker can validate this — "will a less capable model still reliably produce outputs of type X?"

---

## Fine-Tuning as Crystallization

This is the angle that connects model selection back to the JIT crystallization discussion (file 10).

**Code crystallization:** extract stable agent behavior into deterministic rules. Fast, free at runtime, fully inspectable.

**Fine-tuning crystallization:** bake stable agent behavior into model weights. Still fast at runtime (no extra inference), but:
- Less inspectable than code rules
- Harder to rollback (requires retraining or reverting to base model)
- More general (handles similar but not identical inputs without a deoptimization path)
- Requires significant example data (hundreds to thousands of examples)

The decision matrix:

```
Behavior type:    highly specific, rule-expressible  → code crystallization
Behavior type:    general pattern, many variations   → fine-tuning crystallization
Behavior type:    irreducibly reasoning-dependent    → stay agentic

Data available:   < 50 examples                      → too little for fine-tuning
Data available:   50-500 examples                    → borderline, consider it
Data available:   > 500 examples                     → fine-tuning viable
```

Fine-tuned models are a form of L5 memory (file 04) — behavior baked permanently into the model, zero retrieval cost. But unlike code crystallization, you cannot read a fine-tuned model and verify what it learned.

---

## Model Selection for Multi-Agent Systems

In a multi-agent system (file 09), model selection becomes an architecture question:

**One model for the swarm:** simple, coherent reasoning style across agents. All agents share the same "language" and assumptions. Poor cost optimization — complex orchestrator uses the same model as a trivial router.

**Model per agent class:** good balance. Orchestrators get capable models. Leaf agents doing mechanical work get cheap models. The most practical default for most systems.

**Model per task phase:** the most sophisticated. The same agent class uses different models depending on what phase of work it's in:

```
CodeReviewAgent phases:
  Parsing phase:     cheap model (structured, predictable)
  Analysis phase:    capable reasoning model (complex judgment)
  Summary phase:     medium model (formatting, communication)
```

This requires the planner (file 05) to track task phases and update the ModelSelector accordingly.

---

## Observability for Model Selection

Model selection decisions must be logged in the call trace (file 07):

```go
type CallTraceEntry struct {
    PrimitiveID  PrimitiveID
    ModelUsed    string          // which model was actually selected
    ModelReason  string          // why it was selected
    CostActual   float64         // actual cost at this model's rate
    Timestamp    time.Time
}
```

Without this, debugging cost anomalies is nearly impossible. "Why did this run cost $2 instead of $0.20?" requires knowing which model selection decisions were made and why.

The DriftDetector (file 16) should also monitor model selection patterns — if the system is systematically falling back to cheaper models, that's a signal that either budgets are too tight or task complexity has increased.

---

## Summary: Model Selection as a Dimension of the Primitive Graph

| Decision | Recommendation |
|---|---|
| One model for all? | No — wasteful and suboptimal |
| Model per class vs per primitive? | Start per-class, evolve to per-primitive for hot paths |
| Runtime switching? | Only at safe switching points (checkpoints, stage gates) |
| Fine-tuning? | When > 500 examples of stable behavior exist, and inspectability is less critical than performance |
| Fallback chains? | Always — model APIs are not 100% reliable |
| Selection logged? | Always — required for cost observability |
| ModelSelector type? | Primitive (rule-based for most, tiny-model for complex) |
