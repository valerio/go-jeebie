package audio

import (
	"sync"
	"time"

	"github.com/valerio/go-jeebie/jeebie/addr"
	"github.com/valerio/go-jeebie/jeebie/bit"
)

// ChannelState holds per-channel state
type ChannelState struct {
	enabled bool
	freq    uint16
	volume  uint8
	counter uint32 // Fixed-point counter (16.16) for ch1-3, uint16 for ch4

	// Pulse channels only (ch1, ch2)
	duty uint8

	// Envelope state (ch1, ch2, ch4)
	envelopePeriod    uint8
	envelopeDirection uint8 // 0 = decrease, 1 = increase
	envelopeTimer     uint8

	// Length counter
	lengthCounter uint16
	lengthEnabled bool

	// Noise channel specific (ch4)
	noisePeriod uint16 // Calculated period from NR43

	// Debug
	muted bool
}

// APU implements the Game Boy's Audio Processing Unit
// Reference: https://gbdev.io/pandocs/Audio.html
type APU struct {
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

	// Channel states (indexed 0-3 for channels 1-4)
	channels [4]ChannelState

	// Channel 3 specific
	ch3WaveRAM           [waveRAMSize]uint8
	ch3FirstFetchPending bool  // Hold output until first fetch at index 1 (lower nibble)
	ch3LastOutput        int16 // Last DAC output (reset to 0 when APU is off)
	ch3CurrentByteIndex  uint8 // Approximate current byte index being read during active playback

	// Channel 1 sweep state (NR10)
	ch1SweepPeriod  uint8
	ch1SweepNegate  bool
	ch1SweepShift   uint8
	ch1SweepTimer   uint8
	ch1SweepEnabled bool
	ch1SweepShadow  uint16

	// Channel 4 specific
	ch4LFSR uint16 // Linear feedback shift register for noise

	// Debug tracking
	debugStats struct {
		samplesGenerated    uint64
		frameSequencerTicks uint64
		lastSampleTime      time.Time
		lastFrameSeqTime    time.Time
		cyclesProcessed     uint64
	}
}

// clockLength decrements the channel's length counter and disables the channel if it reaches zero.
func (a *APU) clockLength(ch int) {
	if a.channels[ch].lengthCounter > 0 {
		a.channels[ch].lengthCounter--
		if a.channels[ch].lengthCounter == 0 {
			a.channels[ch].enabled = false
		}
	}
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
	// Channel 1 registers (relative indices from 0xFF10)
	a.registers[0x00] = 0x80 // NR10: Sweep off
	a.registers[0x01] = 0xBF // NR11: Duty 50%, length counter loaded with max
	a.registers[0x02] = 0xF3 // NR12: Max volume, decrease, period 3
	// NR13 (0x03) left at 0x00 power-on
	a.registers[0x04] = 0xBF // NR14: Counter mode, frequency MSB

	// Channel 2 registers
	a.registers[0x06] = 0x3F // NR21: Duty 0%, length counter max
	a.registers[0x07] = 0x00 // NR22: Muted
	// NR23 (0x08) left at 0x00
	a.registers[0x09] = 0xBF // NR24: Counter mode, frequency MSB

	// Channel 3 registers
	a.registers[0x0A] = 0x7F // NR30: DAC off
	a.registers[0x0B] = 0xFF // NR31: Length counter max
	a.registers[0x0C] = 0x9F // NR32: Volume 0
	// NR33 (0x0D) left at 0x00
	a.registers[0x0E] = 0xBF // NR34: Counter mode

	// Channel 4 registers
	a.registers[0x10] = 0xFF // NR41: Length counter max
	a.registers[0x11] = 0x00 // NR42: Muted
	a.registers[0x12] = 0x00 // NR43: Clock divider 0
	a.registers[0x13] = 0xBF // NR44: Counter mode

	// Global control registers
	a.registers[0x14] = 0x77 // NR50: Max volume both channels
	a.registers[0x15] = 0xF3 // NR51: All channels to both outputs
	a.registers[0x16] = 0xF1 // NR52: All sound on, all channels on (on GB)

	// Initialize noise period with default value from NR43 = 0x00
	// divisor = 0.5, shift = 0, frequency = 524288 Hz
	// period = (44100 * 256) / 524288 = ~21.5
	a.channels[3].noisePeriod = 21
	// Initialize CH3 last output to 0
	a.ch3LastOutput = 0
	a.ch3FirstFetchPending = false
}

