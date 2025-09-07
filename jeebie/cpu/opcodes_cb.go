package cpu

// RLC B
// 0xCB00:
func opcode0xCB00(c *CPU) int {
	c.rlc(&c.b)
	c.bus.Tick(8)
	return 8
}

// RLC C
// 0xCB01:
func opcode0xCB01(c *CPU) int {
	c.rlc(&c.c)
	c.bus.Tick(8)
	return 8
}

// RLC D
// 0xCB02:
func opcode0xCB02(c *CPU) int {
	c.rlc(&c.d)
	c.bus.Tick(8)
	return 8
}

// RLC E
// 0xCB03:
func opcode0xCB03(c *CPU) int {
	c.rlc(&c.e)
	c.bus.Tick(8)
	return 8
}

// RLC H
// 0xCB04:
func opcode0xCB04(c *CPU) int {
	c.rlc(&c.h)
	c.bus.Tick(8)
	return 8
}

// RLC L
// 0xCB05:
func opcode0xCB05(c *CPU) int {
	c.rlc(&c.l)
	c.bus.Tick(8)
	return 8
}

// RLC (HL)
// 0xCB06:
func opcode0xCB06(c *CPU) int {
	c.bus.Tick(8)
	addr := c.getHL()
	value := c.bus.Read(addr)
	c.bus.Tick(4)
	c.rlc(&value)
	c.bus.Write(addr, value)
	c.bus.Tick(4)
	return 16
}

// RLC A
// 0xCB07:
func opcode0xCB07(c *CPU) int {
	c.rlc(&c.a)
	c.bus.Tick(8)
	return 8
}

// RRC B
// 0xCB08:
func opcode0xCB08(c *CPU) int {
	c.rrc(&c.b)
	c.bus.Tick(8)
	return 8
}

// RRC C
// 0xCB09:
func opcode0xCB09(c *CPU) int {
	c.rrc(&c.c)
	c.bus.Tick(8)
	return 8
}

// RRC D
// 0xCB0A:
func opcode0xCB0A(c *CPU) int {
	c.rrc(&c.d)
	c.bus.Tick(8)
	return 8
}

// RRC E
// 0xCB0B:
func opcode0xCB0B(c *CPU) int {
	c.rrc(&c.e)
	c.bus.Tick(8)
	return 8
}

// RRC H
// 0xCB0C:
func opcode0xCB0C(c *CPU) int {
	c.rrc(&c.h)
	c.bus.Tick(8)
	return 8
}

// RRC L
// 0xCB0D:
func opcode0xCB0D(c *CPU) int {
	c.rrc(&c.l)
	c.bus.Tick(8)
	return 8
}

// RRC (HL)
// 0xCB0E:
func opcode0xCB0E(c *CPU) int {
	c.bus.Tick(8)
	addr := c.getHL()
	value := c.bus.Read(addr)
	c.rrc(&value)
	c.bus.Tick(4)
	c.bus.Write(addr, value)
	c.bus.Tick(4)
	return 16
}

// RRC A
// 0xCB0F:
func opcode0xCB0F(c *CPU) int {
	c.rrc(&c.a)
	c.bus.Tick(8)
	return 8
}

// RL B
// 0xCB10:
func opcode0xCB10(c *CPU) int {
	c.rl(&c.b)
	c.bus.Tick(8)
	return 8
}

// RL C
// 0xCB11:
func opcode0xCB11(c *CPU) int {
	c.rl(&c.c)
	c.bus.Tick(8)
	return 8
}

// RL D
// 0xCB12:
func opcode0xCB12(c *CPU) int {
	c.rl(&c.d)
	c.bus.Tick(8)
	return 8
}

// RL E
// 0xCB13:
func opcode0xCB13(c *CPU) int {
	c.rl(&c.e)
	c.bus.Tick(8)
	return 8
}

// RL H
// 0xCB14:
func opcode0xCB14(c *CPU) int {
	c.rl(&c.h)
	c.bus.Tick(8)
	return 8
}

// RL L
// 0xCB15:
func opcode0xCB15(c *CPU) int {
	c.rl(&c.l)
	c.bus.Tick(8)
	return 8
}

// RL (HL)
// 0xCB16:
func opcode0xCB16(c *CPU) int {
	c.bus.Tick(8)
	addr := c.getHL()
	value := c.bus.Read(addr)
	c.rl(&value)
	c.bus.Tick(4)
	c.bus.Write(addr, value)
	c.bus.Tick(4)
	return 16
}

// RL A
// 0xCB17:
func opcode0xCB17(c *CPU) int {
	c.rl(&c.a)
	c.bus.Tick(8)
	return 8
}

// RR B
// 0xCB18:
func opcode0xCB18(c *CPU) int {
	c.rr(&c.b)
	c.bus.Tick(8)
	return 8
}

// RR C
// 0xCB19:
func opcode0xCB19(c *CPU) int {
	c.rr(&c.c)
	c.bus.Tick(8)
	return 8
}

// RR D
// 0xCB1A:
func opcode0xCB1A(c *CPU) int {
	c.rr(&c.d)
	c.bus.Tick(8)
	return 8
}

// RR E
// 0xCB1B:
func opcode0xCB1B(c *CPU) int {
	c.rr(&c.e)
	c.bus.Tick(8)
	return 8
}

// RR H
// 0xCB1C:
func opcode0xCB1C(c *CPU) int {
	c.rr(&c.h)
	c.bus.Tick(8)
	return 8
}

// RR L
// 0xCB1D:
func opcode0xCB1D(c *CPU) int {
	c.rr(&c.l)
	c.bus.Tick(8)
	return 8
}

// RR (HL)
// 0xCB1E:
func opcode0xCB1E(c *CPU) int {
	c.bus.Tick(8)
	addr := c.getHL()
	value := c.bus.Read(addr)
	c.rr(&value)
	c.bus.Tick(4)
	c.bus.Write(addr, value)
	c.bus.Tick(4)
	return 16
}

// RR A
// 0xCB1F:
func opcode0xCB1F(c *CPU) int {
	c.rr(&c.a)
	c.bus.Tick(8)
	return 8
}

// SLA B
// 0xCB20:
func opcode0xCB20(c *CPU) int {
	c.sla(&c.b)
	c.bus.Tick(8)
	return 8
}

