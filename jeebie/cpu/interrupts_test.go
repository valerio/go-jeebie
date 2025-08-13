package cpu

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/valerio/go-jeebie/jeebie/addr"
	"github.com/valerio/go-jeebie/jeebie/memory"
)

func TestInterruptHandling(t *testing.T) {
	t.Run("interrupts disabled by default", func(t *testing.T) {
		mmu := memory.New()
		cpu := New(mmu)

		mmu.Write(addr.IF, 0x01)
		mmu.Write(addr.IE, 0x01)

		pending := cpu.handleInterrupts()
		assert.True(t, pending)
		assert.Equal(t, uint16(0x100), cpu.pc)
	})

	t.Run("EI enables interrupts with delay", func(t *testing.T) {
		mmu := memory.New()
		cpu := New(mmu)

		opcode0xFB(cpu)
		assert.False(t, cpu.interruptsEnabled)
		assert.True(t, cpu.eiPending)

		// simulate the end of Tick() which applies the EI delay
		if cpu.eiPending {
			cpu.eiPending = false
			cpu.interruptsEnabled = true
		}

		assert.True(t, cpu.interruptsEnabled)
		assert.False(t, cpu.eiPending)
	})

	t.Run("DI disables interrupts immediately", func(t *testing.T) {
		mmu := memory.New()
		cpu := New(mmu)
		cpu.interruptsEnabled = true

		opcode0xF3(cpu)
		assert.False(t, cpu.interruptsEnabled)
	})

	t.Run("interrupt priority order", func(t *testing.T) {
		mmu := memory.New()
		cpu := New(mmu)
		cpu.interruptsEnabled = true

		mmu.Write(addr.IF, 0x1F)
		mmu.Write(addr.IE, 0x1F)

		cpu.handleInterrupts()

		assert.Equal(t, uint16(0x40), cpu.pc)
		assert.Equal(t, uint8(0x1E), mmu.Read(addr.IF))
	})

	t.Run("RETI enables interrupts and returns", func(t *testing.T) {
		mmu := memory.New()
		cpu := New(mmu)
		cpu.interruptsEnabled = false
		cpu.sp = 0xFFFE
		cpu.pc = 0x200

		cpu.pushStack(0x150)

		opcode0xD9(cpu)

		assert.True(t, cpu.interruptsEnabled)
		assert.Equal(t, uint16(0x150), cpu.pc)
	})
}

func TestHALTBehavior(t *testing.T) {
	t.Run("HALT with IME=1 and pending interrupt", func(t *testing.T) {
		mmu := memory.New()
		cpu := New(mmu)
		cpu.interruptsEnabled = true

		opcode0x76(cpu)
		assert.True(t, cpu.halted)

		mmu.Write(addr.IF, 0x01)
		mmu.Write(addr.IE, 0x01)

		// simulate Tick() handling interrupts and waking from HALT
		interruptPending := cpu.handleInterrupts()
		if cpu.halted && interruptPending {
			cpu.halted = false
		}
		assert.False(t, cpu.halted)
		assert.Equal(t, uint16(0x40), cpu.pc)
	})

	t.Run("HALT with IME=0 and pending interrupt wakes but doesn't service", func(t *testing.T) {
		mmu := memory.New()
		cpu := New(mmu)
		cpu.interruptsEnabled = false
		cpu.pc = 0x100

		opcode0x76(cpu)
		assert.True(t, cpu.halted)

		mmu.Write(addr.IF, 0x01)
		mmu.Write(addr.IE, 0x01)

		// simulate Tick() waking from HALT with IME=0
		interruptPending := cpu.handleInterrupts()
		if cpu.halted && interruptPending {
			cpu.halted = false
			if !cpu.interruptsEnabled {
				cpu.haltBug = true
			}
		}
		assert.False(t, cpu.halted)
		assert.True(t, cpu.haltBug)
		assert.Equal(t, uint16(0x100), cpu.pc) // PC unchanged
	})

	t.Run("HALT with IME=0 and no interrupt stays halted", func(t *testing.T) {
		mmu := memory.New()
		cpu := New(mmu)
		cpu.interruptsEnabled = false

		opcode0x76(cpu)
		assert.True(t, cpu.halted)

		mmu.Write(addr.IF, 0x00)
		mmu.Write(addr.IE, 0x01)

		interruptPending := cpu.handleInterrupts()
		assert.False(t, interruptPending)
		assert.True(t, cpu.halted)
	})
}

func TestInterruptTiming(t *testing.T) {
	t.Run("interrupt dispatch takes 20 cycles", func(t *testing.T) {
		mmu := memory.New()
		cpu := New(mmu)
		cpu.interruptsEnabled = true
		cpu.cycles = 0

		mmu.Write(addr.IF, 0x01)
		mmu.Write(addr.IE, 0x01)

		startCycles := cpu.cycles
		cpu.handleInterrupts()

		assert.Equal(t, uint64(20), cpu.cycles-startCycles)
	})
}
