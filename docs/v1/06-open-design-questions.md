# 06 — Open Design Questions (research-informed)

These are two design forks surfaced by the mid-2026 coding-agent research survey (`docs/research/coding-agents-sota.md`). Neither is a kernel addition. Both are decisions about conventions and the reference implementation that the survey argues we should make deliberately rather than by default.

The discipline from `02-design-philosophy.md` still holds: nothing here adds a fifth kernel concept. Each resolves into a convention, a Store implementation choice, or a `Meta` field. They are recorded so that when we implement, the choice is explicit.

---

## Q1: Prompt-cache stability is an unmodeled, load-bearing property

**The evidence.** Long-horizon agent cost is dominated by KV-cache reuse. An append-only, prefix-stable context is cheap because the cache hits every turn. Any edit to earlier context invalidates the cached prefix and forces recomputation of everything after it. ("Don't Break the Cache," arXiv:2601.06007, Jan 2026.) This is why OpenHands' append-only event log is not just tidy, it is economically load-bearing.

**The tension with the kernel.** Chakra's `Store` is mutable: `Set` and `Update` change values in place. If an LLM-Primitive builds its prompt from Store keys, a `Store.Set` on one of those keys between iterations silently invalidates that primitive's cached prefix. The kernel says nothing about this, so a naive implementation pays full recompute cost on every turn without any signal that it is happening.

**Why this is not a kernel change.** Cache-stability is a property of how a Primitive assembles its prompt and how the developer uses the Store. The kernel cannot know which keys feed which model. The fix lives one layer up.

**Options.**
- **A (convention, recommended).** Document that context feeding an LLM-Primitive should be treated as append-only. Prefer accumulating history in a growing slice (as the ReAct example already does with `history`) over rebuilding the prompt from mutable Store keys each turn. Name Store mutation on prompt-feeding keys as a cache hazard in the docs.
- **B (metadata hint).** Add an optional convention to `Meta.Description` (not a new field) noting whether a Primitive's output is cache-stable, so a composing agent can reason about cost. Lighter than A, weaker signal.
- **C (do nothing).** Accept that cache economics are entirely the developer's problem and out of scope. Cheapest now, but the survey suggests this is the single largest hidden operating cost of long-horizon agents.

**Recommendation:** A. It costs one docs section and one line in the ReAct/SWE-bench examples, and it prevents the most expensive silent mistake the survey identifies.

**Cross-reference:** `docs/research/coding-agents-sota.md`, Thread 5 and Mapping Gap 1.

---

## Q2: Event-sourcing versus mutable Store

**The evidence.** The OpenHands Software Agent SDK (arXiv:2511.03690, Nov 2025) makes an append-only `EventLog` the single source of truth and derives everything else from it. A single `ConversationState` is the only mutable object; all interactions are immutable events. This buys three properties for free: deterministic replay, auditability, and cache-friendliness (it is the same append-only shape Q1 wants). This is an independent re-derivation of much of the Chakra kernel, which makes the one place it differs worth examining.

**The tension with the kernel.** Chakra splits state into a mutable `Store` (Get/Set/Update/Watch) plus an append-only `Trace []string` (audit only, just ordered primitive IDs). The `Trace` is already the append-only spine, but it records only who ran, not what changed. The mutable-Store default gives up the free replay and cache-friendliness that an event-log projection would provide.

**Why this is not a kernel change.** The `Store` interface is four methods and does not mandate an implementation. A `Store` can be a plain map (today's `MemStore`) or a projection over an event log. `StoreDelta` already captures reads and writes per invocation, which is most of what an event log needs. The kernel is compatible with event-sourcing; it just does not encourage it.

**Options.**
- **A (reference implementation, recommended).** Keep the `Store` interface as is. Add an `EventStore` implementation alongside `MemStore` that records every write as an appended event and serves `Get` from the projection. Demonstrate deterministic replay from the event log in an example. This proves the kernel supports event-sourcing without forcing it, and gives users who want replay/audit a drop-in.
- **B (documentation only).** Note in the docs that `Store` may be implemented as an event-log projection and that `StoreDelta` is the hook, but ship no implementation. Lighter, but leaves the claim untested.
- **C (prefer mutable, document the trade).** State explicitly that mutable-Store is the default and that replay/audit is a userland concern, and accept the loss. Only choose this if an `EventStore` turns out to be operationally painful.

**Recommendation:** A. It is additive (a new Store implementation, not a kernel change), it validates the closure claim against the strongest external comparison point, and it directly serves Q1 by making the cache-friendly append-only shape available.

**Cross-reference:** `docs/research/coding-agents-sota.md`, Thread 1 and Mapping Gap 2.

---

## Note: verification as a first-class userland pattern

The survey's third suggestion (verification is where agents fail, and Gate/DeltaGate is the trust boundary) does not need a design question because it needs no decision. It is a pattern to document and an example to write, done in `examples/generator-verifier/`. The benchmark thread shows flaky verification and contamination are central failure modes, and the training thread shows labs train verifiers explicitly (SWE-Gym). Chakra already has the hooks: a Critic/Verifier is a for loop, a `Gate` blocks a bad edge before invocation, and a `DeltaGate` rejects a bad write before it merges. The `examples/generator-verifier/` example demonstrates both the best-of-N verifier-reranking pattern and the verifier-as-DeltaGate trust boundary.
