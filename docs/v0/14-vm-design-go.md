# 14 — VM Design in Go

## Why Go

Go's primitives map to this system's requirements with unusual precision:

| System Concept | Go Primitive | Why It Fits |
|---|---|---|
| Agent instance | goroutine | Cheap (4KB stack), thousands concurrent |
| Event bus | channel + select | Typed, first-class, priority via select ordering |
| ResourceGovernor | context.Context | Cancellation, deadlines, value threading |
| Primitive type system | interface | Structural, any type implementing it qualifies |
| Rollback semantics | defer / recover | Guaranteed cleanup, panic recovery |
| Shared class memory | sync.RWMutex | Many readers, one writer |
| Instance isolation | no shared mutable state | Enforced by channel-only communication |

The only gap: Go's scheduler is runtime-level preemptive. Application-level preemption (for the reactivity layer) is handled by checking `ctx.Done()` at every primitive boundary — standard Go idiom.

---

## Core Interfaces

```go
// The Primitive interface — everything implements this
type Primitive interface {
    ID()       PrimitiveID
    Metadata() PrimitiveMetadata
    Execute(ctx context.Context, input Input, env Envelope) (Output, error)
    Interrupt(policy InterruptPolicy) (State, error)
    Rollback(state State) error
}

// The Envelope — threaded through every call
type Envelope struct {
    Depth     int
    CostSpent float64
    TimeSpent time.Duration
    CallTrace []PrimitiveID
    Budget    Budget
    TraceID   string
}

// PrimitiveMetadata — factual + contextual
type PrimitiveMetadata struct {
    BaseCost      float64
    BaseLatency   time.Duration
    Reversible    bool
    Preemptible   bool
    Criticality   float64
    TrustLevel    float64
    TypeLevel     int  // 0-3
}
```

---

## The Event Bus

```go
type Event struct {
    Source      EventSource
    Criticality float64
    Scope       EventScope
    Payload     any
    Timestamp   time.Time
    IdempotencyKey string
}

type EventBus struct {
    critical   chan Event   // criticality > 0.8
    interrupt  chan Event   // criticality > 0.5
    enqueue    chan Event   // criticality > 0.2
    observe    chan Event   // criticality <= 0.2
    replay     []Event      // append-only log
}

// Priority routing via select — higher channels checked first
func (bus *EventBus) Route(e Event) {
    bus.replay = append(bus.replay, e)
    switch {
    case e.Criticality > 0.8:
        bus.critical <- e
    case e.Criticality > 0.5:
        bus.interrupt <- e
    case e.Criticality > 0.2:
        bus.enqueue <- e
    default:
        bus.observe <- e
    }
}
```

---

## The Resource Governor

```go
type ResourceGovernor struct {
    target   Primitive
    budget   Budget
    policy   DegradationPolicy
    fallback Primitive // for FALLBACK policy
}

func (g *ResourceGovernor) Execute(
    ctx context.Context, 
    input Input, 
    env Envelope,
) (Output, error) {

    // Check limits before executing
    if env.Depth >= g.budget.MaxDepth {
        return g.degrade(ctx, input, env, "max_depth exceeded")
    }
    if env.CostSpent >= g.budget.CostCeiling {
        return g.degrade(ctx, input, env, "cost ceiling exceeded")
    }

    // Thread context deadline
    ctx, cancel := context.WithTimeout(ctx, g.budget.TimeRemaining(env))
    defer cancel()

    // Increment depth for child calls
    childEnv := env.WithDepth(env.Depth + 1)

    return g.target.Execute(ctx, input, childEnv)
}

func (g *ResourceGovernor) degrade(
    ctx context.Context, 
    input Input, 
    env Envelope, 
    reason string,
) (Output, error) {
    switch g.policy {
    case HALT:
        return nil, fmt.Errorf("halted: %s", reason)
    case SUMMARIZE:
        return g.summarize(ctx, env)
    case ESCALATE:
        return g.escalateToHuman(ctx, input, env, reason)
    case FALLBACK:
        return g.fallback.Execute(ctx, input, env)
    }
}
```

---

## Agent Instance as a Goroutine

