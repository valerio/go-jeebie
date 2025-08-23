package audio

type Provider interface {
	// GetSamples retrieves audio samples for playback
	GetSamples(count int) []int16

	// Audio debugging controls

	ToggleChannel(channel int)
	SoloChannel(channel int)
	GetChannelStatus() (ch1, ch2, ch3, ch4 bool)
}

var _ Provider = (*APU)(nil)
