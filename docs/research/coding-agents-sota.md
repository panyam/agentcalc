# The State of the Art in Coding Agents, and What It Means for Chakra

**Date:** 2026-07-09
**Purpose:** Survey where coding-agent architecture, benchmarks, memory, and training stand as of mid-2026, then map those findings onto the Chakra kernel (Primitive / Message / Envelope / Store / Gate / Registry).

## Method and how to read the confidence markers

This report was produced by a fan-out research harness: one question decomposed into five search angles, five parallel web searches, 24 sources fetched and mined for falsifiable claims, then 108 extracted claims ranked and the top 25 put through 3-vote adversarial verification (a claim needed 2 of 3 skeptical voters to refute it to be killed). 23 claims survived, 2 were refuted.

Two tiers of evidence appear below:

- **[V]**: part of the adversarially-verified core (23 claims). Each has a primary-source quote and survived the 3-vote panel. Cited with `[n]` and the vote (e.g. `3-0`).
- **[S]**: supporting context from a primary source that was fetched but whose specific claim fell outside the verified top-25. Reliable sources, but treat the specific numbers as "reported by the source," not "independently vote-checked in this run."

The verified core is dense on harness architecture, benchmark validity, and the scaffold-minimalism debate. It is thinner on the closed-source product internals (Claude Code, Cursor, Devin) and on the training papers, because the adversarial panel favored claims with clean primary-source backing and killed marketing-flavored ones. Where a thread leans on [S] sources, that is called out.

---

## Executive summary

Four findings matter most for anyone building an agent calculus.

1. **The field has converged on a small set of composable primitives, not a grand architecture.** A source-code taxonomy of 13 open-source scaffolds found the same four capability categories (read, search, edit, execute) in every LLM-driven agent despite tool counts ranging from 0 to 37, and five loop primitives (ReAct, generate-test-repair, plan-execute, multi-attempt retry, tree search) that 11 of 13 agents layer in combinations rather than picking one [1]. Convergence is real and it is at the primitive layer.

2. **Independent systems keep re-deriving the same kernel.** The OpenHands Software Agent SDK models the agent as an event-sourced loop: append-only event log, one mutable state object, immutable components validated at construction, tools as a uniform Action-Execution-Observation triad, and sub-agents and delegation implemented as compositions of the existing tool primitive with no changes to the core [4][5][6]. This is a different vocabulary for the bet Chakra makes.

3. **A thin scaffold plus a strong model beats an elaborate scaffold.** A 100-line, bash-only agent that does not even use the tool-calling interface scores over 74% on SWE-bench Verified [21][22][23]. Agentless, a fixed non-agentic pipeline, beat autonomous agents in 2024 [17][18][19]. The caveat that matters: at the 2026 frontier, agentic harnesses paired with the strongest models now surpass agentless again, so the lesson is "minimal scaffold around a strong model," not "no scaffold."

4. **Capability is migrating into the weights, and the benchmarks that measure it are partly contaminated.** RL on execution feedback and on software-evolution data (SWE-RL, RLEF, SWE-Gym) trains agentic coding into the model directly [24]. Meanwhile SWE-bench Verified shows heavy memorization signal (76% buggy-file identification from issue text alone, dropping to under 53% on comparable off-benchmark repos) [10][12], and an automated audit found flaws in over 25% of tasks across 168 benchmarks, with filtering shifting model rankings by around 10 points [14][15].

The through-line for Chakra: the field's empirical evidence points toward a thin composition-and-trust layer over increasingly capable models, which is the bet the kernel already makes. The sharpest open gaps the research surfaces for Chakra are prompt-cache economics and the event-sourcing-vs-mutable-Store question, covered in the mapping section.

---

## Thread 1: Agent loop and harness architecture

### The primitive layer has converged

The most useful architectural source is "Inside the Scaffold: A Source-Code Taxonomy of Coding Agent Architectures" (arXiv:2604.03515, April 2026) [1], the first implementation-level comparative study of coding agents. It reads the actual source of 13 open-source scaffolds at pinned commits. Three findings are load-bearing:

