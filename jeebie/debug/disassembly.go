package debug

import (
	"github.com/valerio/go-jeebie/jeebie/disasm"
)

type DisasmLine struct {
	Address     uint16
	Instruction string
	IsCurrent   bool
}

// DisasmBuffer holds pre-allocated buffers for disassembly lines
type DisasmBuffer struct {
	Lines    []DisasmLine
	AllLines []DisasmLine
}

func NewDisasmBuffer(maxLines int) *DisasmBuffer {
	return &DisasmBuffer{
		Lines:    make([]DisasmLine, 0, maxLines),
		AllLines: make([]DisasmLine, 0, maxLines*3), // Extra space for context
	}
}

func CreateDisassembly(snapshot *MemorySnapshot, pc uint16, maxLines int) []DisasmLine {
	buf := NewDisasmBuffer(maxLines)
	return CreateDisassemblyWithBuffer(snapshot, pc, maxLines, buf)
}

func CreateDisassemblyWithBuffer(snapshot *MemorySnapshot, pc uint16, maxLines int, buf *DisasmBuffer) []DisasmLine {
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
		// Reset buffer and reuse it
		buf.Lines = buf.Lines[:0]
		for i := 0; i < len(snapshot.Bytes) && len(buf.Lines) < maxLines-1; {
			addr := snapshot.StartAddr + uint16(i)
			instruction, length := disasm.DisassembleBytes(snapshot.Bytes, i)
			buf.Lines = append(buf.Lines, DisasmLine{
				Address:     addr,
				Instruction: instruction,
				IsCurrent:   false,
			})
			i += length
		}
		// Add a special line indicating PC is outside snapshot
		buf.Lines = append(buf.Lines, DisasmLine{
			Address:     pc,
			Instruction: "[PC outside snapshot range]",
			IsCurrent:   true,
		})
		return buf.Lines
	}

	// Reset buffer and reuse it
	buf.AllLines = buf.AllLines[:0]

	backwardBytes := 30
	startOffset := pcOffset - backwardBytes
	if startOffset < 0 {
		startOffset = 0
	}

	for i := startOffset; i < len(snapshot.Bytes); {
		addr := snapshot.StartAddr + uint16(i)
		instruction, length := disasm.DisassembleBytes(snapshot.Bytes, i)

		buf.AllLines = append(buf.AllLines, DisasmLine{
			Address:     addr,
			Instruction: instruction,
			IsCurrent:   addr == pc,
		})

		i += length
		if addr > pc && len(buf.AllLines) > maxLines*2 {
			break
		}
	}

	pcIndex := -1
	for i, line := range buf.AllLines {
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
		if endIdx > len(buf.AllLines) {
			endIdx = len(buf.AllLines)
			startIdx = endIdx - maxLines
			if startIdx < 0 {
				startIdx = 0
			}
		}

		// Reset Lines buffer and copy the visible range
		buf.Lines = buf.Lines[:0]
		buf.Lines = append(buf.Lines, buf.AllLines[startIdx:endIdx]...)
		return buf.Lines
	}

	// PC should be in snapshot but we didn't find it in our disassembly
	// This can happen if we started disassembling from the middle of an instruction
	// Try to show instructions around where PC should be
	if len(buf.AllLines) > 0 {
		// Find the closest instruction to PC
		closestIdx := 0
		closestDist := uint16(0xFFFF)
		for i, line := range buf.AllLines {
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
		if endIdx > len(buf.AllLines) {
			endIdx = len(buf.AllLines)
			startIdx = endIdx - maxLines
			if startIdx < 0 {
				startIdx = 0
			}
		}

		// Reset Lines buffer and copy the visible range
		buf.Lines = buf.Lines[:0]
		buf.Lines = append(buf.Lines, buf.AllLines[startIdx:endIdx]...)
		return buf.Lines
	}

	return buf.AllLines
}
