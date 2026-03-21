# 03 — Start With The For Loop

## The Agent Is A For Loop

Every reasoning pattern we designed — ReAct, Plan→Execute, Critic/Verifier, Tree of Thought — is a for loop with different termination conditions and different things inside the body.

```
ReAct:           for !done { think; act; observe }
Plan→Execute:    plan = make_plan(); for step in plan { execute(step) }
Critic/Verifier: for !critic_passes { generate; critique }
Tree of Thought: for !converged { branch; evaluate; select }
```

This is not a simplification. This IS the structure. Everything else is what you put inside the loop and what terminates it.

Start here. Literally.

---

## The Organic Growth Path

### Stage 0: The Actual For Loop

```go
func runAgent(issue string, tools map[string]Tool) string {
    ctx := ""
    for i := 0; i < 20; i++ {
        response := llm.Call(ctx + issue)
        if response.IsDone {
            return response.Answer
        }
        result := tools[response.ToolName](response.ToolArgs)
        ctx += fmt.Sprintf("\nAction: %s\nResult: %s", response.ToolCall, result)
    }
    return "did not complete"
}
```

This is a working agent. No kernel. No primitives. No gates. Run it on real problems first.

### Stage 1: You Feel Pain — Extract The Interface

After running on real problems you will feel specific pain. Common ones:

- "I can't swap tools without changing the agent code" → extract Tool interface
- "I can't see what happened" → extract the trace
- "It keeps running when it should stop" → extract the budget check
- "I need the same tool in multiple agents" → extract the registry

Each pain point justifies one extraction. Not ten. One.

### Stage 2: The Kernel Emerges

After stage 1 on a few real problems, you will have approximately:
- A common interface for things the loop can call (Primitive)
- A way to pass context through the loop (Envelope)
- A place to look up what tools exist (Registry)
- A way to stop things (Done channel + Budget)

This is the kernel. You didn't design it — you extracted it from real code.

### Stage 3: Gates Emerge From Repeated Defensive Code

You will start writing the same checks before certain calls:
- "Is the path within the allowed directory?" before every file write
- "Is the budget still available?" before every LLM call
- "Does this schema match?" before every agent handoff

When you write the same check three times, extract it to a Gate. Not before.

### Stage 4: The Store Emerges From Shared State Pain

Multiple primitives need to share state. You start passing it explicitly. It gets messy. You extract a Store. The Store starts with Get/Set. Update emerges when you hit race conditions. Watch emerges when you need reactivity. Not before.

---

## The Question To Ask At Every Stage

> "What specific failure in the running system requires this addition?"

If you cannot point to a specific failure — a trace, a log, a wrong output — the addition is speculative. Speculative additions are how you end up with 18 design documents and no running code.

---

## Applying This To Ourselves

Could we have built the elaborate calculus organically using its own philosophy? Yes. It would have looked like:

```
Week 1: For loop + 4 tools + SWE-bench
        Pain found: can't reuse tools across agents
        Addition: Registry

Week 2: Registry working
        Pain found: runaway cost on hard instances
        Addition: Budget in Envelope

Week 3: Budget working
        Pain found: can't see what went wrong in failures
        Addition: Trace in Envelope

Week 4: Trace working, looking at failures
        Pain found: writing to wrong files, no validation
        Addition: Gate on file write edges

...and so on
```

By week 8 you would have something close to the kernel — derived from real failures, not designed from first principles. And you would have skipped the 12 features that sounded good but turned out not to matter.

The for loop is not the starting point before the real work. The for loop IS the real work, until it isn't.
