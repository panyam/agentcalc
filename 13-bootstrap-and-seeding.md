# 13 — Bootstrap & Seeding: Starting the System

## The Bootstrap Problem

> Who creates the first Generator? Who evaluates the first Evaluator?

Any self-modifying system faces this. The answer is that the bootstrap isn't a problem to solve — it's a **boundary to acknowledge**. Some things must be human-authored. That's a feature, not a limitation.

The system has three seeding layers:

---

## Layer 0: The Bedrock (Never Generated)

Written by humans. Never touched by the system. Contains:

- Fixed validators (what is never permitted)
- Fixed signals (what always triggers halt or escalation)
- The ResourceGovernor with hard limits
- The HumanEventClassifier
- The evaluation ground truth — what "good" means in this domain

This layer should feel like prompt engineering — because it is. It is the expression of your values, domain constraints, and safety requirements. The fact that it requires deliberate human authorship is appropriate. You are encoding what the system is *for*.

**The bedrock is your answer to the regress problem.** Every Evaluator needs something to evaluate against. The bedrock contains those ground truths. They don't need justification from within the system — they are the axioms.

---

## Layer 1: The Seed Specification (Structured, Human-Authored)

Above bedrock, define the initial primitive graph as a declarative spec — not as code, but as a human-readable document:

```yaml
agent_class: coding_assistant
version: 0.1.0

goal: "Help a developer navigate, understand, and modify a codebase"

initial_tools:
  - read_file
  - search_codebase
  - run_tests
  - write_file

evaluators:
  primary:   test_pass_rate
  secondary: type_check_clean

memory:
  working:   in_context_scratch
  retrieval: codebase_rag

governor:
  max_depth:    4
  cost_ceiling: $0.50
  time_budget:  120s
  on_exceed:    SUMMARIZE

type_level: 2
```

This is the seed. It is minimal, human-readable, and auditable. The system starts here. Everything generatable grows from this specification.

---

## Layer 2: The Bootstrap Agent (Minimal, Constrained)

The first agent that actually runs is the most constrained in the system. Its only job:

> Inspect the problem at hand and propose what primitives to generate.

It does **not** generate them directly. It returns a proposal that a human (or the Governor) approves before generation happens:

```
Human spec
    ↓
Bootstrap Agent (depth=1, no tools, single pass)
    ↓
Proposed primitive graph spec
    ↓
Human or Governor approves
    ↓
Generator instantiates approved primitives
```

The Bootstrap Agent has a narrow scope: produce a primitive graph spec, nothing else. This makes it much more controllable than a general agent prompt.

---

## Yes, This Is Prompt Engineering

And that's correct. The bootstrap is prompt engineering. Framing it as a problem undersells what's happening: the bootstrap is the **constitution** of your system. Constitutions are written by humans deliberately. They should not be generated.

The difference from naive prompt engineering:
- The prompt has a **formal output spec** (a primitive graph YAML, not free text)
- The output is **validated** before use (type checker, human review)
- The prompt is **versioned** (it's a design artifact, not a chat message)
- The system can **evolve away from it** (via graduation) without changing it

---

## The Evaluation Ground Truth Problem

The deepest bootstrap challenge: Evaluators need ground truth. Generators of Evaluators need even more ground truth. This regress must bottom out.

The bottom is the bedrock: human-specified criteria that are never generated:

```
For a coding agent:
  Ground truth: tests pass, types check, no regressions
  These are not discovered — they are declared by the human

For a design agent:
  Ground truth: accessibility score, design system compliance
  These are declared — subjective quality is not ground truth at this layer
```

The generated periphery can *refine* how these ground truths are measured. It cannot *replace* the ground truths themselves. If the system can redefine what "good" means, it has no anchor.

---

## Practical Bootstrap Sequence

For a new coding agent system:

```
1. Write bedrock.yaml
   - Define: what the agent must never do
   - Define: what always triggers halt
   - Define: cost limits you're comfortable with

2. Write seed.yaml
   - Define: goal, initial tools, evaluators, governor defaults

3. Run Bootstrap Agent on seed.yaml + target codebase
   - Review proposed primitive graph
   - Approve or modify

4. Run first instance with max_depth=1
   - Observe call_trace
   - Check crystallization candidates after 5-10 runs

5. Expand depth and capability incrementally
   - Raise max_depth only when you have observability to justify it
```

The temptation is to start with the fully capable system. The discipline is to start minimal and let the instance behavior tell you what to add.
