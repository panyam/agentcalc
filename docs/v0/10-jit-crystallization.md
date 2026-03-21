# 10 — JIT Crystallization: Reducing Agents to Code

## The JIT Analogy

In a JIT compiler:
```
Interpreter runs code     → profiles hot paths
Hot path identified       → compile to native code
Future calls              → skip interpreter, run native directly
Deoptimize if needed      → fall back to interpreter for novel inputs
```

For agents, the exact same structure applies:
```
Agent reasons about X     → profile decisions over N runs
Pattern crystallized      → emit deterministic code/rule
Future calls of X         → skip LLM, run deterministic directly
Deoptimize if needed      → fall back to agent for novel inputs
```

---

## Two Modes of Crystallization

### Profile-Guided (Behavioral)
Watch the agent make the same decision many times. When confidence is high, extract the rule:

```
AgentProfiler observes:
  classify_bug_severity("NullPointerException") → "high"   (847 times)
  classify_bug_severity("NullPointerException") → "medium" (3 times)

Crystallizer emits:
  if "NullPointerException" in input: return "high"   # 99.6% confidence
```

The rule is a legible explanation of what the agent was doing. If it looks wrong, you've found a bug. If it looks right, you've found an optimization.

**The exceptions are the most valuable output.** The 3 cases where the agent returned "medium" are exactly where the interesting edge case reasoning lives. Examine them first.

### Semantic (Introspective)
An agent reads another agent's system prompt and tool usage, then asks: *can I express what this does as code?*

```
SemanticReducer(target_agent) →
  reads: system prompt, tool usage patterns, output distributions
  asks:  is this fundamentally a classifier? a parser? a lookup? a formatter?
  if yes: emits deterministic equivalent
  if no:  flags as "irreducibly agentic" — requires reasoning on each call
```

---

## The Crystallizer Primitive

```
Crystallizer:
  input:    AgentRunLogs, confidence_threshold (e.g. 0.95)
  output:   DeterministicPrimitive | "irreducibly agentic"
  cost:     high
  latency:  high
  timing:   offline only — never in production hot path
  side_effect: emits new primitive into the primitive graph
```

This is a **development and optimization tool**, not a runtime component. It runs like a compiler — between deployments, not during execution.

---

## Deoptimization — The Critical Safety Valve

Crystallized rules must have a fallback. Never remove the original agent:

```
DeterministicPrimitive(input, envelope):
  if input in crystallization_domain:        ← cheap classifier
    return deterministic_result              ← fast path
  else:
    record: "deoptimized on input X"
    invoke: original_agent(input, envelope)  ← fall back
    flag:   consider re-crystallizing        ← update the rule
```

The `crystallization_domain` classifier is cheap and fast. Novel inputs fall through to the full agent. The system never breaks — it gracefully falls back.

---

## Crystallization as Debugging

When you crystallize, you've made reasoning **inspectable**. The extracted rule is a hypothesis about what the agent was doing. Three outcomes:

```
Rule looks correct → optimization confirmed, deploy it
Rule looks wrong   → agent had a bug, now visible and fixable
Rule is partial    → agent behavior is context-dependent in a way
                     you didn't realize, now you understand it better
```

All three outcomes are valuable. Crystallization is a form of automated behavioral analysis.

---

## The Memory Connection

Crystallization is how memory moves from L4 (semantic/vector) to L5 (crystallized/procedural):

```
L4 retrieval always returns same result for input class X
  → Crystallizer detects stable pattern
  → Emits L5 deterministic rule
  → Future calls to L4 for X are intercepted at L5 (free, instant)
```

This mirrors how human expertise develops: a novice retrieves knowledge; an expert has crystallized it into fast pattern recognition. The system grows more efficient over time through use.

---

## Lifetime of a Crystallized Primitive

```
Agent runs      → accumulates logs
Profiler watches → identifies stable patterns
Crystallizer runs (offline) → emits deterministic primitive
Type checker validates → structural + semantic compatibility confirmed
Human reviews   → approves the crystallized rule
Deploy          → crystallized primitive enters production graph
Monitor         → deoptimization rate tracked
                  if deopt_rate rises: re-profile and re-crystallize
```

Human review before deployment is non-optional for crystallized primitives. The crystallized rule becomes part of the system's behavior indefinitely — it deserves the same scrutiny as hand-written code.
