package tui

import (
	"fmt"
	"regexp"
	"strconv"
	"testing"
)

var hexColor = regexp.MustCompile(`^#[0-9a-f]{6}$`)

// TestPalettesAreComplete keeps every built-in theme on the same role set, so
// a preset can never leave a role to fall back to an uncoloured style.
func TestPalettesAreComplete(t *testing.T) {
	for name, p := range palettes {
		if len(p) != len(Roles) {
			t.Errorf("theme %s defines %d roles, want %d", name, len(p), len(Roles))
		}
		for _, role := range Roles {
			v, ok := p[role]
			if !ok {
				t.Errorf("theme %s is missing role %s", name, role)
				continue
			}
			if v == "" { // an empty colour is "use the terminal default"
				continue
			}
			if hexColor.MatchString(v) {
				continue
			}
			n, err := strconv.Atoi(v)
			if err != nil || n < 0 || n > 255 {
				t.Errorf("theme %s role %s: %q is neither #rrggbb nor an ANSI index", name, role, v)
			}
		}
	}
}

// TestPresetsLoad checks the presets the issue asked for are reachable by name
// and come back with their own colours, not the default fallback.
func TestPresetsLoad(t *testing.T) {
	for _, name := range []string{"nord", "dracula", "gruvbox", "catppuccin", "tokyo-night", "one-dark", "rose-pine", "monokai", "flexoki"} {
		th, err := LoadTheme(name, "", nil, false)
		if err != nil {
			t.Errorf("LoadTheme(%s): %v", name, err)
			continue
		}
		if th.Name != name {
			t.Errorf("LoadTheme(%s) named itself %s", name, th.Name)
		}
		if th.Colors["accent"] != palettes[name]["accent"] {
			t.Errorf("theme %s fell back to another palette", name)
		}
	}
}

func TestThemeNamesListsEveryPreset(t *testing.T) {
	names := ThemeNames()
	if len(names) != len(palettes) {
		t.Fatalf("ThemeNames returned %d of %d themes", len(names), len(palettes))
	}
	for i := 1; i < len(names); i++ {
		if names[i-1] >= names[i] {
			t.Fatalf("ThemeNames is not sorted: %v", names)
		}
	}
}

// TestEveryPaletteNamesItsBackground covers the role the capture renderer and
// the site player draw on. The terminal supplies its own background, so this
// is the one role that exists for everything else.
func TestEveryPaletteNamesItsBackground(t *testing.T) {
	for name, p := range palettes {
		bg, ok := p["bg"]
		if !ok || bg == "" {
			t.Errorf("theme %s names no background", name)
			continue
		}
		if !hexColor.MatchString(bg) {
			t.Errorf("theme %s has background %q, which a renderer cannot use", name, bg)
		}
	}
}

// TestPaperIsLight keeps the one light theme light: a reader on paper should
// not get a dark background with dark text on it.
func TestPaperIsLight(t *testing.T) {
	p, ok := palettes["paper"]
	if !ok {
		t.Fatal("there is no light theme")
	}
	if l := luminance(t, p["bg"]); l < 0.7 {
		t.Errorf("the paper background has luminance %.2f, which is not paper", l)
	}
	if l := luminance(t, p["text"]); l > 0.4 {
		t.Errorf("the paper text has luminance %.2f, which will not read on it", l)
	}
	for _, role := range []string{"ok", "warn", "crit", "mid", "info", "accent"} {
		if l := luminance(t, p[role]); l > 0.65 {
			t.Errorf("paper %s has luminance %.2f, too light for a light background", role, l)
		}
	}
}

// luminance is the rough brightness of a #rrggbb colour, 0 black to 1 white.
func luminance(t *testing.T, hex string) float64 {
	t.Helper()
	if !hexColor.MatchString(hex) {
		t.Fatalf("%q is not a hex colour", hex)
	}
	var r, g, b int
	if _, err := fmt.Sscanf(hex, "#%02x%02x%02x", &r, &g, &b); err != nil {
		t.Fatal(err)
	}
	return (0.2126*float64(r) + 0.7152*float64(g) + 0.0722*float64(b)) / 255
}
