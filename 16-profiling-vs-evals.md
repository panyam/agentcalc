# 16 — Profiling vs Evals: A Measurement Taxonomy

## The Critical Asymmetry

The JIT crystallization discussion (file 10) covers one specific use of measurement: identifying stable input→output patterns to replace with deterministic code. But profiling is only one slice of a much larger measurement space.

The asymmetry that matters:

> **Profiling answers:** "Is this behavior stable?"  
> **Evals answer:** "Is this behavior correct?"  
> **Drift detection answers:** "Is this behavior changing?"

These are orthogonal questions. A profiler without evals will crystallize bad behavior. An eval without a profiler finds problems but can't locate where they're cheapest to fix. Drift detection catches neither — it only knows something shifted.

**The correct crystallization pipeline requires both:**

```
Profiler:   "this decision fires 900 times, very stable pattern"
Eval:       "and it's correct 97% of the time"
Crystallizer: "NOW we can emit deterministic code"
```

Stability alone is never sufficient justification for crystallization.

---

## The Full Measurement Stack

```
Type             Question                    When              Cost
─────────────────────────────────────────────────────────────────
Unit eval        Does this primitive work?   Dev time          low
Integration eval Does this pipeline work?    Dev + staging     medium
Behavioral prof  Is this stable enough?      Continuous        low
Online eval      Is this correct right now?  Production        varies
Drift detection  Is behavior changing?       Continuous        low
Offline eval     What went wrong in run X?   Post-failure      high
A/B eval         Is version B better?        Controlled        high
Crystallization  Can we replace this?        Scheduled         high
```

---

## Behavioral Profiling (The JIT Layer)

Already covered in file 10, but to place it in context:

**What it measures:** frequency, stability, and confidence of input→output mappings across runs.

**Primitive form:**
```
AgentProfiler:
  observes:    every primitive invocation
  tracks:      (input_hash → output) distributions
  outputs:     crystallization_candidates with confidence scores
  runs:        continuously, offline, never in hot path
  cost:        low (just logging + statistical analysis)
```

**What it does NOT tell you:** whether outputs are correct. An agent that consistently gives the wrong answer will look like an excellent crystallization candidate to the profiler.

---

## Continuous Evals

Unlike profiling (which measures stability), evals measure quality against ground truth. They exist at every level of the primitive hierarchy.

### Primitive-Level Evals
Does this individual tool/primitive behave correctly?

```
Eval(search_tool):
  test_cases:   [(query, expected_result_shape), ...]
  metrics:      precision, recall, latency p99
  runs:         every deploy, on schedule
  threshold:    precision > 0.9 → pass
```

These are essentially unit tests. Fast, cheap, high signal for regressions.

### Agent-Level Evals
Does this agent accomplish its stated goal?

```
Eval(CodeReviewAgent):
  test_cases:   [(PR_diff, expected_issues_found), ...]
  metrics:      issues_caught_rate, false_positive_rate, cost_per_review
  ground_truth: human-labeled review set
  runs:         nightly or on class change
```

More expensive. Require labeled ground truth. Essential before any class promotion or graduation.

### System-Level Evals
Does the multi-agent pipeline achieve the overall objective?

```
Eval(ResearchWritePipeline):
  test_cases:   [(topic, quality_rubric), ...]
  metrics:      factual_accuracy, coherence, completeness
  ground_truth: human judgment (expensive) or LLM-as-judge (cheap proxy)
  runs:         weekly or on architecture change
```

---

## The Multi-Level Tension

Evals at different levels can conflict. This is one of the hardest debugging problems in agent systems:

```
Tool eval:    search returns valid results         → PASS
Agent eval:   agent misreads results, wrong conclusion → FAIL
System eval:  another agent corrects it downstream → PASS
```

Where do you fix? The tool is fine. The system output is fine. But there's a latent agent-level bug that will surface differently in different system configurations.

**Rule:** fix at the lowest level where the failure occurs. If agent A fails its agent-level eval, fix agent A — don't rely on agent B to compensate. Compensation masking is how bugs become invisible until they compound.

---

## Drift Detection

Different from both profiling and evals. Not "is it stable?" or "is it correct?" but "is behavior *changing over time?*"

Sources of drift in agent systems:
- **Model drift:** underlying model weights change (API provider updates)
- **Data drift:** retrieval corpus changes (codebase grows, docs updated)
- **Input drift:** user query distribution shifts over time
- **World drift:** the environment the agent acts in changes

**Drift as a primitive:**
```
DriftDetector:
  monitors:    behavioral distribution over a rolling window
  baseline:    distribution from a known-good period
  test:        statistical distance (KL divergence, PSI, etc.)
  threshold:   configurable per metric
  on_detect:   ALERT | TRIGGER_EVAL | SUSPEND_CRYSTALLIZATION
  cost:        low (statistics over logged data)
```

`SUSPEND_CRYSTALLIZATION` is the important action — if drift is detected, halt any pending crystallization until evals confirm the new behavior is still correct.

---

## Online Evals: The Production Safety Net

Offline evals catch problems after the fact. Online evals catch them during execution — at the cost of added latency and compute.

```
Online eval spectrum:
  Validator (schema check)        → free, always on, catches structural failures
  Deterministic scorer            → cheap, always on, catches measurable failures
  LLM-as-judge (single call)      → moderate cost, spot-check critical outputs
  Human-in-loop                   → expensive, reserved for high-stakes
```

The inline validator IS an online eval — it just happens to be deterministic. The escalation rule from design principles (file 12) applies here too: use the cheapest online eval that catches the failure class you care about.

**Online eval as a primitive:**
```
OnlineEval(
  target:      P,
  eval_fn:     Output → Score,
  threshold:   float,
  sample_rate: float,      ← don't eval every call, just sample
  on_fail:     RETRY | FALLBACK | ESCALATE
) → P   ← wraps target, transparent to caller
```

The `sample_rate` is critical for cost control. You don't need to eval every output — sampling at 10% still catches systematic failures quickly.

---

## The Feedback Loops

There are three distinct feedback loops, each at a different timescale:

```
Fast loop (seconds to minutes):
  Online eval → immediate retry / fallback
  Subsumption signals → preempt
  Validators → gate pipeline stages

Medium loop (hours to days):
  Offline evals → flag regressions
  Drift detection → alert on behavior shift
  Profiler → update crystallization candidates

Slow loop (days to weeks):
  Crystallization → promote stable+correct to deterministic code
  Graduation → promote instance divergences to class updates
  Class versioning → deploy updated agent classes
```

These loops must not be confused. A fast-loop online eval should never trigger a class graduation — that requires the slow loop with human review. A drift alert should suspend crystallization but not roll back deployed agents automatically.

---

## Comparison Table: Profiling vs Evals vs Drift

| Dimension | Profiling | Evals | Drift Detection |
|---|---|---|---|
| Core question | How stable? | How correct? | How changed? |
| Ground truth needed | No | Yes | Baseline distribution |
| Runs when | Continuously | Scheduled / on deploy | Continuously |
| Triggers | Crystallization | Class graduation, rollback | Eval run, suspension |
| Timescale | Across many runs | Batch | Rolling window |
| Cost | Low | Medium-high | Low |
| Human involvement | No | Yes (ground truth labeling) | On alert |
| Latency sensitivity | None (offline) | None (offline) | None (offline) |
