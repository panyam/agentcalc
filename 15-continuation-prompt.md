# 15 — Continuation Prompt for Claude Code

## Context

This document series describes a **closed algebraic system for building agent systems**. The core idea: every component — tools, memory, evaluators, agents, type checkers, governors — is an instance of the same `Primitive` interface. Agents are primitives. The system is closed under composition.

The documents in this directory represent a complete conceptual design. The next phase is implementation.

---

## Prompt for Claude Code

Use this prompt to continue the work in Claude Code:

---

I have a series of design documents in this directory describing an **agent calculus** — a closed system where agents, tools, memory, evaluators, type checkers, and governors are all instances of a single `Primitive` interface.

Please read all the `.md` files in this directory first (01 through 14), then help me implement this system in Go.

**What we've designed:**

- A `Primitive` interface that everything implements (02-primitive-algebra.md)
- A layered memory system with explicit hierarchy and escalating retrieval (04-memory-as-primitive.md)
- A dual type system: structural + NL-aware semantic checking (06-type-system.md)
- A ResourceGovernor with depth/cost/time limits and graceful degradation (07-resource-governor.md)
- A reactivity layer with priority-based event routing (08-reactivity-layer.md)
- Pipeline-first coordination patterns with stage gates (09-coordination-patterns.md)
- JIT crystallization: reducing agents to deterministic code (10-jit-crystallization.md)
- Agent classes vs instances with a graduation mechanism (11-classes-and-instances.md)
- A Go VM design with goroutine-per-instance concurrency model (14-vm-design-go.md)

**Where to start:**

1. Scaffold the Go module with the core interfaces from `02-primitive-algebra.md` and `14-vm-design-go.md`
2. Implement the `Envelope` and `ResourceGovernor` first — they thread through everything
3. Implement the `EventBus` with priority channel routing
4. Implement a minimal `AgentInstance` as a goroutine
5. Write a simple end-to-end test: a coding agent with 2 tools, a governor, and one event interrupt

**Design constraints to maintain:**
- Bedrock primitives are never swappable (enforce at the registry level)
- LLM can propose metadata changes but never assign them unilaterally
- Type checking has two layers: structural (always, cheap) + semantic (at swap boundaries, expensive)
- The Crystallizer is an offline tool, never in the production hot path
- Instances communicate only via EventBus channels — no shared mutable state

**Key questions to explore during implementation:**
- How do we serialize `State` for checkpointing across process restarts?
- What does the `DualType` representation look like as a Go struct?
- How do we make the SemanticTypeChecker an Agent that implements Primitive?
- What's the cleanest way to model the memory hierarchy (L1-L5) as Go types?
- How does the Crystallizer interface with the PrimitiveRegistry?

Start by reading the design docs, then propose a package structure before writing any code.

---

## Files in This Directory

```
01-overview.md              Big picture and document map
02-primitive-algebra.md     The core type system and closure property
03-tools-and-actions.md     Leaf primitives — tools and their design
04-memory-as-primitive.md   Memory hierarchy and retrieval stack
05-planning-and-reasoning.md Thinking patterns as primitives
06-type-system.md           Structural + NL-aware dual types
07-resource-governor.md     Cost, depth, time governance
08-reactivity-layer.md      Event bus, interrupts, human input
09-coordination-patterns.md Pipelines vs meshes, multi-agent patterns
10-jit-crystallization.md   Reducing agents to deterministic code
11-classes-and-instances.md Agent class hierarchy and lifecycle
12-design-principles.md     Anti-patterns and guardrails
13-bootstrap-and-seeding.md How to start the system
14-vm-design-go.md          Go implementation architecture
15-continuation-prompt.md   File containing this prompt
16-profiling-vs-evals.md    Measurement taxonomy: profiling, evals, drift
17-model-selection.md       Model per primitive, runtime switching, fine-tuning
```
