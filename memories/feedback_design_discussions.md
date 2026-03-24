---
name: design-discussion style
description: User likes to reason through architecture collaboratively — present design tensions, not just solutions
type: feedback
---

The user thinks architecturally and wants to stress-test designs through discussion before implementation. When presenting a design choice, surface the tensions and tradeoffs explicitly — don't just present the conclusion. The user will push back with edge cases ("what about async children?", "what about prompt injection?") to verify the design holds.

**Why:** The user is building a calculus, not just software. The formal properties matter as much as the implementation.

**How to apply:** When proposing changes, name what the design gains AND what it gives up. Present edge cases proactively. Be honest when a gap exists rather than hand-waving. The user respects "this is a real gap, here's how we could close it" over "this works fine."
