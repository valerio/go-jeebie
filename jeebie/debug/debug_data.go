package debug

// CPUState contains all CPU register information for debugging
type CPUState struct {
	A uint8
	F uint8
	B uint8
	C uint8
	D uint8
	E uint8
	H uint8
	L uint8

	SP     uint16
	PC     uint16
	IME    bool
	Cycles uint64
}

// MemorySnapshot contains a snapshot of memory for disassembly
type MemorySnapshot struct {
	StartAddr uint16
	Bytes     []uint8
}

// DebuggerState represents the current debugger state
type DebuggerState int

const (
	DebuggerRunning DebuggerState = iota
	DebuggerPaused
	DebuggerStepInstruction
	DebuggerStepFrame
)

// CompleteDebugData contains all debug information needed by debug displays
type CompleteDebugData struct {
	OAM             *OAMData
	VRAM            *VRAMData
	CPU             *CPUState
	Memory          *MemorySnapshot
	DebuggerState   DebuggerState
	InterruptEnable uint8 // IE register at 0xFFFF
	InterruptFlags  uint8 // IF register at 0xFF0F
}
