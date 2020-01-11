package cpu

import "fmt"

import "github.com/valerio/go-jeebie/jeebie/bit"

func unimplemented(cpu *CPU) int {
	msg := fmt.Sprintf("Unimplemented opcode 0x%X was called.", cpu.currentOpcode)
	panic(msg)
}

//NOP
//#0x00:
func opcode0x00(_ *CPU) int {
	return 4
}

//LD BC, nn
//#0x01:
func opcode0x01(cpu *CPU) int {
	cpu.setBC(cpu.readImmediateWord())
	return 12
}

//LD (BC), A
//#0x02:
func opcode0x02(cpu *CPU) int {
	cpu.memory.Write(cpu.getBC(), cpu.a)
	return 8
}

//INC BC
//#0x03:
func opcode0x03(cpu *CPU) int {
	cpu.setBC(cpu.getBC() + 1)
	return 8
}

//INC B
//#0x04:
func opcode0x04(cpu *CPU) int {
	cpu.inc(&cpu.b)
	return 4
}

//DEC B
//#0x05:
func opcode0x05(cpu *CPU) int {
	cpu.dec(&cpu.b)
	return 4
}

//LD B, n
//#0x06:
func opcode0x06(cpu *CPU) int {
	cpu.b = cpu.readImmediate()
	return 8
}

//RLCA
//#0x07:
func opcode0x07(cpu *CPU) int {
	cpu.rlc(&cpu.a)
	return 4
}

//LD (nn), SP
//#0x08:
func opcode0x08(cpu *CPU) int {
	addr := cpu.readImmediateWord()
	cpu.memory.Write(addr, bit.Low(cpu.sp))
	cpu.memory.Write(addr+1, bit.High(cpu.sp))
	return 20
}

//ADD HL, BC
//#0x09:
func opcode0x09(cpu *CPU) int {
	cpu.addToHL(cpu.getBC())
	return 8
}

//LD A, (BC)
//#0x0A:
func opcode0x0A(cpu *CPU) int {
	cpu.a = cpu.memory.Read(cpu.getBC())
	return 8
}

//DEC BC
//#0x0B:
func opcode0x0B(cpu *CPU) int {
	cpu.setBC(cpu.getBC() - 1)
	return 8
}

//INC C
//#0x0C:
func opcode0x0C(cpu *CPU) int {
	cpu.inc(&cpu.c)
	return 4
}

//DEC C
//#0x0D:
func opcode0x0D(cpu *CPU) int {
	cpu.dec(&cpu.c)
	return 4
}

//LD C, n
//#0x0E:
func opcode0x0E(cpu *CPU) int {
	cpu.c = cpu.readImmediate()
	return 8
}

//RRCA
//#0x0F:
func opcode0x0F(cpu *CPU) int {
	cpu.rrc(&cpu.a)
	return 4
}

//STOP
//#0x10:
func opcode0x10(cpu *CPU) int {
	cpu.stopped = true
	return 4
}

//LD DE, nn
//#0x11:
func opcode0x11(cpu *CPU) int {
	cpu.setDE(cpu.readImmediateWord())
	return 12
}

//LD (DE), A
//#0x12:
func opcode0x12(cpu *CPU) int {
	cpu.memory.Write(cpu.getDE(), cpu.a)
	return 8
}

//INC DE
//#0x13:
func opcode0x13(cpu *CPU) int {
	cpu.setDE(cpu.getDE() + 1)
	return 8
}

//INC D
//#0x14:
func opcode0x14(cpu *CPU) int {
	cpu.inc(&cpu.d)
	return 4
}

//DEC D
//#0x15:
func opcode0x15(cpu *CPU) int {
	cpu.dec(&cpu.d)
	return 4
}

//LD D, n
//#0x16:
func opcode0x16(cpu *CPU) int {
	cpu.d = cpu.readImmediate()
	return 8
}

//RLA
//#0x17:
func opcode0x17(cpu *CPU) int {
	cpu.rl(&cpu.a)
	return 4
}

//JR n
//#0x18:
func opcode0x18(cpu *CPU) int {
	cpu.jr()
	return 12
}

//ADD HL, DE
//#0x19:
func opcode0x19(cpu *CPU) int {
	cpu.addToHL(cpu.getDE())
	return 8
}

//LD A, (DE)
//#0x1A:
func opcode0x1A(cpu *CPU) int {
	cpu.a = cpu.memory.Read(cpu.getDE())
	return 8
}

//DEC DE
//#0x1B:
func opcode0x1B(cpu *CPU) int {
	cpu.setDE(cpu.getDE())
	return 8
}

