// A small 2D drawing library: paths, gradients, images and text over
// golang.org/x/image, drawn in the game's 1600x900 logical coordinates and
// scaled to the window's real pixels so everything stays sharp at any size.
// (c) 2026 Ironite
package main

import (
	"image"
	"image/color"
	"image/draw"
	"math"
	"strconv"
	"strings"

	xdraw "golang.org/x/image/draw"
	"golang.org/x/image/vector"
)

type pt struct{ X, Y float64 }

// Path collects contours in logical coordinates. Curves are flattened when added.
type Path struct {
	subs [][]pt
	cur  []pt
}

func (p *Path) MoveTo(x, y float64) {
	p.flush()
	p.cur = []pt{{x, y}}
}
func (p *Path) LineTo(x, y float64) {
	if len(p.cur) == 0 {
		p.cur = []pt{{x, y}}
		return
	}
	p.cur = append(p.cur, pt{x, y})
}
func (p *Path) last() pt { return p.cur[len(p.cur)-1] }
func (p *Path) QuadTo(cx, cy, x, y float64) {
	a := p.last()
	n := 12
	for i := 1; i <= n; i++ {
		t := float64(i) / float64(n)
		u := 1 - t
		p.cur = append(p.cur, pt{u*u*a.X + 2*u*t*cx + t*t*x, u*u*a.Y + 2*u*t*cy + t*t*y})
	}
}
func (p *Path) CubicTo(c1x, c1y, c2x, c2y, x, y float64) {
	a := p.last()
	n := 16
	for i := 1; i <= n; i++ {
		t := float64(i) / float64(n)
		u := 1 - t
		p.cur = append(p.cur, pt{u*u*u*a.X + 3*u*u*t*c1x + 3*u*t*t*c2x + t*t*t*x, u*u*u*a.Y + 3*u*u*t*c1y + 3*u*t*t*c2y + t*t*t*y})
	}
}

// Arc adds an elliptical arc (angles in radians, clockwise on screen as in canvas).
func (p *Path) Arc(cx, cy, rx, ry, a0, a1 float64) {
	n := int(math.Max(8, math.Abs(a1-a0)/(2*math.Pi)*64))
	for i := 0; i <= n; i++ {
		a := a0 + (a1-a0)*float64(i)/float64(n)
		x, y := cx+math.Cos(a)*rx, cy+math.Sin(a)*ry
		if i == 0 && len(p.cur) == 0 {
			p.cur = []pt{{x, y}}
		} else {
			p.cur = append(p.cur, pt{x, y})
		}
	}
}
func (p *Path) Close() { p.flush() }
func (p *Path) flush() {
	if len(p.cur) > 1 {
		p.subs = append(p.subs, p.cur)
	}
	p.cur = nil
}
func (p *Path) contours() [][]pt { p.flush(); return p.subs }

func RectPath(x, y, w, h float64) *Path {
	p := &Path{}
	p.MoveTo(x, y)
	p.LineTo(x+w, y)
	p.LineTo(x+w, y+h)
	p.LineTo(x, y+h)
	p.Close()
	return p
}

// RoundRectPath: clockwise rounded rectangle.
func RoundRectPath(x, y, w, h, r float64) *Path {
	p := &Path{}
	addRoundRect(p, x, y, w, h, r, false)
	return p
}
func addRoundRect(p *Path, x, y, w, h, r float64, reverse bool) {
	r = math.Max(0, math.Min(r, math.Min(w, h)/2))
	var pts []pt
	corner := func(cx, cy, a0 float64) {
		for i := 0; i <= 8; i++ {
			a := a0 + math.Pi/2*float64(i)/8
			pts = append(pts, pt{cx + math.Cos(a)*r, cy + math.Sin(a)*r})
		}
	}
	corner(x+w-r, y+r, -math.Pi/2)
	corner(x+w-r, y+h-r, 0)
	corner(x+r, y+h-r, math.Pi/2)
	corner(x+r, y+r, math.Pi)
	if reverse {
		for i, j := 0, len(pts)-1; i < j; i, j = i+1, j-1 {
			pts[i], pts[j] = pts[j], pts[i]
		}
	}
	p.flush()
	p.subs = append(p.subs, pts)
}
func EllipsePath(cx, cy, rx, ry float64) *Path {
	p := &Path{}
	p.Arc(cx, cy, rx, ry, 0, 2*math.Pi)
	p.Close()
	return p
}

// ---------------------------------------------------------------- paints

