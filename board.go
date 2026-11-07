//go:build !headless

// The board, drawn every frame: pieces, highlights, legal-move dots, the
// gold rings on pieces that must jump, the hint arrow, move animations,
// confetti, speech bubbles and banners. (c) 2026 Ironite
package main

import (
	"image"
	"image/color"
	"math"
	"strings"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

var whitePx = func() *ebiten.Image {
	im := ebiten.NewImage(3, 3)
	im.Fill(color.White)
	return im.SubImage(image.Rect(1, 1, 2, 2)).(*ebiten.Image)
}()

func nrgba(r, g, b int, a float64) color.NRGBA { return RGBA(r, g, b, a) }

// pieceTex returns the piece image pre-scaled to one square at the current scale.
func (u *UI) pieceTex(code int8) *ebiten.Image {
	if u.texScale != u.s {
		for _, t := range u.pieceTex_ {
			t.Deallocate()
		}
		u.pieceTex_ = map[string]*ebiten.Image{}
		u.texScale = u.s
	}
	key := u.a.S.Pieces + string(rune('a'+code+2))
	if t, ok := u.pieceTex_[key]; ok {
		return t
	}
	px := int(math.Round(SQ * u.s))
	t := ebiten.NewImageFromImage(Scaled(PieceImg(u.a.S.Pieces, code), px, px, false))
	u.pieceTex_[key] = t
	return t
}

// drawPiece draws a piece in a logical square box (x, y = top-left in frame coordinates).
func (u *UI) drawPiece(dst *ebiten.Image, code int8, x, y, size, alpha float64) {
	t := u.pieceTex(code)
	op := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
	k := size * u.s / float64(t.Bounds().Dx())
	op.GeoM.Scale(k, k)
	dx, dy := u.dev(BX+x, BY+y)
	op.GeoM.Translate(math.Round(dx), math.Round(dy))
	op.ColorScale.ScaleAlpha(float32(alpha))
	dst.DrawImage(t, op)
}

func (u *UI) fillSq(dst *ebiten.Image, s int, col color.NRGBA) {
	x, y := u.a.sqXY(s)
	dx, dy := u.dev(BX+x, BY+y)
	vector.DrawFilledRect(dst, float32(dx), float32(dy), float32(SQ*u.s), float32(SQ*u.s), col, false)
}

func (u *UI) sqCentre(s int) (float32, float32) {
	x, y := u.a.sqXY(s)
	dx, dy := u.dev(BX+x+SQ/2, BY+y+SQ/2)
	return float32(dx), float32(dy)
}

func (u *UI) drawBoard(dst *ebiten.Image) {
	a := u.a
	if a.G == nil || a.pos == nil {
		return
	}
	now := time.Now()
	p := a.curPos()
	S := float32(u.s)
	// last move
	var last *histItem
	if a.live() && len(a.hist) > 0 {
		last = &a.hist[len(a.hist)-1]
	} else if !a.live() && a.view > 0 {
		last = &a.hist[a.view-1]
	}
	if last != nil && !(len(a.hops) > 0 && a.live()) {
		for _, s := range append([]int{last.m.From}, last.m.Path...) {
			u.fillSq(dst, s, nrgba(255, 196, 64, .26))
		}
	}
	// pieces that must jump
	mustJump := a.live() && a.G.Result == nil && a.yourTurn() && !a.busy && a.mustJumpNow() && a.sel < 0
	pulse := .6 + .4*math.Sin(float64(now.UnixMilli())/240)
	var jumpers []int
	if mustJump {
		seen := map[int]bool{}
		for _, m := range a.legal {
			if !seen[m.From] {
				seen[m.From] = true
				jumpers = append(jumpers, m.From)
			}
		}
		for _, s := range jumpers {
			u.fillSq(dst, s, nrgba(255, 205, 70, .22+.18*pulse))
		}
	}
	if a.flash != nil && now.Before(a.flash.until) {
		al := float64(a.flash.until.Sub(now)) / float64(900*time.Millisecond)
		for _, s := range a.flash.squares {
			x, y := a.sqXY(s)
			dx, dy := u.dev(BX+x+4, BY+y+4)
			vector.StrokeRect(dst, float32(dx), float32(dy), float32((SQ-8)*u.s), float32((SQ-8)*u.s), 6*S, nrgba(255, 210, 80, al), true)
		}
	}
	// selection and the path so far
	if a.sel >= 0 && a.live() {
		trail := append([]int{a.sel}, a.hops...)
		for i, s := range trail {
			al := .28
			if i == len(trail)-1 {
				al = .5
			}
			u.fillSq(dst, s, nrgba(255, 214, 90, al))
		}
	}
	// pieces
	dragging := a.drag != nil && a.drag.moved
	for s := 0; s < 32; s++ {
		pc := p.Board[s]
		if pc == 0 || a.hidden[s] || (dragging && a.drag.from == s) {
			continue
		}
		if a.live() && a.sel >= 0 && len(a.hops) > 0 && s == a.sel {
			continue // drawn at the last hop below
		}
		x, y := a.sqXY(s)
		al := 1.0
		if a.ghosts[s] {
			al = .32
		}
		u.drawPiece(dst, pc, x, y, SQ, al)
	}
	for s, pc := range a.lingering {
		x, y := a.sqXY(s)
		al := 1.0
		if a.ghosts[s] {
			al = .32
		}
		u.drawPiece(dst, pc, x, y, SQ, al)
	}
	if mustJump {
		for _, s := range jumpers {
			cx, cy := u.sqCentre(s)
			for k, w := range []float32{14, 9, 5} {
				vector.StrokeCircle(dst, cx, cy, float32(SQ*.42)*S, w*S, nrgba(255, 200, 60, (.08+.06*float64(k))*pulse), true)
			}
			vector.StrokeCircle(dst, cx, cy, float32(SQ*.42)*S, 4*S, nrgba(255, 214, 90, .55+.45*pulse), true)
		}
	}
	if a.live() && a.sel >= 0 && len(a.hops) > 0 && !dragging {
		x, y := a.sqXY(a.hops[len(a.hops)-1])
		u.drawPiece(dst, a.pos.Board[a.sel], x, y, SQ, 1)
	}
	// legal destinations
	if a.sel >= 0 && a.live() && a.S.Dots == "on" && !a.busy {
		jump := false
		for _, m := range a.selMoves() {
			if len(m.Caps) > 0 {
				jump = true
			}
		}
		for _, s := range a.nextSquares() {
			cx, cy := u.sqCentre(s)
			r, col := float32(SQ*.15)*S, nrgba(255, 240, 210, .45)
			if jump {
				r, col = float32(SQ*.2)*S, nrgba(255, 205, 70, .75)
			}
			vector.DrawFilledCircle(dst, cx, cy, r, col, true)
			vector.StrokeCircle(dst, cx, cy, r, 2*S, nrgba(40, 20, 5, .5), true)
		}
	}
	// animations (their timing and callbacks run in App.stepAnims)
	for _, an := range a.anims {
		if now.Before(an.t0) {
			continue
		}
		t := math.Min(1, float64(now.Sub(an.t0))/float64(an.dur))
		e := 2 * t * t
		if t >= .5 {
			e = 1 - math.Pow(-2*t+2, 2)/2
		}
		if an.fade {
			u.drawPiece(dst, an.piece, an.x0, an.y0-t*10, SQ, 1-t)
			continue
		}
		lf := .08
		if an.jump {
			lf = .32
		}
		lift := math.Sin(t*math.Pi) * SQ * lf
		x := an.x0 + (an.x1-an.x0)*e - lift*.1
		y := an.y0 + (an.y1-an.y0)*e - lift
		if lift > 1 { // a soft shadow under the lifted piece
			cx, cy := u.dev(BX+an.x0+(an.x1-an.x0)*e+SQ/2, BY+an.y0+(an.y1-an.y0)*e+SQ*.56)
			vector.DrawFilledCircle(dst, float32(cx), float32(cy), float32(SQ*.36)*S, nrgba(0, 0, 0, .18*lift/(SQ*lf)), true)
		}
		u.drawPiece(dst, an.piece, x, y, SQ+lift*.2, 1)
	}
	if a.hintPath != nil {
		u.pathArrow(dst, a.hintPath, nrgba(226, 170, 40, .9))
	}
	if dragging {
		pc := a.pos.Board[a.drag.from]
		sz := SQ * 1.14
		cx, cy := u.dev(BX+a.drag.x, BY+a.drag.y+SQ*.18)
		vector.DrawFilledCircle(dst, float32(cx), float32(cy), float32(SQ*.4)*S, nrgba(0, 0, 0, .25), true)
		u.drawPiece(dst, pc, a.drag.x-sz/2, a.drag.y-sz*.56, sz, 1)
	}
}

func (u *UI) pathArrow(dst *ebiten.Image, squares []int, col color.NRGBA) {
	S := float32(u.s)
	pts := make([][2]float32, len(squares))
	for i, s := range squares {
		x, y := u.sqCentre(s)
		pts[i] = [2]float32{x, y}
	}
	n := len(pts)
	q, pp := pts[n-1], pts[n-2]
	ang := math.Atan2(float64(q[1]-pp[1]), float64(q[0]-pp[0]))
	back := float32(SQ*.3) * S
	end := [2]float32{q[0] - float32(math.Cos(ang))*back, q[1] - float32(math.Sin(ang))*back}
	lw := float32(SQ*.16) * S
	// draw the shaft onto an offscreen layer so overlapping joints don't double the alpha
	off := ebiten.NewImage(dst.Bounds().Dx(), dst.Bounds().Dy())
	defer off.Deallocate()
	solid := color.NRGBA{col.R, col.G, col.B, 255}
	for i := 0; i < n-1; i++ {
		b := pts[i+1]
		if i == n-2 {
			b = end
		}
		vector.StrokeLine(off, pts[i][0], pts[i][1], b[0], b[1], lw, solid, true)
		vector.DrawFilledCircle(off, pts[i][0], pts[i][1], lw/2, solid, true)
	}
	vector.DrawFilledCircle(off, end[0], end[1], lw/2, solid, true)
	hl := float64(SQ*.38) * u.s
	var path vector.Path
	path.MoveTo(q[0], q[1])
	path.LineTo(q[0]-float32(math.Cos(ang-.5)*hl), q[1]-float32(math.Sin(ang-.5)*hl))
	path.LineTo(q[0]-float32(math.Cos(ang+.5)*hl), q[1]-float32(math.Sin(ang+.5)*hl))
	path.Close()
	vs, is := path.AppendVerticesAndIndicesForFilling(nil, nil)
	for i := range vs {
		vs[i].ColorR, vs[i].ColorG, vs[i].ColorB, vs[i].ColorA = float32(col.R)/255, float32(col.G)/255, float32(col.B)/255, 1
		vs[i].SrcX, vs[i].SrcY = 1, 1
	}
	off.DrawTriangles(vs, is, whitePx, &ebiten.DrawTrianglesOptions{AntiAlias: true})
	op := &ebiten.DrawImageOptions{}
	op.ColorScale.ScaleAlpha(float32(col.A) / 255)
	dst.DrawImage(off, op)
}

/* ---------------------------------------------------------------- particles */

func (u *UI) drawParticles(dst *ebiten.Image) {
	for _, p := range u.parts {
		al := 1 - math.Pow(p.age/p.life, 3)
		cx, cy := u.dev(p.x, p.y)
		hw, hh := p.w*u.s/2*math.Abs(math.Cos(p.rot*.7)), p.h*u.s/2
		c, s := math.Cos(p.rot), math.Sin(p.rot)
		corner := func(dx, dy float64) ebiten.Vertex {
			return ebiten.Vertex{DstX: float32(cx + dx*c - dy*s), DstY: float32(cy + dx*s + dy*c), SrcX: 1, SrcY: 1,
				ColorR: float32(p.col.R) / 255 * float32(al), ColorG: float32(p.col.G) / 255 * float32(al), ColorB: float32(p.col.B) / 255 * float32(al), ColorA: float32(al)}
		}
		vs := []ebiten.Vertex{corner(-hw, -hh), corner(hw, -hh), corner(hw, hh), corner(-hw, hh)}
		dst.DrawTriangles(vs, []uint16{0, 1, 2, 0, 2, 3}, whitePx, nil)
	}
}

/* ---------------------------------------------------------------- overlays */

func (u *UI) drawOverlays(dst *ebiten.Image) {
	now := time.Now()
	// speech bubbles
	keep := u.bubbles[:0]
	for _, b := range u.bubbles {
		t := now.Sub(b.start)
		if t > b.dur+250*time.Millisecond {
			b.img.Deallocate()
			continue
		}
		al := math.Min(1, float64(t)/float64(250*time.Millisecond))
		if t > b.dur {
			al = 1 - float64(t-b.dur)/float64(250*time.Millisecond)
		}
		sc := .85 + .15*math.Min(1, float64(t)/float64(250*time.Millisecond))
		op := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
		bw, bh := float64(b.img.Bounds().Dx()), float64(b.img.Bounds().Dy())
		op.GeoM.Translate(-bw/2, -bh/2)
		op.GeoM.Scale(sc, sc)
		cx, cy := u.dev(b.x+b.w/2, b.y+b.h/2)
		op.GeoM.Translate(cx, cy)
		op.ColorScale.ScaleAlpha(float32(al))
		dst.DrawImage(b.img, op)
		keep = append(keep, b)
	}
	u.bubbles = keep
	// banner over the board
	if b := u.banner_; b != nil {
		t := float64(now.Sub(b.start)) / float64(time.Millisecond)
		if t > 1950 {
			b.img.Deallocate()
			u.banner_ = nil
		} else {
			al, sc := 1.0, 1.0
			if t < 350 {
				al, sc = t/350, .8+.2*t/350
			} else if t > 1600 {
				al = 1 - (t-1600)/350
			}
			op := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
			bw, bh := float64(b.img.Bounds().Dx()), float64(b.img.Bounds().Dy())
			op.GeoM.Translate(-bw/2, -bh/2)
			op.GeoM.Scale(sc, sc)
			cx, cy := u.dev(BX+FR/2, BarH+844*.46)
			op.GeoM.Translate(cx, cy)
			op.ColorScale.ScaleAlpha(float32(al))
			dst.DrawImage(b.img, op)
		}
	}
	// the tune that's playing, bottom left, for five seconds
	if u.tune != "" {
		t := now.Sub(u.tuneAt).Seconds()
		if t < 6 {
			if u.tuneImg == nil {
				name := u.tune
				u.tuneImg = u.sprite(420, 24, func(c *Canvas) {
					noteIcon(c, 7, 12, Hex("#e9d2a6"))
					c.Text(name, 22, 17, TS{F: FItalic, Size: 13, C: Hex("#e9d2a6")}, 0)
				})
			}
			al := .75 * math.Min(1, t)
			if t > 5 {
				al = .75 * (6 - t)
			}
			op := &ebiten.DrawImageOptions{}
			dx, dy := u.dev(16, LH-30)
			op.GeoM.Translate(math.Round(dx), math.Round(dy))
			op.ColorScale.ScaleAlpha(float32(al))
			dst.DrawImage(u.tuneImg, op)
		}
	}
}

// animating reports whether anything on screen is moving (so frames must be drawn).
func (u *UI) animating() bool {
	a := u.a
	return len(a.anims) > 0 || len(u.parts) > 0 || len(u.bubbles) > 0 || u.banner_ != nil || a.drag != nil ||
		(a.flash != nil && time.Now().Before(a.flash.until)) || (u.tune != "" && time.Since(u.tuneAt) < 6*time.Second)
}

// The dim backdrop behind dialogs: a small radial gradient stretched over the window.
var veilTex = func() *ebiten.Image {
	const w, h = 160, 90
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			d := math.Hypot(float64(x)-w/2+.5, float64(y)-h/2+.5) / 90 // 0 centre .. ~1 corner
			al := .35 + (.82-.35)*math.Min(1, d)
			c := RGBA(int(10-5*d), int(6-3*d), int(3-d), al)
			i := y*img.Stride + x*4
			img.Pix[i], img.Pix[i+1], img.Pix[i+2], img.Pix[i+3] = uint8(float64(c.R)*al), uint8(float64(c.G)*al), uint8(float64(c.B)*al), c.A
		}
	}
	return ebiten.NewImageFromImage(img)
}()

