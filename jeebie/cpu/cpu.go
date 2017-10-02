package cpu

import "github.com/valep27/go-jeebie/jeebie/memory"

// Flag is one of the 4 possible flags used in the flag register (high part of AF)
type Flag uint8

const (
	zeroFlag      Flag = 0x80
	subFlag            = 0x40
	halfCarryFlag      = 0x20
	carryFlag          = 0x10
)

// CPU is the main struct holding Z80 state
type CPU struct {
	memory *memory.MMU
	af     Register16
	bc     Register16
	de     Register16
	hl     Register16
	sp     Register16
	pc     Register16
}

// New returns an uninitialized CPU instance
func New() CPU {
	return CPU{}
}

//Tick emulates a single step during the main loop for the cpu.
func (c *CPU) Tick() {

}

func (c *CPU) setFlag(flag Flag) {
	c.af.setLow(uint8(flag))
}

func (c *CPU) resetFlag(flag Flag) {
	c.af.setLow(uint8(flag) ^ 0xFF)
}

func (c CPU) isSetFlag(flag Flag) bool {
	return c.af.getHigh() & uint8(flag) != 0
}
