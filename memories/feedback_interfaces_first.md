---
name: anti-elaboration discipline
description: User wants minimal kernel — add things only when the running system demonstrates the need, not from speculation
type: feedback
---

Start minimal and grow from real pain. The v0 design had 8 packages and 710 lines of interface stubs — all speculation, no running code. The v1 rewrite is ~505 lines of working kernel.

**Why:** The user experienced "local justification, global complexity" — every addition looked cheap at the margin but the cumulative tax was enormous. 18 design docs and no running code.

**How to apply:** Before adding anything to the kernel, ask: (1) Has the running system actually failed? (2) Can it be a Gate/DeltaGate? (3) Can it be a user-registered Primitive? (4) Does removing it break closure? If question 1 is no, stop. When in doubt, write an example showing the pattern in user-space before adding to the kernel.
