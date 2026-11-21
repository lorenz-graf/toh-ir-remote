// Package irfile reads and writes Flipper Zero .ir files ("IR signals file" and "IR library file").
// A remote is one file; several signals may share a button name (universal remotes: all are sent in turn).
// TOH IR keeps its own metadata in comment lines starting with "#@", which Flipper ignores.
package irfile

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// Signal is one entry of a .ir file.
type Signal struct {
	Name     string
	Type     string // "parsed" or "raw"
	Protocol string // parsed
	Address  uint32
	Command  uint32
	Freq     int // raw
	Duty     string
	Data     []uint32 // raw
}

// File is a whole .ir file.
type File struct {
	Filetype string
	Meta     map[string]string // "#@key: value" lines
	Signals  []Signal
}

func parseHex4(s string) (uint32, error) {
	f := strings.Fields(s)
	var v uint32
	for i, b := range f {
		if i >= 4 {
			break
		}
		x, err := strconv.ParseUint(b, 16, 8)
		if err != nil {
			return 0, err
		}
		v |= uint32(x) << (8 * uint(i))
	}
	return v, nil
}

// Parse reads a .ir file. Broken entries are skipped.
func Parse(r io.Reader) (*File, error) {
	f := &File{Meta: map[string]string{}}
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 1<<20), 1<<24)
	var cur *Signal
	flush := func() {
		if cur != nil && cur.Name != "" && (cur.Type == "parsed" && cur.Protocol != "" || cur.Type == "raw" && len(cur.Data) > 0) {
			f.Signals = append(f.Signals, *cur)
		}
		cur = nil
	}
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if strings.HasPrefix(line, "#@") {
			if k, v, ok := strings.Cut(line[2:], ":"); ok {
				f.Meta[strings.TrimSpace(k)] = strings.TrimSpace(v)
			}
			continue
		}
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		k, v, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		k, v = strings.TrimSpace(k), strings.TrimSpace(v)
		switch k {
		case "Filetype":
			f.Filetype = v
		case "name":
			flush()
			cur = &Signal{Name: v}
		default:
			if cur == nil {
				continue
			}
			switch k {
			case "type":
				cur.Type = v
			case "protocol":
				cur.Protocol = v
			case "address":
				cur.Address, _ = parseHex4(v)
			case "command":
				cur.Command, _ = parseHex4(v)
			case "frequency":
				cur.Freq, _ = strconv.Atoi(v)
			case "duty_cycle":
				cur.Duty = v
			case "data":
				for _, x := range strings.Fields(v) {
					n, err := strconv.ParseUint(x, 10, 32)
					if err == nil {
						cur.Data = append(cur.Data, uint32(n))
					}
				}
			}
		}
	}
	flush()
	if err := sc.Err(); err != nil {
		return nil, err
	}
	return f, nil
}

func hex4(v uint32) string {
	return fmt.Sprintf("%02X %02X %02X %02X", byte(v), byte(v>>8), byte(v>>16), byte(v>>24))
}

// Write writes the file in Flipper's format.
func (f *File) Write(w io.Writer) error {
	bw := bufio.NewWriter(w)
	ft := f.Filetype
	if ft == "" {
		ft = "IR signals file"
	}
	fmt.Fprintf(bw, "Filetype: %s\nVersion: 1\n", ft)
	for _, k := range []string{"title", "kind", "source", "layout"} {
		if v := f.Meta[k]; v != "" {
			fmt.Fprintf(bw, "#@%s: %s\n", k, v)
		}
	}
	for _, s := range f.Signals {
		fmt.Fprintf(bw, "# \nname: %s\ntype: %s\n", s.Name, s.Type)
		if s.Type == "parsed" {
			fmt.Fprintf(bw, "protocol: %s\naddress: %s\ncommand: %s\n", s.Protocol, hex4(s.Address), hex4(s.Command))
		} else {
			duty := s.Duty
			if duty == "" {
				duty = "0.330000"
			}
			parts := make([]string, len(s.Data))
			for i, v := range s.Data {
				parts[i] = strconv.FormatUint(uint64(v), 10)
			}
			fmt.Fprintf(bw, "frequency: %d\nduty_cycle: %s\ndata: %s\n", s.Freq, duty, strings.Join(parts, " "))
		}
	}
	return bw.Flush()
}
