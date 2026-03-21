package chakra

import "sync"

// Store is shared state, opaque to the kernel.
// Four methods. Not five.
type Store interface {
	Get(key string) (any, bool)
	Set(key string, value any)
	Update(key string, fn func(any) any) // atomic read-modify-write
	Watch(key string) <-chan any
}

// MemStore is the default in-memory Store implementation.
// Needed to run anything at all.
type MemStore struct {
	mu       sync.RWMutex
	data     map[string]any
	watchers map[string][]chan any
}

// NewMemStore creates a ready-to-use in-memory store.
func NewMemStore() *MemStore {
	return &MemStore{
		data:     make(map[string]any),
		watchers: make(map[string][]chan any),
	}
}

func (s *MemStore) Get(key string) (any, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.data[key]
	return v, ok
}

func (s *MemStore) Set(key string, value any) {
	s.mu.Lock()
	s.data[key] = value
	chs := s.watchers[key]
	s.mu.Unlock()
	for _, ch := range chs {
		select {
		case ch <- value:
		default: // non-blocking — drop if watcher is slow
		}
	}
}

func (s *MemStore) Update(key string, fn func(any) any) {
	s.mu.Lock()
	old := s.data[key]
	nv := fn(old)
	s.data[key] = nv
	chs := s.watchers[key]
	s.mu.Unlock()
	for _, ch := range chs {
		select {
		case ch <- nv:
		default:
		}
	}
}

func (s *MemStore) Watch(key string) <-chan any {
	s.mu.Lock()
	defer s.mu.Unlock()
	ch := make(chan any, 8)
	s.watchers[key] = append(s.watchers[key], ch)
	return ch
}

// StoreWrite is a single write captured by a ScopedStore.
// Published to the observation channel so the parent can monitor
// a child's writes in real time without breaking isolation.
type StoreWrite struct {
	Key   string
	Value any
}

// ScopedStore isolates a primitive's Store access. Reads fall through
// to the parent. Writes are captured locally. Nothing reaches the
// parent until explicitly merged by the caller (the kernel's Invoke).
//
// If an observation channel is set, every write is published to it.
// The primitive doesn't know it's being observed — it just calls Set().
// The parent drains the channel and can cancel the child if the writes
// look wrong.
type ScopedStore struct {
	parent  Store
	writes  map[string]any    // captured locally, not committed
	reads   []string          // audit trail: what this invocation read
	observe chan<- StoreWrite // optional: live write observation
}

// NewScopedStore wraps a parent Store with an isolated write scope.
func NewScopedStore(parent Store) *ScopedStore {
	return &ScopedStore{
		parent: parent,
		writes: make(map[string]any),
	}
}

// Get checks local writes first, then falls through to the parent.
// Parent reads are recorded for audit.
func (s *ScopedStore) Get(key string) (any, bool) {
	if v, ok := s.writes[key]; ok {
		return v, true
	}
	s.reads = append(s.reads, key)
	return s.parent.Get(key)
}

// Set writes to the local scope only. The parent Store is not touched.
// If an observer is attached, the write is published non-blocking.
func (s *ScopedStore) Set(key string, value any) {
	s.writes[key] = value
	if s.observe != nil {
		select {
		case s.observe <- StoreWrite{Key: key, Value: value}:
		default: // non-blocking — don't slow the primitive
		}
	}
}

// Update reads from local (or parent), applies fn, writes to local scope.
func (s *ScopedStore) Update(key string, fn func(any) any) {
	old, _ := s.Get(key) // uses local-then-parent read
	s.Set(key, fn(old))  // goes through Set so observer sees it
}

// Watch delegates to the parent Store. Watchers see committed
// (merged) state, not uncommitted local writes.
func (s *ScopedStore) Watch(key string) <-chan any {
	return s.parent.Watch(key)
}

// Delta returns what this scope read and wrote.
func (s *ScopedStore) Delta() StoreDelta {
	return StoreDelta{
		Reads:  s.reads,
		Writes: s.writes,
	}
}
