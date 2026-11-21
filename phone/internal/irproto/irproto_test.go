package irproto

import "testing"

func TestRoundTrip(t *testing.T) {
	cases := []Code{
		{"NEC", 0x03, 0x1D}, {"NEC", 0x7F, 0x12}, {"NECext", 0x1234, 0xF708}, {"Samsung32", 0x07, 0x02},
		{"RC5", 0x00, 0x0C}, {"RC5", 0x10, 0x35}, {"RC5X", 0x05, 0x01}, {"RC6", 0x10, 0x0C}, {"RC6", 0x00, 0xFF},
		{"SIRC", 0x01, 0x15}, {"SIRC15", 0x97, 0x2A}, {"SIRC20", 0x10FA, 0x3C},
	}
	for _, c := range cases {
		for press := 0; press < 2; press++ { // both toggle states
			s, err := Encode(c.Protocol, c.Address, c.Cmd)
			if err != nil {
				t.Fatal(err)
			}
			if len(s.Durations)%2 != 1 {
				t.Errorf("%v: even number of durations", c)
			}
			got, ok := Decode(s.Durations)
			if !ok || got != c {
				t.Errorf("%v press %d: decoded %v ok=%v (%v)", c, press, got, ok, s.Durations)
			}
		}
	}
	for _, p := range []string{"NEC42", "NEC42ext", "Kaseikyo", "RCA", "Pioneer"} {
		s, err := Encode(p, 0x12, 0x34)
		if err != nil || len(s.Durations) < 20 || len(s.Durations)%2 != 1 {
			t.Errorf("%s: %v %d", p, err, len(s.Durations))
		}
	}
}

func TestRC5Shape(t *testing.T) {
	toggle["RC5"] = false
	s, _ := Encode("RC5", 0, 0x0C) // 11 T 00000 001100
	if s.Durations[0] != 889 || s.Freq != 36000 {
		t.Errorf("RC5 starts %v", s.Durations[:4])
	}
}
