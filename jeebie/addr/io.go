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
	TAC uint16 = 0xFF
)
