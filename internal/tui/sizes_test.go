package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// TestEveryTabSurvivesEverySize renders every tab at sizes from a single
// cell up to a wide window, after a run of keys, filters, and commands. A
// terminal can be any shape while it is being resized, and the monitor may
// not fall over at any of them.
func TestEveryTabSurvivesEverySize(t *testing.T) {
	e := demoEngine(t)
	keys := []string{"j", "k", "s", "S", ",", ".", "n", "m", "M", "+", "-", "{", "}", "c", "w", "p", "enter", "esc", "tab"} // d and l shell out to kubectl
	for _, size := range [][2]int{{1, 1}, {5, 3}, {20, 10}, {80, 24}, {200, 60}} {
		for tab := range TabKeys() {
			func() {
				defer func() {
					if r := recover(); r != nil {
						t.Errorf("panic at %dx%d tab %d: %v", size[0], size[1], tab, r)
					}
				}()
				m := New(e, Options{Theme: NewTheme("default", nil)})
				mm, _ := m.Update(tea.WindowSizeMsg{Width: size[0], Height: size[1]})
				m = mm.(Model)
				m.setTab(tab)
				_ = m.View()
				for _, k := range keys {
					var msg tea.KeyMsg
					switch k {
					case "enter":
						msg = tea.KeyMsg{Type: tea.KeyEnter}
					case "esc":
						msg = tea.KeyMsg{Type: tea.KeyEsc}
					case "tab":
						msg = tea.KeyMsg{Type: tea.KeyTab}
					default:
						msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(k)}
					}
					mm, _ := m.Update(msg)
					m = mm.(Model)
					_ = m.View()
				}
				for _, q := range []string{"util>50", "nothingmatchesthis", ""} {
					m.setSearch(q)
					_ = m.View()
				}
				for _, cmd := range []string{"compare 0 1", "window 5m", "metric power", "dashboard"} {
					m.command(cmd)
					_ = m.View()
				}
			}()
		}
	}
}
