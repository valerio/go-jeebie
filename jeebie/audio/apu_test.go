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

	apu.WriteRegister(addr.NR52, 0x80)

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

func TestAPU_SweepCalculation(t *testing.T) {
	tests := []struct {
		name         string
		shadowFreq   uint16
		sweepStep    uint8
		sweepDown    bool
		wantFreq     uint16
		wantOverflow bool
	}{
		{
			name:       "no shift returns same frequency",
			shadowFreq: 1024, sweepStep: 0, sweepDown: false,
			wantFreq: 1024, wantOverflow: false,
		},
		{
			name:       "sweep up increases period",
			shadowFreq: 1024, sweepStep: 1, sweepDown: false,
			wantFreq: 1536, wantOverflow: false, // 1024 + (1024 >> 1)
		},
		{
			name:       "sweep down decreases period",
			shadowFreq: 1024, sweepStep: 2, sweepDown: true,
			wantFreq: 768, wantOverflow: false, // 1024 - (1024 >> 2)
		},
		{
			name:       "overflow detection",
			shadowFreq: 2000, sweepStep: 3, sweepDown: false,
			wantFreq: 2250, wantOverflow: true, // 2000 + (2000 >> 3) = 2250 > 2047
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ch := &Channel{
				shadowFreq: tt.shadowFreq,
				sweepStep:  tt.sweepStep,
				sweepDown:  tt.sweepDown,
			}
			gotFreq, gotOverflow := ch.calculateSweepFrequency()
			assert.Equal(t, tt.wantFreq, gotFreq)
			assert.Equal(t, tt.wantOverflow, gotOverflow)
		})
	}
}

func TestAPU_SweepTrigger(t *testing.T) {
	apu := New()
	apu.WriteRegister(addr.NR52, 0x80) // Enable APU
	apu.WriteRegister(addr.NR10, 0x11) // period=1, up, shift=1
	apu.WriteRegister(addr.NR13, 0x00)
	apu.WriteRegister(addr.NR14, 0x04) // Frequency = 0x400
	apu.WriteRegister(addr.NR12, 0xF0) // Enable DAC
	apu.WriteRegister(addr.NR14, 0x84) // Trigger

	assert.True(t, apu.ch[0].sweepEnabled)
	assert.Equal(t, uint16(1024), apu.ch[0].shadowFreq)
	assert.Equal(t, uint8(1), apu.ch[0].sweepTimer)
	assert.True(t, apu.ch[0].enabled)
}

func TestAPU_SweepOverflow(t *testing.T) {
	apu := New()
	apu.WriteRegister(addr.NR52, 0x80)
	apu.WriteRegister(addr.NR10, 0x01) // period=0, up, shift=1
	apu.WriteRegister(addr.NR13, 0xFF)
	apu.WriteRegister(addr.NR14, 0x07) // Frequency = 0x7FF (2047)
	apu.WriteRegister(addr.NR12, 0xF0)
	apu.WriteRegister(addr.NR14, 0x87) // Trigger

	assert.False(t, apu.ch[0].enabled, "Channel should be disabled due to overflow")
}

func TestAPU_SweepTimer(t *testing.T) {
	apu := New()
	apu.WriteRegister(addr.NR52, 0x80)
	apu.WriteRegister(addr.NR10, 0x21) // period=2, up, shift=1
	apu.WriteRegister(addr.NR13, 0x00)
	apu.WriteRegister(addr.NR14, 0x04) // Frequency = 0x400
	apu.WriteRegister(addr.NR12, 0xF0)
	apu.WriteRegister(addr.NR14, 0x84) // Trigger

	initialFreq := apu.ch[0].shadowFreq

	apu.tickSweep()
	assert.Equal(t, initialFreq, apu.ch[0].shadowFreq, "No change on first tick")

	apu.tickSweep()
	expectedFreq := uint16(1536) // 1024 + 512
	assert.Equal(t, expectedFreq, apu.ch[0].shadowFreq)
	assert.Equal(t, expectedFreq, apu.ch[0].period)

	reconstructed := uint16(apu.NR14&0x07)<<8 | uint16(apu.NR13)
	assert.Equal(t, expectedFreq, reconstructed, "NR13/NR14 should be updated")
}

func TestAPU_LengthTimer(t *testing.T) {
	apu := New()
	apu.WriteRegister(addr.NR52, 0x80)
	apu.WriteRegister(addr.NR12, 0xF0)
	apu.WriteRegister(addr.NR11, 0x3F) // Length timer = 63
	apu.WriteRegister(addr.NR14, 0xC0) // Trigger with length enable

	assert.Equal(t, uint16(1), apu.ch[0].length) // 64 - 63 = 1
	assert.True(t, apu.ch[0].enabled)

	apu.tickLength()
	assert.False(t, apu.ch[0].enabled, "Channel should be disabled after length expires")
}

func TestAPU_CH3LengthTimer(t *testing.T) {
	apu := New()
	apu.WriteRegister(addr.NR52, 0x80)
	apu.WriteRegister(addr.NR30, 0x80) // Enable CH3 DAC
	apu.WriteRegister(addr.NR31, 0xFF) // Length timer = 255
	apu.WriteRegister(addr.NR34, 0xC0) // Trigger with length enable

	assert.Equal(t, uint16(1), apu.ch[2].length) // 256 - 255 = 1
	assert.True(t, apu.ch[2].enabled)

	apu.tickLength()
	assert.False(t, apu.ch[2].enabled, "CH3 should be disabled after length expires")
}