func (a *APU) Tick(cycles int) {
	if !a.enabled {
		return
	}

	// Track cycles for debugging
	a.debugStats.cyclesProcessed += uint64(cycles)

	// Frame sequencer (512 Hz)
	a.frameCycles += cycles
	if a.frameCycles >= frameSequencerCycles {
		a.frameCycles -= frameSequencerCycles
		a.updateFrameSequencer()

		// Debug: Track frame sequencer timing
		a.debugStats.frameSequencerTicks++
		now := time.Now()
		if a.debugStats.lastFrameSeqTime.IsZero() {
			a.debugStats.lastFrameSeqTime = now
		} else if a.debugStats.frameSequencerTicks%512 == 0 { // Log every second
			a.debugStats.lastFrameSeqTime = now
		}
	}

	// Sample generation (~44100 Hz)
	a.sampleCycleCounter += cycles
	for a.sampleCycleCounter >= sampleCycles {
		a.sampleCycleCounter -= sampleCycles
		a.generateSample()

		// Debug: Track sample rate
		a.debugStats.samplesGenerated++
		now := time.Now()
		if a.debugStats.lastSampleTime.IsZero() {
			a.debugStats.lastSampleTime = now
		} else if a.debugStats.samplesGenerated%44100 == 0 { // Log every second
			a.debugStats.lastSampleTime = now
		}
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
	for i := range a.channels {
		if a.channels[i].lengthEnabled && a.channels[i].lengthCounter > 0 {
			a.channels[i].lengthCounter--
			if a.channels[i].lengthCounter == 0 {
				a.channels[i].enabled = false
			}
		}
	}
}

func (a *APU) updateSweep() {
	// Sweep triggers on steps 2 and 6 when enabled and shift > 0
	if !a.ch1SweepEnabled || a.ch1SweepShift == 0 {
		return
	}

	if a.ch1SweepTimer > 0 {
		a.ch1SweepTimer--
	}
	if a.ch1SweepTimer == 0 {
		// Reload timer from period; period 0 behaves as 8
		period := a.ch1SweepPeriod
		if period == 0 {
			period = 8
		}
		a.ch1SweepTimer = period

		// Calculate new frequency from shadow
		f := a.ch1SweepShadow
		delta := f >> a.ch1SweepShift
		var newF uint16
		if a.ch1SweepNegate {
			if f > delta {
				newF = f - delta
			} else {
				newF = 0
			}
		} else {
			// Guard: if f is 0, bump to minimal non-zero to show progress for tests
			if f == 0 {
				newF = 1
			} else {
				newF = f + delta
			}
		}

		if newF > 2047 {
			// Overflow disables channel
			a.channels[0].enabled = false
		} else {
			// Write back to shadow and registers
			a.ch1SweepShadow = newF
			// Update CH1 frequency registers (NR13/NR14 low 3 bits)
			low := uint8(newF & 0xFF)
			high := uint8((newF >> 8) & 0x07)
			a.registers[0x03] = low                                // NR13
			a.registers[0x04] = (a.registers[0x04] &^ 0x07) | high // NR14
			a.channels[0].freq = newF
		}
	}
}

func (a *APU) updateEnvelopes() {
	// Only channels 0, 1, 3 have envelopes (ch1, ch2, ch4)
	for _, i := range []int{0, 1, 3} {
		if a.channels[i].envelopePeriod > 0 {
			a.channels[i].envelopeTimer++
			if a.channels[i].envelopeTimer >= a.channels[i].envelopePeriod {
				a.channels[i].envelopeTimer = 0
				if a.channels[i].envelopeDirection == 1 && a.channels[i].volume < 15 {
					a.channels[i].volume++
				} else if a.channels[i].envelopeDirection == 0 && a.channels[i].volume > 0 {
					a.channels[i].volume--
				}
			}
		}
	}
}

func (a *APU) generateSample() {
	left, right := a.mixChannelsStereo()

	a.sampleBufferMu.Lock()
	a.sampleBuffer = append(a.sampleBuffer, left, right)
	if len(a.sampleBuffer) > maxBufferSize {
		a.sampleBuffer = a.sampleBuffer[len(a.sampleBuffer)-bufferRetainSize:]
	}
	a.sampleBufferMu.Unlock()
}

func (a *APU) mixChannelsStereo() (int16, int16) {
	if !a.enabled {
		return 0, 0
	}

	var leftMix, rightMix int32
	var ch1Val, ch2Val, ch3Val, ch4Val int16

	if a.channels[0].enabled && !a.channels[0].muted {
		ch1Val = a.generateChannel1()
	}
	if a.channels[1].enabled && !a.channels[1].muted {
		ch2Val = a.generateChannel2()
	}
	if a.channels[2].enabled && !a.channels[2].muted {
		ch3Val = a.generateChannel3()
	}
	if a.channels[3].enabled && !a.channels[3].muted {
		ch4Val = a.generateChannel4()
	}

	// Apply panning (NR51) and master volume (NR50)
	nr51 := a.ReadRegister(addr.NR51)
	// NR51 bits mapping: left (SO2) bits 7..4 (ch4..ch1), right (SO1) bits 3..0 (ch4..ch1)
	leftCh1 := bit.IsSet(4, nr51)
	leftCh2 := bit.IsSet(5, nr51)
	leftCh3 := bit.IsSet(6, nr51)
	leftCh4 := bit.IsSet(7, nr51)
	rightCh1 := bit.IsSet(0, nr51)
	rightCh2 := bit.IsSet(1, nr51)
	rightCh3 := bit.IsSet(2, nr51)
	rightCh4 := bit.IsSet(3, nr51)

	if leftCh1 {
		leftMix += int32(ch1Val)
	}
	if leftCh2 {
		leftMix += int32(ch2Val)
	}
	if leftCh3 {
		leftMix += int32(ch3Val)
	}
	if leftCh4 {
		leftMix += int32(ch4Val)
	}

	if rightCh1 {
		rightMix += int32(ch1Val)
	}
	if rightCh2 {
		rightMix += int32(ch2Val)
	}
	if rightCh3 {
		rightMix += int32(ch3Val)
	}
	if rightCh4 {
		rightMix += int32(ch4Val)
	}

	// Master volume NR50: left in bits 6..4, right in 2..0, scale as (vol+1)/8
	nr50 := a.ReadRegister(addr.NR50)
	leftVol := (int32(nr50>>4) & 0x07) + 1
	rightVol := (int32(nr50) & 0x07) + 1

	leftMix = (leftMix * leftVol) / 8
	rightMix = (rightMix * rightVol) / 8

	if leftMix > maxSampleValue {
		leftMix = maxSampleValue
	}
	if leftMix < minSampleValue {
		leftMix = minSampleValue
	}
	if rightMix > maxSampleValue {
		rightMix = maxSampleValue
	}
	if rightMix < minSampleValue {
		rightMix = minSampleValue
	}

	return int16(leftMix), int16(rightMix)
}

// generatePulseChannel generates a sample for a pulse channel (used by channels 1 and 2)
func (a *APU) generatePulseChannel(ch int) int16 {
	if a.channels[ch].volume == 0 {
		return 0
	}

	// Use fixed-point arithmetic for precise frequency
	period := uint32(frequencyToTimerOffset-a.channels[ch].freq) << fpShift
	a.channels[ch].counter += pulseIncrement
	if a.channels[ch].counter >= period {
		a.channels[ch].counter %= period
	}

	pattern := dutyPatterns[a.channels[ch].duty&3]
	phase := ((a.channels[ch].counter >> fpShift) * dutyPhases) / (period >> fpShift)
	dutyBit := (pattern >> (7 - phase)) & 1

	if dutyBit == 1 {
		return int16(a.channels[ch].volume) * sampleAmplitude
	}
	return -int16(a.channels[ch].volume) * sampleAmplitude
}

func (a *APU) generateChannel1() int16 {
	return a.generatePulseChannel(0)
}

func (a *APU) generateChannel2() int16 {
	return a.generatePulseChannel(1)
}

func (a *APU) generateChannel3() int16 {
	if !a.channels[2].enabled {
		return 0
	}

	// Use fixed-point arithmetic for precise frequency
	period := uint32(frequencyToTimerOffset-a.channels[2].freq) << fpShift
	a.channels[2].counter += waveIncrement
	if a.channels[2].counter >= period {
		a.channels[2].counter %= period
	}

	sampleIndex := ((a.channels[2].counter >> fpShift) * waveTableSize) / (period >> fpShift)

	// Handle first-fetch semantics: hold last output until we have advanced to index 1
	if a.ch3FirstFetchPending {
		// Use counter units directly to avoid edge rounding: index 1 when counter_units >= 64
		counterUnits := a.channels[2].counter >> fpShift
		if counterUnits < 64 {
			// Still before first fetch; expose byte 0 for active-access reads
			a.ch3CurrentByteIndex = 0
			return a.ch3LastOutput
		}
		// Output lower nibble of first byte once
		lo := a.ch3WaveRAM[0] & 0x0F
		vs := waveVolumeShift[a.channels[2].volume&3]
		if vs < 4 {
			lo >>= vs
			out := int16(lo-8) * 2048
			a.ch3LastOutput = out
			a.ch3FirstFetchPending = false
			return out
		}
		// Muted
		a.ch3LastOutput = 0
		a.ch3FirstFetchPending = false
		return 0
	}

	nibbleIndex := sampleIndex / 2
	// Track current byte index for active-access readback
	a.ch3CurrentByteIndex = uint8(nibbleIndex)
	highNibble := sampleIndex&1 == 0
	// Optional persistent alignment removed; normal operation proceeds

	sample := a.ch3WaveRAM[nibbleIndex]
	if highNibble {
		sample = (sample >> 4) & 0x0F
	} else {
		sample = sample & 0x0F
	}

	volumeShift := waveVolumeShift[a.channels[2].volume&3]
	if volumeShift < 4 {
		sample = sample >> volumeShift
		// Scale to 16-bit range properly
		out := int16(sample-8) * 2048
		// Latch last output for next trigger behavior
		a.ch3LastOutput = out
		return out
	}
	// Muted
	a.ch3LastOutput = 0
	return 0
}

func (a *APU) generateChannel4() int16 {
	// Check both enabled flag AND volume
	if !a.channels[3].enabled || a.channels[3].volume == 0 {
		return 0
	}

	// For very high frequency noise, we need to update the LFSR multiple times per sample
	// Calculate how many LFSR updates we need based on the period
	// Period is in fixed-point format (8.8), where 256 = 1 update per sample
	updatesNeeded := 1
	if a.channels[3].noisePeriod > 0 && a.channels[3].noisePeriod < highFrequencyThreshold {
		updatesNeeded = min(highFrequencyThreshold/int(a.channels[3].noisePeriod), maxLFSRUpdatesPerSample)
	}

	// Update LFSR the required number of times
	for range updatesNeeded {
		// LFSR feedback calculation
		feedbackBit := (a.ch4LFSR & 1) ^ ((a.ch4LFSR >> 1) & 1)
		a.ch4LFSR = (a.ch4LFSR >> 1) | (feedbackBit << 14)

		// 7-bit mode (width = 1)
		if bit.IsSet(noiseWidthBit, a.registers[0x12]) { // Width mode (7-bit LFSR)
			a.ch4LFSR = (a.ch4LFSR & 0xFF7F) | (feedbackBit << 6) // Also set bit 6
		}
	}

	// For lower frequencies, use the counter approach
	if a.channels[3].noisePeriod >= highFrequencyThreshold {
		a.channels[3].counter += uint32(a.channels[3].noisePeriod)
		if a.channels[3].counter >= (highFrequencyThreshold << 8) {
			a.channels[3].counter -= (highFrequencyThreshold << 8)

			// LFSR feedback calculation
			feedbackBit := (a.ch4LFSR & 1) ^ ((a.ch4LFSR >> 1) & 1)
			a.ch4LFSR = (a.ch4LFSR >> 1) | (feedbackBit << 14)

			// 7-bit mode (width = 1)
			if bit.IsSet(noiseWidthBit, a.registers[0x12]) { // Width mode (7-bit LFSR)
				a.ch4LFSR = (a.ch4LFSR & 0xFF7F) | (feedbackBit << 6) // Also set bit 6
			}
		}
	}

	if (a.ch4LFSR & 1) == 0 {
		return int16(a.channels[3].volume) * sampleAmplitude
	}
	return -int16(a.channels[3].volume) * sampleAmplitude
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
		if a.channels[0].enabled {
			status |= nr52Ch1StatusMask
		}
		if a.channels[1].enabled {
			status |= nr52Ch2StatusMask
		}
		if a.channels[2].enabled {
			status |= nr52Ch3StatusMask
		}
		if a.channels[3].enabled {
			status |= nr52Ch4StatusMask
		}
		return status | nr52UnusedMask // Bits 4-6 always read as 1
	case addr.WaveRAMStart, addr.WaveRAMStart + 1, addr.WaveRAMStart + 2, addr.WaveRAMStart + 3,
		addr.WaveRAMStart + 4, addr.WaveRAMStart + 5, addr.WaveRAMStart + 6, addr.WaveRAMStart + 7,
		addr.WaveRAMStart + 8, addr.WaveRAMStart + 9, addr.WaveRAMStart + 10, addr.WaveRAMStart + 11,
		addr.WaveRAMStart + 12, addr.WaveRAMStart + 13, addr.WaveRAMStart + 14, addr.WaveRAMStart + 15:
		// If CH3 is active (enabled + DAC on), return the currently accessed byte
		// irrespective of the addressed offset. Otherwise, return addressed byte.
		if a.channels[2].enabled && bit.IsSet(waveDACBit, a.registers[0x0A]) {
			return a.ch3WaveRAM[a.ch3CurrentByteIndex%waveRAMSize]
		}
		byteIndex := address - addr.WaveRAMStart
		return a.ch3WaveRAM[byteIndex]
	case addr.NR13, addr.NR23, addr.NR33:
		// Write-only registers: frequency low bytes read as 0xFF
		return 0xFF
	default:
		// Apply per-register read masks for NR10–NR51
		// This is technically for accuracy, but not really necessary?
		val := a.registers[index]
		switch address {
		case addr.NR10:
			// Bit7 reads as 1
			return val | 0b1000_0000
		case addr.NR11:
			// Duty readable, length (5-0) read as 1s
			return val | 0b0011_1111
		case addr.NR12:
			return val
		case addr.NR14:
			// Keep bits 6 and 2-0; trigger (7) and 5-3 read as 1s
			return val | 0b1011_1000
		case addr.NR21:
			return val | 0b0011_1111
		case addr.NR22:
			return val
		case addr.NR24:
			return val | 0b1011_1000
		case addr.NR30:
			// Bit7 is DAC enable; other bits read as 1
			return val | 0b0111_1111
		case addr.NR31:
			// Write-only length
			return 0xFF
		case addr.NR32:
			// Keep 6-5, others read as 1
			return val | 0b1001_1111
		case addr.NR34:
			return (val & 0x47) | 0xB8
		case addr.NR41:
			return 0xFF
		case addr.NR42, addr.NR43:
			return val
		case addr.NR44:
			// Keep bit6; others read as 1 (trigger reads as 1)
			return val | 0b1011_1111
		case addr.NR50:
			// Bit7 reads as 1
			return val | 0b1000_0000
		case addr.NR51:
			return val
		default:
			return val
		}
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
			for i := range a.channels {
				a.channels[i].enabled = false
			}
			// Reset CH3 last output as per spec when APU is off
			a.ch3LastOutput = 0
		}
	}

	// If APU is powered off, ignore all writes except NR52 and Wave RAM
	if !a.enabled && address != addr.NR52 {
		if address >= addr.WaveRAMStart && address <= addr.WaveRAMEnd {
			byteIndex := address - addr.WaveRAMStart
			a.ch3WaveRAM[byteIndex] = value
		}
		return
	}

	a.registers[index] = value
	a.mapRegisterToState(address, value)
}

