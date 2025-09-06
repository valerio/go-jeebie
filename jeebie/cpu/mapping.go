package cpu

import (
	"fmt"

	"github.com/valerio/go-jeebie/jeebie/bit"
)

// Opcode represents a function that executes an opcode
type Opcode func(*CPU) int

// Decode retrieves the instruction identified by the value pointed at by the PC.
// Note: PC must be incremented separately, this is so we can handle the "HALT bug".
func Decode(c *CPU) Opcode {
	// peek PC+1|PC
	instr := c.peekImmediateWord()
	high, low := bit.High(instr), bit.Low(instr)

	// 0xCB is only ever used as a prefix for the next byte.
	if low == 0xCB {
		c.currentOpcode = bit.Combine(0xCB, high)
		return opcodesCB[high]
	}

	c.currentOpcode = bit.Combine(0, low)
	return opcodes[low]
}

var opcodes = [...]Opcode{
	opcode0x00, opcode0x01, opcode0x02, opcode0x03, opcode0x04, opcode0x05, opcode0x06, opcode0x07, opcode0x08, opcode0x09, opcode0x0A, opcode0x0B, opcode0x0C, opcode0x0D, opcode0x0E, opcode0x0F,
	opcode0x10, opcode0x11, opcode0x12, opcode0x13, opcode0x14, opcode0x15, opcode0x16, opcode0x17, opcode0x18, opcode0x19, opcode0x1A, opcode0x1B, opcode0x1C, opcode0x1D, opcode0x1E, opcode0x1F,
	opcode0x20, opcode0x21, opcode0x22, opcode0x23, opcode0x24, opcode0x25, opcode0x26, opcode0x27, opcode0x28, opcode0x29, opcode0x2A, opcode0x2B, opcode0x2C, opcode0x2D, opcode0x2E, opcode0x2F,
	opcode0x30, opcode0x31, opcode0x32, opcode0x33, opcode0x34, opcode0x35, opcode0x36, opcode0x37, opcode0x38, opcode0x39, opcode0x3A, opcode0x3B, opcode0x3C, opcode0x3D, opcode0x3E, opcode0x3F,
	opcode0x40, opcode0x41, opcode0x42, opcode0x43, opcode0x44, opcode0x45, opcode0x46, opcode0x47, opcode0x48, opcode0x49, opcode0x4A, opcode0x4B, opcode0x4C, opcode0x4D, opcode0x4E, opcode0x4F,
	opcode0x50, opcode0x51, opcode0x52, opcode0x53, opcode0x54, opcode0x55, opcode0x56, opcode0x57, opcode0x58, opcode0x59, opcode0x5A, opcode0x5B, opcode0x5C, opcode0x5D, opcode0x5E, opcode0x5F,
	opcode0x60, opcode0x61, opcode0x62, opcode0x63, opcode0x64, opcode0x65, opcode0x66, opcode0x67, opcode0x68, opcode0x69, opcode0x6A, opcode0x6B, opcode0x6C, opcode0x6D, opcode0x6E, opcode0x6F,
	opcode0x70, opcode0x71, opcode0x72, opcode0x73, opcode0x74, opcode0x75, opcode0x76, opcode0x77, opcode0x78, opcode0x79, opcode0x7A, opcode0x7B, opcode0x7C, opcode0x7D, opcode0x7E, opcode0x7F,
	opcode0x80, opcode0x81, opcode0x82, opcode0x83, opcode0x84, opcode0x85, opcode0x86, opcode0x87, opcode0x88, opcode0x89, opcode0x8A, opcode0x8B, opcode0x8C, opcode0x8D, opcode0x8E, opcode0x8F,
	opcode0x90, opcode0x91, opcode0x92, opcode0x93, opcode0x94, opcode0x95, opcode0x96, opcode0x97, opcode0x98, opcode0x99, opcode0x9A, opcode0x9B, opcode0x9C, opcode0x9D, opcode0x9E, opcode0x9F,
	opcode0xA0, opcode0xA1, opcode0xA2, opcode0xA3, opcode0xA4, opcode0xA5, opcode0xA6, opcode0xA7, opcode0xA8, opcode0xA9, opcode0xAA, opcode0xAB, opcode0xAC, opcode0xAD, opcode0xAE, opcode0xAF,
	opcode0xB0, opcode0xB1, opcode0xB2, opcode0xB3, opcode0xB4, opcode0xB5, opcode0xB6, opcode0xB7, opcode0xB8, opcode0xB9, opcode0xBA, opcode0xBB, opcode0xBC, opcode0xBD, opcode0xBE, opcode0xBF,
	opcode0xC0, opcode0xC1, opcode0xC2, opcode0xC3, opcode0xC4, opcode0xC5, opcode0xC6, opcode0xC7, opcode0xC8, opcode0xC9, opcode0xCA, opcode0xCB, opcode0xCC, opcode0xCD, opcode0xCE, opcode0xCF,
	opcode0xD0, opcode0xD1, opcode0xD2, opcode0xD3, opcode0xD4, opcode0xD5, opcode0xD6, opcode0xD7, opcode0xD8, opcode0xD9, opcode0xDA, opcode0xDB, opcode0xDC, opcode0xDD, opcode0xDE, opcode0xDF,
	opcode0xE0, opcode0xE1, opcode0xE2, opcode0xE3, opcode0xE4, opcode0xE5, opcode0xE6, opcode0xE7, opcode0xE8, opcode0xE9, opcode0xEA, opcode0xEB, opcode0xEC, opcode0xED, opcode0xEE, opcode0xEF,
	opcode0xF0, opcode0xF1, opcode0xF2, opcode0xF3, opcode0xF4, opcode0xF5, opcode0xF6, opcode0xF7, opcode0xF8, opcode0xF9, opcode0xFA, opcode0xFB, opcode0xFC, opcode0xFD, opcode0xFE, opcode0xFF,
}