// Paint is a solid colour or a gradient. Gradient coordinates are logical.
type Paint struct {
	C       color.NRGBA
	Kind    int // 0 solid, 1 linear, 2 radial
	X0, Y0  float64
	X1, Y1  float64 // linear end, or radial centre is X0,Y0 with radii R0..R1
	R0, R1  float64
	Stops   []color.NRGBA
	Offsets []float64 // optional; even spacing otherwise
}

func Solid(c color.NRGBA) Paint { return Paint{C: c} }
func Lin(x0, y0, x1, y1 float64, stops ...color.NRGBA) Paint {
	return Paint{Kind: 1, X0: x0, Y0: y0, X1: x1, Y1: y1, Stops: stops}
}
func Rad(cx, cy, r0, r1 float64, stops ...color.NRGBA) Paint {
	return Paint{Kind: 2, X0: cx, Y0: cy, R0: r0, R1: r1, Stops: stops}
}

// Hex parses "#rgb", "#rrggbb" or "#rrggbbaa".
func Hex(s string) color.NRGBA {
	s = strings.TrimPrefix(s, "#")
	if len(s) == 3 {
		s = string([]byte{s[0], s[0], s[1], s[1], s[2], s[2]})
	}
	v, _ := strconv.ParseUint(s, 16, 32)
	if len(s) == 8 {
		return color.NRGBA{uint8(v >> 24), uint8(v >> 16), uint8(v >> 8), uint8(v)}
	}
	return color.NRGBA{uint8(v >> 16), uint8(v >> 8), uint8(v), 255}
}
func RGBA(r, g, b int, a float64) color.NRGBA {
	return color.NRGBA{uint8(r), uint8(g), uint8(b), uint8(math.Round(clamp01(a) * 255))}
}
func WithAlpha(c color.NRGBA, a float64) color.NRGBA {
	c.A = uint8(math.Round(float64(c.A) * clamp01(a)))
	return c
}
func clamp01(v float64) float64 { return math.Max(0, math.Min(1, v)) }

// gradImage evaluates a gradient in device pixels.
type gradImage struct {
	p      Paint
	s, ox  float64
	oy     float64
	alpha  float64
	b      image.Rectangle
	lut    [256]color.RGBA
	dx, dy float64
	len2   float64
}

func newGrad(p Paint, s, ox, oy, alpha float64, b image.Rectangle) *gradImage {
	g := &gradImage{p: p, s: s, ox: ox, oy: oy, alpha: alpha, b: b}
	n := len(p.Stops)
	for i := 0; i < 256; i++ {
		t := float64(i) / 255
		var c color.NRGBA
		if n == 1 {
			c = p.Stops[0]
		} else {
			var k int
			var f float64
			if len(p.Offsets) == n {
				k = 0
				for k < n-2 && t > p.Offsets[k+1] {
					k++
				}
				span := p.Offsets[k+1] - p.Offsets[k]
				if span > 0 {
					f = clamp01((t - p.Offsets[k]) / span)
				}
			} else {
				pos := t * float64(n-1)
				k = int(math.Min(float64(n-2), math.Floor(pos)))
				f = pos - float64(k)
			}
			a, b := p.Stops[k], p.Stops[k+1]
			lerp := func(x, y uint8) float64 { return float64(x) + (float64(y)-float64(x))*f }
			c = color.NRGBA{uint8(lerp(a.R, b.R)), uint8(lerp(a.G, b.G)), uint8(lerp(a.B, b.B)), uint8(lerp(a.A, b.A))}
		}
		al := float64(c.A) / 255 * alpha
		g.lut[i] = color.RGBA{uint8(float64(c.R) * al), uint8(float64(c.G) * al), uint8(float64(c.B) * al), uint8(255 * al)}
	}
	if p.Kind == 1 {
		g.dx, g.dy = p.X1-p.X0, p.Y1-p.Y0
		g.len2 = g.dx*g.dx + g.dy*g.dy
		if g.len2 == 0 {
			g.len2 = 1
		}
	}
	return g
}
func (g *gradImage) ColorModel() color.Model { return color.RGBAModel }
func (g *gradImage) Bounds() image.Rectangle { return g.b }
func (g *gradImage) at(x, y int) color.RGBA {
	lx, ly := (float64(x)+.5-g.ox)/g.s, (float64(y)+.5-g.oy)/g.s
	var t float64
	if g.p.Kind == 1 {
		t = ((lx-g.p.X0)*g.dx + (ly-g.p.Y0)*g.dy) / g.len2
	} else {
		d := math.Hypot(lx-g.p.X0, ly-g.p.Y0)
		if g.p.R1 != g.p.R0 {
			t = (d - g.p.R0) / (g.p.R1 - g.p.R0)
		}
	}
	return g.lut[int(clamp01(t)*255)]
}
func (g *gradImage) At(x, y int) color.Color { return g.at(x, y) }

