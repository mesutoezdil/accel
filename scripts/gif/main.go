// Command gif assembles a frame sequence into an animated GIF.
//
// A terminal capture is mostly unchanged pixels: each frame is written as
// the rectangle that differs from the one before it, with everything else
// transparent, and identical frames become a longer delay instead of more
// data. Colours come from one median-cut palette built over every frame, so
// the animation never shifts hue partway through.
//
//	go run ./scripts/gif -out assets/demo.gif -fps 10 build/film/frame-*.png
package main

import (
	"bytes"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/gif"
	"image/png"
	"os"
	"path/filepath"
	"sort"
)

func main() {
	out := flag.String("out", "demo.gif", "GIF to write")
	fps := flag.Int("fps", 10, "frames per second")
	last := flag.Int("last", 1200, "milliseconds to hold the final frame")
	colors := flag.Int("colors", 255, "palette size, 255 at most: one index is kept for transparency")
	flag.Parse()
	if err := run(flag.Args(), *out, *fps, *last, *colors); err != nil {
		fmt.Fprintln(os.Stderr, "gif:", err)
		os.Exit(1)
	}
}

func run(files []string, out string, fps, last, colors int) error {
	if len(files) == 0 {
		return fmt.Errorf("no frames given")
	}
	if fps <= 0 || fps > 50 {
		return fmt.Errorf("fps %d is outside 1..50", fps)
	}
	if colors < 2 || colors > 255 {
		return fmt.Errorf("colors %d is outside 2..255", colors)
	}
	sort.Strings(files)
	frames := make([]*image.RGBA, 0, len(files))
	for _, f := range files {
		img, err := readPNG(f)
		if err != nil {
			return err
		}
		if len(frames) > 0 && !img.Bounds().Eq(frames[0].Bounds()) {
			return fmt.Errorf("%s is %v, the first frame is %v", filepath.Base(f), img.Bounds().Size(), frames[0].Bounds().Size())
		}
		frames = append(frames, img)
	}
	g := encode(frames, fps, last, colors)
	f, err := os.Create(out)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	if err := gif.EncodeAll(f, g); err != nil {
		return err
	}
	fi, err := f.Stat()
	if err != nil {
		return err
	}
	fmt.Printf("%s: %d frames of %d, %v, %.1f MB\n", out, len(g.Image), len(frames), frames[0].Bounds().Size(), float64(fi.Size())/(1<<20))
	return nil
}

func readPNG(path string) (*image.RGBA, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	src, err := png.Decode(f)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	if rgba, ok := src.(*image.RGBA); ok {
		return rgba, nil
	}
	b := src.Bounds()
	dst := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			dst.Set(x-b.Min.X, y-b.Min.Y, src.At(x, y))
		}
	}
	return dst, nil
}

// transparent is the palette index left free for pixels a frame does not
// touch, so an unchanged area keeps whatever the frame before it drew.
const transparent = 0

// encode turns the frames into one GIF, writing only what each frame changes.
func encode(frames []*image.RGBA, fps, last, colors int) *gif.GIF {
	pal := make(color.Palette, 0, colors+1)
	pal = append(pal, color.RGBA{}) // index 0: transparent
	pal = append(pal, medianCut(frames, colors)...)
	delay := max(100/fps, 2) // GIF delays are hundredths of a second
	g := &gif.GIF{LoopCount: 0, Config: image.Config{ColorModel: pal, Width: frames[0].Bounds().Dx(), Height: frames[0].Bounds().Dy()}}
	near := newMatcher(pal)
	var prev *image.RGBA
	for _, fr := range frames {
		box, same := changed(prev, fr)
		if same {
			g.Delay[len(g.Delay)-1] += delay // hold instead of repeating a frame
			continue
		}
		p := image.NewPaletted(box, pal)
		for y := box.Min.Y; y < box.Max.Y; y++ {
			for x := box.Min.X; x < box.Max.X; x++ {
				c := fr.RGBAAt(x, y)
				if prev != nil && prev.RGBAAt(x, y) == c {
					p.SetColorIndex(x, y, transparent)
					continue
				}
				p.SetColorIndex(x, y, near.index(c))
			}
		}
		g.Image = append(g.Image, p)
		g.Delay = append(g.Delay, delay)
		g.Disposal = append(g.Disposal, gif.DisposalNone)
		prev = fr
	}
	if n := len(g.Delay); n > 0 && last > 0 {
		g.Delay[n-1] += last / 10
	}
	return g
}

// changed is the rectangle b differs from a in, and whether they are equal.
func changed(a, b *image.RGBA) (image.Rectangle, bool) {
	if a == nil {
		return b.Bounds(), false
	}
	r := b.Bounds()
	box := image.Rectangle{Min: r.Max, Max: r.Min}
	for y := r.Min.Y; y < r.Max.Y; y++ {
		row := y * b.Stride
		if bytes.Equal(a.Pix[row:row+b.Stride], b.Pix[row:row+b.Stride]) {
			continue // a row of a terminal changes or it does not
		}
		for x := r.Min.X; x < r.Max.X; x++ {
			i := row + x*4
			if a.Pix[i] != b.Pix[i] || a.Pix[i+1] != b.Pix[i+1] || a.Pix[i+2] != b.Pix[i+2] || a.Pix[i+3] != b.Pix[i+3] {
				box.Min.X, box.Min.Y = min(box.Min.X, x), min(box.Min.Y, y)
				box.Max.X, box.Max.Y = max(box.Max.X, x+1), max(box.Max.Y, y+1)
			}
		}
	}
	if box.Empty() {
		return b.Bounds(), true
	}
	return box, false
}