var opcodesCB = [...]Opcode{
	opcode0xCB00, opcode0xCB01, opcode0xCB02, opcode0xCB03, opcode0xCB04, opcode0xCB05, opcode0xCB06, opcode0xCB07, opcode0xCB08, opcode0xCB09, opcode0xCB0A, opcode0xCB0B, opcode0xCB0C, opcode0xCB0D, opcode0xCB0E, opcode0xCB0F,
	opcode0xCB10, opcode0xCB11, opcode0xCB12, opcode0xCB13, opcode0xCB14, opcode0xCB15, opcode0xCB16, opcode0xCB17, opcode0xCB18, opcode0xCB19, opcode0xCB1A, opcode0xCB1B, opcode0xCB1C, opcode0xCB1D, opcode0xCB1E, opcode0xCB1F,
	opcode0xCB20, opcode0xCB21, opcode0xCB22, opcode0xCB23, opcode0xCB24, opcode0xCB25, opcode0xCB26, opcode0xCB27, opcode0xCB28, opcode0xCB29, opcode0xCB2A, opcode0xCB2B, opcode0xCB2C, opcode0xCB2D, opcode0xCB2E, opcode0xCB2F,
	opcode0xCB30, opcode0xCB31, opcode0xCB32, opcode0xCB33, opcode0xCB34, opcode0xCB35, opcode0xCB36, opcode0xCB37, opcode0xCB38, opcode0xCB39, opcode0xCB3A, opcode0xCB3B, opcode0xCB3C, opcode0xCB3D, opcode0xCB3E, opcode0xCB3F,
	opcode0xCB40, opcode0xCB41, opcode0xCB42, opcode0xCB43, opcode0xCB44, opcode0xCB45, opcode0xCB46, opcode0xCB47, opcode0xCB48, opcode0xCB49, opcode0xCB4A, opcode0xCB4B, opcode0xCB4C, opcode0xCB4D, opcode0xCB4E, opcode0xCB4F,
	opcode0xCB50, opcode0xCB51, opcode0xCB52, opcode0xCB53, opcode0xCB54, opcode0xCB55, opcode0xCB56, opcode0xCB57, opcode0xCB58, opcode0xCB59, opcode0xCB5A, opcode0xCB5B, opcode0xCB5C, opcode0xCB5D, opcode0xCB5E, opcode0xCB5F,
	opcode0xCB60, opcode0xCB61, opcode0xCB62, opcode0xCB63, opcode0xCB64, opcode0xCB65, opcode0xCB66, opcode0xCB67, opcode0xCB68, opcode0xCB69, opcode0xCB6A, opcode0xCB6B, opcode0xCB6C, opcode0xCB6D, opcode0xCB6E, opcode0xCB6F,
	opcode0xCB70, opcode0xCB71, opcode0xCB72, opcode0xCB73, opcode0xCB74, opcode0xCB75, opcode0xCB76, opcode0xCB77, opcode0xCB78, opcode0xCB79, opcode0xCB7A, opcode0xCB7B, opcode0xCB7C, opcode0xCB7D, opcode0xCB7E, opcode0xCB7F,
	opcode0xCB80, opcode0xCB81, opcode0xCB82, opcode0xCB83, opcode0xCB84, opcode0xCB85, opcode0xCB86, opcode0xCB87, opcode0xCB88, opcode0xCB89, opcode0xCB8A, opcode0xCB8B, opcode0xCB8C, opcode0xCB8D, opcode0xCB8E, opcode0xCB8F,
	opcode0xCB90, opcode0xCB91, opcode0xCB92, opcode0xCB93, opcode0xCB94, opcode0xCB95, opcode0xCB96, opcode0xCB97, opcode0xCB98, opcode0xCB99, opcode0xCB9A, opcode0xCB9B, opcode0xCB9C, opcode0xCB9D, opcode0xCB9E, opcode0xCB9F,
	opcode0xCBA0, opcode0xCBA1, opcode0xCBA2, opcode0xCBA3, opcode0xCBA4, opcode0xCBA5, opcode0xCBA6, opcode0xCBA7, opcode0xCBA8, opcode0xCBA9, opcode0xCBAA, opcode0xCBAB, opcode0xCBAC, opcode0xCBAD, opcode0xCBAE, opcode0xCBAF,
	opcode0xCBB0, opcode0xCBB1, opcode0xCBB2, opcode0xCBB3, opcode0xCBB4, opcode0xCBB5, opcode0xCBB6, opcode0xCBB7, opcode0xCBB8, opcode0xCBB9, opcode0xCBBA, opcode0xCBBB, opcode0xCBBC, opcode0xCBBD, opcode0xCBBE, opcode0xCBBF,
	opcode0xCBC0, opcode0xCBC1, opcode0xCBC2, opcode0xCBC3, opcode0xCBC4, opcode0xCBC5, opcode0xCBC6, opcode0xCBC7, opcode0xCBC8, opcode0xCBC9, opcode0xCBCA, opcode0xCBCB, opcode0xCBCC, opcode0xCBCD, opcode0xCBCE, opcode0xCBCF,
	opcode0xCBD0, opcode0xCBD1, opcode0xCBD2, opcode0xCBD3, opcode0xCBD4, opcode0xCBD5, opcode0xCBD6, opcode0xCBD7, opcode0xCBD8, opcode0xCBD9, opcode0xCBDA, opcode0xCBDB, opcode0xCBDC, opcode0xCBDD, opcode0xCBDE, opcode0xCBDF,
	opcode0xCBE0, opcode0xCBE1, opcode0xCBE2, opcode0xCBE3, opcode0xCBE4, opcode0xCBE5, opcode0xCBE6, opcode0xCBE7, opcode0xCBE8, opcode0xCBE9, opcode0xCBEA, opcode0xCBEB, opcode0xCBEC, opcode0xCBED, opcode0xCBEE, opcode0xCBEF,
	opcode0xCBF0, opcode0xCBF1, opcode0xCBF2, opcode0xCBF3, opcode0xCBF4, opcode0xCBF5, opcode0xCBF6, opcode0xCBF7, opcode0xCBF8, opcode0xCBF9, opcode0xCBFA, opcode0xCBFB, opcode0xCBFC, opcode0xCBFD, opcode0xCBFE, opcode0xCBFF,
}

