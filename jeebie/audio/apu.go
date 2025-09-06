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

	// Channel states (indexed 0-3 for channels 1-4)
	channels [4]ChannelState

	// Channel 3 specific
	ch3WaveRAM [waveRAMSize]uint8

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

	// Initialize noise period with default value from NR43 = 0x00
	// divisor = 0.5, shift = 0, frequency = 524288 Hz
	// period = (44100 * 256) / 524288 = ~21.5
	a.channels[3].noisePeriod = 21
}

func (a *APU) Tick(cycles int) {
	if !a.enabled {
		return
	}

	a.mu.Lock()
	defer a.mu.Unlock()

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
	var ch1Val, ch2Val, ch3Val, ch4Val int16

	if a.channels[0].enabled && !a.channels[0].muted {
		ch1Val = a.generateChannel1()
		mixed += int32(ch1Val)
	}
	if a.channels[1].enabled && !a.channels[1].muted {
		ch2Val = a.generateChannel2()
		mixed += int32(ch2Val)
	}
	if a.channels[2].enabled && !a.channels[2].muted {
		ch3Val = a.generateChannel3()
		mixed += int32(ch3Val)
	}
	if a.channels[3].enabled && !a.channels[3].muted {
		ch4Val = a.generateChannel4()
		mixed += int32(ch4Val)
	}

	if mixed > maxSampleValue {
		mixed = maxSampleValue
	} else if mixed < minSampleValue {
		mixed = minSampleValue
	}

	return int16(mixed)
}

// generatePulseChannel generates a sample for a pulse channel (used by channels 1 and 2)
func (a *APU) generatePulseChannel(ch int) int16 {
	if a.channels[ch].volume == 0 || a.channels[ch].freq == 0 {
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
	if !a.channels[2].enabled || a.channels[2].freq == 0 {
		return 0
	}

	// Use fixed-point arithmetic for precise frequency
	period := uint32(frequencyToTimerOffset-a.channels[2].freq) << fpShift
	a.channels[2].counter += waveIncrement
	if a.channels[2].counter >= period {
		a.channels[2].counter %= period
	}

	sampleIndex := ((a.channels[2].counter >> fpShift) * waveTableSize) / (period >> fpShift)
	nibbleIndex := sampleIndex / 2
	highNibble := sampleIndex&1 == 0

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
		return int16(sample-8) * 2048
	}
	return 0 // Muted
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
		updatesNeeded = highFrequencyThreshold / int(a.channels[3].noisePeriod)
		if updatesNeeded > maxLFSRUpdatesPerSample {
			updatesNeeded = maxLFSRUpdatesPerSample
		}
	}

	// Update LFSR the required number of times
	for i := 0; i < updatesNeeded; i++ {
		// LFSR feedback calculation
		feedbackBit := (a.ch4LFSR & 1) ^ ((a.ch4LFSR >> 1) & 1)
		a.ch4LFSR = (a.ch4LFSR >> 1) | (feedbackBit << 14)

		// 7-bit mode (width = 1)
		if bit.IsSet(noiseWidthBit, a.registers[0x22]) { // Width mode (7-bit LFSR)
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
			if bit.IsSet(noiseWidthBit, a.registers[0x22]) { // Width mode (7-bit LFSR)
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
			for i := range a.channels {
				a.channels[i].enabled = false
			}
		}
	}

	a.registers[index] = value
	a.mapRegisterToState(address, value)
}

