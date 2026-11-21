// toh-ird: session D-Bus service between the TOH IR app and the IR firmware on the cover (I2C 0x42).
// Started on demand by D-Bus activation and exits after a few idle minutes. It keeps the user's remotes as
// Flipper .ir files in ~/.local/share/toh-ir/remotes/, encodes protocols, sends and learns.
//
// Interface io.github.lorenz_graf.TohIr at /io/github/lorenz_graf/TohIr (results are JSON strings):
//
//	Probe() -> s                          "" if the cover answers, else the error
//	Remotes() -> s                        [{id,title,kind,count}]
//	Remote(s id) -> s                     {id,title,kind,source,buttons:[{name,key,count,type,info}]}
//	DbCategories() -> s                   [{name,label,count}]           (bundled Flipper-IRDB index)
//	DbBrands(s cat) -> s                  [{name,label,count}]
//	DbModels(s cat, s brand) -> s         [{path,label}]
//	DbSearch(s query) -> s                [{path,label}] (max 200)
//	AddFromDb(s path) -> s id             downloads the file and adds it as a remote
//	AddUniversal(s kind) -> s id          Flipper's universal remote (tv, audio, projector, ac), downloaded
//	AddBlank(s title, s kind) -> s id     empty remote, every button learned by hand
//	Rename(s id, s title), Delete(s id), DeleteButton(s id, s name)
//	Press(s id, s name)                   sends all signals of that button (async, see State)
//	PressIndex(s id, s name, i n)         sends only the n-th signal of that button (universal: find the code)
//	Keep(s id, s name, i n)               keeps only the n-th signal of that button ("it worked")
//	Learn(s id, s name)                   waits for a remote and stores the signal under that name (async)
//	Cancel()                              stops sending / learning
//	State() -> s                          {busy,progress,result,error,seq}; seq counts finished operations
//
// CLI for testing (same binary, so it may borrow the bus too):
//
//	toh-ird probe | toh-ird code <protocol> <address> <command> | toh-ird send <file.ir> <name> | toh-ird learn
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/godbus/dbus/v5"
	"github.com/godbus/dbus/v5/introspect"

	"toh-ir/internal/cover"
	"toh-ir/internal/irfile"
	"toh-ir/internal/irproto"
)

const (
	busName   = "io.github.lorenz_graf.TohIr"
	objPath   = "/io/github/lorenz_graf/TohIr"
	idleFor   = 5 * time.Minute
	indexPath = "/usr/share/toh-ir/irdb-index.json"
	// Flipper's universal remotes are GPL-3.0: downloaded by the user's request, not shipped.
	universalURL = "https://raw.githubusercontent.com/flipperdevices/flipperzero-firmware/0c87c9e2df068848dff786f2851dd360e24a75f9/applications/main/infrared/resources/infrared/assets/"
)

func dataDir() string {
	base := os.Getenv("XDG_DATA_HOME")
	if base == "" {
		base = filepath.Join(os.Getenv("HOME"), ".local", "share")
	}
	return filepath.Join(base, "toh-ir", "remotes")
}

// ---------------- remotes on disk ----------------

var idRe = regexp.MustCompile(`^[a-z0-9-]+$`)

func remotePath(id string) (string, error) {
	if !idRe.MatchString(id) {
		return "", fmt.Errorf("bad id")
	}
	return filepath.Join(dataDir(), id+".ir"), nil
}

func loadRemote(id string) (*irfile.File, error) {
	p, err := remotePath(id)
	if err != nil {
		return nil, err
	}
	fh, err := os.Open(p)
	if err != nil {
		return nil, err
	}
	defer fh.Close()
	return irfile.Parse(fh)
}

func saveRemote(id string, f *irfile.File) error {
	p, err := remotePath(id)
	if err != nil {
		return err
	}
	os.MkdirAll(filepath.Dir(p), 0o755)
	tmp := p + ".tmp"
	fh, err := os.Create(tmp)
	if err != nil {
		return err
	}
	if err := f.Write(fh); err != nil {
		fh.Close()
		return err
	}
	fh.Close()
	return os.Rename(tmp, p)
}

