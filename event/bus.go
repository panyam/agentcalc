package event

// Subscription represents a registered event handler.
type Subscription struct {
	ID       string
	Scope    EventScope
	MinCrit  float64 // minimum criticality to receive
	Handler  func(Event)
}

// EventBus routes events through 4 priority channels based on criticality.
// All events are also appended to an append-only replay log.
type EventBus struct {
	critical  chan Event // criticality > 0.8
	interrupt chan Event // criticality > 0.5
	enqueue   chan Event // criticality > 0.2
	observe   chan Event // criticality <= 0.2
	replay    []Event   // append-only log
	subs      []Subscription
}

// Route sends an event to the appropriate priority channel and appends to replay.
func (bus *EventBus) Route(e Event) {
	panic("not implemented")
}

// Subscribe registers a subscription for events matching the given criteria.
func (bus *EventBus) Subscribe(sub Subscription) {
	panic("not implemented")
}

// Replay returns all events recorded since bus creation, optionally filtered.
func (bus *EventBus) Replay(filter func(Event) bool) []Event {
	panic("not implemented")
}
