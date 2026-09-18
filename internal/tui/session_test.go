package tui

import (
	"os"
	"strings"
	"testing"
)

func TestSessionRoundTrip(t *testing.T) {
	t.Setenv("SILTIDE_STATE_DIR", t.TempDir())
	e := demoEngine(t)

	// a first run lands on the default view and leaves a session behind
	m := New(e, Options{Theme: NewTheme("mono", nil)})
	if m.tab != tabOverview {
		t.Fatalf("a first run opened on tab %d", m.tab)
	}
	m.setTab(tabProcesses)
	m.setSearch("util>50")
	m.node, m.ns = "node-a", "ml"
	m.sortCol, m.sortDesc = 3, false
	if err := m.saveSession(); err != nil {
		t.Fatal(err)
	}

	// the next one opens where it was left
	next := New(e, Options{Theme: NewTheme("mono", nil)})
	if next.tab != tabProcesses || next.search != "util>50" || next.node != "node-a" || next.ns != "ml" {
		t.Fatalf("restored tab %d filter %q node %q ns %q", next.tab, next.search, next.node, next.ns)
	}
	if next.sortCol != 3 || next.sortDesc {
		t.Errorf("restored sort %d desc %v", next.sortCol, next.sortDesc)
	}
	if next.filter.Empty() {
		t.Error("the restored filter was not parsed")
	}

	// a run that should not disturb it neither reads nor writes
	demo := New(e, Options{Theme: NewTheme("mono", nil), NoSession: true})
	if demo.tab != tabOverview || demo.search != "" {
		t.Errorf("a no-session run restored tab %d filter %q", demo.tab, demo.search)
	}
	demo.setTab(tabEvents)
	if demo.session {
		t.Error("a no-session run would still save")
	}

	// nothing readable is simply no session
	if err := os.WriteFile(sessionPath(), []byte("{{{"), 0o644); err != nil {
		t.Fatal(err)
	}
	if broken := New(e, Options{Theme: NewTheme("mono", nil)}); broken.tab != tabOverview {
		t.Errorf("a broken session file opened tab %d", broken.tab)
	}
}

func TestOpenFromFlags(t *testing.T) {
	t.Setenv("SILTIDE_STATE_DIR", t.TempDir())
	e := demoEngine(t)

	m := New(e, Options{Theme: NewTheme("mono", nil), Open: OpenView("", "processes", "util>80", "node-b", "ml")})
	if m.tab != tabProcesses || m.search != "util>80" || m.node != "node-b" || m.ns != "ml" {
		t.Fatalf("tab %d filter %q node %q ns %q", m.tab, m.search, m.node, m.ns)
	}

	// a prefix is enough, the way the command bar takes one
	if got := New(e, Options{Theme: NewTheme("mono", nil), Open: OpenView("", "proc", "", "", "")}); got.tab != tabProcesses {
		t.Errorf("--tab proc opened %d", got.tab)
	}
	// and a name that matches nothing says so instead of opening something else
	bad := New(e, Options{Theme: NewTheme("mono", nil), Open: OpenView("", "nope", "", "", "")})
	if bad.tab != tabOverview || !strings.Contains(bad.notice, "no tab called nope") {
		t.Errorf("tab %d notice %q", bad.tab, bad.notice)
	}

	// a saved view opens by name, and the flags still win over what it held
	seed := New(e, Options{Theme: NewTheme("mono", nil)})
	seed.setTab(tabHealth)
	seed.setSearch("vendor:nvidia")
	seed.command("bookmark save incident")

	opened := New(e, Options{Theme: NewTheme("mono", nil), Open: OpenView("incident", "", "", "", "")})
	if opened.tab != tabHealth || opened.search != "vendor:nvidia" {
		t.Fatalf("the bookmark did not open: tab %d filter %q", opened.tab, opened.search)
	}
	both := New(e, Options{Theme: NewTheme("mono", nil), Open: OpenView("incident", "events", "", "", "")})
	if both.tab != tabEvents || both.search != "vendor:nvidia" {
		t.Errorf("a flag should override the saved view: tab %d filter %q", both.tab, both.search)
	}

	// asking for a view means not restoring the last one
	last := New(e, Options{Theme: NewTheme("mono", nil)})
	last.setTab(tabLinks)
	if err := last.saveSession(); err != nil {
		t.Fatal(err)
	}
	flagged := New(e, Options{Theme: NewTheme("mono", nil), Open: OpenView("", "memory", "", "", "")})
	if flagged.tab != tabMemory {
		t.Errorf("the session won over the flag: tab %d", flagged.tab)
	}
}
