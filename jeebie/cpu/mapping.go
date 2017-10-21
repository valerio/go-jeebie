package cpu

// Opcode represents a function that executes an opcode
type Opcode func(*CPU) int

// Decode takes a byte and retrieves the corresponding instruction
func decode(opcode uint16) Opcode {
	if (opcode & 0xCB00) == 0xCB00 {
		return opcodeCBMap[uint8(opcode&0xFF)]
	}

	return opcodeMap[uint8(opcode&0xFF)]
}

var opcodeMap = map[uint8]Opcode{
	0x00: opcode0x00,
	0x01: opcode0x01,
	0x02: opcode0x02,
	0x03: opcode0x03,
	0x04: opcode0x04,
	0x05: opcode0x05,
	0x06: opcode0x06,
	0x07: opcode0x07,
	0x08: opcode0x08,
	0x09: opcode0x09,
	0x0a: opcode0x0A,
	0x0b: opcode0x0B,
	0x0c: opcode0x0C,
	0x0d: opcode0x0D,
	0x0e: opcode0x0E,
	0x0f: opcode0x0F,
	0x10: opcode0x10,
	0x11: opcode0x11,
	0x12: opcode0x12,
	0x13: opcode0x13,
	0x14: opcode0x14,
	0x15: opcode0x15,
	0x16: opcode0x16,
	0x17: opcode0x17,
	0x18: opcode0x18,
	0x19: opcode0x19,
	0x1a: opcode0x1A,
	0x1b: opcode0x1B,
	0x1c: opcode0x1C,
	0x1d: opcode0x1D,
	0x1e: opcode0x1E,
	0x1f: opcode0x1F,
	0x20: opcode0x20,
	0x21: opcode0x21,
	0x22: opcode0x22,
	0x23: opcode0x23,
	0x24: opcode0x24,
	0x25: opcode0x25,
	0x26: opcode0x26,
	0x27: opcode0x27,
	0x28: opcode0x28,
	0x29: opcode0x29,
	0x2a: opcode0x2A,
	0x2b: opcode0x2B,
	0x2c: opcode0x2C,
	0x2d: opcode0x2D,
	0x2e: opcode0x2E,
	0x2f: opcode0x2F,
	0x30: opcode0x30,
	0x31: opcode0x31,
	0x32: opcode0x32,
	0x33: opcode0x33,
	0x34: opcode0x34,
	0x35: opcode0x35,
	0x36: opcode0x36,
	0x37: opcode0x37,
	0x38: opcode0x38,
	0x39: opcode0x39,
	0x3a: opcode0x3A,
	0x3b: opcode0x3B,
	0x3c: opcode0x3C,
	0x3d: opcode0x3D,
	0x3e: opcode0x3E,
	0x3f: opcode0x3F,
	0x40: opcode0x40,
	0x41: opcode0x41,
	0x42: opcode0x42,
	0x43: opcode0x43,
	0x44: opcode0x44,
	0x45: opcode0x45,
	0x46: opcode0x46,
	0x47: opcode0x47,
	0x48: opcode0x48,
	0x49: opcode0x49,
	0x4a: opcode0x4A,
	0x4b: opcode0x4B,
	0x4c: opcode0x4C,
	0x4d: opcode0x4D,
	0x4e: opcode0x4E,
	0x4f: opcode0x4F,
	0x50: opcode0x50,
	0x51: opcode0x51,
	0x52: opcode0x52,
	0x53: opcode0x53,
	0x54: opcode0x54,
	0x55: opcode0x55,
	0x56: opcode0x56,
	0x57: opcode0x57,
	0x58: opcode0x58,
	0x59: opcode0x59,
	0x5a: opcode0x5A,
	0x5b: opcode0x5B,
	0x5c: opcode0x5C,
	0x5d: opcode0x5D,
	0x5e: opcode0x5E,
	0x5f: opcode0x5F,
	0x60: opcode0x60,
	0x61: opcode0x61,
	0x62: opcode0x62,
	0x63: opcode0x63,
	0x64: opcode0x64,
	0x65: opcode0x65,
	0x66: opcode0x66,
	0x67: opcode0x67,
	0x68: opcode0x68,
	0x69: opcode0x69,
	0x6a: opcode0x6A,
	0x6b: opcode0x6B,
	0x6c: opcode0x6C,
	0x6d: opcode0x6D,
	0x6e: opcode0x6E,
	0x6f: opcode0x6F,
	0x70: opcode0x70,
	0x71: opcode0x71,
	0x72: opcode0x72,
	0x73: opcode0x73,
	0x74: opcode0x74,
	0x75: opcode0x75,
	0x76: opcode0x76,
	0x77: opcode0x77,
	0x78: opcode0x78,
	0x79: opcode0x79,
	0x7a: opcode0x7A,
	0x7b: opcode0x7B,
	0x7c: opcode0x7C,
	0x7d: opcode0x7D,
	0x7e: opcode0x7E,
	0x7f: opcode0x7F,
	0x80: opcode0x80,
	0x81: opcode0x81,
	0x82: opcode0x82,
	0x83: opcode0x83,
	0x84: opcode0x84,
	0x85: opcode0x85,
	0x86: opcode0x86,
	0x87: opcode0x87,
	0x88: opcode0x88,
	0x89: opcode0x89,
	0x8a: opcode0x8A,
	0x8b: opcode0x8B,
	0x8c: opcode0x8C,
	0x8d: opcode0x8D,
	0x8e: opcode0x8E,
	0x8f: opcode0x8F,
	0x90: opcode0x90,
	0x91: opcode0x91,
	0x92: opcode0x92,
	0x93: opcode0x93,
	0x94: opcode0x94,
	0x95: opcode0x95,
	0x96: opcode0x96,
	0x97: opcode0x97,
	0x98: opcode0x98,
	0x99: opcode0x99,
	0x9a: opcode0x9A,
	0x9b: opcode0x9B,
	0x9c: opcode0x9C,
	0x9d: opcode0x9D,
	0x9e: opcode0x9E,
	0x9f: opcode0x9F,
	0xa0: opcode0xA0,
	0xa1: opcode0xA1,
	0xa2: opcode0xA2,
	0xa3: opcode0xA3,
	0xa4: opcode0xA4,
	0xa5: opcode0xA5,
	0xa6: opcode0xA6,
	0xa7: opcode0xA7,
	0xa8: opcode0xA8,
	0xa9: opcode0xA9,
	0xaa: opcode0xAA,
	0xab: opcode0xAB,
	0xac: opcode0xAC,
	0xad: opcode0xAD,
	0xae: opcode0xAE,
	0xaf: opcode0xAF,
	0xb0: opcode0xB0,
	0xb1: opcode0xB1,
	0xb2: opcode0xB2,
	0xb3: opcode0xB3,
	0xb4: opcode0xB4,
	0xb5: opcode0xB5,
	0xb6: opcode0xB6,
	0xb7: opcode0xB7,
	0xb8: opcode0xB8,
	0xb9: opcode0xB9,
	0xba: opcode0xBA,
	0xbb: opcode0xBB,
	0xbc: opcode0xBC,
	0xbd: opcode0xBD,
	0xbe: opcode0xBE,
	0xbf: opcode0xBF,
	0xc0: opcode0xC0,
	0xc1: opcode0xC1,
	0xc2: opcode0xC2,
	0xc3: opcode0xC3,
	0xc4: opcode0xC4,
	0xc5: opcode0xC5,
	0xc6: opcode0xC6,
	0xc7: opcode0xC7,
	0xc8: opcode0xC8,
	0xc9: opcode0xC9,
	0xca: opcode0xCA,
	0xcb: opcode0xCB,
	0xcc: opcode0xCC,
	0xcd: opcode0xCD,
	0xce: opcode0xCE,
	0xcf: opcode0xCF,
	0xd0: opcode0xD0,
	0xd1: opcode0xD1,
	0xd2: opcode0xD2,
	0xd3: opcode0xD3,
	0xd4: opcode0xD4,
	0xd5: opcode0xD5,
	0xd6: opcode0xD6,
	0xd7: opcode0xD7,
	0xd8: opcode0xD8,
	0xd9: opcode0xD9,
	0xda: opcode0xDA,
	0xdb: opcode0xDB,
	0xdc: opcode0xDC,
	0xdd: opcode0xDD,
	0xde: opcode0xDE,
	0xdf: opcode0xDF,
	0xe0: opcode0xE0,
	0xe1: opcode0xE1,
	0xe2: opcode0xE2,
	0xe3: opcode0xE3,
	0xe4: opcode0xE4,
	0xe5: opcode0xE5,
	0xe6: opcode0xE6,
	0xe7: opcode0xE7,
	0xe8: opcode0xE8,
	0xe9: opcode0xE9,
	0xea: opcode0xEA,
	0xeb: opcode0xEB,
	0xec: opcode0xEC,
	0xed: opcode0xED,
	0xee: opcode0xEE,
	0xef: opcode0xEF,
	0xf0: opcode0xF0,
	0xf1: opcode0xF1,
	0xf2: opcode0xF2,
	0xf3: opcode0xF3,
	0xf4: opcode0xF4,
	0xf5: opcode0xF5,
	0xf6: opcode0xF6,
	0xf7: opcode0xF7,
	0xf8: opcode0xF8,
	0xf9: opcode0xF9,
	0xfa: opcode0xFA,
	0xfb: opcode0xFB,
	0xfc: opcode0xFC,
	0xfd: opcode0xFD,
	0xfe: opcode0xFE,
	0xff: opcode0xFF,
}

