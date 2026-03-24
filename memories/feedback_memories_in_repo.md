---
name: memories in repo
description: User wants memory files stored in the repo's memories/ folder, not in ~/.claude, so they're tracked via git
type: feedback
---

Store all memory files in `/Users/dzshrh/projects/chakra/memories/` instead of `~/.claude/projects/.../memory/`. This way memories are version-controlled alongside the code.

**Why:** User wants memories tracked through GitHub, not hidden in a local dotfolder.

**How to apply:** When saving memories for this project, write to `memories/` in the repo root. Update `memories/MEMORY.md` as the index. Commit alongside other changes.
