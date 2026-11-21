// Package irproto turns IR codes into mark/space timings and back.
//
// Protocol names, and what "address" and "command" mean for each, follow the Flipper Zero .ir file format, so files
// from the Flipper-IRDB database can be sent as they are. The timings come from the public protocol descriptions
// (NEC, Samsung, RC5, RC6, Sony SIRC, Kaseikyo, RCA, Pioneer); no Flipper code is used.
package irproto

import (
	"fmt"
	"strings"
)

// Signal is one IR transmission: alternating durations in microseconds, starting with a mark (carrier on).
type Signal struct {
	Freq      int // carrier, Hz
	Durations []uint32
}

// Toggle state for RC5/RC6 (the toggle bit flips on every new key press).
var toggle = map[string]bool{}

// Protocols lists the protocol names Encode understands.
var Protocols = []string{"NEC", "NECext", "NEC42", "NEC42ext", "Samsung32", "RC5", "RC5X", "RC6", "SIRC", "SIRC15",
	"SIRC20", "Kaseikyo", "RCA", "Pioneer"}

// builder collects levels and merges equal neighbours.
type builder struct {
	d    []uint32
	mark bool // level of the last entry
}

func (b *builder) add(mark bool, us uint32) {
	if us == 0 {
		return
	}
	if len(b.d) == 0 {
		if !mark {
			return // a signal starts with a mark
		}
		b.d = append(b.d, us)
		b.mark = true
		return
	}
	if mark == b.mark {
		b.d[len(b.d)-1] += us
		return
	}
	b.d = append(b.d, us)
	b.mark = mark
}

// pulseDistance: header, then per bit (LSB first) a mark and a space whose length carries the bit, then a stop mark.
func pulseDistance(b *builder, hdrMark, hdrSpace, bitMark, zero, one uint32, bits uint64, n int) {
	b.add(true, hdrMark)
	b.add(false, hdrSpace)
	for i := 0; i < n; i++ {
		b.add(true, bitMark)
		if bits>>uint(i)&1 == 1 {
			b.add(false, one)
		} else {
			b.add(false, zero)
		}
	}
	b.add(true, bitMark)
}

func total(d []uint32) (t uint32) {
	for _, v := range d {
		t += v
	}
	return
}

