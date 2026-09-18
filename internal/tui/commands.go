package tui

import (
	"sort"
	"strings"
)

// commandDef is one entry of the command bar: what to type, what it takes,
// and what it does. The bar, its suggestions and the help screen all read
// this, so a command cannot exist without a line describing it.
type commandDef struct {
	name string
	args string // placeholder shown after the name, "" when it takes none
	help string
}

// commands are every command the bar accepts, in the order they are offered.
var commands = []commandDef{
	{"bookmark", "<name> | save <name> | rm <name>", "open, record, or forget a saved view"},
	{"compare", "<dev> <dev> | <dev> <time>", "two devices side by side, or one at two times"},
	{"describe", "[pod]", "Kubernetes: describe the selected pod"},
	{"filter", "<query>", "set the filter, as / does"},
	{"live", "", "leave the time machine and follow now"},
	{"logs", "[pod]", "Kubernetes: container logs for the selected pod"},
	{"metric", "<name>", "which metric the history charts draw"},
	{"node", "[name]", "limit every tab to one node"},
	{"ns", "[namespace]", "limit the Kubernetes tab to one namespace"},
	{"pause", "", "stop and resume collection"},
	{"refresh", "", "collect now, without waiting for the interval"},
	{"reload", "", "re-read the config file and apply it"},
	{"sort", "<column>", "sort the current table by a column"},
	{"theme", "<name>", "switch the colour theme"},
	{"window", "<duration>", "how much history the charts cover"},
}

// suggestions are the commands or argument values that still match input,
// best first. An empty input offers everything, which is how someone who has
// never opened the bar finds out what it holds.
func (m Model) suggestions(input string) []commandDef {
	// Only the left side is trimmed: a trailing space is what says the name
	// is finished and an argument is being typed.
	q := strings.TrimLeft(strings.TrimPrefix(strings.TrimLeft(input, " "), ":"), " ")
	name, rest, typing := strings.Cut(q, " ")
	if typing {
		return m.argSuggestions(strings.ToLower(name), strings.TrimLeft(rest, " "))
	}

	var out []commandDef
	for _, c := range commands {
		if strings.HasPrefix(c.name, strings.ToLower(q)) {
			out = append(out, c)
		}
	}
	for _, t := range tabs {
		if lower := strings.ToLower(t.name); strings.HasPrefix(lower, strings.ToLower(q)) {
			out = append(out, commandDef{lower, "", "go to the " + t.name + " tab (" + t.key + ")"})
		}
	}
	return out
}

// argSuggestions offers the values a command takes once its name is typed.
func (m Model) argSuggestions(name, arg string) []commandDef {
	var vals []commandDef
	switch name {
	case "theme":
		for _, t := range ThemeNames() {
			vals = append(vals, commandDef{t, "", "theme"})
		}
	case "bookmark", "bm":
		vals = append(vals, commandDef{"save", "<name>", "record the current view"}, commandDef{"rm", "<name>", "forget a saved view"})
		for _, b := range m.marks {
			vals = append(vals, commandDef{b.Name, "", strings.ToLower(b.Tab) + describeFilters(b)})
		}
	case "metric":
		for _, h := range histMetrics {
			vals = append(vals, commandDef{h.name, "", "history metric"})
		}
	case "window":
		for _, w := range []string{"5m", "15m", "30m", "1h", "6h", "24h"} {
			vals = append(vals, commandDef{w, "", "history window"})
		}
	case "sort":
		for _, c := range m.columnNames() {
			if c = strings.ToLower(strings.TrimSpace(c)); c != "" {
				vals = append(vals, commandDef{c, "", "column of this tab"})
			}
		}
	case "node":
		for _, n := range m.nodes() {
			vals = append(vals, commandDef{n.name, "", "node"})
		}
	case "ns", "namespace":
		seen := map[string]bool{}
		for _, p := range m.pods() {
			if !seen[p.ns] {
				seen[p.ns], vals = true, append(vals, commandDef{p.ns, "", "namespace"})
			}
		}
	case "tab":
		for _, t := range tabs {
			vals = append(vals, commandDef{strings.ToLower(t.name), "", "tab " + t.key})
		}
	case "describe", "logs":
		for _, p := range m.pods() {
			vals = append(vals, commandDef{p.name, "", p.ns})
		}
	default:
		return nil
	}

	sort.SliceStable(vals, func(i, j int) bool { return vals[i].name < vals[j].name })
	var out []commandDef
	for _, v := range vals {
		if strings.HasPrefix(strings.ToLower(v.name), strings.ToLower(arg)) {
			out = append(out, commandDef{name + " " + v.name, v.args, v.help})
		}
	}
	return out
}

// complete fills the bar with the suggestion under the cursor.
func (m Model) complete(input string) string {
	s := m.suggestions(input)
	if len(s) == 0 {
		return input
	}
	pick := s[min(max(m.cmdSel, 0), len(s)-1)]
	if pick.args != "" {
		return pick.name + " "
	}
	return pick.name
}

// suggestionMax is how many suggestions the bar shows at once. Enough to
// discover a command, short enough to leave the view it sits over readable.
const suggestionMax = 6

// suggestionRows is how many lines the list takes under the current input.
func (m Model) suggestionRows() int {
	if !m.cmdOpen {
		return 0
	}
	return min(len(m.suggestions(m.input)), suggestionMax)
}

// suggestionList draws the commands still matching what has been typed, with
// the one Tab would take highlighted, above the bar itself.
func (m Model) suggestionList() string {
	all := m.suggestions(m.input)
	if len(all) == 0 {
		return ""
	}
	sel := min(max(m.cmdSel, 0), len(all)-1)
	start := 0
	if sel >= suggestionMax {
		start = sel - suggestionMax + 1 // scroll the window down to the cursor
	}
	end := min(start+suggestionMax, len(all))

	th, w := m.th, max(m.width, 20)
	var b strings.Builder
	for i := start; i < end; i++ {
		c := all[i]
		left := c.name
		if c.args != "" {
			left += " " + c.args
		}
		line := pad("  "+left, min(w/2, 40)) + th.dim.Render(c.help)
		if i == sel {
			line = selected(th.sel, pad(line, w))
		}
		b.WriteString(trunc(line, w) + "\n")
	}
	return b.String()
}
