# 01 — Overview: A Calculus for Agent Systems

## The Central Idea

Most agent frameworks are collections of patterns bolted together. This document series describes a **closed algebraic system** for agents — where every component, including agents themselves, is an instance of the same primitive type, composable by the same rules.

The goal: a system expressive enough to model any agent behavior, constrained enough to reason about cost, safety, and correctness.

---

## The Four Layers

Every agent system is a set of decisions across four layers:

```
┌─────────────────────────────────────┐
│  4. COORDINATION                    │  How agents work together
├─────────────────────────────────────┤
│  3. PLANNING & REASONING            │  How an agent thinks
├─────────────────────────────────────┤
│  2. MEMORY & STATE                  │  What an agent knows / remembers
├─────────────────────────────────────┤
│  1. TOOLS & ACTIONS                 │  What an agent can do
└─────────────────────────────────────┘
```

Each layer depends on the one below. Most "agent frameworks" are just opinions about these four layers.

---

## The Closure Condition

The system is **closed** when:

> An agent IS a primitive. It takes primitives as arguments, returns primitives as outputs, and can itself be passed to another agent without special casing.

This is the key property that makes composition, hot-swapping, and meta-agents coherent.

---

## The Stable Core / Generatable Periphery Principle

Every design decision in this system flows from one principle:

> The system needs a **stable core** it cannot modify, surrounding a **generatable periphery** it can evolve freely.

```
Layer 0:  Bedrock          fixed, human-authored, never touched by system
Layer 1:  Resource Governor fixed limits, cost/depth/time
Layer 2:  Evaluators        stable goals — what "good" means
Layer 3:  Primitive Graph   generatable, hot-swappable, evolvable
```

The power comes from Layer 3. The safety comes from Layers 0–2.

---

## Document Map

| File | Topic |
|------|-------|
| 02 | Primitive Algebra — the type system |
| 03 | Tools & Actions — leaf primitives |
| 04 | Memory as a Primitive — hierarchy and retrieval |
| 05 | Planning & Reasoning — thinking patterns |
| 06 | Type System — structural + NL-aware checking |
| 07 | Resource Governor — cost, depth, time |
| 08 | Reactivity Layer — events, interrupts, human input |
| 09 | Coordination Patterns — pipelines vs meshes |
| 10 | JIT Crystallization — reducing agents to code |
| 11 | Classes & Instances — agent lifecycle |
| 12 | Design Principles — avoiding agent overreach |
| 13 | Bootstrap & Seeding — starting the system |
| 14 | VM Design in Go — implementation architecture |
