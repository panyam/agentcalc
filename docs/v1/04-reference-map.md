# 04 — The Reference Map

## What We Designed (And Won't Build Yet)

The 18 documents in `/archive` are a map of the territory. They describe real problems and reasonable solutions. They are not a construction plan.

This file is a lookup table: when you hit a specific pain in the running system, here is what the archive says about it.

---

## Pain → Archive Reference

**"I need different memory strategies for different things"**
→ See archive/04-memory-as-primitive.md
→ Build a Primitive that wraps your storage. Start with in-memory map. Extend when slow.

**"Agents are running up unbounded cost"**
→ The Envelope.Budget already handles this at the kernel level.
→ If you need policy logic beyond a flat ceiling: see archive/07-resource-governor.md
→ Build it as a Gate, not a kernel addition.

**"I can't tell if one primitive's output is compatible with the next"**
→ See archive/06-type-system.md
→ Start with a schema validator Gate (deterministic, free).
→ Only reach for NL type checking if schema validation genuinely isn't enough. It is expensive and probabilistic.

**"The agent keeps doing the same wrong thing"**
→ See archive/10-jit-crystallization.md
→ Read traces first. The answer is usually a Gate or a better prompt, not crystallization.

**"I need agents to coordinate"**
→ See archive/09-coordination-patterns.md
→ Start with a linear pipeline: agent A's output is agent B's input. Add complexity only when linear fails.

**"The pattern I need isn't ReAct"**
→ See archive/05-planning-and-reasoning.md
→ The pattern is a different for loop. Change what's inside the loop before adding new primitives.

**"I need the agent to respond to events mid-run"**
→ See archive/08-reactivity-layer.md
→ Check `env.Done` at the top of each loop iteration. Add a channel to the Store if you need external signals.

**"I want different models for different primitives"**
→ See archive/17-model-selection.md
→ Model is a parameter to the Primitive, not a kernel concern. Pass it in at construction time.

**"I want to switch reasoning patterns mid-task"**
→ See archive/18-dynamic-topology-extension.md
→ This is a for loop that changes what's in its body based on a condition. Not a new primitive.

**"I want to learn from past runs and improve"**
→ See archive/10-jit-crystallization.md and archive/11-classes-and-instances.md
→ Read your traces. Extract patterns manually first. Automate only when manual extraction is clearly worth automating.

---

## The Rule For Using This Map

The archive describes where you might go. The running system tells you where you need to go.

Never build something from the archive preemptively. Use the map only when you are standing in front of a specific wall.

---

## What Is Genuinely Unknown

Some things in the archive are speculative and untested. Be more skeptical of these:

- **NL type checking reliability** — we don't know if LLM-based semantic checking is reliable enough to be load-bearing. Treat as research.
- **Topology mutation safety** — the checkpoint barrier and drain semantics are theoretically sound but untested in a real concurrent system.
- **Crystallization at scale** — the profiler → crystallizer → class graduation loop sounds good. Whether it produces correct rules reliably is an open question.
- **Graduation and class evolution** — elegant in theory, may be operationally painful in practice.

If you find yourself implementing these, you are probably ahead of where the evidence supports. Stop and measure first.
