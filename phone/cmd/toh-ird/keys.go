package main

import (
	"strings"
	"unicode"
)

// Button names in .ir files vary ("Vol_up", "VOL+", "Volume Up"...). keyFor maps a name to the slot key the app
// uses to put the button in its proper place; unknown names get "" and go to the "more buttons" area.
var aliases = map[string][]string{
	"power":    {"power", "pwr", "onoff", "poweronoff", "powertoggle", "standby", "toggle"},
	"on":       {"on", "poweron"},
	"off":      {"off", "poweroff"},
	"input":    {"input", "source", "src", "inputs", "av", "tvav", "inputsource", "sources", "inputselect"},
	"mute":     {"mute", "muting", "mutetoggle", "silence"},
	"vol_up":   {"volup", "vol+", "volumeup", "volume+", "vplus", "volplus", "volu", "volumeplus"},
	"vol_dn":   {"voldn", "voldown", "vol-", "volumedown", "volume-", "vminus", "volminus", "vold", "volumeminus"},
	"ch_up":    {"chnext", "chup", "ch+", "channelup", "channel+", "chplus", "progup", "prog+", "p+", "pageup", "channelnext"},
	"ch_dn":    {"chprev", "chdn", "chdown", "ch-", "channeldown", "channel-", "chminus", "progdown", "prog-", "p-", "pagedown", "channelprev"},
	"up":       {"up", "arrowup", "navup", "cursorup", "dpadup"},
	"down":     {"down", "arrowdown", "navdown", "cursordown", "dpaddown", "dn"},
	"left":     {"left", "arrowleft", "navleft", "cursorleft", "dpadleft"},
	"right":    {"right", "arrowright", "navright", "cursorright", "dpadright"},
	"ok":       {"ok", "enter", "select", "confirm", "center", "dpadcenter"},
	"back":     {"back", "return", "exit", "goback", "ret"},
	"home":     {"home", "smarthub", "homemenu", "smarttv"},
	"menu":     {"menu", "settings", "setup", "options", "tools", "setting"},
	"info":     {"info", "display", "osd", "information", "disp"},
	"guide":    {"guide", "epg", "tvguide"},
	"play":     {"play"},
	"pause":    {"pause"},
	"playpause": {"playpause", "pp", "play/pause"},
	"stop":     {"stop"},
	"prev":     {"prev", "previous", "skipback", "skipprev", "prevtrack", "skipbackward"},
	"next":     {"next", "skipnext", "skipfwd", "nexttrack", "skip", "skipforward"},
	"rew":      {"rew", "rewind", "rwd", "fastback", "fr", "fastrewind"},
	"ffwd":     {"ff", "ffwd", "fastforward", "fwd", "fastfwd", "fastfo", "forward"},
	"rec":      {"rec", "record"},
	"red":      {"red"},
	"green":    {"green"},
	"yellow":   {"yellow"},
	"blue":     {"blue"},
	"temp_up":  {"tempup", "temp+", "temperatureup", "templus", "tup"},
	"temp_dn":  {"tempdn", "tempdown", "temp-", "temperaturedown", "tempminus", "tdown"},
	"cool_hi":  {"coolhi", "coolhigh"},
	"cool_lo":  {"coollo", "coollow"},
	"heat_hi":  {"heathi", "heathigh"},
	"heat_lo":  {"heatlo", "heatlow"},
	"dry":      {"dh", "dry", "dehumidify", "dehum"},
	"fan":      {"fan", "fanspeed", "speed", "wind"},
	"mode":     {"mode"},
	"swing":    {"swing", "oscillate", "osc", "oscillation"},
	"timer":    {"timer", "sleep"},
}

var aliasKey = func() map[string]string {
	m := map[string]string{}
	for k, list := range aliases {
		for _, a := range list {
			m[a] = k
		}
	}
	for d := '0'; d <= '9'; d++ {
		for _, p := range []string{"", "num", "key", "digit", "btn", "number", "n"} {
			m[p+string(d)] = "n" + string(d)
		}
	}
	return m
}()

func normalize(name string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(name) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '+' || r == '-' || r == '/' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func keyFor(name string) string {
	return aliasKey[normalize(name)]
}