func newID(title string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(title) {
		switch {
		case r >= 'a' && r <= 'z' || r >= '0' && r <= '9':
			b.WriteRune(r)
		case b.Len() > 0 && !strings.HasSuffix(b.String(), "-"):
			b.WriteByte('-')
		}
	}
	s := b.String()
	if len(s) > 30 {
		s = s[:30]
	}
	s = strings.Trim(s, "-")
	if s == "" {
		s = "remote"
	}
	return fmt.Sprintf("%s-%04x", s, rand.Intn(0x10000))
}

func addRemote(title, kind, source string, f *irfile.File) (string, error) {
	if f == nil {
		f = &irfile.File{}
	}
	f.Filetype = "IR signals file"
	if f.Meta == nil {
		f.Meta = map[string]string{}
	}
	f.Meta["title"], f.Meta["kind"], f.Meta["source"] = title, kind, source
	id := newID(title)
	return id, saveRemote(id, f)
}

// ---------------- database index ----------------

type dbIndex struct {
	Repo   string   `json:"repo"`
	Commit string   `json:"commit"`
	Files  []string `json:"files"`
}

var (
	dbOnce sync.Once
	db     dbIndex
)

func index() *dbIndex {
	dbOnce.Do(func() {
		b, err := os.ReadFile(indexPath)
		if err == nil {
			json.Unmarshal(b, &db)
		}
	})
	return &db
}

func label(s string) string {
	s = strings.TrimSuffix(s, ".ir")
	return strings.TrimSpace(strings.ReplaceAll(s, "_", " "))
}

func kindFor(category string) string {
	switch category {
	case "TVs", "Universal_TV_Remotes", "Monitors", "Touchscreen_Displays", "Digital_Signs", "TV_Tuner", "DVB-T":
		return "tv"
	case "Projectors":
		return "projector"
	case "SoundBars", "Audio_and_Video_Receivers", "Speakers", "CD_Players", "Car_Multimedia", "Head_Units":
		return "audio"
	case "Blu-Ray", "DVD_Players", "Streaming_Devices", "Cable_Boxes", "VCR", "Laserdisc", "Multimedia", "Consoles":
		return "player"
	case "ACs", "Heaters", "Fans", "Air_Purifiers", "Humidifiers", "Fireplaces":
		return "climate"
	case "LED_Lighting":
		return "light"
	}
	return "other"
}

type entry struct {
	Name  string `json:"name,omitempty"`
	Path  string `json:"path,omitempty"`
	Label string `json:"label"`
	Count int    `json:"count,omitempty"`
}

func groupBy(files []string, level int, prefix string) []entry {
	counts := map[string]int{}
	for _, f := range files {
		if !strings.HasPrefix(f, prefix) {
			continue
		}
		parts := strings.Split(f[len(prefix):], "/")
		if len(parts) > 1 {
			counts[parts[0]]++
		}
	}
	var out []entry
	for k, n := range counts {
		out = append(out, entry{Name: k, Label: label(k), Count: n})
	}
	sort.Slice(out, func(i, j int) bool { return strings.ToLower(out[i].Label) < strings.ToLower(out[j].Label) })
	return out
}

