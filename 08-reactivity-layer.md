# 08 — Reactivity Layer

## The Core Insight

Most agent systems treat human input as valid only at the start or end of a run. The reactivity layer makes events **orthogonal to the call tree** — they can reach any node at any depth, at any time.

The call tree is top-down. The event bus is beside it, not inside it.

```
World Events ──────────────────────────────┐
Human Prompt ──────────────────────────┐   │
Test Failure ────────────────────────┐ │   │
External Change ───────────────────┐ │ │   │
                                   ↓ ↓ ↓   ↓
Orchestrator ←────────────── [ EVENT BUS ]
  └── Agent A       ←──────────────┘  │
        └── Agent B ←─────────────────┘
              └── Primitive P
```

---

## Event Structure

Events are first-class primitives with the same metadata contract:

```
Event = {
  source:      Human | Tool | Environment | Agent
  criticality: float        ← how urgently must this be handled?
  scope:       Broadcast | Targeted(node_id)
  payload:     any
  timestamp:   Time
  idempotency_key: string   ← prevents duplicate handling
}
```

---

## Criticality Levels and Their Semantics

```
criticality → 1.0   PREEMPT    Stop everything now.
                               e.g. "stop, this is completely wrong"
                               e.g. safety signal fires

criticality → 0.7   INTERRUPT  Finish current atomic primitive, then handle.
                               e.g. "actually, prioritize the auth bug"

criticality → 0.4   ENQUEUE    Finish current task, handle next.
                               e.g. "when done, also check performance"

criticality → 0.1   OBSERVE    Log it, factor it in passively.
                               e.g. a metric that drifted slightly
```

This maps directly to the subsumption priority axis — but applied to incoming events rather than competing behaviors.

---

## The Human Event Classifier

Human messages don't come with criticality labels. A fast, cheap classifier assigns them:

```
HumanEventClassifier:
  cost:      very low (small model or rule-based)
  latency:   very low
  always_on: true
  swappable: false         ← lives in bedrock layer

Input:  "stop, this is going in totally the wrong direction"
Output: Event(criticality=1.0, PREEMPT)

Input:  "actually, make the function return a list not a dict"
Output: Event(criticality=0.7, INTERRUPT)

Input:  "also add logging when you're done"
Output: Event(criticality=0.4, ENQUEUE)

Input:  "fyi the external API changed their rate limits"
Output: Event(criticality=0.1, OBSERVE)
```

The human doesn't need to understand agent internals. They just talk. The classifier handles interpretation.

---

## The Interrupt Contract

For a node to be safely interruptible, it declares a contract:

```
Primitive.interrupt_policy = {
  preemptible:    bool          ← can I be stopped mid-execution?
  checkpoint_fn:  () → State    ← what state to save if stopped
  resume_fn:      State → ()    ← can I resume from checkpoint?
  rollback_fn:    () → ()       ← how to undo if I must
}
```

The event bus respects this contract:

```
if event.criticality == PREEMPT:
  if current_node.preemptible:
    checkpoint → handle event → resume or redirect
  else:
    propagate PREEMPT up to nearest preemptible ancestor
```

**A high-criticality event always gets handled.** The system finds the nearest safe interruption point rather than tearing down state. The interrupt propagates up the call tree until it finds a preemptible node.

---

## State Coherence: Transaction Semantics

When an interrupt arrives mid-execution, partial execution is a problem. The Governor (file 07) handles this via transaction scope:

```
On interrupt mid-transaction:

DRAIN    → finish current atomic group, then handle event
           use when: interrupt criticality < 0.9, transaction almost done

SUSPEND  → checkpoint entire transaction state, handle event, resume if possible
           use when: long-running transaction, event may redirect but not cancel

ABORT    → rollback transaction via rollback_fn on each primitive
           use when: preempt-level event, or resumption would be meaningless
```

Primitives that cannot rollback must declare `reversibility: false` — the system will never interrupt them mid-execution.

---

## The Event Bus as a Primitive

```
EventBus:
  subscribe(node_id, event_filter)  → Subscription
  publish(event)                    → DeliveryReceipts
  replay(from_timestamp)            → Event[]         ← critical for debugging
  always_on:  true
  swappable:  false
```

The `replay` method reconstructs exactly what events arrived, when, and what the system was doing at that moment. This is the observability story for the reactivity layer — combine with `call_trace` from the Governor envelope for full execution reconstruction.

---

## Full Architecture with Reactivity

```
┌──────────────────────────────────────────────────────┐
│  WORLD: humans, tools, environments, other agents    │
└─────────────────────────┬────────────────────────────┘
                          │ events
┌─────────────────────────▼────────────────────────────┐
│  REACTIVITY LAYER                                    │
│  EventBus + HumanEventClassifier + CriticalityRouter │
│  always-on, non-swappable                            │
└─────────────────────────┬────────────────────────────┘
                          │ interrupt / enqueue / observe
┌─────────────────────────▼────────────────────────────┐
│  LAYER 0: BEDROCK                                    │
│  Fixed validators, safety signals, hard limits       │
└─────────────────────────┬────────────────────────────┘
                          │
┌─────────────────────────▼────────────────────────────┐
│  LAYER 1: RESOURCE GOVERNOR                          │
│  Depth, cost, time, transaction scope                │
└─────────────────────────┬────────────────────────────┘
                          │
┌─────────────────────────▼────────────────────────────┐
│  LAYER 2: EVALUATORS & STABLE GOALS                  │
└─────────────────────────┬────────────────────────────┘
                          │
┌─────────────────────────▼────────────────────────────┐
│  LAYER 3: GENERATABLE PRIMITIVE GRAPH                │
│  Tools, memory, agents — hot-swappable               │
└──────────────────────────────────────────────────────┘
```

The reactivity layer sits above everything and can reach down to any layer depending on criticality. Safety signals at bedrock level can be triggered by events. The system is event-responsive at every layer.