// SLA C
// 0xCB21:
func opcode0xCB21(c *CPU) int {
	c.sla(&c.c)
	c.bus.Tick(8)
	return 8
}

// SLA D
// 0xCB22:
func opcode0xCB22(c *CPU) int {
	c.sla(&c.d)
	c.bus.Tick(8)
	return 8
}

// SLA E
// 0xCB23:
func opcode0xCB23(c *CPU) int {
	c.sla(&c.e)
	c.bus.Tick(8)
	return 8
}

// SLA H
// 0xCB24:
func opcode0xCB24(c *CPU) int {
	c.sla(&c.h)
	c.bus.Tick(8)
	return 8
}

// SLA L
// 0xCB25:
func opcode0xCB25(c *CPU) int {
	c.sla(&c.l)
	c.bus.Tick(8)
	return 8
}

// SLA (HL)
// 0xCB26:
func opcode0xCB26(c *CPU) int {
	c.bus.Tick(8)
	addr := c.getHL()
	value := c.bus.Read(addr)
	c.bus.Tick(4)
	c.sla(&value)
	c.bus.Write(addr, value)
	c.bus.Tick(4)
	return 16
}

// SLA A
// 0xCB27:
func opcode0xCB27(c *CPU) int {
	c.sla(&c.a)
	c.bus.Tick(8)
	return 8
}

// SRA B
// 0xCB28:
func opcode0xCB28(c *CPU) int {
	c.sra(&c.b)
	c.bus.Tick(8)
	return 8
}

// SRA C
// 0xCB29:
func opcode0xCB29(c *CPU) int {
	c.sra(&c.c)
	c.bus.Tick(8)
	return 8
}

// SRA D
// 0xCB2A:
func opcode0xCB2A(c *CPU) int {
	c.sra(&c.d)
	c.bus.Tick(8)
	return 8
}

// SRA E
// 0xCB2B:
func opcode0xCB2B(c *CPU) int {
	c.sra(&c.e)
	c.bus.Tick(8)
	return 8
}

// SRA H
// 0xCB2C:
func opcode0xCB2C(c *CPU) int {
	c.sra(&c.h)
	c.bus.Tick(8)
	return 8
}

// SRA L
// 0xCB2D:
func opcode0xCB2D(c *CPU) int {
	c.sra(&c.l)
	c.bus.Tick(8)
	return 8
}

// SRA (HL)
// 0xCB2E:
func opcode0xCB2E(c *CPU) int {
	c.bus.Tick(8)
	addr := c.getHL()
	value := c.bus.Read(addr)
	c.bus.Tick(4)
	c.sra(&value)
	c.bus.Write(addr, value)
	c.bus.Tick(4)
	return 16
}

// SRA A
// 0xCB2F:
func opcode0xCB2F(c *CPU) int {
	c.sra(&c.a)
	c.bus.Tick(8)
	return 8
}

// SWAP B
// 0xCB30:
func opcode0xCB30(c *CPU) int {
	c.swap(&c.b)
	c.bus.Tick(8)
	return 8
}

// SWAP C
// 0xCB31:
func opcode0xCB31(c *CPU) int {
	c.swap(&c.c)
	c.bus.Tick(8)
	return 8
}

// SWAP D
// 0xCB32:
func opcode0xCB32(c *CPU) int {
	c.swap(&c.d)
	c.bus.Tick(8)
	return 8
}

// SWAP E
// 0xCB33:
func opcode0xCB33(c *CPU) int {
	c.swap(&c.e)
	c.bus.Tick(8)
	return 8
}

// SWAP H
// 0xCB34:
func opcode0xCB34(c *CPU) int {
	c.swap(&c.h)
	c.bus.Tick(8)
	return 8
}

// SWAP L
// 0xCB35:
func opcode0xCB35(c *CPU) int {
	c.swap(&c.l)
	c.bus.Tick(8)
	return 8
}

// SWAP (HL)
// 0xCB36":
func opcode0xCB36(c *CPU) int {
	c.bus.Tick(8)
	addr := c.getHL()
	value := c.bus.Read(addr)
	c.swap(&value)
	c.bus.Tick(4)
	c.bus.Write(addr, value)
	c.bus.Tick(4)
	return 16
}

// SWAP A
// 0xCB37:
func opcode0xCB37(c *CPU) int {
	c.swap(&c.a)
	c.bus.Tick(8)
	return 8
}

// SRL B
// 0xCB38:
func opcode0xCB38(c *CPU) int {
	c.srl(&c.b)
	c.bus.Tick(8)
	return 8
}

// SRL C
// 0xCB39:
func opcode0xCB39(c *CPU) int {
	c.srl(&c.c)
	c.bus.Tick(8)
	return 8
}

// SRL D
// 0xCB3A:
func opcode0xCB3A(c *CPU) int {
	c.srl(&c.d)
	c.bus.Tick(8)
	return 8
}

// SRL E
// 0xCB3B:
func opcode0xCB3B(c *CPU) int {
	c.srl(&c.e)
	c.bus.Tick(8)
	return 8
}

// SRL H
// 0xCB3C:
func opcode0xCB3C(c *CPU) int {
	c.srl(&c.h)
	c.bus.Tick(8)
	return 8
}

// SRL L
// 0xCB3D:
func opcode0xCB3D(c *CPU) int {
	c.srl(&c.l)
	c.bus.Tick(8)
	return 8
}

// SRL (HL)
// 0xCB3E:
func opcode0xCB3E(c *CPU) int {
	c.bus.Tick(8)
	addr := c.getHL()
	value := c.bus.Read(addr)
	c.srl(&value)
	c.bus.Tick(4)
	c.bus.Write(addr, value)
	c.bus.Tick(4)
	return 16
}

// SRL A
// 0xCB3F:
func opcode0xCB3F(c *CPU) int {
	c.srl(&c.a)
	c.bus.Tick(8)
	return 8
}

// BIT 0 B
// 0xCB40:
func opcode0xCB40(c *CPU) int {
	c.bit(0, c.b)
	c.bus.Tick(8)
	return 8
}

// BIT 0 C
// 0xCB41:
func opcode0xCB41(c *CPU) int {
	c.bit(0, c.c)
	c.bus.Tick(8)
	return 8
}

// BIT 0 D
// 0xCB42:
func opcode0xCB42(c *CPU) int {
	c.bit(0, c.d)
	c.bus.Tick(8)
	return 8
}

