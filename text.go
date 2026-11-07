// Text drawing with the game's TrueType fonts. (c) 2026 Ironite
package main

import (
	"image"
	"image/color"
	"image/draw"
	"math"
	"strings"
	"unicode/utf8"

	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/font/sfnt"
	"golang.org/x/image/math/fixed"
)

type FontID int

const (
	FBody    FontID = iota // Work Sans
	FBold                  // Work Sans Bold
	FItalic                // Work Sans Italic
	FDisplay               // Young Serif
	FSign                  // National Park Bold
	FSymbol                // fallback for symbols (Work Sans lacks a few)
)

var fonts = map[FontID]*sfnt.Font{}

type faceKey struct {
	f  FontID
	px int // size in 1/4 device pixels
}

var faces = map[faceKey]font.Face{}

func loadFonts() {
	for id, name := range map[FontID]string{FBody: "WorkSans-Regular.ttf", FBold: "WorkSans-Bold.ttf", FItalic: "WorkSans-Italic.ttf", FDisplay: "YoungSerif-Regular.ttf", FSign: "NationalPark-Bold.ttf"} {
		b, err := assetFS.ReadFile("assets/fonts/" + name)
		if err != nil {
			panic(err)
		}
		f, err := opentype.Parse(b)
		if err != nil {
			panic(err)
		}
		fonts[id] = f
	}
	fonts[FSymbol] = fonts[FBody]
}

func face(f FontID, devPx float64) font.Face {
	k := faceKey{f, int(math.Round(devPx * 4))}
	if fc, ok := faces[k]; ok {
		return fc
	}
	fc, err := opentype.NewFace(fonts[f], &opentype.FaceOptions{Size: float64(k.px) / 4, DPI: 72, Hinting: font.HintingNone})
	if err != nil {
		panic(err)
	}
	faces[k] = fc
	return fc
}

// Text style: font, logical size, letter spacing (logical px), colour.
type TS struct {
	F    FontID
	Size float64
	LS   float64
	C    color.NRGBA
}

type textKey struct {
	f  FontID
	px int
	ls int
	s  string
}

type textImg struct {
	mask *image.Alpha
	asc  int // baseline offset from top, device px
	adv  float64
}

var textCache = map[textKey]*textImg{}
var textScale float64

func (c *Canvas) textImage(s string, t TS) *textImg {
	if textScale != c.S || len(textCache) > 3000 {
		textCache = map[textKey]*textImg{}
		textScale = c.S
	}
	px := t.Size * c.S
	k := textKey{t.F, int(math.Round(px * 4)), int(math.Round(t.LS * c.S * 4)), s}
	if ti, ok := textCache[k]; ok {
		return ti
	}
	fc := face(t.F, px)
	m := fc.Metrics()
	asc, desc := m.Ascent.Ceil()+1, m.Descent.Ceil()+1
	adv := measureDev(fc, s, t.LS*c.S)
	w := int(math.Ceil(adv)) + 4
	mask := image.NewAlpha(image.Rect(0, 0, w, asc+desc))
	d := &font.Drawer{Dst: mask, Src: image.Opaque, Face: fc, Dot: fixed.P(1, asc)}
	if t.LS == 0 {
		d.DrawString(s)
	} else {
		for _, r := range s {
			d.DrawString(string(r))
			d.Dot.X += fixed.Int26_6(t.LS * c.S * 64)
		}
	}
	ti := &textImg{mask: mask, asc: asc, adv: adv}
	textCache[k] = ti
	return ti
}

func measureDev(fc font.Face, s string, ls float64) float64 {
	w := float64(font.MeasureString(fc, s)) / 64
	if ls != 0 {
		w += ls * float64(utf8.RuneCountInString(s))
	}
	return w
}

// Measure returns the logical width of s.
func (c *Canvas) Measure(s string, t TS) float64 {
	return measureDev(face(t.F, t.Size*c.S), s, t.LS*c.S) / c.S
}

// Text draws s with its baseline at y. align: 0 left, 1 centre, 2 right.
func (c *Canvas) Text(s string, x, y float64, t TS, align int) float64 {
	if s == "" {
		return 0
	}
	ti := c.textImage(s, t)
	w := ti.adv / c.S
	switch align {
	case 1:
		x -= w / 2
	case 2:
		x -= w
	}
	dx, dy := c.dev(x, y)
	x0, y0 := int(math.Round(dx))-1, int(math.Round(dy))-ti.asc
	r := image.Rect(x0, y0, x0+ti.mask.Rect.Dx(), y0+ti.mask.Rect.Dy())
	col := WithAlpha(t.C, c.Alpha)
	draw.DrawMask(c.Dst, r, image.NewUniform(col), image.Point{}, ti.mask, image.Point{}, draw.Over)
	return w
}

// TextShadow draws text with a soft dark shadow underneath (for light text on images).
func (c *Canvas) TextShadow(s string, x, y float64, t TS, align int, sh color.NRGBA, off float64) {
	t2 := t
	t2.C = sh
	c.Text(s, x+off*.3, y+off, t2, align)
	c.Text(s, x, y, t, align)
}

// Wrap splits text into lines no wider than maxW.
func (c *Canvas) Wrap(s string, t TS, maxW float64) []string {
	var lines []string
	for _, para := range strings.Split(s, "\n") {
		words := strings.Fields(para)
		line := ""
		for _, w := range words {
			try := w
			if line != "" {
				try = line + " " + w
			}
			if line != "" && c.Measure(try, t) > maxW {
				lines = append(lines, line)
				line = w
			} else {
				line = try
			}
		}
		lines = append(lines, line)
	}
	return lines
}

// Rich text: runs of plain and bold/italic text, wrapped together.
type Run struct {
	S string
	F FontID
}

// RichParagraph lays out runs within maxW starting at (x, y) baseline; returns the y after the last line.
func (c *Canvas) RichParagraph(runs []Run, x, y, maxW, size, lineH float64, col color.NRGBA, draw bool) float64 {
	type word struct {
		s     string
		f     FontID
		space bool // a space precedes this word
	}
	var words []word
	for _, r := range runs {
		parts := strings.Split(r.S, " ")
		for i, p := range parts {
			if p == "" {
				continue
			}
			sp := i > 0 || strings.HasPrefix(r.S, " ")
			words = append(words, word{p, r.F, sp})
		}
	}
	cx := 0.0
	line := 0
	for i, w := range words {
		t := TS{F: w.f, Size: size, C: col}
		ww := c.Measure(w.s, t)
		spW := 0.0
		if w.space && cx > 0 {
			spW = c.Measure(" ", TS{F: FBody, Size: size})
		}
		if cx > 0 && cx+spW+ww > maxW {
			line++
			cx, spW = 0, 0
		}
		_ = i
		if draw {
			c.Text(w.s, x+cx+spW, y+float64(line)*lineH, t, 0)
		}
		cx += spW + ww
	}
	return y + float64(line+1)*lineH
}
