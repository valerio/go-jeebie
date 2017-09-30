package cpu

func unimplemented(cpu *CPU) int {
	panic("unimplemented opcode was called")
}

func opcode0x00(cpu *CPU) int {
	return 4
}
