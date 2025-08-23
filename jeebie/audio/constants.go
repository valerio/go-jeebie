package audio

// Timing constants
// Reference: https://gbdev.io/pandocs/Audio_details.html
const (
	// frameSequencerCycles is the number of CPU cycles per frame sequencer tick
	// The frame sequencer runs at 512 Hz: 4194304 Hz / 512 Hz = 8192 cycles
	frameSequencerCycles = 8192

	// sampleCycles is the number of CPU cycles per audio sample
	// Target sample rate ~44100 Hz: 4194304 Hz / 44100 Hz ≈ 95 cycles
	sampleCycles = 95

	// cpuFrequency is the Game Boy CPU clock frequency in Hz
	cpuFrequency = 4194304
)

// Fixed-point constants for precise frequency calculation
const (
	// Fixed-point shift for 16.16 fixed point arithmetic
	fpShift = 16

	// Fixed-point increments per sample for accurate pitch
	// Pulse channels: 131072 Hz / 44100 Hz = 2.972789...
	pulseIncrement = 194783 // uint32(131072.0 * 65536.0 / 44100.0)

	// Wave channel: 65536 Hz / 44100 Hz = 1.486394...
	waveIncrement = 97391 // uint32(65536.0 * 65536.0 / 44100.0)
)

// Channel constants
const (
	// maxVolume is the maximum volume level for envelope (4 bits)
	maxVolume = 15

	// maxSweepPeriod is the maximum sweep period value (3 bits)
	maxSweepPeriod = 7

	// waveRAMSize is the size of wave pattern RAM in nibbles (32 x 4-bit samples)
	waveRAMSize = 16 // 16 bytes = 32 nibbles

	// waveRAMRegisterOffset is the offset in the register array where wave RAM starts
	// Wave RAM is at 0xFF30-0xFF3F, and registers start at 0xFF10, so offset is 0x20
	waveRAMRegisterOffset = 0x20

	// lfsrInitialValue is the initial value for the noise channel LFSR
	// All bits set to 1 for maximum length sequence
	lfsrInitialValue = 0x7FFF

	// frequencyToTimerOffset is added to frequency to get the timer period
	// Timer = 2048 - Frequency for channels 1, 2, and 3
	frequencyToTimerOffset = 2048

	// sampleAmplitude is the multiplier for converting volume to sample amplitude
	// This gives us a reasonable 16-bit sample range
	sampleAmplitude = 1024

	// waveTableSize is the number of samples in the wave pattern
	waveTableSize = 32

	// noiseChannelPeriod is the base period for noise channel updates
	noiseChannelPeriod = 64

	// dutyPhases is the number of phases in a duty cycle
	dutyPhases = 8

	// waveOutputBias is subtracted from wave samples to center them
	waveOutputBias = 16384

	// waveOutputScale is the multiplier for wave channel samples
	waveOutputScale = 2048

	// Buffer size constants
	initialBufferCapacity = 1024
	maxBufferSize         = 8192
	bufferRetainSize      = 4096

	// Sample limits for 16-bit audio
	maxSampleValue = 32767
	minSampleValue = -32768
)

// Duty cycle patterns for pulse channels
// Each bit represents high (1) or low (0) output over 8 phases
// Reference: https://gbdev.io/pandocs/Audio_Registers.html#ff11--nr11-channel-1-length-timer--duty-cycle
var dutyPatterns = [4]uint8{
	0b00000001, // 12.5% duty: -------+
	0b10000001, // 25% duty:   +------+
	0b10000111, // 50% duty:   +----+++
	0b01111110, // 75% duty:   -++++++−
}

// Volume shift amounts for wave channel
// Maps the 2-bit volume code to right-shift amount
// 0: 100% (shift 0), 1: 50% (shift 1), 2: 25% (shift 2), 3: mute (shift 4)
var waveVolumeShift = [4]uint8{4, 0, 1, 2}

// Register bit masks
const (
	// NR52 (0xFF26) bit masks
	nr52PowerMask     = 0x80 // Bit 7: All sound on/off
	nr52Ch1StatusMask = 0x01 // Bit 0: Channel 1 on
	nr52Ch2StatusMask = 0x02 // Bit 1: Channel 2 on
	nr52Ch3StatusMask = 0x04 // Bit 2: Channel 3 on
	nr52Ch4StatusMask = 0x08 // Bit 3: Channel 4 on
	nr52UnusedMask    = 0x70 // Bits 4-6: Always read as 1

	// Trigger bit (bit 7) for NR14, NR24, NR34, NR44
	triggerMask = 0x80

	// Envelope direction bit
	envelopeIncreaseMask = 0x08

	// Wave channel DAC enable bit
	waveDACMask = 0x80

	// Noise channel width mode bit
	noiseWidthMask = 0x08
)