func download(u string) ([]byte, error) {
	c := http.Client{Timeout: 20 * time.Second}
	resp, err := c.Get(u)
	if err != nil {
		return nil, fmt.Errorf("download failed (internet?): %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("download failed: %s", resp.Status)
	}
	return io.ReadAll(io.LimitReader(resp.Body, 8<<20))
}

func escapePath(p string) string {
	parts := strings.Split(p, "/")
	for i := range parts {
		parts[i] = url.PathEscape(parts[i])
	}
	return strings.Join(parts, "/")
}

// ---------------- sending ----------------

func signalFor(s irfile.Signal) (irproto.Signal, error) {
	if s.Type == "raw" {
		f := s.Freq
		if f == 0 {
			f = 38000
		}
		d := s.Data
		if len(d)%2 == 0 && len(d) > 0 {
			d = d[:len(d)-1]
		}
		return irproto.Signal{Freq: f, Durations: d}, nil
	}
	return irproto.Encode(s.Protocol, s.Address, s.Command)
}

// ---------------- D-Bus service ----------------

type state struct {
	Busy     string `json:"busy"`
	Progress string `json:"progress"`
	Result   string `json:"result"`
	Error    string `json:"error"`
	Seq      int    `json:"seq"`
}

type service struct {
	sys    *dbus.Conn
	mu     sync.Mutex
	last   time.Time
	st     state
	cancel chan struct{}
}

func fail(err error) *dbus.Error {
	return dbus.NewError(busName+".Error.Failed", []any{err.Error()})
}

func jsonStr(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}

func (s *service) touch() {
	s.mu.Lock()
	s.last = time.Now()
	s.mu.Unlock()
}

func (s *service) Probe() (string, *dbus.Error) {
	s.touch()
	s.mu.Lock()
	busy := s.st.Busy != ""
	s.mu.Unlock()
	if busy {
		return "", nil
	}
	b, err := cover.Open(s.sys)
	if err != nil {
		return "Cover not attached (or no TOH IR config)", nil
	}
	defer b.Close()
	if err := b.Probe(); err != nil {
		return err.Error(), nil
	}
	return "", nil
}

type remoteInfo struct {
	ID      string       `json:"id"`
	Title   string       `json:"title"`
	Kind    string       `json:"kind"`
	Source  string       `json:"source,omitempty"`
	Count   int          `json:"count"`
	Buttons []buttonInfo `json:"buttons,omitempty"`
}

type buttonInfo struct {
	Name  string `json:"name"`
	Key   string `json:"key"`
	Count int    `json:"count"`
	Type  string `json:"type"`
	Info  string `json:"info"`
}

func describe(id string, f *irfile.File, withButtons bool) remoteInfo {
	r := remoteInfo{ID: id, Title: f.Meta["title"], Kind: f.Meta["kind"], Source: f.Meta["source"]}
	if r.Title == "" {
		r.Title = id
	}
	idx := map[string]int{}
	used := map[string]bool{}
	for _, sg := range f.Signals {
		if i, ok := idx[sg.Name]; ok {
			r.Buttons[i].Count++
			continue
		}
		k := keyFor(sg.Name)
		if used[k] {
			k = "" // a second name for the same slot goes to "more buttons"
		}
		if k != "" {
			used[k] = true
		}
		info := "raw"
		if sg.Type == "parsed" {
			info = fmt.Sprintf("%s %X/%X", sg.Protocol, sg.Address, sg.Command)
		} else if c, ok := irproto.Decode(sg.Data); ok {
			info = c.String() + " (raw)"
		}
		idx[sg.Name] = len(r.Buttons)
		r.Buttons = append(r.Buttons, buttonInfo{Name: sg.Name, Key: k, Count: 1, Type: sg.Type, Info: info})
	}
	r.Count = len(r.Buttons)
	if !withButtons {
		r.Buttons = nil
	}
	return r
}

func (s *service) Remotes() (string, *dbus.Error) {
	s.touch()
	files, _ := filepath.Glob(filepath.Join(dataDir(), "*.ir"))
	out := []remoteInfo{}
	for _, p := range files {
		id := strings.TrimSuffix(filepath.Base(p), ".ir")
		f, err := loadRemote(id)
		if err != nil {
			continue
		}
		out = append(out, describe(id, f, false))
	}
	sort.Slice(out, func(i, j int) bool { return strings.ToLower(out[i].Title) < strings.ToLower(out[j].Title) })
	return jsonStr(out), nil
}

func (s *service) Remote(id string) (string, *dbus.Error) {
	s.touch()
	f, err := loadRemote(id)
	if err != nil {
		return "", fail(err)
	}
	return jsonStr(describe(id, f, true)), nil
}

func (s *service) DbCategories() (string, *dbus.Error) {
	s.touch()
	return jsonStr(groupBy(index().Files, 0, "")), nil
}

func (s *service) DbBrands(cat string) (string, *dbus.Error) {
	s.touch()
	return jsonStr(groupBy(index().Files, 1, cat+"/")), nil
}

func (s *service) DbModels(cat, brand string) (string, *dbus.Error) {
	s.touch()
	prefix := cat + "/" + brand + "/"
	out := []entry{}
	for _, f := range index().Files {
		if strings.HasPrefix(f, prefix) {
			out = append(out, entry{Path: f, Label: label(f[len(prefix):])})
		}
	}
	return jsonStr(out), nil
}

func (s *service) DbSearch(q string) (string, *dbus.Error) {
	s.touch()
	words := strings.Fields(strings.ToLower(strings.ReplaceAll(q, "_", " ")))
	out := []entry{}
	if len(words) == 0 {
		return jsonStr(out), nil
	}
	for _, f := range index().Files {
		l := strings.ToLower(strings.ReplaceAll(f, "_", " "))
		all := true
		for _, w := range words {
			if !strings.Contains(l, w) {
				all = false
				break
			}
		}
		if all {
			out = append(out, entry{Path: f, Label: label(strings.ReplaceAll(f, "/", " · "))})
			if len(out) >= 200 {
				break
			}
		}
	}
	return jsonStr(out), nil
}

func (s *service) AddFromDb(path string) (string, *dbus.Error) {
	s.touch()
	ix := index()
	found := false
	for _, f := range ix.Files {
		if f == path {
			found = true
			break
		}
	}
	if !found {
		return "", fail(fmt.Errorf("not in the database: %s", path))
	}
	b, err := download("https://raw.githubusercontent.com/" + ix.Repo + "/" + ix.Commit + "/" + escapePath(path))
	if err != nil {
		return "", fail(err)
	}
	f, err := irfile.Parse(strings.NewReader(string(b)))
	if err != nil || len(f.Signals) == 0 {
		return "", fail(fmt.Errorf("the file has no usable buttons"))
	}
	parts := strings.Split(path, "/")
	title := label(parts[len(parts)-1])
	if len(parts) >= 3 && !strings.Contains(strings.ToLower(title), strings.ToLower(label(parts[1]))) {
		title = label(parts[1]) + " " + title
	}
	id, err := addRemote(title, kindFor(parts[0]), "Flipper-IRDB: "+path, f)
	if err != nil {
		return "", fail(err)
	}
	return id, nil
}

func (s *service) AddUniversal(kind string) (string, *dbus.Error) {
	s.touch()
	titles := map[string]string{"tv": "Universal TV", "audio": "Universal audio", "projector": "Universal projector", "ac": "Universal air conditioner"}
	kinds := map[string]string{"tv": "tv", "audio": "audio", "projector": "projector", "ac": "climate"}
	t, ok := titles[kind]
	if !ok {
		return "", fail(fmt.Errorf("unknown kind %q", kind))
	}
	b, err := download(universalURL + kind + ".ir")
	if err != nil {
		return "", fail(err)
	}
	f, err := irfile.Parse(strings.NewReader(string(b)))
	if err != nil || len(f.Signals) == 0 {
		return "", fail(fmt.Errorf("the file has no usable buttons"))
	}
	id, err := addRemote(t, kinds[kind], "Flipper Zero universal remote ("+kind+".ir, GPL-3.0)", f)
	if err != nil {
		return "", fail(err)
	}
	return id, nil
}

func (s *service) AddBlank(title, kind string) (string, *dbus.Error) {
	s.touch()
	if strings.TrimSpace(title) == "" {
		title = "My remote"
	}
	id, err := addRemote(strings.TrimSpace(title), kind, "", nil)
	if err != nil {
		return "", fail(err)
	}
	return id, nil
}

func (s *service) Rename(id, title string) *dbus.Error {
	s.touch()
	f, err := loadRemote(id)
	if err != nil {
		return fail(err)
	}
	f.Meta["title"] = strings.TrimSpace(title)
	if err := saveRemote(id, f); err != nil {
		return fail(err)
	}
	return nil
}

func (s *service) Delete(id string) *dbus.Error {
	s.touch()
	p, err := remotePath(id)
	if err != nil {
		return fail(err)
	}
	if err := os.Remove(p); err != nil {
		return fail(err)
	}
	return nil
}

func (s *service) DeleteButton(id, name string) *dbus.Error {
	s.touch()
	f, err := loadRemote(id)
	if err != nil {
		return fail(err)
	}
	var keep []irfile.Signal
	for _, sg := range f.Signals {
		if sg.Name != name {
			keep = append(keep, sg)
		}
	}
	f.Signals = keep
	if err := saveRemote(id, f); err != nil {
		return fail(err)
	}
	return nil
}

func (s *service) State() (string, *dbus.Error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.last = time.Now()
	return jsonStr(s.st), nil
}

func (s *service) Cancel() *dbus.Error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cancel != nil {
		close(s.cancel)
		s.cancel = nil
	}
	return nil
}