// BIT 0 E
// 0xCB43:
func opcode0xCB43(c *CPU) int {
	c.bit(0, c.e)
	c.bus.Tick(8)
	return 8
}

// BIT 0 H
// 0xCB44:
func opcode0xCB44(c *CPU) int {
	c.bit(0, c.h)
	c.bus.Tick(8)
	return 8
}

// BIT 0 L
// 0xCB45:
func opcode0xCB45(c *CPU) int {
	c.bit(0, c.l)
	c.bus.Tick(8)
	return 8
}

// BIT 0 (HL)
// 0xCB46:
func opcode0xCB46(c *CPU) int {
	// CB prefix and opcode already fetched (8 cycles)
	c.bus.Tick(8)

	// Read from (HL) (4 cycles)
	value := c.bus.Read(c.getHL())
	c.bus.Tick(4)

	// Execute bit test (no additional cycles for BIT)
	c.bit(0, value)

	return 12
}

// BIT 0 A
// 0xCB47:
func opcode0xCB47(c *CPU) int {
	c.bit(0, c.a)
	c.bus.Tick(8)
	return 8
}

// BIT 1 B
// 0xCB48:
func opcode0xCB48(c *CPU) int {
	c.bit(1, c.b)
	c.bus.Tick(8)
	return 8
}

// BIT 1 C
// 0xCB49:
func opcode0xCB49(c *CPU) int {
	c.bit(1, c.c)
	c.bus.Tick(8)
	return 8
}

// BIT 1 D
// 0xCB4A:
func opcode0xCB4A(c *CPU) int {
	c.bit(1, c.d)
	c.bus.Tick(8)
	return 8
}

// BIT 1 E
// 0xCB4B:
func opcode0xCB4B(c *CPU) int {
	c.bit(1, c.e)
	c.bus.Tick(8)
	return 8
}

// BIT 1 H
// 0xCB4C:
func opcode0xCB4C(c *CPU) int {
	c.bit(1, c.h)
	c.bus.Tick(8)
	return 8
}

// BIT 1 L
// 0xCB4D:
func opcode0xCB4D(c *CPU) int {
	c.bit(1, c.l)
	c.bus.Tick(8)
	return 8
}

// BIT 1 (HL)
// 0xCB4E:
func opcode0xCB4E(c *CPU) int {
	// CB prefix and opcode already fetched (8 cycles)
	c.bus.Tick(8)

	// Read from (HL) (4 cycles)
	value := c.bus.Read(c.getHL())
	c.bus.Tick(4)

	// Execute bit test (no additional cycles for BIT)
	c.bit(1, value)

	return 12
}

// BIT 1 A
// 0xCB4F:
func opcode0xCB4F(c *CPU) int {
	c.bit(1, c.a)
	c.bus.Tick(8)
	return 8
}

// BIT 2 B
// 0xCB50:
func opcode0xCB50(c *CPU) int {
	c.bit(2, c.b)
	c.bus.Tick(8)
	return 8
}

// BIT 2 C
// 0xCB51:
func opcode0xCB51(c *CPU) int {
	c.bit(2, c.c)
	c.bus.Tick(8)
	return 8
}

// BIT 2 D
// 0xCB52:
func opcode0xCB52(c *CPU) int {
	c.bit(2, c.d)
	c.bus.Tick(8)
	return 8
}

// BIT 2 E
// 0xCB53:
func opcode0xCB53(c *CPU) int {
	c.bit(2, c.e)
	c.bus.Tick(8)
	return 8
}

// BIT 2 H
// 0xCB54:
func opcode0xCB54(c *CPU) int {
	c.bit(2, c.h)
	c.bus.Tick(8)
	return 8
}

// BIT 2 L
// 0xCB55:
func opcode0xCB55(c *CPU) int {
	c.bit(2, c.l)
	c.bus.Tick(8)
	return 8
}

// BIT 2 (HL)
// 0xCB56:
func opcode0xCB56(c *CPU) int {
	// CB prefix and opcode already fetched (8 cycles)
	c.bus.Tick(8)

	// Read from (HL) (4 cycles)
	value := c.bus.Read(c.getHL())
	c.bus.Tick(4)

	// Execute bit test (no additional cycles for BIT)
	c.bit(2, value)

	return 12
}

// BIT 2 A
// 0xCB57:
func opcode0xCB57(c *CPU) int {
	c.bit(2, c.a)
	c.bus.Tick(8)
	return 8
}

// BIT 3 B
// 0xCB58:
func opcode0xCB58(c *CPU) int {
	c.bit(3, c.b)
	c.bus.Tick(8)
	return 8
}

// BIT 3 C
// 0xCB59:
func opcode0xCB59(c *CPU) int {
	c.bit(3, c.c)
	c.bus.Tick(8)
	return 8
}

// BIT 3 D
// 0xCB5A:
func opcode0xCB5A(c *CPU) int {
	c.bit(3, c.d)
	c.bus.Tick(8)
	return 8
}

// BIT 3 E
// 0xCB5B:
func opcode0xCB5B(c *CPU) int {
	c.bit(3, c.e)
	c.bus.Tick(8)
	return 8
}

// BIT 3 H
// 0xCB5C:
func opcode0xCB5C(c *CPU) int {
	c.bit(3, c.h)
	c.bus.Tick(8)
	return 8
}

// BIT 3 L
// 0xCB5D:
func opcode0xCB5D(c *CPU) int {
	c.bit(3, c.l)
	c.bus.Tick(8)
	return 8
}

// BIT 3 (HL)
// 0xCB5E:
func opcode0xCB5E(c *CPU) int {
	// CB prefix and opcode already fetched (8 cycles)
	c.bus.Tick(8)

	// Read from (HL) (4 cycles)
	value := c.bus.Read(c.getHL())
	c.bus.Tick(4)

	// Execute bit test (no additional cycles for BIT)
	c.bit(3, value)

	return 12
}

// BIT 3 A
// 0xCB5F:
func opcode0xCB5F(c *CPU) int {
	c.bit(3, c.a)
	c.bus.Tick(8)
	return 8
}

// BIT 4 B
// 0xCB60:
func opcode0xCB60(c *CPU) int {
	c.bit(4, c.b)
	c.bus.Tick(8)
	return 8
}

