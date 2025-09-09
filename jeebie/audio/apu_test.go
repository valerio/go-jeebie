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

// TODO: Add tests for power-off logic
func TestAPU_PowerOffLogic(t *testing.T) {
	t.Skip("Power-off logic not implemented yet")
}
