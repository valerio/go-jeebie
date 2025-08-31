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

	// Handle edge case at address 0
	if currentPC == 0 {
		// Can't go backwards from 0, just show forward
		return DisassembleRange(0, beforeCount+1+afterCount, mmu)
	}

	// Simple approach: try different starting points and see which gives us the right number of instructions
	// This is needed because we can't easily go backwards in variable-length instruction sets
	maxOffset := beforeCount * 3
	if uint16(maxOffset) > currentPC {
		maxOffset = int(currentPC)
	}

	for offset := maxOffset; offset >= 0; offset-- {
		testPC := currentPC - uint16(offset)
		if testPC >= currentPC && currentPC != 0 {
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

	// Handle edge case near 0xFFFF
	// If we're near the end of memory and don't have enough instructions after,
	// try to get more instructions before
	if len(lines) < beforeCount+1+afterCount && currentPC > 0x8000 {
		// Try to get more context before if we're running out of space after
		additionalBefore := (beforeCount + 1 + afterCount) - len(lines)
		if additionalBefore > 0 && startPC > uint16(additionalBefore*3) {
			// Try to add more instructions before
			newStartPC := startPC - uint16(additionalBefore*3)
			lines = DisassembleRange(newStartPC, totalCount+additionalBefore, mmu)

			// Find and trim to center around currentPC if possible
			currentIndex := -1
			for i, line := range lines {
				if line.Address == currentPC {
					currentIndex = i
					break
				}
			}

			if currentIndex >= 0 {
				// Try to center, but keep within bounds
				idealStart := currentIndex - beforeCount
				if idealStart < 0 {
					idealStart = 0
				}
				idealEnd := idealStart + beforeCount + 1 + afterCount
				if idealEnd > len(lines) {
					idealEnd = len(lines)
				}
				lines = lines[idealStart:idealEnd]
			}
		}
	}

	return lines
}

// DisassembleBytes disassembles a single instruction from a byte slice
// Returns the formatted instruction string and its length in bytes
// This is useful for disassembling from memory snapshots
func DisassembleBytes(bytes []uint8, offset int) (string, int) {
	if offset >= len(bytes) {
		return "???", 1
	}

	opcode := bytes[offset]
	if opcode == 0xCB {
		if offset+1 < len(bytes) {
			cbOpcode := bytes[offset+1]
			return CBInstructionTemplates[cbOpcode], CBInstructionLengths[cbOpcode]
		}
		return "CB ???", 1
	}

	template := InstructionTemplates[opcode]
	length := InstructionLengths[opcode]
	switch length {
	case 1:
		return template, length
	case 2:
		if offset+1 < len(bytes) {
			n := bytes[offset+1]
			return fmt.Sprintf(template, n), length
		}
		return fmt.Sprintf(template, 0), 1
	case 3:
		if offset+2 < len(bytes) {
			low := bytes[offset+1]
			high := bytes[offset+2]
			nn := uint16(high)<<8 | uint16(low)
			return fmt.Sprintf(template, nn), length
		} else if offset+1 < len(bytes) {
			return fmt.Sprintf(template, 0), 2
		}
		return fmt.Sprintf(template, 0), 1
	default:
		return template, length
	}
}

// FormatDisassemblyLine formats a disassembly line for display
func FormatDisassemblyLine(line DisassemblyLine, isCurrentPC bool) string {
	prefix := " "
	if isCurrentPC {
		prefix = "→"
	}

	return fmt.Sprintf("%s0x%04X: %s", prefix, line.Address, line.Instruction)
}