// start begins an async operation; false if another one runs.
func (s *service) start(kind string) (chan struct{}, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.st.Busy != "" {
		return nil, false
	}
	s.st = state{Busy: kind, Seq: s.st.Seq}
	s.cancel = make(chan struct{})
	s.last = time.Now()
	return s.cancel, true
}

func (s *service) finish(result string, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.st.Busy, s.st.Progress, s.st.Result, s.st.Error = "", "", result, ""
	if err != nil {
		s.st.Result, s.st.Error = "", err.Error()
	}
	s.st.Seq++
	s.cancel = nil
	s.last = time.Now()
}

func (s *service) progress(p string) {
	s.mu.Lock()
	s.st.Progress = p
	s.last = time.Now()
	s.mu.Unlock()
}

func signalsOf(id, name string) ([]irfile.Signal, *irfile.File, error) {
	f, err := loadRemote(id)
	if err != nil {
		return nil, nil, err
	}
	var sigs []irfile.Signal
	for _, sg := range f.Signals {
		if sg.Name == name {
			sigs = append(sigs, sg)
		}
	}
	if len(sigs) == 0 {
		return nil, f, fmt.Errorf("no signal for %q yet: long press to record one", name)
	}
	return sigs, f, nil
}

func (s *service) Press(id, name string) *dbus.Error {
	sigs, _, err := signalsOf(id, name)
	if err != nil {
		return fail(err)
	}
	return s.pressSigs(name, sigs)
}

