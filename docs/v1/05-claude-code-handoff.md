# 05 — Claude Code Handoff

## What Has Changed

We over-designed. The 18 documents you may have seen earlier describe real problems but solve them at the wrong layer — in the kernel rather than in user-land. This document supersedes all of them.

The new direction: minimal kernel, organic growth, real benchmarks.

---

## The Kernel (The Only Thing To Build Right Now)

Four things:

```go
type Primitive interface {
    Meta() Meta
    Run(ctx context.Context, in Message, env Envelope) (Message, error)
}

type Meta struct {
    ID          string
    Description string
    Reversible  bool
}

type Message struct {
    Payload any
    Schema  string
}

type Envelope struct {
    Depth  int
    Budget Budget
    Trace  []string
    Done   <-chan struct{}
    Store  Store
}

type Store interface {
    Get(key string) (any, bool)
    Set(key string, value any)
    Update(key string, fn func(any) any)
    Watch(key string) <-chan any
}

type Gate func(out Message, next Primitive, env Envelope) error

type Registry struct{}
func (r *Registry) Register(p Primitive) error
func (r *Registry) Connect(from, to string, gates ...Gate) error
func (r *Registry) Invoke(id string, msg Message, env Envelope) (Message, error)
```

Target: under 500 lines of Go including tests. If it is growing past that, something is wrong.

---

## What To Build On Top Of The Kernel (In Order)

### 1. Four Tools For SWE-bench

```
ReadFile(path) → content
SearchCode(query) → []match          // grep/ast search, NOT vector search yet
EditFile(path, patch) → ok|error
RunTests(ids) → pass|fail
```

These are leaf Primitives. Simple functions wrapped in the Primitive interface.

`SearchCode` uses the filesystem directly. Not a vector index. Not embeddings. When grep is not good enough — and only then — upgrade the Store implementation.

### 2. A ReAct Loop As A Primitive

```go
type ReActAgent struct {
    registry *Registry
    model    string
    maxIter  int
}

func (a *ReActAgent) Run(ctx context.Context, in Message, env Envelope) (Message, error) {
    for i := 0; i < a.maxIter; i++ {
        if isDone(env) { break }
        // call model, get action
        // invoke action via registry
        // append to trace
    }
}
```

The for loop. That is the agent. No topology. No planner primitive. No pattern template.

### 3. One Gate: Budget Enforcement

```go
func BudgetGate(budget Budget) Gate {
    return func(out Message, next Primitive, env Envelope) error {
        if env.Budget.Exhausted() {
            return ErrBudgetExceeded
        }
        return nil
    }
}
```

Attach globally. This is one of the rare cases where global is correct.

### 4. Run On SWE-bench Lite

Pick 10 instances. Run. Read the traces. Do not add anything until you have read the traces.

---

## The Critical Discipline

When you feel the urge to add something, ask in order:

```
1. Has the running system actually failed in a way that requires this?
   No → do not add it.

2. Can it be built as a Gate on specific edges?
   Yes → build it there, not in the kernel.

3. Can it be built as a Primitive that users register?
   Yes → build it there, not in the kernel.

4. Does removing it break the closure property?
   (agent is a primitive, system closed under composition)
   No → do not add it to the kernel.
```

The kernel has four things. If it has five things after this session, one of them needs a very strong justification.

---

## What To Do With The Old Design

The archive (18 documents) is a reference map. When you hit a specific failure in the running system, check the map to see if we already thought about that problem. Use it reactively, never proactively.

Specifically do NOT build:
- NL type checker (speculative, probabilistic, expensive)
- TopologyRewriter (premature, for loop handles it)
- Crystallizer (offline tooling, not now)
- Memory hierarchy (Store + Primitives covers it when needed)
- ResourceGovernor as a separate struct (Envelope.Budget IS the governor)

---

## The Right Question At Every Decision Point

> "What specific failure in the running system requires this?"

If the answer is "it might fail" or "it will eventually fail" or "it's cleaner this way" — stop. Run the system. Let it fail. Then fix the actual failure.
