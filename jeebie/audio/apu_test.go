package audio

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/valerio/go-jeebie/jeebie/addr"
)

func TestAPU_RegisterMapping(t *testing.T) {
	tests := []struct {
		name     string
		register uint16
		value    uint8
		testFunc func(t *testing.T, apu *APU)
	}{
		{
			name:     "NR52 power control",
			register: addr.NR52, value: 0x80,
			testFunc: func(t *testing.T, apu *APU) {
				assert.True(t, apu.enabled, "APU should be enabled when NR52 bit 7 is set")
			},
		},
		{
			name:     "NR51 panning",
			register: addr.NR51, value: 0xFF, // all channels to both sides
			testFunc: func(t *testing.T, apu *APU) {
				for i := range 4 {
					assert.True(t, apu.ch[i].left, "Channel %d should be panned left", i)
					assert.True(t, apu.ch[i].right, "Channel %d should be panned right", i)
				}
			},
		},
		{
			name:     "NR50 master volume",
			register: addr.NR50, value: 0x77, // max volume both sides
			testFunc: func(t *testing.T, apu *APU) {
				assert.Equal(t, uint8(7), apu.volLeft, "Left volume should be 7")
				assert.Equal(t, uint8(7), apu.volRight, "Right volume should be 7")
			},
		},
		{
			name:     "NR11 duty and length timer",
			register: addr.NR11, value: 0xBF, // duty=2, length timer=63
			testFunc: func(t *testing.T, apu *APU) {
				assert.Equal(t, uint8(2), apu.ch[0].duty, "CH1 duty should be 2")
				assert.Equal(t, uint8(63), apu.ch[0].timer, "CH1 timer should be 63")
			},
		},
		{
			name:     "NR12 volume and envelope",
			register: addr.NR12, value: 0xF7, // vol=15, up=0, pace=7
			testFunc: func(t *testing.T, apu *APU) {
				assert.Equal(t, uint8(15), apu.ch[0].volume, "CH1 volume should be 15")
				assert.False(t, apu.ch[0].envelopeUp, "CH1 envelope should be down")
				assert.Equal(t, uint8(7), apu.ch[0].envelopePace, "CH1 envelope pace should be 7")
				assert.True(t, apu.ch[0].dacEnabled, "CH1 DAC should be enabled (volume > 0)")
			},
		},
		{
			name:     "Wave RAM write/read",
			register: addr.WaveRAMStart, value: 0xAB,
			testFunc: func(t *testing.T, apu *APU) {
				read := apu.ReadRegister(addr.WaveRAMStart)
				assert.Equal(t, uint8(0xAB), read, "Wave RAM should store and return values")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			apu := New()
			// Power on
			apu.WriteRegister(addr.NR52, 0x80)
			apu.WriteRegister(tt.register, tt.value)
			tt.testFunc(t, apu)
		})
	}
}

func TestAPU_ReadMasks(t *testing.T) {
	apu := New()

	// Write-only registers should return 0xFF
	for _, addr := range []uint16{addr.NR13, addr.NR23, addr.NR33, addr.NR41} {
		apu.WriteRegister(addr, 0x00)
		assert.Equal(t, uint8(0xFF), apu.ReadRegister(addr), "Register 0x%X should read as 0xFF (write-only)", addr)
	}
}

func TestAPU_PowerOffLogic(t *testing.T) {
	apu := New()

	// Power on and set up some state
	apu.WriteRegister(addr.NR52, 0x80) // Power on
	apu.WriteRegister(addr.NR10, 0x5E) // CH1 sweep: period=5, down=1, step=6
	apu.WriteRegister(addr.NR11, 0xC3) // CH1: duty=3, length=3
	apu.WriteRegister(addr.NR12, 0xFB) // CH1: volume=15, up=1, pace=3
	apu.WriteRegister(addr.NR50, 0x77) // Master volume: 7/7
	apu.WriteRegister(addr.NR51, 0xFF) // All channels panned to both sides
	apu.WriteRegister(addr.WaveRAMStart, 0xAA)
	apu.WriteRegister(addr.WaveRAMStart+1, 0xBB)

	// Power off
	apu.WriteRegister(addr.NR52, 0x00)
	assert.False(t, apu.enabled, "APU should be disabled")

	// Check that all computed state was cleared
	assert.Equal(t, uint8(0), apu.ch[0].sweepPeriod, "CH1 sweep period should be cleared")
	assert.False(t, apu.ch[0].sweepDown, "CH1 sweep down should be cleared")
	assert.Equal(t, uint8(0), apu.ch[0].sweepStep, "CH1 sweep step should be cleared")
	assert.Equal(t, uint8(0), apu.ch[0].duty, "CH1 duty should be cleared")
	assert.Equal(t, uint8(0), apu.ch[0].volume, "CH1 volume should be cleared")
	assert.False(t, apu.ch[0].envelopeUp, "CH1 envelope up should be cleared")
	assert.Equal(t, uint8(0), apu.volLeft, "Left volume should be cleared")
	assert.Equal(t, uint8(0), apu.volRight, "Right volume should be cleared")
	assert.False(t, apu.ch[0].left, "CH1 left panning should be cleared")
	assert.False(t, apu.ch[0].right, "CH1 right panning should be cleared")
	for i := range 4 {
		assert.False(t, apu.ch[i].enabled, "Channel %d should be disabled", i)
		assert.False(t, apu.ch[i].dacEnabled, "Channel %d DAC should be disabled", i)
	}

	assert.Equal(t, uint8(0xAA), apu.waveRAM[0], "Wave RAM[0] should be preserved")
	assert.Equal(t, uint8(0xBB), apu.waveRAM[1], "Wave RAM[1] should be preserved")

	// Ignore writes while powered off
	apu.WriteRegister(addr.NR10, 0x77)
	apu.WriteRegister(addr.NR50, 0x55)
	assert.Equal(t, uint8(0), apu.ch[0].sweepPeriod, "CH1 sweep should remain 0 (write ignored)")
	assert.Equal(t, uint8(0), apu.volLeft, "Volume should remain 0 (write ignored)")
	// Wave RAM writes still allowed
	apu.WriteRegister(addr.WaveRAMStart+2, 0xCC)
	assert.Equal(t, uint8(0xCC), apu.waveRAM[2], "Wave RAM should be writable while powered off")
	apu.WriteRegister(addr.NR52, 0x80) // Power back on
	assert.True(t, apu.enabled, "APU should be enabled again")

	// Test that registers become writable again after power on
	apu.WriteRegister(addr.NR52, 0x80)
	apu.WriteRegister(addr.NR10, 0x34)
	apu.WriteRegister(addr.NR50, 0x66)
	assert.Equal(t, uint8(3), apu.ch[0].sweepPeriod, "CH1 sweep period should be writable after power on")
	assert.Equal(t, uint8(6), apu.volLeft, "Volume should be writable after power on")
}

// TODO: Add tests for frame sequencer timing
func TestAPU_FrameSequencer(t *testing.T) {
	t.Skip("Frame sequencer not implemented yet")
}

// TODO: Add tests for sample generation
func TestAPU_SampleGeneration(t *testing.T) {
	t.Skip("Sample generation not implemented yet")
}

// TODO: Add tests for trigger behavior
func TestAPU_TriggerBehavior(t *testing.T) {
	t.Skip("Trigger behavior not implemented yet")
}