func (s *service) PressIndex(id, name string, n int32) *dbus.Error {
	sigs, _, err := signalsOf(id, name)
	if err != nil {
		return fail(err)
	}
	if n < 0 || int(n) >= len(sigs) {
		return fail(fmt.Errorf("no code %d", n+1))
	}
	return s.pressSigs(name, sigs[n:n+1])
}

func (s *service) Keep(id, name string, n int32) *dbus.Error {
	sigs, f, err := signalsOf(id, name)
	if err != nil {
		return fail(err)
	}
	if n < 0 || int(n) >= len(sigs) {
		return fail(fmt.Errorf("no code %d", n+1))
	}
	keep := sigs[n]
	var out []irfile.Signal
	placed := false
	for _, sg := range f.Signals {
		if sg.Name == name {
			if !placed {
				out = append(out, keep)
				placed = true
			}
			continue
		}
		out = append(out, sg)
	}
	f.Signals = out
	if err := saveRemote(id, f); err != nil {
		return fail(err)
	}
	return nil
}

func (s *service) pressSigs(name string, sigs []irfile.Signal) *dbus.Error {
	cancel, ok := s.start("send")
	if !ok {
		return fail(fmt.Errorf("busy"))
	}
	go func() {
		b, err := cover.Open(s.sys)
		if err != nil {
			s.finish("", fmt.Errorf("cover not attached"))
			return
		}
		defer b.Close()
		sent := 0
		var lastErr error
		for i, sg := range sigs {
			select {
			case <-cancel:
				s.finish(fmt.Sprintf("stopped after %d of %d", sent, len(sigs)), nil)
				return
			default:
			}
			if len(sigs) > 1 {
				s.progress(fmt.Sprintf("%d / %d", i+1, len(sigs)))
			}
			sig, err := signalFor(sg)
			if err != nil {
				lastErr = err
				continue
			}
			if err := b.Send(sig.Freq, sig.Durations); err != nil {
				s.finish("", err)
				return
			}
			sent++
			if len(sigs) > 1 {
				time.Sleep(60 * time.Millisecond)
			}
		}
		if sent == 0 && lastErr != nil {
			s.finish("", lastErr)
			return
		}
		s.finish(fmt.Sprintf("sent %s", name), nil)
	}()
	return nil
}

