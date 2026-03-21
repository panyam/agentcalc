# 01 — The Kernel

Four things. Not five.

```go
type Primitive interface {
    Meta() Meta
    Run(ctx context.Context, in Message, env Envelope) (Message, error)
}

type Meta struct {
    ID          string
    Description string // NL description — the only "type" info at kernel level
    Reversible  bool   // the one safety flag the kernel needs to know
}

type Message struct {
    Payload any
    Schema  string // a label, not enforced by kernel
}

type Envelope struct {
    Depth  int
    Budget Budget        // cost ceiling + iteration limit
    Trace  []string      // ordered primitive IDs — the audit log
    Done   <-chan struct{} // cancellation — this IS the governor
    Store  Store         // shared state, opaque to kernel
}

type Store interface {
    Get(key string) (any, bool)
    Set(key string, value any)
    Update(key string, fn func(any) any) // atomic read-modify-write
    Watch(key string) <-chan any
}

type Gate func(out Message, next Primitive, env Envelope) error

type Registry struct{}
func (r *Registry) Register(p Primitive) error
func (r *Registry) Connect(from, to string, gates ...Gate) error
func (r *Registry) Invoke(id string, msg Message, env Envelope) (Message, error)
```

That is the entire kernel. Target: under 500 lines of Go including tests.

---

## What The Kernel Guarantees

```
1. Any primitive can connect to any other     → Registry.Connect
2. Connections can be validated               → Gates
3. Execution can be stopped / contained       → Envelope.Done + Budget
4. Agent IS a primitive                       → Run() can invoke Registry
```

Closure holds. Everything else is built on top by users, not the kernel.

---

## What Is Deliberately Absent

- No NL type system (user builds if needed, as a Gate)
- No memory hierarchy (user builds if needed, as Primitives + Store)
- No ResourceGovernor (Envelope.Budget is sufficient, policy is user's)
- No topology primitives (Registry.Connect IS topology)
- No crystallization (offline tooling, not kernel)
- No model selection (primitive metadata, not kernel)
- No event bus (channels, not kernel)

These are real concerns. They are not kernel concerns. The distinction matters.

---

## An Agent Is Just A Primitive That Captures A Registry

```go
type ReActAgent struct {
    registry *Registry
    model    string
}

func (a *ReActAgent) Meta() Meta {
    return Meta{ID: "react-agent", Description: "ReAct loop over registered tools"}
}

func (a *ReActAgent) Run(ctx context.Context, in Message, env Envelope) (Message, error) {
    for {
        select {
        case <-env.Done:
            return Message{}, ErrCancelled
        default:
        }
        // think, act, observe — one iteration
        // invoke tools via a.registry
        // check env.Budget
        // update env.Trace
    }
}
```

The for loop is the agent. Everything else is what you put inside it.