// BIT 4 C
// 0xCB61:
func opcode0xCB61(c *CPU) int {
	c.bit(4, c.c)
	c.bus.Tick(8)
	return 8
}

// BIT 4 D
// 0xCB62:
func opcode0xCB62(c *CPU) int {
	c.bit(4, c.d)
	c.bus.Tick(8)
	return 8
}

// BIT 4 E
// 0xCB63:
func opcode0xCB63(c *CPU) int {
	c.bit(4, c.e)
	c.bus.Tick(8)
	return 8
}

// BIT 4 H
// 0xCB64:
func opcode0xCB64(c *CPU) int {
	c.bit(4, c.h)
	c.bus.Tick(8)
	return 8
}

// BIT 4 L
// 0xCB65:
func opcode0xCB65(c *CPU) int {
	c.bit(4, c.l)
	c.bus.Tick(8)
	return 8
}

// BIT 4 (HL)
// 0xCB66:
func opcode0xCB66(c *CPU) int {
	// CB prefix and opcode already fetched (8 cycles)
	c.bus.Tick(8)

	// Read from (HL) (4 cycles)
	value := c.bus.Read(c.getHL())
	c.bus.Tick(4)

	// Execute bit test (no additional cycles for BIT)
	c.bit(4, value)

	return 12
}

// BIT 4 A
// 0xCB67:
func opcode0xCB67(c *CPU) int {
	c.bit(4, c.a)
	c.bus.Tick(8)
	return 8
}

// BIT 5 B
// 0xCB68:
func opcode0xCB68(c *CPU) int {
	c.bit(5, c.b)
	c.bus.Tick(8)
	return 8
}

// BIT 5 C
// 0xCB69:
func opcode0xCB69(c *CPU) int {
	c.bit(5, c.c)
	c.bus.Tick(8)
	return 8
}

// BIT 5 D
// 0xCB6A:
func opcode0xCB6A(c *CPU) int {
	c.bit(5, c.d)
	c.bus.Tick(8)
	return 8
}

// BIT 5 E
// 0xCB6B:
func opcode0xCB6B(c *CPU) int {
	c.bit(5, c.e)
	c.bus.Tick(8)
	return 8
}

// BIT 5 H
// 0xCB6C:
func opcode0xCB6C(c *CPU) int {
	c.bit(5, c.h)
	c.bus.Tick(8)
	return 8
}

// BIT 5 L
// 0xCB6D:
func opcode0xCB6D(c *CPU) int {
	c.bit(5, c.l)
	c.bus.Tick(8)
	return 8
}

// BIT 5 (HL)
// 0xCB6E:
func opcode0xCB6E(c *CPU) int {
	// CB prefix and opcode already fetched (8 cycles)
	c.bus.Tick(8)

	// Read from (HL) (4 cycles)
	value := c.bus.Read(c.getHL())
	c.bus.Tick(4)

	// Execute bit test (no additional cycles for BIT)
	c.bit(5, value)

	return 12
}

// BIT 5 A
// 0xCB6F:
func opcode0xCB6F(c *CPU) int {
	c.bit(5, c.a)
	c.bus.Tick(8)
	return 8
}

// BIT 6 B
// 0xCB70:
func opcode0xCB70(c *CPU) int {
	c.bit(6, c.b)
	c.bus.Tick(8)
	return 8
}

// BIT 6 C
// 0xCB71:
func opcode0xCB71(c *CPU) int {
	c.bit(6, c.c)
	c.bus.Tick(8)
	return 8
}

// BIT 6 D
// 0xCB72:
func opcode0xCB72(c *CPU) int {
	c.bit(6, c.d)
	c.bus.Tick(8)
	return 8
}

// BIT 6 E
// 0xCB73:
func opcode0xCB73(c *CPU) int {
	c.bit(6, c.e)
	c.bus.Tick(8)
	return 8
}

// BIT 6 H
// 0xCB74:
func opcode0xCB74(c *CPU) int {
	c.bit(6, c.h)
	c.bus.Tick(8)
	return 8
}

// BIT 6 L
// 0xCB75:
func opcode0xCB75(c *CPU) int {
	c.bit(6, c.l)
	c.bus.Tick(8)
	return 8
}

// BIT 6 (HL)
// 0xCB76:
func opcode0xCB76(c *CPU) int {
	// CB prefix and opcode already fetched (8 cycles)
	c.bus.Tick(8)

	// Read from (HL) (4 cycles)
	value := c.bus.Read(c.getHL())
	c.bus.Tick(4)

	// Execute bit test (no additional cycles for BIT)
	c.bit(6, value)

	return 12
}

// BIT 6 A
// 0xCB77:
func opcode0xCB77(c *CPU) int {
	c.bit(6, c.a)
	c.bus.Tick(8)
	return 8
}

// BIT 7 B
// 0xCB78:
func opcode0xCB78(c *CPU) int {
	c.bit(7, c.b)
	c.bus.Tick(8)
	return 8
}

// BIT 7 C
// 0xCB79:
func opcode0xCB79(c *CPU) int {
	c.bit(7, c.c)
	c.bus.Tick(8)
	return 8
}

// BIT 7 D
// 0xCB7A:
func opcode0xCB7A(c *CPU) int {
	c.bit(7, c.d)
	c.bus.Tick(8)
	return 8
}

// BIT 7 E
// 0xCB7B:
func opcode0xCB7B(c *CPU) int {
	c.bit(7, c.e)
	c.bus.Tick(8)
	return 8
}

// BIT 7 H
// 0xCB7C:
func opcode0xCB7C(c *CPU) int {
	c.bit(7, c.h)
	c.bus.Tick(8)
	return 8
}

// BIT 7 L
// 0xCB7D:
func opcode0xCB7D(c *CPU) int {
	c.bit(7, c.l)
	c.bus.Tick(8)
	return 8
}

// BIT 7 (HL)
// 0xCB7E:
func opcode0xCB7E(c *CPU) int {
	// CB prefix and opcode already fetched (8 cycles)
	c.bus.Tick(8)

	// Read from (HL) (4 cycles)
	value := c.bus.Read(c.getHL())
	c.bus.Tick(4)

	// Execute bit test (no additional cycles for BIT)
	c.bit(7, value)

	return 12
}

// BIT 7 A
// 0xCB7F:
func opcode0xCB7F(c *CPU) int {
	c.bit(7, c.a)
	c.bus.Tick(8)
	return 8
}