//INC E
//#0x1C:
func opcode0x1C(cpu *CPU) int {
	cpu.inc(&cpu.e)
	return 4
}

//DEC E
//#0x1D:
func opcode0x1D(cpu *CPU) int {
	cpu.dec(&cpu.e)
	return 4
}

//LD E, n
//#0x1E:
func opcode0x1E(cpu *CPU) int {
	cpu.e = cpu.readImmediate()
	return 8
}

//RRA
//#0x1F:
func opcode0x1F(cpu *CPU) int {
	cpu.rr(&cpu.a)
	return 4
}

//JR NZ, n
//#0x20:
func opcode0x20(cpu *CPU) int {
	if !cpu.isSetFlag(zeroFlag) {
		cpu.jr()
		return 12
	}

	return 8
}

//LD HL, nn
//#0x21:
func opcode0x21(cpu *CPU) int {
	cpu.setHL(cpu.readImmediateWord())
	return 12
}

//LDI (HL), A
//#0x22:
func opcode0x22(cpu *CPU) int {
	cpu.memory.Write(cpu.getHL(), cpu.a)
	cpu.setHL(cpu.getHL() + 1)
	return 8
}

//INC HL
//#0x23:
func opcode0x23(cpu *CPU) int {
	cpu.setHL(cpu.getHL() + 1)
	return 8
}

//INC H
//#0x24:
func opcode0x24(cpu *CPU) int {
	cpu.inc(&cpu.h)
	return 4
}

//DEC H
//#0x25:
func opcode0x25(cpu *CPU) int {
	cpu.dec(&cpu.h)
	return 4
}

//LD H, n
//#0x26:
func opcode0x26(cpu *CPU) int {
	cpu.h = cpu.readImmediate()
	return 8
}

//DAA
//#0x27:
func opcode0x27(cpu *CPU) int {

	return 4
}

//JR Z, n
//#0x28:
func opcode0x28(cpu *CPU) int {
	if cpu.isSetFlag(zeroFlag) {
		cpu.jr()
		return 12
	}

	return 8
}

//ADD HL, HL
//#0x29:
func opcode0x29(cpu *CPU) int {
	cpu.addToHL(cpu.getHL())
	return 8
}

//LDI A, (HL)
//#0x2A:
func opcode0x2A(cpu *CPU) int {
	cpu.a = cpu.memory.Read(cpu.getHL())
	cpu.setHL(cpu.getHL() + 1)
	return 8
}

//DEC HL
//#0x2B:
func opcode0x2B(cpu *CPU) int {
	cpu.setHL(cpu.getHL() - 1)
	return 8
}

//INC L
//#0x2C:
func opcode0x2C(cpu *CPU) int {
	cpu.inc(&cpu.l)
	return 4
}

//DEC L
//#0x2D:
func opcode0x2D(cpu *CPU) int {
	cpu.dec(&cpu.l)
	return 4
}

//LD L, n
//#0x2E:
func opcode0x2E(cpu *CPU) int {
	cpu.l = cpu.readImmediate()
	return 8
}

//CPL
//#0x2F:
func opcode0x2F(cpu *CPU) int {
	value := cpu.a
	cpu.a = value ^ 0xFF
	cpu.setFlag(halfCarryFlag)
	cpu.setFlag(subFlag)
	return 4
}

//JR NC, n
//#0x30:
func opcode0x30(cpu *CPU) int {
	if !cpu.isSetFlag(carryFlag) {
		cpu.jr()
		return 12
	}

	return 8
}

//LD SP, nn
//#0x31:
func opcode0x31(cpu *CPU) int {
	cpu.sp = cpu.readImmediateWord()
	return 12
}

//LDD (HL), A
//#0x32:
func opcode0x32(cpu *CPU) int {
	cpu.memory.Write(cpu.getHL(), cpu.a)
	cpu.setHL(cpu.getHL() - 1)
	return 8
}

//INC SP
//#0x33:
func opcode0x33(cpu *CPU) int {
	cpu.sp++
	return 4
}

//INC (HL)
//#0x34:
func opcode0x34(cpu *CPU) int {
	addr := cpu.getHL()
	value := cpu.memory.Read(addr)
	cpu.memory.Write(addr, value+1)
	return 12
}

//DEC (HL)
//#0x35:
func opcode0x35(cpu *CPU) int {
	addr := cpu.getHL()
	value := cpu.memory.Read(addr)
	cpu.memory.Write(addr, value-1)
	return 12
}

//LD (HL), n
//#0x36:
func opcode0x36(cpu *CPU) int {
	cpu.memory.Write(cpu.getHL(), cpu.readImmediate())
	return 12
}

