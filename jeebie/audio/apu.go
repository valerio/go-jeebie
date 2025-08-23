package audio

import (
	"sync"

	"github.com/valerio/go-jeebie/jeebie/addr"
	"github.com/valerio/go-jeebie/jeebie/bit"
)

// APU implements the Game Boy's Audio Processing Unit
// Reference: https://gbdev.io/pandocs/Audio.html
type APU struct {
	// mu protects APU state during concurrent write operations
	// Reads don't need protection for simple types (bool, uint8, uint16)
	// but complex operations (like power-off clearing registers) do
	mu sync.Mutex

	enabled   bool       // Master audio enable (NR52 bit 7)
	registers [0x30]byte // Audio registers FF10-FF3F (48 bytes)

	// Frame sequencer state
	// Runs at 512 Hz, advances every frameSequencerCycles (8192) CPU cycles
	frameCounter int // Current step (0-7) in frame sequence
	frameCycles  int // CPU cycles since last frame sequencer tick

	// Sample generation state
	sampleCycleCounter int        // CPU cycles since last sample
	sampleBuffer       []int16    // Generated samples awaiting consumption
	sampleBufferMu     sync.Mutex // Protects sample buffer

	ch1Enabled bool
	ch1Freq    uint16
	ch1Volume  uint8
	ch1Duty    uint8
	ch1Counter uint16

	ch2Enabled bool
	ch2Freq    uint16
	ch2Volume  uint8
	ch2Duty    uint8
	ch2Counter uint16

	ch3Enabled bool
	ch3Freq    uint16
	ch3Volume  uint8
	ch3Counter uint16
	ch3WaveRAM [waveRAMSize]uint8

	ch4Enabled bool
	ch4Volume  uint8
	ch4LFSR    uint16 // Linear feedback shift register for noise
	ch4Counter uint16

	// Debug channel muting
	ch1Muted bool
	ch2Muted bool
	ch3Muted bool
	ch4Muted bool
}

// New creates a new APU instance with initial register values
func New() *APU {
	apu := &APU{
		enabled:      true,
		sampleBuffer: make([]int16, 0, initialBufferCapacity),
		ch4LFSR:      lfsrInitialValue,
	}
	apu.initRegisters()
	return apu
}

// initRegisters sets the initial power-on values for audio registers
// These values are from the official Game Boy documentation
// Reference: https://gbdev.io/pandocs/Power_Up_Sequence.html#hardware-registers
func (a *APU) initRegisters() {
	// Channel 1 registers
	a.registers[0x10] = 0x80 // NR10: Sweep off
	a.registers[0x11] = 0xBF // NR11: Duty 50%, length counter loaded with max
	a.registers[0x12] = 0xF3 // NR12: Max volume, decrease, period 3
	a.registers[0x14] = 0xBF // NR14: Counter mode, frequency MSB

	// Channel 2 registers
	a.registers[0x16] = 0x3F // NR21: Duty 0%, length counter max
	a.registers[0x17] = 0x00 // NR22: Muted
	a.registers[0x19] = 0xBF // NR24: Counter mode, frequency MSB

	// Channel 3 registers
	a.registers[0x1A] = 0x7F // NR30: DAC off
	a.registers[0x1B] = 0xFF // NR31: Length counter max
	a.registers[0x1C] = 0x9F // NR32: Volume 0
	a.registers[0x1E] = 0xBF // NR34: Counter mode

	// Channel 4 registers
	a.registers[0x20] = 0xFF // NR41: Length counter max
	a.registers[0x21] = 0x00 // NR42: Muted
	a.registers[0x22] = 0x00 // NR43: Clock divider 0
	a.registers[0x23] = 0xBF // NR44: Counter mode

	// Global control registers
	a.registers[0x24] = 0x77 // NR50: Max volume both channels
	a.registers[0x25] = 0xF3 // NR51: All channels to both outputs
	a.registers[0x26] = 0xF1 // NR52: All sound on, all channels on (on GB)
}

