package cpu

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/valerio/go-jeebie/jeebie/memory"
)

func TestCPU_stack(t *testing.T) {
	mmu := memory.New()
	cpu := New(mmu)

	cpu.sp = 0xFFFF
	cpu.pushStack(0x0102)

	assert.Equal(t, uint16(0xFFFD), cpu.sp)

	popped := cpu.popStack()

	assert.Equal(t, uint16(0x0102), popped)
	assert.Equal(t, uint16(0xFFFF), cpu.sp)
}

func TestCPU_inc(t *testing.T) {
	mmu := memory.New()
	cpu := New(mmu)

	testCases := []struct {
		desc  string
		reg   *uint8
		arg   uint8
		want  uint8
		flags Flag
	}{
		{desc: "increases", reg: &cpu.a, arg: 0x0A, want: 0x0B},
		{desc: "sets zero flag", reg: &cpu.a, arg: 0xFF, want: 0, flags: zeroFlag | halfCarryFlag},
		{desc: "sets half carry flag", reg: &cpu.a, arg: 0x0F, want: 0x10, flags: halfCarryFlag},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			cpu.f = 0
			*tC.reg = tC.arg
			cpu.inc(tC.reg)
			assert.Equal(t, tC.want, *tC.reg)
			assert.Equal(t, uint8(tC.flags), cpu.f)
		})
	}
}

func TestCPU_dec(t *testing.T) {
	mmu := memory.New()
	cpu := New(mmu)

	testCases := []struct {
		desc  string
		reg   *uint8
		arg   uint8
		want  uint8
		flags Flag
	}{
		{desc: "decreases", reg: &cpu.a, arg: 0x0A, want: 0x09, flags: subFlag},
		{desc: "sets half carry flags", reg: &cpu.a, arg: 0, want: 0xFF, flags: subFlag | halfCarryFlag},
		{desc: "sets zero flag", reg: &cpu.a, arg: 0x01, want: 0, flags: subFlag | zeroFlag},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			cpu.f = 0
			*tC.reg = tC.arg
			cpu.dec(tC.reg)
			assert.Equal(t, tC.want, *tC.reg)
			assert.Equal(t, uint8(tC.flags), cpu.f)
		})
	}
}

func TestCPU_rlc(t *testing.T) {
	mmu := memory.New()
	cpu := New(mmu)

	testCases := []struct {
		desc  string
		reg   *uint8
		arg   uint8
		want  uint8
		flags Flag
	}{
		{desc: "rotates left", reg: &cpu.a, arg: 0x01, want: 0x02},
		{desc: "sets carry flag", reg: &cpu.a, arg: 0x80, want: 0x01, flags: carryFlag},
		{desc: "sets zero flag", reg: &cpu.b, arg: 0, want: 0, flags: zeroFlag},
		{desc: "sets zero flag for register A", reg: &cpu.a, arg: 0, want: 0, flags: zeroFlag},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			cpu.f = 0
			*tC.reg = tC.arg
			cpu.rlc(tC.reg)
			assert.Equal(t, tC.want, *tC.reg)
			assert.Equalf(t, uint8(tC.flags), cpu.f, "flags don't match")
		})
	}
}

func TestCPU_rl(t *testing.T) {
	mmu := memory.New()
	cpu := New(mmu)

	testCases := []struct {
		desc         string
		reg          *uint8
		arg          uint8
		want         uint8
		initialFlags Flag
		flags        Flag
	}{
		{desc: "rotates left", reg: &cpu.a, arg: 0x01, want: 0x02},
		{desc: "adds carry bit", reg: &cpu.a, arg: 0x01, want: 0x03, initialFlags: carryFlag},
		{desc: "sets carry flag", reg: &cpu.a, arg: 0x80, want: 0, flags: carryFlag | zeroFlag},
		{desc: "sets zero flag", reg: &cpu.b, arg: 0, want: 0, flags: zeroFlag},
		{desc: "sets zero flag for register A", reg: &cpu.a, arg: 0, want: 0, flags: zeroFlag},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			cpu.f = uint8(tC.initialFlags)
			*tC.reg = tC.arg
			cpu.rl(tC.reg)
			assert.Equal(t, tC.want, *tC.reg)
			assert.Equalf(t, uint8(tC.flags), cpu.f, "flags don't match")
		})
	}
}

