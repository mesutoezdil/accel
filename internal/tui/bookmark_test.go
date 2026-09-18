package tui

import (
	"os"
	"strings"
	"testing"
)

func TestBookmarkSaveOpenRemove(t *testing.T) {
	t.Setenv("SILTIDE_STATE_DIR", t.TempDir())
	e := demoEngine(t)
	m := New(e, Options{Theme: NewTheme("mono", nil)})

	m.command("bookmark")
	if !strings.Contains(m.notice, "no bookmarks yet") {
		t.Errorf("notice %q", m.notice)
	}

	m.setTab(tabProcesses)
	m.setSearch("ns:inference util>10")
	m.node, m.ns = "node-a", "inference"
	m.command("bookmark save incident")
	if !strings.Contains(m.notice, "saved incident") {
		t.Fatalf("notice %q", m.notice)
	}
	if _, err := os.Stat(bookmarkPath()); err != nil {
		t.Fatalf("bookmark file: %v", err)
	}

	// a fresh model reads the file, and opening restores every part of the view
	m2 := New(e, Options{Theme: NewTheme("mono", nil)})
	m2.setTab(tabOverview)
	m2.command("bookmark incident")
	if m2.tab != tabProcesses {
		t.Errorf("tab %d, want processes", m2.tab)
	}
	if m2.search != "ns:inference util>10" || m2.node != "node-a" || m2.ns != "inference" {
		t.Errorf("restored filter %q node %q ns %q", m2.search, m2.node, m2.ns)
	}
	if m2.filter.Empty() {
		t.Error("the restored filter was not parsed")
	}

	m2.command("bookmark")
	if !strings.Contains(m2.notice, "incident") {
		t.Errorf("list notice %q", m2.notice)
	}
	m2.command("bookmark rm incident")
	if !strings.Contains(m2.notice, "removed incident") {
		t.Errorf("notice %q", m2.notice)
	}
	if len(New(e, Options{}).marks) != 0 {
		t.Error("a removed bookmark came back")
	}
}

func TestBookmarkSaveReplacesAndReportsUnknown(t *testing.T) {
	t.Setenv("SILTIDE_STATE_DIR", t.TempDir())
	e := demoEngine(t)
	m := New(e, Options{Theme: NewTheme("mono", nil)})

	m.setTab(tabEvents)
	m.command("bookmark save watch")
	m.setTab(tabHealth)
	m.setSearch("vendor:nvidia")
	m.command("bookmark save WATCH") // the same name in another case replaces it, and keeps the new spelling
	if len(m.marks) != 1 {
		t.Fatalf("saving over a name kept %d bookmarks", len(m.marks))
	}
	m.setTab(tabOverview)
	m.command("bookmark watch")
	if m.tab != tabHealth || m.search != "vendor:nvidia" {
		t.Errorf("the second save did not replace the first: tab %d filter %q", m.tab, m.search)
	}

	m.command("bookmark nope")
	if !strings.Contains(m.notice, "no bookmark nope") || !strings.Contains(m.notice, "WATCH") {
		t.Errorf("notice %q", m.notice)
	}
	m.command("bookmark save")
	if !strings.Contains(m.notice, "bookmark save <name>") {
		t.Errorf("notice %q", m.notice)
	}
}

func TestBookmarkCompletion(t *testing.T) {
	t.Setenv("SILTIDE_STATE_DIR", t.TempDir())
	e := demoEngine(t)
	m := New(e, Options{Theme: NewTheme("mono", nil)})
	m.command("bookmark save hot-nodes")
	if got := m.complete("bookmark h"); got != "bookmark hot-nodes" {
		t.Errorf("complete(%q) = %q", "bookmark h", got)
	}
	if got := m.complete("book"); got != "bookmark" {
		t.Errorf("complete(%q) = %q", "book", got)
	}
}