// RES 0 B
// 0xCB80:
func opcode0xCB80(c *CPU) int {
	c.res(0, &c.b)
	c.bus.Tick(8)
	return 8
}

// RES 0 C
// 0xCB81:
func opcode0xCB81(c *CPU) int {
	c.res(0, &c.c)
	c.bus.Tick(8)
	return 8
}

// RES 0 D
// 0xCB82:
func opcode0xCB82(c *CPU) int {
	c.res(0, &c.d)
	c.bus.Tick(8)
	return 8
}

// RES 0 E
// 0xCB83:
func opcode0xCB83(c *CPU) int {
	c.res(0, &c.e)
	c.bus.Tick(8)
	return 8
}

// RES 0 H
// 0xCB84:
func opcode0xCB84(c *CPU) int {
	c.res(0, &c.h)
	c.bus.Tick(8)
	return 8
}

// RES 0 L
// 0xCB85:
func opcode0xCB85(c *CPU) int {
	c.res(0, &c.l)
	c.bus.Tick(8)
	return 8
}

// RES 0 (HL)
// 0xCB86:
func opcode0xCB86(c *CPU) int {
	c.bus.Tick(8)
	addr := c.getHL()
	value := c.bus.Read(addr)
	c.res(0, &value)
	c.bus.Tick(4)
	c.bus.Write(addr, value)
	c.bus.Tick(4)
	return 16
}

// RES 0 A
// 0xCB87:
func opcode0xCB87(c *CPU) int {
	c.res(0, &c.a)
	c.bus.Tick(8)
	return 8
}

// RES 1 B
// 0xCB88:
func opcode0xCB88(c *CPU) int {
	c.res(1, &c.b)
	c.bus.Tick(8)
	return 8
}

// RES 1 C
// 0xCB89:
func opcode0xCB89(c *CPU) int {
	c.res(1, &c.c)
	c.bus.Tick(8)
	return 8
}

// RES 1 D
// 0xCB8A:
func opcode0xCB8A(c *CPU) int {
	c.res(1, &c.d)
	c.bus.Tick(8)
	return 8
}

// RES 1 E
// 0xCB8B:
func opcode0xCB8B(c *CPU) int {
	c.res(1, &c.e)
	c.bus.Tick(8)
	return 8
}

// RES 1 H
// 0xCB8C:
func opcode0xCB8C(c *CPU) int {
	c.res(1, &c.h)
	c.bus.Tick(8)
	return 8
}

// RES 1 L
// 0xCB8D:
func opcode0xCB8D(c *CPU) int {
	c.res(1, &c.l)
	c.bus.Tick(8)
	return 8
}

// RES 1 (HL)
// 0xCB8E:
func opcode0xCB8E(c *CPU) int {
	c.bus.Tick(8)
	addr := c.getHL()
	value := c.bus.Read(addr)
	c.res(1, &value)
	c.bus.Tick(4)
	c.bus.Write(addr, value)
	c.bus.Tick(4)
	return 16
}

// RES 1 A
// 0xCB8F:
func opcode0xCB8F(c *CPU) int {
	c.res(1, &c.a)
	c.bus.Tick(8)
	return 8
}

// RES 2 B
// 0xCB90:
func opcode0xCB90(c *CPU) int {
	c.res(2, &c.b)
	c.bus.Tick(8)
	return 8
}

// RES 2 C
// 0xCB91:
func opcode0xCB91(c *CPU) int {
	c.res(2, &c.c)
	c.bus.Tick(8)
	return 8
}

// RES 2 D
// 0xCB92:
func opcode0xCB92(c *CPU) int {
	c.res(2, &c.d)
	c.bus.Tick(8)
	return 8
}

// RES 2 E
// 0xCB93:
func opcode0xCB93(c *CPU) int {
	c.res(2, &c.e)
	c.bus.Tick(8)
	return 8
}

// RES 2 H
// 0xCB94:
func opcode0xCB94(c *CPU) int {
	c.res(2, &c.h)
	c.bus.Tick(8)
	return 8
}

// RES 2 L
// 0xCB95:
func opcode0xCB95(c *CPU) int {
	c.res(2, &c.l)
	c.bus.Tick(8)
	return 8
}

// RES 2 (HL)
// 0xCB96:
func opcode0xCB96(c *CPU) int {
	c.bus.Tick(8)
	addr := c.getHL()
	value := c.bus.Read(addr)
	c.res(2, &value)
	c.bus.Tick(4)
	c.bus.Write(addr, value)
	c.bus.Tick(4)
	return 16
}

// RES 2 A
// 0xCB97:
func opcode0xCB97(c *CPU) int {
	c.res(2, &c.a)
	c.bus.Tick(8)
	return 8
}

// RES 3 B
// 0xCB98:
func opcode0xCB98(c *CPU) int {
	c.res(3, &c.b)
	c.bus.Tick(8)
	return 8
}

// RES 3 C
// 0xCB99:
func opcode0xCB99(c *CPU) int {
	c.res(3, &c.c)
	c.bus.Tick(8)
	return 8
}

// RES 3 D
// 0xCB9A:
func opcode0xCB9A(c *CPU) int {
	c.res(3, &c.d)
	c.bus.Tick(8)
	return 8
}

// RES 3 E
// 0xCB9B:
func opcode0xCB9B(c *CPU) int {
	c.res(3, &c.e)
	c.bus.Tick(8)
	return 8
}

// RES 3 H
// 0xCB9C:
func opcode0xCB9C(c *CPU) int {
	c.res(3, &c.h)
	c.bus.Tick(8)
	return 8
}

// RES 3 L
// 0xCB9D:
func opcode0xCB9D(c *CPU) int {
	c.res(3, &c.l)
	c.bus.Tick(8)
	return 8
}

// RES 3 (HL)
// 0xCB9E:
func opcode0xCB9E(c *CPU) int {
	c.bus.Tick(8)
	addr := c.getHL()
	value := c.bus.Read(addr)
	c.res(3, &value)
	c.bus.Tick(4)
	c.bus.Write(addr, value)
	c.bus.Tick(4)
	return 16
}

// RES 3 A
// 0xCB9F:
func opcode0xCB9F(c *CPU) int {
	c.res(3, &c.a)
	c.bus.Tick(8)
	return 8
}

