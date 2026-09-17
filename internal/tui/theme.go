package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"go.yaml.in/yaml/v3"
)

// Roles are the semantic colours a theme defines. A theme file may set any
// subset; the rest come from the theme it extends.
var Roles = []string{
	"accent", "text", "dim", "title", "border",
	"ok", "mid", "warn", "crit", "info",
	"bar_low", "bar_mid", "bar_high", "bar_track", "spark",
	"tab_active_bg", "tab_active_fg", "selection_bg", "selection_fg",
}

var palettes = map[string]map[string]string{
	"default": {
		"accent": "12", "text": "", "dim": "8", "title": "12", "border": "8",
		"ok": "10", "mid": "11", "warn": "9", "crit": "13", "info": "14",
		"bar_low": "10", "bar_mid": "11", "bar_high": "9", "bar_track": "8", "spark": "10",
		"tab_active_bg": "12", "tab_active_fg": "0", "selection_bg": "", "selection_fg": "",
	},
	"mono": {
		"accent": "", "text": "", "dim": "8", "title": "", "border": "8",
		"ok": "", "mid": "", "warn": "", "crit": "", "info": "",
		"bar_low": "", "bar_mid": "", "bar_high": "", "bar_track": "8", "spark": "",
		"tab_active_bg": "", "tab_active_fg": "", "selection_bg": "", "selection_fg": "",
	},
	"solarized": {
		"accent": "#268bd2", "text": "#839496", "dim": "#586e75", "title": "#268bd2", "border": "#586e75",
		"ok": "#859900", "mid": "#b58900", "warn": "#dc322f", "crit": "#d33682", "info": "#2aa198",
		"bar_low": "#859900", "bar_mid": "#b58900", "bar_high": "#dc322f", "bar_track": "#073642", "spark": "#2aa198",
		"tab_active_bg": "#268bd2", "tab_active_fg": "#002b36", "selection_bg": "#073642", "selection_fg": "#eee8d5",
	},
	"dracula": {
		"accent": "#bd93f9", "text": "#f8f8f2", "dim": "#6272a4", "title": "#bd93f9", "border": "#6272a4",
		"ok": "#50fa7b", "mid": "#f1fa8c", "warn": "#ff5555", "crit": "#ff79c6", "info": "#8be9fd",
		"bar_low": "#50fa7b", "bar_mid": "#f1fa8c", "bar_high": "#ff5555", "bar_track": "#44475a", "spark": "#8be9fd",
		"tab_active_bg": "#bd93f9", "tab_active_fg": "#282a36", "selection_bg": "#44475a", "selection_fg": "#f8f8f2",
	},
	"amber": {
		"accent": "214", "text": "223", "dim": "137", "title": "214", "border": "137",
		"ok": "214", "mid": "208", "warn": "196", "crit": "199", "info": "222",
		"bar_low": "214", "bar_mid": "208", "bar_high": "196", "bar_track": "94", "spark": "214",
		"tab_active_bg": "214", "tab_active_fg": "0", "selection_bg": "94", "selection_fg": "223",
	},
	"ice": {
		"accent": "39", "text": "153", "dim": "67", "title": "39", "border": "67",
		"ok": "51", "mid": "45", "warn": "203", "crit": "201", "info": "87",
		"bar_low": "51", "bar_mid": "45", "bar_high": "203", "bar_track": "24", "spark": "87",
		"tab_active_bg": "39", "tab_active_fg": "0", "selection_bg": "24", "selection_fg": "153",
	},
}

// Theme is the set of styles by role.
type Theme struct {
	Name        string
	Colors      map[string]string
	Transparent bool

	accent, dim, ok, mid, warn, crit, info, text, title, bold, border lipgloss.Style
	sel, tab                                                          lipgloss.Style
}

// ThemeNames lists the built-in themes.
func ThemeNames() []string {
	out := make([]string, 0, len(palettes))
	for n := range palettes {
		out = append(out, n)
	}
	sort.Strings(out)
	return out
}

