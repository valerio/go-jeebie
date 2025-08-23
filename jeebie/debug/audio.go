package debug

import (
	"github.com/valerio/go-jeebie/jeebie/addr"
)

type ChannelStatus struct {
	Enabled   bool
	Frequency float64
	Volume    uint8
	DutyCycle uint8
	Note      string
}

type AudioData struct {
	APUEnabled   bool
	MasterVolume struct {
		Left  uint8
		Right uint8
	}
	Channels struct {
		Ch1 ChannelStatus
		Ch2 ChannelStatus
		Ch3 ChannelStatus
		Ch4 ChannelStatus
	}
	WaveformSamples struct {
		Ch1 []float32
		Ch2 []float32
		Ch3 []float32
		Ch4 []float32
		Mix []float32
	}
	FrameSequencerStep uint8
	SampleRate         int
}

// VolumeProvider interface for getting actual channel volumes
type VolumeProvider interface {
	GetChannelVolumes() (ch1, ch2, ch3, ch4 uint8)
}

func ExtractAudioData(reader MemoryReader, volumeProvider VolumeProvider) *AudioData {
	data := &AudioData{}

	nr52 := reader.Read(addr.NR52)
	data.APUEnabled = (nr52 & 0x80) != 0

	nr50 := reader.Read(addr.NR50)
	data.MasterVolume.Left = (nr50 >> 4) & 0x07
	data.MasterVolume.Right = nr50 & 0x07

	// Extract channel data with actual volumes if available
	if volumeProvider != nil {
		ch1Vol, ch2Vol, ch3Vol, ch4Vol := volumeProvider.GetChannelVolumes()
		extractChannel1(reader, &data.Channels.Ch1, &ch1Vol)
		extractChannel2(reader, &data.Channels.Ch2, &ch2Vol)
		extractChannel3(reader, &data.Channels.Ch3, &ch3Vol)
		extractChannel4(reader, &data.Channels.Ch4, &ch4Vol)
	} else {
		extractChannel1(reader, &data.Channels.Ch1, nil)
		extractChannel2(reader, &data.Channels.Ch2, nil)
		extractChannel3(reader, &data.Channels.Ch3, nil)
		extractChannel4(reader, &data.Channels.Ch4, nil)
	}

	data.SampleRate = 44100

	return data
}

func extractChannel1(reader MemoryReader, ch *ChannelStatus, actualVolume *uint8) {
	nr52 := reader.Read(addr.NR52)
	ch.Enabled = (nr52 & 0x01) != 0

	nr14 := reader.Read(addr.NR14)
	nr13 := reader.Read(addr.NR13)
	freqReg := uint16(nr14&0x07)<<8 | uint16(nr13)
	if freqReg > 0 {
		ch.Frequency = 131072.0 / float64(2048-freqReg)
	}

	if actualVolume != nil {
		ch.Volume = *actualVolume
	} else {
		nr12 := reader.Read(addr.NR12)
		ch.Volume = (nr12 >> 4) & 0x0F
	}

	nr11 := reader.Read(addr.NR11)
	ch.DutyCycle = (nr11 >> 6) & 0x03

	ch.Note = frequencyToNote(ch.Frequency)
}

func extractChannel2(reader MemoryReader, ch *ChannelStatus, actualVolume *uint8) {
	nr52 := reader.Read(addr.NR52)
	ch.Enabled = (nr52 & 0x02) != 0

	nr24 := reader.Read(addr.NR24)
	nr23 := reader.Read(addr.NR23)
	freqReg := uint16(nr24&0x07)<<8 | uint16(nr23)
	if freqReg > 0 {
		ch.Frequency = 131072.0 / float64(2048-freqReg)
	}

	if actualVolume != nil {
		ch.Volume = *actualVolume
	} else {
		nr22 := reader.Read(addr.NR22)
		ch.Volume = (nr22 >> 4) & 0x0F
	}

	nr21 := reader.Read(addr.NR21)
	ch.DutyCycle = (nr21 >> 6) & 0x03

	ch.Note = frequencyToNote(ch.Frequency)
}

func extractChannel3(reader MemoryReader, ch *ChannelStatus, actualVolume *uint8) {
	nr52 := reader.Read(addr.NR52)
	ch.Enabled = (nr52 & 0x04) != 0

	nr34 := reader.Read(addr.NR34)
	nr33 := reader.Read(addr.NR33)
	freqReg := uint16(nr34&0x07)<<8 | uint16(nr33)
	if freqReg > 0 {
		ch.Frequency = 65536.0 / float64(2048-freqReg)
	}

	nr32 := reader.Read(addr.NR32)
	volumeCode := (nr32 >> 5) & 0x03
	switch volumeCode {
	case 0:
		ch.Volume = 0
	case 1:
		ch.Volume = 15
	case 2:
		ch.Volume = 7
	case 3:
		ch.Volume = 3
	}

	ch.Note = frequencyToNote(ch.Frequency)
}

func extractChannel4(reader MemoryReader, ch *ChannelStatus, actualVolume *uint8) {
	nr52 := reader.Read(addr.NR52)
	ch.Enabled = (nr52 & 0x08) != 0

	if actualVolume != nil {
		ch.Volume = *actualVolume
	} else {
		nr42 := reader.Read(addr.NR42)
		ch.Volume = (nr42 >> 4) & 0x0F
	}

	nr43 := reader.Read(addr.NR43)
	shift := (nr43 >> 4) & 0x0F
	divisorCode := nr43 & 0x07
	var divisor float64
	if divisorCode == 0 {
		divisor = 0.5
	} else {
		divisor = float64(divisorCode)
	}
	ch.Frequency = 524288.0 / divisor / float64(uint(1)<<uint(shift+1))

	ch.Note = "Noise"
}

func frequencyToNote(freq float64) string {
	if freq < 20 || freq > 20000 {
		return "--"
	}

	notes := []string{"C", "C#", "D", "D#", "E", "F", "F#", "G", "G#", "A", "A#", "B"}
	a4 := 440.0

	halfSteps := 12.0 * (logBase2(freq / a4))
	noteIndex := int(halfSteps+69.5) % 12
	octave := (int(halfSteps+69.5) - noteIndex) / 12

	if noteIndex < 0 {
		noteIndex += 12
		octave--
	}

	if octave < 0 || octave > 9 {
		return "--"
	}

	return notes[noteIndex] + string('0'+octave)
}

func logBase2(x float64) float64 {
	if x <= 0 {
		return 0
	}
	result := 0.0
	for x >= 2 {
		x /= 2
		result++
	}
	for x < 1 {
		x *= 2
		result--
	}

	for i := 0; i < 5; i++ {
		x = x * x
		if x >= 2 {
			x /= 2
			result += 1.0 / float64(uint(1)<<uint(i+1))
		}
	}
	return result
}

func GenerateWaveformSamples(channelData []float32, dutyCycle uint8, frequency float64, volume uint8, enabled bool, sampleCount int) {
	if !enabled || volume == 0 || frequency == 0 {
		for i := range channelData {
			channelData[i] = 0
		}
		return
	}

	dutyTable := []float64{0.125, 0.25, 0.5, 0.75}
	duty := dutyTable[dutyCycle&0x03]

	samplesPerPeriod := 44100.0 / frequency
	normalizedVolume := float32(volume) / 15.0

	for i := 0; i < sampleCount; i++ {
		phase := float64(i) / samplesPerPeriod
		phase = phase - float64(int(phase))

		if phase < duty {
			channelData[i] = normalizedVolume
		} else {
			channelData[i] = -normalizedVolume
		}
	}
}
