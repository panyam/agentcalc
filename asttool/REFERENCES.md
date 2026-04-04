# AST Tool — References

## Trajectory Datasets

### SWE-bench/experiments
- **URL**: https://github.com/SWE-bench/experiments
- **What**: Central collection of 100+ agent submissions to the SWE-bench leaderboard. Each submission includes predictions (`all_preds.jsonl`), metadata, and trajectory logs hosted on S3.
- **Agents**: SWE-agent, OpenHands, AutoCodeRover, Agentless, Amazon Q, Google Jules, Augment, Devin, MASAI, Factory Code Droid, Trae, Warp, and many more.
- **Format**: `.traj` (JSON), `.md`, `.json`, `.yaml` — one file per task instance. Requires AWS CLI to download trajectories from S3.
- **Relevance**: Broadest coverage of agents and edit mechanisms. Best source for cross-agent comparison.

### SWE-smith-trajectories (HuggingFace)
- **URL**: https://huggingface.co/datasets/SWE-bench/SWE-smith-trajectories
- **What**: 76,000 trajectories across 3 splits. SWE-agent with Claude 3.5 Sonnet.
- **Format**: Parquet with `messages`, `patch`, `resolved`, `model` fields.
- **Relevance**: Large-scale SWE-agent data. Uses line-range edit commands — good baseline for comparing line-addressed vs text-match efficiency.

### nebius/SWE-agent-trajectories (HuggingFace)
- **URL**: https://huggingface.co/datasets/nebius/SWE-agent-trajectories
- **What**: 80,036 trajectories. SWE-agent with Llama-70B variants.
- **Format**: Parquet with `trajectory` (list of role/text turns), `generated_patch`, `eval_logs`.
- **Relevance**: Same line-range edit mechanism as SWE-smith but different model. Useful for isolating model vs tool effects on edit quality.

### nebius/SWE-rebench-openhands-trajectories (HuggingFace)
- **URL**: https://huggingface.co/datasets/nebius/SWE-rebench-openhands-trajectories
- **What**: 67,074 trajectories. OpenHands + Qwen3-Coder-480B.
- **Format**: Parquet with `trajectory` (conversation), `model_patch` (unified diff), `resolved`.
- **Relevance**: Uses `str_replace_editor` with `old_str`/`new_str` — same text-match paradigm as Claude Code Edit. Direct comparison target for echo waste analysis.

### ByteDance Multi-SWE-bench trajectories (HuggingFace)
- **URL**: https://huggingface.co/datasets/ByteDance-Seed/Multi-SWE-bench_trajs
- **What**: Multi-SWE-agent with Claude 3.5 Sonnet. Multi-repo variant of SWE-bench.
- **Format**: `.traj` JSON files.
- **Relevance**: Tests edit mechanisms across multi-repo contexts where disambiguation is harder.

### CoderForge trajectories (HuggingFace)
- **URL**: https://huggingface.co/datasets/togethercomputer/CoderForge-Preview-32B-SWE-Bench-Verified-Evaluation-trajectories
- **What**: CoderForge 32B model evaluation trajectories.
- **Format**: Parquet.
- **Relevance**: Smaller model — useful for testing whether tree-addressed edits help weaker models more.

## Agent Source Code & Edit Mechanisms

### SWE-agent
- **URL**: https://github.com/princeton-nlp/SWE-agent
- **Trajectory docs**: https://swe-agent.com/latest/usage/trajectories/
- **Edit mechanism**: Custom shell with `edit <start_line>:<end_line>` + new content + `end_of_edit`. Also `open`, `search_file`, `find_file`, `scroll_up/down`.
- **Relevance**: Line-range addressing — avoids echo tax but line numbers shift after edits. The mid-point between text-match and structural addressing.

### OpenHands (CodeAct)
- **URL**: https://github.com/All-Hands-AI/OpenHands
- **Edit mechanism**: `str_replace_editor` tool with `command`, `path`, `old_str`, `new_str`, `file_text`, `view_range`. Also has bash executor.
- **Relevance**: Same text-match paradigm as Claude Code Edit. High echo cost. Good comparison target.

