package util

import (
	"testing"
)

func TestRegister16_low(t *testing.T) {
	tests := []struct {
		name string
		r    Register16
		want Register8
	}{
		{"Returns Low value", newRegister16(0xABCD), Register8(0xCD)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.r.Low; got != tt.want {
				t.Errorf("Register16.Low() = %v, want %v", got, tt.want)
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
		{"Returns High value", newRegister16(0xABCD), Register8(0xAB)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.r.High; got != tt.want {
				t.Errorf("Register16.High = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRegister16_set(t *testing.T) {
	r := newRegister16(0xFFFF)
	r.Set(0)

	if r.Get() != 0 {
		t.Fail()
	}
}

func TestRegister16_setHigh(t *testing.T) {
	r := newRegister16(0xFFFF)
	r.SetHigh(1)

	if r.Get() != 0x01FF {
		t.Fail()
	}
}

func TestRegister16_setLow(t *testing.T) {
	r := newRegister16(0xFFFF)
	r.SetLow(1)

	if r.Get() != 0xFF01 {
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
			if got := tt.r.Get(); got != tt.want {
				t.Errorf("Register16.Get() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRegister16_incr(t *testing.T) {
	r := newRegister16(0)
	r.Incr()

	if r.Get() != 1 {
		t.Fail()
	}

	r = newRegister16(0xFFFF)
	r.Incr()

	if r.Get() != 0 {
		t.Fail()
	}
}

func TestRegister16_decr(t *testing.T) {
	r := newRegister16(0xFFFF)
	r.Decr()

	if r.Get() != 0xFFFE {
		t.Fail()
	}

	r = newRegister16(0)
	r.Decr()

	if r.Get() != 0xFFFF {
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
			if got := tt.r.Get(); got != tt.want {
				t.Errorf("Register8.Get() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRegister8_set(t *testing.T) {
	r := Register8(0xFF)
	r.Set(0x10)

	if r != 0x10 {
		t.Fail()
	}
}

func TestRegister8_incr(t *testing.T) {
	r := Register8(0)
	r.Incr()

	if r != 1 {
		t.Fail()
	}

	r = Register8(0xFF)
	r.Incr()

	if r != 0 {
		t.Fail()
	}
}

func TestRegister8_decr(t *testing.T) {
	r := Register8(0xFF)
	r.Decr()

	if r != 0xFE {
		t.Fail()
	}

	r = Register8(0)
	r.Decr()

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
		{"Gets the Low value as uint8", newRegister16(0xCAFE), 0xFE},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.r.GetLow(); got != tt.want {
				t.Errorf("Register16.GetLow() = %v, want %v", got, tt.want)
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
		{"Gets the High value as uint8", newRegister16(0xCAFE), 0xCA},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.r.GetHigh(); got != tt.want {
				t.Errorf("Register16.GetHigh = %v, want %v", got, tt.want)
			}
		})
	}
}
