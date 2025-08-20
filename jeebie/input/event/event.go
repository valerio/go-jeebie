package event

// Type represents the type of input event
type Type int

const (
	Press   Type = iota // Button pressed down (debounced)
	Release             // Button released (debounced)
	Hold                // Continuous while pressed (not debounced)
)
