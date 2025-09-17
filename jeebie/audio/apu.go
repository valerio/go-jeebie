package audio

import (
	"github.com/valerio/go-jeebie/jeebie/addr"
	"github.com/valerio/go-jeebie/jeebie/bit"
	"github.com/valerio/go-jeebie/jeebie/timing"
)

// APU is the Audio Processing Unit of a DMG Game Boy. It generates 4-channel audio:
// CH1 (square+sweep), CH2 (square), CH3 (wave), CH4 (noise), all mixed to stereo output.
// This is basically a bunch of counters and timers that tick at certain frequency steps!
type APU struct {

	// state, this is information derived from registers/memory.
	enabled           bool
	ch                [4]Channel
	vinLeft, vinRight bool  // from NR50
	volLeft, volRight uint8 // volume for left/right, values 0 to 7
	vinSample         int16 // external VIN input sample (Pan Docs: Audio mixing - VIN)

	// accumulators for mixing samples
	mixLeftAcc         int64
	mixRightAcc        int64
	mixAccumCycles     int
	pcmBuffer          []int16
	pcmCursor          int
	pcmCycleAcc        float64
	pcmCyclesPerSample float64
	hostSampleRate     int

	// frame sequencer state
	step   int // current step (0-7)
	cycles int // cycles since last frame sequencer tick

	// raw memory + registers
	NR10, NR11, NR12, NR13, NR14 uint8 // Channel 1
	NR21, NR22, NR23, NR24       uint8 // Channel 2
	NR30, NR31, NR32, NR33, NR34 uint8 // Channel 3
	NR41, NR42, NR43, NR44       uint8 // Channel 4
	NR50, NR51, NR52             uint8 // Global controls
	waveRAM                      [waveRAMSize]uint8
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

	duty   uint8  // for square waves, values 0 to 3
	timer  uint8  // initial length timer value, 6 bits for ch1-2-4 -> values 0 to 63, 8 bits for ch3 -> values 0 to 255
	length uint16 // current length counter, can hold up to 256 for CH3
	volume uint8  // initial volume, 4 bits -> values 0 to 15

	// Frequency sweep (CH1 only)
	sweepPeriod  uint8  // "pace" per pandocs (NR10 6-4), 3 bits -> values 0 to 7
	sweepDown    bool   // sweep direction, 0=up, 1=down
	sweepStep    uint8  // sweep step, 3 bits -> values 0 to 7
	sweepEnabled bool   // true if sweep is enabled (either period or step non-zero)
	sweepTimer   uint8  // timer for sweep calculations
	shadowFreq   uint16 // shadow frequency for sweep calculations
	sweepNegUsed bool   // flag for subtract-mode calculations (Pan Docs: Audio details - sweep negate bug)

	envelopePace    uint8 // NRx2 bits 7-4, 3 bits -> values 0 to 7
	envelopeUp      bool  // NRx2 bit 3, 0=down, 1=up
	envelopeCounter uint8
	envelopeLatched bool

	period       uint16 // frequency period, 11 bits -> values 0 to 2047
	trigger      bool   // trigger flag, write-only, when written it "triggers" the channel
	lengthEnable bool   // length enable flag
	freqTimer    int
	dutyStep     uint8
	waveIndex    uint8
	waveSample   uint8
	noiseTimer   int

	// CH4 Noise channel specific
	lfsr        uint16 // 15-bit LFSR for noise generation
	use7bitLFSR bool   // from NR43 bit 3, when set use 7-bit LFSR, otherwise 15-bit
	shift       uint8  // from NR43, 4 bits -> values 0 to 15
	divider     uint8  // from NR43, 3 bits -> values 0 to 7

	dacEnabled bool // for channel 3, DAC enable flag

	// Debug state
	muted bool // separate from enabled/dac
}

// calculateSweepFrequency performs the sweep frequency calculation.
func (ch *Channel) calculateSweepFrequency() (newFreq uint16, overflow bool) {
	if ch.sweepStep == 0 {
		return ch.shadowFreq, false
	}
	return ch.checkSweepOverflow()
}

