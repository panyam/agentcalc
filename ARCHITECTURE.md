# Architecture

Chakra implements a closed algebraic system for agent composition. Every component — agents, tools, memory stores, evaluators, type checkers, governors — is an instance of a single `Primitive` interface.

## Package Structure

| Package | Purpose | Dependencies |
|---------|---------|-------------|
| `primitive/` | Core `Primitive` interface, `Input`/`Output`, `State`, sentinel errors | none |
| `envelope/` | Immutable `Envelope` threaded through calls, `Budget` limits | `primitive` |
| `governor/` | `ResourceGovernor` wraps primitives with budget enforcement | `primitive`, `envelope` |
| `event/` | `Event` type, `EventBus` with 4 priority channels | none |
| `types/` | `DualType` (structural + semantic), `TypeChecker` interface | `primitive` |
| `registry/` | `PrimitiveRegistry` with type-checked hot-swapping, bedrock protection | `primitive`, `types` |
| `agent/` | `AgentClass` (static spec), `AgentInstance` (runtime goroutine) | `primitive`, `envelope`, `event`, `memory`, `governor`, `types` |
| `memory/` | `MemoryLevel` L1-L5 hierarchy, `MemoryStore` interface, `ContextWindow` | `primitive` |

## Core Design Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Universal type | `Primitive` interface | Closure — everything composes uniformly |
| Input/Output | `any` wrapper, not generics | Heterogeneous composition; generics force monomorphization |
| Envelope | Immutable (With* returns copy) | Prevents child calls from corrupting parent state |
| State serialization | `json.RawMessage` | Each primitive owns its checkpoint format |
| Agent concurrency | goroutine per instance | Go's goroutines are cheap (4KB), channels for communication |
| Event priority | Two-phase select | Go select doesn't guarantee priority ordering |
| Bedrock enforcement | Set of IDs in registry | Simple O(1) check, clear error on swap attempt |
| No third-party deps in core | stdlib only | Core must be stable and dependency-free |

## Concurrency Model

```
Main goroutine:       orchestrator / pipeline coordinator
Per-agent goroutine:  each AgentInstance.Run()
EventBus goroutine:   routing loop, always running
Consolidator:         background goroutine, runs between sessions
```

Agent-to-agent communication happens only through EventBus channels. No shared mutable state between instances except read-only class-level memory.

## Layer Model

```
Layer 0:  Bedrock          — fixed, human-authored, never touched by system
Layer 1:  Resource Governor — fixed limits (cost/depth/time)
Layer 2:  Evaluators        — stable goals defining "good"
Layer 3:  Primitive Graph   — generatable, hot-swappable, evolvable
```

Power comes from Layer 3. Safety comes from Layers 0–2.

## Design Docs Reference

Full theory in `01-overview.md` through `17-model-selection.md`. Key docs:
- `02-primitive-algebra.md` — the type system foundation
- `06-type-system.md` — dual structural + semantic types
- `07-resource-governor.md` — budget enforcement
- `08-reactivity-layer.md` — event bus and interrupts
- `11-classes-and-instances.md` — agent lifecycle
- `14-vm-design-go.md` — Go VM concrete interface sketches
