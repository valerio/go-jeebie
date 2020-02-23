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
		carry bool
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
			if tC.carry {
				cpu.setFlag(carryFlag)
			}
			cpu.a = tC.a
			cpu.sub(tC.arg)
			assert.Equal(t, tC.want, cpu.a)
			assert.Equalf(t, uint8(tC.flags), cpu.f, "flags don't match")
		})
	}
}

func TestCPU_sbc(t *testing.T) {
}

func TestCPU_and(t *testing.T) {
}

func TestCPU_or(t *testing.T) {
}

func TestCPU_xor(t *testing.T) {
}

func TestCPU_cp(t *testing.T) {
}

func TestCPU_swap(t *testing.T) {
}

func TestCPU_daa(t *testing.T) {
}

func TestCPU_cpl(t *testing.T) {
}

func TestCPU_ccf(t *testing.T) {
}

func TestCPU_scf(t *testing.T) {
}

func TestCPU_bit(t *testing.T) {
}

func TestCPU_set(t *testing.T) {
}

func TestCPU_res(t *testing.T) {
}

func TestCPU_jr(t *testing.T) {
}