func (a *APU) Step(cycles int) {
	if !a.enabled {
		return
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	a.frameCycles += cycles
	if a.frameCycles >= frameSequencerCycles {
		a.frameCycles -= frameSequencerCycles
		a.updateFrameSequencer()
	}

	a.sampleCycleCounter += cycles
	for a.sampleCycleCounter >= sampleCycles {
		a.sampleCycleCounter -= sampleCycles
		a.generateSample()
	}
}

// updateFrameSequencer advances the frame sequencer which controls
// sweep, length counter, and envelope timing
// The frame sequencer has 8 steps (0-7) and runs at 512 Hz
// Frame sequencer step actions:
//
//	Step   Length  Sweep  Envelope
//	0      Clock   -      -
//	1      -       -      -
//	2      Clock   Clock  -
//	3      -       -      -
//	4      Clock   -      -
//	5      -       -      -
//	6      Clock   Clock  -
//	7      -       -      Clock
//
// Reference: https://gbdev.io/pandocs/Audio_details.html#frame-sequencer
func (a *APU) updateFrameSequencer() {
	a.frameCounter = (a.frameCounter + 1) & 7
	switch a.frameCounter {
	case 0, 4:
		a.updateLengthCounters() // 256 Hz (every 2 steps)
	case 2, 6:
		a.updateLengthCounters() // 256 Hz
		a.updateSweep()          // 128 Hz (every 4 steps)
	case 7:
		a.updateEnvelopes() // 64 Hz (every 8 steps)
	}
}

func (a *APU) updateLengthCounters() {
}

func (a *APU) updateSweep() {
}

func (a *APU) updateEnvelopes() {
}

func (a *APU) generateSample() {
	sample := a.mixChannels()

	a.sampleBufferMu.Lock()
	a.sampleBuffer = append(a.sampleBuffer, sample, sample)
	if len(a.sampleBuffer) > maxBufferSize {
		a.sampleBuffer = a.sampleBuffer[len(a.sampleBuffer)-bufferRetainSize:]
	}
	a.sampleBufferMu.Unlock()
}

func (a *APU) mixChannels() int16 {
	if !a.enabled {
		return 0
	}

	var mixed int32

	if a.ch1Enabled && !a.ch1Muted {
		mixed += int32(a.generateChannel1())
	}
	if a.ch2Enabled && !a.ch2Muted {
		mixed += int32(a.generateChannel2())
	}
	if a.ch3Enabled && !a.ch3Muted {
		mixed += int32(a.generateChannel3())
	}
	if a.ch4Enabled && !a.ch4Muted {
		mixed += int32(a.generateChannel4())
	}

	if mixed > maxSampleValue {
		mixed = maxSampleValue
	} else if mixed < minSampleValue {
		mixed = minSampleValue
	}

	return int16(mixed)
}

// generatePulseChannel generates a sample for a pulse channel (used by channels 1 and 2)
func (a *APU) generatePulseChannel(counter *uint16, volume uint8, freq uint16, duty uint8) int16 {
	if volume == 0 || freq == 0 {
		return 0
	}

	period := uint16(frequencyToTimerOffset - freq)
	updateCounterWithPeriod(counter, period)

	pattern := dutyPatterns[duty&3]
	phase := (*counter * dutyPhases) / period
	dutyBit := (pattern >> (7 - phase)) & 1

	if dutyBit == 1 {
		return int16(volume) * sampleAmplitude
	}
	return -int16(volume) * sampleAmplitude
}

func (a *APU) generateChannel1() int16 {
	return a.generatePulseChannel(&a.ch1Counter, a.ch1Volume, a.ch1Freq, a.ch1Duty)
}

func (a *APU) generateChannel2() int16 {
	return a.generatePulseChannel(&a.ch2Counter, a.ch2Volume, a.ch2Freq, a.ch2Duty)
}

// updateCounterWithPeriod updates a counter and returns true when it wraps
func updateCounterWithPeriod(counter *uint16, period uint16) bool {
	*counter++
	if *counter >= period {
		*counter = 0
		return true
	}
	return false
}

func (a *APU) generateChannel3() int16 {
	if !a.ch3Enabled || a.ch3Freq == 0 {
		return 0
	}

	period := uint16(frequencyToTimerOffset - a.ch3Freq)
	updateCounterWithPeriod(&a.ch3Counter, period)

	sampleIndex := (a.ch3Counter * waveTableSize) / period
	nibbleIndex := sampleIndex / 2
	highNibble := sampleIndex&1 == 0

	sample := a.ch3WaveRAM[nibbleIndex]
	if highNibble {
		sample = (sample >> 4) & 0x0F
	} else {
		sample = sample & 0x0F
	}

	volumeShift := waveVolumeShift[a.ch3Volume&3]
	sample = sample >> volumeShift

	return int16(sample)*waveOutputScale - waveOutputBias
}

func (a *APU) generateChannel4() int16 {
	if a.ch4Volume == 0 {
		return 0
	}

	if updateCounterWithPeriod(&a.ch4Counter, noiseChannelPeriod) {
		// LFSR feedback calculation
		feedbackBit := (a.ch4LFSR & 1) ^ ((a.ch4LFSR >> 1) & 1)
		a.ch4LFSR = (a.ch4LFSR >> 1) | (feedbackBit << 14)

		// 7-bit mode (width = 1)
		if bit.IsSet(3, a.registers[0x22]) { // Bit 3: Width mode (7-bit LFSR)
			a.ch4LFSR = (a.ch4LFSR & 0xFF7F) | (feedbackBit << 6) // Also set bit 6
		}
	}

	if (a.ch4LFSR & 1) == 0 {
		return int16(a.ch4Volume) * sampleAmplitude
	}
	return -int16(a.ch4Volume) * sampleAmplitude
}

// ReadRegister reads from an audio register
// Most reads don't need mutex protection as they read simple types
func (a *APU) ReadRegister(address uint16) uint8 {

	if address < addr.AudioStart || address > addr.AudioEnd {
		return 0xFF
	}

	index := address - addr.AudioStart

	switch address {
	case addr.NR52: // NR52
		// NR52 returns power status and channel status bits
		status := a.registers[index] & nr52PowerMask
		if a.ch1Enabled {
			status |= nr52Ch1StatusMask
		}
		if a.ch2Enabled {
			status |= nr52Ch2StatusMask
		}
		if a.ch3Enabled {
			status |= nr52Ch3StatusMask
		}
		if a.ch4Enabled {
			status |= nr52Ch4StatusMask
		}
		return status | nr52UnusedMask // Bits 4-6 always read as 1
	case addr.WaveRAMStart, addr.WaveRAMStart + 1, addr.WaveRAMStart + 2, addr.WaveRAMStart + 3,
		addr.WaveRAMStart + 4, addr.WaveRAMStart + 5, addr.WaveRAMStart + 6, addr.WaveRAMStart + 7,
		addr.WaveRAMStart + 8, addr.WaveRAMStart + 9, addr.WaveRAMStart + 10, addr.WaveRAMStart + 11,
		addr.WaveRAMStart + 12, addr.WaveRAMStart + 13, addr.WaveRAMStart + 14, addr.WaveRAMStart + 15:
		waveIndex := address - addr.WaveRAMStart
		return a.registers[waveRAMRegisterOffset+waveIndex]
	default:
		return a.registers[index]
	}
}

// updateFrequencyLow updates the low 8 bits of a frequency value
func updateFrequencyLow(current uint16, lowByte uint8) uint16 {
	return (current & 0x700) | uint16(lowByte)
}

// updateFrequencyHigh updates the high 3 bits of a frequency value
func updateFrequencyHigh(current uint16, highBits uint8) uint16 {
	return (current & 0xFF) | (uint16(highBits&0x07) << 8)
}

// WriteRegister writes to an audio register
// Needs mutex protection as it modifies shared state
func (a *APU) WriteRegister(address uint16, value uint8) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if address < addr.AudioStart || address > addr.AudioEnd {
		return
	}

	index := address - addr.AudioStart

	// Special case: NR52 power control clears registers when disabled
	if address == addr.NR52 {
		wasEnabled := a.enabled
		a.enabled = bit.IsSet(7, value)
		if !a.enabled && wasEnabled {
			// Clear all registers except NR52
			for i := range a.registers {
				if i != 0x16 { // Keep NR52 itself
					a.registers[i] = 0
				}
			}
			a.ch1Enabled = false
			a.ch2Enabled = false
			a.ch3Enabled = false
			a.ch4Enabled = false
		}
	}

	a.registers[index] = value
	a.mapRegisterToState(address, value)
}