// Encode builds the signal for a parsed code. Address and command are the values from the .ir file
// (4 bytes little endian there, e.g. "07 00 00 00" = 7).
func Encode(protocol string, addr, cmd uint32) (Signal, error) {
	var b builder
	freq := 38000
	switch protocol {
	case "NEC":
		a, c := uint64(addr&0xFF), uint64(cmd&0xFF)
		pulseDistance(&b, 9000, 4500, 560, 560, 1690, a|(^a&0xFF)<<8|c<<16|(^c&0xFF)<<24, 32)
	case "NECext":
		pulseDistance(&b, 9000, 4500, 560, 560, 1690, uint64(addr&0xFFFF)|uint64(cmd&0xFFFF)<<16, 32)
	case "NEC42":
		a, c := uint64(addr&0x1FFF), uint64(cmd&0xFF)
		pulseDistance(&b, 9000, 4500, 560, 560, 1690, a|(^a&0x1FFF)<<13|c<<26|(^c&0xFF)<<34, 42)
	case "NEC42ext":
		pulseDistance(&b, 9000, 4500, 560, 560, 1690, uint64(addr&0x3FFFFFF)|uint64(cmd&0xFFFF)<<26, 42)
	case "Samsung32":
		a, c := uint64(addr&0xFF), uint64(cmd&0xFF)
		pulseDistance(&b, 4500, 4500, 550, 550, 1650, a|a<<8|c<<16|(^c&0xFF)<<24, 32)
	case "RCA":
		a, c := uint64(addr&0xF), uint64(cmd&0xFF)
		pulseDistance(&b, 4000, 4000, 500, 1000, 2000, a|c<<4|(^a&0xF)<<12|(^c&0xFF)<<16, 24)
	case "Pioneer":
		a, c := uint64(addr&0xFF), uint64(cmd&0xFF)
		var one builder
		pulseDistance(&one, 8500, 4225, 500, 500, 1500, a|(^a&0xFF)<<8|c<<16|(^c&0xFF)<<24, 33)
		frame := one.d
		for r := 0; r < 2; r++ { // sent twice
			if r > 0 {
				b.add(false, 26000)
			}
			for i, v := range frame {
				b.add(i%2 == 0, v)
			}
		}
		freq = 40000
	case "Kaseikyo":
		id := uint64(addr>>24) & 3
		vendor := uint64(addr>>8) & 0xFFFF
		g1, g2 := uint64(addr>>4)&0xF, uint64(addr)&0xF
		c := uint64(cmd & 0x3FF)
		var by [6]uint64
		by[0], by[1] = vendor&0xFF, vendor>>8
		par := by[0] ^ by[1]
		par = (par & 0xF) ^ (par >> 4)
		by[2] = par&0xF | g1<<4
		by[3] = g2 | (c&0xF)<<4
		by[4] = id<<6 | c>>4
		by[5] = by[2] ^ by[3] ^ by[4]
		var bits uint64
		for i, v := range by {
			bits |= (v & 0xFF) << (8 * uint(i))
		}
		u := uint32(432)
		pulseDistance(&b, 8*u, 4*u, u, u, 3*u, bits, 48)
	case "SIRC", "SIRC15", "SIRC20":
		n, amask := 12, uint64(0x1F)
		if protocol == "SIRC15" {
			n, amask = 15, 0xFF
		} else if protocol == "SIRC20" {
			n, amask = 20, 0x1FFF
		}
		bits := uint64(cmd&0x7F) | (uint64(addr)&amask)<<7
		var one builder
		one.add(true, 2400)
		for i := 0; i < n; i++ {
			one.add(false, 600)
			if bits>>uint(i)&1 == 1 {
				one.add(true, 1200)
			} else {
				one.add(true, 600)
			}
		}
		frame := one.d
		for r := 0; r < 3; r++ { // sent three times, 45 ms apart
			if r > 0 {
				b.add(false, 45000-total(frame))
			}
			for i, v := range frame {
				b.add(i%2 == 0, v)
			}
		}
		freq = 40000
	case "RC5", "RC5X":
		// 14 bits, MSB first: S1=1, S2 (1 = RC5, 0 = RC5X, i.e. command + 64), toggle, 5 address, 6 command.
		// Manchester, 889 us per half bit: 1 = space then mark, 0 = mark then space.
		t := toggle[protocol]
		toggle[protocol] = !t
		s2 := uint32(1)
		if protocol == "RC5X" {
			s2 = 0
		}
		word := uint32(1)<<13 | s2<<12 | b2u(t)<<11 | (addr&0x1F)<<6 | cmd&0x3F
		b.add(false, 0)
		for i := 13; i >= 0; i-- {
			if word>>uint(i)&1 == 1 {
				b.add(false, 889)
				b.add(true, 889)
			} else {
				b.add(true, 889)
				b.add(false, 889)
			}
		}
		freq = 36000
	case "RC6":
		// Mode 0: leader 2666/889, start bit 1, mode 000, toggle (double length), 8 address, 8 command, MSB first.
		// Manchester, 444 us per half bit: 1 = mark then space, 0 = space then mark.
		t := toggle[protocol]
		toggle[protocol] = !t
		b.add(true, 2666)
		b.add(false, 889)
		bit := func(v bool, half uint32) {
			if v {
				b.add(true, half)
				b.add(false, half)
			} else {
				b.add(false, half)
				b.add(true, half)
			}
		}
		bit(true, 444)
		for i := 0; i < 3; i++ {
			bit(false, 444)
		}
		bit(t, 889)
		for i := 7; i >= 0; i-- {
			bit(addr>>uint(i)&1 == 1, 444)
		}
		for i := 7; i >= 0; i-- {
			bit(cmd>>uint(i)&1 == 1, 444)
		}
		freq = 36000
	default:
		return Signal{}, fmt.Errorf("unknown protocol %q", protocol)
	}
	d := b.d
	if len(d)%2 == 0 { // ends with a space: drop it
		d = d[:len(d)-1]
	}
	return Signal{Freq: freq, Durations: d}, nil
}

func b2u(v bool) uint32 {
	if v {
		return 1
	}
	return 0
}

// ---------------- decoding (for learned signals) ----------------

// Code is a decoded signal.
type Code struct {
	Protocol      string
	Address, Cmd  uint32
}

func (c Code) String() string { return fmt.Sprintf("%s %02X/%02X", c.Protocol, c.Address, c.Cmd) }

func near(v, want uint32) bool { return v*10 > want*7 && v*10 < want*13 }

// Decode tries the common protocols. ok is false for anything else (send it raw).
func Decode(d []uint32) (Code, bool) {
	if c, ok := decodePD(d); ok {
		return c, true
	}
	if c, ok := decodeSIRC(d); ok {
		return c, true
	}
	if c, ok := decodeRC6(d); ok {
		return c, true
	}
	if c, ok := decodeRC5(d); ok {
		return c, true
	}
	return Code{}, false
}

// NEC family and Samsung32 (32 bits after a header)
func decodePD(d []uint32) (Code, bool) {
	if len(d) < 67 {
		return Code{}, false
	}
	var zero, one uint32
	var samsung bool
	switch {
	case near(d[0], 9000) && near(d[1], 4500):
		zero, one = 560, 1690
	case near(d[0], 4500) && near(d[1], 4500):
		zero, one, samsung = 550, 1650, true
	default:
		return Code{}, false
	}
	var bits uint32
	for i := 0; i < 32; i++ {
		m, s := d[2+2*i], d[3+2*i]
		if !near(m, zero) {
			return Code{}, false
		}
		switch {
		case near(s, one):
			bits |= 1 << uint(i)
		case near(s, zero):
		default:
			return Code{}, false
		}
	}
	a, na, c, nc := bits&0xFF, bits>>8&0xFF, bits>>16&0xFF, bits>>24&0xFF
	if samsung {
		if a == na && c^nc == 0xFF {
			return Code{"Samsung32", a, c}, true
		}
		return Code{}, false
	}
	if c^nc != 0xFF {
		return Code{"NECext", bits & 0xFFFF, bits >> 16}, true
	}
	if a^na == 0xFF {
		return Code{"NEC", a, c}, true
	}
	return Code{"NECext", bits & 0xFFFF, c | nc<<8}, true
}

