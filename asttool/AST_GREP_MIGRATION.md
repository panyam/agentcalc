# Replacing edittool/go with ast-grep

## Summary

Four custom Go tools in `~/newstack/edittool/go/` can be replaced by `ast-grep` (`sg`). Three are straightforward. One (`rewrite_imports` / qualify) requires YAML rules with edge-case handling.

## Tool-by-Tool Analysis

### 1. rename (cmd/rename) — FULLY REPLACEABLE

**What it does:** Rename all occurrences of an identifier across files.

**Custom tool:** 80 lines Go, uses `go/ast` + `go/parser` + `go/format`. Walks AST, matches `*ast.Ident` by name, replaces.

**ast-grep equivalent:**
```bash
sg --pattern 'oldName' --rewrite 'newName' --lang go <files...>
```

**Verdict:** One-liner. No YAML needed. ast-grep's pattern matching on identifiers handles declarations and references.

**Caveat:** `sg` matches identifiers in expression context but NOT `type_identifier` nodes (Go tree-sitter distinguishes them). To rename a type that appears in both positions, you need two commands or a YAML rule:
```bash
# Expression context (variable names, function names)
sg --pattern 'oldName' --rewrite 'newName' --lang go .

# Type context (type annotations, return types, field types) — needs YAML
# See test case T1b below
```

---

### 2. unqualify (cmd/unqualify) — FULLY REPLACEABLE

**What it does:** Strip package qualifier: `core.Symbol` → `Symbol`. Remove unused import.

**Custom tool:** 110 lines Go, uses `go/ast` + `astutil.Apply` + `astutil.DeleteNamedImport`.

**ast-grep equivalent:**
```bash
# Strip qualifier
sg --pattern 'core.$METHOD' --rewrite '$METHOD' --lang go <files...>

# For composite literals: core.Response{...} → Response{...}
sg --pattern 'core.$TYPE{$$$FIELDS}' --rewrite '$TYPE{$$$FIELDS}' --lang go <files...>
```

**Remaining gap:** ast-grep does NOT manage imports. After unqualifying, you still need `goimports` to clean up the now-unused import:
```bash
sg --pattern 'core.$METHOD' --rewrite '$METHOD' --lang go --update-all .
goimports -w .
```

**Verdict:** Replaceable. Pair with `goimports` for import cleanup.

---

### 3. replacecall (cmd/replacecall) — FULLY REPLACEABLE

**What it does:** Replace function calls with a template expression. `WithInMemoryServer(args)` → `client.WithTransport(server.NewInProcessTransport(args))`.

**Custom tool:** 134 lines Go, uses `astutil.Apply`, builds replacement AST by parsing template with `{args}` placeholder.

**ast-grep equivalent:**
```bash
sg --pattern 'WithInMemoryServer($$$ARGS)' \
   --rewrite 'client.WithTransport(server.NewInProcessTransport($$$ARGS))' \
   --lang go <files...>
```

**Verdict:** One-liner. `$$$ARGS` captures variadic arguments naturally. Exact feature parity.

---

### 4. wrapcall (cmd/wrapcall) — FULLY REPLACEABLE

**What it does:** Wrap a call: `old(args)` → `wrapper(old(args))`.

**Custom tool:** 143 lines Go, uses `astutil.Apply`, constructs nested `*ast.CallExpr`.

**ast-grep equivalent:**
```bash
sg --pattern 'dangerousOp($$$ARGS)' \
   --rewrite 'retry(dangerousOp($$$ARGS))' \
   --lang go <files...>
```

**Verdict:** One-liner. The pattern naturally nests the original call in the replacement.

---

### 5. rewrite_imports (qualify) — PARTIALLY REPLACEABLE

**What it does:** Add package qualifier to bare symbols: `Message` → `core.Message`, `NewMessage(x)` → `core.NewMessage(x)`. Skips definitions, struct field names, already-qualified references.

**Custom tool:** 225 lines Go. The most complex tool. Handles: local symbol exclusion, `isDefinition`, `isQualified`, `isStructFieldName`, import addition.

**ast-grep equivalent:** Requires YAML rules with exclusions:

