package cpu

import (
	"testing"
)

func TestRegister16_low(t *testing.T) {
	tests := []struct {
		name string
		r    Register16
		want Register8
	}{
		{"Returns low value", newRegister16(0xABCD), Register8(0xCD)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.r.low; got != tt.want {
				t.Errorf("Register16.low() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRegister16_high(t *testing.T) {
	tests := []struct {
		name string
		r    Register16
		want Register8
	}{
		{"Returns high value", newRegister16(0xABCD), Register8(0xAB)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.r.high; got != tt.want {
				t.Errorf("Register16.high = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRegister16_set(t *testing.T) {
	r := newRegister16(0xFFFF)
	r.set(0)

	if r.get() != 0 {
		t.Fail()
	}
}

func TestRegister16_setHigh(t *testing.T) {
	r := newRegister16(0xFFFF)
	r.setHigh(1)

	if r.get() != 0x01FF {
		t.Fail()
	}
}

func TestRegister16_setLow(t *testing.T) {
	r := newRegister16(0xFFFF)
	r.setLow(1)

	if r.get() != 0xFF01 {
		t.Fail()
	}
}

func TestRegister16_get(t *testing.T) {
	tests := []struct {
		name string
		r    Register16
		want uint16
	}{
		{"Gets the internal value", newRegister16(0xBEEF), 0xBEEF},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.r.get(); got != tt.want {
				t.Errorf("Register16.get() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRegister16_incr(t *testing.T) {
	r := newRegister16(0)
	r.incr()

	if r.get() != 1 {
		t.Fail()
	}

	r = newRegister16(0xFFFF)
	r.incr()

	if r.get() != 0 {
		t.Fail()
	}
}

func TestRegister16_decr(t *testing.T) {
	r := newRegister16(0xFFFF)
	r.decr()

	if r.get() != 0xFFFE {
		t.Fail()
	}

	r = newRegister16(0)
	r.decr()

	if r.get() != 0xFFFF {
		t.Fail()
	}
}

func TestRegister8_get(t *testing.T) {
	tests := []struct {
		name string
		r    Register8
		want uint8
	}{
		{"Gets the internal value", Register8(0xAE), 0xAE},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.r.get(); got != tt.want {
				t.Errorf("Register8.get() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRegister8_set(t *testing.T) {
	r := Register8(0xFF)
	r.set(0x10)

	if r != 0x10 {
		t.Fail()
	}
}

func TestRegister8_incr(t *testing.T) {
	r := Register8(0)
	r.incr()

	if r != 1 {
		t.Fail()
	}

	r = Register8(0xFF)
	r.incr()

	if r != 0 {
		t.Fail()
	}
}

func TestRegister8_decr(t *testing.T) {
	r := Register8(0xFF)
	r.decr()

	if r != 0xFE {
		t.Fail()
	}

	r = Register8(0)
	r.decr()

	if r != 0xFF {
		t.Fail()
	}
}

func TestRegister16_getLow(t *testing.T) {
	tests := []struct {
		name string
		r    Register16
		want uint8
	}{
		{"Gets the low value as uint8", newRegister16(0xCAFE), 0xFE},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.r.getLow(); got != tt.want {
				t.Errorf("Register16.getLow() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRegister16_getHigh(t *testing.T) {
	tests := []struct {
		name string
		r    Register16
		want uint8
	}{
		{"Gets the high value as uint8", newRegister16(0xCAFE), 0xCA},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.r.getHigh(); got != tt.want {
				t.Errorf("Register16.getHigh = %v, want %v", got, tt.want)
			}
		})
	}
}
