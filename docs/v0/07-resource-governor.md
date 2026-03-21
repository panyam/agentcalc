# 07 — Resource Governor

## Purpose

The Resource Governor is the system's budget enforcer. It wraps any primitive and ensures that cost, depth, and time limits are respected — with graceful degradation rather than hard crashes.

It is a **fixed primitive** — it lives in Layer 1, cannot be swapped, and cannot be bypassed by generated primitives.

---

## Structure

```
ResourceGovernor(
  target:        P,                  ← the primitive being governed
  max_depth:     int,                ← maximum recursive agent spawning depth
  cost_ceiling:  float,              ← maximum total cost for this subtree
  time_budget:   Duration,           ← wall clock limit
  on_exceed:     DegradationPolicy   ← what to do when a limit is hit
) → P   ← returns a governed version of P
```

---

## The Context Envelope

The Governor threads a **context envelope** through every primitive call in the tree. Every primitive receives it and passes it forward:

```
Envelope = {
  depth:       int             ← current recursion depth
  cost_spent:  float           ← cumulative cost so far
  time_spent:  Duration        ← wall clock elapsed
  call_trace:  PrimitiveID[]   ← ordered log of every primitive called
  budget:      Budget          ← limits for this subtree
}
```

Before each primitive invocation, the Governor checks:

```
if envelope.depth >= budget.max_depth:     → trigger on_exceed
if envelope.cost_spent >= budget.ceiling:  → trigger on_exceed
if envelope.time_spent >= budget.time:     → trigger on_exceed
```

The `call_trace` is your observability record — it is the audit log for the entire execution.

---

## Degradation Policies

When a limit is exceeded, the system doesn't crash — it degrades gracefully:

```
HALT        Stop immediately, return what we have so far.
            Best for: hard safety limits, irreversible action thresholds.

SUMMARIZE   Compress current context, continue at shallower depth.
            Best for: cost limits hit mid-task, task still completable.

ESCALATE    Surface to human-in-loop primitive.
            Best for: ambiguous situations requiring judgment.

FALLBACK(P) Hot-swap to a cheaper primitive and continue.
            Best for: budget pressure where a simpler approach exists.
```

`FALLBACK` is the richest policy — the Governor itself performs a hot-swap when limits are hit. The agent doesn't notice; it just starts using a cheaper primitive. This requires the fallback primitive to pass the type checker (see file 06).

---

## Depth as the Anti-"Agent Crazy" Mechanism

The `max_depth` parameter is the primary guard against unbounded agent spawning:

```
depth 0:  the root task
depth 1:  agents spawned by root
depth 2:  agents spawned by depth-1 agents
...
depth N:  → HALT or FALLBACK, no more spawning
```

At max depth, only **leaf primitives** are allowed — no more agent invocations. This guarantees termination.

Recommended defaults by use case:
```
Simple task assistant:    max_depth = 2
Coding agent:             max_depth = 4
Research agent:           max_depth = 5
Autonomous pipeline:      max_depth = 6, with ESCALATE at limit
```

These are conservative starting points. Increase only when you have observability to justify it.

---

## Transaction Scope

For interrupt safety (see file 08), the Governor tracks transaction boundaries:

```
TransactionScope = {
  atomic:    bool       ← if true, all-or-nothing execution
  group_id:  string     ← primitives sharing a group_id are transacted together
}
```

When an interrupt arrives mid-transaction:

```
DRAIN    → finish current transaction group, then handle interrupt
SUSPEND  → checkpoint entire transaction, handle interrupt, resume if possible
ABORT    → rollback transaction (via rollback_fn on each primitive), restart
```

The Governor selects the policy based on interrupt criticality and transaction cost.

---

## Composing Governors

Governors can nest — each sub-agent can have its own Governor with tighter limits:

```
Root Governor:          max_depth=4, cost_ceiling=$1.00
  └── Sub-agent Governor: max_depth=2, cost_ceiling=$0.20
        └── Leaf Governor: max_depth=0, cost_ceiling=$0.05
```

Each governor only sees its own subtree's budget. The root governor enforces the global ceiling. This means a runaway sub-agent can exhaust its own budget without affecting the root — clean containment.
