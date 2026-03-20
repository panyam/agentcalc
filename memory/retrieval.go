package memory

// RetrievalResult represents a single result from a memory query, with a
// confidence score for ranking.
type RetrievalResult struct {
	Key        string
	Value      any
	Level      MemoryLevel
	Confidence float64 // 0.0–1.0
}

// RetrievalStack manages queries across multiple memory levels with automatic
// escalation — if L1 doesn't have a confident result, escalate to L2, etc.
type RetrievalStack struct {
	stores   map[MemoryLevel]MemoryStore
	minConf  float64 // minimum confidence before escalating
}