// ---------------------------------------------------------------- canvas

// Canvas draws in logical coordinates onto Dst: device = O + logical*S.
type Canvas struct {
	Dst    *image.RGBA
	S      float64
	OX, OY float64
	Alpha  float64
	ras    vector.Rasterizer
}

func NewCanvas(dst *image.RGBA, s, ox, oy float64) *Canvas {
	return &Canvas{Dst: dst, S: s, OX: ox, OY: oy, Alpha: 1}
}

func (c *Canvas) dev(x, y float64) (float64, float64) { return c.OX + x*c.S, c.OY + y*c.S }

func orient(poly []pt) float64 {
	a := 0.0
	for i := range poly {
		j := (i + 1) % len(poly)
		a += poly[i].X*poly[j].Y - poly[j].X*poly[i].Y
	}
	return a
}

// fillContours rasterizes device-space contours with paint.
func (c *Canvas) fillContours(cs [][]pt, p Paint) {
	if len(cs) == 0 {
		return
	}
	minX, minY, maxX, maxY := math.Inf(1), math.Inf(1), math.Inf(-1), math.Inf(-1)
	for _, ct := range cs {
		for _, q := range ct {
			minX, minY = math.Min(minX, q.X), math.Min(minY, q.Y)
			maxX, maxY = math.Max(maxX, q.X), math.Max(maxY, q.Y)
		}
	}
	bb := image.Rect(int(math.Floor(minX)), int(math.Floor(minY)), int(math.Ceil(maxX))+1, int(math.Ceil(maxY))+1)
	db := c.Dst.Bounds()
	vis := bb.Intersect(db)
	if vis.Empty() {
		return
	}
	var src image.Image
	var grad *gradImage
	if p.Kind == 0 {
		col := WithAlpha(p.C, c.Alpha)
		if col.A == 0 {
			return
		}
		src = image.NewUniform(col)
	} else {
		grad = newGrad(p, c.S, c.OX, c.OY, c.Alpha, db)
		src = grad
	}
	w, h := bb.Dx(), bb.Dy()
	c.ras.Reset(w, h)
	c.ras.DrawOp = draw.Over
	for _, ct := range cs {
		c.ras.MoveTo(float32(ct[0].X-float64(bb.Min.X)), float32(ct[0].Y-float64(bb.Min.Y)))
		for _, q := range ct[1:] {
			c.ras.LineTo(float32(q.X-float64(bb.Min.X)), float32(q.Y-float64(bb.Min.Y)))
		}
		c.ras.ClosePath()
	}
	if grad != nil {
		mask := image.NewAlpha(image.Rect(0, 0, w, h))
		c.ras.DrawOp = draw.Src
		c.ras.Draw(mask, mask.Bounds(), image.Opaque, image.Point{})
		blendGrad(c.Dst, vis, bb.Min, mask, grad)
		return
	}
	if vis == bb {
		c.ras.Draw(c.Dst, bb, src, bb.Min)
		return
	}
	// partly outside the target: rasterize to a mask, then let image/draw clip it
	mask := image.NewAlpha(image.Rect(0, 0, w, h))
	c.ras.DrawOp = draw.Src
	c.ras.Draw(mask, mask.Bounds(), image.Opaque, image.Point{})
	draw.DrawMask(c.Dst, vis, src, vis.Min, mask, vis.Min.Sub(bb.Min), draw.Over)
}

func (c *Canvas) toDev(p *Path, sameWinding bool) [][]pt {
	var out [][]pt
	for _, ct := range p.contours() {
		d := make([]pt, len(ct))
		for i, q := range ct {
			d[i].X, d[i].Y = c.dev(q.X, q.Y)
		}
		if sameWinding && orient(d) < 0 {
			for i, j := 0, len(d)-1; i < j; i, j = i+1, j-1 {
				d[i], d[j] = d[j], d[i]
			}
		}
		out = append(out, d)
	}
	return out
}

// Fill fills a path. Contours keep their own winding, so a reversed inner contour makes a hole.
func (c *Canvas) Fill(p *Path, paint Paint) { c.fillContours(c.toDev(p, false), paint) }

