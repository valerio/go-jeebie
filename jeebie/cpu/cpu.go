package cpu

import "github.com/valep27/go-jeebie/jeebie/memory"

// CPU is the main struct holding Z80 state
type CPU struct {
	memory *memory.MMU
}

//Tick emulates a single step during the main loop for the cpu.
func (c *CPU) Tick() {

}