func TestCPU_rrc(t *testing.T) {
	mmu := memory.New()
	cpu := New(mmu)

	testCases := []struct {
		desc  string
		reg   *uint8
		arg   uint8
		want  uint8
		flags Flag
	}{
		{desc: "rotates right", reg: &cpu.a, arg: 0x02, want: 0x01},
		{desc: "sets carry flag", reg: &cpu.a, arg: 0x01, want: 0x80, flags: carryFlag},
		{desc: "sets zero flag", reg: &cpu.b, arg: 0, want: 0, flags: zeroFlag},
		{desc: "sets zero flag for register A", reg: &cpu.a, arg: 0, want: 0, flags: zeroFlag},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			cpu.f = 0
			*tC.reg = tC.arg
			cpu.rrc(tC.reg)
			assert.Equal(t, tC.want, *tC.reg)
			assert.Equalf(t, uint8(tC.flags), cpu.f, "flags don't match")
		})
	}
}

func TestCPU_rr(t *testing.T) {
	mmu := memory.New()
	cpu := New(mmu)

	testCases := []struct {
		desc         string
		reg          *uint8
		arg          uint8
		want         uint8
		initialFlags Flag
		flags        Flag
	}{
		{desc: "rotates right", reg: &cpu.a, arg: 0x02, want: 0x01},
		{desc: "adds carry bit", reg: &cpu.a, arg: 0x02, want: 0x81, initialFlags: carryFlag},
		{desc: "sets carry flag", reg: &cpu.a, arg: 1, want: 0, flags: carryFlag | zeroFlag},
		{desc: "sets zero flag", reg: &cpu.b, arg: 0, want: 0, flags: zeroFlag},
		{desc: "sets zero flag for register A", reg: &cpu.a, arg: 0, want: 0, flags: zeroFlag},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			cpu.f = uint8(tC.initialFlags)
			*tC.reg = tC.arg
			cpu.rr(tC.reg)
			assert.Equal(t, tC.want, *tC.reg)
			assert.Equalf(t, uint8(tC.flags), cpu.f, "flags don't match")
		})
	}
}

func TestCPU_sla(t *testing.T) {
	mmu := memory.New()
	cpu := New(mmu)

	testCases := []struct {
		desc  string
		reg   *uint8
		arg   uint8
		want  uint8
		flags Flag
	}{
		{desc: "shifts left", reg: &cpu.a, arg: 0x01, want: 0x02},
		{desc: "sets flags", reg: &cpu.a, arg: 0x80, want: 0, flags: carryFlag | zeroFlag},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			cpu.f = 0
			*tC.reg = tC.arg
			cpu.sla(tC.reg)
			assert.Equal(t, tC.want, *tC.reg)
			assert.Equalf(t, uint8(tC.flags), cpu.f, "flags don't match")
		})
	}
}

func TestCPU_sra(t *testing.T) {
	mmu := memory.New()
	cpu := New(mmu)

	testCases := []struct {
		desc  string
		reg   *uint8
		arg   uint8
		want  uint8
		flags Flag
	}{
		{desc: "shifts right", reg: &cpu.a, arg: 0x22, want: 0x11},
		{desc: "preserves the MSb", reg: &cpu.a, arg: 0x82, want: 0xc1},
		{desc: "sets flags", reg: &cpu.a, arg: 1, want: 0, flags: carryFlag | zeroFlag},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			cpu.f = 0
			*tC.reg = tC.arg
			cpu.sra(tC.reg)
			assert.Equal(t, tC.want, *tC.reg)
			assert.Equalf(t, uint8(tC.flags), cpu.f, "flags don't match")
		})
	}
}

func TestCPU_srl(t *testing.T) {
	mmu := memory.New()
	cpu := New(mmu)

	testCases := []struct {
		desc  string
		reg   *uint8
		arg   uint8
		want  uint8
		flags Flag
	}{
		{desc: "shifts right", reg: &cpu.a, arg: 0x88, want: 0x44},
		{desc: "sets flags", reg: &cpu.a, arg: 1, want: 0, flags: carryFlag | zeroFlag},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			cpu.f = 0
			*tC.reg = tC.arg
			cpu.srl(tC.reg)
			assert.Equal(t, tC.want, *tC.reg)
			assert.Equalf(t, uint8(tC.flags), cpu.f, "flags don't match")
		})
	}
}

