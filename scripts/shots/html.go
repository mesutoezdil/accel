package main

import (
	"math"
	"strconv"
	"strings"
)

// htmlLines turns a rendered view (text with SGR escapes) into one HTML
// string per line, so a page can show the real output as selectable text
// instead of a picture of it. Only the escapes lipgloss emits are handled,
// the same set the SVG renderer reads.
func htmlLines(view, fg, bg string) []string {
	src := strings.Split(strings.TrimRight(view, "\n"), "\n")
	out := make([]string, len(src))
	for i, line := range src {
		var b strings.Builder
		st := style{fg: fg}
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
			if css := st.css(fg, bg); css != "" {
				b.WriteString(`<span style="` + css + `">` + escapeHTML(run) + `</span>`)
				continue
			}
			b.WriteString(escapeHTML(run))
		}
		out[i] = b.String()
	}
	return out
}

// defaultFg is the colour a cell takes when nothing sets one on a dark
// terminal, matching the SVG renderer.
const defaultFg = "#c9d1d9"

// css renders a style as inline CSS, leaving out what the page already has.
// defFg and defBg are the terminal's own two colours, so a light capture
// reverses to its own paper rather than to a dark page's.
func (s style) css(defFg, defBg string) string {
	fg, bg := s.fg, s.bg
	if s.reverse {
		fg, bg = defBg, s.fg
		if bg == "" {
			bg = defFg // a cell with no colour of its own reverses to the terminal's
		}
		if s.bg != "" {
			fg = s.bg
		}
	}
	var a []string
	if fg != "" && fg != defFg {
		a = append(a, "color:"+fg)
	}
	if bg != "" {
		a = append(a, "background:"+bg)
	}
	if s.bold {
		a = append(a, "font-weight:700")
	}
	if s.faint {
		a = append(a, "opacity:"+faintOn(defBg))
	}
	if s.italic {
		a = append(a, "font-style:italic")
	}
	if s.underline {
		a = append(a, "text-decoration:underline")
	}
	return strings.Join(a, ";")
}

// faintOn is how far a faint cell fades, which cannot be one number. On a
// dark terminal the dim colour is far from the background and .6 still reads;
// on paper it is already close to it, and the same fade puts the text under
// three to one against the page. Light backgrounds fade less.
func faintOn(bg string) string {
	if luminance(bg) > 0.5 {
		return ".86"
	}
	return ".7"
}

// luminance is the relative luminance of #rrggbb, 0 for black and 1 for white.
func luminance(hex string) float64 {
	if len(hex) != 7 || hex[0] != '#' {
		return 0
	}
	var l float64
	for i, w := range [3]float64{0.2126, 0.7152, 0.0722} {
		n, err := strconv.ParseUint(hex[1+i*2:3+i*2], 16, 8)
		if err != nil {
			return 0
		}
		c := float64(n) / 255
		if c <= 0.04045 {
			c /= 12.92
		} else {
			c = math.Pow((c+0.055)/1.055, 2.4)
		}
		l += w * c
	}
	return l
}

var htmlEscapes = strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;")

func escapeHTML(s string) string { return htmlEscapes.Replace(s) }
