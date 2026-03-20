# 06 — Type System: Structural + NL-Aware

## The Problem

Classical type systems check shapes. When you hot-swap one primitive for another, you need to know not just that the output schema matches, but that the *meaning* of the output is compatible. A `string` output that means "a SQL query" is not compatible with a `string` input that expects "a user-facing message" — even though they're structurally identical.

---

## Dual Types

Every primitive in the system has a **dual type** — a structural part and a semantic part:

```
Type = {
  structural:  JSONSchema / function signature    ← classical type checking
  semantic:    NL description                     ← LLM-aware type checking
}
```

Example:

```
Type: CodeDiff
  structural:  { patch: string, files_changed: string[], line_count: int }
  semantic:    "A unified diff representing a focused, reviewable change to source
                code. Suitable as input to a code review primitive. Not suitable
                for direct user presentation without explanation."
```

The semantic description is the **contract** — it tells the type checker what the value means and what it's for.

---

## The Type Checker Stack

Type checking is a two-layer primitive chain:

```
StructuralChecker(P_old, P_new) → Bool    fast, deterministic, always runs
        ↓ (only if structural passes)
SemanticChecker(P_old, P_new)   → Bool    LLM call, runs only at composition boundaries
```

**StructuralChecker** — free, deterministic  
Standard JSON schema or function signature comparison. If this fails, reject immediately. No LLM needed.

**SemanticChecker** — moderate cost, only at swap time  
An LLM call with a focused prompt:

> "Consuming primitive expects: [semantic type A]. Replacement primitive produces: [semantic type B]. In the context of [task description], are these compatible? Reply: YES/NO, confidence 0–1, and one sentence explaining why."

The SemanticChecker is itself a `Primitive` with metadata:

```
SemanticChecker:
  base_cost:    moderate
  base_latency: high
  when_to_run:  composition and hot-swap boundaries only
                NOT on every invocation
  reversibility: true (pure check, no side effects)
```

---

## Gradual Typing

Not every primitive needs full dual types. Levels escalate with risk:

```
Level 0: untyped          anything connects to anything   highest risk
Level 1: structural only  schema checked                  baseline safety
Level 2: structural + NL  human-readable contract         recommended default
Level 3: full dual type   semantic checker enforced        for critical primitives
```

Primitives declare their type level. The orchestrator requires a minimum level for composition in sensitive parts of the graph. The bedrock layer always requires Level 3.

---

## Type Compatibility at Hot-Swap Time

When `Swapper(P_old, P_new)` is called:

```
1. StructuralChecker(P_old.output_type, P_new.output_type) → must pass
2. StructuralChecker(P_new.input_type, caller.input_expectation) → must pass
3. SemanticChecker(P_old, P_new, context) → must pass if either is Level 3
4. MetadataCheck: is P_new within allowed metadata bounds?
   - cost ≤ budget_remaining
   - criticality compatible with current call depth
   - reversibility not downgraded for irreversible context
```

If any check fails, the swap is rejected and the current primitive remains active.

---

## Type Inference for Generated Primitives

When the Generator emits a new primitive, it must also emit its type:

```
Generator output:
  primitive:  run_project_tests()
  type: {
    structural:  { test_results: TestResult[], passed: bool, duration_ms: int }
    semantic:    "Runs this project's full test suite and returns pass/fail
                  per test with timing. Suitable as input to any Evaluator
                  that measures test coverage or correctness."
  }
```

The Generator agent is prompted to produce the semantic description as part of its output — it's part of the primitive spec, not an afterthought.

---

## The Semantic Checker as an Agent

The SemanticChecker can itself be an Agent — this is the closure property in action. A specialized, constrained agent whose only job is type compatibility judgment:

```
SemanticTypeCheckerAgent:
  tools:    [none — reasoning only]
  memory:   [type_description_A, type_description_B, context]
  goal:     "Are these types compatible for this purpose?"
  output:   { compatible: bool, confidence: float, reason: string }
  governor: { max_depth: 1, cost_ceiling: low }  ← tightly constrained
```

This agent cannot spawn sub-agents, cannot use tools, and must respond in one pass. It is the most constrained agent in the system — deliberately so.
