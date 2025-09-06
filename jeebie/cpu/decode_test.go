package cpu

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/valerio/go-jeebie/jeebie/memory"
)

func TestDecode(t *testing.T) {
	tests := []struct {
		name           string
		memorySetup    map[uint16]uint8
		pc             uint16
		expectedOpcode uint16
	}{
		{
			name: "NOP",
			memorySetup: map[uint16]uint8{
				0xC000: 0x00,
			},
			pc:             0xC000,
			expectedOpcode: 0x00,
		},
		{
			name: "INC B",
			memorySetup: map[uint16]uint8{
				0xC000: 0x04,
			},
			pc:             0xC000,
			expectedOpcode: 0x04,
		},
		{
			name: "CB BIT 0,B",
			memorySetup: map[uint16]uint8{
				0xC000: 0xCB,
				0xC001: 0x40,
			},
			pc:             0xC000,
			expectedOpcode: 0xCB40,
		},
		{
			name: "CB SET 7,A",
			memorySetup: map[uint16]uint8{
				0xC000: 0xCB,
				0xC001: 0xFF,
			},
			pc:             0xC000,
			expectedOpcode: 0xCBFF,
		},
		{
			name: "CB at page boundary",
			memorySetup: map[uint16]uint8{
				0xC0FF: 0xCB,
				0xC100: 0x80,
			},
			pc:             0xC0FF,
			expectedOpcode: 0xCB80,
		},
		{
			name: "LD B,0xCB (not CB prefix)",
			memorySetup: map[uint16]uint8{
				0xC000: 0x06, // LD B,n
				0xC001: 0xCB, // immediate value
			},
			pc:             0xC000,
			expectedOpcode: 0x06,
		},
		{
			name: "HALT",
			memorySetup: map[uint16]uint8{
				0xC000: 0x76,
			},
			pc:             0xC000,
			expectedOpcode: 0x76,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mmu := memory.New()
			cpu := &CPU{
				bus: mmu,
				pc:  tt.pc,
			}

			for addr, value := range tt.memorySetup {
				mmu.Write(addr, value)
			}

			initialPC := cpu.pc
			opcode := Decode(cpu)

			assert.Equal(t, initialPC, cpu.pc, "PC should not change")
			assert.Equal(t, tt.expectedOpcode, cpu.currentOpcode)
			assert.NotNil(t, opcode)
		})
	}
}
