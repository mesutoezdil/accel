package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/mesutoezdil/siltide/internal/collect"
	"github.com/mesutoezdil/siltide/internal/provider/sim"
	"github.com/mesutoezdil/siltide/internal/tui"
)

// beat is one move of the demo: keys someone presses, then the frames the
// view is held for. typed is entered one character per frame, the way a
// person fills the filter bar.
type beat struct {
	key   string
	typed string
	hold  int
}

// storyboard is the demo loop: the fleet, a device and its processes, a
// filter typed live, history scrubbed back, then the fabric and health
// views, ending where it started so the loop closes cleanly.
var storyboard = []beat{
	{hold: 12},
	{key: "j", hold: 3}, {key: "j", hold: 10},
	{key: "2", hold: 10},
	{key: "j", hold: 3}, {key: "j", hold: 12},
	{key: "/", hold: 2}, {typed: "util>80", hold: 2}, {key: "enter", hold: 14},
	{key: "esc", hold: 4},
	{key: "3", hold: 12},
	{key: "8", hold: 10},
	{key: ",", hold: 2}, {key: ",", hold: 2}, {key: ",", hold: 8},
	{key: "n", hold: 4}, {key: "m", hold: 10},
	{key: "7", hold: 12},
	{key: "H", hold: 12},
	{key: "D", hold: 14},
	{key: "1", hold: 10},
}

// clockText matches the timestamp in the header, which the film rewrites so
// the clock moves with the simulated fleet rather than with the render.
var clockText = regexp.MustCompile(`\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}`)

// film writes one SVG per frame of the storyboard. The simulated fleet runs
// on a clock of its own, a second per frame, so numbers move between frames
// instead of standing still for the length of the render.
func film(o options, eng *collect.Engine, th tui.Theme, host string) error {
	if err := os.MkdirAll(o.film, 0o755); err != nil {
		return err
	}
	old, _ := filepath.Glob(filepath.Join(o.film, "frame-*.svg"))
	for _, f := range old {
		if err := os.Remove(f); err != nil {
			return err
		}
	}

	clock := time.Now().Truncate(time.Minute)
	sim.Now = func() time.Time { return clock }
	defer func() { sim.Now = time.Now }()

	var pressed []tea.KeyMsg
	n := 0
	// The fleet steps a second every other frame: fast enough to read as
	// live at 10 frames a second, and it halves the animation, since a
	// frame that repeats the one before it costs a delay and nothing else.
	shoot := func() error {
		if n%2 == 0 {
			clock = clock.Add(time.Second)
			eng.Collect(context.Background())
		}
		return frame(o, eng, th, host, pressed, clock, &n)
	}
	for _, b := range storyboard {
		for _, r := range b.typed {
			pressed = append(pressed, keyMsg(string(r)))
			if err := shoot(); err != nil {
				return err
			}
		}
		if b.key != "" {
			pressed = append(pressed, keyMsg(b.key))
		}
		for range max(b.hold, 1) {
			if err := shoot(); err != nil {
				return err
			}
		}
	}
	fmt.Printf("%s: %d frames\n", o.film, n)
	return nil
}

// frame replays the keys pressed so far against the latest snapshot and
// writes the view.
// The model reads its snapshot when it is built, so it is rebuilt per frame:
// that is what lets the numbers move while the keys stay where they were.
func frame(o options, eng *collect.Engine, th tui.Theme, host string, pressed []tea.KeyMsg, clock time.Time, n *int) error {
	m := tui.New(eng, tui.Options{Theme: th, Mouse: true, Currency: "$"})
	m = update(m, tea.WindowSizeMsg{Width: o.w, Height: o.h})
	for _, k := range pressed {
		m = update(m, k)
	}
	view := m.View()
	if o.host != "" {
		view = strings.ReplaceAll(view, host, o.host)
	}
	view = dropBadge(view)
	view = clockText.ReplaceAllString(view, clock.Format("2006-01-02 15:04:05"))
	view = padRows(view, o.h)
	*n++
	name := filepath.Join(o.film, fmt.Sprintf("frame-%04d.svg", *n))
	return os.WriteFile(name, []byte(svg(view, o.w, o.h, "siltide")), 0o644)
}

// padRows makes every frame exactly rows tall. A tab whose view is shorter
// would otherwise give the window a different height, and the animation has
// to keep one size from the first frame to the last. The filler is a reset
// escape rather than an empty line, which the renderer would trim off again.
func padRows(view string, rows int) string {
	lines := strings.Split(strings.TrimRight(view, "\n"), "\n")
	for len(lines) < rows {
		lines = append(lines, "\x1b[0m")
	}
	return strings.Join(lines[:rows], "\n")
}

// keyMsg turns a key name into the message Bubble Tea would deliver.
func keyMsg(k string) tea.KeyMsg {
	switch k {
	case "enter":
		return tea.KeyMsg{Type: tea.KeyEnter}
	case "esc":
		return tea.KeyMsg{Type: tea.KeyEsc}
	case "tab":
		return tea.KeyMsg{Type: tea.KeyTab}
	case "space":
		return tea.KeyMsg{Type: tea.KeySpace}
	}
	return key(k)
}