// checkSweepOverflow computes the sweep target regardless of
// sweepStep being zero. This is used for the periodic overflow check that occurs
// even when shift==0. It does NOT mutate channel state.
func (ch *Channel) checkSweepOverflow() (newFreq uint16, overflow bool) {
	freqChange := ch.shadowFreq >> ch.sweepStep
	if ch.sweepDown {
		if freqChange > ch.shadowFreq {
			newFreq = 0
		} else {
			newFreq = ch.shadowFreq - freqChange
		}
	} else {
		newFreq = ch.shadowFreq + freqChange
	}
	return newFreq, newFreq > 2047
}

func New() *APU {
	apu := &APU{hostSampleRate: 44100} // 44100Hz
	apu.pcmCyclesPerSample = float64(timing.CPUFrequency) / float64(apu.hostSampleRate)
	return apu
}

// Tick advances the APU by CPU T-cycles.
func (a *APU) Tick(cycles int) {
	if !a.enabled {
		return
	}

	a.tickGenerators(cycles)

	a.cycles += cycles

	// Audio stepping plan during Tick:
	//  1. Consume CPU cycles into per-channel timers and generate raw amplitude ticks.
	//  2. Push each raw tick into an intermediate mix accumulator at the hardware rate.
	//  3. When the accumulator spans the host sample period, average and store it for GetSamples.

	// Every 512Hz, advance the frame sequencer
	for a.cycles >= cyclesPerStep {
		a.cycles -= cyclesPerStep
		a.tickSequence()
	}
}

func (a *APU) tickGenerators(cycles int) {
	// Channel generator plan:
	//  1. Add CPU cycles to each channel's timer and reload when the period elapses.
	//  2. Update the duty/wave/LFSR position to produce the next raw amplitude for that channel.
	//  3. Gate the amplitude by the channel's DAC/envelope state to get the audible level.
	//  4. Mix the level into left/right accumulators according to NR51 so GetSamples can downsample later.
	if cycles <= 0 {
		return
	}

	var leftLevel, rightLevel int64
	for i := range 4 {
		ch := &a.ch[i]
		if !ch.enabled || !ch.dacEnabled || ch.muted {
			continue
		}

		var level int64
		switch i {
		case 0, 1:
			level = a.stepSquare(ch, cycles)
		case 2:
			level = a.stepWave(ch, cycles)
		case 3:
			level = a.stepNoise(ch, cycles)
		}
		if level == 0 {
			continue
		}

		if ch.left {
			leftLevel += level
		}
		if ch.right {
			rightLevel += level
		}
	}
	// VIN pin is optional, it feeds each mixer lane
	if a.vinLeft {
		leftLevel += int64(a.vinSample)
	}
	if a.vinRight {
		rightLevel += int64(a.vinSample)
	}

	a.mixLeftAcc += leftLevel * int64(cycles)
	a.mixRightAcc += rightLevel * int64(cycles)
	a.mixAccumCycles += cycles
	a.flushMix(cycles)
}

func (a *APU) flushMix(cycles int) {
	if a.hostSampleRate <= 0 || a.pcmCyclesPerSample == 0 {
		return
	}

	a.pcmCycleAcc += float64(cycles)
	if a.pcmCycleAcc < a.pcmCyclesPerSample {
		return
	}
	a.pcmCycleAcc -= a.pcmCyclesPerSample

	left, right := a.exportMixedSample()
	a.pcmBuffer = append(a.pcmBuffer, left, right)
}

func (a *APU) exportMixedSample() (int16, int16) {
	if a.mixAccumCycles == 0 {
		return 0, 0
	}

	leftAvg := float64(a.mixLeftAcc) / float64(a.mixAccumCycles)
	rightAvg := float64(a.mixRightAcc) / float64(a.mixAccumCycles)

	left, right := scaleToPCM(leftAvg, a.volLeft), scaleToPCM(rightAvg, a.volRight)

	a.mixLeftAcc = 0
	a.mixRightAcc = 0
	a.mixAccumCycles = 0

	return left, right
}

