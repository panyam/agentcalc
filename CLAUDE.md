# Chakra — Agent Calculus

## Quick Reference

- **Module**: `github.com/panyam/chakra`
- **Go version**: 1.25
- **Build**: `go build ./...`
- **Vet**: `go vet ./...`
- **No third-party deps** in core packages (`primitive/`, `envelope/`, `governor/`, `event/`). stdlib only.

## Current State

**Interfaces-only scaffold.** All packages compile but contain no implementations — method bodies are `panic("not implemented")`. The goal is to model problems against the architecture before committing to implementations. See ARCHITECTURE.md for package details.

## Design Docs

Design docs `01-overview.md` through `17-model-selection.md` describe the full agent calculus theory. Key reference for implementation:
- `14-vm-design-go.md` — Go VM design with concrete interface sketches

## Key Conventions

- **Primitive is the universal type.** Agents, tools, memory, evaluators, governors all implement `primitive.Primitive`.
- **Envelope is immutable.** `With*()` methods return copies — never mutate an envelope.
- **Input/Output use `any`, not generics.** Heterogeneous composition requires runtime typing.
- **State uses `json.RawMessage`** — each primitive owns its checkpoint format.
- **No shared mutable state** between agent instances. Communication via EventBus channels only.
- **Bedrock primitives are immutable** — the registry prevents swapping them.

## Package Dependency Order

```
primitive (no deps)
  └── envelope
  └── types
  └── event
  └── governor (imports envelope)
  └── memory
  └── registry (imports types)
  └── agent (imports envelope, event, memory, governor, types)
```