//SCF
//#0x37:
func opcode0x37(cpu *CPU) int {
	cpu.setFlag(carryFlag)
	cpu.resetFlag(subFlag)
	cpu.resetFlag(halfCarryFlag)
	return 4
}

//JR C, n
//#0x38:
func opcode0x38(cpu *CPU) int {
	if cpu.isSetFlag(carryFlag) {
		cpu.jr()
		return 12
	}

	return 8
}

//ADD HL, SP
//#0x39:
func opcode0x39(cpu *CPU) int {
	cpu.addToHL(cpu.sp)
	return 8
}

//LDD A, (HL)
//#0x3A:
func opcode0x3A(cpu *CPU) int {
	cpu.a = cpu.memory.Read(cpu.getHL())
	cpu.setHL(cpu.getHL() - 1)
	return 8
}

//DEC SP
//#0x3B:
func opcode0x3B(cpu *CPU) int {
	cpu.sp--
	return 8
}

//INC A
//#0x3C:
func opcode0x3C(cpu *CPU) int {
	cpu.inc(&cpu.a)
	return 4
}

//DEC A
//#0x3D:
func opcode0x3D(cpu *CPU) int {
	cpu.dec(&cpu.a)
	return 4
}

//LD A, n
//#0x3E:
func opcode0x3E(cpu *CPU) int {
	cpu.a = cpu.readImmediate()
	return 8
}

//CCF
//#0x3F:
func opcode0x3F(cpu *CPU) int {
	cpu.resetFlag(subFlag)
	cpu.resetFlag(halfCarryFlag)
	cpu.setFlagToCondition(carryFlag, !cpu.isSetFlag(carryFlag))
	return 4
}

//LD B, B
//#0x40:
func opcode0x40(cpu *CPU) int {
	return 4
}

//LD B, C
//#0x41:
func opcode0x41(cpu *CPU) int {
	cpu.b = cpu.c
	return 4
}

//LD B, D
//#0x42:
func opcode0x42(cpu *CPU) int {
	cpu.b = cpu.d
	return 4
}

//LD B, E
//#0x43:
func opcode0x43(cpu *CPU) int {
	cpu.b = cpu.e
	return 4
}

//LD B, H
//#0x44:
func opcode0x44(cpu *CPU) int {
	cpu.b = cpu.h
	return 4
}

//LD B, L
//#0x45:
func opcode0x45(cpu *CPU) int {
	cpu.b = cpu.l
	return 4
}

//LD B, (HL)
//#0x46:
func opcode0x46(cpu *CPU) int {
	cpu.b = cpu.memory.Read(cpu.getHL())
	return 8
}

//LD B, A
//#0x47:
func opcode0x47(cpu *CPU) int {
	cpu.b = cpu.a
	return 4
}

//LD C, B
//#0x48:
func opcode0x48(cpu *CPU) int {
	cpu.c = cpu.b
	return 4
}

//LD C, C
//#0x49:
func opcode0x49(cpu *CPU) int {
	return 4
}

//LD C, D
//#0x4A:
func opcode0x4A(cpu *CPU) int {
	cpu.c = cpu.d
	return 4
}

//LD C, E
//#0x4B:
func opcode0x4B(cpu *CPU) int {
	cpu.c = cpu.e
	return 4
}

//LD C, H
//#0x4C:
func opcode0x4C(cpu *CPU) int {
	cpu.c = cpu.h
	return 4
}

//LD C, L
//#0x4D:
func opcode0x4D(cpu *CPU) int {
	cpu.c = cpu.l
	return 4
}

//LD C, (HL)
//#0x4E:
func opcode0x4E(cpu *CPU) int {
	cpu.c = cpu.memory.Read(cpu.getHL())
	return 8
}

//LD C, A
//#0x4F:
func opcode0x4F(cpu *CPU) int {
	cpu.c = cpu.a
	return 4
}

//LD D, B
//#0x50:
func opcode0x50(cpu *CPU) int {
	cpu.d = cpu.b
	return 4
}

//LD D, C
//#0x51:
func opcode0x51(cpu *CPU) int {
	cpu.d = cpu.c
	return 4
}

//LD D, D
//#0x52:
func opcode0x52(cpu *CPU) int {
	return 4
}

//LD D, E
//#0x53:
func opcode0x53(cpu *CPU) int {
	cpu.d = cpu.e
	return 4
}

//LD D, H
//#0x54:
func opcode0x54(cpu *CPU) int {
	cpu.d = cpu.h
	return 4
}

//LD D, L
//#0x55:
func opcode0x55(cpu *CPU) int {
	cpu.d = cpu.l
	return 4
}

