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
}

// Keymap resolves keys to actions.
type Keymap map[string]Action

// NewKeymap applies config overrides (action: "key" or "key1,key2").
func NewKeymap(overrides map[string]string) Keymap {
	km := Keymap{}
	for act, keys := range defaultKeys {
		if o, ok := overrides[string(act)]; ok {
			keys = strings.Split(o, ",")
		}
		for _, k := range keys {
			km[strings.TrimSpace(k)] = act
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