// RES 4 B
// 0xCBA0:
func opcode0xCBA0(c *CPU) int {
	c.res(4, &c.b)
	c.bus.Tick(8)
	return 8
}

// RES 4 C
// 0xCBA1:
func opcode0xCBA1(c *CPU) int {
	c.res(4, &c.c)
	c.bus.Tick(8)
	return 8
}

// RES 4 D
// 0xCBA2:
func opcode0xCBA2(c *CPU) int {
	c.res(4, &c.d)
	c.bus.Tick(8)
	return 8
}

// RES 4 E
// 0xCBA3:
func opcode0xCBA3(c *CPU) int {
	c.res(4, &c.e)
	c.bus.Tick(8)
	return 8
}

// RES 4 H
// 0xCBA4:
func opcode0xCBA4(c *CPU) int {
	c.res(4, &c.h)
	c.bus.Tick(8)
	return 8
}

// RES 4 L
// 0xCBA5:
func opcode0xCBA5(c *CPU) int {
	c.res(4, &c.l)
	c.bus.Tick(8)
	return 8
}

// RES 4 (HL)
// 0xCBA6:
func opcode0xCBA6(c *CPU) int {
	c.bus.Tick(8)
	addr := c.getHL()
	value := c.bus.Read(addr)
	c.res(4, &value)
	c.bus.Tick(4)
	c.bus.Write(addr, value)
	c.bus.Tick(4)
	return 16
}

// RES 4 A
// 0xCBA7:
func opcode0xCBA7(c *CPU) int {
	c.res(4, &c.a)
	c.bus.Tick(8)
	return 8
}

// RES 5 B
// 0xCBA8:
func opcode0xCBA8(c *CPU) int {
	c.res(5, &c.b)
	c.bus.Tick(8)
	return 8
}

// RES 5 C
// 0xCBA9:
func opcode0xCBA9(c *CPU) int {
	c.res(5, &c.c)
	c.bus.Tick(8)
	return 8
}

// RES 5 D
// 0xCBAA:
func opcode0xCBAA(c *CPU) int {
	c.res(5, &c.d)
	c.bus.Tick(8)
	return 8
}

// RES 5 E
// 0xCBAB:
func opcode0xCBAB(c *CPU) int {
	c.res(5, &c.e)
	c.bus.Tick(8)
	return 8
}

// RES 5 H
// 0xCBAC:
func opcode0xCBAC(c *CPU) int {
	c.res(5, &c.h)
	c.bus.Tick(8)
	return 8
}

// RES 5 L
// 0xCBAD:
func opcode0xCBAD(c *CPU) int {
	c.res(5, &c.l)
	c.bus.Tick(8)
	return 8
}

// RES 5 (HL)
// 0xCBAE:
func opcode0xCBAE(c *CPU) int {
	c.bus.Tick(8)
	addr := c.getHL()
	value := c.bus.Read(addr)
	c.res(5, &value)
	c.bus.Tick(4)
	c.bus.Write(addr, value)
	c.bus.Tick(4)
	return 16
}

// RES 5 A
// 0xCBAF:
func opcode0xCBAF(c *CPU) int {
	c.res(5, &c.a)
	c.bus.Tick(8)
	return 8
}

// RES 6 B
// 0xCBB0:
func opcode0xCBB0(c *CPU) int {
	c.res(6, &c.b)
	c.bus.Tick(8)
	return 8
}

// RES 6 C
// 0xCBB1:
func opcode0xCBB1(c *CPU) int {
	c.res(6, &c.c)
	c.bus.Tick(8)
	return 8
}

// RES 6 D
// 0xCBB2:
func opcode0xCBB2(c *CPU) int {
	c.res(6, &c.d)
	c.bus.Tick(8)
	return 8
}

// RES 6 E
// 0xCBB3:
func opcode0xCBB3(c *CPU) int {
	c.res(6, &c.e)
	c.bus.Tick(8)
	return 8
}

// RES 6 H
// 0xCBB4:
func opcode0xCBB4(c *CPU) int {
	c.res(6, &c.h)
	c.bus.Tick(8)
	return 8
}

// RES 6 L
// 0xCBB5:
func opcode0xCBB5(c *CPU) int {
	c.res(6, &c.l)
	c.bus.Tick(8)
	return 8
}

// RES 6 (HL)
// 0xCBB6:
func opcode0xCBB6(c *CPU) int {
	c.bus.Tick(8)
	addr := c.getHL()
	value := c.bus.Read(addr)
	c.res(6, &value)
	c.bus.Tick(4)
	c.bus.Write(addr, value)
	c.bus.Tick(4)
	return 16
}

// RES 6 A
// 0xCBB7:
func opcode0xCBB7(c *CPU) int {
	c.res(6, &c.a)
	c.bus.Tick(8)
	return 8
}

// RES 7 B
// 0xCBB8:
func opcode0xCBB8(c *CPU) int {
	c.res(7, &c.b)
	c.bus.Tick(8)
	return 8
}

// RES 7 C
// 0xCBB9:
func opcode0xCBB9(c *CPU) int {
	c.res(7, &c.c)
	c.bus.Tick(8)
	return 8
}

// RES 7 D
// 0xCBBA:
func opcode0xCBBA(c *CPU) int {
	c.res(7, &c.d)
	c.bus.Tick(8)
	return 8
}

// RES 7 E
// 0xCBBB:
func opcode0xCBBB(c *CPU) int {
	c.res(7, &c.e)
	c.bus.Tick(8)
	return 8
}

// RES 7 H
// 0xCBBC:
func opcode0xCBBC(c *CPU) int {
	c.res(7, &c.h)
	c.bus.Tick(8)
	return 8
}

// RES 7 L
// 0xCBBD:
func opcode0xCBBD(c *CPU) int {
	c.res(7, &c.l)
	c.bus.Tick(8)
	return 8
}

// RES 7 (HL)
// 0xCBBE:
func opcode0xCBBE(c *CPU) int {
	c.bus.Tick(8)
	addr := c.getHL()
	value := c.bus.Read(addr)
	c.res(7, &value)
	c.bus.Tick(4)
	c.bus.Write(addr, value)
	c.bus.Tick(4)
	return 16
}

// RES 7 A
// 0xCBBF:
func opcode0xCBBF(c *CPU) int {
	c.res(7, &c.a)
	c.bus.Tick(8)
	return 8
}