//LD D, (HL)
//#0x56:
func opcode0x56(cpu *CPU) int {
	cpu.d = cpu.memory.Read(cpu.getHL())
	return 8
}

//LD D, A
//#0x57:
func opcode0x57(cpu *CPU) int {
	cpu.d = cpu.a
	return 4
}

//LD E, B
//#0x58:
func opcode0x58(cpu *CPU) int {
	cpu.e = cpu.b
	return 4
}

//LD E, C
//#0x59:
func opcode0x59(cpu *CPU) int {
	cpu.e = cpu.c
	return 4
}

//LD E, D
//#0x5A:
func opcode0x5A(cpu *CPU) int {
	cpu.e = cpu.d
	return 4
}

//LD E, E
//#0x5B:
func opcode0x5B(cpu *CPU) int {
	return 4
}

//LD E, H
//#0x5C:
func opcode0x5C(cpu *CPU) int {
	cpu.e = cpu.h
	return 4
}

//LD E, L
//#0x5D:
func opcode0x5D(cpu *CPU) int {
	cpu.e = cpu.l
	return 4
}

//LD E, (HL)
//#0x5E:
func opcode0x5E(cpu *CPU) int {
	cpu.e = cpu.memory.Read(cpu.getHL())
	return 8
}

//LD E, A
//#0x5F:
func opcode0x5F(cpu *CPU) int {
	cpu.e = cpu.a
	return 4
}

//LD H, B
//#0x60:
func opcode0x60(cpu *CPU) int {
	cpu.h = cpu.b
	return 4
}

//LD H, C
//#0x61:
func opcode0x61(cpu *CPU) int {
	cpu.h = cpu.c
	return 4
}

//LD H, D
//#0x62:
func opcode0x62(cpu *CPU) int {
	cpu.h = cpu.d
	return 4
}

//LD H, E
//#0x63:
func opcode0x63(cpu *CPU) int {
	cpu.h = cpu.e
	return 4
}

//LD H, H
//#0x64:
func opcode0x64(cpu *CPU) int {
	return 4
}

//LD H, L
//#0x65:
func opcode0x65(cpu *CPU) int {
	cpu.h = cpu.l
	return 4
}

//LD H, (HL)
//#0x66:
func opcode0x66(cpu *CPU) int {
	cpu.h = cpu.memory.Read(cpu.getHL())
	return 8
}

//LD H, A
//#0x67:
func opcode0x67(cpu *CPU) int {
	cpu.h = cpu.a
	return 4
}

//LD L, B
//#0x68:
func opcode0x68(cpu *CPU) int {
	cpu.l = cpu.b
	return 4
}

//LD L, C
//#0x69:
func opcode0x69(cpu *CPU) int {
	cpu.l = cpu.c
	return 4
}

//LD L, D
//#0x6A:
func opcode0x6A(cpu *CPU) int {
	cpu.l = cpu.d
	return 4
}

//LD L, E
//#0x6B:
func opcode0x6B(cpu *CPU) int {
	cpu.l = cpu.e
	return 4
}

//LD L, H
//#0x6C:
func opcode0x6C(cpu *CPU) int {
	cpu.l = cpu.h
	return 4
}

//LD L, L
//#0x6D:
func opcode0x6D(cpu *CPU) int {
	return 4
}

//LD L, (HL)
//#0x6E:
func opcode0x6E(cpu *CPU) int {
	cpu.l = cpu.memory.Read(cpu.getHL())
	return 8
}

//LD L, A
//#0x6F:
func opcode0x6F(cpu *CPU) int {
	cpu.l = cpu.a
	return 4
}

//LD (HL), B
//#0x70:
func opcode0x70(cpu *CPU) int {
	cpu.memory.Write(cpu.getHL(), cpu.b)
	return 8
}

//LD (HL), C
//#0x71:
func opcode0x71(cpu *CPU) int {
	cpu.memory.Write(cpu.getHL(), cpu.c)
	return 8
}

//LD (HL), D
//#0x72:
func opcode0x72(cpu *CPU) int {
	cpu.memory.Write(cpu.getHL(), cpu.d)
	return 8
}

//LD (HL), E
//#0x73:
func opcode0x73(cpu *CPU) int {
	cpu.memory.Write(cpu.getHL(), cpu.e)
	return 8
}

//LD (HL), H
//#0x74:
func opcode0x74(cpu *CPU) int {
	cpu.memory.Write(cpu.getHL(), cpu.h)
	return 8
}

//LD (HL), L
//#0x75:
func opcode0x75(cpu *CPU) int {
	cpu.memory.Write(cpu.getHL(), cpu.l)
	return 8
}