func (u *UI) drawVeil(dst *ebiten.Image) {
	op := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
	op.GeoM.Scale(float64(u.W)/160, float64(u.H)/90)
	dst.DrawImage(veilTex, op)
}

// drawHover lightens whatever button the pointer is over.
func (u *UI) drawHover(dst *ebiten.Image) {
	h := u.hoverHit
	if h == nil || h.fn == nil {
		return
	}
	switch {
	case h.id == "vol", h.id == "sheet", strings.HasPrefix(h.id, "field-"), h.id[0] == 'l' && len(h.id) > 1 && h.id[1] >= 'A' && h.id[1] <= 'Z':
		return // sliders, fields and text links show a hand cursor only
	}
	r := 7.0
	switch {
	case strings.HasPrefix(h.id, "folk"):
		r = 12
	case strings.HasPrefix(h.id, "hf"), h.id == "of0", h.id == "of1":
		r = 15
	}
	x0, y0 := u.dev(h.x, h.y)
	w, hh, rr := h.w*u.s, h.h*u.s, r*u.s
	var p vector.Path
	p.MoveTo(float32(x0+rr), float32(y0))
	p.LineTo(float32(x0+w-rr), float32(y0))
	p.ArcTo(float32(x0+w), float32(y0), float32(x0+w), float32(y0+rr), float32(rr))
	p.LineTo(float32(x0+w), float32(y0+hh-rr))
	p.ArcTo(float32(x0+w), float32(y0+hh), float32(x0+w-rr), float32(y0+hh), float32(rr))
	p.LineTo(float32(x0+rr), float32(y0+hh))
	p.ArcTo(float32(x0), float32(y0+hh), float32(x0), float32(y0+hh-rr), float32(rr))
	p.LineTo(float32(x0), float32(y0+rr))
	p.ArcTo(float32(x0), float32(y0), float32(x0+rr), float32(y0), float32(rr))
	p.Close()
	vs, is := p.AppendVerticesAndIndicesForFilling(nil, nil)
	al := float32(.16)
	if strings.HasPrefix(h.id, "folk") {
		al = .1
	}
	for i := range vs {
		vs[i].SrcX, vs[i].SrcY = 1, 1
		vs[i].ColorR, vs[i].ColorG, vs[i].ColorB, vs[i].ColorA = al, al, al, al
	}
	dst.DrawTriangles(vs, is, whitePx, &ebiten.DrawTrianglesOptions{AntiAlias: true})
}
