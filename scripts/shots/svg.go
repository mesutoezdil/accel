package main

import (
	"fmt"
	"strconv"
	"strings"
)

// svg turns a rendered view (text with SGR escapes) into an SVG image with a
// window frame. Only the escapes lipgloss emits are handled: reset, bold,
// faint, italic, underline, reverse, and 16-, 256-, and 24-bit colors.
//
// defaultBG is the window a capture is drawn on when the theme names none.
const defaultBG = "#0d1117"

// ponytail: every cell is one column wide; the views use no wide runes.
func svg(view string, cols, rows int, title string) string {
	return svgOn(view, cols, rows, title, defaultBG, defaultFg)
}

// svgOn draws the view on a given background, so a light theme is captured on
// paper rather than on the dark window every other capture uses.
func svgOn(view string, cols, rows int, title, bg, fg string) string {
	const (
		cw, lh, fs = 8.4, 18.0, 14.0 // cell width, line height, and font size in px
		pad        = 20.0
		barH       = 36.0
	)
	lines := strings.Split(strings.TrimRight(view, "\n "), "\n")
	rows = min(rows, len(lines))
	const margin = 36.0 // desk around the window
	winW := float64(cols)*cw + 2*pad
	winH := float64(rows)*lh + 2*pad + barH
	width, height := winW+2*margin, winH+2*margin
	var b strings.Builder
	fmt.Fprintf(&b, `<svg xmlns="http://www.w3.org/2000/svg" width="%.0f" height="%.0f" viewBox="0 0 %.0f %.0f" font-family="JetBrains Mono, SFMono-Regular, Menlo, Consolas, Liberation Mono, monospace" font-size="%.0f">`+"\n", width, height, width, height, fs)
	b.WriteString(`<defs><linearGradient id="desk" x1="0" y1="0" x2="1" y2="1"><stop offset="0" stop-color="#1f2a3a"/><stop offset="1" stop-color="#0b0e14"/></linearGradient>` +
		`<filter id="shadow" x="-10%" y="-10%" width="120%" height="130%"><feDropShadow dx="0" dy="14" stdDeviation="16" flood-color="#000" flood-opacity="0.55"/></filter></defs>` + "\n")
	b.WriteString(`<rect width="100%" height="100%" fill="url(#desk)"/>` + "\n")
	fmt.Fprintf(&b, `<rect x="%.0f" y="%.0f" width="%.0f" height="%.0f" rx="12" fill="%s" filter="url(#shadow)"/>`+"\n", margin, margin, winW, winH, bg)
	fmt.Fprintf(&b, `<rect x="%.0f" y="%.0f" width="%.0f" height="%.0f" rx="12" fill="#161b22"/><rect x="%.0f" y="%.0f" width="%.0f" height="%.0f" fill="#161b22"/>`+"\n", margin, margin, winW, barH, margin, margin+barH-12, winW, 12.0)
	for i, c := range []string{"#ff5f57", "#febc2e", "#28c840"} {
		fmt.Fprintf(&b, `<circle cx="%.0f" cy="%.0f" r="6" fill="%s"/>`+"\n", margin+pad+float64(i)*20, margin+barH/2, c)
	}
	fmt.Fprintf(&b, `<text x="50%%" y="%.0f" text-anchor="middle" fill="#8b949e" font-size="12">%s</text>`+"\n", margin+barH/2+4, esc(title))
	fmt.Fprintf(&b, `<g xml:space="preserve" transform="translate(%.0f %.0f)">`+"\n", margin, margin)
	st := style{fg: fg}
	for row, line := range lines[:rows] {
		y := barH + pad + float64(row)*lh
		col := 0
		var text strings.Builder
		for i := 0; i < len(line); {
			if line[i] == 0x1b && i+1 < len(line) && line[i+1] == '[' {
				end := strings.IndexByte(line[i:], 'm')
				if end < 0 {
					break
				}
				st = st.apply(line[i+2 : i+end])
				i += end + 1
				continue
			}
			// one run of plain text up to the next escape
			next := strings.IndexByte(line[i:], 0x1b)
			if next < 0 {
				next = len(line) - i
			}
			run := line[i : i+next]
			i += next
			n := len([]rune(run))
			if st.bg != "" || st.reverse {
				fill := st.bg
				if st.reverse {
					fill = st.fg
					if fill == "" {
						fill = fg
					}
				}
				fmt.Fprintf(&b, `<rect x="%.1f" y="%.1f" width="%.1f" height="%.1f" fill="%s"/>`+"\n", pad+float64(col)*cw, y, float64(n)*cw, lh, fill)
			}
			if strings.TrimSpace(run) != "" {
				fmt.Fprintf(&text, `<tspan x="%.1f"%s>%s</tspan>`, pad+float64(col)*cw, st.attrs(fg, bg), esc(run))
			}
			col += n
		}
		if text.Len() > 0 {
			fmt.Fprintf(&b, `<text y="%.1f" fill="%s" xml:space="preserve" style="white-space:pre">%s</text>`+"\n", y+fs, fg, text.String())
		}
	}
	b.WriteString("</g>\n</svg>\n")
	return b.String()
}