//HALT
//#0x76:
func opcode0x76(cpu *CPU) int {

	return 4
}

//LD (HL), A
//#0x77:
func opcode0x77(cpu *CPU) int {
	cpu.memory.Write(cpu.getHL(), cpu.a)
	return 8
}

//LD A, B
//#0x78:
func opcode0x78(cpu *CPU) int {
	cpu.a = cpu.b
	return 4
}

//LD A, C
//#0x79:
func opcode0x79(cpu *CPU) int {
	cpu.a = cpu.c
	return 4
}

//LD A, D
//#0x7A:
func opcode0x7A(cpu *CPU) int {
	cpu.a = cpu.d
	return 4
}

//LD A, E
//#0x7B:
func opcode0x7B(cpu *CPU) int {
	cpu.a = cpu.e
	return 4
}

//LD A, H
//#0x7C:
func opcode0x7C(cpu *CPU) int {
	cpu.a = cpu.h
	return 4
}

//LD A, L
//#0x7D:
func opcode0x7D(cpu *CPU) int {
	cpu.a = cpu.l
	return 4
}

//LD, A, (HL)
//#0x7E:
func opcode0x7E(cpu *CPU) int {
	cpu.a = cpu.memory.Read(cpu.getHL())
	return 8
}

//LD A, A
//#0x7F:
func opcode0x7F(cpu *CPU) int {
	return 4
}

//ADD A, B
//#0x80:
func opcode0x80(cpu *CPU) int {
	cpu.addToA(cpu.b)
	return 4
}

//ADD A, C
//#0x81:
func opcode0x81(cpu *CPU) int {
	cpu.addToA(cpu.c)
	return 4
}

//ADD A, D
//#0x82:
func opcode0x82(cpu *CPU) int {
	cpu.addToA(cpu.d)
	return 4
}

//ADD A, E
//#0x83:
func opcode0x83(cpu *CPU) int {
	cpu.addToA(cpu.e)
	return 4
}

//ADD A, H
//#0x84:
func opcode0x84(cpu *CPU) int {
	cpu.addToA(cpu.h)
	return 4
}

//ADD A, L
//#0x85:
func opcode0x85(cpu *CPU) int {
	cpu.addToA(cpu.l)
	return 4
}

//ADD A, (HL)
//#0x86:
func opcode0x86(cpu *CPU) int {
	cpu.addToA(cpu.memory.Read(cpu.getHL()))
	return 8
}

//ADD A, A
//#0x87:
func opcode0x87(cpu *CPU) int {
	cpu.addToA(cpu.a)
	return 4
}

//ADC A, B
//#0x88:
func opcode0x88(cpu *CPU) int {

	return 4
}

//ADC A, C
//#0x89:
func opcode0x89(cpu *CPU) int {

	return 4
}

//ADC A, D
//#0x8A:
func opcode0x8A(cpu *CPU) int {

	return 4
}

//ADC A, E
//#0x8B:
func opcode0x8B(cpu *CPU) int {

	return 4
}

//ADC A, H
//#0x8C:
func opcode0x8C(cpu *CPU) int {

	return 4
}

//ADC A, L
//#0x8D:
func opcode0x8D(cpu *CPU) int {

	return 4
}

//ADC A, (HL)
//#0x8E:
func opcode0x8E(cpu *CPU) int {

	return 8
}

//ADC A, A
//#0x8F:
func opcode0x8F(cpu *CPU) int {

	return 4
}

//SUB A, B
//#0x90:
func opcode0x90(cpu *CPU) int {
	cpu.sub(cpu.b)
	return 4
}

//SUB A, C
//#0x91:
func opcode0x91(cpu *CPU) int {
	cpu.sub(cpu.c)
	return 4
}

//SUB A, D
//#0x92:
func opcode0x92(cpu *CPU) int {
	cpu.sub(cpu.d)
	return 4
}

//SUB A, E
//#0x93:
func opcode0x93(cpu *CPU) int {
	cpu.sub(cpu.e)
	return 4
}

//SUB A, H
//#0x94:
func opcode0x94(cpu *CPU) int {
	cpu.sub(cpu.h)
	return 4
}

//SUB A, L
//#0x95:
func opcode0x95(cpu *CPU) int {
	cpu.sub(cpu.l)
	return 4
}

//SUB A, (HL)
//#0x96:
func opcode0x96(cpu *CPU) int {
	cpu.sub(cpu.memory.Read(cpu.getHL()))
	return 8
}

//SUB A, A
//#0x97:
func opcode0x97(cpu *CPU) int {
	cpu.sub(cpu.a)
	return 4
}

