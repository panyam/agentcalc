# AST Tool — Empirical Findings

## 1. Local Claude Code Edit Analysis

**Data source:** 1,697 Claude Code session files (~679 MB), 197 sessions containing edits.

### Core Numbers

| Metric | Value |
|--------|-------|
| Total Edit tool calls | 6,247 |
| Edit failures | 148 (2.4%) |
| Retries | 657 (10.5% of edits are re-attempts) |
| Write tool calls (comparison) | 2,421 |

### Token Efficiency

| Metric | Approx Tokens |
|--------|---------------|
| Total old_string (echo/waste) | ~490K |
| Total new_string | ~921K |
| **Total Edit input** | **~1,411K** |
| Actual delta (real change) | ~714K |
| **Wasted (echo/context)** | **~697K (49.4%)** |

**~50% of all tokens in Edit calls are pure echo.** They exist solely to tell the tool _where_ to make the change.

### Per-Edit Efficiency Distribution

| Efficiency bucket | % of edits |
|-------------------|-----------|
| <10% (worst — huge context, tiny change) | 19.8% |
| 10–20% | 18.0% |
| 20–50% | ~35% |
| 50–90% | ~21% |
| 90–100% (essentially a rewrite) | 6.2% |

**Mean efficiency: 36.3%. Median: 29.2%.** The typical edit echoes ~71% redundant context.

### The Pain Zone: 50–500 char old_string

66% of all edits fall in this range. Average efficiency: 34.5%. These are the classic "change one line inside a function" edits where 3–10 lines of surrounding code must be echoed to disambiguate.

**This is exactly the case tree-addressed edits would eliminate.**

### Edit Failure Breakdown

| Failure type | Count | % |
|--------------|-------|---|
| file_not_read (forgot to Read first) | 72 | 48.6% |
| string_not_found | 37 | 25.0% |
| file_modified_since_read (race) | 33 | 22.3% |
| multiple matches (not unique) | 6 | 4.1% |

Key insight: **string_not_found + multiple_matches = 29%** of failures are directly caused by text-based addressing. Tree addressing eliminates both categories.

---

## 2. SWE-bench Agent Trace Analysis

### Available Trajectory Data

