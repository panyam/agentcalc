# 02 — Design Philosophy

## The One Rule

> Before adding anything to the kernel, ask: can this be built by a user of the kernel using only what already exists?

If yes: don't add it. Build it as an example. Document it. Let users pull it in.

If no: you may have found a genuine kernel addition. But first ask the rule again, harder.

---

## The Failure Mode We Already Fell Into

We built an elaborate design across 18 documents. Each addition was locally justified:

- Memory coherence is a real problem → add Consolidator primitive
- Type safety is a real problem → add NL type checker
- Runaway agents are a real problem → add ResourceGovernor
- Topology rigidity is a real problem → add TopologyRewriter

Every problem was real. Every solution was at the wrong layer. The result was a system where implementing "hello world" requires fourteen components and a PhD to understand.

The pattern has a name: **local justification, global complexity**. Each addition looks cheap at the margin. The cumulative tax is enormous.

---

## The Test For A New Primitive

A new primitive is justified only when ALL of the following are true:

```
1. We have hit this problem in a running system, not anticipated it in design
2. The problem cannot be solved with existing Gates + Store + Registry
3. The solution is used by most users of the system, not a few
4. Removing it would break the closure property
```

If a proposed addition fails any of these: it is a user-land extension, not a kernel concern.

---

## The Test For A New Gate

Gates are cheaper than primitives — they compose rather than extend. But the same discipline applies:

```
1. Is this gate specific to one edge / one problem? → attach it to that edge
2. Is this gate used on many edges? → document it as a standard gate
3. Is this gate needed on EVERY invocation? → consider global, but be suspicious
```

Global gates are almost always a smell. If everything needs a gate, the gate's logic probably belongs somewhere else.

---

## The Store Is Not A Dumping Ground

The Store is shared state. The temptation is to put everything in it and add methods to it as needed. Resist this.

The Store has four methods: Get, Set, Update, Watch. Search is NOT a Store method. Search is a Primitive that reads from a Store. A vector index is a Store implementation. A filesystem walker is a different Store implementation. The interface stays at four methods.

When you feel the urge to add a fifth method to Store, instead ask: can I build a Primitive that wraps Store and does what I need? The answer is almost always yes.

---

## On Borrowed Abstractions

We drew analogies to: π-calculus, Actor model, Kubernetes operators, JIT compilers, subsumption architecture, database transactions, FRP, and more.

Analogies are useful for understanding. They are dangerous for design. Every time we said "this is like X, so we should also have Y from X" — we imported complexity we hadn't earned. 

The kernel should be justified by its own problems, not by the elegance of the analogies.

---

## The Meta-Lesson: Eat Your Own Cooking

We designed a system that says: start with a for loop, add things only when you feel real pain, let the running system tell you what it needs.

Then we spent 18 documents doing the opposite.

The discipline going forward: every addition to this design should be preceded by a running system that demonstrates the pain. No more speculative architecture. The documents are a map — useful to know the territory exists, not a construction plan.
