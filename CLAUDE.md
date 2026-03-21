# Chakra — Agent Calculus

## Quick Reference

- **Module**: `github.com/panyam/chakra`
- **Go version**: 1.25
- **Build**: `go build .` (v0/ is archived, not part of the build)
- **Vet**: `go vet .`
- **No third-party deps.** stdlib only.

## Current State

**v1 kernel implemented.** Four files, ~505 lines, compiles and vets clean. The kernel has four concepts: Primitive, Envelope, Store, Gate. Everything else is user-space.

Old v0 scaffold (8 packages, 710 lines, all panic stubs) preserved in `v0/` for reference only — not part of the library.

## Kernel Files

| File | Contents | Lines |
|------|----------|-------|
| `chakra.go` | Primitive, Meta, Message, Envelope, Budget, Gate, DeltaGate, StoreDelta | 77 |
| `registry.go` | Registry: Register, Connect, ConnectWithDelta, Disconnect, Invoke, InvokeAsync, Get, Edges | 270 |
| `store.go` | Store interface, MemStore, ScopedStore (isolated writes + observation), StoreWrite | 144 |
| `errors.go` | Minimal sentinel errors | 14 |

## Key Conventions

- **Primitive is the universal type.** Agents, tools, memory, external agents (Claude Code, Codex) — everything implements Primitive.
- **Message is the universal I/O type.** `Payload any`, `Schema string` (label, not enforced).
- **Envelope carries execution context.** Depth, Budget, Trace, Done channel, Store.
- **The for loop IS the agent.** Every reasoning pattern (ReAct, Critic/Verifier, Plan→Execute) is a for loop with different contents. No special primitives needed.
- **Gate validates edges before invocation.** `func(Message, Primitive, Envelope) error`.
- **DeltaGate validates store writes after invocation.** `func(Message, StoreDelta, Envelope) error`. Per-edge via `ConnectWithDelta`.
- **ScopedStore isolates primitive writes.** Every Invoke gets an isolated store. Writes are captured locally, validated by DeltaGates, then merged. The for loop writes directly to the parent store — it IS the trust boundary.
- **Registry.Connect/Disconnect are capabilities, not policies.** Who calls them is not the kernel's concern. Runtime topology changes = more Connect/Disconnect calls.
- **Budget IS the governor.** No separate ResourceGovernor. `env.Budget.Exhausted()` + `env.Done` channel.
- **Events are just Store keys.** No event bus. `Store.Watch("human:steer")` in a `select`. Key names are domain decisions, not kernel decisions.
- **Async via InvokeAsync.** Returns `AsyncHandle` with per-child cancellation (`Done`), live write observation (`Writes`), and result channel (`Result`). The primitive doesn't know it's being observed.

## Trust Model

```
Primitive invocation:
  reads from scoped store    → sees parent state
  writes to scoped store     → isolated, not yet committed
  observation channel        → parent can monitor writes in real time

Return:
  delta surfaces             → kernel inspects
  delta gates fire           → namespace enforcement
  delta merges               → only approved writes reach parent

For loop:
  writes directly to env.Store → always trusted (developer code)
```

The adversary is content flowing through primitives (malicious files, web pages), not the primitives themselves. Namespace conventions (`trusted:`, `world:`, `human:`, `agent:`) enforced by DeltaGates.

## Anti-Elaboration Rule

Before adding anything to the kernel:
1. Has the running system actually failed? No → don't add it.
2. Can it be a Gate/DeltaGate on edges? Yes → build it there.
3. Can it be a user-registered Primitive? Yes → build it there.
4. Does removing it break closure? No → don't add it.

## Design Docs

- `docs/v1/` — current kernel spec and philosophy (source of truth)
- `docs/v0/` — reference map for when you hit specific pain (18 documents)
- `examples/` — developer ergonomics examples (ReAct, Critic, SWE-bench, Interrupt, Adaptive)

## Package Structure

```
chakra     (single package, no internal deps)
v0/        (archived old design — not part of lib)
examples/  (not compiled — ergonomics sketches)
docs/      (v0/ archive, v1/ current spec)
```
