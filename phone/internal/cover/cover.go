// Package cover talks to the IR firmware on the cover's RP2040 (I2C 0x42, register map in
// firmware/ir-fw/ir-fw.ino). Every read is validated (magic bytes or checksums) and retried, because the
// first transfers after the RP2040 restarts can come back garbled.
package cover

import (
	"errors"
	"fmt"
	"time"

	"github.com/godbus/dbus/v5"

	"toh-ir/internal/tohbus"
)

const (
	Addr        = 0x42
	regIR       = 0x20
	regCmd      = 0x24
	regCarrier  = 0x25
	regRxCount  = 0x28
	regTxSum    = 0x2C
	regTxPort   = 0x30
	regRxPort   = 0x31
	cmdSend     = 1
	cmdLearn    = 2
	cmdCancel   = 3
	cmdTxReset  = 4
	cmdRxRewind = 5
	chunk       = 32 // bytes per transfer on the data ports
	MaxDur      = 2048
	stReady     = 1
	stLearning  = 2
	stSending   = 4
	stTrunc     = 8
)

var ErrCancelled = errors.New("cancelled")

// Bus is a borrowed TOH bus. Close returns it.
type Bus struct {
	fd      int
	release func()
}

// Open borrows the bus, retrying for a moment if another program has it.
func Open(sys *dbus.Conn) (*Bus, error) {
	var err error
	for i := 0; i < 10; i++ {
		var fd int
		var rel func()
		fd, rel, err = tohbus.Borrow(sys)
		if err == nil {
			return &Bus{fd: fd, release: rel}, nil
		}
		time.Sleep(200 * time.Millisecond)
	}
	return nil, err
}

func (b *Bus) Close() { b.release() }

func (b *Bus) xfer(wr []byte, rd int) ([]byte, error) {
	var err error
	for i := 0; i < 5; i++ {
		var r []byte
		r, err = tohbus.Xfer(b.fd, Addr, wr, rd)
		if err == nil {
			return r, nil
		}
		time.Sleep(time.Duration(5*(i+1)) * time.Millisecond)
	}
	return nil, err
}

// status reads 0x20-0x23 and checks the "IR" magic, retrying on garbled answers.
func (b *Bus) status() (byte, error) {
	var last []byte
	for i := 0; i < 8; i++ {
		r, err := b.xfer([]byte{regIR}, 4)
		if err != nil {
			return 0, fmt.Errorf("no answer from the cover's board: %v", err)
		}
		if r[0] == 'I' && r[1] == 'R' && r[2] == 1 && r[3]&0xF0 == 0 {
			return r[3], nil
		}
		last = r
	}
	return 0, fmt.Errorf("the cover's board doesn't run the IR firmware (answer % x)", last)
}

// Probe checks that the IR firmware answers.
func (b *Bus) Probe() error {
	_, err := b.status()
	return err
}

func le16(v int) (byte, byte) { return byte(v), byte(v >> 8) }

// Send transmits one signal and waits until the board has sent it.
func (b *Bus) Send(freq int, durations []uint32) error {
	// durations longer than 16 bits: split spaces with zero-length marks
	var d []uint32
	for _, v := range durations {
		for v > 65535 {
			d = append(d, 65535, 0) // same level 65535 us, then 0 us of the other level
			v -= 65535
		}
		d = append(d, v)
	}
	if len(d) > MaxDur {
		return fmt.Errorf("signal too long (%d timings, max %d)", len(d), MaxDur)
	}
	data := make([]byte, 0, 2*len(d))
	var sum uint16
	var us uint32
	for _, v := range d {
		data = append(data, byte(v), byte(v>>8))
		sum += uint16(byte(v)) + uint16(byte(v>>8))
		us += v
	}
	ok := false
	for try := 0; try < 4 && !ok; try++ {
		if _, err := b.xfer([]byte{regCmd, cmdTxReset}, 0); err != nil {
			return err
		}
		for off := 0; off < len(data); off += chunk {
			end := off + chunk
			if end > len(data) {
				end = len(data)
			}
			if _, err := b.xfer(append([]byte{regTxPort}, data[off:end]...), 0); err != nil {
				return err
			}
		}
		r, err := b.xfer([]byte{regTxSum}, 4)
		if err != nil {
			return err
		}
		ok = uint16(r[0])|uint16(r[1])<<8 == sum && int(r[2])|int(r[3])<<8 == len(data)
	}
	if !ok {
		return fmt.Errorf("upload to the cover failed (checksum)")
	}
	khz := (freq + 500) / 1000
	if khz < 20 || khz > 60 {
		khz = 38
	}
	lo, hi := le16(len(d))
	if _, err := b.xfer([]byte{regCarrier, byte(khz), lo, hi}, 0); err != nil {
		return err
	}
	if _, err := b.xfer([]byte{regCmd, cmdSend}, 0); err != nil {
		return err
	}
	deadline := time.Now().Add(time.Duration(us)*time.Microsecond + 500*time.Millisecond)
	time.Sleep(time.Duration(us) * time.Microsecond)
	for time.Now().Before(deadline) {
		st, err := b.status()
		if err != nil {
			return err
		}
		if st&stSending == 0 {
			return nil
		}
		time.Sleep(5 * time.Millisecond)
	}
	return fmt.Errorf("the cover didn't finish sending")
}

// Learn waits for one IR frame from a remote (until timeout or cancel is closed) and returns its durations.
func (b *Bus) Learn(timeout time.Duration, cancel <-chan struct{}) ([]uint32, error) {
	if _, err := b.xfer([]byte{regCmd, cmdLearn}, 0); err != nil {
		return nil, err
	}
	stop := func() { b.xfer([]byte{regCmd, cmdCancel}, 0) }
	end := time.Now().Add(timeout)
	for {
		select {
		case <-cancel:
			stop()
			return nil, ErrCancelled
		case <-time.After(100 * time.Millisecond):
		}
		st, err := b.status()
		if err != nil {
			stop()
			return nil, err
		}
		if st&stReady != 0 {
			break
		}
		if st&stLearning == 0 { // board restarted or someone cancelled: arm again
			b.xfer([]byte{regCmd, cmdLearn}, 0)
		}
		if time.Now().After(end) {
			stop()
			return nil, fmt.Errorf("no signal received")
		}
	}
	for try := 0; try < 5; try++ {
		r, err := b.xfer([]byte{regRxCount}, 4)
		if err != nil {
			return nil, err
		}
		n := int(r[0]) | int(r[1])<<8
		want := uint16(r[2]) | uint16(r[3])<<8
		if n <= 0 || n > MaxDur {
			continue
		}
		if _, err := b.xfer([]byte{regCmd, cmdRxRewind}, 0); err != nil {
			return nil, err
		}
		data := make([]byte, 0, 2*n)
		for len(data) < 2*n {
			k := 2*n - len(data)
			if k > chunk {
				k = chunk
			}
			p, err := b.xfer([]byte{regRxPort}, k)
			if err != nil {
				return nil, err
			}
			data = append(data, p...)
		}
		var sum uint16
		for _, v := range data {
			sum += uint16(v)
		}
		if sum != want {
			continue
		}
		d := make([]uint32, n)
		for i := range d {
			d[i] = uint32(data[2*i]) | uint32(data[2*i+1])<<8
		}
		return d, nil
	}
	return nil, fmt.Errorf("reading the signal from the cover failed (checksum)")
}