func (c *Canvas) FillRect(x, y, w, h float64, p Paint)  { c.Fill(RectPath(x, y, w, h), p) }
func (c *Canvas) FillRR(x, y, w, h, r float64, p Paint) { c.Fill(RoundRectPath(x, y, w, h, r), p) }
func (c *Canvas) FillEllipse(cx, cy, rx, ry float64, p Paint) {
	c.Fill(EllipsePath(cx, cy, rx, ry), p)
}

// StrokeRR strokes a rounded rectangle (the line is centred on the edge).
func (c *Canvas) StrokeRR(x, y, w, h, r, lw float64, p Paint) {
	path := &Path{}
	addRoundRect(path, x-lw/2, y-lw/2, w+lw, h+lw, r+lw/2, false)
	addRoundRect(path, x+lw/2, y+lw/2, w-lw, h-lw, math.Max(0, r-lw/2), true)
	c.Fill(path, p)
}
func (c *Canvas) StrokeEllipse(cx, cy, rx, ry, lw float64, p Paint) {
	path := &Path{}
	path.Arc(cx, cy, rx+lw/2, ry+lw/2, 0, 2*math.Pi)
	path.Close()
	path.Arc(cx, cy, math.Max(0, rx-lw/2), math.Max(0, ry-lw/2), 2*math.Pi, 0)
	path.Close()
	c.Fill(path, p)
}

// StrokePolyline draws a thick line with round joins and caps.
func (c *Canvas) StrokePolyline(pts []pt, lw float64, p Paint) {
	var cs [][]pt
	hw := lw / 2 * c.S
	dp := make([]pt, len(pts))
	for i, q := range pts {
		dp[i].X, dp[i].Y = c.dev(q.X, q.Y)
	}
	disc := func(q pt) []pt {
		n := int(math.Max(10, hw*1.5))
		d := make([]pt, n)
		for i := range d {
			a := float64(i) / float64(n) * 2 * math.Pi
			d[i] = pt{q.X + math.Cos(a)*hw, q.Y + math.Sin(a)*hw}
		}
		return d
	}
	for i := 0; i+1 < len(dp); i++ {
		a, b := dp[i], dp[i+1]
		dx, dy := b.X-a.X, b.Y-a.Y
		l := math.Hypot(dx, dy)
		if l == 0 {
			continue
		}
		nx, ny := -dy/l*hw, dx/l*hw
		cs = append(cs, []pt{{a.X + nx, a.Y + ny}, {b.X + nx, b.Y + ny}, {b.X - nx, b.Y - ny}, {a.X - nx, a.Y - ny}})
	}
	for _, q := range dp {
		cs = append(cs, disc(q))
	}
	for i := range cs {
		if orient(cs[i]) < 0 {
			for a, b := 0, len(cs[i])-1; a < b; a, b = a+1, b-1 {
				cs[i][a], cs[i][b] = cs[i][b], cs[i][a]
			}
		}
	}
	c.fillContours(cs, p)
}
func (c *Canvas) Line(x0, y0, x1, y1, lw float64, p Paint) {
	c.StrokePolyline([]pt{{x0, y0}, {x1, y1}}, lw, p)
}

// ---------------------------------------------------------------- images

type scaleKey struct {
	img  image.Image
	w, h int
	flip bool
}

var scaleCache = map[scaleKey]*image.RGBA{}
var halfCache = map[image.Image][]*image.RGBA{}

func toRGBA(img image.Image) *image.RGBA {
	if r, ok := img.(*image.RGBA); ok && r.Rect.Min == (image.Point{}) {
		return r
	}
	b := img.Bounds()
	r := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(r, r.Rect, img, b.Min, draw.Src)
	return r
}

// halve does a 2x2 box filter.
func halve(src *image.RGBA) *image.RGBA {
	w, h := src.Rect.Dx()/2, src.Rect.Dy()/2
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		r0 := src.Pix[2*y*src.Stride:]
		r1 := src.Pix[(2*y+1)*src.Stride:]
		d := dst.Pix[y*dst.Stride:]
		for x := 0; x < w; x++ {
			for k := 0; k < 4; k++ {
				i := 8*x + k
				d[4*x+k] = uint8((int(r0[i]) + int(r0[i+4]) + int(r1[i]) + int(r1[i+4]) + 2) / 4)
			}
		}
	}
	return dst
}