// SET 0 B
// 0xCBC0:
func opcode0xCBC0(c *CPU) int {
	c.set(0, &c.b)
	c.bus.Tick(8)
	return 8
}

// SET 0 C
// 0xCBC1:
func opcode0xCBC1(c *CPU) int {
	c.set(0, &c.c)
	c.bus.Tick(8)
	return 8
}

// SET 0 D
// 0xCBC2:
func opcode0xCBC2(c *CPU) int {
	c.set(0, &c.d)
	c.bus.Tick(8)
	return 8
}

// SET 0 E
// 0xCBC3:
func opcode0xCBC3(c *CPU) int {
	c.set(0, &c.e)
	c.bus.Tick(8)
	return 8
}

// SET 0 H
// 0xCBC4:
func opcode0xCBC4(c *CPU) int {
	c.set(0, &c.h)
	c.bus.Tick(8)
	return 8
}

// SET 0 L
// 0xCBC5:
func opcode0xCBC5(c *CPU) int {
	c.set(0, &c.l)
	c.bus.Tick(8)
	return 8
}

// SET 0 (HL)
// 0xCBC6:
func opcode0xCBC6(c *CPU) int {
	c.bus.Tick(8)
	addr := c.getHL()
	value := c.bus.Read(addr)
	c.set(0, &value)
	c.bus.Tick(4)
	c.bus.Write(addr, value)
	c.bus.Tick(4)
	return 16
}

// SET 0 A
// 0xCBC7:
func opcode0xCBC7(c *CPU) int {
	c.set(0, &c.a)
	c.bus.Tick(8)
	return 8
}

// SET 1 B
// 0xCBC8:
func opcode0xCBC8(c *CPU) int {
	c.set(1, &c.b)
	c.bus.Tick(8)
	return 8
}

// SET 1 C
// 0xCBC9:
func opcode0xCBC9(c *CPU) int {
	c.set(1, &c.c)
	c.bus.Tick(8)
	return 8
}

// SET 1 D
// 0xCBCA:
func opcode0xCBCA(c *CPU) int {
	c.set(1, &c.d)
	c.bus.Tick(8)
	return 8
}

// SET 1 E
// 0xCBCB:
func opcode0xCBCB(c *CPU) int {
	c.set(1, &c.e)
	c.bus.Tick(8)
	return 8
}

// SET 1 H
// 0xCBCC:
func opcode0xCBCC(c *CPU) int {
	c.set(1, &c.h)
	c.bus.Tick(8)
	return 8
}

// SET 1 L
// 0xCBCD:
func opcode0xCBCD(c *CPU) int {
	c.set(1, &c.l)
	c.bus.Tick(8)
	return 8
}

// SET 1 (HL)
// 0xCBCE:
func opcode0xCBCE(c *CPU) int {
	c.bus.Tick(8)
	addr := c.getHL()
	value := c.bus.Read(addr)
	c.set(1, &value)
	c.bus.Tick(4)
	c.bus.Write(addr, value)
	c.bus.Tick(4)
	return 16
}

// SET 1 A
// 0xCBCF:
func opcode0xCBCF(c *CPU) int {
	c.set(1, &c.a)
	c.bus.Tick(8)
	return 8
}

// SET 2 B
// 0xCBD0:
func opcode0xCBD0(c *CPU) int {
	c.set(2, &c.b)
	c.bus.Tick(8)
	return 8
}

// SET 2 C
// 0xCBD1:
func opcode0xCBD1(c *CPU) int {
	c.set(2, &c.c)
	c.bus.Tick(8)
	return 8
}

// SET 2 D
// 0xCBD2:
func opcode0xCBD2(c *CPU) int {
	c.set(2, &c.d)
	c.bus.Tick(8)
	return 8
}

// SET 2 E
// 0xCBD3:
func opcode0xCBD3(c *CPU) int {
	c.set(2, &c.e)
	c.bus.Tick(8)
	return 8
}

// SET 2 H
// 0xCBD4:
func opcode0xCBD4(c *CPU) int {
	c.set(2, &c.h)
	c.bus.Tick(8)
	return 8
}

// SET 2 L
// 0xCBD5:
func opcode0xCBD5(c *CPU) int {
	c.set(2, &c.l)
	c.bus.Tick(8)
	return 8
}

// SET 2 (HL)
// 0xCBD6:
func opcode0xCBD6(c *CPU) int {
	c.bus.Tick(8)
	addr := c.getHL()
	value := c.bus.Read(addr)
	c.set(2, &value)
	c.bus.Tick(4)
	c.bus.Write(addr, value)
	c.bus.Tick(4)
	return 16
}

// SET 2 A
// 0xCBD7:
func opcode0xCBD7(c *CPU) int {
	c.set(2, &c.a)
	c.bus.Tick(8)
	return 8
}

// SET 3 B
// 0xCBD8:
func opcode0xCBD8(c *CPU) int {
	c.set(3, &c.b)
	c.bus.Tick(8)
	return 8
}

// SET 3 C
// 0xCBD9:
func opcode0xCBD9(c *CPU) int {
	c.set(3, &c.c)
	c.bus.Tick(8)
	return 8
}

// SET 3 D
// 0xCBDA:
func opcode0xCBDA(c *CPU) int {
	c.set(3, &c.d)
	c.bus.Tick(8)
	return 8
}

// SET 3 E
// 0xCBDB:
func opcode0xCBDB(c *CPU) int {
	c.set(3, &c.e)
	c.bus.Tick(8)
	return 8
}

// SET 3 H
// 0xCBDC:
func opcode0xCBDC(c *CPU) int {
	c.set(3, &c.h)
	c.bus.Tick(8)
	return 8
}

// SET 3 L
// 0xCBDD:
func opcode0xCBDD(c *CPU) int {
	c.set(3, &c.l)
	c.bus.Tick(8)
	return 8
}

// SET 3 (HL)
// 0xCBDE:
func opcode0xCBDE(c *CPU) int {
	c.bus.Tick(8)
	addr := c.getHL()
	value := c.bus.Read(addr)
	c.set(3, &value)
	c.bus.Tick(4)
	c.bus.Write(addr, value)
	c.bus.Tick(4)
	return 16
}

// SET 3 A
// 0xCBDF:
func opcode0xCBDF(c *CPU) int {
	c.set(3, &c.a)
	c.bus.Tick(8)
	return 8
}