//SBC A, B
//#0x98:
func opcode0x98(cpu *CPU) int {
	cpu.sbc(cpu.b)
	return 4
}

//SBC A, C
//#0x99:
func opcode0x99(cpu *CPU) int {
	cpu.sbc(cpu.c)
	return 4
}

//SBC A, D
//#0x9A:
func opcode0x9A(cpu *CPU) int {
	cpu.sbc(cpu.d)
	return 4
}

//SBC A, E
//#0x9B:
func opcode0x9B(cpu *CPU) int {
	cpu.sbc(cpu.e)
	return 4
}

//SBC A, H
//#0x9C:
func opcode0x9C(cpu *CPU) int {
	cpu.sbc(cpu.h)
	return 4
}

//SBC A, L
//#0x9D:
func opcode0x9D(cpu *CPU) int {
	cpu.sbc(cpu.l)
	return 4
}

//SBC A, (HL)
//#0x9E:
func opcode0x9E(cpu *CPU) int {
	cpu.sbc(cpu.memory.Read(cpu.getHL()))
	return 8
}

//SBC A, A
//#0x9F:
func opcode0x9F(cpu *CPU) int {
	cpu.sbc(cpu.a)
	return 4
}

//AND B
//#0xA0:
func opcode0xA0(cpu *CPU) int {
	cpu.and(cpu.b)
	return 4
}

//AND C
//#0xA1:
func opcode0xA1(cpu *CPU) int {
	cpu.and(cpu.c)
	return 4
}

//AND D
//#0xA2:
func opcode0xA2(cpu *CPU) int {
	cpu.and(cpu.d)
	return 4
}

//AND E
//#0xA3:
func opcode0xA3(cpu *CPU) int {
	cpu.and(cpu.e)
	return 4
}

//AND H
//#0xA4:
func opcode0xA4(cpu *CPU) int {
	cpu.and(cpu.h)
	return 4
}

//AND L
//#0xA5:
func opcode0xA5(cpu *CPU) int {
	cpu.and(cpu.l)
	return 4
}

//AND (HL)
//#0xA6:
func opcode0xA6(cpu *CPU) int {
	cpu.and(cpu.memory.Read(cpu.getHL()))
	return 8
}

//AND A
//#0xA7:
func opcode0xA7(cpu *CPU) int {
	cpu.and(cpu.a)
	return 4
}

//XOR B
//#0xA8:
func opcode0xA8(cpu *CPU) int {
	cpu.xor(cpu.b)
	return 4
}

//XOR C
//#0xA9:
func opcode0xA9(cpu *CPU) int {
	cpu.xor(cpu.c)
	return 4
}

//XOR D
//#0xAA:
func opcode0xAA(cpu *CPU) int {
	cpu.xor(cpu.d)
	return 4
}

//XOR E
//#0xAB:
func opcode0xAB(cpu *CPU) int {
	cpu.xor(cpu.e)
	return 4
}

//XOR H
//#0xAC:
func opcode0xAC(cpu *CPU) int {
	cpu.xor(cpu.h)
	return 4
}

//XOR L
//#0xAD:
func opcode0xAD(cpu *CPU) int {
	cpu.xor(cpu.l)
	return 4
}

//XOR (HL)
//#0xAE:
func opcode0xAE(cpu *CPU) int {
	cpu.xor(cpu.memory.Read(cpu.getHL()))
	return 8
}

//XOR A
//#0xAF:
func opcode0xAF(cpu *CPU) int {
	cpu.xor(cpu.a)
	return 4
}

//OR B
//#0xB0:
func opcode0xB0(cpu *CPU) int {
	cpu.or(cpu.b)
	return 4
}

//OR C
//#0xB1:
func opcode0xB1(cpu *CPU) int {
	cpu.or(cpu.c)
	return 4
}

//OR D
//#0xB2:
func opcode0xB2(cpu *CPU) int {
	cpu.or(cpu.d)
	return 4
}

//OR E
//#0xB3:
func opcode0xB3(cpu *CPU) int {
	cpu.or(cpu.e)
	return 4
}

//OR H
//#0xB4:
func opcode0xB4(cpu *CPU) int {
	cpu.or(cpu.h)
	return 4
}

//OR L
//#0xB5:
func opcode0xB5(cpu *CPU) int {
	cpu.or(cpu.l)
	return 4
}

//OR (HL)
//#0xB6:
func opcode0xB6(cpu *CPU) int {
	cpu.or(cpu.memory.Read(cpu.getHL()))
	return 8
}

//OR A
//#0xB7:
func opcode0xB7(cpu *CPU) int {
	cpu.or(cpu.a)
	return 4
}