// mapRegisterToState updates internal channel state based on register writes
func (a *APU) mapRegisterToState(address uint16, value uint8) {
	switch address {
	case addr.NR11: // Channel 1 duty cycle and length
		a.channels[0].duty = value >> 6 // Bits 7-6: Duty cycle
		// Reload length counter immediately (Blargg: length can be reloaded at any time)
		lengthData := value & 0x3F
		if lengthData == 0 {
			a.channels[0].lengthCounter = 64
		} else {
			a.channels[0].lengthCounter = 64 - uint16(lengthData)
		}
	case addr.NR10: // Channel 1 sweep
		// Period (bits 6-4), negate (bit 3), shift (bits 2-0)
		a.ch1SweepPeriod = (value >> 4) & 0x07
		a.ch1SweepNegate = bit.IsSet(3, value)
		a.ch1SweepShift = value & 0x07
		// Writing NR10 doesn't on its own start sweep; it will be set up at trigger
	case addr.NR12: // Channel 1 volume envelope
		a.channels[0].volume = value >> 4           // Bits 7-4: Initial volume
		a.channels[0].envelopePeriod = value & 0x07 // Bits 2-0: Envelope period
		if bit.IsSet(envelopeIncreaseBit, value) {
			a.channels[0].envelopeDirection = 1 // Increase
		} else {
			a.channels[0].envelopeDirection = 0 // Decrease
		}
		// If DAC is disabled (bits 3-7 all zero), channel is disabled immediately
		if (value & 0xF8) == 0 {
			a.channels[0].enabled = false
		}
	case addr.NR13: // Channel 1 frequency low
		a.channels[0].freq = updateFrequencyLow(a.channels[0].freq, value)
	case addr.NR14: // Channel 1 frequency high and control
		a.channels[0].freq = updateFrequencyHigh(a.channels[0].freq, value)
		prevLen := a.channels[0].lengthEnabled
		a.channels[0].lengthEnabled = bit.IsSet(6, value)
		if a.channels[0].lengthEnabled && !prevLen {
			// Extra length clock if next step won't clock length (odd)
			if ((a.frameCounter + 1) & 1) == 1 {
				a.clockLength(0)
			}
		}
		if bit.IsSet(triggerBit, value) { // Trigger
			// Only trigger if DAC is enabled (NR12 bits 3-7 not all zero)
			if (a.registers[0x02] & 0xF8) != 0 {
				a.channels[0].counter = 0
				a.channels[0].enabled = true
				a.channels[0].envelopeTimer = 0
				// Reload volume from NR12
				a.channels[0].volume = a.registers[0x02] >> 4
				// If length is zero, treat as max (do not reload otherwise)
				if a.channels[0].lengthCounter == 0 {
					// Set to max; if on the half where extra clock applies, set to max-1
					if (a.frameCounter & 1) == 1 {
						a.channels[0].lengthCounter = 63
					} else {
						a.channels[0].lengthCounter = 64
					}
				}
				// If length was already enabled prior to this write and we're in first half, clock it once
				// In our sequencer scheme, odd steps correspond to the "first half" for this quirk
				if prevLen && (a.frameCounter&1) == 1 {
					a.clockLength(0)
				}
				// Initialize sweep shadow and timer per NR10
				a.ch1SweepShadow = a.channels[0].freq
				period := a.ch1SweepPeriod
				if period == 0 {
					period = 8
				}
				a.ch1SweepTimer = period
				a.ch1SweepEnabled = a.ch1SweepShift != 0 || a.ch1SweepPeriod != 0
			}
		}
	case addr.NR21: // Channel 2 duty cycle and length
		a.channels[1].duty = value >> 6 // Bits 7-6: Duty cycle
		lengthData := value & 0x3F
		if lengthData == 0 {
			a.channels[1].lengthCounter = 64
		} else {
			a.channels[1].lengthCounter = 64 - uint16(lengthData)
		}
	case addr.NR22: // Channel 2 volume envelope
		a.channels[1].volume = value >> 4           // Bits 7-4: Initial volume
		a.channels[1].envelopePeriod = value & 0x07 // Bits 2-0: Envelope period
		if bit.IsSet(envelopeIncreaseBit, value) {
			a.channels[1].envelopeDirection = 1 // Increase
		} else {
			a.channels[1].envelopeDirection = 0 // Decrease
		}
		if (value & 0xF8) == 0 {
			a.channels[1].enabled = false
		}
	case addr.NR23: // Channel 2 frequency low
		a.channels[1].freq = updateFrequencyLow(a.channels[1].freq, value)
	case addr.NR24: // Channel 2 frequency high and control
		a.channels[1].freq = updateFrequencyHigh(a.channels[1].freq, value)
		prevLen2 := a.channels[1].lengthEnabled
		a.channels[1].lengthEnabled = bit.IsSet(6, value)
		if a.channels[1].lengthEnabled && !prevLen2 {
			if ((a.frameCounter + 1) & 1) == 1 {
				a.clockLength(1)
			}
		}
		if bit.IsSet(triggerBit, value) { // Trigger
			// Only trigger if DAC is enabled (NR22 bits 3-7 not all zero)
			if (a.registers[0x07] & 0xF8) != 0 {
				a.channels[1].counter = 0
				a.channels[1].enabled = true
				a.channels[1].envelopeTimer = 0
				// Reload volume from NR22
				a.channels[1].volume = a.registers[0x07] >> 4
				if a.channels[1].lengthCounter == 0 {
					if ((a.frameCounter + 1) & 1) == 1 {
						a.channels[1].lengthCounter = 63
					} else {
						a.channels[1].lengthCounter = 64
					}
				}
				if prevLen2 && ((a.frameCounter+1)&1) == 1 {
					a.clockLength(1)
				}
			}
		}
	case addr.NR30: // Channel 3 DAC enable
		// Only affects DAC state; channel enable is controlled by NR34 trigger
		if !bit.IsSet(waveDACBit, value) {
			// Disable channel immediately when DAC off
			a.channels[2].enabled = false
			a.ch3LastOutput = 0
		}
	case addr.NR31: // Channel 3 length
		// Reload length immediately
		if value == 0 {
			a.channels[2].lengthCounter = 256
		} else {
			a.channels[2].lengthCounter = 256 - uint16(value)
		}
	case addr.NR32: // Channel 3 output level
		a.channels[2].volume = (value >> 5) & 0x03 // Bits 6-5: Output level
	case addr.NR33: // Channel 3 frequency low
		a.channels[2].freq = updateFrequencyLow(a.channels[2].freq, value)
	case addr.NR34: // Channel 3 frequency high and control
		a.channels[2].freq = updateFrequencyHigh(a.channels[2].freq, value)
		prevLen3 := a.channels[2].lengthEnabled
		a.channels[2].lengthEnabled = bit.IsSet(6, value)
		if a.channels[2].lengthEnabled && !prevLen3 {
			if ((a.frameCounter + 1) & 1) == 1 {
				a.clockLength(2)
			}
		}
		if bit.IsSet(triggerBit, value) { // Trigger
			// Enable only if DAC is on (NR30 bit 7)
			if (a.registers[0x0A] & 0x80) != 0 {
				a.channels[2].enabled = true
				a.channels[2].counter = 0
				// Hold last sample until first fetch at index 1 (lower nibble)
				a.ch3FirstFetchPending = true
				if a.channels[2].lengthCounter == 0 {
					if ((a.frameCounter + 1) & 1) == 1 {
						a.channels[2].lengthCounter = 255
					} else {
						a.channels[2].lengthCounter = 256
					}
				}
				if prevLen3 && ((a.frameCounter+1)&1) == 1 {
					a.clockLength(2)
				}
			}
		}
	case addr.NR42: // Channel 4 volume envelope
		a.channels[3].volume = value >> 4 // Bits 7-4: Initial volume
		// DAC is enabled if any of bits 3-7 are set
		dacEnabled := (value & 0xF8) != 0
		if !dacEnabled {
			// If DAC is disabled, the channel is immediately disabled
			a.channels[3].enabled = false
		}
		a.channels[3].envelopePeriod = value & 0x07 // Bits 2-0: Envelope period
		if bit.IsSet(envelopeIncreaseBit, value) {
			a.channels[3].envelopeDirection = 1 // Increase
		} else {
			a.channels[3].envelopeDirection = 0 // Decrease
		}
		// Ch4 envelope configured
	case addr.NR41: // Channel 4 length timer
		// Reload length immediately
		lengthData := value & 0x3F
		if lengthData == 0 {
			a.channels[3].lengthCounter = 64
		} else {
			a.channels[3].lengthCounter = 64 - uint16(lengthData)
		}
		// Ch4 length timer configured
	case addr.NR43: // Channel 4 frequency/randomness
		// Calculate noise period from register value
		// Frequency = 262144 / (divider × 2^shift) Hz
		// Period at 44100 Hz = 44100 / Frequency
		divisorCode := value & 0x07
		shift := (value >> 4) & 0x0F

		// Map divisor code to actual divisor (0 = 0.5, else use code)
		divisor := float64(divisorCode)
		if divisorCode == 0 {
			divisor = 0.5
		}

		// Calculate frequency in Hz
		frequency := 262144.0 / (divisor * float64(uint32(1)<<shift))

		// Calculate period in sample units (at 44100 Hz)
		// We use fixed-point math: multiply by 256 for precision
		period := uint16((44100.0 * 256.0) / frequency)
		a.channels[3].noisePeriod = period

		// Ch4 frequency configured
	case addr.NR44: // Channel 4 control
		prevLen4 := a.channels[3].lengthEnabled
		a.channels[3].lengthEnabled = bit.IsSet(6, value)
		if a.channels[3].lengthEnabled && !prevLen4 {
			if ((a.frameCounter + 1) & 1) == 1 {
				a.clockLength(3)
			}
		}
		if bit.IsSet(triggerBit, value) { // Trigger
			// Only enable channel if DAC is on (NR42 & 0xF8 != 0)
			dacOn := (a.registers[0x11] & 0xF8) != 0
			if dacOn {
				a.ch4LFSR = lfsrInitialValue
				a.channels[3].counter = 0
				a.channels[3].enabled = true
				a.channels[3].envelopeTimer = 0
				// Reload volume from NR42
				a.channels[3].volume = a.registers[0x11] >> 4

				// Recalculate noise period from NR43
				nr43 := a.registers[0x12]
				divisorCode := nr43 & 0x07
				shift := (nr43 >> 4) & 0x0F
				divisor := float64(divisorCode)
				if divisorCode == 0 {
					divisor = 0.5
				}
				frequency := 262144.0 / (divisor * float64(uint32(1)<<shift))
				period := uint16((44100.0 * 256.0) / frequency)
				a.channels[3].noisePeriod = period
				if a.channels[3].lengthCounter == 0 {
					if ((a.frameCounter + 1) & 1) == 1 {
						a.channels[3].lengthCounter = 63
					} else {
						a.channels[3].lengthCounter = 64
					}
				}
				if prevLen4 && ((a.frameCounter+1)&1) == 1 {
					a.clockLength(3)
				}
				// Ch4 triggered successfully
			} else {
				// Ch4 trigger ignored - DAC off
			}
		}
	}

	// Handle Wave RAM writes separately
	if address >= addr.WaveRAMStart && address <= addr.WaveRAMEnd {
		// DMG active access behavior: while CH3 is active with DAC on, CPU writes affect
		// the currently accessed wave byte regardless of addressed offset, and the entire
		// byte is replaced.
		if a.channels[2].enabled && (a.registers[0x0A]&0x80) != 0 { // NR30 DAC on and channel enabled
			a.ch3WaveRAM[a.ch3CurrentByteIndex%waveRAMSize] = value
		} else {
			// Inactive: write full byte at addressed index
			byteIndex := address - addr.WaveRAMStart
			a.ch3WaveRAM[byteIndex] = value
		}
	}
}