// themeFile is `~/.config/siltide/themes/<name>.yaml`.
type themeFile struct {
	Name    string            `yaml:"name"`
	Extends string            `yaml:"extends"`
	Colors  map[string]string `yaml:"colors"`
}

// LoadTheme resolves a theme by name: built-in, or a file in dir (with
// `extends` chains). Unknown names fall back to default with an error.
func LoadTheme(name, dir string, overrides map[string]string, transparent bool) (Theme, error) {
	colors, err := resolve(name, dir, 0)
	if err != nil {
		colors, _ = resolve("default", dir, 0)
	}
	for k, v := range overrides {
		colors[k] = v
	}
	t := build(name, colors)
	t.Transparent = transparent
	return t, err
}

func resolve(name, dir string, depth int) (map[string]string, error) {
	if depth > 5 {
		return nil, fmt.Errorf("theme %s: extends chain too deep", name)
	}
	if p, ok := palettes[name]; ok {
		out := map[string]string{}
		for k, v := range p {
			out[k] = v
		}
		return out, nil
	}
	if dir == "" {
		return nil, fmt.Errorf("unknown theme %q", name)
	}
	b, err := os.ReadFile(filepath.Join(dir, name+".yaml"))
	if err != nil {
		return nil, fmt.Errorf("unknown theme %q", name)
	}
	var f themeFile
	if err := yaml.Unmarshal(b, &f); err != nil {
		return nil, fmt.Errorf("theme %s: %w", name, err)
	}
	base := f.Extends
	if base == "" {
		base = "default"
	}
	out, err := resolve(base, dir, depth+1)
	if err != nil {
		return nil, err
	}
	for k, v := range f.Colors {
		if !contains(Roles, k) {
			return nil, fmt.Errorf("theme %s: unknown role %q (roles: %s)", name, k, strings.Join(Roles, ", "))
		}
		out[k] = v
	}
	return out, nil
}

// NewTheme builds a built-in theme with overrides.
func NewTheme(name string, overrides map[string]string) Theme {
	t, _ := LoadTheme(name, "", overrides, false)
	return t
}

func build(name string, p map[string]string) Theme {
	color := func(role string) lipgloss.Style {
		if p[role] == "" {
			return lipgloss.NewStyle()
		}
		return lipgloss.NewStyle().Foreground(lipgloss.Color(p[role]))
	}
	t := Theme{
		Name: name, Colors: p,
		accent: color("accent"), dim: color("dim"), ok: color("ok"), mid: color("mid"),
		warn: color("warn"), crit: color("crit"), info: color("info"), text: color("text"), border: color("border"),
	}
	t.title = color("title").Bold(true)
	t.bold = lipgloss.NewStyle().Bold(true)
	t.sel = lipgloss.NewStyle().Reverse(true)
	if p["selection_bg"] != "" {
		t.sel = lipgloss.NewStyle().Background(lipgloss.Color(p["selection_bg"]))
		if p["selection_fg"] != "" {
			t.sel = t.sel.Foreground(lipgloss.Color(p["selection_fg"]))
		}
	}
	t.tab = lipgloss.NewStyle().Reverse(true)
	if p["tab_active_bg"] != "" {
		t.tab = lipgloss.NewStyle().Background(lipgloss.Color(p["tab_active_bg"])).Bold(true)
		if p["tab_active_fg"] != "" {
			t.tab = t.tab.Foreground(lipgloss.Color(p["tab_active_fg"]))
		}
	}
	return t
}

// level picks the bar colour for a percentage.
func (t Theme) level(p float64) lipgloss.Style {
	role := "bar_low"
	switch {
	case p >= 90:
		role = "bar_high"
	case p >= 60:
		role = "bar_mid"
	}
	if t.Colors[role] == "" {
		return lipgloss.NewStyle()
	}
	return lipgloss.NewStyle().Foreground(lipgloss.Color(t.Colors[role]))
}

func (t Theme) role(r string) lipgloss.Style {
	if t.Colors[r] == "" {
		return lipgloss.NewStyle()
	}
	return lipgloss.NewStyle().Foreground(lipgloss.Color(t.Colors[r]))
}