func TestCPU_addToA(t *testing.T) {
	mmu := memory.New()
	cpu := New(mmu)

	testCases := []struct {
		desc  string
		a     uint8
		arg   uint8
		want  uint8
		flags Flag
	}{
		{desc: "adds to register A", a: 0, arg: 0x0F, want: 0x0F},
		{desc: "sets half carry", a: 0x0F, arg: 0x0F, want: 0x1E, flags: halfCarryFlag},
		{desc: "sets carry", a: 0xFF, arg: 0x02, want: 1, flags: carryFlag | halfCarryFlag},
		{desc: "sets zero", a: 0xFF, arg: 0x01, want: 0, flags: zeroFlag | carryFlag | halfCarryFlag},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			cpu.f = 0
			cpu.a = tC.a
			cpu.addToA(tC.arg)
			assert.Equal(t, tC.want, cpu.a)
			assert.Equalf(t, uint8(tC.flags), cpu.f, "flags don't match")
		})
	}
}

func TestCPU_adc(t *testing.T) {
	mmu := memory.New()
	cpu := New(mmu)

	testCases := []struct {
		desc  string
		carry bool
		a     uint8
		arg   uint8
		want  uint8
		flags Flag
	}{
		{desc: "adds to register A", a: 0, arg: 0x02, want: 0x02},
		{desc: "adds the carry flag", carry: true, a: 0, arg: 0x02, want: 0x03},
		{desc: "sets half carry", a: 0x0F, arg: 0x0F, want: 0x1E, flags: halfCarryFlag},
		{desc: "sets carry", a: 0xFF, arg: 0x02, want: 1, flags: carryFlag | halfCarryFlag},
		{desc: "sets zero", a: 0xFF, arg: 0x01, want: 0, flags: zeroFlag | carryFlag | halfCarryFlag},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			cpu.f = 0
			if tC.carry {
				cpu.setFlag(carryFlag)
			}
			cpu.a = tC.a
			cpu.adc(tC.arg)
			assert.Equal(t, tC.want, cpu.a)
			assert.Equalf(t, uint8(tC.flags), cpu.f, "flags don't match")
		})
	}
}

func TestCPU_addToHL(t *testing.T) {
	mmu := memory.New()
	cpu := New(mmu)

	testCases := []struct {
		desc  string
		hl    uint16
		arg   uint16
		want  uint16
		flags Flag
	}{
		{desc: "adds to HL", hl: 0, arg: 0x0F, want: 0x0F},
		{desc: "sets half carry if bit 11 carries", hl: 0xFFF, arg: 0x01, want: 0x1000, flags: halfCarryFlag},
		{desc: "sets carry", hl: 0xFFFF, arg: 0x02, want: 1, flags: carryFlag | halfCarryFlag},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			cpu.f = 0
			cpu.setHL(tC.hl)
			cpu.addToHL(tC.arg)
			assert.Equal(t, tC.want, cpu.getHL())
			assert.Equalf(t, uint8(tC.flags), cpu.f, "flags don't match")
		})
	}
}

func TestCPU_sub(t *testing.T) {
	mmu := memory.New()
	cpu := New(mmu)

	testCases := []struct {
		desc  string
		a     uint8
		arg   uint8
		want  uint8
		flags Flag
	}{
		{desc: "subtracts from A", a: 0x3, arg: 0x01, want: 0x02, flags: subFlag},
		{desc: "sets carry", a: 0, arg: 0x01, want: 0xFF, flags: subFlag | carryFlag | halfCarryFlag},
		{desc: "sets halfcarry", a: 0x10, arg: 0x01, want: 0x0F, flags: subFlag | halfCarryFlag},
		{desc: "sets zero", a: 0x1, arg: 0x01, want: 0, flags: subFlag | zeroFlag},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			cpu.f = 0
			cpu.a = tC.a
			cpu.sub(tC.arg)
			assert.Equal(t, tC.want, cpu.a)
			assert.Equalf(t, uint8(tC.flags), cpu.f, "flags don't match")
		})
	}
}