```go
type AgentInstance struct {
    class       *AgentClass
    workingCtx  ContextWindow
    episodic    []Event
    hotSwapped  map[PrimitiveID]Primitive
    envelope    Envelope
    checkpoint  *State
    eventBus    *EventBus
}

func (a *AgentInstance) Run(ctx context.Context, task Task) error {
    for {
        select {
        // Check for preemptive events first (Go select priority via ordering)
        case event := <-a.eventBus.critical:
            if err := a.handlePreempt(ctx, event); err != nil {
                return err
            }
        default:
            // Normal execution
            action, done, err := a.planNextAction(ctx, task)
            if done || err != nil {
                return err
            }
            result, err := a.execute(ctx, action)
            if err != nil {
                a.handleError(ctx, action, err)
                continue
            }
            a.observe(result)
        }
    }
}
```

---

## Checkpointing

```go
type State struct {
    WorkingContext  ContextWindow
    EpisodicLog    []Event
    HotSwapped     map[PrimitiveID]PrimitiveID
    Envelope       Envelope
    PlanRemainder  []Action // remaining plan steps if using Plan→Execute
    Timestamp      time.Time
}

func (a *AgentInstance) Checkpoint() *State {
    return &State{
        WorkingContext: a.workingCtx.Clone(),
        EpisodicLog:   append([]Event{}, a.episodic...),
        HotSwapped:    maps.Clone(a.hotSwapped),
        Envelope:      a.envelope,
        Timestamp:     time.Now(),
    }
}

func (a *AgentInstance) Resume(state *State) {
    a.workingCtx = state.WorkingContext
    a.episodic = state.EpisodicLog
    a.hotSwapped = state.HotSwapped
    a.envelope = state.Envelope
}
```

---

## The Primitive Registry

```go
type PrimitiveRegistry struct {
    mu         sync.RWMutex
    primitives map[PrimitiveID]Primitive
    types      map[PrimitiveID]DualType
    versions   map[PrimitiveID][]Primitive // version history
}

func (r *PrimitiveRegistry) Swap(
    oldID PrimitiveID, 
    newP Primitive,
    checker TypeChecker,
) error {
    r.mu.Lock()
    defer r.mu.Unlock()

    oldType := r.types[oldID]
    newType := checker.Infer(newP)

    // Structural check first (fast)
    if !checker.StructurallyCompatible(oldType, newType) {
        return ErrIncompatibleStructure
    }
    // Semantic check (expensive, only if structural passes)
    if !checker.SemanticCompatible(oldType, newType) {
        return ErrIncompatibleSemantics
    }

    // Archive old version
    r.versions[oldID] = append(r.versions[oldID], r.primitives[oldID])
    r.primitives[newP.ID()] = newP
    return nil
}
```

---

## Concurrency Model Summary

```
Main goroutine:     orchestrator / pipeline coordinator
Per-agent goroutine: each AgentInstance.Run()
EventBus goroutine: routing loop, always running
Consolidator:       background goroutine, runs between sessions
Crystallizer:       offline process, not a goroutine in production

Communication:
  Agent → Agent:     via EventBus channels only
  Agent → Memory:    via MemoryPrimitive.Execute() calls
  Agent → Governor:  implicit (Governor wraps every primitive)
  World → Agent:     via EventBus.Publish()
```

No shared mutable state between agent instances except through:
1. Read-only access to class-level memory
2. Channel-based EventBus messages

This gives you Go's concurrency guarantees — no data races, clean isolation, and the race detector as a free correctness tool during development.

---

## Why Not Python, Rust, or TypeScript?

**Python:** GIL limits true parallelism. Async/await works but goroutines are more ergonomic for this concurrency pattern. Type system is structural but gradual — acceptable, but Go's interfaces are cleaner for the primitive algebra.

**Rust:** Ideal for the bedrock layer (zero-cost, memory-safe, no GC pauses). The borrow checker makes the shared/mutable memory model explicit. But async Rust is complex and goroutines are more natural for the agent instance pattern. Consider Rust for bedrock primitives, Go for the orchestration layer.

**TypeScript:** Good for prototyping and if the team is JS-native. The async model works. But lacks goroutines, channels are not first-class, and the type system doesn't naturally express the primitive algebra. Use for tooling and dashboards around the system, not for the core VM.

**Hybrid approach:** Go for the VM and orchestration. Rust for the bedrock safety layer. TypeScript for the developer tooling and observability UI.