func (a *APU) stepSquare(ch *Channel, cycles int) int64 {
	period := a.squarePeriodCycles(ch)
	if period == 0 {
		return 0
	}
	if ch.freqTimer <= 0 {
		ch.freqTimer = period
	}

	ch.freqTimer -= cycles
	for ch.freqTimer <= 0 {
		ch.freqTimer += period
		ch.dutyStep = (ch.dutyStep + 1) & 0x7
	}

	pattern := dutyPatterns[ch.duty&0x3][ch.dutyStep]
	if ch.volume == 0 {
		return 0
	}
	level := int64(ch.volume)
	if pattern == 0 {
		// Per Pan Docs: if the duty cycle is 0, the output is 0
		// so we mirror the level to have a DC-free signal.
		return -level
	}
	return level
}

func (a *APU) stepWave(ch *Channel, cycles int) int64 {
	period := a.wavePeriodCycles(ch)
	if period == 0 {
		return 0
	}
	if ch.freqTimer <= 0 {
		ch.freqTimer = period
	}

	ch.freqTimer -= cycles
	for ch.freqTimer <= 0 {
		ch.freqTimer += period
		ch.waveIndex = (ch.waveIndex + 1) & 0x1F
	}

	sample := int64(a.readWaveSample(ch.waveIndex)) - 8
	switch ch.volume & 0b11 {
	case 0:
		return 0
	case 1:
		return sample
	case 2:
		return sample / 2
	case 3:
		return sample / 4
	default:
		return sample
	}
}

func (a *APU) stepNoise(ch *Channel, cycles int) int64 {
	period := a.noisePeriodCycles(ch)
	if period == 0 {
		return 0
	}
	if ch.lfsr == 0 {
		ch.lfsr = 0x7FFF
	}
	if ch.noiseTimer <= 0 {
		ch.noiseTimer = period
	}

	ch.noiseTimer -= cycles
	for ch.noiseTimer <= 0 {
		ch.noiseTimer += period
		bit := (ch.lfsr & 1) ^ ((ch.lfsr >> 1) & 1)
		ch.lfsr = (ch.lfsr >> 1) | (bit << 14)
		if ch.use7bitLFSR {
			ch.lfsr = (ch.lfsr &^ (1 << 6)) | (bit << 6)
		}
	}

	if ch.volume == 0 {
		return 0
	}
	level := int64(ch.volume)
	if bit.IsSet(0, uint8(ch.lfsr)) {
		// Per Pan Docs: Noise output bit is inverted before it hits the DAC
		return -level
	}
	return level
}

func (a *APU) squarePeriodCycles(ch *Channel) int {
	period := 2048 - int(ch.period&0x7FF)
	if period <= 0 {
		return 0
	}
	return period * 4
}

func (a *APU) wavePeriodCycles(ch *Channel) int {
	period := 2048 - int(ch.period&0x7FF)
	if period <= 0 {
		return 0
	}
	return period * 2
}

var noiseDividers = [8]int{8, 16, 32, 48, 64, 80, 96, 112}

func (a *APU) noisePeriodCycles(ch *Channel) int {
	div := noiseDividers[ch.divider&0x7]
	period := div << ch.shift
	if period <= 0 {
		return 0
	}
	return period
}

func (a *APU) readWaveSample(index uint8) uint8 {
	byteIdx := index >> 1
	value := a.waveRAM[byteIdx]
	a.ch[2].waveSample = value
	if index&1 == 0 {
		return value >> 4
	}
	return value & 0x0F
}

// Per Pan Docs: Wave RAM is locked to the CPU while
// CH3 is enabled with the DAC on (Wave channel).
func (a *APU) waveRAMLocked() bool {
	return a.enabled && a.ch[2].enabled && a.ch[2].dacEnabled
}

var dutyPatterns = [4][8]int64{
	{0, 1, 0, 0, 0, 0, 0, 0},
	{0, 1, 1, 0, 0, 0, 0, 0},
	{0, 1, 1, 1, 1, 0, 0, 0},
	{1, 0, 0, 1, 1, 1, 1, 1},
}

const sampleScale = 32767.0 / 15.0

func scaleToPCM(avg float64, masterVol uint8) int16 {
	gain := float64(masterVol+1) / 8.0
	value := avg * gain * sampleScale
	if value > 32767 {
		value = 32767
	} else if value < -32768 {
		value = -32768
	}
	return int16(value)
}