// Scaled returns img resized to w x h pixels (cached), with good quality when shrinking.
func Scaled(img image.Image, w, h int, flip bool) *image.RGBA {
	if w < 1 {
		w = 1
	}
	if h < 1 {
		h = 1
	}
	k := scaleKey{img, w, h, flip}
	if r, ok := scaleCache[k]; ok {
		return r
	}
	src := toRGBA(img)
	levels := halfCache[img]
	for src.Rect.Dx() >= w*2 && src.Rect.Dy() >= h*2 {
		i := 0
		for i < len(levels) && levels[i].Rect.Dx() >= src.Rect.Dx() {
			i++
		}
		if i < len(levels) {
			src = levels[i]
		} else {
			src = halve(src)
			levels = append(levels, src)
			halfCache[img] = levels
		}
	}
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	if src.Rect.Dx() == w && src.Rect.Dy() == h {
		copy(dst.Pix, src.Pix)
	} else {
		xdraw.CatmullRom.Scale(dst, dst.Rect, src, src.Rect, draw.Src, nil)
	}
	if flip { // rotate 180 degrees
		n := len(dst.Pix) / 4
		for i := 0; i < n/2; i++ {
			a, b := i*4, (n-1-i)*4
			for k := 0; k < 4; k++ {
				dst.Pix[a+k], dst.Pix[b+k] = dst.Pix[b+k], dst.Pix[a+k]
			}
		}
	}
	scaleCache[k] = dst
	return dst
}

// DrawImage draws img into the logical rect (x, y, w, h).
func (c *Canvas) DrawImage(img image.Image, x, y, w, h float64) { c.drawImg(img, x, y, w, h, false) }
func (c *Canvas) DrawImageFlipped(img image.Image, x, y, w, h float64) {
	c.drawImg(img, x, y, w, h, true)
}
func (c *Canvas) drawImg(img image.Image, x, y, w, h float64, flip bool) {
	if img == nil || c.Alpha <= 0 {
		return
	}
	dx, dy := c.dev(x, y)
	x0, y0 := int(math.Round(dx)), int(math.Round(dy))
	x1, y1 := int(math.Round(dx+w*c.S)), int(math.Round(dy+h*c.S))
	s := Scaled(img, x1-x0, y1-y0, flip)
	c.Blit(s, x0, y0)
}

// Blit draws a device-pixel image at device position x, y with the canvas alpha.
func (c *Canvas) Blit(s *image.RGBA, x, y int) {
	r := image.Rect(x, y, x+s.Rect.Dx(), y+s.Rect.Dy())
	if c.Alpha >= .999 {
		draw.Draw(c.Dst, r, s, s.Rect.Min, draw.Over)
	} else {
		draw.DrawMask(c.Dst, r, s, s.Rect.Min, image.NewUniform(color.Alpha{uint8(c.Alpha * 255)}), image.Point{}, draw.Over)
	}
}

// ---------------------------------------------------------------- sprites
// Shapes that are expensive to draw (gradients, shadows) are drawn once at the
// current scale and cached as images.

type sprite struct {
	img    *image.RGBA
	ox, oy float64 // logical offset of the image origin relative to the requested x, y
}

var spriteCache = map[string]*sprite{}
var spriteScale float64

// Cached draws fn (which draws in local logical coordinates 0..w, 0..h, plus pad
// on every side) once per key and scale, then blits it at x, y.
func (c *Canvas) Cached(key string, x, y, w, h, pad float64, fn func(cv *Canvas)) {
	if spriteScale != c.S {
		spriteCache = map[string]*sprite{}
		scaleCache = map[scaleKey]*image.RGBA{}
		spriteScale = c.S
	}
	if len(spriteCache) > 900 {
		spriteCache = map[string]*sprite{}
	}
	sp := spriteCache[key]
	// sub-pixel position matters little; snap the sprite to the device grid
	dx, dy := c.dev(x-pad, y-pad)
	if sp == nil {
		pw, ph := int(math.Ceil((w+2*pad)*c.S))+2, int(math.Ceil((h+2*pad)*c.S))+2
		img := image.NewRGBA(image.Rect(0, 0, pw, ph))
		cv := NewCanvas(img, c.S, pad*c.S, pad*c.S)
		fn(cv)
		sp = &sprite{img: img}
		spriteCache[key] = sp
	}
	c.Blit(sp.img, int(math.Round(dx)), int(math.Round(dy)))
}

