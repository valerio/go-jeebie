package debug

import (
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

	// Check if PC is within the snapshot range
	pcInSnapshot := pc >= snapshot.StartAddr && pc < snapshot.StartAddr+uint16(len(snapshot.Bytes))
	pcOffset := -1
	if pcInSnapshot {
		pcOffset = int(pc - snapshot.StartAddr)
	}

	if !pcInSnapshot {
		lines := []DisasmLine{}
		for i := 0; i < len(snapshot.Bytes) && len(lines) < maxLines-1; {
			addr := snapshot.StartAddr + uint16(i)
			instruction, length := disasm.DisassembleBytes(snapshot.Bytes, i)
			lines = append(lines, DisasmLine{
				Address:     addr,
				Instruction: instruction,
				IsCurrent:   false,
			})
			i += length
		}
		// Add a special line indicating PC is outside snapshot
		lines = append(lines, DisasmLine{
			Address:     pc,
			Instruction: "[PC outside snapshot range]",
			IsCurrent:   true,
		})
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

	// PC should be in snapshot but we didn't find it in our disassembly
	// This can happen if we started disassembling from the middle of an instruction
	// Try to show instructions around where PC should be
	if len(allLines) > 0 {
		// Find the closest instruction to PC
		closestIdx := 0
		closestDist := uint16(0xFFFF)
		for i, line := range allLines {
			var dist uint16
			if line.Address > pc {
				dist = line.Address - pc
			} else {
				dist = pc - line.Address
			}
			if dist < closestDist {
				closestDist = dist
				closestIdx = i
			}
		}

		// Center around the closest instruction
		halfHeight := maxLines / 2
		startIdx := closestIdx - halfHeight
		endIdx := closestIdx + halfHeight + 1

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

	return allLines
}