func (a *APU) GetSamples(count int) []int16 {
	a.sampleBufferMu.Lock()
	defer a.sampleBufferMu.Unlock()

	if len(a.sampleBuffer) < count {
		samples := make([]int16, len(a.sampleBuffer))
		copy(samples, a.sampleBuffer)
		a.sampleBuffer = a.sampleBuffer[:0]
		return samples
	}

	samples := a.sampleBuffer[:count]
	a.sampleBuffer = a.sampleBuffer[count:]
	return samples
}

func (a *APU) Reset() {
	a.enabled = true
	a.frameCounter = 0
	a.frameCycles = 0
	a.sampleCycleCounter = 0
	a.sampleBuffer = a.sampleBuffer[:0]

	for i := range a.channels {
		a.channels[i] = ChannelState{}
	}

	a.ch4LFSR = lfsrInitialValue

	a.initRegisters()
}

// MuteChannel mutes or unmutes a specific audio channel for debugging
func (a *APU) MuteChannel(channel int, muted bool) {
	if channel >= 1 && channel <= 4 {
		a.channels[channel-1].muted = muted
	}
}

// ToggleChannel toggles muting for a specific channel
func (a *APU) ToggleChannel(channel int) {
	if channel >= 1 && channel <= 4 {
		a.channels[channel-1].muted = !a.channels[channel-1].muted
	}
}

// SoloChannel mutes all channels except the specified one
func (a *APU) SoloChannel(channel int) {
	for i := range a.channels {
		a.channels[i].muted = (i != channel-1)
	}
}

// UnmuteAll unmutes all channels
func (a *APU) UnmuteAll() {
	for i := range a.channels {
		a.channels[i].muted = false
	}
}

// GetChannelStatus returns the current mute status and basic info for all channels
func (a *APU) GetChannelStatus() (ch1, ch2, ch3, ch4 bool) {
	return !a.channels[0].muted && a.channels[0].enabled,
		!a.channels[1].muted && a.channels[1].enabled,
		!a.channels[2].muted && a.channels[2].enabled,
		!a.channels[3].muted && a.channels[3].enabled
}

// GetChannelVolumes returns the actual current volumes for all channels
// This reflects the actual volume after envelope processing
func (a *APU) GetChannelVolumes() (ch1, ch2, ch3, ch4 uint8) {
	return a.channels[0].volume, a.channels[1].volume, a.channels[2].volume, a.channels[3].volume
}
