package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

// press sends one key and runs whatever command it produced, with the
// clipboard captured rather than written to a terminal.
func markModel(t *testing.T, tab int) (*Model, *string) {
	t.Helper()
	m := New(demoEngine(t), Options{Theme: NewTheme("default", nil)})
	m.width, m.height = 200, 60
	m.setTab(tab)
	var got string
	old := copyText
	copyText = func(s string) { got = s }
	t.Cleanup(func() { copyText = old })
	return &m, &got
}

func key(m *Model, s string) tea.Cmd {
	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)})
	*m = next.(Model)
	return cmd
}

func run(cmd tea.Cmd) {
	if cmd != nil {
		cmd()
	}
}

func TestMarkFollowsTheRowAndNotThePosition(t *testing.T) {
	m, _ := markModel(t, tabProcesses)
	rows := m.markable()
	if len(rows) < 3 {
		t.Fatalf("the demo fleet has %d processes", len(rows))
	}
	m.sel = 1
	want := rows[1].key
	key(m, "x")
	if !m.marked[want] {
		t.Fatalf("x did not mark row 1")
	}
	if m.sel != 2 {
		t.Fatalf("the cursor is at %d, marking should move it on", m.sel)
	}
	// Reverse the sort: the marked row is somewhere else now, and still marked.
	key(m, "S")
	after := m.markable()
	found := -1
	for i, r := range after {
		if r.key == want {
			found = i
		}
	}
	if found < 0 {
		t.Fatal("the marked row left the view")
	}
	if !m.markedRows()[found] {
		t.Fatalf("the mark did not follow the row to position %d", found)
	}
}

func TestMarkAllThenClears(t *testing.T) {
	m, _ := markModel(t, tabProcesses)
	n := len(m.markable())
	key(m, "X")
	if len(m.marksHere()) != n {
		t.Fatalf("X marked %d of %d", len(m.marksHere()), n)
	}
	key(m, "X")
	if len(m.marksHere()) != 0 {
		t.Fatalf("X again left %d marked", len(m.marksHere()))
	}
	if !strings.Contains(m.notice, "cleared") {
		t.Fatalf("notice %q", m.notice)
	}
}

func TestYankCopiesTheCommandForTheMarkedProcesses(t *testing.T) {
	m, got := markModel(t, tabProcesses)
	rows := m.markable()
	m.sel = 0
	key(m, "x")
	key(m, "x") // and the row after it
	run(key(m, "Y"))

	want := "kill " + rows[0].id + " " + rows[1].id
	if *got != want {
		t.Fatalf("copied %q, want %q", *got, want)
	}
	if !strings.Contains(m.notice, want) {
		t.Fatalf("the notice does not repeat the command: %q", m.notice)
	}
}

func TestYankIDsWithNothingMarkedTakesTheRowUnderTheCursor(t *testing.T) {
	m, got := markModel(t, tabProcesses)
	m.sel = 2
	run(key(m, "y"))
	if *got != m.markable()[2].id {
		t.Fatalf("copied %q, want the row under the cursor, %q", *got, m.markable()[2].id)
	}
}

// One kubectl call cannot span two namespaces, so two namespaces are two
// commands rather than one that fails.
func TestMarkCommandSplitsKubectlByNamespace(t *testing.T) {
	rows := []markRow{
		{id: "chat-api-0", ns: "inference"},
		{id: "llama-70b-pretrain-0", ns: "ml"},
		{id: "chat-api-1", ns: "inference"},
	}
	got := markCommand(tabKube, rows)
	want := "kubectl delete pod chat-api-0 chat-api-1 -n inference\nkubectl delete pod llama-70b-pretrain-0 -n ml"
	if got != want {
		t.Fatalf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestMarkCommandIsEmptyWhereThereIsNothingToRun(t *testing.T) {
	if got := markCommand(tabDevices, []markRow{{id: "nvidia-0"}}); got != "" {
		t.Fatalf("devices produced %q; y copies the ids there", got)
	}
}

// The whole point of the design: no key in the interface sends anything.
func TestYankNeverRunsTheCommand(t *testing.T) {
	m, got := markModel(t, tabProcesses)
	key(m, "X")
	run(key(m, "Y"))
	if !strings.HasPrefix(*got, "kill ") {
		t.Fatalf("copied %q", *got)
	}
	// The clipboard is the only side effect: the processes are still listed.
	if len(m.markable()) == 0 {
		t.Fatal("the rows went away, so something acted on them")
	}
}

// A mark is a colour on a row, not a change to what the row says.
func TestMarkedRowIsRecolouredAndNotRewritten(t *testing.T) {
	// lipgloss draws nothing under go test unless it is told the terminal can
	// take colour, and a mark is entirely colour.
	old := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.TrueColor)
	t.Cleanup(func() { lipgloss.SetColorProfile(old) })

	m, _ := markModel(t, tabProcesses)
	id := m.markable()[0].id

	m.sel = 3 // the cursor somewhere else, so this is the mark and not the cursor
	before := rowWith(t, m.View(), id)
	m.sel = 0
	key(m, "x")
	m.sel = 3
	after := rowWith(t, m.View(), id)

	if before == after {
		t.Fatal("the marked row is drawn exactly as it was")
	}
	if sgr.ReplaceAllString(before, "") != sgr.ReplaceAllString(after, "") {
		t.Fatalf("the row's text changed, not only its colour:\n%q\n%q",
			sgr.ReplaceAllString(before, ""), sgr.ReplaceAllString(after, ""))
	}
}

// rowWith is the rendered line that begins with a given id.
func rowWith(t *testing.T, view, id string) string {
	t.Helper()
	for _, line := range strings.Split(view, "\n") {
		if strings.HasPrefix(strings.TrimSpace(sgr.ReplaceAllString(line, "")), id+" ") {
			return line
		}
	}
	t.Fatalf("no row for %s in the view", id)
	return ""
}

func TestMarkingSaysSoOnATabThatCannotMark(t *testing.T) {
	m, _ := markModel(t, tabHealth)
	key(m, "x")
	if !strings.Contains(m.notice, "nothing to mark") {
		t.Fatalf("notice %q", m.notice)
	}
}
