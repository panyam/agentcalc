# Summary

Chakra is a Go implementation of an agent calculus — a closed algebraic system where agents, tools, memory, evaluators, type checkers, and governors are all instances of a single `Primitive` interface.

The project is currently in **interfaces-only scaffold** stage. All 8 packages compile with no implementations — method stubs use `panic("not implemented")`. The purpose is to validate the architecture by modeling concrete problems against the type system before committing to implementations.

The design is described across 17 design documents (`01-overview.md` through `17-model-selection.md`) covering the theory, and implemented as Go packages under `github.com/panyam/chakra`.
