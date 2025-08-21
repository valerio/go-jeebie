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
	// SB is the Serial Bus register, used to read/write data to be transferred.
	SB uint16 = 0xFF01
	// SC is the Serial Control register, used to control transmission on the serial port.
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