### Agentless
- **URL**: https://github.com/OpenAutoCoder/Agentless
- **Edit mechanism**: Not an interactive agent. Three-phase pipeline: hierarchical localization → candidate patch generation (unified diff) → patch validation + re-ranking. Generates diffs directly.
- **Relevance**: Shows that direct diff generation (no interactive editing) is viable. Different trade-off: more tokens in generation, zero in addressing.

### AutoCodeRover
- **URL**: https://github.com/nus-apr/auto-code-rover
- **Edit mechanism**: AST-aware search APIs (search by class, method, function name) → localize → patch. Searches by structure, not text.
- **Relevance**: Closest existing system to our AST tool proposal. Demonstrates that structural addressing works in practice. Worth studying their search API design.

### Aider
- **URL**: https://github.com/paul-gauthier/aider
- **Edit format docs**: https://aider.chat/docs/more/edit-formats.html
- **Edit mechanism**: SEARCH/REPLACE blocks with fuzzy matching cascade (exact → whitespace-insensitive → indent-preserving → difflib). Also supports whole-file rewrite mode.
- **Relevance**: Fuzzy matching is an attempt to solve the brittleness of text-match. Interesting middle ground — still echo-heavy but more robust to whitespace mismatches.

## Research Papers

### Understanding Software Engineering Agents (2025)
- **URL**: https://arxiv.org/html/2506.18824v1
- **What**: Study of RepairAgent, AutoCodeRover, OpenHands across 120 trajectories / 2,822 LLM interactions.
- **Key findings**: Thought-action misalignment 1% in successes vs 40% in failures. Failed trajectories exhibit repetitive non-adaptive cycles. Unsuccessful RepairAgent runs average 40 iterations vs 22 for successful.
- **Relevance**: Quantifies the cost of edit failures — failed edits lead to cascading retry loops that waste tokens and often don't converge.

### SWE-bench+ / Rigorous Evaluation (ACL 2025)
- **URL**: https://aclanthology.org/2025.acl-long.189.pdf
- **What**: Analyzes patch quality on SWE-bench. Found 63.75% of SWE-Agent+GPT-4 patches were "suspicious" (answer leak 32.67%, weak tests 31.08%). 7.8% of "passing" patches fail against full test suite.
- **Relevance**: Edit mechanism quality matters beyond token efficiency — poorly addressed edits can produce patches that pass weak tests but aren't actually correct.

### SWE-agent Paper (2024)
- **URL**: https://arxiv.org/pdf/2405.15793
- **What**: Original SWE-agent design paper. 51.7% of GPT-4 Turbo trajectories had 1+ failed edit operation (lint errors).
- **Relevance**: Baseline failure rate for line-range editing. Even with line-range addressing, edit failures are common — suggesting the problem isn't just addressing but also content generation.

## Blog Posts & Surveys

### Code Surgery: How AI Assistants Make Precise Edits
- **URL**: https://fabianhertwig.com/blog/coding-assistants-file-edits/
- **What**: Survey of how different AI coding tools (Cursor, Aider, Claude Code, etc.) implement file editing. Covers diff formats, apply models, streaming, and the trade-offs between whole-file rewrite, search/replace, and diff-based approaches.
- **Relevance**: Good overview of the current landscape and the practical challenges each approach faces (formatting preservation, merge conflicts, partial application).

## Tools & Libraries

### tree-sitter
- **URL**: https://tree-sitter.github.io/tree-sitter/
- **Go bindings**: https://github.com/tree-sitter/go-tree-sitter
- **What**: Incremental parsing library for ~200 languages. Produces concrete syntax trees (CST) with error recovery. Used by GitHub, Neovim, Zed, Helix.
- **Relevance**: Natural foundation for the AST tool. Provides the CST layer with error recovery, incremental re-parsing, and broad language support out of the box.
