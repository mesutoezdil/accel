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
	"nord": {
		"accent": "#88c0d0", "text": "#d8dee9", "dim": "#4c566a", "title": "#88c0d0", "border": "#4c566a",
		"ok": "#a3be8c", "mid": "#ebcb8b", "warn": "#bf616a", "crit": "#b48ead", "info": "#8fbcbb",
		"bar_low": "#a3be8c", "bar_mid": "#ebcb8b", "bar_high": "#bf616a", "bar_track": "#3b4252", "spark": "#88c0d0",
		"tab_active_bg": "#88c0d0", "tab_active_fg": "#2e3440", "selection_bg": "#434c5e", "selection_fg": "#eceff4",
	},
	"gruvbox": {
		"accent": "#83a598", "text": "#ebdbb2", "dim": "#928374", "title": "#fabd2f", "border": "#504945",
		"ok": "#b8bb26", "mid": "#fabd2f", "warn": "#fb4934", "crit": "#d3869b", "info": "#8ec07c",
		"bar_low": "#b8bb26", "bar_mid": "#fabd2f", "bar_high": "#fb4934", "bar_track": "#3c3836", "spark": "#8ec07c",
		"tab_active_bg": "#fabd2f", "tab_active_fg": "#282828", "selection_bg": "#504945", "selection_fg": "#ebdbb2",
	},
	"catppuccin": {
		"accent": "#cba6f7", "text": "#cdd6f4", "dim": "#6c7086", "title": "#cba6f7", "border": "#45475a",
		"ok": "#a6e3a1", "mid": "#f9e2af", "warn": "#f38ba8", "crit": "#f5c2e7", "info": "#89dceb",
		"bar_low": "#a6e3a1", "bar_mid": "#f9e2af", "bar_high": "#f38ba8", "bar_track": "#313244", "spark": "#89b4fa",
		"tab_active_bg": "#cba6f7", "tab_active_fg": "#1e1e2e", "selection_bg": "#45475a", "selection_fg": "#cdd6f4",
	},
	"tokyo-night": {
		"accent": "#7aa2f7", "text": "#c0caf5", "dim": "#565f89", "title": "#7aa2f7", "border": "#3b4261",
		"ok": "#9ece6a", "mid": "#e0af68", "warn": "#f7768e", "crit": "#bb9af7", "info": "#7dcfff",
		"bar_low": "#9ece6a", "bar_mid": "#e0af68", "bar_high": "#f7768e", "bar_track": "#292e42", "spark": "#7dcfff",
		"tab_active_bg": "#7aa2f7", "tab_active_fg": "#1a1b26", "selection_bg": "#283457", "selection_fg": "#c0caf5",
	},
	"one-dark": {
		"accent": "#61afef", "text": "#abb2bf", "dim": "#5c6370", "title": "#61afef", "border": "#4b5263",
		"ok": "#98c379", "mid": "#e5c07b", "warn": "#e06c75", "crit": "#c678dd", "info": "#56b6c2",
		"bar_low": "#98c379", "bar_mid": "#e5c07b", "bar_high": "#e06c75", "bar_track": "#3e4451", "spark": "#56b6c2",
		"tab_active_bg": "#61afef", "tab_active_fg": "#282c34", "selection_bg": "#3e4451", "selection_fg": "#abb2bf",
	},
	"rose-pine": {
		"accent": "#c4a7e7", "text": "#e0def4", "dim": "#6e6a86", "title": "#c4a7e7", "border": "#403d52",
		"ok": "#9ccfd8", "mid": "#f6c177", "warn": "#eb6f92", "crit": "#ebbcba", "info": "#31748f",
		"bar_low": "#9ccfd8", "bar_mid": "#f6c177", "bar_high": "#eb6f92", "bar_track": "#26233a", "spark": "#c4a7e7",
		"tab_active_bg": "#c4a7e7", "tab_active_fg": "#191724", "selection_bg": "#403d52", "selection_fg": "#e0def4",
	},
	"monokai": {
		"accent": "#66d9ef", "text": "#f8f8f2", "dim": "#75715e", "title": "#66d9ef", "border": "#49483e",
		"ok": "#a6e22e", "mid": "#e6db74", "warn": "#f92672", "crit": "#ae81ff", "info": "#66d9ef",
		"bar_low": "#a6e22e", "bar_mid": "#e6db74", "bar_high": "#f92672", "bar_track": "#49483e", "spark": "#66d9ef",
		"tab_active_bg": "#66d9ef", "tab_active_fg": "#272822", "selection_bg": "#49483e", "selection_fg": "#f8f8f2",
	},
	"flexoki": {
		"accent": "#4385be", "text": "#cecdc3", "dim": "#878580", "title": "#4385be", "border": "#403e3c",
		"ok": "#879a39", "mid": "#d0a215", "warn": "#d14d41", "crit": "#ce5d97", "info": "#3aa99f",
		"bar_low": "#879a39", "bar_mid": "#d0a215", "bar_high": "#d14d41", "bar_track": "#282726", "spark": "#3aa99f",
		"tab_active_bg": "#4385be", "tab_active_fg": "#100f0f", "selection_bg": "#343331", "selection_fg": "#cecdc3",
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