func decodeSIRC(d []uint32) (Code, bool) {
	if len(d) < 25 || !near(d[0], 2400) {
		return Code{}, false
	}
	var bits uint32
	n := 0
	for i := 1; i+1 < len(d) && n < 20; i += 2 {
		if !near(d[i], 600) && n > 0 {
			break // gap to the next repetition
		}
		m := d[i+1]
		switch {
		case near(m, 1200):
			bits |= 1 << uint(n)
		case near(m, 600):
		default:
			return Code{}, false
		}
		n++
	}
	switch n {
	case 12:
		return Code{"SIRC", bits >> 7 & 0x1F, bits & 0x7F}, true
	case 15:
		return Code{"SIRC15", bits >> 7 & 0xFF, bits & 0x7F}, true
	case 20:
		return Code{"SIRC20", bits >> 7 & 0x1FFF, bits & 0x7F}, true
	}
	return Code{}, false
}

// halves splits durations into half-bit units of size u (1 or 2 units per duration), as levels (true = mark).
func halves(d []uint32, u uint32, startMark bool) ([]bool, bool) {
	var h []bool
	mark := startMark
	for _, v := range d {
		switch {
		case near(v, u):
			h = append(h, mark)
		case near(v, 2*u):
			h = append(h, mark, mark)
		case near(v, 3*u): // RC6: a normal half next to a toggle half
			h = append(h, mark, mark, mark)
		default:
			return h, false
		}
		mark = !mark
	}
	return h, true
}

func decodeRC5(d []uint32) (Code, bool) {
	if len(d) < 11 {
		return Code{}, false
	}
	// cut at the first long gap (repetitions)
	for i, v := range d {
		if v > 4000 {
			d = d[:i]
			break
		}
	}
	h, ok := halves(d, 889, true)
	if !ok {
		return Code{}, false
	}
	h = append([]bool{false}, h...) // the first bit (1) starts with a space that isn't visible
	if len(h)%2 == 1 {
		h = append(h, false) // last half (space) isn't visible either
	}
	if len(h) != 28 {
		return Code{}, false
	}
	var w uint32
	for i := 0; i < 14; i++ {
		a, b := h[2*i], h[2*i+1]
		if a == b {
			return Code{}, false
		}
		w <<= 1
		if !a && b {
			w |= 1
		}
	}
	if w>>13 != 1 {
		return Code{}, false
	}
	p := "RC5"
	if w>>12&1 == 0 {
		p = "RC5X"
	}
	return Code{p, w >> 6 & 0x1F, w & 0x3F}, true
}

func decodeRC6(d []uint32) (Code, bool) {
	if len(d) < 15 || !near(d[0], 2666) || !near(d[1], 889) {
		return Code{}, false
	}
	rest := d[2:]
	for i, v := range rest {
		if v > 4000 {
			rest = rest[:i]
			break
		}
	}
	h, ok := halves(rest, 444, true)
	if !ok {
		return Code{}, false
	}
	// layout in half bits: start(2) mode(6) toggle(4) address(16) command(16) = 44
	if len(h)%2 == 1 || len(h) < 43 {
		h = append(h, false)
	}
	if len(h) < 44 {
		return Code{}, false
	}
	get := func(pos int) (bool, bool) { // RC6: 1 = mark then space
		a, b := h[pos], h[pos+1]
		return a && !b, a != b
	}
	if v, ok := get(0); !ok || !v {
		return Code{}, false
	}
	for i := 0; i < 3; i++ {
		if v, ok := get(2 + 2*i); !ok || v {
			return Code{}, false // only mode 0
		}
	}
	if h[8] != h[9] || h[10] != h[11] || h[8] == h[10] {
		return Code{}, false // toggle bit is two half bits long each
	}
	var a, c uint32
	for i := 0; i < 8; i++ {
		v, ok := get(12 + 2*i)
		if !ok {
			return Code{}, false
		}
		a = a<<1 | b2u(v)
	}
	for i := 0; i < 8; i++ {
		v, ok := get(28 + 2*i)
		if !ok {
			return Code{}, false
		}
		c = c<<1 | b2u(v)
	}
	return Code{"RC6", a, c}, true
}

// Known reports whether Encode supports the protocol.
func Known(p string) bool {
	for _, k := range Protocols {
		if strings.EqualFold(k, p) {
			return true
		}
	}
	return false
}
