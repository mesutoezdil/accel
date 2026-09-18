package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func names(defs []commandDef) []string {
	out := make([]string, len(defs))
	for i, d := range defs {
		out[i] = d.name
	}
	return out
}

func has(defs []commandDef, want string) bool {
	for _, d := range defs {
		if d.name == want {
			return true
		}
	}
	return false
}

func TestSuggestions(t *testing.T) {
	t.Setenv("SILTIDE_STATE_DIR", t.TempDir())
	e := demoEngine(t)
	m := New(e, Options{Theme: NewTheme("mono", nil)})
	m.width, m.height = 160, 40

	all := m.suggestions("")
	if len(all) < len(commands)+len(tabs) {
		t.Fatalf("an empty bar offers %d entries, want every command and tab", len(all))
	}
	for _, c := range commands {
		if c.help == "" {
			t.Errorf("command %q has no description", c.name)
		}
		if !has(all, c.name) {
			t.Errorf("command %q is not offered", c.name)
		}
	}

	if got := names(m.suggestions("the")); len(got) == 0 || got[0] != "theme" {
		t.Errorf("suggestions(\"the\") = %v", got)
	}
	if got := m.suggestions(":re"); !has(got, "reload") || !has(got, "refresh") {
		t.Errorf("suggestions(\":re\") = %v", names(got))
	}
	if got := m.suggestions("dev"); !has(got, "devices") {
		t.Errorf("a tab name should be offered: %v", names(got))
	}

	// once a command takes an argument, the values it accepts are offered
	if got := m.suggestions("theme "); !has(got, "theme nord") || !has(got, "theme gruvbox") {
		t.Errorf("theme values missing: %v", names(got))
	}
	if got := m.suggestions("theme no"); !has(got, "theme nord") || has(got, "theme gruvbox") {
		t.Errorf("theme no -> %v", names(got))
	}
	if got := m.suggestions("metric "); !has(got, "metric power") {
		t.Errorf("metric values missing: %v", names(got))
	}
	if got := m.suggestions("window "); !has(got, "window 1h") {
		t.Errorf("window values missing: %v", names(got))
	}
	if got := m.suggestions("bookmark "); !has(got, "bookmark save") || !has(got, "bookmark rm") {
		t.Errorf("bookmark verbs missing: %v", names(got))
	}
	if got := m.suggestions("pause "); got != nil {
		t.Errorf("a command with no argument should offer nothing: %v", names(got))
	}
	if got := m.suggestions("nonsense"); len(got) != 0 {
		t.Errorf("unknown input offers %v", names(got))
	}
}

func TestSuggestionsFollowSavedBookmarks(t *testing.T) {
	t.Setenv("SILTIDE_STATE_DIR", t.TempDir())
	e := demoEngine(t)
	m := New(e, Options{Theme: NewTheme("mono", nil)})
	m.setTab(tabProcesses)
	m.setSearch("util>50")
	m.command("bookmark save incident")

	got := m.suggestions("bookmark inc")
	if !has(got, "bookmark incident") {
		t.Fatalf("a saved view should be offered: %v", names(got))
	}
	for _, d := range got {
		if d.name == "bookmark incident" && !strings.Contains(d.help, "util>50") {
			t.Errorf("the suggestion should say what the view holds: %q", d.help)
		}
	}
}

func TestCommandBarShowsAndMovesThroughSuggestions(t *testing.T) {
	t.Setenv("SILTIDE_STATE_DIR", t.TempDir())
	e := demoEngine(t)
	m := New(e, Options{Theme: NewTheme("mono", nil)})
	m.width, m.height = 160, 40

	press := func(s string) {
		var msg tea.KeyMsg
		switch s {
		case "up", "down", "tab":
			msg = tea.KeyMsg{Type: map[string]tea.KeyType{"up": tea.KeyUp, "down": tea.KeyDown, "tab": tea.KeyTab}[s]}
		default:
			msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
		}
		mm, _ := m.Update(msg)
		m = mm.(Model)
	}

	press(":")
	press("t")
	view := m.View()
	if !strings.Contains(view, "theme") {
		t.Fatalf("the bar shows no suggestions:\n%s", view)
	}
	if got := len(strings.Split(view, "\n")); got > m.height {
		t.Errorf("the interface is %d lines in a %d row terminal with the bar open", got, m.height)
	}
	if rows := m.suggestionRows(); rows == 0 || rows > suggestionMax {
		t.Errorf("suggestion rows %d", rows)
	}

	// the cursor moves and stays inside the list
	press("down")
	if m.cmdSel != 1 {
		t.Errorf("down left the cursor at %d", m.cmdSel)
	}
	for range 20 {
		press("down")
	}
	if m.cmdSel >= len(m.suggestions(m.input)) {
		t.Errorf("the cursor ran past the list: %d of %d", m.cmdSel, len(m.suggestions(m.input)))
	}
	for range 30 {
		press("up")
	}
	if m.cmdSel != 0 {
		t.Errorf("up left the cursor at %d", m.cmdSel)
	}

	press("tab")
	if !strings.HasPrefix(m.input, "t") || m.cmdSel != 0 {
		t.Errorf("tab gave input %q cursor %d", m.input, m.cmdSel)
	}
}
