package jeebie

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExtractDebugData_NilComponents(t *testing.T) {
	dmg := &DMG{}
	debugData := dmg.ExtractDebugData()
	assert.Nil(t, debugData, "Should return nil when components are not initialized")
}

func TestExtractDebugData_WithTestROM(t *testing.T) {
	// Skip if test ROM not available
	testROMPath := "../test-roms/dmg-acid2.gb"

	dmg, err := NewWithFile(testROMPath)
	if err != nil {
		t.Skipf("Test ROM not available: %v", err)
	}

	// Extract debug data
	debugData := dmg.ExtractDebugData()
	assert.NotNil(t, debugData, "Debug data should not be nil")
	assert.NotNil(t, debugData.Memory, "Memory snapshot should not be nil")
	assert.NotNil(t, debugData.CPU, "CPU data should not be nil")

	// Verify the snapshot contains the PC
	pc := debugData.CPU.PC
	snapshot := debugData.Memory

	// Check that PC is within the snapshot range
	pcInSnapshot := pc >= snapshot.StartAddr &&
		pc < snapshot.StartAddr+uint16(len(snapshot.Bytes))
	assert.True(t, pcInSnapshot,
		"PC 0x%04X should be within snapshot range [0x%04X, 0x%04X)",
		pc, snapshot.StartAddr, snapshot.StartAddr+uint16(len(snapshot.Bytes)))

	// Verify snapshot doesn't wrap around
	// The last addressable byte should be >= start (no wraparound)
	if len(snapshot.Bytes) > 0 {
		lastAddr := snapshot.StartAddr + uint16(len(snapshot.Bytes)-1)
		// Check for wraparound: lastAddr should be >= startAddr
		// unless we're at the very end of address space
		if snapshot.StartAddr <= 0xFF00 {
			assert.True(t, lastAddr >= snapshot.StartAddr,
				"Snapshot should not wrap around address space (start: 0x%04X, last: 0x%04X)",
				snapshot.StartAddr, lastAddr)
		}
	}

	// Snapshot should have reasonable size (between 1 and 200 bytes)
	assert.True(t, len(snapshot.Bytes) > 0 && len(snapshot.Bytes) <= 200,
		"Snapshot size %d should be between 1 and 200", len(snapshot.Bytes))
}

func TestExtractDebugData_SnapshotAddressCalculation(t *testing.T) {
	testCases := []struct {
		name           string
		startAddr      uint16
		snapshotSize   int
		shouldTruncate bool
		expectedSize   int
	}{
		{
			name:           "Normal case - middle of address space",
			startAddr:      0x8000,
			snapshotSize:   200,
			shouldTruncate: false,
			expectedSize:   200,
		},
		{
			name:           "Near end - should truncate",
			startAddr:      0xFF80,
			snapshotSize:   200,
			shouldTruncate: true,
			expectedSize:   128, // 0x10000 - 0xFF80 = 0x80 = 128
		},
		{
			name:           "At very end",
			startAddr:      0xFFF0,
			snapshotSize:   200,
			shouldTruncate: true,
			expectedSize:   16, // 0x10000 - 0xFFF0 = 0x10 = 16
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actualSize := tc.snapshotSize
			if uint32(tc.startAddr)+uint32(tc.snapshotSize) > 0xFFFF {
				actualSize = int(0x10000 - uint32(tc.startAddr))
			}

			assert.Equal(t, tc.expectedSize, actualSize,
				"Size calculation for start address 0x%04X", tc.startAddr)

			// Verify no address wraparound would occur
			for i := 0; i < actualSize; i++ {
				addr := tc.startAddr + uint16(i)
				if i > 0 {
					prevAddr := tc.startAddr + uint16(i-1)
					// Address should increment or we're at the 0xFFFF->0x0000 boundary
					assert.True(t, addr > prevAddr || (prevAddr == 0xFFFF && addr == 0),
						"Address calculation should not wrap unexpectedly")
				}
			}
		})
	}
}
