# Roadmap

## Phase 1: Kernel (DONE)
Single `chakra` package. Primitive, Envelope, Store, Gate, DeltaGate, ScopedStore, Registry with sync and async invocation. ~505 lines. Compiles clean.

## Phase 2: First Tools + Agent
4 SWE-bench tool primitives + ReActAgent + BudgetGate + NamespaceGate + reflexivity (run:hints). Run on 10 instances. Read the hints.

## Phase 3: Hint-Driven Iteration
The hints from Phase 2 tell you what to build. Not speculation — evidence from the running system. Add one thing at a time. Consult `docs/v0/` reference map when hitting specific pain.

## Phase 4: External Agent Integration
Register Claude Code / Codex as Primitives. Gate with budget and namespace enforcement. Use for tasks where local tools aren't sufficient. Track when delegation happens and why — the hints should tell you how to reduce it over time.

## Long-term Vision
A platform where a developer can write a domain-specific agent in an afternoon, run it on real problems, and have the agent tell them what to build next. Agents built this way are cheap to run (open models), easy to understand (you own the for loop), fast to iterate (hints drive development), and composable (any external agent is a Primitive). The model gap versus frontier systems shrinks when the tooling is tight — and the hints tell you exactly how to tighten it.
