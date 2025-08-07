package disasm

import (
	"fmt"

	"github.com/valerio/go-jeebie/jeebie/bit"
	"github.com/valerio/go-jeebie/jeebie/memory"
)

//go:generate go run generate.go

// DisassemblyLine represents a single disassembled instruction
type DisassemblyLine struct {
	Address     uint16
	Instruction string
	Length      int
}

// DisassembleAt disassembles the instruction at the given program counter
func DisassembleAt(pc uint16, mmu *memory.MMU) DisassemblyLine {
	opcode := mmu.Read(pc)
	
	if opcode == 0xCB {
		// Handle CB-prefixed instructions
		if pc == 0xFFFF {
			return DisassemblyLine{
				Address:     pc,
				Instruction: "CB ??",
				Length:      2,
			}
		}
		
		cbOpcode := mmu.Read(pc + 1)
		length := CBInstructionLengths[cbOpcode]
		template := CBInstructionTemplates[cbOpcode]
		
		instruction := fmt.Sprintf(template)
		
		return DisassemblyLine{
			Address:     pc,
			Instruction: instruction,
			Length:      length,
		}
	}
	
	// Handle regular instructions
	length := InstructionLengths[opcode]
	template := InstructionTemplates[opcode]
	
	var instruction string
	
	// Format with immediate values based on length
	switch length {
	case 1:
		instruction = fmt.Sprintf(template)
	case 2:
		if pc == 0xFFFF {
			instruction = fmt.Sprintf(template, 0)
		} else {
			n := mmu.Read(pc + 1)
			instruction = fmt.Sprintf(template, n)
		}
	case 3:
		if pc >= 0xFFFE {
			instruction = fmt.Sprintf(template, 0)
		} else {
			n := mmu.Read(pc + 1)
			nn := bit.Combine(mmu.Read(pc+2), n)
			instruction = fmt.Sprintf(template, nn)
		}
	default:
		instruction = fmt.Sprintf(template)
	}
	
	return DisassemblyLine{
		Address:     pc,
		Instruction: instruction,
		Length:      length,
	}
}

// DisassembleRange disassembles multiple instructions starting from the given PC
func DisassembleRange(startPC uint16, count int, mmu *memory.MMU) []DisassemblyLine {
	lines := make([]DisassemblyLine, 0, count)
	pc := startPC
	
	for i := 0; i < count && pc <= 0xFFFF; i++ {
		line := DisassembleAt(pc, mmu)
		lines = append(lines, line)
		pc += uint16(line.Length)
	}
	
	return lines
}

// DisassembleAround disassembles instructions around the given PC
// Returns instructions before, at, and after the PC
func DisassembleAround(currentPC uint16, beforeCount, afterCount int, mmu *memory.MMU) []DisassemblyLine {
	// Find the starting PC by working backwards
	startPC := currentPC
	instructionsFound := 0
	
	// Simple approach: try different starting points and see which gives us the right number of instructions
	// This is needed because we can't easily go backwards in variable-length instruction sets
	for offset := beforeCount * 3; offset >= 0 && startPC > uint16(offset); offset-- {
		testPC := currentPC - uint16(offset)
		if testPC >= currentPC {
			break
		}
		
		// Try disassembling from this point and see if we hit currentPC
		pc := testPC
		count := 0
		
		for count < beforeCount*2 && pc <= currentPC {
			if pc == currentPC {
				// Found the right starting point
				if count >= beforeCount {
					startPC = testPC
					instructionsFound = count
					break
				}
			}
			
			line := DisassembleAt(pc, mmu)
			pc += uint16(line.Length)
			count++
		}
		
		if startPC != currentPC {
			break
		}
	}
	
	// If we couldn't find a good starting point, just start from currentPC
	if startPC == currentPC {
		instructionsFound = 0
	}
	
	// Disassemble from the found starting point
	totalCount := instructionsFound + 1 + afterCount // before + current + after
	lines := DisassembleRange(startPC, totalCount, mmu)
	
	return lines
}

// FormatDisassemblyLine formats a disassembly line for display
func FormatDisassemblyLine(line DisassemblyLine, isCurrentPC bool) string {
	prefix := " "
	if isCurrentPC {
		prefix = "→"
	}
	
	return fmt.Sprintf("%s0x%04X: %s", prefix, line.Address, line.Instruction)
}