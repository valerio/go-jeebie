package audio

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/valerio/go-jeebie/jeebie/addr"
)

func TestAPUPowerControl(t *testing.T) {
	apu := New()

	apu.WriteRegister(addr.NR10, 0x12)
	apu.WriteRegister(addr.NR11, 0x34)
	assert.Equal(t, uint8(0x12), apu.ReadRegister(addr.NR10))
	assert.Equal(t, uint8(0x34), apu.ReadRegister(addr.NR11))

	apu.WriteRegister(addr.NR52, 0x00)

	assert.Equal(t, uint8(0x00), apu.ReadRegister(addr.NR10))
	assert.Equal(t, uint8(0x00), apu.ReadRegister(addr.NR11))

	assert.Equal(t, uint8(0x70), apu.ReadRegister(addr.NR52))
}

func TestFrameSequencerTiming(t *testing.T) {
	apu := New()

	initialFrame := apu.frameCounter

	apu.Step(8191)
	assert.Equal(t, initialFrame, apu.frameCounter, "Frame counter should not advance before 8192 cycles")

	apu.Step(1)
	expectedFrame := (initialFrame + 1) & 7
	assert.Equal(t, expectedFrame, apu.frameCounter, "Frame counter should advance after 8192 cycles")

	for i := 0; i < 7; i++ {
		apu.Step(8192)
	}
	assert.Equal(t, initialFrame, apu.frameCounter, "Frame counter should wrap around after 8 steps")
}

func TestBasicSampleGeneration(t *testing.T) {
	apu := New()

	apu.WriteRegister(addr.NR52, 0x80)
	apu.WriteRegister(addr.NR12, 0xF0)
	apu.WriteRegister(addr.NR11, 0x80)
	apu.WriteRegister(addr.NR13, 0x00)
	apu.WriteRegister(addr.NR14, 0x87)

	for i := 0; i < 100; i++ {
		apu.Step(95)
	}

	samples := apu.GetSamples(100)

	hasNonZero := false
	for _, sample := range samples {
		if sample != 0 {
			hasNonZero = true
			break
		}
	}
	assert.True(t, hasNonZero, "Should generate non-zero samples when channel is active")
}

func TestRegisterMasking(t *testing.T) {
	apu := New()

	apu.WriteRegister(addr.NR10, 0xFF)
	assert.Equal(t, uint8(0xFF), apu.ReadRegister(addr.NR10))

	apu.WriteRegister(addr.NR52, 0xFF)
	status := apu.ReadRegister(addr.NR52)
	assert.Equal(t, uint8(0xF0), status&0xF0, "Upper bits should be readable")
	assert.Equal(t, uint8(0x70), status&0x70, "Unused bits should always read as 1")
}

func TestWaveRAMAccess(t *testing.T) {
	apu := New()

	testPattern := []uint8{0x01, 0x23, 0x45, 0x67, 0x89, 0xAB, 0xCD, 0xEF}

	for i, val := range testPattern {
		apu.WriteRegister(addr.WaveRAMStart+uint16(i*2), val)
		apu.WriteRegister(addr.WaveRAMStart+uint16(i*2+1), val)
	}

	for i, val := range testPattern {
		read := apu.ReadRegister(addr.WaveRAMStart + uint16(i*2))
		assert.Equal(t, val, read, "Wave RAM should store and return values correctly")
	}
}
