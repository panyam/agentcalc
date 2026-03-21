package chakra

import (
	"context"
	"sync"
)

// edge represents a connection from one primitive to another,
// with optional gates (pre-invocation) and delta gates (post-invocation).
type edge struct {
	target     string
	gates      []Gate      // fire before Run
	deltaGates []DeltaGate // fire after Run, before delta merge
}

// Registry holds primitives and the edges between them.
// Register adds primitives. Connect creates edges. Invoke calls through edges.
//
// Who calls Connect/Disconnect is not the kernel's concern.
// An orchestrator agent can rewire topology at runtime — Connect/Disconnect
// are capabilities, not policies.
type Registry struct {
	mu    sync.RWMutex
	prims map[string]Primitive
	edges map[string][]edge // from-ID → []edge
}

// NewRegistry creates an empty registry.
func NewRegistry() *Registry {
	return &Registry{
		prims: make(map[string]Primitive),
		edges: make(map[string][]edge),
	}
}

// Register adds a primitive. Returns ErrDuplicateID if the ID is taken.
func (r *Registry) Register(p Primitive) error {
	id := p.Meta().ID
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.prims[id]; exists {
		return ErrDuplicateID
	}
	r.prims[id] = p
	return nil
}

// Connect creates a directed edge from → to with optional pre-invocation gates.
// Both primitives must be registered.
func (r *Registry) Connect(from, to string, gates ...Gate) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.prims[from]; !ok {
		return ErrNotFound
	}
	if _, ok := r.prims[to]; !ok {
		return ErrNotFound
	}
	r.edges[from] = append(r.edges[from], edge{target: to, gates: gates})
	return nil
}

// ConnectWithDelta creates a directed edge from → to with delta gates.
// Delta gates fire after the target primitive returns, before its
// Store writes merge to the parent. Use for namespace enforcement,
// write auditing, etc.
func (r *Registry) ConnectWithDelta(from, to string, dg ...DeltaGate) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.prims[from]; !ok {
		return ErrNotFound
	}
	if _, ok := r.prims[to]; !ok {
		return ErrNotFound
	}
	r.edges[from] = append(r.edges[from], edge{target: to, deltaGates: dg})
	return nil
}

// Disconnect removes all edges from → to. Returns ErrNotConnected
// if no such edge exists.
func (r *Registry) Disconnect(from, to string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	edges := r.edges[from]
	var kept []edge
	for _, e := range edges {
		if e.target != to {
			kept = append(kept, e)
		}
	}
	if len(kept) == len(edges) {
		return ErrNotConnected
	}
	r.edges[from] = kept
	return nil
}

// invokePrep resolves the target primitive and collects gates for an
// invocation. Returns the primitive, gates, delta gates, or an error.
func (r *Registry) invokePrep(caller, target string, msg Message, env Envelope) (Primitive, []Gate, []DeltaGate, error) {
	// Cancellation
	select {
	case <-env.Done:
		return nil, nil, nil, ErrCancelled
	default:
	}
	if env.Budget.Exhausted() {
		return nil, nil, nil, ErrBudgetExceeded
	}
	if env.Budget.MaxDepth > 0 && env.Depth >= env.Budget.MaxDepth {
		return nil, nil, nil, ErrDepthExceeded
	}

	r.mu.RLock()
	prim, ok := r.prims[target]
	if !ok {
		r.mu.RUnlock()
		return nil, nil, nil, ErrNotFound
	}
	edgeList := r.edges[caller]
	r.mu.RUnlock()

	var gates []Gate
	var dGates []DeltaGate
	found := false
	for _, e := range edgeList {
		if e.target == target {
			gates = append(gates, e.gates...)
			dGates = append(dGates, e.deltaGates...)
			found = true
		}
	}
	if !found {
		return nil, nil, nil, ErrNotConnected
	}

	// Run pre-invocation gates
	for _, gate := range gates {
		if err := gate(msg, prim, env); err != nil {
			return nil, nil, nil, err
		}
	}

	return prim, gates, dGates, nil
}

