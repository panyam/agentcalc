package primitive

import "encoding/json"

// Input represents data flowing into a Primitive.Execute call.
type Input struct {
	// Data is the untyped payload — heterogeneous composition over generics.
	Data any

	// Schema is a JSON Schema describing the expected shape of Data.
	Schema json.RawMessage

	// Metadata carries arbitrary key-value context about this input.
	Metadata map[string]any
}

// Output represents data flowing out of a Primitive.Execute call.
type Output struct {
	// Data is the untyped result payload.
	Data any

	// Schema is a JSON Schema describing the shape of Data.
	Schema json.RawMessage

	// Metadata carries arbitrary key-value context about this output.
	Metadata map[string]any
}
