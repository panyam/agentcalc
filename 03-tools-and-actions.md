# 03 — Tools & Actions

## Tools Define the Agent's Surface Area

An agent is only as capable as its tools. Tools are the leaf primitives — the actual contact points with the world. They have no sub-primitives; they just do things.

> The quality of tool *descriptions* matters as much as their implementation. The agent decides when to use a tool based on its description. A bad description is a broken tool.

---

## Categories of Actions

**Read actions** — observe without changing state
```
search_web(query)           → results
read_file(path)             → content
query_database(sql)         → rows
call_api_get(url)           → response
```
Cost: low to medium. Reversibility: always true (reads have no side effects).

**Write actions** — change state in the world
```
write_file(path, content)   → confirmation
send_email(to, body)        → receipt
insert_database(sql)        → rows_affected
call_api_post(url, body)    → response
```
Cost: low to medium. Reversibility: **context-dependent** — must be declared by author.

**Compute actions** — transform or process
```
run_code(code, env)         → output
call_llm(prompt)            → response
parse_document(file)        → structured_data
embed(text)                 → vector
```
Cost: medium to high. Reversibility: true (pure transformation).

**Agent actions** — invoke another agent
```
delegate(agent, task)       → result
request_review(agent, work) → feedback
spawn_parallel(agent[], tasks) → results[]
```
Cost: high. Reversibility: depends on what the agent does.

---

## Tool Design Principles

**1. Tools should have one job**
A tool that does two things is two tools. Composing single-responsibility tools is always cleaner than a multi-purpose tool with conditional behavior.

**2. Declare reversibility explicitly**
The system depends on this for safe interruption and rollback. If you don't declare it, assume irreversible.

**3. Descriptions are contracts**
The NL description is used by the type system's semantic checker. Write it as: *"This tool does X. It is appropriate when Y. It should not be used when Z."*

**4. Fail loudly with structured errors**
Agents need to detect tool failure and decide whether to retry, swap, or escalate. A vague error message forces an expensive LLM call to interpret it. A structured error (`{code, message, retryable: bool}`) enables cheap deterministic handling.

**5. Tools should be stateless where possible**
State should live in the Memory layer, not inside tools. A stateful tool is harder to swap and harder to reason about.

---

## Tool Schemas and the Type System

Every tool has a schema — the structural part of its type:

```
Tool = {
  name:        string
  description: string             ← NL contract (semantic type)
  input:       JSONSchema          ← structural type
  output:      JSONSchema          ← structural type
  metadata:    PrimitiveMetadata
}
```

The `description` + `input` + `output` together form the full dual type used by the type checker (see file 06).

---

## Tool Generation

The Generator meta-primitive can emit new tools at runtime by inspecting the problem context:

```
Generator(inspect_codebase) → emits:
  run_project_tests()         ← specific to this repo's test runner
  search_module(module_name)  ← scoped to this codebase's structure
  check_type_errors()         ← specific to this language's type checker
```

Generated tools specialize the agent to the problem at hand. They are added to the primitive graph and can be hot-swapped or discarded at the end of a session. They are **never promoted to bedrock** — only to the generatable layer.
