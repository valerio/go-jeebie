package audio

import (
	"github.com/valerio/go-jeebie/jeebie/addr"
	"github.com/valerio/go-jeebie/jeebie/bit"
)

// APU is the Audio Processing Unit of a DMG Game Boy. It generates 4-channel audio:
// CH1 (square+sweep), CH2 (square), CH3 (wave), CH4 (noise), all mixed to stereo output.
// This is basically a bunch of counters and timers that tick at certain frequency steps!
//
// TODO: Implement frame sequencer that runs at 512 Hz with this step table:
//
//	Step 0,2,4,6: compute length counters (256 Hz)
//	Step 2,6: compute sweep for CH1 (128 Hz)
//	Step 7: compute envelopes for CH1,CH2,CH4 (64 Hz)
//
// TODO: Implement sample generation at ~44.1 kHz for audio output.
//
// TODO: Implement trigger behavior (NRx4 bit 7) to restart channel state.
//
// TODO: Implement power-off logic (NR52 bit 7=0) to disable channels and ignore register writes.
type APU struct {

	// state, this is information derived from registers/memory.
	enabled           bool
	ch                [4]Channel
	vinLeft, vinRight bool  // from NR50, not used in DMG
	volLeft, volRight uint8 // volume for left/right, values 0 to 7

	// raw memory + registers
	NR10, NR11, NR12, NR13, NR14 uint8 // Channel 1
	NR21, NR22, NR23, NR24       uint8 // Channel 2
	NR30, NR31, NR32, NR33, NR34 uint8 // Channel 3
	NR41, NR42, NR43, NR44       uint8 // Channel 4
	NR50, NR51, NR52             uint8 // Global controls
	waveRAM                      [16]uint8
}

// Channel represents one of the four APU channels.
// Fields might be used depending on channel type.
//
// Some simple explanations of what concepts mean:
//   - duty: for square waves (ch1-2), which pattern/shape to use (0-3)
//   - sweep: changes frequency over time (only for ch1)
//   - envelope: changes volume over time (for ch1-2, ch4)
//   - period: how often to make a cycle, frequency = 2048 - period (for ch1-3)
//   - DAC: Digital-to-Analog Converter, if off the channel is silent (for ch1-3)
//   - LFSR: Linear Feedback Shift Register, a pseudo-random bit generator (for ch4)
type Channel struct {
	enabled bool

	// panning, or "on which side is this channel heard?"
	// can be both or neither, if neither it's effectively muted (we don't mix it)
	left, right bool

	duty   uint8 // for square waves, values 0 to 3
	timer  uint8 // initial length timer value, 6 bits for ch1-2-4 -> values 0 to 64, 8 bits for ch3 -> values 0 to 256
	length uint8 // current length counter, this is set to timer on a trigger, then it counts down to 0
	volume uint8 // initial volume, 4 bits -> values 0 to 15

	// Frequency sweep (CH1 only)
	sweepPeriod uint8 // "pace" per pandocs (NR10 6-4), 3 bits -> values 0 to 7
	sweepDown   bool  // sweep direction, 0=up, 1=down
	sweepStep   uint8 // sweep step, 3 bits -> values 0 to 7

	envelopePace uint8 // NRx2 bits 7-4, 3 bits -> values 0 to 7
	envelopeUp   bool  // NRx2 bit 3, 0=down, 1=up

	period       uint16 // frequency period, 11 bits -> values 0 to 2047
	trigger      bool   // trigger flag, write-only, when written it "triggers" the channel
	lengthEnable bool   // length enable flag

	// CH4 Noise channel specific
	lfsr        uint16 // 15-bit LFSR for noise generation
	use7bitLFSR bool   // from NR43 bit 3, when set use 7-bit LFSR, otherwise 15-bit
	shift       uint8  // from NR43, 4 bits -> values 0 to 15
	divider     uint8  // from NR43, 3 bits -> values 0 to 7

	dacEnabled bool // for channel 3, DAC enable flag
}

func New() *APU { return &APU{} }