// GetOpcodeName returns a string with the opcode name and immediate values
func GetOpcodeName(c *CPU) string {
	code := c.bus.Read(c.pc)

	// 0xCB is only ever used as a prefix for the next byte.
	if code == 0xCB {
		code = c.bus.Read(c.pc + 1)
		n := c.bus.Read(c.pc + 2)
		nn := bit.Combine(c.bus.Read(c.pc+3), n)
		return fmt.Sprintf("0xcb%x (%s) n=0x%x nn=0x%x", code, opcodeNamesCB[code], n, nn)
	}

	n := c.bus.Read(c.pc + 1)
	nn := bit.Combine(c.bus.Read(c.pc+2), n)
	return fmt.Sprintf("0x%x (%s) n=0x%x nn=0x%x", code, opcodeNames[code], n, nn)
}

var opcodeNames = [...]string{
	"NOP", "LD BC,nn", "LD (BC),A", "INC BC", "INC B", "DEC B", "LD B,n", "RLCA", "LD (nn),SP", "ADD HL,BC", "LD A,(BC)", "DEC BC", "INC C", "DEC C", "LD C,n", "RRCA",
	"STOP", "LD DE,nn", "LD (DE),A", "INC DE", "INC D", "DEC D", "LD D,n", "RLA", "JR n", "ADD HL,DE", "LD A,(DE)", "DEC DE", "INC E", "DEC E", "LD E,n", "RRA",
	"JR NZ,n", "LD HL,nn", "LD (HLI),A", "INC HL", "INC H", "DEC H", "LD H,n", "DAA", "JR Z,n", "ADD HL,HL", "LD A,(HLI)", "DEC HL", "INC L", "DEC L", "LD L,n", "CPL",
	"JR NC,n", "LD SP,nn", "LD (HLD),A", "INC SP", "INC (HL)", "DEC (HL)", "LD (HL),n", "SCF", "JR C,n", "ADD HL,SP", "LD A,(HLD)", "DEC SP", "INC A", "DEC A", "LDA,n", "CCF",
	"LD B,B", "LD B,C", "LD B,D", "LD B,E", "LD B,H", "LD B,L", "LD B,(HL)", "LD B,A", "LD C,B", "LD C,C", "LD C,D", "LD C,E", "LD C,H", "LD C,L", "LD C,(HL)", "LD C,A",
	"LD D,B", "LD D,C", "LD D,D", "LD D,E", "LD D,H", "LD D,L", "LD D,(HL)", "LD D,A", "LD E,B", "LD E,C", "LD E,D", "LD E,E", "LD E,H", "LD E,L", "LD E,(HL)", "LD E,A",
	"LD H,B", "LD H,C", "LD H,D", "LD H,E", "LD H,H", "LD H,L", "LD H,(HL)", "LD H,A", "LD L,B", "LD L,C", "LD L,D", "LD L,E", "LD L,H", "LD L,L", "LD L,(HL)", "LD L,A",
	"LD (HL),B", "LD (HL),C", "LD (HL),D", "LD (HL),E", "LD (HL),H", "LD (HL),L", "HALT", "LD (HL),A", "LD A,B", "LD A,C", "LD A,D", "LD A,E", "LD A,H", "LD A,L", "LD A,(HL)", "LD A,A",
	"ADD A,B", "ADD A,C", "ADD A,D", "ADD A,E", "ADD A,H", "ADD A,L", "ADD A,(HL)", "ADD A,A", "ADC A,B", "ADC A,C", "ADC A,D", "ADC A,E", "ADC A,H", "ADC A,L", "ADC A,(HL)", "ADC A,A",
	"SUB B", "SUB C", "SUB D", "SUB E", "SUB H", "SUB L", "SUB (HL)", "SUB A", "SBC A,B", "SBC A,C", "SBC A,D", "SBC A,E", "SBC A,H", "SBC A,L", "SBC A,(HL)", "SBC A,A",
	"AND B", "AND C", "AND D", "AND E", "AND H", "AND L", "AND (HL)", "AND A", "XOR B", "XOR C", "XOR D", "XOR E", "XOR H", "XOR L", "XOR (HL)", "XOR A",
	"OR B", "OR C", "OR D", "OR E", "OR H", "OR L", "OR (HL)", "OR A", "CP B", "CP C", "CP D", "CP E", "CP H", "CP L", "CP (HL)", "CP A",
	"RET NZ", "POP BC", "JP NZ,nn", "JP nn", "CALL NZ,nn", "PUSH BC", "ADD A,n", "RST ", "RET Z", "RET", "JP Z,nn", "cb opcode", "CALL Z,nn", "CALL nn", "ADC A,n", "RST 0x08",
	"RET NC", "POP DE", "JP NC,nn", "unused opcode", "CALL NC,nn", "PUSH DE", "SUB n", "RST 0x10", "RET C", "RETI", "JP C,nn", "unused opcode", "CALL C,nn", "unused opcode", "SBC A,n", "RST 0x18",
	"LD (0xFF00+n),A", "POP HL", "LD (0xFF00+C),A", "unused opcode", "unused opcode", "PUSH HL", "AND n", "RST 0x20", "ADD SP,n", "JP (HL)", "LD (nn),A", "unused opcode", "unused opcode", "unused opcode", "XOR n", "RST 0x28",
	"LD A,(0xFF00+n)", "POP AF", "LD A,(0xFF00+C)", "DI", "unused opcode", "PUSH AF", "OR n", "RST 0x30", "LD HL,SP", "LD SP,HL", "LD A,(nn)", "EI", "unused opcode", "unused opcode", "CP n", "RST 0x38",
}

