package main

import (
	"bytes"
	"image"
	"image/color"
	"image/gif"
	"testing"
)

// frame paints a background with one square of fg at x, so consecutive
// frames differ in a small, known rectangle.
func frame(bg, fg color.RGBA, x int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, 40, 20))
	for y := 0; y < 20; y++ {
		for i := 0; i < 40; i++ {
			img.SetRGBA(i, y, bg)
		}
	}
	for y := 4; y < 8; y++ {
		for i := x; i < x+4; i++ {
			img.SetRGBA(i, y, fg)
		}
	}
	return img
}

func TestEncodeHoldsIdenticalFrames(t *testing.T) {
	bg, fg := color.RGBA{13, 17, 23, 255}, color.RGBA{201, 209, 218, 255}
	frames := []*image.RGBA{frame(bg, fg, 0), frame(bg, fg, 0), frame(bg, fg, 8), frame(bg, fg, 16)}

	g := encode(frames, 10, 500, 255)
	if len(g.Image) != 3 {
		t.Fatalf("wrote %d frames for 4 captures with one repeat, want 3", len(g.Image))
	}
	if g.Delay[0] != 20 {
		t.Errorf("the repeated frame should hold: delay %d, want 20", g.Delay[0])
	}
	if want := 10 + 50; g.Delay[len(g.Delay)-1] != want {
		t.Errorf("last delay %d, want %d with the final hold", g.Delay[len(g.Delay)-1], want)
	}
	if g.LoopCount != 0 {
		t.Errorf("loop count %d, want 0 (forever)", g.LoopCount)
	}

	// only the moved square is written after the first frame
	if got := g.Image[0].Bounds(); !got.Eq(frames[0].Bounds()) {
		t.Errorf("first frame %v, want the whole image", got)
	}
	for i, p := range g.Image[1:] {
		if w, h := p.Bounds().Dx(), p.Bounds().Dy(); w > 16 || h > 8 {
			t.Errorf("frame %d covers %dx%d, want only the square that moved", i+1, w, h)
		}
	}
}

func TestEncodeKeepsColorsAndDecodes(t *testing.T) {
	bg, fg := color.RGBA{13, 17, 23, 255}, color.RGBA{255, 95, 87, 255}
	frames := []*image.RGBA{frame(bg, fg, 0), frame(bg, fg, 10)}

	var buf bytes.Buffer
	if err := gif.EncodeAll(&buf, encode(frames, 10, 0, 255)); err != nil {
		t.Fatal(err)
	}
	out, err := gif.DecodeAll(&buf)
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Image) != 2 {
		t.Fatalf("decoded %d frames, want 2", len(out.Image))
	}

	// composite the frames the way a viewer does, then compare to the source
	canvas := image.NewRGBA(image.Rect(0, 0, 40, 20))
	for i, p := range out.Image {
		b := p.Bounds()
		for y := b.Min.Y; y < b.Max.Y; y++ {
			for x := b.Min.X; x < b.Max.X; x++ {
				if _, _, _, a := p.At(x, y).RGBA(); a == 0 {
					continue // untouched: the frame before it still shows
				}
				canvas.Set(x, y, p.At(x, y))
			}
		}
		for y := 0; y < 20; y++ {
			for x := 0; x < 40; x++ {
				if canvas.RGBAAt(x, y) != frames[i].RGBAAt(x, y) {
					t.Fatalf("frame %d differs at %d,%d: %v, want %v", i, x, y, canvas.RGBAAt(x, y), frames[i].RGBAAt(x, y))
				}
			}
		}
	}
}

func TestMedianCutFitsThePaletteAndKeepsFlatColors(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 64, 64))
	for y := 0; y < 64; y++ {
		for x := 0; x < 64; x++ {
			img.SetRGBA(x, y, color.RGBA{uint8(x * 4), uint8(y * 4), uint8((x + y) * 2), 255})
		}
	}
	pal := medianCut([]*image.RGBA{img}, 64)
	if len(pal) > 64 {
		t.Fatalf("palette has %d colours, want 64 at most", len(pal))
	}
	near := newMatcher(append(color.Palette{color.RGBA{}}, pal...))
	worst := 0
	for y := 0; y < 64; y += 3 {
		for x := 0; x < 64; x += 3 {
			c := img.RGBAAt(x, y)
			p := pal[near.index(c)-1].(color.RGBA)
			for _, d := range []int{int(c.R) - int(p.R), int(c.G) - int(p.G), int(c.B) - int(p.B)} {
				worst = max(worst, max(d, -d))
			}
		}
	}
	if worst > 24 {
		t.Errorf("worst channel error %d over a full gradient, want 24 at most", worst)
	}

	// a capture with few colours keeps them exactly
	flat := image.NewRGBA(image.Rect(0, 0, 8, 8))
	want := []color.RGBA{{13, 17, 23, 255}, {201, 209, 218, 255}, {255, 95, 87, 255}, {40, 200, 64, 255}}
	for i := 0; i < 64; i++ {
		flat.SetRGBA(i%8, i/8, want[i%len(want)])
	}
	got := medianCut([]*image.RGBA{flat}, 255)
	if len(got) != len(want) {
		t.Fatalf("a %d colour image gave a palette of %d", len(want), len(got))
	}
	for _, w := range want {
		if !hasColor(got, w) {
			t.Errorf("palette lost %v", w)
		}
	}
}

func hasColor(pal color.Palette, c color.RGBA) bool {
	for _, p := range pal {
		if p.(color.RGBA) == c {
			return true
		}
	}
	return false
}
