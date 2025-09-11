package audio

// Timing constants
// Reference: https://gbdev.io/pandocs/Audio_details.html
const (
	// cyclesPerStep is the number of CPU cycles per frame sequencer tick.
	// The frame sequencer runs at 512 Hz: 4194304 Hz / 512 Hz = 8192 t-cycles
	cyclesPerStep = 8192
)

// Channel constants
const (
	// waveRAMSize is the size of wave pattern RAM in bytes (16 bytes = 32 nibbles)
	waveRAMSize = 16
)
