package chakra

import "errors"

var (
	ErrNotFound       = errors.New("chakra: primitive not found")
	ErrNotConnected   = errors.New("chakra: primitives not connected")
	ErrDuplicateID    = errors.New("chakra: duplicate primitive ID")
	ErrBudgetExceeded = errors.New("chakra: budget exceeded")
	ErrCancelled      = errors.New("chakra: cancelled")
	ErrGateRejected   = errors.New("chakra: gate rejected invocation")
	ErrDeltaRejected  = errors.New("chakra: delta gate rejected store writes")
	ErrDepthExceeded  = errors.New("chakra: max depth exceeded")
)
