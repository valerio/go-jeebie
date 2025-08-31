package debug

import (
	"fmt"

	"github.com/valerio/go-jeebie/jeebie/disasm"
)

type DisasmLine struct {
	Address     uint16
	Instruction string
	IsCurrent   bool
}

func CreateDisassembly(snapshot *MemorySnapshot, pc uint16, maxLines int) []DisasmLine {
	if snapshot == nil {
		return nil
	}

	pcOffset := -1
	if pc >= snapshot.StartAddr && pc < snapshot.StartAddr+uint16(len(snapshot.Bytes)) {
		pcOffset = int(pc - snapshot.StartAddr)
	}

	// If PC is not in snapshot, just show what we have and mark nothing as current
	if pcOffset < 0 {
		lines := []DisasmLine{}
		for i := 0; i < len(snapshot.Bytes) && len(lines) < maxLines; {
			addr := snapshot.StartAddr + uint16(i)
			instruction, length := disasm.DisassembleBytes(snapshot.Bytes, i)
			lines = append(lines, DisasmLine{
				Address:     addr,
				Instruction: instruction,
				IsCurrent:   addr == pc, // Still mark if it happens to match
			})
			i += length
		}
		// Add a special line indicating PC is outside snapshot
		if len(lines) < maxLines {
			lines = append(lines, DisasmLine{
				Address:     pc,
				Instruction: fmt.Sprintf("[PC: 0x%04X - outside snapshot]", pc),
				IsCurrent:   true,
			})
		}
		return lines
	}

	allLines := []DisasmLine{}

	backwardBytes := 30
	startOffset := pcOffset - backwardBytes
	if startOffset < 0 {
		startOffset = 0
	}

	for i := startOffset; i < len(snapshot.Bytes); {
		addr := snapshot.StartAddr + uint16(i)
		instruction, length := disasm.DisassembleBytes(snapshot.Bytes, i)

		allLines = append(allLines, DisasmLine{
			Address:     addr,
			Instruction: instruction,
			IsCurrent:   addr == pc,
		})

		i += length
		if addr > pc && len(allLines) > maxLines*2 {
			break
		}
	}

	pcIndex := -1
	for i, line := range allLines {
		if line.Address == pc {
			pcIndex = i
			break
		}
	}

	if pcIndex >= 0 {
		halfHeight := maxLines / 2
		startIdx := pcIndex - halfHeight
		endIdx := pcIndex + halfHeight + 1

		if startIdx < 0 {
			startIdx = 0
			endIdx = maxLines
		}
		if endIdx > len(allLines) {
			endIdx = len(allLines)
			startIdx = endIdx - maxLines
			if startIdx < 0 {
				startIdx = 0
			}
		}

		return allLines[startIdx:endIdx]
	}

	if len(allLines) > maxLines {
		return allLines[:maxLines]
	}
	return allLines
}
