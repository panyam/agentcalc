# 12 — Design Principles: Avoiding Agent Overreach

## The Core Failure Mode

The failure mode isn't agents that do bad things. It's systems where **everything is an agent**, every decision spawns an LLM call, costs spiral, and behavior becomes opaque. Systems that *feel* powerful but can't be reasoned about or debugged.

These principles are guardrails against that.

---

## Principle 1: The Escalation Rule

Start with the dumbest primitive that could work. Only escalate when it fails.

```
deterministic code
    → rule-based classifier
        → small LLM (single call, no tools)
            → large LLM (single call, no tools)
                → agent (LLM + tools, one depth level)
                    → multi-agent system
```

Never start at the right end of this chain. Before reaching for an agent, ask: *"Could I write this as an if statement, a regex, or a classifier trained on 50 examples?"* Most things can.

---

## Principle 2: Agents for Uncertainty, Code for Certainty

Agents are for when the branching logic is too complex or unknown to hardcode. Code is for when you can express the decision function explicitly.

The test: *"Can I write the decision function right now, even if it's ugly?"*
- If yes: write the code.
- If no: use an agent, but write the code as soon as you can.

A crystallized agent (file 10) is the ideal — you started with an agent for flexibility, then extracted the code once the behavior stabilized.

---

## Principle 3: Count Your LLM Calls

Every agent spawn increments a counter you are actively watching. Establish a budget per task before building:

```
Simple task:          ≤ 3 LLM calls
Moderate task:        ≤ 10 LLM calls
Complex task:         ≤ 25 LLM calls
Research pipeline:    ≤ 50 LLM calls, with human checkpoints
```

If you hit your budget mid-task, that is signal: either the task decomposition is wrong, or the task genuinely requires more — but you should know which.

---

## Principle 4: Pipelines over Meshes

A pipeline (A→B→C) is auditable. A mesh is not. Default to pipeline topology (see file 09). Meshes are allowed but must be **explicit and human-approved**, not the default structure that emerges from lazy design.

Rule of thumb: if you can't draw the data flow as a DAG in 5 minutes, the architecture needs to be redesigned before any code is written.

---

## Principle 5: Evaluation Primitives Before Evaluation Agents

Before adding an agent to check another agent's work, ask if a deterministic evaluator can do the job:

```
Does the code compile?          → deterministic check
Do the tests pass?              → deterministic check
Is the JSON schema valid?       → deterministic check
Is this diff reviewable?        → LLM evaluator (single call, no tools)
Is this approach architecturally sound? → agent evaluator (tools needed)
```

Agent-checks-agent is sometimes right. It's often overkill. Work down this list before reaching for an evaluator agent.

---

## Principle 6: The Flat Tax

Every agent in the system must justify its existence against a flat cost budget per task. If adding an agent doesn't measurably improve output quality enough to justify its marginal cost, remove it.

This sounds obvious. In practice, agents accumulate because they *feel* more capable. Run the measurement.

---

## Principle 7: Irreversible Actions Require Extra Gates

Any action that cannot be undone needs at least one additional validation step before execution — regardless of how confident the agent is:

```
Irreversible actions:
  send_email, deploy_to_production, delete_records,
  publish_post, make_payment, merge_branch

Required before execution:
  1. Explicit validator primitive (not just agent confidence)
  2. Human-in-loop checkpoint at high stakes, OR
  3. Dry-run with diff shown to agent for self-check
```

The agent's confidence is not sufficient justification for irreversible action. Confidence is an estimate; validators check facts.

---

## Principle 8: Observable by Default

If you cannot reconstruct exactly what happened in a failed run, the system is not production-ready. Required for every deployed agent system:

- `call_trace` in every Governor envelope (which primitives ran, in order)
- `EventBus.replay` available for every run
- Instance episodic log retained for at least one session
- Hot-swaps logged with before/after primitive IDs and reasons

Observability is not a feature to add later. It is a design constraint from the start.

---

## Principle 9: The Stable Core Principle

Summarizes everything above:

> The system needs a **stable core** it cannot modify, surrounding a **generatable periphery** it can evolve freely.

The bedrock layer, evaluators, resource governor, and event bus classifier are the stable core. They are human-authored, never generated, never swapped. The tools, memory strategies, planner selection, and agent specializations are the generatable periphery. Only the periphery evolves.

When you're unsure which layer something belongs to, ask: *"If this is wrong, how bad is it?"* If the answer is "catastrophic," it belongs in the stable core.

---

## Anti-Patterns Summary

| Anti-pattern | Why it's bad | Fix |
|---|---|---|
| Everything is an agent | Opaque, expensive, undebuggable | Apply escalation rule |
| Agents with no Governor | Unbounded cost and depth | Always wrap with Governor |
| Mesh topology by default | Cycles, ownership ambiguity | Design DAG first |
| No observability | Can't debug failures | call_trace + event replay |
| Agents checking agents checking agents | Cost explosion | Use deterministic evaluators first |
| Swapping bedrock primitives | Loss of safety anchor | Bedrock is never swappable |
| LLM assigns its own criticality | Safety hole | LLM proposes only, orchestrator decides |