// mapRegisterToState updates internal channel state based on register writes
func (a *APU) mapRegisterToState(address uint16, value uint8) {
	switch address {
	case addr.NR11: // Channel 1 duty cycle and length
		a.ch1Duty = value >> 6 // Bits 7-6: Duty cycle
	case addr.NR12: // Channel 1 volume envelope
		a.ch1Volume = value >> 4           // Bits 7-4: Initial volume
		a.ch1Enabled = (value & 0xF8) != 0 // DAC enabled if bits 3-7 are not all zero
	case addr.NR13: // Channel 1 frequency low
		a.ch1Freq = updateFrequencyLow(a.ch1Freq, value)
	case addr.NR14: // Channel 1 frequency high and control
		a.ch1Freq = updateFrequencyHigh(a.ch1Freq, value)
		if bit.IsSet(7, value) { // Bit 7: Trigger
			a.ch1Counter = 0
			a.ch1Enabled = true
		}
	case addr.NR21: // Channel 2 duty cycle and length
		a.ch2Duty = value >> 6 // Bits 7-6: Duty cycle
	case addr.NR22: // Channel 2 volume envelope
		a.ch2Volume = value >> 4           // Bits 7-4: Initial volume
		a.ch2Enabled = (value & 0xF8) != 0 // DAC enabled if bits 3-7 are not all zero
	case addr.NR23: // Channel 2 frequency low
		a.ch2Freq = updateFrequencyLow(a.ch2Freq, value)
	case addr.NR24: // Channel 2 frequency high and control
		a.ch2Freq = updateFrequencyHigh(a.ch2Freq, value)
		if bit.IsSet(7, value) { // Bit 7: Trigger
			a.ch2Counter = 0
			a.ch2Enabled = true
		}
	case addr.NR30: // Channel 3 DAC enable
		a.ch3Enabled = bit.IsSet(7, value) // Bit 7: DAC enable
	case addr.NR32: // Channel 3 output level
		a.ch3Volume = (value >> 5) & 0x03 // Bits 6-5: Output level
	case addr.NR33: // Channel 3 frequency low
		a.ch3Freq = updateFrequencyLow(a.ch3Freq, value)
	case addr.NR34: // Channel 3 frequency high and control
		a.ch3Freq = updateFrequencyHigh(a.ch3Freq, value)
		if bit.IsSet(7, value) { // Bit 7: Trigger
			a.ch3Counter = 0
		}
	case addr.NR42: // Channel 4 volume envelope
		a.ch4Volume = value >> 4           // Bits 7-4: Initial volume
		a.ch4Enabled = (value & 0xF8) != 0 // DAC enabled if bits 3-7 are not all zero
	case addr.NR44: // Channel 4 control
		if bit.IsSet(7, value) { // Bit 7: Trigger
			a.ch4LFSR = lfsrInitialValue
			a.ch4Counter = 0
			a.ch4Enabled = true
		}
	}

	// Handle Wave RAM writes separately
	if address >= addr.WaveRAMStart && address <= addr.WaveRAMEnd {
		waveIndex := address - addr.WaveRAMStart
		nibbleIndex := waveIndex / 2
		if (waveIndex & 1) == 0 {
			// Even address: store high nibble in high 4 bits
			a.ch3WaveRAM[nibbleIndex] = (a.ch3WaveRAM[nibbleIndex] & 0x0F) | (value & 0xF0)
		} else {
			// Odd address: store in low 4 bits
			a.ch3WaveRAM[nibbleIndex] = (a.ch3WaveRAM[nibbleIndex] & 0xF0) | (value & 0x0F)
		}
	}
}

