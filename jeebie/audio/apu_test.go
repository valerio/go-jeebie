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
	// NR10 bit7 reads as 1; NR11 lower 6 read as 1s
	assert.Equal(t, uint8((0x12&0x7F)|0x80), apu.ReadRegister(addr.NR10))
	assert.Equal(t, uint8((0x34&0xC0)|0x3F), apu.ReadRegister(addr.NR11))

	apu.WriteRegister(addr.NR52, 0x00)

	// When powered off, reads still apply masks to cleared storage
	assert.Equal(t, uint8(0x80), apu.ReadRegister(addr.NR10))
	assert.Equal(t, uint8(0x3F), apu.ReadRegister(addr.NR11))

	assert.Equal(t, uint8(0x70), apu.ReadRegister(addr.NR52))
}

func TestFrameSequencerTiming(t *testing.T) {
	apu := New()

	initialFrame := apu.frameCounter

	apu.Tick(8191)
	assert.Equal(t, initialFrame, apu.frameCounter, "Frame counter should not advance before 8192 cycles")

	apu.Tick(1)
	expectedFrame := (initialFrame + 1) & 7
	assert.Equal(t, expectedFrame, apu.frameCounter, "Frame counter should advance after 8192 cycles")

	for i := 0; i < 7; i++ {
		apu.Tick(8192)
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
		apu.Tick(95)
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
		apu.WriteRegister(addr.WaveRAMStart+uint16(i), val)
	}

	for i, val := range testPattern {
		read := apu.ReadRegister(addr.WaveRAMStart + uint16(i))
		assert.Equal(t, val, read, "Wave RAM should store and return values correctly")
	}
}

func TestAPU_WritesIgnoredWhenPoweredOff(t *testing.T) {
	apu := New()

	// Power off
	apu.WriteRegister(addr.NR52, 0x00)

	// Writes to other registers should be ignored while off
	apu.WriteRegister(addr.NR11, 0xFF)
	// NR11 lower 6 read as 1s even when underlying is 0; expect masked readback
	assert.Equal(t, uint8(0x3F), apu.ReadRegister(addr.NR11), "Writes should be ignored when APU is powered off")
}

func TestWaveRAM_UnaffectedByPowerToggle(t *testing.T) {
	apu := New()

	// Write a known pattern into wave RAM (encode both nibbles by writing even+odd)
	pattern := []uint8{0x12, 0x23, 0x34, 0x45, 0x56, 0x67, 0x78, 0x89}
	for i, v := range pattern {
		apu.WriteRegister(addr.WaveRAMStart+uint16(i), v)
	}

	// Power off
	apu.WriteRegister(addr.NR52, 0x00)

	// Verify wave RAM bytes are unchanged
	for i, v := range pattern {
		got := apu.ReadRegister(addr.WaveRAMStart + uint16(i))
		assert.Equal(t, v, got, "Wave RAM must be unaffected by power off")
	}
}

func TestNR52_ChannelBitsSetOnlyOnTrigger(t *testing.T) {
	apu := New()
	apu.WriteRegister(addr.NR52, 0x80) // power on

	// CH1: enable DAC via NR12, but do NOT trigger
	apu.WriteRegister(addr.NR12, 0xF0)
	status := apu.ReadRegister(addr.NR52)
	assert.Equal(t, uint8(0), status&0x01, "CH1 status must remain off until trigger")

	// CH3: enable DAC via NR30, but do NOT trigger
	apu.WriteRegister(addr.NR30, 0x80)
	status = apu.ReadRegister(addr.NR52)
	assert.Equal(t, uint8(0), status&0x04, "CH3 status must remain off until trigger")
}

func TestChannel1_SweepUpdatesFrequency(t *testing.T) {
	apu := New()
	apu.WriteRegister(addr.NR52, 0x80)

	// Sweep: period=1, increase, shift=1
	apu.WriteRegister(addr.NR10, 0b00010001)

	// Set base frequency and trigger
	apu.WriteRegister(addr.NR13, 0x00)
	apu.WriteRegister(addr.NR14, 0x80)
	before := apu.channels[0].freq

	// Advance past a sweep tick (frame step 2)
	for i := 0; i < 3; i++ {
		apu.Tick(8192)
	}
	after := apu.channels[0].freq
	assert.NotEqual(t, before, after, "Sweep should update CH1 frequency at 128 Hz steps")
}

func TestWave_TriggerPlaybackDelayOutputsLastSample(t *testing.T) {
	apu := New()
	apu.WriteRegister(addr.NR52, 0x80)

	// DAC on, 100% volume
	apu.WriteRegister(addr.NR30, 0x80)
	apu.WriteRegister(addr.NR32, 0b00100000)

	// Minimal non-zero frequency
	apu.WriteRegister(addr.NR33, 0x01)
	apu.WriteRegister(addr.NR34, 0x80) // trigger

	// Immediately produce one sample
	apu.Tick(95)
	s := apu.GetSamples(2)[0]
	assert.Equal(t, int16(0), s, "CH3 should hold last sample (0) immediately after trigger")
}

