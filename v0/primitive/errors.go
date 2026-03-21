package primitive

import "errors"

var (
	// ErrBudgetExceeded is returned when a resource limit is hit.
	ErrBudgetExceeded = errors.New("primitive: budget exceeded")

	// ErrInterrupted is returned when a primitive is interrupted mid-execution.
	ErrInterrupted = errors.New("primitive: interrupted")

	// ErrRollbackFailed is returned when state restoration fails.
	ErrRollbackFailed = errors.New("primitive: rollback failed")

	// ErrNotReversible is returned when Rollback is called on an irreversible primitive.
	ErrNotReversible = errors.New("primitive: not reversible")

	// ErrInvalidInput is returned when input fails schema validation.
	ErrInvalidInput = errors.New("primitive: invalid input")

	// ErrInvalidState is returned when a checkpoint cannot be deserialized.
	ErrInvalidState = errors.New("primitive: invalid state")

	// ErrNotFound is returned when a requested primitive does not exist.
	ErrNotFound = errors.New("primitive: not found")

	// ErrBedrockImmutable is returned on an attempt to swap a bedrock primitive.
	ErrBedrockImmutable = errors.New("primitive: bedrock primitive cannot be swapped")

	// ErrIncompatibleStructure is returned when a swap fails structural type checking.
	ErrIncompatibleStructure = errors.New("primitive: incompatible structural type")

	// ErrIncompatibleSemantics is returned when a swap fails semantic type checking.
	ErrIncompatibleSemantics = errors.New("primitive: incompatible semantic type")
)
