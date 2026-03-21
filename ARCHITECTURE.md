# Architecture

Chakra implements a minimal agent calculus kernel. ~505 lines, one package, four concepts.

## The Kernel

```
Primitive    — universal interface: Meta() + Run()
Envelope     — execution context: Depth, Budget, Trace, Done, Store
Store        — shared state: Get, Set, Update, Watch
Gate         — pre-invocation edge validation
DeltaGate    — post-invocation store write validation
Registry     — wiring + dispatch: Register, Connect, ConnectWithDelta, Invoke, InvokeAsync
ScopedStore  — isolated store per invocation (trust boundary mechanism)
```

## Core Design Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Universal type | `Primitive` interface (2 methods) | Closure — everything composes uniformly |
| I/O type | `Message{Payload any, Schema string}` | Single type for input and output; schema is a label, not enforced |
| Metadata | `Meta{ID, Description, Reversible}` | 3 fields. Description is NL "type". Reversible is the one safety flag. |
| Cancellation | `Envelope.Done` channel | This IS the governor. No separate ResourceGovernor. |
| Budget | `Budget.Exhausted()` check | Flat struct with MaxDepth/MaxCost/MaxIter. Policy is user-space. |
| Topology | `Registry.Connect/Disconnect` | Directed edges with gates. Who calls Connect is not kernel's concern. |
| State sharing | `Store` interface (4 methods) | Search is a Primitive, not a Store method. |
| Trust boundary | ScopedStore + DeltaGate | Every Invoke isolates writes. DeltaGates validate before merge. The for loop is the only trusted writer. |
| Events | `Store.Watch(key)` | No event bus. Events are store keys. Key names are domain decisions. |
| Async | `InvokeAsync` → `AsyncHandle` | Per-child cancellation, live write observation, result channel. Primitive doesn't know it's async. |
| Single package | No sub-packages | Kernel is small enough. No internal dependency management needed. |

## Invocation Flow

```
Registry.Invoke(ctx, caller, target, msg, env)
  │
  ├── Check: cancellation, budget, depth
  ├── Look up target primitive and edge
  ├── Run pre-invocation Gates
  ├── Create ScopedStore (isolated writes)
  ├── Run primitive.Run() — writes land in scope
  ├── Run DeltaGates on the delta
  ├── Auto-merge approved writes to parent Store
  └── Return result

Registry.InvokeAsync(ctx, caller, target, msg, env)
  │
  ├── Same checks + gate prep (synchronous)
  ├── Create ScopedStore with observation channel
  ├── Create per-child Done channel
  ├── Launch goroutine → invokeRun
  └── Return AsyncHandle {Result, Writes, Done}
```

## Trust Model

The agent's for loop is the only trust boundary. The kernel makes discipline possible and auditable, not automatic.

- **Primitives return Messages.** They write to their ScopedStore, never to the parent directly.
- **The for loop writes directly to env.Store.** It is trusted developer code.
- **DeltaGates enforce namespace isolation.** A primitive writing to `trusted:goal` is caught — it never reaches the parent.
- **The adversary is content, not code.** Malicious files/web pages flowing through primitives, not the primitives themselves.
- **Store.Watch for observation.** Parent can monitor child's scoped writes in real time via InvokeAsync's `Writes` channel.

## The For Loop IS The Agent

Every reasoning pattern is a for loop with different contents:

```
ReAct:           for !done { think; act; observe }
Plan→Execute:    plan = make_plan(); for step in plan { execute(step) }
Critic/Verifier: for !critic_passes { generate; critique }
Tree of Thought:  for !converged { branch; evaluate; select }
```

No new primitives for any of these. They are wiring choices and loop body choices.

## Reflexivity

After every tool call, the agent writes a structured hint to the Store about what worked, what didn't, and what tool improvements would help. This makes every run a requirements-gathering session for the next iteration of tools. The agent participates in its own development.

Not a kernel feature — a convention in the for loop using `Store.Update("run:hints", ...)`.

## What Is Deliberately Absent (And Why)

- **NL type system** → user builds as a Gate if needed
- **Memory hierarchy** → user builds as Primitives + Store
- **ResourceGovernor** → Envelope.Budget IS the governor
- **Event bus** → Store.Watch + channels
- **Agent lifecycle** → an agent is a Primitive that captures a Registry
- **Topology primitives** → Registry.Connect IS topology
- **Crystallization** → offline tooling, not kernel
- **Streaming** → channels passed through Message payload (Go-native)
- **Coroutines** → goroutines + InvokeAsync (Go-native)

These are real concerns. They are not kernel concerns. See `docs/v0/` reference map for when you hit specific pain.

## Old Design (v0)

Preserved in `v0/` for reference. 8 packages, 710 lines, all interface stubs with panic bodies. Over-designed — solved problems at the wrong layer. The v1 kernel was a rewrite from scratch, not a refactor.

## Design Docs

- `docs/v1/` — current kernel spec and philosophy
- `docs/v0/` — reference map for when you hit specific pain points
- `examples/` — developer ergonomics (ReAct, Critic, SWE-bench, Interrupt, Adaptive)
