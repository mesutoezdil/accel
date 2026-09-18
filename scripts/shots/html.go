package main

import (
	"strings"
)

// htmlLines turns a rendered view (text with SGR escapes) into one HTML
// string per line, so a page can show the real output as selectable text
// instead of a picture of it. Only the escapes lipgloss emits are handled,
// the same set the SVG renderer reads.
func htmlLines(view string) []string {
	src := strings.Split(strings.TrimRight(view, "\n"), "\n")
	out := make([]string, len(src))
	for i, line := range src {
		var b strings.Builder
		st := style{fg: defaultFg}
		for j := 0; j < len(line); {
			if line[j] == 0x1b && j+1 < len(line) && line[j+1] == '[' {
				end := strings.IndexByte(line[j:], 'm')
				if end < 0 {
					break
				}
				st = st.apply(line[j+2 : j+end])
				j += end + 1
				continue
			}
			next := strings.IndexByte(line[j:], 0x1b)
			if next < 0 {
				next = len(line) - j
			}
			run := line[j : j+next]
			j += next
			if run == "" {
				continue
			}
			if css := st.css(); css != "" {
				b.WriteString(`<span style="` + css + `">` + escapeHTML(run) + `</span>`)
				continue
			}
			b.WriteString(escapeHTML(run))
		}
		out[i] = b.String()
	}
	return out
}

// defaultFg is the colour a cell takes when nothing sets one, matching the
// SVG renderer and the page's own terminal background.
const defaultFg = "#c9d1d9"

// css renders a style as inline CSS, leaving out what the page already has.
func (s style) css() string {
	fg, bg := s.fg, s.bg
	if s.reverse {
		fg, bg = "#0d1117", s.fg
		if s.bg != "" {
			fg = s.bg
		}
	}
	var a []string
	if fg != "" && fg != defaultFg {
		a = append(a, "color:"+fg)
	}
	if bg != "" {
		a = append(a, "background:"+bg)
	}
	if s.bold {
		a = append(a, "font-weight:700")
	}
	if s.faint {
		a = append(a, "opacity:.6")
	}
	if s.italic {
		a = append(a, "font-style:italic")
	}
	if s.underline {
		a = append(a, "text-decoration:underline")
	}
	return strings.Join(a, ";")
}

var htmlEscapes = strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;")

func escapeHTML(s string) string { return htmlEscapes.Replace(s) }