//CP B
//#0xB8:
func opcode0xB8(cpu *CPU) int {

	return 4
}

//CP C
//#0xB9:
func opcode0xB9(cpu *CPU) int {

	return 4
}

//CP D
//#0xBA:
func opcode0xBA(cpu *CPU) int {

	return 4
}

//CP E
//#0xBB:
func opcode0xBB(cpu *CPU) int {

	return 4
}

//CP H
//#0xBC:
func opcode0xBC(cpu *CPU) int {

	return 4
}

//CP L
//#0xBD:
func opcode0xBD(cpu *CPU) int {

	return 4
}

//CP (HL)
//#0xBE:
func opcode0xBE(cpu *CPU) int {

	return 8
}

//CP A
//#0xBF:
func opcode0xBF(cpu *CPU) int {

	return 4
}

//RET !FZ
//#0xC0:
func opcode0xC0(cpu *CPU) int {

	return 8
}

//POP BC
//#0xC1:
func opcode0xC1(cpu *CPU) int {

	return 12
}

//JP !FZ, nn
//#0xC2:
func opcode0xC2(cpu *CPU) int {
	if !cpu.isSetFlag(zeroFlag) {
		cpu.jp()
		return 16
	}

	return 12
}

//JP nn
//#0xC3:
func opcode0xC3(cpu *CPU) int {
	cpu.jp()
	return 16
}

//CALL !FZ, nn
//#0xC4:
func opcode0xC4(cpu *CPU) int {

	return 12
}

//PUSH BC
//#0xC5:
func opcode0xC5(cpu *CPU) int {

	return 16
}

//ADD, n
//#0xC6:
func opcode0xC6(cpu *CPU) int {

	return 8
}

//RST 0
//#0xC7:
func opcode0xC7(cpu *CPU) int {

	return 16
}

//RET FZ
//#0xC8:
func opcode0xC8(cpu *CPU) int {

	return 8
}

//RET
//#0xC9:
func opcode0xC9(cpu *CPU) int {

	return 16
}

//JP FZ, nn
//#0xCA:
func opcode0xCA(cpu *CPU) int {
	if cpu.isSetFlag(zeroFlag) {
		cpu.jp()
		return 16
	}

	return 12
}

//Secondary OP Code Set:
//#0xCB:
func opcode0xCB(cpu *CPU) int {
	return 0
}

//CALL FZ, nn
//#0xCC:
func opcode0xCC(cpu *CPU) int {

	return 12
}

//CALL nn
//#0xCD:
func opcode0xCD(cpu *CPU) int {

	return 24
}

//ADC A, n
//#0xCE:
func opcode0xCE(cpu *CPU) int {

	return 8
}

//RST 0x8
//#0xCF:
func opcode0xCF(cpu *CPU) int {

	return 16
}

//RET !FC
//#0xD0:
func opcode0xD0(cpu *CPU) int {

	return 8
}

//POP DE
//#0xD1:
func opcode0xD1(cpu *CPU) int {

	return 12
}

//JP !FC, nn
//#0xD2:
func opcode0xD2(cpu *CPU) int {
	if !cpu.isSetFlag(carryFlag) {
		cpu.jp()
		return 16
	}

	return 12
}

//0xD3 - Illegal
//#0xD3:
func opcode0xD3(cpu *CPU) int {
	return 0
}

//CALL !FC, nn
//#0xD4:
func opcode0xD4(cpu *CPU) int {

	return 12
}

//PUSH DE
//#0xD5:
func opcode0xD5(cpu *CPU) int {

	return 16
}

//SUB A, n
//#0xD6:
func opcode0xD6(cpu *CPU) int {

	return 8
}

//RST 0x10
//#0xD7:
func opcode0xD7(cpu *CPU) int {

	return 16
}

//RET FC
//#0xD8:
func opcode0xD8(cpu *CPU) int {

	return 8
}

//RETI
//#0xD9:
func opcode0xD9(cpu *CPU) int {

	return 16
}

//JP FC, nn
//#0xDA:
func opcode0xDA(cpu *CPU) int {
	if cpu.isSetFlag(carryFlag) {
		cpu.jp()
		return 16
	}

	return 12
}

//0xDB - Illegal
//#0xDB:
func opcode0xDB(cpu *CPU) int {
	return 0
}

//CALL FC, nn
//#0xDC:
func opcode0xDC(cpu *CPU) int {

	return 12
}

//0xDD - Illegal
//#0xDD:
func opcode0xDD(cpu *CPU) int {
	return 0
}

//SBC A, n
//#0xDE:
func opcode0xDE(cpu *CPU) int {
	return 8
}