type style struct {
	fg, bg                         string
	bold, faint, italic, underline bool
	reverse                        bool
}

func (s style) attrs(defFg, defBg string) string {
	var a []string
	fg, bg := s.fg, s.bg
	if s.reverse {
		fg = defBg
		if bg != "" {
			fg = bg
		}
	}
	if fg != "" && fg != defFg {
		a = append(a, ` fill="`+fg+`"`)
	}
	if s.bold {
		a = append(a, ` font-weight="bold"`)
	}
	if s.faint {
		a = append(a, ` opacity="`+faintOn(defBg)+`"`)
	}
	if s.italic {
		a = append(a, ` font-style="italic"`)
	}
	if s.underline {
		a = append(a, ` text-decoration="underline"`)
	}
	return strings.Join(a, "")
}

// apply reads one SGR parameter list such as "1;38;2;255;0;0".
func (s style) apply(params string) style {
	if params == "" {
		return style{fg: s.fg}
	}
	p := strings.Split(params, ";")
	for i := 0; i < len(p); i++ {
		n, _ := strconv.Atoi(p[i])
		switch {
		case n == 0:
			s = style{}
		case n == 1:
			s.bold = true
		case n == 2:
			s.faint = true
		case n == 3:
			s.italic = true
		case n == 4:
			s.underline = true
		case n == 7:
			s.reverse = true
		case n == 22:
			s.bold, s.faint = false, false
		case n == 23:
			s.italic = false
		case n == 24:
			s.underline = false
		case n == 27:
			s.reverse = false
		case n == 39:
			s.fg = ""
		case n == 49:
			s.bg = ""
		case n >= 30 && n <= 37:
			s.fg = ansi16[n-30]
		case n >= 90 && n <= 97:
			s.fg = ansi16[n-90+8]
		case n >= 40 && n <= 47:
			s.bg = ansi16[n-40]
		case n >= 100 && n <= 107:
			s.bg = ansi16[n-100+8]
		case (n == 38 || n == 48) && i+1 < len(p):
			var c string
			switch p[i+1] {
			case "5":
				if i+2 < len(p) {
					idx, _ := strconv.Atoi(p[i+2])
					c = ansi256(idx)
					i += 2
				}
			case "2":
				if i+4 < len(p) {
					r, _ := strconv.Atoi(p[i+2])
					g, _ := strconv.Atoi(p[i+3])
					bl, _ := strconv.Atoi(p[i+4])
					c = fmt.Sprintf("#%02x%02x%02x", r, g, bl)
					i += 4
				}
			}
			if n == 38 {
				s.fg = c
			} else {
				s.bg = c
			}
		}
	}
	return s
}

var ansi16 = [16]string{
	"#484f58", "#ff7b72", "#3fb950", "#d29922", "#58a6ff", "#bc8cff", "#39c5cf", "#b1bac4",
	"#6e7681", "#ffa198", "#56d364", "#e3b341", "#79c0ff", "#d2a8ff", "#56d4dd", "#f0f6fc",
}

// ansi256 maps the xterm 256-color index to a hex color.
func ansi256(i int) string {
	switch {
	case i < 16:
		return ansi16[i]
	case i < 232:
		i -= 16
		steps := []int{0, 95, 135, 175, 215, 255}
		return fmt.Sprintf("#%02x%02x%02x", steps[i/36], steps[i/6%6], steps[i%6])
	default:
		v := 8 + (i-232)*10
		return fmt.Sprintf("#%02x%02x%02x", v, v, v)
	}
}

func esc(s string) string {
	return strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;").Replace(s)
}