func TestCPU_sbc(t *testing.T) {
	mmu := memory.New()
	cpu := New(mmu)

	testCases := []struct {
		desc  string
		carry bool
		a     uint8
		arg   uint8
		want  uint8
		flags Flag
	}{
		{desc: "subtracts from A", a: 0x3, arg: 0x01, want: 0x02, flags: subFlag},
		{desc: "uses carry value", carry: true, a: 0x3, arg: 0x01, want: 0x01, flags: subFlag},
		{desc: "sets carry", a: 0, arg: 0x01, want: 0xFF, flags: subFlag | carryFlag | halfCarryFlag},
		{desc: "sets halfcarry", a: 0x10, arg: 0x01, want: 0x0F, flags: subFlag | halfCarryFlag},
		{desc: "sets zero", a: 0x1, arg: 0x01, want: 0, flags: subFlag | zeroFlag},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			cpu.f = 0
			if tC.carry {
				cpu.setFlag(carryFlag)
			}
			cpu.a = tC.a
			cpu.sbc(tC.arg)
			assert.Equal(t, tC.want, cpu.a)
			assert.Equalf(t, uint8(tC.flags), cpu.f, "flags don't match")
		})
	}
}

func TestCPU_and(t *testing.T) {
	mmu := memory.New()
	cpu := New(mmu)

	testCases := []struct {
		desc  string
		a     uint8
		arg   uint8
		want  uint8
		flags Flag
	}{
		{desc: "does bitwise and with A", a: 0x0F, arg: 0x44, want: 0x04, flags: halfCarryFlag},
		{desc: "sets zero flag", a: 0x0F, arg: 0x40, want: 0, flags: zeroFlag | halfCarryFlag},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			cpu.f = 0
			cpu.a = tC.a
			cpu.and(tC.arg)
			assert.Equal(t, tC.want, cpu.a)
			assert.Equalf(t, uint8(tC.flags), cpu.f, "flags don't match")
		})
	}
}

func TestCPU_or(t *testing.T) {
	mmu := memory.New()
	cpu := New(mmu)

	testCases := []struct {
		desc  string
		a     uint8
		arg   uint8
		want  uint8
		flags Flag
	}{
		{desc: "does bitwise or with A", a: 0x40, arg: 0x04, want: 0x44},
		{desc: "sets zero flag", a: 0, arg: 0, want: 0, flags: zeroFlag},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			cpu.f = 0
			cpu.a = tC.a
			cpu.or(tC.arg)
			assert.Equal(t, tC.want, cpu.a)
			assert.Equalf(t, uint8(tC.flags), cpu.f, "flags don't match")
		})
	}
}

func TestCPU_xor(t *testing.T) {
	mmu := memory.New()
	cpu := New(mmu)

	testCases := []struct {
		desc  string
		a     uint8
		arg   uint8
		want  uint8
		flags Flag
	}{
		{desc: "does bitwise xor with A", a: 0x0F, arg: 0x03, want: 0x0c},
		{desc: "sets zero flag", a: 0xFF, arg: 0xFF, want: 0, flags: zeroFlag},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			cpu.f = 0
			cpu.a = tC.a
			cpu.xor(tC.arg)
			assert.Equal(t, tC.want, cpu.a)
			assert.Equalf(t, uint8(tC.flags), cpu.f, "flags don't match")
		})
	}
}

func TestCPU_cp(t *testing.T) {
	mmu := memory.New()
	cpu := New(mmu)

	testCases := []struct {
		desc string
		a    uint8
		arg  uint8

		flags Flag
	}{
		{desc: "sets zero flag (a == n)", a: 0x0F, arg: 0x0F, flags: subFlag | zeroFlag},
		{desc: "sets carry flag (a < n)", a: 0x00, arg: 0x01, flags: subFlag | halfCarryFlag | carryFlag},
		{desc: "sets half carry flag", a: 0x10, arg: 0x01, flags: subFlag | halfCarryFlag},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			cpu.f = 0
			cpu.a = tC.a
			cpu.cp(tC.arg)
			assert.Equalf(t, uint8(tC.flags), cpu.f, "flags don't match")
		})
	}
}

