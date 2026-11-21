package irproto

import (
	"os"
	"path/filepath"
	"testing"

	"toh-ir/internal/irfile"
)

// IRDB_DIR=<dir with .ir files> go test -run Real -v ./internal/irproto
func TestRealFiles(t *testing.T) {
	dir := os.Getenv("IRDB_DIR")
	if dir == "" {
		t.Skip("IRDB_DIR not set")
	}
	files, _ := filepath.Glob(filepath.Join(dir, "*.ir"))
	for _, fn := range files {
		fh, _ := os.Open(fn)
		f, err := irfile.Parse(fh)
		fh.Close()
		if err != nil {
			t.Fatal(err)
		}
		dec, raw := map[string]int{}, 0
		for _, s := range f.Signals {
			if s.Type != "raw" {
				continue
			}
			raw++
			if c, ok := Decode(s.Data); ok {
				dec[c.Protocol]++
			}
		}
		t.Logf("%-40s raw=%d decoded=%v", filepath.Base(fn), raw, dec)
	}
}
