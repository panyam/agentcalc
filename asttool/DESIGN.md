# AST Tool — Design Document

## Problem Statement

Coding agents (Claude Code, Codex, etc.) modify source files primarily through text-level operations (sed, grep, regex, string replacement). This is token-inefficient: most tokens in an edit call are **echo** — repeating existing code solely to disambiguate the edit location. The actual semantic delta is often <20% of the tokens spent.

**Hypothesis**: A tree-addressable edit tool — where the agent refers to CST/AST nodes by structural address and expresses only the delta — would be significantly more token-efficient and less error-prone.

## Core Insight: Three Synchronized Layers

```
Raw Text  <-->  CST (concrete)  <-->  AST (semantic)
```

Most tools pick one layer. This tool keeps all three in sync. The agent edits at whichever level is appropriate; the tool rebuilds the others.

- **Parse error?** CST still exists (tree-sitter gives ERROR nodes, not failure). Agent can edit around it.
- **Unparseable region?** Fall back to text-level edit; CST/AST rebuild what they can.
- **Perfect parse?** Full structural edits available.

## Design Principles

1. **Not an IDE.** No refactoring engine. The LLM composes primitives.
2. **Per-language tools, unified interface.** Each language has its own grammar/tool. The verbs and addressing scheme are universal.
3. **Token efficiency is the metric.** Everything is judged by: does the agent spend fewer tokens to express the same edit?
4. **Graceful degradation.** Parse errors don't block editing. The tool works at whatever layer is available.

## Addressing Scheme (draft)

LLMs already see line numbers from file reads. Addressing should build on that.

Options under consideration:

| Scheme | Example | Pros | Cons |
|--------|---------|------|------|
| Line + kind | `{"line": 42, "kind": "function_declaration"}` | Natural for LLMs, simple | Line numbers shift after edits |
| Query pattern | `function_declaration[name=processOrder]` | Stable across edits | New syntax for LLM to learn |
| Path | `/source_file/function_declaration[3]/parameter_list` | Unambiguous | Verbose, brittle to reordering |
| Kind + name | `{"kind": "function", "name": "processOrder"}` | Readable, stable | Not all nodes have names |

Likely answer: **kind + name as primary, line number as fallback** for anonymous nodes.

## Mutation Verbs (draft)

Five primitives. Everything else is composition.

```
replace_text(node, "new raw text")          — overwrite node's text
insert_child(node, position, "text")        — add inside a node
insert_sibling(node, before|after, "text")  — add next to a node
delete(node)                                — remove node
wrap(node, "template with {hole}")          — surround node
```

## Token Cost Model

For a text-replacement edit:
```
total_tokens = |old_string| + |new_string|
echo_tokens  = |old_string|     (context repeated for disambiguation)
delta_tokens = |new_string| - |shared_with_old|
efficiency   = delta_tokens / total_tokens
```

For a tree-addressed edit:
```
total_tokens = |address| + |verb| + |content|
echo_tokens  = 0          (address is structural, not echoed text)
delta_tokens = |content|
efficiency   = delta_tokens / total_tokens
```

## When Tree Addressing Does NOT Help

- **Whole-block rewrites** — replacing an entire function body. Echo cost is low because you're replacing everything.
- **New file creation** — no existing tree to address into.
- **Non-code files** — plain text, config without grammar support.

The win is proportional to how **surgical** the edits are.

## Empirical Validation Plan

### Experiment 1: Token efficiency on real agent edits

- Source: Claude Code conversation logs, SWE-bench published traces
- For each Edit tool call, compute:
  - `echo_tokens`: tokens in old_string
  - `delta_tokens`: actual change tokens
  - `efficiency`: delta / total
  - `equivalent_tree_tokens`: what the tree-addressed version would cost
- Aggregate: distribution of edit sizes, efficiency histogram, total potential savings

### Experiment 2: Edit failure rate

- Count failed edits (old_string not found) and retries
- Each failure = wasted tokens + retry cost
- Tree addressing should reduce this to ~0 (structural match, not text match)

### Experiment 3: Edits per semantic change

- For multi-site changes (rename, add parameter), count Edit calls per logical operation
- Tree tool could batch these or cascade automatically

## Architecture Sketch

```
+-----------------------------------------+
|         Unified Interface               |  <-- language-agnostic verbs
+-----------+-----------+-----------------+
|  Go Tool  |  Py Tool  |  TS Tool  ...   |  <-- per-language impls
+-----------+-----------+-----------------+
|        Three-Layer Document             |
|  +------+   +------+   +------+        |
|  | Text |<->| CST  |<->| AST  |        |
|  +------+   +------+   +------+        |
|        sync engine (per-language)       |
+-----------------------------------------+
|     tree-sitter (CST foundation)        |  <-- already multi-language
+-----------------------------------------+
```

## Foundation

tree-sitter is the natural base:
- Incremental CST parsing for ~200 languages
- Error recovery (partial parses, ERROR nodes)
- Go bindings available
- Already proven at scale (GitHub, Neovim, Zed)

## Open Questions

1. **Addressing stability**: After an edit, do node addresses shift? How to handle multi-edit sessions?
2. **What granularity do agents naturally want?** Need data from real edits.
3. **Is the CST layer sufficient, or do agents need semantic info (types, scopes)?**
4. **Tool invocation overhead**: Is the round-trip to a tree-sitter process worth it vs. inline text replacement?
5. **How to present the tree to the LLM?** Full tree dump is too many tokens. Query results only?

## Status

- [x] Initial brainstorm and problem framing
- [ ] Empirical analysis of real agent edit patterns (in progress)
- [ ] Prototype interface definition
- [ ] tree-sitter spike
- [ ] Agent integration test