func TestCPU_swap(t *testing.T) {
	mmu := memory.New()
	cpu := New(mmu)

	testCases := []struct {
		desc  string
		reg   *uint8
		arg   uint8
		want  uint8
		flags Flag
	}{
		{desc: "swaps the given register", reg: &cpu.c, arg: 0xAB, want: 0xBA},
		{desc: "sets zero", reg: &cpu.b, arg: 0, want: 0, flags: zeroFlag},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			cpu.f = 0
			*tC.reg = tC.arg
			cpu.swap(tC.reg)
			assert.Equal(t, tC.want, *tC.reg)
			assert.Equalf(t, uint8(tC.flags), cpu.f, "flags don't match")
		})
	}
}

func TestCPU_daa(t *testing.T) {
	mmu := memory.New()
	cpu := New(mmu)

	testCases := []struct {
		desc         string
		initialFlags Flag
		a            uint8
		want         uint8
		flags        Flag
	}{
		{desc: "sets zero flag", a: 0, want: 0, flags: zeroFlag},
		{desc: "(add) adds 0x06", a: 0x7d, want: 0x83},
		{desc: "(add) adds 0x60", a: 0xa1, want: 0x01, flags: carryFlag},
		{desc: "(add) adds 0x66", a: 0xaa, want: 0x10, flags: carryFlag},
		{desc: "(sub+half) removes 0x06", initialFlags: subFlag | halfCarryFlag, a: 0x83, want: 0x7d, flags: subFlag},
		{desc: "(sub+carry) removes 0x60", initialFlags: subFlag | carryFlag, a: 0xa1, want: 0x41, flags: subFlag | carryFlag},
		{desc: "(sub+carry+half) removes 0x66", initialFlags: subFlag | carryFlag | halfCarryFlag, a: 0x10, want: 0xaa, flags: subFlag | carryFlag},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			cpu.f = uint8(tC.initialFlags)
			cpu.a = tC.a
			cpu.daa()
			assert.Equal(t, tC.want, cpu.a)
			assert.Equalf(t, uint8(tC.flags), cpu.f, "flags don't match")
		})
	}
}

func TestCPU_bit(t *testing.T) {
	mmu := memory.New()
	cpu := New(mmu)

	testCases := []struct {
		desc    string
		initial Flag
		idx     uint8
		arg     uint8
		flags   Flag
	}{
		{desc: "sets zero flag", idx: 0, arg: 0xF0, flags: zeroFlag | halfCarryFlag},
		{desc: "resets zero flag", initial: zeroFlag, idx: 7, arg: 0x80, flags: halfCarryFlag},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			cpu.f = uint8(tC.initial)
			cpu.bit(tC.idx, tC.arg)
			assert.Equalf(t, uint8(tC.flags), cpu.f, "flags don't match")
		})
	}
}

func TestCPU_set(t *testing.T) {
	mmu := memory.New()
	cpu := New(mmu)

	testCases := []struct {
		desc string
		reg  *uint8
		idx  uint8
		arg  uint8
		want uint8
	}{
		{desc: "sets bit 0", reg: &cpu.a, idx: 0, arg: 0xf0, want: 0xf1},
		{desc: "sets bit 3", reg: &cpu.c, idx: 3, arg: 0xaa, want: 0xaa},
		{desc: "sets bit 4", reg: &cpu.c, idx: 4, arg: 0xaa, want: 0xba},
		{desc: "sets bit 7", reg: &cpu.b, idx: 7, arg: 0, want: 0x80},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			cpu.f = 0
			*tC.reg = tC.arg
			cpu.set(tC.idx, tC.reg)
			assert.Equal(t, tC.want, *tC.reg)
		})
	}
}

func TestCPU_res(t *testing.T) {
	mmu := memory.New()
	cpu := New(mmu)

	testCases := []struct {
		desc string
		reg  *uint8
		idx  uint8
		arg  uint8
		want uint8
	}{
		{desc: "resets bit 0", reg: &cpu.a, idx: 0, arg: 0xf0, want: 0xf0},
		{desc: "resets bit 3", reg: &cpu.c, idx: 3, arg: 0xaa, want: 0xa2},
		{desc: "resets bit 4", reg: &cpu.c, idx: 4, arg: 0xba, want: 0xaa},
		{desc: "resets bit 7", reg: &cpu.b, idx: 7, arg: 0x80, want: 0},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			cpu.f = 0
			*tC.reg = tC.arg
			cpu.res(tC.idx, tC.reg)
			assert.Equal(t, tC.want, *tC.reg)
		})
	}
}