//RST 0x18
//#0xDF:
func opcode0xDF(cpu *CPU) int {
	return 16
}

//LDH (n), A
//#0xE0:
func opcode0xE0(cpu *CPU) int {

	return 12
}

//POP HL
//#0xE1:
func opcode0xE1(cpu *CPU) int {

	return 12
}

//LD (0xFF00 + C), A
//#0xE2:
func opcode0xE2(cpu *CPU) int {
	addr := 0xFF00 | uint16(cpu.c)
	cpu.memory.Write(addr, cpu.a)
	return 8
}

//0xE3 - Illegal
//#0xE3:
func opcode0xE3(cpu *CPU) int {
	return 0
}

//0xE4 - Illegal
//#0xE4:
func opcode0xE4(cpu *CPU) int {
	return 0
}

//PUSH HL
//#0xE5:
func opcode0xE5(cpu *CPU) int {

	return 16
}

//AND n
//#0xE6:
func opcode0xE6(cpu *CPU) int {

	return 8
}

//RST 0x20
//#0xE7:
func opcode0xE7(cpu *CPU) int {

	return 16
}

//ADD SP, n
//#0xE8:
func opcode0xE8(cpu *CPU) int {
	value := int32(cpu.sp)
	n := int32(cpu.readSignedImmediate())

	result := value + n

	cpu.resetFlag(zeroFlag)
	cpu.resetFlag(subFlag)
	cpu.setFlagToCondition(halfCarryFlag, ((value^n^(result&0xFFFF))&0x10) == 0x10)
	cpu.setFlagToCondition(carryFlag, ((value^n^(result&0xFFFF))&0x100) == 0x100)

	cpu.sp = uint16(result)

	return 16
}

//JP, (HL)
//#0xE9:
func opcode0xE9(cpu *CPU) int {
	cpu.pc = cpu.getHL()
	return 4
}

//LD (nn), A
//#0xEA:
func opcode0xEA(cpu *CPU) int {
	cpu.memory.Write(cpu.readImmediateWord(), cpu.a)
	return 16
}

//0xEB - Illegal
//#0xEB:
func opcode0xEB(cpu *CPU) int {
	return 0
}

//0xEC - Illegal
//#0xEC:
func opcode0xEC(cpu *CPU) int {
	return 0
}

//0xED - Illegal
//#0xED:
func opcode0xED(cpu *CPU) int {
	return 0
}

//XOR n
//#0xEE:
func opcode0xEE(cpu *CPU) int {
	cpu.xor(cpu.readImmediate())
	return 4
}

//RST 0x28
//#0xEF:
func opcode0xEF(cpu *CPU) int {

	return 16
}

//LDH A, (n)
//#0xF0:
func opcode0xF0(cpu *CPU) int {

	return 12
}

//POP AF
//#0xF1:
func opcode0xF1(cpu *CPU) int {

	return 12
}

//LD A, (0xFF00 + C)
//#0xF2:
func opcode0xF2(cpu *CPU) int {
	cpu.a = cpu.memory.Read(0xFF00 | uint16(cpu.c))
	return 8
}

//DI
//#0xF3:
func opcode0xF3(cpu *CPU) int {

	return 4
}

//0xF4 - Illegal
//#0xF4:
func opcode0xF4(cpu *CPU) int {
	return 0
}

//PUSH AF
//#0xF5:
func opcode0xF5(cpu *CPU) int {

	return 16
}

//OR n
//#0xF6:
func opcode0xF6(cpu *CPU) int {
	cpu.or(cpu.readImmediate())
	return 8
}

//RST 0x30
//#0xF7:
func opcode0xF7(cpu *CPU) int {

	return 16
}

//LDHL SP, n
//#0xF8:
func opcode0xF8(cpu *CPU) int {

	return 12
}

//LD SP, HL
//#0xF9:
func opcode0xF9(cpu *CPU) int {
	cpu.sp = cpu.getHL()
	return 8
}

//LD A, (nn)
//#0xFA:
func opcode0xFA(cpu *CPU) int {
	cpu.a = cpu.memory.Read(cpu.readImmediateWord())
	return 16
}

//EI
//#0xFB:
func opcode0xFB(cpu *CPU) int {

	return 4
}

//0xFC - Illegal
//#0xFC:
func opcode0xFC(cpu *CPU) int {
	return 0
}

//0xFD - Illegal
//#0xFD:
func opcode0xFD(cpu *CPU) int {
	return 0
}

//CP n
//#0xFE:
func opcode0xFE(cpu *CPU) int {

	return 8
}

//RST 0x38
//#0xFF:
func opcode0xFF(cpu *CPU) int {

	return 16
}