// tickSequence advances the sequencer by one step.
// We advance one step every 512Hz (8192 T-cycles), then
// depending on the step we tick length, sweep, and/or envelope.
//
//	Step | Length (256Hz) | Sweep (128Hz) | Envelope (64Hz)
//	------------------------------------------------------
//	0    | yes            | -             | -
//	1    | -              | -             | -
//	2    | yes            | yes           | -
//	3    | -              | -             | -
//	4    | yes            | -             | -
//	5    | -              | -             | -
//	6    | yes            | yes           | -
//	7    | -              | -             | yes
func (a *APU) tickSequence() {
	switch a.step {
	case 0:
		a.tickLength()
	case 2:
		a.tickLength()
		a.tickSweep()
	case 4:
		a.tickLength()
	case 6:
		a.tickLength()
		a.tickSweep()
	case 7:
		a.tickEnvelope()
	}

	a.step++
	a.step %= 8
}

func (a *APU) tickLength() {
	// Length counters: when enabled, decrement each channel's length counter
	// When length reaches 0, disable the channel (except for wave channel which has different behavior)
	// CH1/CH2/CH4: length = (64 - NRx1), counts down from 64 to 0
	// CH3: length = (256 - NR31), counts down from 256 to 0

	for i := range 4 {
		if a.ch[i].lengthEnable && a.ch[i].length > 0 {
			a.ch[i].length--

			// if length reaches 0, disable channel
			if a.ch[i].length == 0 {
				a.ch[i].enabled = false
			}
		}
	}

}

func (a *APU) tickSweep() {
	// Frequency sweep only applies to CH1 (channel 0)
	ch := &a.ch[0]

	if !ch.sweepEnabled {
		return
	}

	// tick down, we continue only if it reaches 0
	ch.sweepTimer--
	if ch.sweepTimer > 0 {
		return
	}

	ch.sweepTimer = ch.sweepPeriod
	if ch.sweepTimer == 0 {
		ch.sweepTimer = 8
	}

	// Per dmg_sound tests: if period==0, do not perform calculations on ticks
	if ch.sweepPeriod == 0 {
		return
	}

	// First: perform overflow check.
	newFrequency, overflow := ch.checkSweepOverflow()
	if overflow {
		ch.enabled = false
		return
	}
	// Mark negate-used on any subtract-mode calculation tick, even if shift==0.
	if ch.sweepDown {
		ch.sweepNegUsed = true
	}
	// If shift==0, do not update frequency on tick
	if ch.sweepStep == 0 {
		return
	}
	// Update the frequency registers (NR13/NR14 11 bits total)
	ch.shadowFreq = newFrequency
	ch.period = newFrequency
	a.NR14 = (a.NR14 & 0b11111000) | uint8((newFrequency>>8)&0b111)
	a.NR13 = uint8(newFrequency)

	// Do the calculation AGAIN for overflow check only
	// (This weird behavior is documented in Pan Docs)
	if _, overflow := ch.checkSweepOverflow(); overflow {
		ch.enabled = false
	}
}

func (a *APU) tickEnvelope() {
	for _, idx := range []int{0, 1, 3} {
		ch := &a.ch[idx]
		// Per Pan Docs: Envelope timer continues running even if the channel is currently silent
		// so we avoid checking ch.enabled here.
		if !ch.dacEnabled {
			continue
		}
		if ch.envelopeLatched {
			continue
		}

		pace := ch.envelopePace
		if pace == 0 {
			pace = 8
		}

		if ch.envelopeCounter == 0 {
			ch.envelopeCounter = pace
		}
		ch.envelopeCounter--
		if ch.envelopeCounter > 0 {
			continue
		}

		if ch.envelopeUp {
			if ch.volume < 15 {
				ch.volume++
				ch.envelopeCounter = pace
			} else {
				ch.envelopeLatched = true
				ch.envelopeCounter = 0
			}
		} else {
			if ch.volume > 0 {
				ch.volume--
				ch.envelopeCounter = pace
			} else {
				ch.envelopeLatched = true
				ch.envelopeCounter = 0
			}
		}
	}
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
		return a.NR14 | 0b1011_1111
	case addr.NR21:
		return a.NR21 | 0b0011_1111
	case addr.NR22:
		return a.NR22
	case addr.NR23:
		return 0xFF // write-only reg
	case addr.NR24:
		return a.NR24 | 0b1011_1111
	case addr.NR30:
		return a.NR30 | 0b0111_1111
	case addr.NR31:
		return 0xFF // write-only reg
	case addr.NR32:
		return a.NR32 | 0b1001_1111
	case addr.NR33:
		return 0xFF // write-only reg
	case addr.NR34:
		return a.NR34 | 0b1011_1111
	case addr.NR41:
		return 0xFF // write-only reg
	case addr.NR42:
		return a.NR42
	case addr.NR43:
		return a.NR43
	case addr.NR44:
		return a.NR44 | 0b1011_1111
	case addr.NR50:
		return a.NR50
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
		if a.waveRAMLocked() {
			// Per Pan Docs: When wave channel is active the CPU
			// sees the current sample buffer instead of RAM.
			return a.ch[2].waveSample
		}
		return a.waveRAM[address-addr.WaveRAMStart]
	}
	// unmapped - panic?
	return 0xFF
}