// Tick advances the APU by CPU cycles.
func (a *APU) Tick(cycles int) {
	if !a.enabled {
		return
	}

	// TODO: Implement APU timing with these components:
	//   - Frame sequencer at 512 Hz (every 8192 CPU cycles)
	//   - Sample generation at ~44.1 kHz (every ~95 CPU cycles)
	//   - Call clockLength(), clockSweep(), clockEnvelope() per frame step table above
	//   - Generate and mix samples from all 4 channels
}

// ReadRegister returns masked register values.
// Note: write-only and unused bits are fixed to 1 when reading.
func (a *APU) ReadRegister(address uint16) uint8 {
	switch address {
	case addr.NR10:
		return a.NR10 | 0b1000_0000
	case addr.NR11:
		return a.NR11 | 0b0011_1111
	case addr.NR12:
		return a.NR12
	case addr.NR13:
		return 0xFF // write-only reg
	case addr.NR14:
		return a.NR14 | 0b1011_1000
	case addr.NR21:
		return a.NR21 | 0b0011_1111
	case addr.NR22:
		return a.NR22
	case addr.NR23:
		return 0xFF // write-only reg
	case addr.NR24:
		return a.NR24 | 0b1011_1000
	case addr.NR30:
		return a.NR30 | 0b0111_1111
	case addr.NR31:
		return 0xFF // write-only reg
	case addr.NR32:
		return a.NR32 | 0b1001_1111
	case addr.NR33:
		return 0xFF // write-only reg
	case addr.NR34:
		return a.NR34 | 0b1011_1000
	case addr.NR41:
		return 0xFF // write-only reg
	case addr.NR42:
		return a.NR42
	case addr.NR43:
		return a.NR43
	case addr.NR44:
		return a.NR44 | 0b1011_1111
	case addr.NR50:
		return a.NR50 | 0b1000_0000
	case addr.NR51:
		return a.NR51
	case addr.NR52:
		// NR52 status: bit 7 = power, bits 6-4 always 1, bits 3-0 = channel active status
		status := uint8(0b0111_0000)
		if a.enabled {
			status = bit.Set(7, status)
		}
		// set the low 4 bits based on channel enabled flags
		for i := range 4 {
			if a.ch[i].enabled {
				status = bit.Set(uint8(i), status)
			}
		}
		return status
	}
	if address >= addr.WaveRAMStart && address <= addr.WaveRAMEnd {
		// TODO: wave RAM read behavior changes when channel 3 is playing
		// if ch3_enabled and ch3_dac_on:
		//   return wave_ram[current_wave_byte_index]
		// else:
		return a.waveRAM[address-addr.WaveRAMStart]
	}
	// unmapped - panic?
	return 0xFF
}

// WriteRegister stores the value of the given register/memory, then updates
// internal state accordingly.
func (a *APU) WriteRegister(address uint16, value uint8) {

	// TODO: power off logic
	// if setting NR52 bit7 to 0, disable all channels, clear most registers to known values
	// and ignore writes to audio regs except NR52/RAM

	switch address {
	case addr.NR10:
		a.NR10 = value
	case addr.NR11:
		a.NR11 = value
	case addr.NR12:
		a.NR12 = value
	case addr.NR13:
		a.NR13 = value
	case addr.NR14:
		a.NR14 = value
	case addr.NR21:
		a.NR21 = value
	case addr.NR22:
		a.NR22 = value
	case addr.NR23:
		a.NR23 = value
	case addr.NR24:
		a.NR24 = value
	case addr.NR30:
		a.NR30 = value
	case addr.NR31:
		a.NR31 = value
	case addr.NR32:
		a.NR32 = value
	case addr.NR33:
		a.NR33 = value
	case addr.NR34:
		a.NR34 = value
	case addr.NR41:
		a.NR41 = value
	case addr.NR42:
		a.NR42 = value
	case addr.NR43:
		a.NR43 = value
	case addr.NR44:
		a.NR44 = value
	case addr.NR50:
		a.NR50 = value
	case addr.NR51:
		a.NR51 = value
	case addr.NR52:
		a.NR52 = value
	default:
		// ignore
	}

	if address >= addr.WaveRAMStart && address <= addr.WaveRAMEnd {
		a.waveRAM[address-addr.WaveRAMStart] = value
	}

	a.mapRegistersToState()
}

