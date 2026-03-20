package memory

// ContextWindow is the L1 in-memory working context for an agent instance.
// It holds the current session's state and is the fastest tier in the hierarchy.
type ContextWindow struct {
	entries  []ContextEntry
	maxSize  int
}

// ContextEntry is a single item in the context window.
type ContextEntry struct {
	Role    string // "system", "user", "assistant", "tool"
	Content any
}

// Clone returns a deep copy of the context window.
func (cw ContextWindow) Clone() ContextWindow {
	entries := make([]ContextEntry, len(cw.entries))
	copy(entries, cw.entries)
	return ContextWindow{
		entries: entries,
		maxSize: cw.maxSize,
	}
}