| Source | Trajectories | Agents | Format |
|--------|-------------|--------|--------|
| [SWE-bench/experiments](https://github.com/SWE-bench/experiments) | 100+ submissions | All leaderboard agents | .traj, .json, .md (S3-hosted) |
| [SWE-smith-trajectories](https://huggingface.co/datasets/SWE-bench/SWE-smith-trajectories) | 76,000 | SWE-agent + Claude 3.5 | Parquet |
| [nebius/SWE-agent-trajectories](https://huggingface.co/datasets/nebius/SWE-agent-trajectories) | 80,036 | SWE-agent + Llama-70B | Parquet |
| [nebius/SWE-rebench-openhands](https://huggingface.co/datasets/nebius/SWE-rebench-openhands-trajectories) | 67,074 | OpenHands + Qwen3 | Parquet |
| [ByteDance Multi-SWE-bench](https://huggingface.co/datasets/ByteDance-Seed/Multi-SWE-bench_trajs) | Unknown | Multi-SWE-agent + Claude 3.5 | .traj JSON |

**Total: 200K+ downloadable trajectories** across multiple agents and models.

### Edit Mechanisms by Agent (Three Paradigms)

| Agent | Edit mechanism | Addressing | Echo cost |
|-------|---------------|------------|-----------|
| **SWE-agent** | `edit <start_line>:<end_line>` + new content + `end_of_edit` | Line range | Low — only specifies line numbers + new content |
| **OpenHands/CodeAct** | `str_replace_editor` with `old_str`/`new_str` | Text match (like Claude Code Edit) | High — full old_str echo |
| **Agentless** | Direct unified diff generation | Line-based diff | Medium — diff context lines |
| **Aider** | SEARCH/REPLACE blocks with fuzzy matching | Text match (with fallbacks) | High — full SEARCH block echo |
| **AutoCodeRover** | AST-aware search APIs → patch | Class/method names | Low — searches by structure |

**Notable: SWE-agent already uses line-range addressing**, which avoids the echo problem. AutoCodeRover uses AST-aware search. The worst echo costs come from str_replace/SEARCH-REPLACE patterns (OpenHands, Aider, Claude Code).

### Failure Rates from Published Research

| Finding | Source |
|---------|--------|
| **51.7%** of SWE-agent + GPT-4 trajectories had 1+ failed edit (lint error) | SWE-agent paper |
| Thought-action misalignment: **1% in successes vs 40% in failures** | arxiv:2506.18824 |
| Failed trajectories average **40 iterations** vs 22 for successful | arxiv:2506.18824 |
| **63.75%** of SWE-Agent+GPT-4 patches were "suspicious" quality | ACL 2025 / SWE-bench+ |

### Key Papers

- [Understanding SE Agents: Thought-Action-Result Trajectories](https://arxiv.org/html/2506.18824v1) — 120 trajectories, 2,822 LLM interactions across RepairAgent, AutoCodeRover, OpenHands
- [Rigorous Evaluation of Coding Agents on SWE-Bench](https://aclanthology.org/2025.acl-long.189.pdf) — ACL 2025, patch quality analysis
- [SWE-agent paper](https://arxiv.org/pdf/2405.15793) — original SWE-agent design
- [Code Surgery: How AI Assistants Make Precise Edits](https://fabianhertwig.com/blog/coding-assistants-file-edits/) — survey of edit mechanisms

---

## 3. Cross-Cutting Observations

### The echo tax is real and measurable

From our local data: **~697K tokens wasted** on echo across 6,247 edits. That's ~112 tokens of waste per edit on average. At scale (thousands of agent runs across SWE-bench), this is millions of tokens.

### Three paradigms, ranked by token efficiency

1. **Line-range addressing** (SWE-agent): specify lines to replace, provide new content only. No echo.
2. **Structural addressing** (AutoCodeRover, proposed AST tool): address by class/function/node. No echo, and stable across edits.
3. **Text-match addressing** (Claude Code Edit, OpenHands, Aider): echo existing code for disambiguation. 50%+ waste.

### Line-range is better than text-match but worse than structural

Line-range addressing (SWE-agent's approach) avoids echo but has a problem: **line numbers shift after each edit.** In a multi-edit session, the agent must re-read or mentally track line number changes. SWE-agent handles this by re-displaying the file after each edit, but that's still token cost.

Structural addressing (by node kind + name) is stable across edits — "the function named processOrder" doesn't change when you edit a different function.

### The biggest win: eliminating the 50–500 char echo bucket

66% of edits, 34.5% efficiency. These are one-line changes inside multi-line blocks. A tree tool that says `{"at": "function[processOrder]/if_statement[0]/return", "verb": "replace", "text": "return ctx.Err()"}` costs ~15 tokens vs. ~80 tokens for the equivalent text-match edit.

### Edit failures are a secondary but real cost

2.4% failure rate locally, but **51.7% of SWE-agent trajectories had at least one failed edit**. Each failure burns the full token cost of the attempt plus retry overhead. Structural addressing should reduce string_not_found and multiple_matches to zero.

---

## 4. Error Resilience: The Compilation Requirement Gap

### The problem agents actually hit

Agents need refactoring tools **most when code is broken** — mid-refactor, missing deps, partial rewrites. But the tools that do it correctly require the code to compile.

### Existing Go tools, ranked by error tolerance

| Tool | Needs types? | Needs parse? | Broken code? | Maintained? |
|------|-------------|-------------|-------------|-------------|
| `gorename` | Yes | Yes | No | **Deprecated** (v0.1.0-deprecated) |
| `gopls rename` | Partially | Partially | Moderate | Yes (v0.20.0) |
| `rf` (rsc.io/rf) | Yes | Yes | No | Yes (experimental) |
| `gofmt -r` | No | Yes | No | Yes (ships with Go) |
| `eg` | Yes | Yes | No | Stale |
| **`ast-grep`** | No | Tolerant | **Yes** | Yes (actively developed) |
| `comby` | No | No | **Yes** | **Dying** (Homebrew deprecated) |
| `fastmod` | No | No | **Yes** | Yes |

### The spectrum

```
sed/regex          — works on broken code, no semantic awareness
tree-sitter/CST    — works on broken code, structural awareness  ← THE GAP
go/ast (no types)  — needs valid syntax, structural awareness
gopls/gorename     — needs full typecheck, full semantic awareness
```

### Key findings

- **`gopls rename`** is better than expected — it succeeds on isolated parse/type errors. But [issue #71908](https://github.com/golang/go/issues/71908) (Feb 2025) reports it fails "multiple times per day" during active development with cascading errors across packages.
- **`ast-grep`** (tree-sitter based, Rust) is the strongest option for broken code. Understands tree structure (won't match inside strings), tolerates parse errors via tree-sitter's error recovery. Not type-aware, so renames are purely structural.
- **`comby`** was the other structural option but is dying — Homebrew deprecated, depends on EOL `pcre`.
- **`gofmt -r`** requires valid syntax and only handles expression-level rewrites. Not useful for renames.
- **`rf`** is powerful (mv, add, rm for functions/types/fields) but requires clean compilation. Experimental, maintained by Russ Cox.

### Implication for asttool

The gap between "regex on text" and "requires compilation" is exactly where tree-sitter sits. An agent-facing edit tool built on tree-sitter would:
1. Work on broken code (error-recovering parse)
2. Have structural awareness (won't corrupt strings/comments)
3. Be stable across edits (node kind + name addressing doesn't shift)
4. Not require the full Go toolchain to be functional

`ast-grep` already exists in this space and is worth evaluating as a foundation rather than building from scratch.

---

## 5. Go tree-sitter Grammar Gotcha: `identifier` vs `type_identifier`

Go's tree-sitter grammar splits what `go/ast` treats as a single `*ast.Ident` into two distinct node kinds:

| Node kind | Where it appears | Example |
|-----------|-----------------|---------|
| `identifier` | Variable names, function names, labels, package names | `func processOrder(...)`, `x := foo` |
| `type_identifier` | Type annotations, return types, field types, composite literal types | `var m Message`, `func f() Response` |
| `field_identifier` | Struct field names in literals and selectors | `Response{Body: x}`, `r.Body` |

**Why this matters for agents:**
- `sg --pattern 'Message'` matches `identifier` nodes only — it will NOT find `Message` used as a type
- To match types, you must use YAML rules with `kind: type_identifier` and `regex` instead of `pattern`
- `field_identifier` separation is actually helpful — it means qualify/unqualify rules naturally skip struct field names without needing the `isStructFieldName` check that our Go tool needed

**This is a tree-sitter design decision, not an ast-grep bug.** Tree-sitter grammars are language-specific, and Go's grammar makes this distinction because types and expressions occupy different syntactic positions in Go. Other languages (Python, JS) may not have this split.

See `SG_TEST_CASES.md` for validated patterns that handle both node kinds.

---

## 6. Next Steps

1. **Download SWE-smith trajectories** (76K, Parquet) and compute per-edit token efficiency across SWE-agent's line-range edits. Compare to OpenHands' str_replace edits on the same tasks.
2. **Build a conversion tool**: take real Edit calls from our logs, convert to equivalent tree-addressed form, compare token counts concretely.
3. **Evaluate `ast-grep`** as a foundation — test it on broken Go files, measure pattern quality for rename/refactor operations, assess whether it can be wrapped as an MCP tool for agents.
4. **Prototype the tree-address tool** with tree-sitter (or ast-grep) and test on a sample of the worst-efficiency edits from our data.

---

## References

- SWE-bench experiments: https://github.com/SWE-bench/experiments
- SWE-agent trajectories docs: https://swe-agent.com/latest/usage/trajectories/
- SWE-smith trajectories: https://huggingface.co/datasets/SWE-bench/SWE-smith-trajectories
- nebius SWE-agent trajectories: https://huggingface.co/datasets/nebius/SWE-agent-trajectories
- nebius OpenHands trajectories: https://huggingface.co/datasets/nebius/SWE-rebench-openhands-trajectories
- ByteDance Multi-SWE-bench: https://huggingface.co/datasets/ByteDance-Seed/Multi-SWE-bench_trajs
- Agentless: https://github.com/OpenAutoCoder/Agentless
- Aider edit formats: https://aider.chat/docs/more/edit-formats.html
- SE Agents trajectory study: https://arxiv.org/html/2506.18824v1
- SWE-bench+ (ACL 2025): https://aclanthology.org/2025.acl-long.189.pdf