// SET 4 B
// 0xCBE0:
func opcode0xCBE0(c *CPU) int {
	c.set(4, &c.b)
	c.bus.Tick(8)
	return 8
}

// SET 4 C
// 0xCBE1:
func opcode0xCBE1(c *CPU) int {
	c.set(4, &c.c)
	c.bus.Tick(8)
	return 8
}

// SET 4 D
// 0xCBE2:
func opcode0xCBE2(c *CPU) int {
	c.set(4, &c.d)
	c.bus.Tick(8)
	return 8
}

// SET 4 E
// 0xCBE3:
func opcode0xCBE3(c *CPU) int {
	c.set(4, &c.e)
	c.bus.Tick(8)
	return 8
}

// SET 4 H
// 0xCBE4:
func opcode0xCBE4(c *CPU) int {
	c.set(4, &c.h)
	c.bus.Tick(8)
	return 8
}

// SET 4 L
// 0xCBE5:
func opcode0xCBE5(c *CPU) int {
	c.set(4, &c.l)
	c.bus.Tick(8)
	return 8
}

// SET 4 (HL)
// 0xCBE6:
func opcode0xCBE6(c *CPU) int {
	c.bus.Tick(8)
	addr := c.getHL()
	value := c.bus.Read(addr)
	c.set(4, &value)
	c.bus.Tick(4)
	c.bus.Write(addr, value)
	c.bus.Tick(4)
	return 16
}

// SET 4 A
// 0xCBE7:
func opcode0xCBE7(c *CPU) int {
	c.set(4, &c.a)
	c.bus.Tick(8)
	return 8
}

// SET 5 B
// 0xCBE8:
func opcode0xCBE8(c *CPU) int {
	c.set(5, &c.b)
	c.bus.Tick(8)
	return 8
}

// SET 5 C
// 0xCBE9:
func opcode0xCBE9(c *CPU) int {
	c.set(5, &c.c)
	c.bus.Tick(8)
	return 8
}

// SET 5 D
// 0xCBEA:
func opcode0xCBEA(c *CPU) int {
	c.set(5, &c.d)
	c.bus.Tick(8)
	return 8
}

// SET 5 E
// 0xCBEB:
func opcode0xCBEB(c *CPU) int {
	c.set(5, &c.e)
	c.bus.Tick(8)
	return 8
}

// SET 5 H
// 0xCBEC:
func opcode0xCBEC(c *CPU) int {
	c.set(5, &c.h)
	c.bus.Tick(8)
	return 8
}

// SET 5 L
// 0xCBED:
func opcode0xCBED(c *CPU) int {
	c.set(5, &c.l)
	c.bus.Tick(8)
	return 8
}

// SET 5 (HL)
// 0xCBEE:
func opcode0xCBEE(c *CPU) int {
	c.bus.Tick(8)
	addr := c.getHL()
	value := c.bus.Read(addr)
	c.set(5, &value)
	c.bus.Tick(4)
	c.bus.Write(addr, value)
	c.bus.Tick(4)
	return 16
}

// SET 5 A
// 0xCBEF:
func opcode0xCBEF(c *CPU) int {
	c.set(5, &c.a)
	c.bus.Tick(8)
	return 8
}

// SET 6 B
// 0xCBF0:
func opcode0xCBF0(c *CPU) int {
	c.set(6, &c.b)
	c.bus.Tick(8)
	return 8
}

// SET 6 C
// 0xCBF1:
func opcode0xCBF1(c *CPU) int {
	c.set(6, &c.c)
	c.bus.Tick(8)
	return 8
}

// SET 6 D
// 0xCBF2:
func opcode0xCBF2(c *CPU) int {
	c.set(6, &c.d)
	c.bus.Tick(8)
	return 8
}

// SET 6 E
// 0xCBF3:
func opcode0xCBF3(c *CPU) int {
	c.set(6, &c.e)
	c.bus.Tick(8)
	return 8
}

// SET 6 H
// 0xCBF4:
func opcode0xCBF4(c *CPU) int {
	c.set(6, &c.h)
	c.bus.Tick(8)
	return 8
}

// SET 6 L
// 0xCBF5:
func opcode0xCBF5(c *CPU) int {
	c.set(6, &c.l)
	c.bus.Tick(8)
	return 8
}

// SET 6 (HL)
// 0xCBF6:
func opcode0xCBF6(c *CPU) int {
	c.bus.Tick(8)
	addr := c.getHL()
	value := c.bus.Read(addr)
	c.set(6, &value)
	c.bus.Tick(4)
	c.bus.Write(addr, value)
	c.bus.Tick(4)
	return 16
}

// SET 6 A
// 0xCBF7:
func opcode0xCBF7(c *CPU) int {
	c.set(6, &c.a)
	c.bus.Tick(8)
	return 8
}

// SET 7 B
// 0xCBF8:
func opcode0xCBF8(c *CPU) int {
	c.set(7, &c.b)
	c.bus.Tick(8)
	return 8
}

// SET 7 C
// 0xCBF9:
func opcode0xCBF9(c *CPU) int {
	c.set(7, &c.c)
	c.bus.Tick(8)
	return 8
}

// SET 7 D
// 0xCBFA:
func opcode0xCBFA(c *CPU) int {
	c.set(7, &c.d)
	c.bus.Tick(8)
	return 8
}

// SET 7 E
// 0xCBFB:
func opcode0xCBFB(c *CPU) int {
	c.set(7, &c.e)
	c.bus.Tick(8)
	return 8
}

// SET 7 H
// 0xCBFC:
func opcode0xCBFC(c *CPU) int {
	c.set(7, &c.h)
	c.bus.Tick(8)
	return 8
}

// SET 7 L
// 0xCBFD:
func opcode0xCBFD(c *CPU) int {
	c.set(7, &c.l)
	c.bus.Tick(8)
	return 8
}

// SET 7 (HL)
// 0xCBFE:
func opcode0xCBFE(c *CPU) int {
	c.bus.Tick(8)
	addr := c.getHL()
	value := c.bus.Read(addr)
	c.set(7, &value)
	c.bus.Tick(4)
	c.bus.Write(addr, value)
	c.bus.Tick(4)
	return 16
}

// SET 7 A
// 0xCBFF:
func opcode0xCBFF(c *CPU) int {
	c.set(7, &c.a)
	c.bus.Tick(8)
	return 8
}