// WriteRegister stores the value of the given register/memory, then updates
// internal state accordingly.
func (a *APU) WriteRegister(address uint16, value uint8) {
	isInWaveRAM := address >= addr.WaveRAMStart && address <= addr.WaveRAMEnd

	if !a.enabled && address != addr.NR52 && !isInWaveRAM {
		// and ignore writes to audio regs except NR52/RAM when powered off
		return
	}

	switch address {
	case addr.NR10:
		a.NR10 = value
	case addr.NR11:
		a.NR11 = value
		a.ch[0].length = 64 - uint16(bit.ExtractBits(value, 5, 0))
	case addr.NR12:
		a.NR12 = value
		ch := &a.ch[0]
		pace := bit.ExtractBits(value, 2, 0)
		if pace == 0 {
			ch.envelopeCounter = 8
		} else {
			ch.envelopeCounter = pace
		}
		ch.envelopeLatched = false
	case addr.NR13:
		a.NR13 = value
	case addr.NR14:
		a.NR14 = value
	case addr.NR21:
		a.NR21 = value
		a.ch[1].length = 64 - uint16(bit.ExtractBits(value, 5, 0))
	case addr.NR22:
		a.NR22 = value
		ch := &a.ch[1]
		pace := bit.ExtractBits(value, 2, 0)
		if pace == 0 {
			ch.envelopeCounter = 8
		} else {
			ch.envelopeCounter = pace
		}
		ch.envelopeLatched = false
	case addr.NR23:
		a.NR23 = value
	case addr.NR24:
		a.NR24 = value
	case addr.NR30:
		a.NR30 = value
	case addr.NR31:
		a.NR31 = value
		a.ch[2].length = 256 - uint16(value)
	case addr.NR32:
		a.NR32 = value
	case addr.NR33:
		a.NR33 = value
	case addr.NR34:
		a.NR34 = value
	case addr.NR41:
		a.NR41 = value
		a.ch[3].length = 64 - uint16(bit.ExtractBits(value, 5, 0))
	case addr.NR42:
		a.NR42 = value
		ch := &a.ch[3]
		pace := bit.ExtractBits(value, 2, 0)
		if pace == 0 {
			ch.envelopeCounter = 8
		} else {
			ch.envelopeCounter = pace
		}
		ch.envelopeLatched = false
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

	if isInWaveRAM {
		offset := address - addr.WaveRAMStart
		if a.waveRAMLocked() {
			// Per Pan Docs: Writes during playback update
			// the currently buffered sample instead of RAM.
			idx := a.ch[2].waveIndex >> 1
			a.waveRAM[idx] = value
			a.ch[2].waveSample = value
		} else {
			a.waveRAM[offset] = value
		}
	}

	a.mapRegistersToState()
}

// handleLengthEnableTransition centralizes the oddities around enabling length
// and triggering channels:
//   - enabling length in the second half of a sequencer period clocks once
//   - triggers reload length from zero before that clock
//   - a trigger after a clocked-to-zero reloads before the forced extra clock
//   - the extra clock also occurs while already enabled when a trigger reloads
//     from zero (the "force" path)
//
// Reference: https://gbdev.io/pandocs/Audio_details.html#obscure-behavior.
func (a *APU) handleLengthEnableTransition(prevEnabled bool, lengthBefore uint16, triggered bool, maxLength uint16, chIdx int) {
	ch := &a.ch[chIdx]
	lengthWasZero := lengthBefore == 0
	clockOnEnable := !prevEnabled && ch.lengthEnable && a.step%2 == 1 && lengthBefore > 0

	if triggered && (lengthWasZero || (clockOnEnable && lengthBefore == 1)) {
		ch.length = maxLength
	}

	if !ch.lengthEnable {
		return
	}

	forceClock := lengthWasZero && triggered && ch.length > 0
	if !forceClock && prevEnabled {
		return
	}

	if a.step%2 == 1 && ch.length > 0 {
		ch.length--
		if ch.length == 0 {
			ch.enabled = false
		}
	}
}

func (a *APU) mapRegistersToState() {
	// NR52 - Master Audio Control
	// 7: Audio on/off | 6-4: Always 1 | 3: CH4 on | 2: CH3 on | 1: CH2 on | 0: CH1 on
	a.enabled = bit.IsSet(7, a.NR52) // audio on/off
	// Bits 3-0 are read-only, ignore writes.

	if !a.enabled {
		// If setting NR52 bit7 to 0, disable all channels,
		// set all registers to 0x00 except NR52
		a.NR10, a.NR11, a.NR12, a.NR13, a.NR14 = 0, 0, 0, 0, 0
		a.NR21, a.NR22, a.NR23, a.NR24 = 0, 0, 0, 0
		a.NR30, a.NR31, a.NR32, a.NR33, a.NR34 = 0, 0, 0, 0, 0
		a.NR41, a.NR42, a.NR43, a.NR44 = 0, 0, 0, 0
		a.NR50, a.NR51 = 0, 0
		for i := range a.ch {
			a.ch[i].enabled = false
		}
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
	prevSweepDown := a.ch[0].sweepDown
	a.ch[0].sweepPeriod = bit.ExtractBits(a.NR10, 6, 4)
	a.ch[0].sweepDown = bit.IsSet(3, a.NR10)
	a.ch[0].sweepStep = bit.ExtractBits(a.NR10, 2, 0)
	if !a.ch[0].sweepDown && prevSweepDown && a.ch[0].sweepNegUsed && (a.ch[0].sweepPeriod > 0 || a.ch[0].sweepStep > 0) {
		// Per Pan Docs: Switching sweep from subtract to add
		// after a subtract calc disables CH1 immediately.
		a.ch[0].enabled = false
	}

	// NR11 - Channel 1 Length Timer & Duty Cycle
	// 7-6: Duty | 5-0: Length Timer (0-63, actual = 64-value)
	a.ch[0].duty = bit.ExtractBits(a.NR11, 7, 6)
	a.ch[0].timer = bit.ExtractBits(a.NR11, 5, 0)

	// NR12 - Channel 1 Volume & Envelope
	// 7-4: Initial Volume | 3: Direction | 2-0: Period
	a.ch[0].volume = bit.ExtractBits(a.NR12, 7, 4)
	a.ch[0].envelopeUp = bit.IsSet(3, a.NR12)
	a.ch[0].envelopePace = bit.ExtractBits(a.NR12, 2, 0)

	// DAC for a channel is enabled if initial volume > 0 or envelope is increasing (i.e. bits 7-3 are not all zero)
	a.ch[0].dacEnabled = (a.ch[0].volume > 0) || a.ch[0].envelopeUp

	// frequency = 131072/(2048-value) Hz
	// NR13 - 7-0: low bits of period for Channel 1
	// NR14 - 2-0: upper 3 bits of period for Channel 1
	a.ch[0].period = bit.Combine(a.NR14&0b111, a.NR13)

	// NR14 - Channel 1 Frequency High & Control
	// 7: Trigger | 6: Length Enable | 5-3: - | 2-0: Upper 3 bits of freq
	prevLenEnable := a.ch[0].lengthEnable
	lengthBefore := a.ch[0].length
	triggered := bit.IsSet(7, a.NR14)
	a.ch[0].lengthEnable = bit.IsSet(6, a.NR14)
	a.ch[0].trigger = triggered
	if a.ch[0].trigger {
		if a.ch[0].dacEnabled {
			a.ch[0].enabled = true
		}
		a.ch[0].envelopeLatched = false
		if a.ch[0].envelopePace == 0 {
			a.ch[0].envelopeCounter = 8
		} else {
			a.ch[0].envelopeCounter = a.ch[0].envelopePace
		}
		a.ch[0].dutyStep = 0
		a.ch[0].freqTimer = a.squarePeriodCycles(&a.ch[0])
		// On trigger, reset sweep timer and shadow frequency
		a.ch[0].sweepEnabled = a.ch[0].sweepPeriod > 0 || a.ch[0].sweepStep > 0
		a.ch[0].sweepTimer = a.ch[0].sweepPeriod
		if a.ch[0].sweepTimer == 0 {
			a.ch[0].sweepTimer = 8
		}
		a.ch[0].shadowFreq = a.ch[0].period
		a.ch[0].sweepNegUsed = false

		// Dummy calculation to immediately disable channel if overflow
		if a.ch[0].sweepStep != 0 {
			if a.ch[0].sweepDown {
				a.ch[0].sweepNegUsed = true
			}
			if _, overflow := a.ch[0].calculateSweepFrequency(); overflow {
				a.ch[0].enabled = false
			}
		}

		// reset the bit, since it's write-only this effectively gets triggered only on a write from 0 to 1
		a.NR14 = bit.Reset(7, a.NR14)
		a.ch[0].trigger = false
	}
	a.handleLengthEnableTransition(prevLenEnable, lengthBefore, triggered, 64, 0)

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

	// DAC for a channel is enabled if initial volume > 0 or envelope is increasing
	a.ch[1].dacEnabled = (a.ch[1].volume > 0) || a.ch[1].envelopeUp

	// frequency = 131072/(2048-value) Hz
	// NR23 - 7-0: low bits of period for Channel 2
	// NR24 - 2-0: upper 3 bits of period for Channel 2
	a.ch[1].period = bit.Combine(a.NR24&0b111, a.NR23)

	// NR24 - Channel 2 Frequency High & Control
	// 7: Trigger | 6: Length Enable | 5-3: - | 2-0: Upper 3 bits of freq
	prevLenEnable = a.ch[1].lengthEnable
	lengthBefore = a.ch[1].length
	triggered = bit.IsSet(7, a.NR24)
	a.ch[1].lengthEnable = bit.IsSet(6, a.NR24)
	a.ch[1].trigger = triggered
	if a.ch[1].trigger {
		if a.ch[1].dacEnabled {
			a.ch[1].enabled = true
		}
		a.ch[1].envelopeLatched = false
		if a.ch[1].envelopePace == 0 {
			a.ch[1].envelopeCounter = 8
		} else {
			a.ch[1].envelopeCounter = a.ch[1].envelopePace
		}
		a.ch[1].dutyStep = 0
		a.ch[1].freqTimer = a.squarePeriodCycles(&a.ch[1])
		// reset the bit, since it's write-only this effectively gets triggered only on a write from 0 to 1
		a.NR24 = bit.Reset(7, a.NR24)
		a.ch[1].trigger = false
	}
	a.handleLengthEnableTransition(prevLenEnable, lengthBefore, triggered, 64, 1)

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
	prevLenEnable = a.ch[2].lengthEnable
	lengthBefore = a.ch[2].length
	triggered = bit.IsSet(7, a.NR34)
	a.ch[2].lengthEnable = bit.IsSet(6, a.NR34)
	a.ch[2].trigger = triggered
	if a.ch[2].trigger {
		if a.ch[2].dacEnabled {
			a.ch[2].enabled = true
		}
		a.ch[2].freqTimer = a.wavePeriodCycles(&a.ch[2])
		a.ch[2].waveIndex = 0
		a.ch[2].waveSample = a.waveRAM[0]
		// reset the bit, since it's write-only this effectively gets triggered only on a write from 0 to 1
		a.NR34 = bit.Reset(7, a.NR34)
		a.ch[2].trigger = false
	}
	a.handleLengthEnableTransition(prevLenEnable, lengthBefore, triggered, 256, 2)

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

	// DAC for a channel is enabled if initial volume > 0 or envelope is increasing
	a.ch[3].dacEnabled = (a.ch[3].volume > 0) || a.ch[3].envelopeUp

	// NR44 - Channel 4 Control
	// 7: Trigger | 6: Length Enable | 5-0: -
	prevLenEnable = a.ch[3].lengthEnable
	lengthBefore = a.ch[3].length
	triggered = bit.IsSet(7, a.NR44)
	a.ch[3].lengthEnable = bit.IsSet(6, a.NR44)
	a.ch[3].trigger = triggered
	if a.ch[3].trigger {
		if a.ch[3].dacEnabled {
			a.ch[3].enabled = true
		}
		a.ch[3].envelopeLatched = false
		if a.ch[3].envelopePace == 0 {
			a.ch[3].envelopeCounter = 8
		} else {
			a.ch[3].envelopeCounter = a.ch[3].envelopePace
		}
		a.ch[3].lfsr = 0x7FFF
		a.ch[3].noiseTimer = a.noisePeriodCycles(&a.ch[3])
		// reset the bit, since it's write-only this effectively gets triggered only on a write from 0 to 1
		a.NR44 = bit.Reset(7, a.NR44)
		a.ch[3].trigger = false
	}
	a.handleLengthEnableTransition(prevLenEnable, lengthBefore, triggered, 64, 3)

	// disable channel immediately if DAC is off
	for i := range a.ch {
		if !a.ch[i].dacEnabled {
			a.ch[i].enabled = false
		}
	}
}

// GetSamples returns interleaved stereo samples.
// Mixing plan:
//  1. Pull per-channel sample buffers produced by Tick (or generate lazily here).
//  2. Scale channel outputs by NR50/NR51 panning/volume and sum into left/right lanes.
//  3. Clamp the mixed values to int16 and write them interleaved into the output slice.
//  4. Expose/reserve any remainder so subsequent calls continue seamlessly.
func (a *APU) GetSamples(count int) []int16 {
	if count <= 0 {
		return nil
	}

	needed := count * 2
	available := len(a.pcmBuffer) - a.pcmCursor
	if available <= 0 {
		return make([]int16, needed)
	}

	out := make([]int16, needed)
	toCopy := min(available, needed)
	copy(out, a.pcmBuffer[a.pcmCursor:a.pcmCursor+toCopy])
	a.pcmCursor += toCopy

	if a.pcmCursor >= len(a.pcmBuffer) {
		a.pcmBuffer = a.pcmBuffer[:0]
		a.pcmCursor = 0
	}

	return out
}

// Debug helpers required by Provider.

// ToggleChannel toggles the mute state of a channel.
func (a *APU) ToggleChannel(idx int) {
	if idx < 0 || idx >= 4 {
		return
	}
	a.ch[idx].muted = !a.ch[idx].muted
}

// SoloChannel sets a channel to solo mode (only that channel is heard).
// Calling with the same channel again disables solo.
func (a *APU) SoloChannel(channel int) {
	if channel < 0 || channel >= 4 {
		return
	}

	// if the channel is already soloed, unmute all channels
	if !a.ch[channel].muted {
		for i := range a.ch {
			a.ch[i].muted = false
		}
	}

	for i := range a.ch {
		if i == channel {
			a.ch[i].muted = false
		} else {
			a.ch[i].muted = true
		}
	}
}

// GetChannelStatus returns the enabled status of each channel.
// This reflects whether the channel is currently producing sound,
// not whether it's muted/soloed for debug purposes.
func (a *APU) GetChannelStatus() (bool, bool, bool, bool) {
	return a.ch[0].enabled, a.ch[1].enabled, a.ch[2].enabled, a.ch[3].enabled
}

// GetChannelVolumes returns actual post-envelope volumes per channel.
// For now returns the initial volumes; will be updated when envelope is implemented.
func (a *APU) GetChannelVolumes() (ch1, ch2, ch3, ch4 uint8) {
	// TODO: return actual post-envelope volumes
	return a.ch[0].volume, a.ch[1].volume, a.ch[2].volume, a.ch[3].volume
}