// Shadow draws a soft drop shadow for a rounded rect.
func (c *Canvas) Shadow(x, y, w, h, r, blur, offY float64, col color.NRGBA) {
	key := "sh" + ftoa(w, h, r, blur) + hexOf(col)
	c.Cached(key, x, y+offY, w, h, blur*1.5, func(cv *Canvas) {
		cv.FillRR(0, 0, w, h, r, Solid(col)) // the sprite canvas is already offset by its padding
		boxBlur(cv.Dst, int(math.Max(1, blur*cv.S/2)))
	})
}

func ftoa(v ...float64) string {
	var b strings.Builder
	for _, x := range v {
		b.WriteString(strconv.FormatFloat(x, 'f', 1, 64))
		b.WriteByte(',')
	}
	return b.String()
}
func hexOf(c color.NRGBA) string {
	return strconv.FormatUint(uint64(c.R)<<24|uint64(c.G)<<16|uint64(c.B)<<8|uint64(c.A), 16)
}

// boxBlur blurs an RGBA image in place (3 passes ~ gaussian).
func boxBlur(img *image.RGBA, r int) {
	if r < 1 {
		return
	}
	w, h := img.Rect.Dx(), img.Rect.Dy()
	tmp := make([]uint8, len(img.Pix))
	pass := func(src, dst []uint8, horiz bool) {
		n, m := w, h
		if !horiz {
			n, m = h, w
		}
		for j := 0; j < m; j++ {
			var sum [4]int
			idx := func(i int) int {
				if i < 0 {
					i = 0
				}
				if i >= n {
					i = n - 1
				}
				if horiz {
					return j*img.Stride + i*4
				}
				return i*img.Stride + j*4
			}
			for i := -r; i <= r; i++ {
				o := idx(i)
				for k := 0; k < 4; k++ {
					sum[k] += int(src[o+k])
				}
			}
			d := 2*r + 1
			for i := 0; i < n; i++ {
				o := idx(i)
				for k := 0; k < 4; k++ {
					dst[o+k] = uint8(sum[k] / d)
				}
				a, b := idx(i-r), idx(i+r+1)
				for k := 0; k < 4; k++ {
					sum[k] += int(src[b+k]) - int(src[a+k])
				}
			}
		}
	}
	for i := 0; i < 3; i++ {
		pass(img.Pix, tmp, true)
		pass(tmp, img.Pix, false)
	}
}

// blendGrad composites a gradient through a coverage mask onto dst (source-over),
// stepping the gradient along each row: much faster than image/draw's generic path.
func blendGrad(dst *image.RGBA, vis image.Rectangle, maskOrigin image.Point, mask *image.Alpha, g *gradImage) {
	lin := g.p.Kind == 1
	var dr float64
	if !lin && g.p.R1 != g.p.R0 {
		dr = 1 / (g.p.R1 - g.p.R0)
	}
	for y := vis.Min.Y; y < vis.Max.Y; y++ {
		ly := (float64(y) + .5 - g.oy) / g.s
		mrow := mask.Pix[(y-maskOrigin.Y)*mask.Stride:]
		drow := dst.Pix[(y-dst.Rect.Min.Y)*dst.Stride:]
		lx0 := (float64(vis.Min.X) + .5 - g.ox) / g.s
		step := 1 / g.s
		var t, dt float64
		if lin {
			t = ((lx0-g.p.X0)*g.dx + (ly-g.p.Y0)*g.dy) / g.len2
			dt = step * g.dx / g.len2
		}
		for x := vis.Min.X; x < vis.Max.X; x++ {
			m := uint32(mrow[x-maskOrigin.X])
			if m != 0 {
				var tt float64
				if lin {
					tt = t
				} else {
					lx := lx0 + float64(x-vis.Min.X)*step
					tt = (math.Hypot(lx-g.p.X0, ly-g.p.Y0) - g.p.R0) * dr
				}
				if tt < 0 {
					tt = 0
				} else if tt > 1 {
					tt = 1
				}
				col := g.lut[int(tt*255)]
				sr, sg, sb, sa := uint32(col.R)*m/255, uint32(col.G)*m/255, uint32(col.B)*m/255, uint32(col.A)*m/255
				i := (x - dst.Rect.Min.X) * 4
				d := drow[i : i+4 : i+4]
				inv := 255 - sa
				d[0] = uint8(sr + uint32(d[0])*inv/255)
				d[1] = uint8(sg + uint32(d[1])*inv/255)
				d[2] = uint8(sb + uint32(d[2])*inv/255)
				d[3] = uint8(sa + uint32(d[3])*inv/255)
			}
			t += dt
		}
	}
}
