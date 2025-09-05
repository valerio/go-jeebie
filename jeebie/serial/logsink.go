package serial

import (
	"log/slog"

	"github.com/valerio/go-jeebie/jeebie/addr"
	"github.com/valerio/go-jeebie/jeebie/bit"
)

// LogSink implements a dummy serial device that just logs outgoing bytes as text.
// Handy for debugging test roms that output to serial.
type LogSink struct {
	irqHandler     func()
	sb, sc         byte
	transferActive bool
	countdown      int
	logger         *slog.Logger

	// settings
	immediate bool
	defaultRX byte // returned value on SB when no transfer is active

	// Optional line buffer for readable output
	line []byte
}

type LogSinkOption func(*LogSink)

// WithFixedTiming sets the sink to complete transfers after a fixed countdown
// (~4096 CPU cycles per byte on DMG) instead of immediately.
func WithFixedTiming() LogSinkOption { return func(s *LogSink) { s.immediate = false } }

// NewLogSink creates a new logging serial device.
// The passed function is called when a transfer is completed, should be wired
// to request the Serial interrupt.
func NewLogSink(irq func(), opts ...LogSinkOption) *LogSink {
	s := &LogSink{
		irqHandler: irq,
		immediate:  true,
		defaultRX:  0xFF,
		logger:     slog.Default(),
	}
	for _, opt := range opts {
		opt(s)
	}
	s.Reset()
	return s
}

func (s *LogSink) Write(address uint16, value byte) {
	switch address {
	case addr.SB:
		s.sb = value
	case addr.SC:
		s.sc = value
		s.maybeStartTransfer()
	default:
		panic("serial.LogSink: invalid write address")
	}
}

func (s *LogSink) Read(address uint16) byte {
	switch address {
	case addr.SB:
		return s.sb
	case addr.SC:
		return s.sc
	default:
		panic("serial.LogSink: invalid read address")
	}
}

func (s *LogSink) Tick(cycles int) {
	if s.immediate || !s.transferActive {
		return
	}
	s.countdown -= cycles
	if s.countdown <= 0 {
		s.completeTransfer()
		s.countdown = 0
	}
}

func (s *LogSink) Reset() {
	s.sb = 0x00
	s.sc = 0x00
	s.transferActive = false
	s.countdown = 0
	s.line = s.line[:0]
}

func (s *LogSink) maybeStartTransfer() {
	if s.transferActive {
		return
	}
	// a transfer should start when bit 7 (start) and bit 0 (clock source) of SC are set.
	if !bit.IsSet(7, s.sc) || !bit.IsSet(0, s.sc) {
		return
	}

	// log the outgoing byte as text; buffer until newline for readability
	b := s.sb
	if b == 0 || b == '\n' || b == '\r' {
		if len(s.line) > 0 {
			s.logger.Info("serial", "line", string(s.line))
			s.line = s.line[:0]
		}
	} else {
		s.line = append(s.line, b)
	}

	if s.immediate {
		s.completeTransfer()
		return
	}

	// fixed timing: DMG ~4096 CPU cycles per byte
	s.transferActive = true
	s.countdown = 4096
}

func (s *LogSink) completeTransfer() {
	s.sb = s.defaultRX
	// Clear start bit (bit7) to indicate completion
	s.sc = bit.Clear(7, s.sc)
	s.transferActive = false
	if s.irqHandler != nil {
		s.irqHandler()
	}
}