```yaml
# sg_qualify.yml
id: qualify-message-type
language: go
rule:
  regex: "^Message$"
  kind: type_identifier
  not:
    any:
      - inside:
          kind: type_spec
          stopBy: end
      - inside:
          kind: qualified_type
          stopBy: end
fix: core.Message
---
id: qualify-newmessage-call
language: go
rule:
  pattern: NewMessage($$$ARGS)
  not:
    any:
      - inside:
          kind: function_declaration
          has:
            kind: identifier
            regex: "^NewMessage$"
      - inside:
          kind: call_expression
          has:
            kind: selector_expression
fix: core.NewMessage($$$ARGS)
```

```bash
sg scan --rule sg_qualify.yml --update-all .
goimports -w .  # add the core import
```

**What works:**
- Type references in field types, return types, parameter types — correctly qualified
- Composite literals (`Response{...}` → `core.Response{...}`) — correct
- Struct field names (`Message: "hello"`) — correctly skipped (they're `field_identifier`, not `type_identifier`)
- Selector expressions (`r.Message`) — correctly skipped
- Already-qualified (`core.Message`) — correctly skipped with `qualified_type` exclusion
- **Broken code** — works! tree-sitter error recovery handles missing braces/parens

**What doesn't work:**
- Bare identifier in expression context (`_ = Message`) — not matched by `type_identifier` rule. Unusual Go but possible.
- Import management — must pair with `goimports`
- Local symbol exclusion — the Go tool loads a symbols file and skips locally-defined names. ast-grep has no equivalent; you'd need to generate the rule file dynamically or accept some manual review.

**Verdict:** 80% replaceable. The core rewriting works. Import management and local-vs-external symbol discrimination still need `goimports` and possibly a small script to generate the YAML from a symbol list.

---

## Error Resilience Comparison

| Scenario | edittool/go | ast-grep |
|----------|------------|----------|
| Valid Go code | Works | Works |
| Type errors (missing deps) | Works (parse-only) | Works |
| Syntax errors (missing braces) | **Fails** (`go/parser` errors) | **Works** (tree-sitter error recovery) |
| Missing closing paren | **Fails** | **Works** |
| Partial file | **Fails** | **Works** |

ast-grep wins on error resilience — the main reason to prefer it.

---

## What ast-grep Cannot Do (need goimports or gopls)

1. **Add/remove imports** — ast-grep rewrites node text, it doesn't understand Go import semantics
2. **Scope-aware rename** — distinguishing local `Message` from imported `Message` requires type info
3. **Cross-package rename** — finding all callers across a module requires `gopls` or `go/packages`

**Recommended workflow:**
```bash
sg scan --rule rules.yml --update-all .   # structural rewrite
goimports -w .                             # fix imports
go vet ./...                               # verify
```

---

## Library Bindings

ast-grep can be used as a library, not just a CLI. Bindings exist for several languages — but not Go.

| Binding | Package | Pattern syntax | Rewriting | Maturity |
|---------|---------|---------------|-----------|----------|
| **Rust** | `ast-grep-core` | `$A` / `$$$` | Yes | Production (powers the CLI) |
| **Node/JS** | `@ast-grep/napi` | `$A` / `$$$` | Yes | Most robust binding |
| **Python** | `ast-grep-py` | `$A` / `$$$` | Yes (manual metavar sub) | Alpha but functional |
| **WASM** | `@ast-grep/wasm` | `$A` / `$$$` | Yes | Available |
| **Go** | **None** | — | — | — |

### Go options for programmatic use

1. **Shell out to `sg` CLI** — use `--json` for structured output. Zero integration overhead. Best option for orchestrating transforms.
2. **`go-tree-sitter`** (`github.com/smacker/go-tree-sitter`) — Go bindings to tree-sitter's C library via CGo. Parsing + S-expression queries + match positions. No `$A`-style patterns, no built-in rewriting (DIY byte-offset splicing). 30+ language grammars included.
3. **`@ast-grep/napi` or `ast-grep-py`** — if Go isn't a hard requirement, these give the full pattern+rewrite API as a library.

### Recommendation

For tool-building: just use `sg` CLI. The four edittool commands become one-liners or YAML rule files. No Go code to maintain.

For programmatic Go integration: `go-tree-sitter` gives you parsing and queries, but you'd rebuild some of ast-grep's convenience. Only worth it if you need tight in-process control.
