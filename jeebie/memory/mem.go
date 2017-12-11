package memory

// MMU allows access to all memory mapped I/O and data/registers
type MMU struct {
}

func New() *MMU {
	return &MMU{}
}

// TODO: implement stubbed methods
func (m *MMU) ReadByte(addr uint16) byte {
	return 0
}

func (m *MMU) WriteByte(addr uint16, value byte) {

}