// Reproduces the behavior from dmg_sound test 3: "Enabling in first half of length period should clock length"
func TestAPU_LengthEnableClocking(t *testing.T) {
	tests := []struct {
		name          string
		channelIndex  int
		lengthAddr    uint16
		controlAddr   uint16
		lengthWrite   uint8
		initialLen    uint16
		sequencerStep int
		wantLen       uint16
	}{
		{
			name:          "ch1_first_half_should_clock",
			channelIndex:  0,
			lengthAddr:    addr.NR11,
			controlAddr:   addr.NR14,
			lengthWrite:   64 - 2, // length = 2
			initialLen:    2,
			sequencerStep: 1, // step that doesn't clock length
			wantLen:       1, // should be clocked when enabling
		},
		{
			name:          "ch1_second_half_should_not_clock",
			channelIndex:  0,
			lengthAddr:    addr.NR11,
			controlAddr:   addr.NR14,
			lengthWrite:   64 - 2, // length = 2
			initialLen:    2,
			sequencerStep: 0, // step that clocks length
			wantLen:       2, // should not be clocked when enabling
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			a := New()
			a.WriteRegister(addr.NR52, 0x80) // Enable APU

			// Set up sequencer step
			a.step = tc.sequencerStep
			a.cycles = 0

			// Initialize channel with length but length disabled
			a.WriteRegister(tc.controlAddr, 0x00)
			a.WriteRegister(tc.lengthAddr, tc.lengthWrite)

			// Verify initial length
			if got := a.ch[tc.channelIndex].length; got != tc.initialLen {
				t.Fatalf("length setup mismatch: got %d want %d", got, tc.initialLen)
			}

			// Enable length counter (write 0x40 to set bit 6)
			a.WriteRegister(tc.controlAddr, 0x40)

			// Check if length was clocked
			if got := a.ch[tc.channelIndex].length; got != tc.wantLen {
				t.Errorf("length after enable = %d, want %d", got, tc.wantLen)
			}
		})
	}
}

func TestAPU_TriggerUnfreezesEnabledLength(t *testing.T) {
	channelCases := []struct {
		name        string
		index       int
		lengthAddr  uint16
		controlAddr uint16
		lengthWrite uint8
		maxLen      uint16
		init        func(a *APU)
	}{
		{
			name:        "ch1",
			index:       0,
			lengthAddr:  addr.NR11,
			controlAddr: addr.NR14,
			lengthWrite: 0x3F, // length = 1
			maxLen:      64,
			init: func(a *APU) {
				a.WriteRegister(addr.NR12, 0xF0) // enable DAC
			},
		},
		{
			name:        "ch2",
			index:       1,
			lengthAddr:  addr.NR21,
			controlAddr: addr.NR24,
			lengthWrite: 0x3F, // length = 1
			maxLen:      64,
			init: func(a *APU) {
				a.WriteRegister(addr.NR22, 0xF0)
			},
		},
		{
			name:        "ch3",
			index:       2,
			lengthAddr:  addr.NR31,
			controlAddr: addr.NR34,
			lengthWrite: 0xFF, // length = 1
			maxLen:      256,
			init: func(a *APU) {
				a.WriteRegister(addr.NR30, 0x80)
			},
		},
		{
			name:        "ch4",
			index:       3,
			lengthAddr:  addr.NR41,
			controlAddr: addr.NR44,
			lengthWrite: 0x3F, // length = 1
			maxLen:      64,
			init: func(a *APU) {
				a.WriteRegister(addr.NR42, 0xF0)
			},
		},
	}

	variants := []struct {
		name        string
		disableStep bool
	}{
		{name: "with_disable", disableStep: true},
		{name: "without_disable", disableStep: false},
	}

	for _, cc := range channelCases {
		for _, variant := range variants {
			cc := cc
			variant := variant
			t.Run(cc.name+"_"+variant.name, func(t *testing.T) {
				a := New()
				a.WriteRegister(addr.NR52, 0x80) // power on APU
				cc.init(a)

				// ensure known state
				a.WriteRegister(cc.controlAddr, 0x00)
				a.WriteRegister(cc.lengthAddr, cc.lengthWrite)
				if got := a.ch[cc.index].length; got != 1 {
					t.Fatalf("length setup mismatch: got %d want 1", got)
				}

				a.step = 1 // first half of length period
				a.cycles = 0

				a.WriteRegister(cc.controlAddr, 0x40)
				if got := a.ch[cc.index].length; got != 0 {
					t.Fatalf("enable clock mismatch: got %d want 0", got)
				}
				if a.ch[cc.index].enabled {
					t.Fatalf("channel %s should be disabled when length hits zero", cc.name)
				}

				if variant.disableStep {
					a.WriteRegister(cc.controlAddr, 0x00)
				}

				a.WriteRegister(cc.controlAddr, 0xC0)
				wantLen := cc.maxLen - 1
				if got := a.ch[cc.index].length; got != wantLen {
					t.Fatalf("trigger clock mismatch: got %d want %d", got, wantLen)
				}
				if !a.ch[cc.index].enabled {
					t.Fatalf("channel %s should be enabled after trigger", cc.name)
				}
			})
		}
	}
}