func (a *APU) mapRegistersToState() {
	// NR52 - Master Audio Control
	// 7: Audio on/off | 6-4: Always 1 | 3: CH4 on | 2: CH3 on | 1: CH2 on | 0: CH1 on
	a.enabled = bit.IsSet(7, a.NR52) // audio on/off
	for i := range 4 {
		// TODO: pandocs say these should be read-only.
		// might just replace this with setting enabled directly for debug/testing
		a.ch[i].enabled = bit.IsSet(uint8(i), a.NR52)
	}

	// NR51 - Sound Panning
	// 7: CH4L | 6: CH3L | 5: CH2L | 4: CH1L | 3: CH4R | 2: CH3R | 1: CH2R | 0: CH1R
	for i := range 4 {
		a.ch[i].right = bit.IsSet(uint8(i), a.NR51)
		a.ch[i].left = bit.IsSet(uint8(i+4), a.NR51)
	}

	// NR50 - Master Volume & VIN Panning
	// 7: VIN L | 6-4: Vol L | 3: VIN R | 2-0: Vol R
	a.vinLeft, a.vinRight = bit.IsSet(7, a.NR50), bit.IsSet(3, a.NR50)
	a.volLeft, a.volRight = bit.ExtractBits(a.NR50, 6, 4), bit.ExtractBits(a.NR50, 2, 0)

	// Channel 1 (Square + Sweep) - NR10-NR14

	// NR10 - Channel 1 Sweep Control
	// 7: - | 6-4: Period | 3: Direction | 2-0: Shift
	a.ch[0].sweepPeriod = bit.ExtractBits(a.NR10, 6, 4)
	a.ch[0].sweepDown = bit.IsSet(3, a.NR10)
	a.ch[0].sweepStep = bit.ExtractBits(a.NR10, 2, 0)

	// NR11 - Channel 1 Length Timer & Duty Cycle
	// 7-6: Duty | 5-0: Length Timer (0-63, actual = 64-value)
	a.ch[0].duty = bit.ExtractBits(a.NR11, 7, 6)
	a.ch[0].timer = bit.ExtractBits(a.NR11, 5, 0)

	// NR12 - Channel 1 Volume & Envelope
	// 7-4: Initial Volume | 3: Direction | 2-0: Period
	a.ch[0].volume = bit.ExtractBits(a.NR12, 7, 4)
	a.ch[0].envelopeUp = bit.IsSet(3, a.NR12)
	a.ch[0].envelopePace = bit.ExtractBits(a.NR12, 2, 0)

	// frequency = 131072/(2048-value) Hz
	// NR13 - 7-0: low bits of period for Channel 1
	// NR14 - 2-0: upper 3 bits of period for Channel 1
	a.ch[0].period = bit.Combine(a.NR14&0b111, a.NR13)

	// NR14 - Channel 1 Frequency High & Control
	// 7: Trigger | 6: Length Enable | 5-3: - | 2-0: Upper 3 bits of freq
	a.ch[0].lengthEnable = bit.IsSet(6, a.NR14)
	// TODO: when this is set, handle trigger behavior
	a.ch[0].trigger = bit.IsSet(7, a.NR14)

	// DAC for a channel is enabled if initial volume > 0 or envelope is increasing
	a.ch[0].dacEnabled = (a.ch[0].volume > 0) || a.ch[0].envelopeUp

	// Channel 2 (Square) - NR21-NR24

	// NR21 - Channel 2 Length Timer & Duty Cycle
	// 7-6: Duty | 5-0: Length Timer (0-63, actual = 64-value)
	a.ch[1].duty = bit.ExtractBits(a.NR21, 7, 6)
	a.ch[1].timer = bit.ExtractBits(a.NR21, 5, 0)

	// NR22 - Channel 2 Volume & Envelope
	// 7-4: Initial Volume | 3: Direction | 2-0: Period
	a.ch[1].volume = bit.ExtractBits(a.NR22, 7, 4)
	a.ch[1].envelopeUp = bit.IsSet(3, a.NR22)
	a.ch[1].envelopePace = bit.ExtractBits(a.NR22, 2, 0)

	// frequency = 131072/(2048-value) Hz
	// NR23 - 7-0: low bits of period for Channel 2
	// NR24 - 2-0: upper 3 bits of period for Channel 2
	a.ch[1].period = bit.Combine(a.NR24&0b111, a.NR23)

	// NR24 - Channel 2 Frequency High & Control
	// 7: Trigger | 6: Length Enable | 5-3: - | 2-0: Upper 3 bits of freq
	a.ch[1].lengthEnable = bit.IsSet(6, a.NR24)
	a.ch[1].trigger = bit.IsSet(7, a.NR24) // TODO: handle trigger behavior

	// DAC for a channel is enabled if initial volume > 0 or envelope is increasing
	a.ch[1].dacEnabled = (a.ch[1].volume > 0) || a.ch[1].envelopeUp

	// Channel 3 (Wave) - NR30-NR34

	// NR30 - Channel 3 DAC Enable - this is set differently from other channels
	// 7: DAC Enable | 6-0: - (read as 1)
	a.ch[2].dacEnabled = bit.IsSet(7, a.NR30)

	// NR31 - Channel 3 Length Timer (write-only)
	// 7-0: Length Timer (0-255, actual = 256-value)
	a.ch[2].timer = a.NR31

	// NR32 - Channel 3 Output Level
	// 7: - | 6-5: Output Level | 4-0: -
	// Level: 00=mute, 01=100%, 10=50%, 11=25%
	a.ch[2].volume = bit.ExtractBits(a.NR32, 6, 5)

	// frequency = 65536/(2048-value) Hz (twice as fast as square channels)
	// NR33 - 7-0: Lower 8 bits of frequency
	// NR34 - 2-0: Upper 3 bits of frequency
	a.ch[2].period = bit.Combine(a.NR34&0b111, a.NR33)

	// NR34 - Channel 3 Frequency High & Control
	// 7: Trigger | 6: Length Enable | 5-3: - | 2-0: Upper 3 bits of freq
	a.ch[2].lengthEnable = bit.IsSet(6, a.NR34)
	a.ch[2].trigger = bit.IsSet(7, a.NR34) // TODO: handle trigger behavior

	// Channel 4 (Noise) - NR41-NR44

	// NR41 - Channel 4 Length Timer (write-only)
	// 7-6: - | 5-0: Length Timer (0-63, actual = 64-value)
	a.ch[3].timer = bit.ExtractBits(a.NR41, 5, 0)

	// NR42 - Channel 4 Volume & Envelope
	// 7-4: Initial Volume | 3: Direction | 2-0: Period
	a.ch[3].volume = bit.ExtractBits(a.NR42, 7, 4)
	a.ch[3].envelopeUp = bit.IsSet(3, a.NR42)
	a.ch[3].envelopePace = bit.ExtractBits(a.NR42, 2, 0)

	// NR43 - Channel 4 Frequency & Randomness
	// 7-4: Clock Shift | 3: LFSR Width | 2-0: Clock Divider
	// frequency = 524288 / r / 2^(s+1) where r=divider, s=shift
	a.ch[3].shift = bit.ExtractBits(a.NR43, 7, 4)
	a.ch[3].use7bitLFSR = bit.IsSet(3, a.NR43)
	a.ch[3].divider = bit.ExtractBits(a.NR43, 2, 0)

	// NR44 - Channel 4 Control
	// 7: Trigger | 6: Length Enable | 5-0: -
	a.ch[3].lengthEnable = bit.IsSet(6, a.NR44)
	a.ch[3].trigger = bit.IsSet(7, a.NR44) // TODO: handle trigger behavior

	// DAC for a channel is enabled if initial volume > 0 or envelope is increasing
	a.ch[3].dacEnabled = (a.ch[3].volume > 0) || a.ch[3].envelopeUp
}

// GetSamples returns interleaved stereo samples.
func (a *APU) GetSamples(count int) []int16 { return nil }

// Debug helpers required by Provider.
func (a *APU) MuteChannel(channel int, muted bool)        {}
func (a *APU) ToggleChannel(channel int)                  {}
func (a *APU) SoloChannel(channel int)                    {}
func (a *APU) UnmuteAll()                                 {}
func (a *APU) GetChannelStatus() (bool, bool, bool, bool) { return false, false, false, false }

// GetChannelVolumes returns actual post-envelope volumes per channel.
func (a *APU) GetChannelVolumes() (ch1, ch2, ch3, ch4 uint8) { return 0, 0, 0, 0 }