func TestWave_FirstSampleIsLowerNibble(t *testing.T) {
	apu := New()
	apu.WriteRegister(addr.NR52, 0x80)

	// First wave byte = 0x12 (hi=1, lo=2); write both nibbles
	apu.WriteRegister(addr.WaveRAMStart, 0x12)
	apu.WriteRegister(addr.WaveRAMStart+1, 0x12)

	// 100% volume
	apu.WriteRegister(addr.NR32, 0b00100000)
	apu.WriteRegister(addr.NR30, 0x80) // DAC on

	// Minimal non-zero frequency and trigger
	apu.WriteRegister(addr.NR33, 0x01)
	apu.WriteRegister(addr.NR34, 0x80)

	// Generate enough frames so first fetched nibble is index 1 (lower nibble)
	frames := 70
	for i := 0; i < frames; i++ {
		apu.Tick(95)
	}
	samples := apu.GetSamples(frames * 2)
	lastLeft := samples[len(samples)-2]

	// Expect lower nibble (2): amplitude = (2-8)*2048 = -12288
	assert.Equal(t, int16(-12288), lastLeft, "CH3 must start reading from lower nibble of first byte")
}

func TestPanningAndMasterVolume_AffectStereoOutput(t *testing.T) {
	apu := New()
	apu.WriteRegister(addr.NR52, 0x80)

	// Enable CH1 with constant volume and trigger
	apu.WriteRegister(addr.NR12, 0xF0)
	apu.WriteRegister(addr.NR11, 0x80)
	apu.WriteRegister(addr.NR13, 0x00)
	apu.WriteRegister(addr.NR14, 0x80)

	// Route CH1 to left only; set non-zero master volumes
	apu.WriteRegister(addr.NR51, 0b00010000)
	apu.WriteRegister(addr.NR50, 0b01110111)

	frames := 64
	for i := 0; i < frames; i++ {
		apu.Tick(95)
	}
	samples := apu.GetSamples(frames * 2)

	leftNonZero := false
	rightAllZero := true
	for i := 0; i+1 < len(samples); i += 2 {
		if samples[i] != 0 {
			leftNonZero = true
		}
		if samples[i+1] != 0 {
			rightAllZero = false
			break
		}
	}
	assert.True(t, leftNonZero && rightAllZero, "NR51/NR50 should route sound to left only with right silent")
}

func TestWaveRAM_WriteRedirectWhenActive(t *testing.T) {
	apu := New()
	apu.WriteRegister(addr.NR52, 0x80) // power on

	// Set CH3 DAC on and trigger to mark active
	apu.WriteRegister(addr.NR30, 0x80)
	apu.WriteRegister(addr.NR32, 0b00100000) // full volume
	apu.WriteRegister(addr.NR33, 0x20)
	apu.WriteRegister(addr.NR34, 0x80) // trigger

	// Force current byte index to 5 for deterministic test
	apu.ch3CurrentByteIndex = 5

	// Write to an address that maps to a different index (e.g., index 2)
	targetAddr := addr.WaveRAMStart + 4
	apu.WriteRegister(targetAddr, 0xA0)
	// Since active: write should have affected current byte (index 5) regardless of addressed offset
	got := apu.ReadRegister(addr.WaveRAMStart + 5)
	assert.Equal(t, uint8(0xA0), got)
}

func TestWriteOnlyRegisters_ReadAsFF(t *testing.T) {
	apu := New()
	apu.WriteRegister(addr.NR52, 0x80)

	apu.WriteRegister(addr.NR13, 0x12)
	apu.WriteRegister(addr.NR23, 0x34)
	apu.WriteRegister(addr.NR33, 0x56)

	assert.Equal(t, uint8(0xFF), apu.ReadRegister(addr.NR13))
	assert.Equal(t, uint8(0xFF), apu.ReadRegister(addr.NR23))
	assert.Equal(t, uint8(0xFF), apu.ReadRegister(addr.NR33))
}

func TestLengthReloadOnNR11Write(t *testing.T) {
	apu := New()
	apu.WriteRegister(addr.NR52, 0x80)

	// Trigger CH1 so it is active
	apu.WriteRegister(addr.NR12, 0xF0)
	apu.WriteRegister(addr.NR14, 0x80)

	// Write length to NR11 and ensure counter reloads immediately
	apu.WriteRegister(addr.NR11, 0x80|0x01) // duty=2, length=1 -> counter=63
	assert.Equal(t, uint16(63), apu.channels[0].lengthCounter)

	apu.WriteRegister(addr.NR11, 0x80|0x00) // length=0 -> 64
	assert.Equal(t, uint16(64), apu.channels[0].lengthCounter)
}

func TestDACDisableTurnsChannelOffImmediately(t *testing.T) {
	apu := New()
	apu.WriteRegister(addr.NR52, 0x80)

	// CH1: enable and trigger
	apu.WriteRegister(addr.NR12, 0xF0)
	apu.WriteRegister(addr.NR14, 0x80)
	assert.True(t, apu.channels[0].enabled)
	// Disable DAC -> channel should turn off
	apu.WriteRegister(addr.NR12, 0x00)
	assert.False(t, apu.channels[0].enabled)

	// CH3: enable DAC and trigger
	apu.WriteRegister(addr.NR30, 0x80)
	apu.WriteRegister(addr.NR34, 0x80)
	assert.True(t, apu.channels[2].enabled)
	// Disable DAC -> channel off
	apu.WriteRegister(addr.NR30, 0x00)
	assert.False(t, apu.channels[2].enabled)
}