// matcher maps a colour to the nearest palette entry. A capture asks about
// millions of pixels drawn from a few thousand colours, so answers are kept
// in a table over the whole 24-bit space: 16 MB, and no hashing per pixel.
// Zero means "not worked out yet"; index 0 is transparency and never a match.
type matcher struct {
	pal   color.Palette
	cache []uint8
}

func newMatcher(pal color.Palette) *matcher {
	return &matcher{pal: pal, cache: make([]uint8, 1<<24)}
}

func (m *matcher) index(c color.RGBA) uint8 {
	key := uint32(c.R)<<16 | uint32(c.G)<<8 | uint32(c.B)
	if i := m.cache[key]; i != 0 {
		return i
	}
	best, bestD := uint8(1), 1<<62
	for i := 1; i < len(m.pal); i++ {
		p := m.pal[i].(color.RGBA)
		dr, dg, db := int(c.R)-int(p.R), int(c.G)-int(p.G), int(c.B)-int(p.B)
		if d := dr*dr + dg*dg + db*db; d < bestD {
			best, bestD = uint8(i), d
		}
	}
	m.cache[key] = best
	return best
}

// bucket is a set of colours median cut has not split yet.
type bucket struct {
	colors []weighted
	count  int
}

type weighted struct {
	c color.RGBA
	n int
}

// medianCut builds a palette of at most n colours over every frame. Terminal
// output is flat colour plus antialiased text, so the split is on the widest
// channel of each bucket, weighted by how often a colour actually appears.
func medianCut(frames []*image.RGBA, n int) color.Palette {
	counts := make([]uint32, 1<<24) // every colour, counted by pixel
	for _, fr := range frames {
		for i := 0; i+3 < len(fr.Pix); i += 4 {
			counts[uint32(fr.Pix[i])<<16|uint32(fr.Pix[i+1])<<8|uint32(fr.Pix[i+2])]++
		}
	}
	var all []weighted // already in colour order, which keeps the palette stable
	total := 0
	for key, k := range counts {
		if k == 0 {
			continue
		}
		all = append(all, weighted{color.RGBA{uint8(key >> 16), uint8(key >> 8), uint8(key), 255}, int(k)})
		total += int(k)
	}
	if len(all) <= n {
		pal := make(color.Palette, 0, len(all))
		for _, w := range all {
			pal = append(pal, w.c)
		}
		return pal
	}

	buckets := []bucket{{colors: all, count: total}}
	for len(buckets) < n {
		// split the bucket that covers the most pixels and can still be cut
		pick := -1
		for i, b := range buckets {
			if len(b.colors) < 2 {
				continue
			}
			if pick < 0 || b.count > buckets[pick].count {
				pick = i
			}
		}
		if pick < 0 {
			break
		}
		a, b := split(buckets[pick])
		buckets[pick] = a
		buckets = append(buckets, b)
	}
	pal := make(color.Palette, 0, len(buckets))
	for _, b := range buckets {
		pal = append(pal, average(b))
	}
	return pal
}

// split cuts a bucket in half at the median of its widest channel.
func split(b bucket) (bucket, bucket) {
	lo, hi := color.RGBA{R: 255, G: 255, B: 255, A: 255}, color.RGBA{}
	for _, w := range b.colors {
		lo.R, lo.G, lo.B = min(lo.R, w.c.R), min(lo.G, w.c.G), min(lo.B, w.c.B)
		hi.R, hi.G, hi.B = max(hi.R, w.c.R), max(hi.G, w.c.G), max(hi.B, w.c.B)
	}
	ch := 0 // red, then green, then blue
	if int(hi.G)-int(lo.G) > int(hi.R)-int(lo.R) {
		ch = 1
	}
	if wide := int(hi.B) - int(lo.B); (ch == 0 && wide > int(hi.R)-int(lo.R)) || (ch == 1 && wide > int(hi.G)-int(lo.G)) {
		ch = 2
	}
	value := func(c color.RGBA) uint8 {
		switch ch {
		case 0:
			return c.R
		case 1:
			return c.G
		}
		return c.B
	}
	cs := append([]weighted(nil), b.colors...)
	sort.SliceStable(cs, func(i, j int) bool { return value(cs[i].c) < value(cs[j].c) })
	half, run := b.count/2, 0
	cut := 1
	for i, w := range cs {
		run += w.n
		if run >= half {
			cut = min(max(i, 1), len(cs)-1)
			break
		}
	}
	return bucket{cs[:cut], weight(cs[:cut])}, bucket{cs[cut:], weight(cs[cut:])}
}

func weight(cs []weighted) int {
	n := 0
	for _, w := range cs {
		n += w.n
	}
	return n
}

// average is the colour a bucket collapses to, weighted by pixel count.
func average(b bucket) color.RGBA {
	var r, g, bl, n float64
	for _, w := range b.colors {
		r += float64(w.c.R) * float64(w.n)
		g += float64(w.c.G) * float64(w.n)
		bl += float64(w.c.B) * float64(w.n)
		n += float64(w.n)
	}
	if n == 0 {
		return color.RGBA{A: 255}
	}
	return color.RGBA{uint8(r/n + 0.5), uint8(g/n + 0.5), uint8(bl/n + 0.5), 255}
}