- **Four capability categories, universally.** Read, search, edit, execute appear across all LLM-driven agents, even though registered tool counts range from 0 (Aider) to 37 action classes (Moatless Tools). Low-tool agents reach coverage by composition (mini-swe-agent's single bash tool), high-tool agents by decomposing categories into fine-grained ops. The convergence is masked by implementation granularity but it is there. **[V, 3-0]** [1]

- **Five composable loop primitives.** ReAct, generate-test-repair, plan-execute, multi-attempt retry, and tree search function as building blocks, and 11 of 13 agents compose several rather than relying on a single control structure. **[V, 3-0]** [1]

- **The loop driver is the deepest distinction.** The paper calls "what drives the loop" the most fundamental architectural axis: user-driven (Aider, where the LLM has zero callable tools), scaffold-driven (Agentless, AutoCodeRover, where the scaffold sequences and calls the LLM at fixed points), or LLM-driven (nine of thirteen, where the model decides the next action). **[V, 3-0]** [1]

### One production SDK, re-derived from first principles

The OpenHands Software Agent SDK (arXiv:2511.03690, Nov 2025) [4][5][6] is the clearest published architecture and independently arrives at a kernel that rhymes with Chakra:

- **Event-sourced state.** All interactions are immutable events appended to an append-only `EventLog`. A single `ConversationState` object is the only mutable state in the system. Agents, tools, LLMs, and condensers are immutable Pydantic models validated at construction. The agent is a `step(state)` function that maps event history to the next action, which the runtime executes into an observation. **[V, 3-0]** [4][7]

- **Uniform tool contract.** Tools follow a uniform Action-Execution-Observation pattern. MCP tools are auto-translated from their JSON Schemas into Action models and surfaced as structured Observations. Tools are lightweight specification objects (registered name plus JSON-serializable parameters), which lets a tool spec cross process or network boundaries as pure JSON and gives a uniform interface for local and remote execution. **[V, 3-0]** [5]

- **Sub-agents and delegation are compositions, not new machinery.** Sub-agents operate as independent conversations that inherit the parent's model configuration and workspace context, giving parallelism and context isolation with no change to the core SDK. Delegation in the earlier ICLR-2025 OpenHands is a single `AgentDelegateAction` (for example a generalist `CodeActAgent` handing web browsing to a specialist `BrowsingAgent`), tracked as metadata in shared state rather than through a separate orchestration layer. **[V, 3-0 and 2-1]** [4][8][9]

- **CodeAct action space.** The agent acts by executing code rather than emitting JSON function calls: `IPythonRunCellAction` and `CmdRunAction` run arbitrary Python and bash inside a Docker-sandboxed Linux OS, and skills are Python utilities invoked through the IPython action rather than registered as first-class actions. **[V, 3-0]** [7]

### Claude Code and Codex CLI (supporting)

Two 2026 analyses map the closed-source terminal agents [2][25]. They describe Claude Code and Codex CLI as LLM-driven single-loop harnesses with a small fixed tool set, a planning or todo scratchpad, sub-agent spawning for context isolation, and aggressive context management. These are secondary and blog-tier sources, so treat the specifics as directional. **[S]** [2][25]. The architecturally verified point stands on its own: the terminal agents sit in the same "LLM-driven, four capability categories, compose loop primitives" bucket the taxonomy paper identifies.

---

## Thread 2: Reasoning patterns and control flow

### The scaffold-minimalism result

The strongest empirical finding in this thread is that stripping the scaffold and foregrounding the model works surprisingly well.

- **mini-swe-agent** (Princeton/Stanford SWE-bench team): roughly 100 lines of Python for the agent class, bash as the only tool, and it does not use the tool-calling interface of the LLMs at all (actions are parsed from triple-backtick blocks). It scores over 74% on SWE-bench Verified. **[V, 3-0 each]** [21][22]. It is explicitly framed as a baseline that puts "the language model (rather than the agent scaffold) in the middle of our attention." **[V, 3-0]** [23]. The load-bearing caveat, stated by the project itself: the 74% figure requires a frontier model (Gemini 3 Pro), so the harness alone does not produce the number.

- **Agentless** (FSE 2025) replaces autonomous control flow with a fixed three-phase pipeline (localization, repair, patch validation) in which the LLM never decides future actions or operates complex tools. **[V, 3-0]** [17]. On SWE-bench Lite it resolved 32.00% (96 of 300) at low cost, the best among open-source agents at publication. **[V, 3-0]** [18]. The paper's thesis is that a simple, interpretable, non-agentic technique can match or beat complex autonomous agents, challenging the assumption that elaborate scaffolding is necessary. **[V, 3-0]** [19].

### The nuance that keeps this honest

The Agentless result was 2024. By mid-2026, agentic harnesses paired with frontier models exceed agentless approaches at the top of the leaderboard (the verifier flagged Kimi-Dev and 2026 SWE-bench Verified leaderboards as counter-evidence to the durability, not the historical accuracy, of the agentless claim) [19]. So the defensible synthesis is not "scaffolding is useless." It is: the marginal value of scaffold complexity shrinks as the model gets stronger, a minimal loop is a strong baseline, and complexity has to earn its place against that baseline. The taxonomy finding that 11 of 13 agents compose multiple loop primitives [1] says the same thing from the other side: what wins is composition of simple primitives, not a monolithic clever architecture.

This is the single most important external result for Chakra, and it is discussed in the mapping section.

---

## Thread 3: Benchmarks and capabilities

### SWE-bench is contaminated, and the contamination is measurable

- **Memorization signal.** "The SWE-Bench Illusion" (arXiv:2506.12286, NeurIPS 2025) [10][11][12] shows state-of-the-art models identify the correct buggy file path for SWE-bench Verified issues up to 76% of the time using only the issue text and repo name, with no access to repo structure, code, or metadata. **[V, 3-0]** [10]. Given only function names and issue descriptions, models reproduce ground-truth functions with up to 34.9% 5-gram overlap. **[V, 3-0]** [11]. On equally popular repos outside the benchmark (jupyter, celery, aiohttp, scipy, numpy, pytorch, pandas) the same diagnostic accuracy is uniformly under 53%, versus 60-76% on SWE-bench Verified. **[V, 3-0]** [12]. The benchmark advantage does not generalize to comparably exposed code.

- **Widespread task defects.** The Auto Benchmark Audit framework (arXiv:2605.26079, May 2026) ran an agentic auditor over 168 benchmarks and 34,285 tasks across nine domains and found critical issues (ambiguous task design, execution-environment conflicts, or incorrect ground truths) in over 25.7% of tasks. **[V, 3-0]** [14]. Filtering the flawed tasks shifts model rankings and raises average performance on SWE-bench Verified by 9.9% and on Terminal-Bench 2 by 9.6%. **[V, 3-0]** [15].

### The response: live, contamination-resistant benchmarks

SWE-bench-Live [16] is positioned as a contamination-resistant successor: an automatically-updating, multi-language, multi-OS task set built from a curation pipeline that includes only issues created after January 2024 and refreshes with new GitHub issues (updated through August 2025 at capture, on a stated monthly cadence). **[V, 2-1 and 3-0]** [16]. The residual caveat, flagged by the verifier: correctness still leans on automated LLM-judge proxies that approximate gold labels, so "contamination-free" is design intent, not proof.

### What agents can and cannot do

The verified evidence supports a specific, unromantic read. Frontier agents are strong at localized bug fixes on well-represented repositories, partly because they have seen them. They degrade on comparable but unseen code, which is the honest measure of generalization. Known failure modes named across sources: long-horizon multi-turn tasks, multi-file refactors, and flaky verification (tests that pass or fail nondeterministically, which both inflates and deflates measured capability). The Terminal-Bench headline numbers were the two claims the panel refused (see appendix), so treat specific command-line-agent percentages as unsettled.

---

## Thread 4: Product landscape

The verified core does not contain reliable claims about the internal control flow of the closed-source products, because those claims came from marketing or blog sources and the adversarial panel discounted them. What the verified architecture evidence supports is a taxonomy by form factor and loop driver, not a feature comparison.

- **Form factor.** Terminal agents (Claude Code, Codex CLI, Gemini CLI, mini-swe-agent), IDE-embedded agents (Cursor), and autonomous cloud agents (Devin, Google Jules) draw the human-in-the-loop boundary differently. Terminal and IDE agents keep the human close and interrupt-capable; cloud agents run long-horizon and report back.

- **Loop driver, the deeper cut.** Per the taxonomy [1], the meaningful distinction is user-driven vs scaffold-driven vs LLM-driven, which cuts across form factor. Aider (user-driven, zero LLM tools) and mini-swe-agent (LLM-driven, one bash tool) are both terminal agents but architecturally opposite.

The takeaway for a calculus is that "product" is the wrong unit of analysis. The right units are the loop driver, the capability set, and the loop primitives composed, all of which are architecture, not branding.

---

## Thread 5: Memory systems and token optimization

This thread rests mostly on [S] primary sources (the specific claims were fetched but fell outside the verified top-25). The sources themselves are first-party and strong.

- **Context engineering as a discipline.** Anthropic's engineering writeups [20][22-context] frame long-horizon agents around a budget: keep the working context small and high-signal, compact or summarize history when it grows, and clear stale tool results. The Claude platform exposes context editing and a memory tool for this. **[S]** [20].

- **Sub-agent context isolation.** The strongest cross-validated point bridges to Thread 1. OpenHands implements sub-agents as independent conversations with isolated context, composed from the existing tool primitive with no core change [6]. This is the same move Claude Code's sub-agents make: isolate a subtask's context so it does not pollute the parent's window. Context isolation is achieved by composition, not by a memory subsystem. **[V, 3-0]** [6].

- **External agentic memory.** Mem0 (arXiv:2504.19413) [23-mem0] and the MemGPT/Letta and A-MEM lines treat memory as an external store with extraction, consolidation, and retrieval, rather than stuffing everything in context. These are Primitives-plus-Store designs in Chakra terms. **[S]** [23-mem0].

- **Prompt-cache economics.** "Don't Break the Cache: Evaluating Prompt Caching for Long-Horizon Agentic Tasks" (arXiv:2601.06007, Jan 2026) [24-cache] is the most Chakra-relevant memory source. Long-horizon agent cost is dominated by KV-cache reuse: an append-only, prefix-stable context is cheap because the cache hits, and any edit to earlier context invalidates the cached prefix and forces recomputation. **[S]** [24-cache]. This is why OpenHands' append-only event log is not just tidy, it is economically load-bearing. It is also a constraint the Chakra kernel is currently silent on (mapping section).

---

## Thread 6: Training (research-paper depth)

The question is how much coding-agent capability lives in the model versus the harness. The evidence says the model's share is large and growing, and it is being trained in deliberately.

### RL from verifiable and execution rewards

- **SWE-RL** (Meta, arXiv:2502.18449, Feb 2025, NeurIPS 2025) trains LLMs on software-evolution data (code snapshots, changes, issues, pull requests) using a lightweight rule-based reward defined as the similarity score between the ground-truth patch and the model's solution, rather than a learned reward model. **[V, 3-0]** [24]. This is RL from verifiable rewards (RLVR) applied to software: the reward is a cheap deterministic function of an oracle, not a trained critic.

- **RLEF** (Meta, arXiv:2410.02089, Oct 2024) [25-rlef] grounds code LLMs in execution feedback with RL: the model iterates against real test execution during training, learning to use error output to repair its own code. **[S]** [25-rlef].

- **SWE-Gym** (arXiv:2412.21139, ICML 2025) [26-swegym] provides a training environment for SE agents and, notably, trains both agents and verifiers, using the learned verifier for inference-time reranking of candidate solutions. **[S]** [26-swegym]. This is the generator-verifier pattern pushed into training.

- **Long-context multi-turn agent RL** (arXiv:2508.03501, Aug 2025) [27-longrl] trains SWE agents with RL over long, multi-turn horizons, targeting exactly the failure mode (long-horizon tasks) that the benchmark thread flagged. **[S]** [27-longrl].

### The model-vs-scaffold verdict

Three verified data points triangulate the answer:

1. The same 100-line harness scores 65% with a mid-2025 model and over 74% with Gemini 3 Pro [21]. The harness was constant. The model moved the number.
2. mini-swe-agent is deliberately built so that its score measures the model, not the scaffold [23]. The community treats a thin harness as a model probe.
3. Labs are spending training compute (SWE-RL, RLEF, SWE-Gym, long-context RL) specifically to put agentic coding behavior into the weights [24][25-rlef][26-swegym][27-longrl].

The synthesis: harness improvements give diminishing returns as models get stronger, and the trend line is that verification-in-the-loop, tool use, and multi-turn planning are increasingly fine-tuned in rather than scaffolded on. The harness's durable job is not to be smart. It is to compose primitives, enforce trust boundaries, and manage the context budget, which are exactly the jobs that do not transfer into the weights.

---

## Mapping to Chakra

Chakra bets on a four-concept kernel (Primitive, Message, Envelope, Store, with Gate and Registry as the composition and validation surface) and the thesis that "the for loop IS the agent." The research is, on balance, strong external validation of that bet, with two concrete gaps worth acting on.

### Where the field validates Chakra

| SOTA finding (source) | Chakra construct | Verdict |
|---|---|---|
| Four capability categories: read, search, edit, execute, universal across agents [1] | The four leaf Primitives named in the handoff doc (ReadFile, SearchCode, EditFile, RunTests) | Direct hit. Chakra already picked the convergent set. |
| Five loop primitives composed, not one architecture; 11/13 compose [1] | "The for loop IS the agent"; each pattern is a for loop with a different body | Strong validation. Composition of simple primitives is the empirical winner. |
| Agent = `step(state)` mapping event history to next action [7] | `Primitive.Run(ctx, in Message, env Envelope) (Message, error)` | Convergent contract. Chakra's Run is the step function; Message in/out is Action/Observation. |
| Uniform Action-Execution-Observation triad [5] | Message (Action) into Run, Message (Observation) out | Convergent. Chakra's universal Message type is the same uniformity. |
| Sub-agents = independent conversations, composed from the tool primitive, no core change [6] | Agent IS a Primitive (closure); ScopedStore gives write isolation | Direct hit. This is Chakra's closure property, independently re-derived in a production SDK. |
| Delegation = single action + metadata in shared state, no orchestration layer [8][9] | `Registry.Connect` / `Invoke`; no orchestrator primitive | Direct hit. Chakra's refusal to add an orchestrator matches OpenHands' refusal to add one. |
| Thin scaffold + strong model beats elaborate scaffold [17][21][23] | The Anti-Elaboration Rule; the sub-500-line kernel | Strong validation of the whole design philosophy. |
| Capability migrating into RL-trained weights [24][25-rlef][26-swegym] | "Primitives are model-agnostic; the LLM is one Primitive, not the default" | Supports the bet. If smartness lives in the model, the kernel should stay dumb. |
| Generator-verifier trained and used for reranking [26-swegym] | Critic/Verifier as a for loop; Gate / DeltaGate as validation | Convergent. Verification is a first-class pattern, and Chakra already has the hook. |

The closure property deserves emphasis. The single most striking result is that OpenHands, a production SDK built by a different team for different reasons, independently landed on: one shared state object, immutable components, a uniform action/observation tool contract, and sub-agents and delegation as compositions of existing primitives rather than new machinery [4][5][6][8]. That is Chakra's Store + Primitive + Message + closure, in a different language. When two designs converge from opposite starting points, the shared core is likely real.

### Where the research pushes back or exposes a gap

**Gap 1: prompt-cache economics are unmodeled, and they are load-bearing.** The "Don't Break the Cache" result [24-cache] shows long-horizon agent cost is dominated by KV-cache reuse: an append-only, prefix-stable context is cheap, and editing earlier context invalidates the cached prefix. OpenHands' append-only `EventLog` is cache-friendly by construction. Chakra's `Store` is mutable (`Set`, `Update`), and `Store.Set` on a key that feeds an LLM-Primitive's prompt could silently invalidate that primitive's cache on every iteration. The kernel says nothing about this. This is not an argument to add to the kernel. It is an argument that the reference LLM-Primitive and the recommended Store usage should treat context as append-only where it feeds a model, and that the docs should name cache-stability as an architectural property. Worth a design note, and possibly a `Meta` hint or a documented convention.

**Gap 2: event-sourcing versus mutable Store.** OpenHands makes an append-only event log the single source of truth and derives everything else from it, which buys deterministic replay, auditability, and cache-friendliness [4][7]. Chakra splits state into a mutable `Store` plus an append-only `Trace []string` (audit only). These are not incompatible: a `Store` could be implemented as a projection over an event log, and the `Trace` is already the append-only spine. But the kernel does not encourage that implementation, and the mutable-Store default is at mild tension with the three properties event-sourcing buys. This is a genuine design question rather than a defect, and it is the one place the research suggests a fork worth deciding deliberately (see below).

**Gap 3: verification is where agents fail, and the design attention it gets should reflect that.** The benchmark thread shows flaky verification and contamination are central failure modes [12][14][15], and the training thread shows labs are training verifiers explicitly [26-swegym]. Chakra's `Gate` / `DeltaGate` is the right layer for this, and the `Reversible` flag in `Meta` is the right kind of safety primitive. The research argues for documenting the generator-verifier loop as a first-class userland pattern (a for loop whose body is generate-then-Gate) and for treating verifier Primitives as a named category, without adding anything to the kernel.

**Nuance on model-agnosticism.** Chakra's "primitives are model-agnostic" stance is architecturally correct: the LLM is one Primitive among many (NER models, regex extractors, SQL queries). But the research adds an asterisk. The system's capability is dominated by which model backs the LLM-Primitive [21], and that capability is increasingly trained in [24]. So the convention of putting cost, speed, and determinism in `Meta.Description` to steer the composing agent toward cheaper tools is well-aimed, and it is worth extending the convention to record which model a given LLM-Primitive wraps, because that is now the dominant capability variable.

### The one philosophical confirmation

Chakra's design docs worry, at length, about the failure mode of "local justification, global complexity" (18 documents of speculative architecture). The external evidence is that the field walked into exactly that trap and is climbing out of it. The taxonomy paper's finding that agents converge on four capabilities and five composable primitives [1], the Agentless and mini-swe-agent results that a minimal loop is a strong baseline [17][21], and OpenHands' explicit principle that "advanced orchestration requires no modification to the core framework" [6] are the same lesson Chakra states as the Anti-Elaboration Rule. The kernel's discipline is not idiosyncratic. It is where the empirical frontier is heading.

### Suggested next decisions for Chakra

1. **Decide the event-sourcing question deliberately.** Either document that `Store` may be a projection over the `Trace`/event log (and make the reference implementation demonstrate it), or state explicitly why mutable-Store is preferred and accept the loss of free replay. Do not leave it implicit.
2. **Name cache-stability as a property.** Add a docs note (and possibly a `Meta` hint) that context feeding an LLM-Primitive should be append-only, and that Store mutation on prompt-feeding keys is a cache hazard.
3. **Write the generator-verifier example.** The `examples/critic` sketch already gestures at this. Promote it with the framing that verification is where agents fail and Gate/DeltaGate is the trust boundary.
4. **Record the wrapped model in LLM-Primitive metadata.** Capability now tracks the model more than the harness. Make that visible in `Meta`.

None of these adds a fifth kernel concept. All four are Gates, Primitives, conventions, or docs, which is the test the kernel already sets for itself.

---

## Appendix A: refuted claims (transparency)

Two claims were killed by the adversarial panel (1-2 votes to survive, i.e. 2 of 3 voters refuted). Both concern Terminal-Bench specifics from arXiv:2601.11868, and both should be treated as unestablished:

- "On Terminal-Bench (89 tasks), all frontier models score under 65%, with Codex CLI + GPT-5.2 top at 63% and smaller models around 15%." Refuted 1-2. Specific leaderboard numbers did not hold up.
- "For Terminal-Bench performance, the model matters more than the scaffold." Refuted 1-2. Note this is narrower than, and does not undercut, the broadly-supported model-vs-scaffold synthesis in Thread 6, which rests on the verified mini-swe-agent and training evidence rather than on Terminal-Bench.

## Appendix B: method notes and limitations

- 24 sources fetched, 108 claims extracted, top 25 verified, 23 survived. Verification was a 3-vote skeptical panel per claim, biased toward refutation (default-refute on uncertainty).
- The verified core over-represents open-source and academic sources, because closed-source product claims and marketing were discounted by the panel. Product-internals claims (Claude Code, Cursor, Devin control flow) are therefore [S] or absent, not [V].
- Several 2026 arXiv IDs (2601.x, 2604.x, 2605.x, 2606.x) postdate the assistant's training cutoff. Their handling relies on the fetched primary-source text and the verifier panel rather than prior knowledge.
- This is a snapshot dated 2026-07-09. The field moves monthly. Benchmark numbers especially will be stale within a quarter.

---

## References

Verified-core sources [V] and supporting sources [S], with fetch dates.

1. Rombaut, B. *Inside the Scaffold: A Source-Code Taxonomy of Coding Agent Architectures.* arXiv:2604.03515v2, 2026-04-10. https://arxiv.org/html/2604.03515v2 **[V]**
2. *Dive into Claude Code: The Design Space of Today's and Future AI Agent Systems.* arXiv:2604.14228v1, 2026-04-14. https://arxiv.org/html/2604.14228v1 **[S, secondary]**
4. *The OpenHands Software Agent SDK: A Composable and Extensible Foundation for Production Agents.* arXiv:2511.03690v1, 2025-11-05. https://arxiv.org/html/2511.03690v1 **[V]**
5. (same as [4], tool-contract claims) **[V]**
6. (same as [4], sub-agent isolation claims) **[V]**
7. *OpenHands: An Open Platform for AI Software Developers as Generalist Agents.* ICLR 2025 (arXiv:2407.16741). https://proceedings.iclr.cc/paper_files/paper/2025/file/a4b6ad6b48850c0c331d1259fc66a69c-Paper-Conference.pdf **[V]**
8. (same as [7], AgentDelegateAction) **[V, 2-1]**
9. (same as [7], delegation metadata) **[V]**
10. Liang, Garg, Zilouchian Moghaddam. *The SWE-Bench Illusion: When State-of-the-Art LLMs Remember Instead of Reason.* arXiv:2506.12286v3, 2025-06 (NeurIPS 2025). https://arxiv.org/html/2506.12286v3 **[V]**
11. (same as [10], 5-gram reproduction) **[V]**
12. (same as [10], off-benchmark generalization) **[V]**
14. Wang, Bianchi, Zou, et al. *Automated Benchmark Auditing for AI Agents and Large Language Models.* arXiv:2605.26079, 2026-05-26. https://arxiv.org/pdf/2605.26079 **[V]**
15. (same as [14], ranking-shift result) **[V]**
16. *SWE-bench-Live.* Project site + "SWE-bench Goes Live!" arXiv:2505.23419. https://swe-bench-live.github.io/ **[V]**
17. Xia, et al. *Agentless: Demystifying LLM-based Software Engineering Agents.* arXiv:2407.01489, 2024-07 (FSE 2025). https://arxiv.org/abs/2407.01489 **[V]**
18. (same as [17], 32.00% SWE-bench Lite) **[V]**
19. (same as [17], non-agentic thesis) **[V]**
20. Anthropic. *Effective Context Engineering for AI Agents.* 2025-09-29. https://www.anthropic.com/engineering/effective-context-engineering-for-ai-agents **[S, primary first-party]**
21. *mini-swe-agent.* GitHub (Princeton/Stanford SWE-agent team). https://github.com/SWE-agent/mini-swe-agent **[V]**
22. (same as [21], bash-only / no tool-calling) **[V]**
23. (same as [21], baseline framing) **[V]**
24. Wei, et al. (Meta). *SWE-RL: Advancing LLM Reasoning via Reinforcement Learning on Open Software Evolution.* arXiv:2502.18449, 2025-02-25 (NeurIPS 2025). https://arxiv.org/abs/2502.18449 **[V]**
25. *The Design Space of Coding Agent Harnesses: Seven Architectural Lessons from Claude Code Applied to Codex CLI.* 2026-04-29. https://codex.danielvaughan.com/2026/04/29/design-space-of-coding-agent-harnesses-codex-cli-claude-code-architectural-lessons/ **[S, blog]**

Supporting training and memory sources (fetched, claims outside verified top-25):

- **[25-rlef]** *RLEF: Grounding Code LLMs in Execution Feedback with Reinforcement Learning.* Meta, arXiv:2410.02089, 2024-10-02. https://arxiv.org/abs/2410.02089
- **[26-swegym]** *Training Software Engineering Agents and Verifiers with SWE-Gym.* arXiv:2412.21139, 2024-12-30 (ICML 2025). https://arxiv.org/abs/2412.21139
- **[27-longrl]** *Training Long-Context, Multi-Turn Software Engineering Agents with Reinforcement Learning.* arXiv:2508.03501, 2025-08-05. https://arxiv.org/abs/2508.03501
- **[22-context]** Anthropic. *Managing Context on the Claude Developer Platform.* 2025-09-29. https://www.anthropic.com/news/context-management
- **[23-mem0]** *Mem0: Building Production-Ready AI Agents with Scalable Long-Term Memory.* arXiv:2504.19413, 2025-04-28. https://arxiv.org/pdf/2504.19413
- **[24-cache]** *Don't Break the Cache: Evaluating Prompt Caching for Long-Horizon Agentic Tasks.* arXiv:2601.06007v2, 2026-01-31. https://arxiv.org/html/2601.06007v2
- Terminal-Bench (refuted claims, see Appendix A): arXiv:2601.11868v1, 2026-01-17. https://arxiv.org/html/2601.11868v1