func (s *service) Learn(id, name string) *dbus.Error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fail(fmt.Errorf("empty button name"))
	}
	if _, err := loadRemote(id); err != nil {
		return fail(err)
	}
	cancel, ok := s.start("learn")
	if !ok {
		return fail(fmt.Errorf("busy"))
	}
	go func() {
		b, err := cover.Open(s.sys)
		if err != nil {
			s.finish("", fmt.Errorf("cover not attached"))
			return
		}
		d, err := b.Learn(30*time.Second, cancel)
		b.Close()
		if err == cover.ErrCancelled {
			s.finish("cancelled", nil)
			return
		}
		if err != nil {
			s.finish("", err)
			return
		}
		sg := irfile.Signal{Name: name, Type: "raw", Freq: 38000, Duty: "0.330000", Data: d}
		res := fmt.Sprintf("saved (raw, %d timings)", len(d))
		if c, ok := irproto.Decode(d); ok {
			sg = irfile.Signal{Name: name, Type: "parsed", Protocol: c.Protocol, Address: c.Address, Command: c.Cmd}
			res = "saved: " + c.String()
		}
		f, err := loadRemote(id) // reload: the file may have changed meanwhile
		if err != nil {
			s.finish("", err)
			return
		}
		var out []irfile.Signal
		placed := false
		for _, old := range f.Signals {
			if old.Name == name {
				if !placed {
					out = append(out, sg)
					placed = true
				}
				continue
			}
			out = append(out, old)
		}
		if !placed {
			out = append(out, sg)
		}
		f.Signals = out
		if err := saveRemote(id, f); err != nil {
			s.finish("", err)
			return
		}
		s.finish(res, nil)
	}()
	return nil
}

// ---------------- main ----------------

func cli(sys *dbus.Conn, args []string) error {
	b, err := cover.Open(sys)
	if err != nil {
		return err
	}
	defer b.Close()
	switch args[0] {
	case "probe":
		return b.Probe()
	case "code":
		if len(args) != 4 {
			return fmt.Errorf("usage: code <protocol> <address> <command> (hex)")
		}
		a, _ := strconv.ParseUint(args[2], 16, 32)
		c, _ := strconv.ParseUint(args[3], 16, 32)
		sig, err := irproto.Encode(args[1], uint32(a), uint32(c))
		if err != nil {
			return err
		}
		return b.Send(sig.Freq, sig.Durations)
	case "send":
		if len(args) != 3 {
			return fmt.Errorf("usage: send <file.ir> <name>")
		}
		fh, err := os.Open(args[1])
		if err != nil {
			return err
		}
		f, err := irfile.Parse(fh)
		fh.Close()
		if err != nil {
			return err
		}
		n := 0
		for _, sg := range f.Signals {
			if sg.Name == args[2] {
				sig, err := signalFor(sg)
				if err != nil {
					return err
				}
				if err := b.Send(sig.Freq, sig.Durations); err != nil {
					return err
				}
				n++
			}
		}
		fmt.Println("sent", n)
		return nil
	case "learn":
		fmt.Println("press a button on a remote (30 s)")
		d, err := b.Learn(30*time.Second, make(chan struct{}))
		if err != nil {
			return err
		}
		if c, ok := irproto.Decode(d); ok {
			fmt.Println(c)
		}
		fmt.Println(len(d), "timings:", d)
		return nil
	}
	return fmt.Errorf("commands: probe, code, send, learn")
}

func main() {
	sys, err := dbus.SystemBus()
	if err != nil {
		fmt.Fprintln(os.Stderr, "system bus:", err)
		os.Exit(1)
	}
	if len(os.Args) > 1 {
		if err := cli(sys, os.Args[1:]); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}
	ses, err := dbus.SessionBus()
	if err != nil {
		fmt.Fprintln(os.Stderr, "session bus:", err)
		os.Exit(1)
	}
	s := &service{sys: sys, last: time.Now()}
	ses.Export(s, objPath, busName)
	node := &introspect.Node{Name: objPath, Interfaces: []introspect.Interface{
		introspect.IntrospectData,
		{Name: busName, Methods: introspect.Methods(s)},
	}}
	ses.Export(introspect.NewIntrospectable(node), objPath, "org.freedesktop.DBus.Introspectable")
	reply, err := ses.RequestName(busName, dbus.NameFlagDoNotQueue)
	if err != nil || reply != dbus.RequestNameReplyPrimaryOwner {
		fmt.Fprintln(os.Stderr, "name already taken or error:", err)
		os.Exit(1)
	}
	for range time.Tick(30 * time.Second) {
		s.mu.Lock()
		idle := time.Since(s.last) > idleFor && s.st.Busy == ""
		s.mu.Unlock()
		if idle {
			os.Exit(0)
		}
	}
}
