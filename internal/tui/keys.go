package tui

import "strings"

// Action is something a key can do.
type Action string

const (
	ActQuit     Action = "quit"
	ActUp       Action = "up"
	ActDown     Action = "down"
	ActPageUp   Action = "page_up"
	ActPageDown Action = "page_down"
	ActTop      Action = "top"
	ActBottom   Action = "bottom"
	ActEnter    Action = "enter"
	ActBack     Action = "back"
	ActNextTab  Action = "next_tab"
	ActPrevTab  Action = "prev_tab"
	ActPause    Action = "pause"
	ActRefresh  Action = "refresh"
	ActHelp     Action = "help"
	ActCommand  Action = "command"
	ActSearch   Action = "search"
	ActSortNext Action = "sort_next"
	ActSortRev  Action = "sort_reverse"
	ActHistBack Action = "hist_back"
	ActHistFwd  Action = "hist_forward"
	ActHistJump Action = "hist_jump"
	ActHistLive Action = "hist_live"
	ActMetric   Action = "metric_next"
	ActMetricPr Action = "metric_prev"
	ActZoomIn   Action = "zoom_in"
	ActZoomOut  Action = "zoom_out"
	ActPrevDev  Action = "prev_device"
	ActNextDev  Action = "next_device"
	ActDescribe Action = "describe"
	ActLogs     Action = "logs"
	ActNextCont Action = "next_container"
	ActWrap     Action = "wrap"
	ActDash     Action = "dashboard"
	ActNextNode Action = "next_node"
	ActPrevNode Action = "prev_node"
	ActExport   Action = "export"
	ActMark     Action = "mark"
	ActMarkAll  Action = "mark_all"
	ActYank     Action = "yank"
	ActYankCmd  Action = "yank_command"
)

// defaultKeys maps actions to keys; several keys per action are allowed.
var defaultKeys = map[Action][]string{
	ActQuit: {"q", "ctrl+c"}, ActUp: {"up", "k"}, ActDown: {"down", "j"},
	ActPageUp: {"pgup", "ctrl+u"}, ActPageDown: {"pgdown", "ctrl+d"}, ActTop: {"home", "g"}, ActBottom: {"end", "G"},
	ActEnter: {"enter"}, ActBack: {"esc"},
	ActNextTab: {"tab", "right", "]"}, ActPrevTab: {"shift+tab", "left", "["},
	ActPause: {"p", " "}, ActRefresh: {"r"}, ActHelp: {"?"}, ActCommand: {":"}, ActSearch: {"/"},
	ActSortNext: {"s"}, ActSortRev: {"S"},
	ActHistBack: {","}, ActHistFwd: {"."}, ActHistJump: {"<", ">"}, ActHistLive: {"n"},
	ActMetric: {"m"}, ActMetricPr: {"M"}, ActZoomIn: {"+", "="}, ActZoomOut: {"-"},
	ActPrevDev: {"{"}, ActNextDev: {"}"},
	ActDescribe: {"d"}, ActLogs: {"l"}, ActNextCont: {"c"}, ActWrap: {"w"}, ActDash: {"D"},
	ActNextNode: {"ctrl+n"}, ActPrevNode: {"ctrl+p"}, ActExport: {"ctrl+e"},
	ActMark: {"x"}, ActMarkAll: {"X"}, ActYank: {"y"}, ActYankCmd: {"Y"},
}

// Keymap resolves keys to actions.
type Keymap map[string]Action

// NewKeymap applies config overrides (action: "key" or "key1,key2").
//
// The defaults go down first and the config on top, in two passes rather than
// one. A key the config asks for is taken off whatever holds it by default:
// binding quit to x when x already marks a row used to be decided by map
// order, which is to say by a coin toss on every start.
func NewKeymap(overrides map[string]string) Keymap {
	km := Keymap{}
	for act, keys := range defaultKeys {
		for _, k := range keys {
			km[strings.TrimSpace(k)] = act
		}
	}
	for name, spec := range overrides {
		act := Action(name)
		if _, known := defaultKeys[act]; !known {
			continue
		}
		for k, held := range km {
			if held == act {
				delete(km, k) // this action's own defaults give way to the config
			}
		}
		for _, k := range strings.Split(spec, ",") {
			if k = strings.TrimSpace(k); k != "" {
				km[k] = act // and the key gives way to this action
			}
		}
	}
	return km
}

// Keys lists the keys bound to an action, for the help screen.
func (km Keymap) Keys(a Action) string {
	var out []string
	for _, k := range defaultKeys[a] {
		if km[k] == a {
			out = append(out, k)
		}
	}
	for k, act := range km {
		if act == a && !contains(out, k) {
			out = append(out, k)
		}
	}
	for i, k := range out {
		if k == " " {
			out[i] = "space"
		}
	}
	return strings.Join(out, " ")
}

func contains(xs []string, s string) bool {
	for _, x := range xs {
		if x == s {
			return true
		}
	}
	return false
}