// invokeRun executes a primitive with a scoped store, runs delta gates,
// and merges approved writes. The scoped store's observe channel, if set,
// publishes every write in real time.
func invokeRun(ctx context.Context, prim Primitive, target string, msg Message, env Envelope, dGates []DeltaGate, scoped *ScopedStore) (Message, error) {
	child := Envelope{
		Depth:  env.Depth + 1,
		Budget: env.Budget,
		Trace:  append(append([]string{}, env.Trace...), target),
		Done:   env.Done,
		Store:  scoped,
	}

	result, err := prim.Run(ctx, msg, child)
	if err != nil {
		return result, err
	}

	// Run DeltaGates — validate what the primitive wrote
	delta := scoped.Delta()
	for _, dg := range dGates {
		if err := dg(result, delta, env); err != nil {
			return Message{}, err
		}
	}

	// Auto-merge approved delta to parent store
	for k, v := range delta.Writes {
		env.Store.Set(k, v)
	}

	return result, nil
}

// Invoke calls a target primitive synchronously.
// The primitive gets an isolated ScopedStore. After it returns,
// DeltaGates validate the writes, then approved writes merge to
// the parent Store.
func (r *Registry) Invoke(ctx context.Context, caller string, target string, msg Message, env Envelope) (Message, error) {
	prim, _, dGates, err := r.invokePrep(caller, target, msg, env)
	if err != nil {
		return Message{}, err
	}
	scoped := NewScopedStore(env.Store)
	return invokeRun(ctx, prim, target, msg, env, dGates, scoped)
}

// InvokeResult is the outcome of an async invocation.
type InvokeResult struct {
	Message Message
	Err     error
}

// AsyncHandle is the parent's control surface for an async invocation.
// Result delivers the final outcome. Writes publishes every Store write
// the child makes in real time — the parent can monitor and cancel.
// Close Done to cancel the child.
type AsyncHandle struct {
	Result <-chan InvokeResult  // final result — closed when done
	Writes <-chan StoreWrite    // live store writes from the child
	Done   chan struct{}        // close to cancel this child
}

// InvokeAsync launches a primitive in a goroutine. The parent gets an
// AsyncHandle to monitor writes in real time and cancel if needed.
//
//	h := reg.InvokeAsync(ctx, myID, "slow-tool", msg, env)
//	select {
//	case w := <-h.Writes:
//	    if looksWrong(w) { close(h.Done) }
//	case r := <-h.Result:
//	    use(r.Message)
//	}
func (r *Registry) InvokeAsync(ctx context.Context, caller string, target string, msg Message, env Envelope) (*AsyncHandle, error) {
	prim, _, dGates, err := r.invokePrep(caller, target, msg, env)
	if err != nil {
		return nil, err
	}

	writes := make(chan StoreWrite, 16)
	done := make(chan struct{})
	result := make(chan InvokeResult, 1)

	// Scoped store with observation channel — parent sees every write
	scoped := NewScopedStore(env.Store)
	scoped.observe = writes

	// Child gets its own Done channel so parent can cancel independently
	childEnv := env
	childEnv.Done = done

	go func() {
		defer close(result)
		defer close(writes)
		res, err := invokeRun(ctx, prim, target, msg, childEnv, dGates, scoped)
		result <- InvokeResult{Message: res, Err: err}
	}()

	return &AsyncHandle{Result: result, Writes: writes, Done: done}, nil
}

// Get returns a registered primitive by ID, or nil if not found.
// Convenience for introspection — Invoke is the normal call path.
func (r *Registry) Get(id string) Primitive {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.prims[id]
}

// Edges returns the IDs of all primitives that `from` is connected to.
// Useful for topology-aware agents.
func (r *Registry) Edges(from string) []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	seen := make(map[string]bool)
	var out []string
	for _, e := range r.edges[from] {
		if !seen[e.target] {
			seen[e.target] = true
			out = append(out, e.target)
		}
	}
	return out
}