var opcodeCBMap = map[uint8]Opcode{
	0x00: unimplemented,
	0x01: unimplemented,
	0x02: unimplemented,
	0x03: unimplemented,
	0x04: unimplemented,
	0x05: unimplemented,
	0x06: unimplemented,
	0x07: unimplemented,
	0x08: unimplemented,
	0x09: unimplemented,
	0x0a: unimplemented,
	0x0b: unimplemented,
	0x0c: unimplemented,
	0x0d: unimplemented,
	0x0e: unimplemented,
	0x0f: unimplemented,
	0x10: unimplemented,
	0x11: unimplemented,
	0x12: unimplemented,
	0x13: unimplemented,
	0x14: unimplemented,
	0x15: unimplemented,
	0x16: unimplemented,
	0x17: unimplemented,
	0x18: unimplemented,
	0x19: unimplemented,
	0x1a: unimplemented,
	0x1b: unimplemented,
	0x1c: unimplemented,
	0x1d: unimplemented,
	0x1e: unimplemented,
	0x1f: unimplemented,
	0x20: unimplemented,
	0x21: unimplemented,
	0x22: unimplemented,
	0x23: unimplemented,
	0x24: unimplemented,
	0x25: unimplemented,
	0x26: unimplemented,
	0x27: unimplemented,
	0x28: unimplemented,
	0x29: unimplemented,
	0x2a: unimplemented,
	0x2b: unimplemented,
	0x2c: unimplemented,
	0x2d: unimplemented,
	0x2e: unimplemented,
	0x2f: unimplemented,
	0x30: unimplemented,
	0x31: unimplemented,
	0x32: unimplemented,
	0x33: unimplemented,
	0x34: unimplemented,
	0x35: unimplemented,
	0x36: unimplemented,
	0x37: unimplemented,
	0x38: unimplemented,
	0x39: unimplemented,
	0x3a: unimplemented,
	0x3b: unimplemented,
	0x3c: unimplemented,
	0x3d: unimplemented,
	0x3e: unimplemented,
	0x3f: unimplemented,
	0x40: unimplemented,
	0x41: unimplemented,
	0x42: unimplemented,
	0x43: unimplemented,
	0x44: unimplemented,
	0x45: unimplemented,
	0x46: unimplemented,
	0x47: unimplemented,
	0x48: unimplemented,
	0x49: unimplemented,
	0x4a: unimplemented,
	0x4b: unimplemented,
	0x4c: unimplemented,
	0x4d: unimplemented,
	0x4e: unimplemented,
	0x4f: unimplemented,
	0x50: unimplemented,
	0x51: unimplemented,
	0x52: unimplemented,
	0x53: unimplemented,
	0x54: unimplemented,
	0x55: unimplemented,
	0x56: unimplemented,
	0x57: unimplemented,
	0x58: unimplemented,
	0x59: unimplemented,
	0x5a: unimplemented,
	0x5b: unimplemented,
	0x5c: unimplemented,
	0x5d: unimplemented,
	0x5e: unimplemented,
	0x5f: unimplemented,
	0x60: unimplemented,
	0x61: unimplemented,
	0x62: unimplemented,
	0x63: unimplemented,
	0x64: unimplemented,
	0x65: unimplemented,
	0x66: unimplemented,
	0x67: unimplemented,
	0x68: unimplemented,
	0x69: unimplemented,
	0x6a: unimplemented,
	0x6b: unimplemented,
	0x6c: unimplemented,
	0x6d: unimplemented,
	0x6e: unimplemented,
	0x6f: unimplemented,
	0x70: unimplemented,
	0x71: unimplemented,
	0x72: unimplemented,
	0x73: unimplemented,
	0x74: unimplemented,
	0x75: unimplemented,
	0x76: unimplemented,
	0x77: unimplemented,
	0x78: unimplemented,
	0x79: unimplemented,
	0x7a: unimplemented,
	0x7b: unimplemented,
	0x7c: unimplemented,
	0x7d: unimplemented,
	0x7e: unimplemented,
	0x7f: unimplemented,
	0x80: unimplemented,
	0x81: unimplemented,
	0x82: unimplemented,
	0x83: unimplemented,
	0x84: unimplemented,
	0x85: unimplemented,
	0x86: unimplemented,
	0x87: unimplemented,
	0x88: unimplemented,
	0x89: unimplemented,
	0x8a: unimplemented,
	0x8b: unimplemented,
	0x8c: unimplemented,
	0x8d: unimplemented,
	0x8e: unimplemented,
	0x8f: unimplemented,
	0x90: unimplemented,
	0x91: unimplemented,
	0x92: unimplemented,
	0x93: unimplemented,
	0x94: unimplemented,
	0x95: unimplemented,
	0x96: unimplemented,
	0x97: unimplemented,
	0x98: unimplemented,
	0x99: unimplemented,
	0x9a: unimplemented,
	0x9b: unimplemented,
	0x9c: unimplemented,
	0x9d: unimplemented,
	0x9e: unimplemented,
	0x9f: unimplemented,
	0xa0: unimplemented,
	0xa1: unimplemented,
	0xa2: unimplemented,
	0xa3: unimplemented,
	0xa4: unimplemented,
	0xa5: unimplemented,
	0xa6: unimplemented,
	0xa7: unimplemented,
	0xa8: unimplemented,
	0xa9: unimplemented,
	0xaa: unimplemented,
	0xab: unimplemented,
	0xac: unimplemented,
	0xad: unimplemented,
	0xae: unimplemented,
	0xaf: unimplemented,
	0xb0: unimplemented,
	0xb1: unimplemented,
	0xb2: unimplemented,
	0xb3: unimplemented,
	0xb4: unimplemented,
	0xb5: unimplemented,
	0xb6: unimplemented,
	0xb7: unimplemented,
	0xb8: unimplemented,
	0xb9: unimplemented,
	0xba: unimplemented,
	0xbb: unimplemented,
	0xbc: unimplemented,
	0xbd: unimplemented,
	0xbe: unimplemented,
	0xbf: unimplemented,
	0xc0: unimplemented,
	0xc1: unimplemented,
	0xc2: unimplemented,
	0xc3: unimplemented,
	0xc4: unimplemented,
	0xc5: unimplemented,
	0xc6: unimplemented,
	0xc7: unimplemented,
	0xc8: unimplemented,
	0xc9: unimplemented,
	0xca: unimplemented,
	0xcb: unimplemented,
	0xcc: unimplemented,
	0xcd: unimplemented,
	0xce: unimplemented,
	0xcf: unimplemented,
	0xd0: unimplemented,
	0xd1: unimplemented,
	0xd2: unimplemented,
	0xd3: unimplemented,
	0xd4: unimplemented,
	0xd5: unimplemented,
	0xd6: unimplemented,
	0xd7: unimplemented,
	0xd8: unimplemented,
	0xd9: unimplemented,
	0xda: unimplemented,
	0xdb: unimplemented,
	0xdc: unimplemented,
	0xdd: unimplemented,
	0xde: unimplemented,
	0xdf: unimplemented,
	0xe0: unimplemented,
	0xe1: unimplemented,
	0xe2: unimplemented,
	0xe3: unimplemented,
	0xe4: unimplemented,
	0xe5: unimplemented,
	0xe6: unimplemented,
	0xe7: unimplemented,
	0xe8: unimplemented,
	0xe9: unimplemented,
	0xea: unimplemented,
	0xeb: unimplemented,
	0xec: unimplemented,
	0xed: unimplemented,
	0xee: unimplemented,
	0xef: unimplemented,
	0xf0: unimplemented,
	0xf1: unimplemented,
	0xf2: unimplemented,
	0xf3: unimplemented,
	0xf4: unimplemented,
	0xf5: unimplemented,
	0xf6: unimplemented,
	0xf7: unimplemented,
	0xf8: unimplemented,
	0xf9: unimplemented,
	0xfa: unimplemented,
	0xfb: unimplemented,
	0xfc: unimplemented,
	0xfd: unimplemented,
	0xfe: unimplemented,
	0xff: unimplemented,
}
