# ast-grep (sg) Edit Pattern Test Cases

Test cases validated empirically against Go source files. Each case documents the `sg` command/rule, input, expected output, and known edge cases.

## T1: Rename identifier (expression context)

**Command:**
```bash
sg --pattern 'processOrder' --rewrite 'handleOrder' --lang go --update-all .
```

**Input:**
```go
func processOrder(p ProcessOrder) error {
    fmt.Println("processing", p.Name)
    return nil
}

func main() {
    processOrder(o)
}
```

**Expected:** Both declaration and call site renamed to `handleOrder`. Type `ProcessOrder` untouched.

**Status:** PASS

---

## T1b: Rename type (type_identifier context)

`sg --pattern 'TypeName'` does NOT match type references in Go. Tree-sitter Go uses `type_identifier` for types and `identifier` for expressions. YAML rule required.

**Rule:**
```yaml
id: rename-type
language: go
rule:
  regex: "^ProcessOrder$"
  kind: type_identifier
fix: Order
```

**Input:**
```go
type ProcessOrder struct { Name string }
func handle(p ProcessOrder) ProcessOrder { return p }
```

**Expected:** All three `ProcessOrder` become `Order` (including the definition).

**Note:** If you want to skip the definition, add:
```yaml
  not:
    inside:
      kind: type_spec
      stopBy: end
```

**Status:** PASS

---

## T2: Replace function call

**Command:**
```bash
sg --pattern 'WithInMemoryServer($$$ARGS)' \
   --rewrite 'client.WithTransport(server.NewInProcessTransport($$$ARGS))' \
   --lang go --update-all .
```

**Input:**
```go
conn := WithInMemoryServer(srv)
other := WithInMemoryServer(srv, opts)
```

**Expected:**
```go
conn := client.WithTransport(server.NewInProcessTransport(srv))
other := client.WithTransport(server.NewInProcessTransport(srv, opts))
```

**Status:** PASS — `$$$ARGS` captures variadic args correctly.

---

## T3: Unqualify (strip package prefix)

**Command:**
```bash
sg --pattern 'core.$METHOD' --rewrite '$METHOD' --lang go --update-all .
sg --pattern 'core.$TYPE{$$$FIELDS}' --rewrite '$TYPE{$$$FIELDS}' --lang go --update-all .
goimports -w .
```

**Input:**
```go
func handle(r core.Request) core.Response {
    msg := core.NewMessage("hello")
    return core.Response{Body: msg}
}
```

**Expected:** All `core.` prefixes stripped. `goimports` removes the unused import.

**Status:** PASS

---

## T4: Qualify type (add package prefix, skip definitions)

**Rule:**
```yaml
id: qualify-message-type
language: go
rule:
  regex: "^Message$"
  kind: type_identifier
  not:
    any:
      - inside: { kind: type_spec, stopBy: end }
      - inside: { kind: qualified_type, stopBy: end }
fix: core.Message
```

**Command:**
```bash
sg scan --rule sg_qualify.yml --update-all .
goimports -w .
```

### T4a: References only — skip definitions

**Input:**
```go
type Message struct { Body string }
func NewMessage(s string) Message { return Message{Body: s} }
func handle(m Message) {}
```

**Expected:** `type Message struct` untouched. Return type, parameter type, and composite literal qualified. `NewMessage` declaration untouched.

**Status:** PASS

### T4b: Struct field names — should NOT be qualified

**Input:**
```go
r := Response{
    Message: "hello",
    Body:    NewMessage("test"),
}
_ = r.Message
```

**Expected:** `Message:` (field name) and `r.Message` (selector) untouched. `Response` and `NewMessage()` qualified.

**Status:** PASS — tree-sitter classifies field names as `field_identifier`, not `type_identifier`.

### T4c: Already-qualified — should NOT double-qualify

**Input:**
```go
var m core.Message
r := core.Response{Body: core.NewMessage("x")}
```

**Expected:** No changes. Already qualified.

**Status:** PASS — `qualified_type` exclusion prevents double-qualification.

### T4d: Broken code — error resilience

**Input:**
```go
package main

func broken( {
    var m Message
    r := Response{Body: NewMessage("x")}
    // missing closing brace and paren
```

**Expected:** `Message`, `Response`, `NewMessage` all qualified despite syntax errors.

**Status:** PASS — tree-sitter error recovery handles missing braces/parens.

---

## T5: Qualify function call (skip definition)

**Rule:**
```yaml
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

**Input:**
```go
func NewMessage(s string) Message { return Message{Body: s} }
func use() { m := NewMessage("x") }
```

**Expected:** Definition `func NewMessage(...)` untouched. Call `NewMessage("x")` → `core.NewMessage("x")`.

**Status:** PASS

---

## Known Gotchas

### G1: `identifier` vs `type_identifier` in Go tree-sitter

Go's tree-sitter grammar distinguishes:
- `identifier` — variable names, function names, labels (expression context)
- `type_identifier` — type names in annotations, return types, field types (type context)

`sg --pattern 'Foo'` matches `identifier` nodes only. To match types, use a YAML rule with `kind: type_identifier` and `regex` instead of `pattern`.

### G2: No import management

ast-grep rewrites node text only. It does not add or remove import statements. Always pair with:
- **Go:** `goimports -w .`
- **Python:** `isort .` or `autoflake`
- **TS/JS:** your linter's auto-import

### G3: `regex` vs `pattern` in YAML rules

- `pattern: Foo` — matches the AST node that tree-sitter would produce by parsing `Foo` as source code
- `regex: "^Foo$"` — matches the text content of any node (filtered by `kind`). Required when `pattern` doesn't match the right node kind.

For Go types, `regex` + `kind: type_identifier` is the way.

### G4: Composite literals need separate pattern

`sg --pattern 'pkg.$SYM'` matches selector expressions (`pkg.Foo`) but NOT composite literals (`pkg.Foo{...}`). For those:
```bash
sg --pattern 'pkg.$TYPE{$$$FIELDS}' --rewrite '$TYPE{$$$FIELDS}' --lang go .
```