// mapRegisterToState updates internal channel state based on register writes
func (a *APU) mapRegisterToState(address uint16, value uint8) {
	switch address {
	case addr.NR11: // Channel 1 duty cycle and length
		a.channels[0].duty = value >> 6 // Bits 7-6: Duty cycle
		// Length data is stored in register, loaded on trigger
	case addr.NR12: // Channel 1 volume envelope
		a.channels[0].volume = value >> 4           // Bits 7-4: Initial volume
		a.channels[0].enabled = (value & 0xF8) != 0 // DAC enabled if bits 3-7 are not all zero
		a.channels[0].envelopePeriod = value & 0x07 // Bits 2-0: Envelope period
		if bit.IsSet(envelopeIncreaseBit, value) {
			a.channels[0].envelopeDirection = 1 // Increase
		} else {
			a.channels[0].envelopeDirection = 0 // Decrease
		}
	case addr.NR13: // Channel 1 frequency low
		a.channels[0].freq = updateFrequencyLow(a.channels[0].freq, value)
	case addr.NR14: // Channel 1 frequency high and control
		a.channels[0].freq = updateFrequencyHigh(a.channels[0].freq, value)
		a.channels[0].lengthEnabled = bit.IsSet(6, value)
		if bit.IsSet(triggerBit, value) { // Trigger
			// Only trigger if DAC is enabled
			if (a.registers[0x12] & 0xF8) != 0 {
				a.channels[0].counter = 0
				a.channels[0].enabled = true
				a.channels[0].envelopeTimer = 0
				// Reload volume from NR12
				a.channels[0].volume = a.registers[0x12] >> 4
				// Always reload length counter on trigger
				lengthData := a.registers[0x11] & 0x3F
				if lengthData == 0 {
					a.channels[0].lengthCounter = 64
				} else {
					a.channels[0].lengthCounter = 64 - uint16(lengthData)
				}
			}
		}
	case addr.NR21: // Channel 2 duty cycle and length
		a.channels[1].duty = value >> 6 // Bits 7-6: Duty cycle
		// Length data is stored in register, loaded on trigger
	case addr.NR22: // Channel 2 volume envelope
		a.channels[1].volume = value >> 4           // Bits 7-4: Initial volume
		a.channels[1].enabled = (value & 0xF8) != 0 // DAC enabled if bits 3-7 are not all zero
		a.channels[1].envelopePeriod = value & 0x07 // Bits 2-0: Envelope period
		if bit.IsSet(envelopeIncreaseBit, value) {
			a.channels[1].envelopeDirection = 1 // Increase
		} else {
			a.channels[1].envelopeDirection = 0 // Decrease
		}
	case addr.NR23: // Channel 2 frequency low
		a.channels[1].freq = updateFrequencyLow(a.channels[1].freq, value)
	case addr.NR24: // Channel 2 frequency high and control
		a.channels[1].freq = updateFrequencyHigh(a.channels[1].freq, value)
		a.channels[1].lengthEnabled = bit.IsSet(6, value)
		if bit.IsSet(triggerBit, value) { // Trigger
			// Only trigger if DAC is enabled
			if (a.registers[0x17] & 0xF8) != 0 {
				a.channels[1].counter = 0
				a.channels[1].enabled = true
				a.channels[1].envelopeTimer = 0
				// Reload volume from NR22
				a.channels[1].volume = a.registers[0x17] >> 4
				// Always reload length counter on trigger
				lengthData := a.registers[0x16] & 0x3F
				if lengthData == 0 {
					a.channels[1].lengthCounter = 64
				} else {
					a.channels[1].lengthCounter = 64 - uint16(lengthData)
				}
			}
		}
	case addr.NR30: // Channel 3 DAC enable
		a.channels[2].enabled = bit.IsSet(waveDACBit, value) // DAC enable
	case addr.NR31: // Channel 3 length
		// Length data is stored in register, loaded on trigger
	case addr.NR32: // Channel 3 output level
		a.channels[2].volume = (value >> 5) & 0x03 // Bits 6-5: Output level
	case addr.NR33: // Channel 3 frequency low
		a.channels[2].freq = updateFrequencyLow(a.channels[2].freq, value)
	case addr.NR34: // Channel 3 frequency high and control
		a.channels[2].freq = updateFrequencyHigh(a.channels[2].freq, value)
		a.channels[2].lengthEnabled = bit.IsSet(6, value)
		if bit.IsSet(triggerBit, value) { // Trigger
			a.channels[2].counter = 0
			// Always reload length counter on trigger
			lengthData := a.registers[0x1B]
			if lengthData == 0 {
				a.channels[2].lengthCounter = 256
			} else {
				a.channels[2].lengthCounter = 256 - uint16(lengthData)
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
		// Length data is stored in register, loaded on trigger
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
		a.channels[3].lengthEnabled = bit.IsSet(6, value)
		if bit.IsSet(triggerBit, value) { // Trigger
			// Only enable channel if DAC is on (NR42 & 0xF8 != 0)
			dacOn := (a.registers[0x21] & 0xF8) != 0
			if dacOn {
				a.ch4LFSR = lfsrInitialValue
				a.channels[3].counter = 0
				a.channels[3].enabled = true
				a.channels[3].envelopeTimer = 0
				// Reload volume from NR42
				a.channels[3].volume = a.registers[0x21] >> 4

				// Recalculate noise period from NR43
				nr43 := a.registers[0x22]
				divisorCode := nr43 & 0x07
				shift := (nr43 >> 4) & 0x0F
				divisor := float64(divisorCode)
				if divisorCode == 0 {
					divisor = 0.5
				}
				frequency := 262144.0 / (divisor * float64(uint32(1)<<shift))
				period := uint16((44100.0 * 256.0) / frequency)
				a.channels[3].noisePeriod = period
				// Always reload length counter on trigger
				lengthData := a.registers[0x20] & 0x3F
				if lengthData == 0 {
					a.channels[3].lengthCounter = 64
				} else {
					a.channels[3].lengthCounter = 64 - uint16(lengthData)
				}
				// Ch4 triggered successfully
			} else {
				// Ch4 trigger ignored - DAC off
			}
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

	for i := range a.channels {
		a.channels[i] = ChannelState{}
	}

	a.ch4LFSR = lfsrInitialValue

	a.initRegisters()
}

// MuteChannel mutes or unmutes a specific audio channel for debugging
func (a *APU) MuteChannel(channel int, muted bool) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if channel >= 1 && channel <= 4 {
		a.channels[channel-1].muted = muted
	}
}

// ToggleChannel toggles muting for a specific channel
func (a *APU) ToggleChannel(channel int) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if channel >= 1 && channel <= 4 {
		a.channels[channel-1].muted = !a.channels[channel-1].muted
	}
}

// SoloChannel mutes all channels except the specified one
func (a *APU) SoloChannel(channel int) {
	a.mu.Lock()
	defer a.mu.Unlock()

	for i := range a.channels {
		a.channels[i].muted = (i != channel-1)
	}
}

// UnmuteAll unmutes all channels
func (a *APU) UnmuteAll() {
	a.mu.Lock()
	defer a.mu.Unlock()

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
	a.mu.Lock()
	defer a.mu.Unlock()

	return a.channels[0].volume, a.channels[1].volume, a.channels[2].volume, a.channels[3].volume
}
