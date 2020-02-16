package memory

import (
	"log"

	"github.com/valerio/go-jeebie/jeebie/bit"
)

const titleLength = 11

const (
	entryPointAddress       = 0x100
	logoAddress             = 0x104
	titleAddress            = 0x134
	manufacturerCodeAddress = 0x13F
	cgbFlagAddress          = 0x143
	newLicenseCodeAddress   = 0x144
	sgbFlagAddress          = 0x146
	cartridgeTypeAddress    = 0x147
	romSizeAddress          = 0x148
	ramSizeAddress          = 0x149
	destinationCodeAddress  = 0x14A
	oldLicenseCodeAddress   = 0x14B
	versionNumberAddress    = 0x14C
	headerChecksumAddress   = 0x14D
	globalChecksumAddress   = 0x14E
)

type MBCType int

const (
	NoMBC      MBCType = iota
	MBC1               = iota
	MBC2               = iota
	MBC3               = iota
	MBC5               = iota
	MBC1Multi          = iota
	MBCUnknown         = iota
)

// Cartridge holds the data and metadata of a gameboy cartridge.
type Cartridge struct {
	data           []byte
	title          string
	headerChecksum uint16
	globalChecksum uint16
	version        uint8
	cartType       uint8
	romSize        uint8
	ramSize        uint8
	isCGB          bool
	isSGB          bool
	mbcType        MBCType
	hasRTC         bool
	hasRumble      bool
	hasBattery     bool
	ramBankCount   uint8
}

// NewCartridge creates an empty cartridge, useful only for debugging purposes.
func NewCartridge() *Cartridge {
	return &Cartridge{
		data: make([]byte, 0x10000),
	}
}

// NewCartridgeWithData initializes a new Cartridge from a slice of bytes.
func NewCartridgeWithData(bytes []byte) *Cartridge {
	// load cartridge title
	title := string(bytes[titleAddress : titleAddress+titleLength])

	// determine if cart is for gameboy color (CGB)
	isCGB := bytes[cgbFlagAddress] == 0x80 || bytes[cgbFlagAddress] == 0xC0
	// determine if cart is for super gameboy (SGB)
	isSGB := bytes[sgbFlagAddress] == 0x03

	cartType := bytes[cartridgeTypeAddress]
	romSize := bytes[romSizeAddress]
	ramSize := bytes[ramSizeAddress]
	version := bytes[versionNumberAddress]

	mbcType := getMBCType(cartType)
	hasRTC := hasRealTimeClock(cartType)
	hasRumble := hasRumble(cartType)
	hasBattery := hasBattery(cartType)

	ramBankCount := getRAMBankCount(ramSize, mbcType)

	data := make([]byte, len(bytes))
	copy(data, bytes)

	cart := &Cartridge{
		data:           data,
		title:          title,
		headerChecksum: bit.Combine(bytes[headerChecksumAddress], bytes[headerChecksumAddress+1]),
		globalChecksum: bit.Combine(bytes[globalChecksumAddress], bytes[globalChecksumAddress+1]),
		version:        version,
		cartType:       cartType,
		romSize:        romSize,
		ramSize:        ramSize,
		isCGB:          isCGB,
		isSGB:          isSGB,
		mbcType:        mbcType,
		hasRTC:         hasRTC,
		hasRumble:      hasRumble,
		hasBattery:     hasBattery,
		ramBankCount:   ramBankCount,
	}

	isValid := isValidCheckSum(bytes[titleAddress:globalChecksumAddress])
	if !isValid {
		log.Fatalln("Cartridge has invalid checksum.")
	}

	log.Printf("Cartridge loaded: %s\n", cart.title)

	return cart
}

// ReadByte reads a byte at the specified address. Does not check bounds, so the caller must make sure the
// address is valid for the cartridge.
func (c *Cartridge) Read(addr uint16) uint8 {
	return c.data[addr]
}

// WriteByte attempts a write to the specified address. Writing to a cartridge has sense if the cartridge
// has extra RAM or for some special operations, like switching ROM banks.
func (c *Cartridge) Write(addr uint16, value uint8) uint8 {
	return c.data[addr]
}

func getRAMBankCount(ramSize uint8, mbcType MBCType) uint8 {
	switch ramSize {
	case 0x00:
		if mbcType == MBC2 {
			return 1
		}
		return 0
	case 0x01:
	case 0x02:
		return 1
	case 0x04:
		return 16
	}

	return 4
}

func getMBCType(cartType uint8) MBCType {
	switch cartType {
	case 0x00:
	case 0x08:
	case 0x09:
		return NoMBC
	case 0x01:
	case 0x02:
	case 0x03:
	case 0xEA:
	case 0xFF:
		return MBC1
	case 0x05:
	case 0x06:
		return MBC2
	case 0x0F:
	case 0x10:
	case 0x11:
	case 0x12:
	case 0x13:
	case 0xFC:
		return MBC3
	case 0x19:
	case 0x1A:
	case 0x1B:
	case 0x1C:
	case 0x1D:
	case 0x1E:
		return MBC5
	}

	return MBCUnknown
}

func hasRumble(cartType uint8) bool {
	switch cartType {
	case 0x1C:
	case 0x1D:
	case 0x1E:
		return true
	}
	return false
}

func hasRealTimeClock(cartType uint8) bool {
	switch cartType {
	case 0x0F:
	case 0x10:
		return true
	}
	return false
}

func hasBattery(cartType uint8) bool {
	switch cartType {
	case 0x03:
	case 0x06:
	case 0x09:
	case 0x0D:
	case 0x0F:
	case 0x10:
	case 0x13:
	case 0x17:
	case 0x1B:
	case 0x1E:
	case 0x22:
	case 0xFD:
	case 0xFF:
		return true
	}

	return false
}

func isValidCheckSum(bytes []byte) bool {
	checksum := 0

	for _, n := range bytes {
		checksum += int(n)
	}

	isValid := ((checksum + 25) & 0xFF) == 0

	return isValid
}
