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
		{desc: "does not set zero for register A", reg: &cpu.a, arg: 0, want: 0},
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
		{desc: "sets carry flag", reg: &cpu.a, arg: 0x80, want: 0, flags: carryFlag},
		{desc: "sets zero flag", reg: &cpu.b, arg: 0, want: 0, flags: zeroFlag},
		{desc: "does not set zero for register A", reg: &cpu.a, arg: 0, want: 0},
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
		{desc: "does not set zero for register A", reg: &cpu.a, arg: 0, want: 0},
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
		{desc: "sets carry flag", reg: &cpu.a, arg: 1, want: 0, flags: carryFlag},
		{desc: "sets zero flag", reg: &cpu.b, arg: 0, want: 0, flags: zeroFlag},
		{desc: "does not set zero for register A", reg: &cpu.a, arg: 0, want: 0},
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