func TestCPU_jr(t *testing.T) {
	mmu := memory.New()
	cpu := New(mmu)

	testCases := []struct {
		desc string
		n    uint8
		pc   uint16
		want uint16
	}{
		{desc: "jumps back", n: 0xFE, pc: 0xC000, want: 0xC000 - 2 + 1},
		{desc: "jumps back 16", n: 0xF0, pc: 0xC000, want: 0xC000 - 16 + 1},
		{desc: "jumps forward", n: 0x10, pc: 0xC000, want: 0xC000 + 16 + 1},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			cpu.pc = tC.pc
			mmu.Write(cpu.pc, tC.n)
			cpu.jr()
			assert.Equal(t, tC.want, cpu.pc)
		})
	}
}

func TestCallRetInstructions(t *testing.T) {
	t.Run("CALL pushes return address and jumps", func(t *testing.T) {
		mmu := memory.New()
		cpu := New(mmu)

		cpu.pc = 0xC000
		cpu.sp = 0xFFFE

		// Setup CALL nn instruction: CD 34 12 (CALL 0x1234)
		mmu.Write(0xC000, 0xCD) // opcode
		mmu.Write(0xC001, 0x34) // low byte
		mmu.Write(0xC002, 0x12) // high byte

		cpu.pc++
		cycles := opcode0xCD(cpu)

		// Check that PC jumped to destination
		assert.Equal(t, uint16(0x1234), cpu.pc)

		// Check that return address (0xC003) was pushed to stack
		assert.Equal(t, uint16(0xFFFC), cpu.sp)
		assert.Equal(t, uint8(0x03), mmu.Read(0xFFFC)) // low byte of return address
		assert.Equal(t, uint8(0xC0), mmu.Read(0xFFFD)) // high byte of return address

		// Check cycle count
		assert.Equal(t, 24, cycles)
	})

	t.Run("RET pops address and returns", func(t *testing.T) {
		mmu := memory.New()
		cpu := New(mmu)

		// Setup stack with return address 0x2000
		cpu.sp = 0xFFFC
		mmu.Write(0xFFFC, 0x00) // low byte
		mmu.Write(0xFFFD, 0x20) // high byte

		cpu.pc = 0x1500 // Current PC (irrelevant for RET)

		cpu.pc++
		cycles := opcode0xC9(cpu)

		// Check that PC was set to popped value
		assert.Equal(t, uint16(0x2000), cpu.pc)

		// Check that SP was restored
		assert.Equal(t, uint16(0xFFFE), cpu.sp)

		// Check cycle count
		assert.Equal(t, 16, cycles)
	})

	t.Run("CALL NZ conditional - flag not set", func(t *testing.T) {
		mmu := memory.New()
		cpu := New(mmu)

		cpu.pc = 0xC000
		cpu.sp = 0xFFFE
		cpu.f = 0 // Zero flag not set

		// Setup CALL NZ,nn instruction: C4 78 56 (CALL NZ,0x5678)
		mmu.Write(0xC000, 0xC4)
		mmu.Write(0xC001, 0x78) // low byte
		mmu.Write(0xC002, 0x56) // high byte

		cpu.pc++
		cycles := opcode0xC4(cpu)

		// Should execute the call
		assert.Equal(t, uint16(0x5678), cpu.pc)
		assert.Equal(t, uint16(0xFFFC), cpu.sp)
		assert.Equal(t, uint8(0x03), mmu.Read(0xFFFC)) // return address low
		assert.Equal(t, uint8(0xC0), mmu.Read(0xFFFD)) // return address high
		assert.Equal(t, 24, cycles)
	})

	t.Run("CALL NZ conditional - flag set", func(t *testing.T) {
		mmu := memory.New()
		cpu := New(mmu)

		cpu.pc = 0xC000
		cpu.sp = 0xFFFE
		cpu.f = uint8(zeroFlag) // Zero flag set

		// Setup CALL NZ,nn instruction: C4 78 56 (CALL NZ,0x5678)
		mmu.Write(0xC000, 0xC4)
		mmu.Write(0xC001, 0x78) // low byte
		mmu.Write(0xC002, 0x56) // high byte

		cpu.pc++
		cycles := opcode0xC4(cpu)

		// Should NOT execute the call, just skip operand
		assert.Equal(t, uint16(0xC003), cpu.pc) // PC advanced by 3
		assert.Equal(t, uint16(0xFFFE), cpu.sp) // SP unchanged
		assert.Equal(t, 12, cycles)
	})

	t.Run("CALL Z conditional - flag set", func(t *testing.T) {
		mmu := memory.New()
		cpu := New(mmu)

		cpu.pc = 0xC000
		cpu.sp = 0xFFFE
		cpu.f = uint8(zeroFlag) // Zero flag set

		// Setup CALL Z,nn instruction: CC 78 56 (CALL Z,0x5678)
		mmu.Write(0xC000, 0xCC)
		mmu.Write(0xC001, 0x78)
		mmu.Write(0xC002, 0x56)

		cpu.pc++
		cycles := opcode0xCC(cpu)

		// Should execute the call
		assert.Equal(t, uint16(0x5678), cpu.pc)
		assert.Equal(t, uint16(0xFFFC), cpu.sp)
		assert.Equal(t, uint8(0x03), mmu.Read(0xFFFC))
		assert.Equal(t, uint8(0xC0), mmu.Read(0xFFFD))
		assert.Equal(t, 24, cycles)
	})

	t.Run("CALL NC conditional - carry not set", func(t *testing.T) {
		mmu := memory.New()
		cpu := New(mmu)

		cpu.pc = 0xC000
		cpu.sp = 0xFFFE
		cpu.f = 0 // Carry flag not set

		// Setup CALL NC,nn instruction: D4 AB CD
		mmu.Write(0xC000, 0xD4)
		mmu.Write(0xC001, 0xAB)
		mmu.Write(0xC002, 0xCD)

		cpu.pc++
		cycles := opcode0xD4(cpu)

		// Should execute the call
		assert.Equal(t, uint16(0xCDAB), cpu.pc)
		assert.Equal(t, uint16(0xFFFC), cpu.sp)
		assert.Equal(t, 24, cycles)
	})

	t.Run("CALL C conditional - carry set", func(t *testing.T) {
		mmu := memory.New()
		cpu := New(mmu)

		cpu.pc = 0xC000
		cpu.sp = 0xFFFE
		cpu.f = uint8(carryFlag) // Carry flag set

		// Setup CALL C,nn instruction: DC EF BE
		mmu.Write(0xC000, 0xDC)
		mmu.Write(0xC001, 0xEF)
		mmu.Write(0xC002, 0xBE)

		cpu.pc++
		cycles := opcode0xDC(cpu)

		// Should execute the call
		assert.Equal(t, uint16(0xBEEF), cpu.pc)
		assert.Equal(t, uint16(0xFFFC), cpu.sp)
		assert.Equal(t, 24, cycles)
	})

	t.Run("Full CALL/RET cycle", func(t *testing.T) {
		mmu := memory.New()
		cpu := New(mmu)

		cpu.pc = 0xC100
		cpu.sp = 0xFFFE

		// Setup CALL 0xC200 at 0xC100
		mmu.Write(0xC100, 0xCD)
		mmu.Write(0xC101, 0x00)
		mmu.Write(0xC102, 0xC2)

		// Setup RET at 0xC200
		mmu.Write(0xC200, 0xC9)

		// Execute CALL
		cpu.pc++
		opcode0xCD(cpu)
		assert.Equal(t, uint16(0xC200), cpu.pc)
		assert.Equal(t, uint16(0xFFFC), cpu.sp)

		// Execute RET
		cpu.pc++
		opcode0xC9(cpu)
		assert.Equal(t, uint16(0xC103), cpu.pc) // Should return to instruction after CALL
		assert.Equal(t, uint16(0xFFFE), cpu.sp)
	})
}

