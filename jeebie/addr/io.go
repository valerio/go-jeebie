package addr

// gpu registers
const (
	// LCD Control register.
	LCDC uint16 = 0xFF40
	// LCDC Status register.
	STAT uint16 = 0xFF41
	// Scroll Y (SCY) register.
	SCY uint16 = 0xFF42
	// Scroll X (SCX) register.
	SCX uint16 = 0xFF43
	// LCDC Y-Coordinate (readonly) register.
	LY uint16 = 0xFF44
	// LY Compare register.
	LYC uint16 = 0xFF45
	// DMA Transfer and Start register.
	DMA uint16 = 0xFF46
	// BG Palette register.
	BGP uint16 = 0xFF47
	// Object Palette 0 register.
	OBP0 uint16 = 0xFF48
	// Object Palette 1 register.
	OBP1 uint16 = 0xFF49
	// Window Y Position register.
	WY uint16 = 0xFF4A
	// Window X Position register.
	WX uint16 = 0xFF4B
)

// Audio/Sound registers - APU (Audio Processing Unit)
// Reference: https://gbdev.io/pandocs/Audio_Registers.html
const (
	// Audio register range
	AudioStart uint16 = 0xFF10
	AudioEnd   uint16 = 0xFF3F

	// Channel 1 - Square wave with sweep
	NR10 uint16 = 0xFF10 // Channel 1 sweep
	NR11 uint16 = 0xFF11 // Channel 1 length timer & duty cycle
	NR12 uint16 = 0xFF12 // Channel 1 volume & envelope
	NR13 uint16 = 0xFF13 // Channel 1 period low
	NR14 uint16 = 0xFF14 // Channel 1 period high & control

	// Channel 2 - Square wave
	NR21 uint16 = 0xFF16 // Channel 2 length timer & duty cycle
	NR22 uint16 = 0xFF17 // Channel 2 volume & envelope
	NR23 uint16 = 0xFF18 // Channel 2 period low
	NR24 uint16 = 0xFF19 // Channel 2 period high & control

	// Channel 3 - Custom wave
	NR30 uint16 = 0xFF1A // Channel 3 DAC enable
	NR31 uint16 = 0xFF1B // Channel 3 length timer
	NR32 uint16 = 0xFF1C // Channel 3 output level
	NR33 uint16 = 0xFF1D // Channel 3 period low
	NR34 uint16 = 0xFF1E // Channel 3 period high & control

	// Channel 4 - Noise
	NR41 uint16 = 0xFF20 // Channel 4 length timer
	NR42 uint16 = 0xFF21 // Channel 4 volume & envelope
	NR43 uint16 = 0xFF22 // Channel 4 frequency & randomness
	NR44 uint16 = 0xFF23 // Channel 4 control

	// Global sound control
	NR50 uint16 = 0xFF24 // Master volume & VIN panning
	NR51 uint16 = 0xFF25 // Sound panning
	NR52 uint16 = 0xFF26 // Sound on/off and channel status

	// Wave pattern RAM (32 samples, 4-bit each)
	WaveRAMStart uint16 = 0xFF30
	WaveRAMEnd   uint16 = 0xFF3F
)

// OAM (Object Attribute Memory) - sprite data
const (
	// OAMStart is the start of OAM memory (40 sprites * 4 bytes each)
	OAMStart uint16 = 0xFE00
	// OAMEnd is the end of OAM memory
	OAMEnd uint16 = 0xFE9F
)

// tile data and tile maps
const (
	// TileData0 is the start of unsigned tile data (tiles 0-255)
	TileData0 uint16 = 0x8000
	// TileData1 is the start of signed tile data region (tiles -128 to -1)
	TileData1 uint16 = 0x8800
	// TileData2 is the continuation of signed tile data (tiles 0-127)
	TileData2 uint16 = 0x9000

	// TileMap0 is background/window tile map 0
	TileMap0 uint16 = 0x9800
	// TileMap1 is background/window tile map 1
	TileMap1 uint16 = 0x9C00
)

// interrupts
const (
	// IF is the address for the Interrupt Flags register.
	IF uint16 = 0xFF0F
	// IE is the address for the Interrupt Enable register.
	IE uint16 = 0xFFFF
)

// joypad
const (
	// P1 is used to read the Joypad state.
	P1 uint16 = 0xFF00
)

// serial I/O
const (
	// SB (Serial transfer data, 0xFF01)
	//
	// Holds the 8-bit data to be transmitted. During a transfer, bits shift out MSB-first
	// on SO and incoming bits shift in from SI. After completion, SB contains the received
	// byte from the peer (typically 0xFF when no peer is connected).
	SB uint16 = 0xFF01
	// SC (Serial transfer control, 0xFF02)
	//  - Bit 7 (Start): Writing 1 starts an 8-bit transfer; hardware clears to 0 when done.
	//  - Bit 0 (Clock): 1=internal clock (DMG master at ~8192 Hz bit clock), 0=external clock
	//    (peer provides 8 pulses). CGB uses bit 1 for double-speed; ignored on DMG.
	//  - On completion, the Serial interrupt (IF bit 3) is requested by hardware.
	SC uint16 = 0xFF02
)

// timers
const (
	// DIV is the divider register. Incremented 16384 times/s, writing to it resets it.
	DIV uint16 = 0xFF04
	// TIMA is the timer counter register. Generates an interrupt when it overflows.
	TIMA uint16 = 0xFF05
	// TMA is the timer modulo register. When TIMA overflows, this data will be loaded.
	TMA uint16 = 0xFF06
	// TAC is the timer control register. Used to start/stop and control the timer clock.
	TAC uint16 = 0xFF07
)

// Interrupt is an enum that represents one of the possible interrupts.
type Interrupt uint8

const (
	// VBlankInterrupt is fired when the GPU has completed a frame.
	VBlankInterrupt Interrupt = 1
	// LCDSTATInterrupt is fired based on one of the conditions in the LCDSTAT register.
	LCDSTATInterrupt = 1 << 1
	// TimerInterrupt is fired when the timer register (TIMA) overflows (i.e. goes from 0xFF to 0x00).
	TimerInterrupt = 1 << 2
	// SerialInterrupt is fired when a serial transfer has completed on the game link port.
	SerialInterrupt = 1 << 3
	// JoypadInterrupt is fired when any of the keypad inputs goes from high to low.
	JoypadInterrupt = 1 << 4
)
