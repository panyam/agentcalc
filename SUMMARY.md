# Summary

Chakra is a Go implementation of an agent calculus — a minimal kernel where agents, tools, memory, evaluators, and external agents (Claude Code, Codex) are all instances of a single `Primitive` interface.

The project is in **v1 kernel** stage. The kernel is implemented as a single `chakra` package (~505 lines, 4 files). It compiles and vets clean with no third-party dependencies.

The kernel has four concepts: Primitive (universal interface), Envelope (execution context with budget/trace/cancellation), Store (shared state with watch), and Gate/DeltaGate (edge validation before and after invocation). A Registry wires primitives together, routes invocations through gates, and enforces store isolation via ScopedStore.

Key architectural properties:
- **The for loop IS the agent** — every reasoning pattern is a for loop with different contents
- **ScopedStore + DeltaGate** — trust boundary mechanism: primitive writes are isolated, validated, then merged
- **InvokeAsync** — async invocations with per-child cancellation and live write observation
- **Events are Store keys** — no event bus, just `Store.Watch` in a `select`
- **Reflexivity** — agents write hints about what tools they need, making every run a requirements-gathering session
- **Any agent is a Primitive** — Claude Code, Codex, humans all register and connect like any tool

The old v0 design (8 packages, 710 lines of panic stubs) is preserved in `v0/` for reference. Design docs in `docs/v1/` (current) and `docs/v0/` (archive). Developer ergonomics examples in `examples/`.

The goal: a platform where a developer can write a domain-specific agent in an afternoon, run it on real problems, and have the agent tell them what to build next.