func TestJRInstructions(t *testing.T) {
	t.Run("JR unconditional - forward jump", func(t *testing.T) {
		mmu := memory.New()
		cpu := New(mmu)

		cpu.pc = 0xC000 // PC points to operand (after decode)
		cpu.sp = 0xFFFE

		// JR +5 (0x05)
		mmu.Write(0xC000, 0x05)

		cycles := opcode0x18(cpu)

		// Should jump to 0xC000 + 1 + 5 = 0xC006
		assert.Equal(t, uint16(0xC006), cpu.pc)
		assert.Equal(t, 12, cycles)
	})

	t.Run("JR unconditional - backward jump", func(t *testing.T) {
		mmu := memory.New()
		cpu := New(mmu)

		cpu.pc = 0xC010 // PC points to operand

		// JR -5 (0xFB = -5 in two's complement)
		mmu.Write(0xC010, 0xFB)

		cycles := opcode0x18(cpu)

		// Should jump to 0xC010 + 1 + (-5) = 0xC00C
		assert.Equal(t, uint16(0xC00C), cpu.pc)
		assert.Equal(t, 12, cycles)
	})

	t.Run("JR unconditional - infinite loop (JR 0xFE)", func(t *testing.T) {
		mmu := memory.New()
		cpu := New(mmu)

		cpu.pc = 0xCC5F // The problematic case from your ROM

		// JR -2 (0xFE = -2 in two's complement)
		mmu.Write(0xCC5F, 0xFE)

		cycles := opcode0x18(cpu)

		// Should jump to 0xCC5F + 1 + (-2) = 0xCC5E
		// This would be an infinite loop if 0xCC5E contains the JR instruction
		assert.Equal(t, uint16(0xCC5E), cpu.pc)
		assert.Equal(t, 12, cycles)
	})

	t.Run("JR NZ - condition false", func(t *testing.T) {
		mmu := memory.New()
		cpu := New(mmu)

		cpu.pc = 0xC000
		cpu.f = uint8(zeroFlag) // Zero flag set, so NZ condition is false

		// JR NZ, +10
		mmu.Write(0xC000, 0x0A)

		cycles := opcode0x20(cpu)

		// Should NOT jump, just advance PC by 1
		assert.Equal(t, uint16(0xC001), cpu.pc)
		assert.Equal(t, 8, cycles)
	})

	t.Run("JR NZ - condition true", func(t *testing.T) {
		mmu := memory.New()
		cpu := New(mmu)

		cpu.pc = 0xC000
		cpu.f = 0 // Zero flag not set, so NZ condition is true

		// JR NZ, +10
		mmu.Write(0xC000, 0x0A)

		cycles := opcode0x20(cpu)

		// Should jump to 0xC000 + 1 + 10 = 0xC00B
		assert.Equal(t, uint16(0xC00B), cpu.pc)
		assert.Equal(t, 12, cycles)
	})

	t.Run("JR Z - condition true", func(t *testing.T) {
		mmu := memory.New()
		cpu := New(mmu)

		cpu.pc = 0xC000
		cpu.f = uint8(zeroFlag) // Zero flag set, so Z condition is true

		// JR Z, -3
		mmu.Write(0xC000, 0xFD) // -3 in two's complement

		cycles := opcode0x28(cpu)

		// Should jump to 0xC000 + 1 + (-3) = 0xBFFE
		assert.Equal(t, uint16(0xBFFE), cpu.pc)
		assert.Equal(t, 12, cycles)
	})

	t.Run("JR NC - condition true", func(t *testing.T) {
		mmu := memory.New()
		cpu := New(mmu)

		cpu.pc = 0xC000
		cpu.f = 0 // Carry flag not set, so NC condition is true

		// JR NC, +8
		mmu.Write(0xC000, 0x08)

		cycles := opcode0x30(cpu)

		// Should jump to 0xC000 + 1 + 8 = 0xC009
		assert.Equal(t, uint16(0xC009), cpu.pc)
		assert.Equal(t, 12, cycles)
	})

	t.Run("JR C - condition true", func(t *testing.T) {
		mmu := memory.New()
		cpu := New(mmu)

		cpu.pc = 0xC000
		cpu.f = uint8(carryFlag) // Carry flag set, so C condition is true

		// JR C, -1
		mmu.Write(0xC000, 0xFF) // -1 in two's complement

		cycles := opcode0x38(cpu)

		// Should jump to 0xC000 + 1 + (-1) = 0xC000 (stays at same place)
		assert.Equal(t, uint16(0xC000), cpu.pc)
		assert.Equal(t, 12, cycles)
	})
}