var opcodeNamesCB = [...]string{
	"RLC B", "RLC C", "RLC D", "RLC E", "RLC H", "RLC L", "RLC (HL)", "RLC A", "RRC B", "RRC C", "RRC D", "RRC E", "RRC H", "RRC L", "RRC (HL)", "RRC A",
	"RL B", "RL C", "RL D", "RL E", "RL H", "RL L ", "RL (HL)", "RL A", "RR B", "RR C", "RR D", "RR E", "RR H", "RR L", "RR (HL)", "RR A",
	"SLA B", "SLA C", "SLA D", "SLA E", "SLA H", "SLA L", "SLA (HL)", "SLA A", "SRA B", "SRA C", "SRA D", "SRA E", "SRA H", "SRA L", "SRA (HL)", "SRA A",
	"SWAP B", "SWAP C", "SWAP D", "SWAP E", "SWAP H", "SWAP L", "SWAP (HL)", "SWAP A", "SRL B", "SRL C", "SRL D", "SRL E", "SRL H", "SRL L", "SRL (HL)", "SRL A",
	"BIT 0 B", "BIT 0 C", "BIT 0 D", "BIT 0 E", "BIT 0 H", "BIT 0 L", "BIT 0 (HL)", "BIT 0 A", "BIT 1 B", "BIT 1 C", "BIT 1 D", "BIT 1 E", "BIT 1 H", "BIT 1 L", "BIT 1 (HL)", "BIT 1 A",
	"BIT 2 B", "BIT 2 C", "BIT 2 D", "BIT 2 E", "BIT 2 H", "BIT 2 L", "BIT 2 (HL)", "BIT 2 A", "BIT 3 B", "BIT 3 C", "BIT 3 D", "BIT 3 E", "BIT 3 H", "BIT 3 L", "BIT 3 (HL)", "BIT 3 A",
	"BIT 4 B", "BIT 4 C", "BIT 4 D", "BIT 4 E", "BIT 4 H", "BIT 4 L", "BIT 4 (HL)", "BIT 4 A", "BIT 5 B", "BIT 5 C", "BIT 5 D", "BIT 5 E", "BIT 5 H", "BIT 5 L", "BIT 5 (HL)", "BIT 5 A",
	"BIT 6 B", "BIT 6 C", "BIT 6 D", "BIT 6 E", "BIT 6 H", "BIT 6 L", "BIT 6 (HL)", "BIT 6 A", "BIT 7 B", "BIT 7 C", "BIT 7 D", "BIT 7 E", "BIT 7 H", "BIT 7 L", "BIT 7 (HL)", "BIT 7 A",
	"RES 0 B", "RES 0 C", "RES 0 D", "RES 0 E", "RES 0 H", "RES 0 L", "RES 0 (HL)", "RES 0 A", "RES 1 B", "RES 1 C", "RES 1 D", "RES 1 E", "RES 1 H", "RES 1 L", "RES 1 (HL)", "RES 1 A",
	"RES 2 B", "RES 2 C", "RES 2 D", "RES 2 E", "RES 2 H", "RES 2 L", "RES 2 (HL)", "RES 2 A", "RES 3 B", "RES 3 C", "RES 3 D", "RES 3 E", "RES 3 H", "RES 3 L", "RES 3 (HL)", "RES 3 A",
	"RES 4 B", "RES 4 C", "RES 4 D", "RES 4 E", "RES 4 H", "RES 4 L", "RES 4 (HL)", "RES 4 A", "RES 5 B", "RES 5 C", "RES 5 D", "RES 5 E", "RES 5 H", "RES 5 L", "RES 5 (HL)", "RES 5 A",
	"RES 6 B", "RES 6 C", "RES 6 D", "RES 6 E", "RES 6 H", "RES 6 L", "RES 6 (HL)", "RES 6 A", "RES 7 B", "RES 7 C", "RES 7 D", "RES 7 E", "RES 7 H", "RES 7 L", "RES 7 (HL)", "RES 7 A",
	"SET 0 B", "SET 0 C", "SET 0 D", "SET 0 E", "SET 0 H", "SET 0 L", "SET 0 (HL)", "SET 0 A", "SET 1 B", "SET 1 C", "SET 1 D", "SET 1 E", "SET 1 H", "SET 1 L", "SET 1 (HL)", "SET 1 A",
	"SET 2 B", "SET 2 C", "SET 2 D", "SET 2 E", "SET 2 H", "SET 2 L", "SET 2 (HL)", "SET 2 A", "SET 3 B", "SET 3 C", "SET 3 D", "SET 3 E", "SET 3 H", "SET 3 L", "SET 3 (HL)", "SET 3 A",
	"SET 4 B", "SET 4 C", "SET 4 D", "SET 4 E", "SET 4 H", "SET 4 L", "SET 4 (HL)", "SET 4 A", "SET 5 B", "SET 5 C", "SET 5 D", "SET 5 E", "SET 5 H", "SET 5 L", "SET 5 (HL)", "SET 5 A",
	"SET 6 B", "SET 6 C", "SET 6 D", "SET 6 E", "SET 6 H", "SET 6 L", "SET 6 (HL)", "SET 6 A", "SET 7 B", "SET 7 C", "SET 7 D", "SET 7 E", "SET 7 H", "SET 7 L", "SET 7 (HL)", "SET 7 A",
}
