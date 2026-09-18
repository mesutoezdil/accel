package tui

import (
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