func (a *APU) GetSamples(count int) []int16 {
	a.sampleBufferMu.Lock()
	defer a.sampleBufferMu.Unlock()

	if len(a.sampleBuffer) < count {
		samples := make([]int16, count)
		copy(samples, a.sampleBuffer)
		a.sampleBuffer = a.sampleBuffer[:0]
		return samples
	}

	samples := a.sampleBuffer[:count]
	a.sampleBuffer = a.sampleBuffer[count:]
	return samples
}

func (a *APU) Reset() {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.enabled = true
	a.frameCounter = 0
	a.frameCycles = 0
	a.sampleCycleCounter = 0
	a.sampleBuffer = a.sampleBuffer[:0]

	a.ch1Enabled = false
	a.ch1Freq = 0
	a.ch1Volume = 0
	a.ch1Duty = 0
	a.ch1Counter = 0

	a.ch2Enabled = false
	a.ch2Freq = 0
	a.ch2Volume = 0
	a.ch2Duty = 0
	a.ch2Counter = 0

	a.ch3Enabled = false
	a.ch3Freq = 0
	a.ch3Volume = 0
	a.ch3Counter = 0

	a.ch4Enabled = false
	a.ch4Volume = 0
	a.ch4LFSR = lfsrInitialValue
	a.ch4Counter = 0

	a.initRegisters()
}

// MuteChannel mutes or unmutes a specific audio channel for debugging
func (a *APU) MuteChannel(channel int, muted bool) {
	a.mu.Lock()
	defer a.mu.Unlock()

	switch channel {
	case 1:
		a.ch1Muted = muted
	case 2:
		a.ch2Muted = muted
	case 3:
		a.ch3Muted = muted
	case 4:
		a.ch4Muted = muted
	}
}

// ToggleChannel toggles muting for a specific channel
func (a *APU) ToggleChannel(channel int) {
	a.mu.Lock()
	defer a.mu.Unlock()

	switch channel {
	case 1:
		a.ch1Muted = !a.ch1Muted
	case 2:
		a.ch2Muted = !a.ch2Muted
	case 3:
		a.ch3Muted = !a.ch3Muted
	case 4:
		a.ch4Muted = !a.ch4Muted
	}
}

// SoloChannel mutes all channels except the specified one
func (a *APU) SoloChannel(channel int) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.ch1Muted = (channel != 1)
	a.ch2Muted = (channel != 2)
	a.ch3Muted = (channel != 3)
	a.ch4Muted = (channel != 4)
}

// UnmuteAll unmutes all channels
func (a *APU) UnmuteAll() {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.ch1Muted = false
	a.ch2Muted = false
	a.ch3Muted = false
	a.ch4Muted = false
}

// GetChannelStatus returns the current mute status and basic info for all channels
func (a *APU) GetChannelStatus() (ch1, ch2, ch3, ch4 bool) {
	return !a.ch1Muted && a.ch1Enabled,
		!a.ch2Muted && a.ch2Enabled,
		!a.ch3Muted && a.ch3Enabled,
		!a.ch4Muted && a.ch4Enabled
}
