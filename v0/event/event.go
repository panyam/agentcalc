// Package event defines the Event type and EventBus — the reactive communication
// backbone of the agent calculus system.
package event

import "time"

// EventSource identifies the origin category of an event.
type EventSource int

const (
	EventSourceAgent EventSource = iota
	EventSourceTool
	EventSourceMemory
	EventSourceGovernor
	EventSourceExternal
)

// EventScope controls the visibility/routing of an event.
type EventScope int

const (
	EventScopeLocal EventScope = iota
	EventScopeGroup
	EventScopeGlobal
)

// Event is the unit of communication in the reactive layer.
type Event struct {
	Source         EventSource
	Criticality   float64 // 0.0–1.0, determines channel routing
	Scope         EventScope
	Payload       any
	Timestamp      time.Time
	IdempotencyKey string
}
