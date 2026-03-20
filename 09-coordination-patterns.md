# 09 — Coordination Patterns: Pipelines over Meshes

## Why Pipelines Win

Meshes feel flexible. They are actually traps.

```
Mesh topology (avoid):
  A ←→ B
  ↕     ↕
  C ←→ D

Problems:
  Cycles:             A influences B influences A...
  Ownership ambiguity: who is the authority on X?
  Cascading failures:  B's bad output corrupts C and D simultaneously
  Debugging:           any bad output could have come from anywhere
```

The discipline: **default to pipelines, make meshes explicit and justified**.

---

## The Design Tool: Dependency Analysis

If you can express a task as a DAG (no cycles), you get a pipeline for free.

The question for each pair of agents (A, B): *does A's output feed B, or does B's output feed A, or neither?*

```
Example: "Research and write a technical blog post"

Dependencies:
  Research    ← depends on: topic only
  Outline     ← depends on: Research
  Draft       ← depends on: Outline
  Fact-check  ← depends on: Draft + Research
  Edit        ← depends on: Draft + Fact-check
  Publish     ← depends on: Edit

DAG:
  Topic → Research → Outline → Draft → Fact-check → Edit → Publish
                                    ↗ (Research used again)
```

No cycles → this is a pipeline.

---

## Three Pipeline Topologies

**Linear** — each stage consumes the previous
```
A → B → C → D
```
Best for: sequential, well-understood tasks. Easiest to debug.

**Fork-Join** — parallel branches, then merge
```
         ┌→ B ─┐
A → fork │     │→ join → D
         └→ C ─┘
```
Best for: independent subtasks that need synthesis. Still a DAG.

**Layered** — stages share a common read layer
```
Stage 1 → Stage 2 → Stage 3
    ↓          ↓         ↓
        [Shared Read Memory]
```
Each stage writes only to its own output. All read from shared state.  
**Single writer principle:** no stage reads another stage's work-in-progress.

---

## Stage Gates

Every pipeline stage boundary has a gate primitive:

```
StageGate(
  validator:  schema check + semantic check
  on_fail:    RETRY | SEND_BACK | HALT
  on_pass:    forward to next stage
)
```

Gates are cheap and deterministic. They catch bad output before it propagates downstream. Most bugs in pipelines are caught here before they compound.

---

## The Pipeline Designer — Yes, It's an Agent

Designing the pipeline topology is itself a task for an agent — run **once at design time**, not at runtime:

```
PipelineDesigner(task_spec):
  1. Decompose task into subtasks
  2. For each pair (A, B): does A depend on B or vice versa?
  3. Build dependency graph
  4. Check for cycles → flag for human review if found
  5. Topological sort → emit ordered pipeline spec
  6. Assign primitive type to each stage

Output: static YAML/JSON topology, human-reviewable before deployment
```

When the designer finds a cycle, it offers three options:
```
Option 1: Restructure to eliminate cycle (preferred)
Option 2: Merge the two agents (they're probably the same concern)
Option 3: Add an Arbitrator agent between them (controlled, explicit mesh)
```

Meshes are allowed — they just must be **explicit and human-approved**, not emergent defaults.

---

## Multi-Agent Coordination Patterns

### Orchestrator → Workers
```
[Orchestrator]
    ├── [Research Agent]
    ├── [Analysis Agent]
    └── [Writer Agent]
```
One agent is the brain, delegates to specialists. The orchestrator can be thin (routing only) or thick (actively synthesizing results). Prefer thin orchestrators — less reasoning in the orchestrator means fewer orchestrator bugs.

### Critic / Verifier
```
[Generator Agent] → [Critic Agent] → [Revised Output]
```
Generator proposes, critic checks. Massive reliability improvement for correctness-critical tasks. The critic is often cheaper than the generator — it just needs to check, not create.

### Parallel / MapReduce
```
              ┌→ [Agent A] ─┐
[Orchestrator]├→ [Agent B] ─┤→ [Merge] → Result
              └→ [Agent C] ─┘
```
Fan out work, fan back in. Good for large tasks with independent decomposable subtasks. The merge step is usually a deterministic primitive, not another agent.

### Human-in-Loop
```
[Agent] → [Checkpoint] → [Human review] → [Approve/Redirect] → [Agent continues]
```
Required for irreversible actions above a trust threshold. The checkpoint primitive saves full state so the agent can resume cleanly after human input.

---

## Subsumption as a Pipeline Layer

The subsumption architecture (see overview) fits naturally as a **base layer** of any pipeline:

```
Application pipeline (slow, deliberative):
  Agent A → Agent B → Agent C

Below it, always running (fast, reactive):
  Signal(test_failing) → halt_pipeline
  Signal(build_broken) → preempt_all
  Signal(budget_exceeded) → FALLBACK
```

The subsumption layer is bedrock. It can interrupt any pipeline stage at any depth. The pipeline doesn't know it exists — it just finds itself interrupted at a safe boundary.
