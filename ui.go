// Screens and panels: toolbar, side panel, score sheet and every dialog,
// painted with the 2D library at the window's real resolution (so text and
// edges stay sharp from 720p to 4K) and uploaded as layers.
// (c) 2026 Ironite
package main

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"strings"
	"time"
)

// layer is a full-window image painted by the 2D library.
type layer struct {
	rgba *image.RGBA
	img  gpuImg
}

func (l *layer) paint(w, h int, s, ox, oy float64, fn func(c *Canvas)) {
	if l.rgba == nil || l.rgba.Rect.Dx() != w || l.rgba.Rect.Dy() != h {
		l.rgba = image.NewRGBA(image.Rect(0, 0, w, h))
		freeImg(l.img)
		l.img = nil
	} else {
		clear(l.rgba.Pix)
	}
	fn(NewCanvas(l.rgba, s, ox, oy))
	l.img = uploadRGBA(l.rgba, l.img)
}

type hit struct {
	x, y, w, h float64
	id         string
	fn         func()
}

// gpuDraw is a cached background drawn by the GPU under a layer.
type gpuDraw struct {
	img  gpuImg
	x, y float64 // device pixels
}

// backdrop draws a static background (a panel with its shadow, say) from a
// cached GPU texture instead of painting it into the layer on every repaint.
// fn draws in local coordinates: (0, 0) is (x, y); pad leaves room for shadows.
func (u *UI) backdrop(c *Canvas, key string, x, y, w, h, pad float64, fn func(cv *Canvas)) {
	stacked := 0
	for _, id := range modalOrder {
		if u.shown[id] {
			stacked++
		}
	}
	if u.paintingModal && stacked > 1 { // dialogs on top of dialogs: paint in order
		c.Cached("bd"+key, x, y, w, h, pad, fn)
		return
	}
	if u.backdropScale != u.s {
		for _, im := range u.backdrops {
			freeImg(im)
		}
		u.backdrops = map[string]gpuImg{}
		u.backdropScale = u.s
	}
	im, ok := u.backdrops[key]
	if !ok {
		pw, ph := int(math.Ceil((w+2*pad)*u.s))+2, int(math.Ceil((h+2*pad)*u.s))+2
		rgba := image.NewRGBA(image.Rect(0, 0, pw, ph))
		fn(NewCanvas(rgba, u.s, pad*u.s, pad*u.s))
		im = uploadRGBA(rgba, nil)
		u.backdrops[key] = im
	}
	dx, dy := u.dev(x-pad, y-pad)
	d := gpuDraw{im, math.Round(dx), math.Round(dy)}
	if u.paintingModal {
		u.modalSprites = append(u.modalSprites, d)
	} else {
		u.mainSprites = append(u.mainSprites, d)
	}
}

type overInfo struct{ face, kick, title, text, quote string }

type confirmInfo struct {
	title, text, yes, no string
	cb                   func(bool)
}

type bannerT struct {
	img   gpuImg
	start time.Time
	w, h  float64
}

type bubbleT struct {
	img        gpuImg
	x, y, w, h float64 // logical, top-left
	start      time.Time
	dur        time.Duration
}

type UI struct {
	a                 *App
	W, H              int
	s, ox, oy         float64
	bg, main, modalL  layer
	bgDirty           bool
	uiDirty           bool
	hits, modalHits   []hit
	hover             string
	hoverHit          *hit
	paintHover        string // always empty: hover is drawn on the GPU, not painted
	mainSprites       []gpuDraw
	modalSprites      []gpuDraw
	backdrops         map[string]gpuImg
	backdropScale     float64
	paintingModal     bool
	shown             map[string]bool
	over              *overInfo
	conf              *confirmInfo
	statusOverride    string
	statusAlert       bool
	scrollToCurrent   bool
	sheetScroll       float64
	sheetMax          float64
	panelScroll       float64
	panelMax          float64
	volDrag           bool
	banner_           *bannerT
	bubbles           []*bubbleT
	tune              string
	tuneAt            time.Time
	parts             []*particle
	pieceTex_         map[string]gpuImg
	tuneImg           gpuImg
	volX              float64
	texScale          float64
	cursorHand        bool
	focus             string // the text field being typed in
	fieldMax          int
	fieldVal          *string
	msgTitle, msgText string
}

var modalOrder = []string{"title", "over", "lobby", "joining", "msg", "confirm", "settings", "rules", "credits"}

func NewUI(a *App) *UI {
	u := &UI{a: a, shown: map[string]bool{}, over: &overInfo{}, pieceTex_: map[string]gpuImg{}}
	a.ui = u
	return u
}

func (u *UI) dirty()      { u.uiDirty = true }
func (u *UI) boardDirty() { u.bgDirty = true; u.uiDirty = true }
func (u *UI) show(id string) {
	u.shown[id] = true
	u.panelScroll = 0
	u.dirty()
}
func (u *UI) hide(id string) { delete(u.shown, id); u.dirty() }
func (u *UI) topModal() string {
	top := ""
	for _, id := range modalOrder {
		if u.shown[id] {
			top = id
		}
	}
	return top
}

func (u *UI) message(title, text string) {
	u.msgTitle, u.msgText = title, text
	u.show("msg")
}

func (u *UI) ask(title, text, yes, no string, cb func(bool)) {
	u.conf = &confirmInfo{title, text, yes, no, cb}
	u.show("confirm")
}

// toDev converts logical to device coordinates.
func (u *UI) dev(x, y float64) (float64, float64) { return u.ox + x*u.s, u.oy + y*u.s }

/* ---------------------------------------------------------------- painting */

func (u *UI) layout(w, h int) {
	s := math.Min(float64(w)/LW, float64(h)/LH)
	ox, oy := (float64(w)-LW*s)/2, (float64(h)-LH*s)/2
	if w != u.W || h != u.H || s != u.s {
		u.W, u.H, u.s, u.ox, u.oy = w, h, s, math.Round(ox), math.Round(oy)
		u.bgDirty, u.uiDirty = true, true
	}
}

func (u *UI) repaint() {
	if u.bgDirty {
		u.bg.paint(u.W, u.H, u.s, u.ox, u.oy, u.paintBG)
		u.bgDirty = false
	}
	if u.uiDirty {
		u.hits = u.hits[:0]
		u.mainSprites = u.mainSprites[:0]
		u.paintingModal = false
		u.main.paint(u.W, u.H, u.s, u.ox, u.oy, u.paintMain)
		u.modalHits = u.modalHits[:0]
		u.modalSprites = u.modalSprites[:0]
		if u.topModal() != "" {
			u.paintingModal = true
			u.modalL.paint(u.W, u.H, u.s, u.ox, u.oy, u.paintModals)
			u.paintingModal = false
		}
		u.uiDirty = false
	}
}

func (u *UI) paintBG(c *Canvas) {
	a := u.a
	c.DrawImage(Img("scene.jpg"), 0, BarH, LW, LH-BarH)
	if a.flipped {
		c.DrawImageFlipped(Img("board-"+a.S.Board+".jpg"), BX, BY, FR, FR)
	} else {
		c.DrawImage(Img("board-"+a.S.Board+".jpg"), BX, BY, FR, FR)
	}
	if a.S.Nums == "on" {
		ts := TS{F: FBold, Size: 15, C: boardByID(a.S.Board).Num}
		for s := 0; s < 32; s++ {
			x, y := a.sqXY(s)
			c.Text(fmt.Sprint(s+1), BX+x+6, BY+y+19, ts, 0)
		}
	}
}

func (u *UI) addHit(x, y, w, h float64, id string, fn func()) {
	u.hits = append(u.hits, hit{x, y, w, h, id, fn})
}
func (u *UI) addModalHit(x, y, w, h float64, id string, fn func()) {
	u.modalHits = append(u.modalHits, hit{x, y, w, h, id, fn})
}

func mix(a, b color.NRGBA, t float64) color.NRGBA {
	l := func(x, y uint8) uint8 { return uint8(math.Round(float64(x) + (float64(y)-float64(x))*t)) }
	return color.NRGBA{l(a.R, b.R), l(a.G, b.G), l(a.B, b.B), l(a.A, b.A)}
}

// button draws a toolbar-style button; returns its width.
func (u *UI) button(c *Canvas, x, y float64, label, id string, enabled, off bool, fn func(), modal bool) float64 {
	t := TS{F: FBody, Size: 15, C: cBtnInk}
	note := strings.HasPrefix(label, "♫ ")
	if note {
		label = strings.TrimPrefix(label, "♫ ")
	}
	w := c.Measure(label, t) + 26
	if note {
		w += 18
	}
	h := 34.0
	c1, c2 := cBtn1, cBtn2
	if u.paintHover == id && enabled {
		c1, c2 = mix(c1, color.NRGBA{255, 255, 255, 255}, .15), mix(c2, color.NRGBA{255, 255, 255, 255}, .1)
	}
	al := c.Alpha
	if !enabled {
		c.Alpha = .45
	} else if off {
		c.Alpha = .55
	}
	c.Shadow(x, y, w, h, 7, 3, 1, RGBA(0, 0, 0, .5))
	c.FillRR(x, y, w, h, 7, Lin(0, y, 0, y+h, c1, c2))
	c.StrokeRR(x+.5, y+.5, w-1, h-1, 7, 1, Solid(cBtnEdge))
	c.Line(x+5, y+1.5, x+w-5, y+1.5, 1, Solid(RGBA(255, 255, 255, .5)))
	if note {
		noteIcon(c, x+15, y+17, cBtnInk)
		c.Text(label, x+w/2+9, y+22.5, t, 1)
	} else {
		c.Text(label, x+w/2, y+22.5, t, 1)
	}
	c.Alpha = al
	if enabled && fn != nil {
		if modal {
			u.addModalHit(x, y, w, h, id, fn)
		} else {
			u.addHit(x, y, w, h, id, fn)
		}
	}
	return w
}

func (u *UI) paintMain(c *Canvas) {
	u.paintBar(c)
	u.paintSide(c)
	// version, bottom right, and the tune that's playing, bottom left
	c.Alpha = .55
	c.Text("V"+Version+" · © 2026 IRONITE", LW-14, LH-9, TS{F: FBody, Size: 12, LS: 2, C: Hex("#e9d2a6")}, 2)
	c.Alpha = 1
}

func (u *UI) paintBar(c *Canvas) {
	a := u.a
	u.backdrop(c, "bar", 0, 0, LW, BarH, 12, func(cv *Canvas) {
		cv.Shadow(0, 0, LW, BarH, 0, 6, 3, RGBA(0, 0, 0, .55))
		cv.FillRect(0, 0, LW, BarH, Lin(0, 0, 0, BarH, cBar1, cBar2))
		cv.FillRect(0, BarH-1, LW, 1, Solid(RGBA(0, 0, 0, .6)))
	})
	x := 16.0
	tt := TS{F: FSign, Size: 27, LS: 3, C: cGold}
	c.TextShadow("Checkers", x, 37, tt, 0, RGBA(0, 0, 0, 1), 1)
	x += c.Measure("Checkers", tt) + 8
	st := TS{F: FBody, Size: 12, LS: 2, C: WithAlpha(cBarText, .75)}
	c.Text("HOLLOW CREEK", x, 36, st, 0)
	x += c.Measure("HOLLOW CREEK", st) + 22
	y := 11.0
	g := a.G
	ok := g != nil && g.Result == nil && a.live()
	gap := 7.0
	onl := g != nil && g.Mode == "online"
	x += u.button(c, x, y, "New Game", "bNew", true, false, func() { a.sfx.Click(); u.newGamePressed() }, false) + gap
	x += u.button(c, x, y, "Undo", "bUndo", g != nil && g.Result == nil && len(a.hist) > 0 && !onl, false, a.undo, false) + gap
	x += u.button(c, x, y, "Hint", "bHint", ok && a.yourTurn() && !a.busy && !onl, false, a.hint, false) + gap
	x += u.button(c, x, y, "Flip Board", "bFlip", true, false, func() { a.sfx.Click(); a.setFlip(!a.flipped); a.persist() }, false) + gap
	drawOK := ok && a.yourTurn() && len(a.hist) >= 2 && !(a.isAI() && len(a.hist)-a.drawPly < 10)
	x += u.button(c, x, y, "Offer Draw", "bDraw", drawOK, false, a.offerDraw, false) + gap
	x += u.button(c, x, y, "Resign", "bResign", ok && len(a.hist) >= 1, false, a.resign, false) + gap + 10
	x += u.button(c, x, y, "♫ Music", "bMusic", true, !a.S.Music, u.toggleMusic, false) + gap
	// volume slider
	sx, sw := x+4, 90.0
	u.volX = sx
	c.FillRR(sx, 25, sw, 6, 3, Solid(RGBA(0, 0, 0, .45)))
	fw := sw * a.S.Vol / 100
	c.FillRR(sx, 25, fw, 6, 3, Solid(cGold))
	c.FillEllipse(sx+fw, 28, 8, 8, Lin(0, 20, 0, 36, cBtn1, cBtn2))
	c.StrokeEllipse(sx+fw, 28, 8, 8, 1, Solid(cBtnEdge))
	u.addHit(sx-8, 14, sw+16, 28, "vol", func() { u.volDrag = true })
	x += sw + 8 + gap + 4
	x += u.button(c, x, y, "Sounds", "bSfx", true, !a.S.Sfx, u.toggleSfx, false) + gap
	x += u.button(c, x, y, "Options", "bSettings", true, false, func() { a.sfx.Click(); u.show("settings") }, false) + gap
	u.button(c, x, y, "Credits", "bCredits2", true, false, func() { u.show("credits") }, false)
	// stats on the right
	st2 := TS{F: FBody, Size: 16, C: cBarText}
	bt := TS{F: FBold, Size: 16, C: cGold}
	rx := LW - 16.0
	for _, p := range [][2]string{{"Lost", fmt.Sprint(a.sd.Stats.Lost)}, {"Drawn", fmt.Sprint(a.sd.Stats.Drawn)}, {"Won", fmt.Sprint(a.sd.Stats.Won)}} {
		rx -= c.Text(p[1], rx, 34, bt, 2)
		rx -= 5
		rx -= c.Text(p[0], rx, 34, st2, 2)
		rx -= 16
	}
}

func (u *UI) toggleMusic() {
	a := u.a
	a.S.Music = !a.S.Music
	u.applyAudio()
	a.sd.write(false)
	u.dirty()
}
func (u *UI) toggleSfx() {
	a := u.a
	a.S.Sfx = !a.S.Sfx
	u.applyAudio()
	a.sd.write(false)
	u.dirty()
}
func (u *UI) applyAudio() {
	a := u.a
	a.synth.SetLevels(a.S.Music, a.S.Sfx, a.S.Vol)
	if a.S.Music {
		a.synth.StartMusic()
	} else {
		a.synth.StopMusic()
	}
}

/* ---------------------------------------------------------------- side panel */

const (
	sideX, sideY, sideW = 872.0, BarH + 22.0, 700.0
	plaqueH             = 128.0
	sheetY              = sideY + plaqueH + 14
	statusY             = sideY + 790 - plaqueH - 14 - 50
	sheetH              = statusY - 14 - sheetY
	botY                = sideY + 790 - plaqueH
)

type plaqueInfo struct{ face, nm, rt string }

func (u *UI) plaqueFor(side int) plaqueInfo {
	a := u.a
	if a.G == nil {
		return plaqueInfo{"you", "You", ""}
	}
	if a.G.Mode == "pvp" {
		rt := a.sideName(side)
		if side == Black {
			rt += " · moves first"
		}
		return plaqueInfo{a.faceOf(side), a.playerName(side), rt}
	}
	if a.G.Mode == "online" {
		r := a.sd.Stats.Net
		if side == a.yourSide() {
			return plaqueInfo{a.S.Face, a.setupName(), fmt.Sprintf("You · %s · online record %d–%d–%d", a.sideName(side), r.W, r.D, r.L)}
		}
		where := "joined your game"
		if a.G.Online == "guest" {
			where = "hosting"
		}
		return plaqueInfo{a.faceOf(side), a.playerName(side), a.sideName(side) + " · " + where}
	}
	if side == a.yourSide() {
		r := a.sd.Stats.Per[a.G.Opp]
		if r == nil {
			r = &Rec{}
		}
		return plaqueInfo{"you", "You", fmt.Sprintf("Playing %s · vs %s: %d–%d–%d", a.sideName(side), a.opp().Name, r.W, r.D, r.L)}
	}
	o := a.opp()
	return plaqueInfo{o.ID, o.Full, o.Rating + " · " + a.sideName(side)}
}

func (u *UI) topSide() int {
	if u.a.flipped {
		return Red
	}
	return Black
}

func (u *UI) paintSide(c *Canvas) {
	a := u.a
	if a.G == nil || a.pos == nil {
		return
	}
	top := u.topSide()
	u.paintPlaque(c, sideY, top)
	u.paintPlaque(c, botY, -top)
	u.paintSheet(c)
	// status bar
	u.backdrop(c, "status", sideX, statusY, sideW, 50, 2, func(cv *Canvas) {
		cv.FillRR(0, 0, sideW, 50, 10, Solid(RGBA(20, 10, 4, .6)))
		cv.StrokeRR(.5, .5, sideW-1, 49, 10, 1, Solid(RGBA(234, 196, 106, .25)))
	})
	text, col := u.statusText()
	c.TextShadow(text, sideX+sideW/2, statusY+32, TS{F: FBody, Size: 21, C: col}, 1, RGBA(0, 0, 0, 1), 1)
}

func (u *UI) statusText() (string, color.NRGBA) {
	a := u.a
	alert, review := Hex("#ffd27a"), Hex("#bfe0ff")
	if !a.live() {
		return fmt.Sprintf("Reviewing move %d · press End to return", (a.view+1)/2), review
	}
	if u.statusOverride != "" {
		if u.statusAlert {
			return u.statusOverride, alert
		}
		return u.statusOverride, cCream
	}
	if a.G.Result != nil {
		return a.resultLine(), alert
	}
	jump := a.mustJumpNow()
	if a.G.Mode == "pvp" {
		who := a.playerName(a.pos.Side) + " (" + a.sideName(a.pos.Side) + ")"
		if len(a.hops) > 0 {
			return who + " · keep jumping!", alert
		}
		if jump {
			return who + " to move · must jump!", alert
		}
		return who + " to move", cCream
	}
	if a.G.Mode == "online" && !a.yourTurn() {
		return a.playerName(-a.yourSide()) + "'s move", cCream
	}
	if a.yourTurn() {
		switch {
		case len(a.hops) > 0:
			return "Keep jumping!", alert
		case jump:
			return "You must jump!", alert
		}
		return "Your move", cCream
	}
	return a.opp().Name + " is thinking…", cCream
}

func (u *UI) paintPlaque(c *Canvas, y float64, side int) {
	a := u.a
	x, w, h := sideX, sideW, plaqueH
	turn := a.G.Result == nil && a.pos.Side == side && a.live()
	u.backdrop(c, fmt.Sprint("plaque", turn), x, y, w, h, 30, func(cv *Canvas) {
		if turn {
			cv.Shadow(-2, -2, w+4, h+4, 16, 14, 0, RGBA(240, 190, 90, .45))
		}
		cv.Shadow(0, 0, w, h, 14, 11, 8, RGBA(0, 0, 0, .5))
		cv.FillRR(0, 0, w, h, 14, Lin(0, 0, 0, h, Hex("#4a2e18"), Hex("#2a180a")))
		if turn {
			cv.StrokeRR(1, 1, w-2, h-2, 14, 2, Solid(RGBA(240, 205, 120, .95)))
		} else {
			cv.StrokeRR(.5, .5, w-1, h-1, 14, 1, Solid(RGBA(234, 196, 106, .35)))
		}
		// the dark box for the piece count
		bx := w - 20 - 118
		cv.FillRR(bx, (h-82)/2, 118, 82, 10, Solid(RGBA(15, 8, 3, .55)))
		cv.StrokeRR(bx+.5, (h-82)/2+.5, 117, 81, 10, 1, Solid(RGBA(0, 0, 0, .5)))
	})
	info := u.plaqueFor(side)
	// portrait with a gold ring
	ax, ay := x+12+52, y+h/2
	c.Cached("ring104", ax-52, ay-52, 104, 104, 10, func(cv *Canvas) {
		cv.Shadow(0, 0, 104, 104, 52, 6, 4, RGBA(0, 0, 0, .6))
		cv.FillEllipse(52, 52, 55, 55, Solid(Hex("#c9a24c")))
	})
	c.DrawImage(Img("avatar-"+info.face+".png"), ax-52, ay-52, 104, 104)
	tx := x + 12 + 112 + 16
	c.Text(info.nm, tx, y+50, TS{F: FDisplay, Size: 26, LS: .5, C: cCream}, 0)
	c.Alpha = .78
	c.Text(info.rt, tx, y+73, TS{F: FBody, Size: 15, C: cBarText}, 0)
	c.Alpha = 1
	if a.thinking && a.isAI() && side == a.aiSide() {
		c.Text("Thinking…", tx+c.Measure(info.rt, TS{F: FBody, Size: 15})+14, y+73, TS{F: FItalic, Size: 15, C: cGold}, 0)
	}
	// captured pieces, and the material lead
	p := a.curPos()
	cnt := map[int]*[2]int{1: {}, -1: {}}
	for s := 0; s < 32; s++ {
		if pc := p.Board[s]; pc != 0 {
			cnt[sign(pc)][abs8(pc)-1]++
		}
	}
	taken := 12 - cnt[-side][0] - cnt[-side][1]
	cx := tx
	for i := 0; i < taken; i++ {
		c.DrawImage(PieceImg(a.S.Pieces, int8(-side)), cx, y+86, 32, 32)
		cx += 20
	}
	mine, theirs := cnt[side][0]+cnt[side][1], cnt[-side][0]+cnt[-side][1]
	if float64(cnt[side][0])+float64(cnt[side][1])*1.5 > float64(cnt[-side][0])+float64(cnt[-side][1])*1.5 && taken > 0 {
		if mine == theirs { // even on pieces, ahead on kings
			tw := c.Text("+", cx+24, y+108, TS{F: FBold, Size: 16, C: cGold}, 0)
			crownIcon(c, cx+24+tw+9, y+102, 7, cGold)
		} else {
			c.Text(fmt.Sprintf("+%d", mine-theirs), cx+24, y+108, TS{F: FBold, Size: 16, C: cGold}, 0)
		}
	}
	// pieces left
	bx := x + w - 20 - 118
	c.Text(fmt.Sprint(mine), bx+59, y+h/2+8, TS{F: FDisplay, Size: 36, C: Hex("#f6e2b8")}, 1)
	label := "PIECES"
	if k := cnt[side][1]; k > 0 {
		label = fmt.Sprintf("%d KING", k)
		if k > 1 {
			label += "S"
		}
	}
	c.Alpha = .75
	c.Text(label, bx+59, y+h/2+30, TS{F: FBody, Size: 13, LS: 1, C: cBarText}, 1)
	c.Alpha = 1
}

const sheetHeadH, sheetNavH, rowH = 40.0, 50.0, 34.0

func (u *UI) paintSheet(c *Canvas) {
	a := u.a
	x, y, w, h := sideX, sheetY, sideW, sheetH
	u.backdrop(c, "sheet", x, y, w, h, 34, func(cv *Canvas) {
		cv.Shadow(0, 0, w, h, 12, 13, 10, RGBA(0, 0, 0, .55))
		cv.FillRR(0, 0, w, h, 12, Lin(0, 0, 0, h, cPaper, Hex("#ead6ad")))
		cv.FillRect(0, sheetHeadH-1, w, 2, Solid(RGBA(190, 60, 50, .35)))
		ny := h - sheetNavH
		cv.FillRect(0, ny, w, 12, Solid(Hex("#ecd9b2")))
		cv.FillRR(0, ny, w, sheetNavH, 12, Solid(Hex("#ecd9b2")))
		cv.FillRect(0, ny, w, 12, Solid(Hex("#ecd9b2")))
		cv.FillRect(0, ny, w, 1, Solid(RGBA(110, 70, 30, .25)))
	})
	// header
	c.Text("SCORE SHEET", x+20, y+27, TS{F: FBold, Size: 15, LS: 3, C: cAccent}, 0)
	if a.G != nil && !a.forced() {
		c.Alpha = .75
		c.Text("Jumping optional", x+w-20, y+27, TS{F: FItalic, Size: 15, C: cInk}, 2)
		c.Alpha = 1
	}
	// the list, clipped and scrolled
	ly, lh := y+sheetHeadH+1, h-sheetHeadH-sheetNavH-1
	rows := (len(a.hist) + 1) / 2
	content := float64(rows)*rowH + 12
	if a.G.Result != nil {
		content += 40
	}
	u.sheetMax = math.Max(0, content-lh)
	shown := len(a.hist)
	if !a.live() {
		shown = a.view
	}
	if u.scrollToCurrent {
		u.scrollToCurrent = false
		cur := (shown - 1) / 2
		if shown == len(a.hist) {
			u.sheetScroll = u.sheetMax
		} else if top := float64(cur)*rowH + 6; top < u.sheetScroll {
			u.sheetScroll = top
		} else if top+rowH > u.sheetScroll+lh {
			u.sheetScroll = top + rowH - lh
		}
	}
	u.sheetScroll = math.Max(0, math.Min(u.sheetMax, u.sheetScroll))
	ps := setByID(a.S.Pieces)
	clip(c, x, ly, w, lh, func(cc *Canvas) {
		// ruled lines
		for ry := ly + rowH - 1 - math.Mod(u.sheetScroll, rowH); ry < ly+lh; ry += rowH {
			cc.FillRect(x, ry, w, 1, Solid(RGBA(110, 140, 190, .18)))
		}
		colW := (w - 28 - 70) / 2
		for i := 0; i < len(a.hist); i += 2 {
			ry := ly + 6 + float64(i/2)*rowH - u.sheetScroll
			if ry+rowH < ly || ry > ly+lh {
				continue
			}
			cc.Alpha = .55
			cc.Text(fmt.Sprintf("%d.", i/2+1), x+14+58, ry+23, TS{F: FBody, Size: 20, C: cInk}, 2)
			cc.Alpha = 1
			for j := 0; j < 2 && i+j < len(a.hist); j++ {
				k := i + j
				hi := a.hist[k]
				mx := x + 14 + 70 + float64(j)*colW
				cur := k == shown-1
				tc := cInk
				if cur {
					cc.FillRR(mx, ry, colW-6, 30, 6, Solid(cAccent))
					tc = Hex("#fff4dc")
				} else if u.paintHover == fmt.Sprint("mv", k) {
					cc.FillRR(mx, ry, colW-6, 30, 6, Solid(RGBA(110, 60, 20, .1)))
				}
				dc := ps.BCol
				if hi.m.Piece < 0 {
					dc = ps.RCol
				}
				cc.FillEllipse(mx+17, ry+15, 7, 7, Solid(dc))
				cc.FillEllipse(mx+17, ry+16.5, 7, 5.5, Solid(RGBA(0, 0, 0, .25)))
				cc.FillEllipse(mx+17, ry+14.2, 6.3, 6.3, Solid(dc))
				tw := cc.Text(hi.n, mx+30, ry+22, TS{F: FBody, Size: 20, C: tc}, 0)
				if hi.m.Crown {
					crownIcon(cc, mx+30+tw+12, ry+15, 7, Hex("#b07a1a"))
				}
				k2 := k
				if ry >= ly-10 && ry+30 <= ly+lh+10 {
					u.addHit(mx, math.Max(ry, ly), colW-6, math.Min(30, ly+lh-ry), fmt.Sprint("mv", k), func() { a.setView(k2 + 1) })
				}
			}
		}
		if a.G.Result != nil {
			ry := ly + 6 + float64(rows)*rowH - u.sheetScroll
			cc.Text(a.resultLine(), x+w/2, ry+26, TS{F: FBold, Size: 20, C: cAccent}, 1)
		}
	})
	// navigation
	ny := y + h - sheetNavH
	n := len(a.hist)
	v := n
	if !a.live() {
		v = a.view
	}
	type nb struct {
		kind   int
		target int
		ok     bool
	}
	btns := []nb{{0, 0, v > 0}, {1, v - 1, v > 0}, {2, v + 1, v < n}, {3, n, v < n}}
	bx := x + w/2 - (4*64+3*8)/2
	for i, b := range btns {
		id := fmt.Sprint("nav", i)
		bxx := bx + float64(i)*72
		al := 1.0
		if !b.ok {
			al = .4
		}
		c.Alpha = al
		c1, c2 := cBtn1, cBtn2
		if u.paintHover == id && b.ok {
			c1 = mix(c1, color.NRGBA{255, 255, 255, 255}, .15)
		}
		c.FillRR(bxx, ny+8, 64, 34, 7, Lin(0, ny+8, 0, ny+42, c1, c2))
		c.StrokeRR(bxx+.5, ny+8.5, 63, 33, 7, 1, Solid(cBtnEdge))
		navGlyph(c, b.kind, bxx+32, ny+25, cBtnInk)
		c.Alpha = 1
		if b.ok {
			t := b.target
			u.addHit(bxx, ny+8, 64, 34, id, func() { a.setView(t) })
		}
	}
	u.addHit(x, ly, w, lh, "sheet", nil)
}

// clip draws fn into the logical rect only.
func clip(c *Canvas, x, y, w, h float64, fn func(cc *Canvas)) {
	dx, dy := c.dev(x, y)
	x0, y0 := int(math.Floor(dx)), int(math.Floor(dy))
	pw, ph := int(math.Ceil(dx+w*c.S))-x0, int(math.Ceil(dy+h*c.S))-y0
	if pw <= 0 || ph <= 0 {
		return
	}
	tmp := image.NewRGBA(image.Rect(0, 0, pw, ph))
	cc := NewCanvas(tmp, c.S, c.OX-float64(x0), c.OY-float64(y0))
	fn(cc)
	c.Blit(tmp, x0, y0)
}

func navGlyph(c *Canvas, kind int, cx, cy float64, col color.NRGBA) {
	tri := func(x, dir float64) {
		p := &Path{}
		p.MoveTo(x+dir*6, cy-7)
		p.LineTo(x+dir*6, cy+7)
		p.LineTo(x-dir*6, cy)
		p.Close()
		c.Fill(p, Solid(col))
	}
	switch kind {
	case 0:
		c.FillRect(cx-10, cy-7, 2.5, 14, Solid(col))
		tri(cx-1, 1)
		tri(cx+8, 1)
	case 1:
		tri(cx, 1)
	case 2:
		tri(cx, -1)
	case 3:
		tri(cx-8, -1)
		tri(cx+1, -1)
		c.FillRect(cx+7.5, cy-7, 2.5, 14, Solid(col))
	}
}

// noteIcon: two beamed eighth notes, about 14px tall, centred at cx, cy.
func noteIcon(c *Canvas, cx, cy float64, col color.NRGBA) {
	c.FillEllipse(cx-4, cy+5, 3.4, 2.6, Solid(col))
	c.FillEllipse(cx+5, cy+3.5, 3.4, 2.6, Solid(col))
	c.FillRect(cx-1.4, cy-6, 1.4, 11, Solid(col))
	c.FillRect(cx+7.6, cy-7.5, 1.4, 11, Solid(col))
	p := &Path{}
	p.MoveTo(cx-1.4, cy-6)
	p.LineTo(cx+9, cy-7.5)
	p.LineTo(cx+9, cy-4.8)
	p.LineTo(cx-1.4, cy-3.3)
	p.Close()
	c.Fill(p, Solid(col))
}

func crownIcon(c *Canvas, cx, cy, r float64, col color.NRGBA) {
	p := &Path{}
	p.MoveTo(cx-r, cy+r*.6)
	p.LineTo(cx-r, cy-r*.4)
	p.LineTo(cx-r*.5, cy+r*.1)
	p.LineTo(cx, cy-r*.8)
	p.LineTo(cx+r*.5, cy+r*.1)
	p.LineTo(cx+r, cy-r*.4)
	p.LineTo(cx+r, cy+r*.6)
	p.Close()
	c.Fill(p, Solid(col))
}

/* ---------------------------------------------------------------- dialogs */

func (u *UI) paintModals(c *Canvas) {
	first := true
	for _, id := range modalOrder {
		if !u.shown[id] {
			continue
		}
		top := id == u.topModal()
		if !first {
			c.FillRect(0, 0, LW, LH, Solid(RGBA(5, 3, 2, .55)))
		}
		first = false
		saved := u.modalHits
		switch id {
		case "title":
			u.paintTitle(c)
		case "over":
			u.paintOver(c)
		case "confirm":
			u.paintConfirm(c)
		case "lobby":
			u.paintLobby(c)
		case "joining":
			u.paintJoining(c)
		case "msg":
			u.paintMsg(c)
		case "settings":
			u.paintSettings(c)
		case "rules":
			u.paintRules(c)
		case "credits":
			u.paintCredits(c)
		}
		if !top { // only the top dialog takes clicks
			u.modalHits = saved
		}
	}
}

func (u *UI) veil(c *Canvas) {
	c.FillRect(0, 0, LW, LH, Rad(LW/2, LH/2, 0, 900, RGBA(10, 6, 3, .35), RGBA(5, 3, 2, .82)))
}

func (u *UI) panel(c *Canvas, x, y, w, h float64) {
	u.backdrop(c, fmt.Sprintf("panel%.0fx%.0f", w, h), x, y, w, h, 80, func(cv *Canvas) {
		cv.Shadow(0, 0, w, h, 14, 34, 24, RGBA(0, 0, 0, .7))
		cv.FillRR(0, 0, w, h, 14, Lin(0, 0, 0, h, cPaper, cPaper2))
	})
}

type chipOpt struct{ v, label string }

func (u *UI) chipsWidth(c *Canvas, opts []chipOpt, size, padX float64) float64 {
	w := 0.0
	for _, o := range opts {
		w += c.Measure(o.label, TS{F: FBody, Size: size}) + 2*padX
	}
	return w + 2
}

// chips draws a segmented choice. x is the left edge.
func (u *UI) chips(c *Canvas, x, y float64, id string, opts []chipOpt, cur string, size, padX, h float64, pick func(v string)) float64 {
	w := u.chipsWidth(c, opts, size, padX)
	c.FillRR(x, y, w, h, 8, Solid(cBtnEdge))
	cx := x + 1
	for i, o := range opts {
		bw := c.Measure(o.label, TS{F: FBody, Size: size}) + 2*padX
		bg := RGBA(255, 255, 255, .55)
		tc := cInk
		if o.v == cur {
			bg, tc = cAccent, color.NRGBA{255, 255, 255, 255}
		} else if u.paintHover == id+o.v {
			bg = RGBA(255, 255, 255, .8)
		}
		r := 0.0
		if i == 0 || i == len(opts)-1 {
			r = 7
		}
		if r > 0 {
			c.FillRR(cx, y+1, bw, h-2, r, Solid(bg))
			// square the inner side
			if i == 0 && len(opts) > 1 {
				c.FillRect(cx+bw-r, y+1, r, h-2, Solid(bg))
			} else if i == len(opts)-1 && len(opts) > 1 {
				c.FillRect(cx, y+1, r, h-2, Solid(bg))
			}
		} else {
			c.FillRect(cx, y+1, bw, h-2, Solid(bg))
		}
		if i > 0 {
			c.FillRect(cx, y+1, 1, h-2, Solid(RGBA(0, 0, 0, .15)))
		}
		c.Text(o.label, cx+bw/2, y+h/2+size*.36, TS{F: FBody, Size: size, C: tc}, 1)
		v := o.v
		u.addModalHit(cx, y, bw, h, id+o.v, func() { u.a.sfx.Click(); pick(v) })
		cx += bw
	}
	return w
}

func (u *UI) bigButton(c *Canvas, x, y, w float64, label, id string, ghost bool, enabled bool, fn func()) {
	h := 52.0
	al := c.Alpha
	if !enabled {
		c.Alpha = .45
	}
	if ghost {
		if u.paintHover == id {
			c.FillRR(x, y, w, h, 10, Solid(RGBA(110, 60, 20, .08)))
		}
		c.StrokeRR(x+.5, y+.5, w-1, h-1, 10, 1.2, Solid(cInk))
	} else {
		c1, c2 := cBtn1, cBtn2
		if u.paintHover == id {
			c1, c2 = mix(c1, color.NRGBA{255, 255, 255, 255}, .15), mix(c2, color.NRGBA{255, 255, 255, 255}, .08)
		}
		c.Shadow(x, y, w, h, 10, 6, 3, RGBA(0, 0, 0, .4))
		c.FillRR(x, y, w, h, 10, Lin(0, y, 0, y+h, c1, c2))
		c.StrokeRR(x+.5, y+.5, w-1, h-1, 10, 1, Solid(cBtnEdge))
		c.Line(x+8, y+1.5, x+w-8, y+1.5, 1, Solid(RGBA(255, 255, 255, .55)))
	}
	tc := cBtnInk
	if ghost {
		tc = cInk
	}
	c.Text(label, x+w/2, y+34, TS{F: FDisplay, Size: 22, C: tc}, 1)
	c.Alpha = al
	if enabled {
		u.addModalHit(x, y, w, h, id, fn)
	}
}

// links draws centred underlined text links separated by dots.
func (u *UI) links(c *Canvas, cx, y float64, items [][2]string, fns []func()) {
	t := TS{F: FBody, Size: 15, C: cAccent}
	dot := "  ·  "
	total := 0.0
	for i, it := range items {
		total += c.Measure(it[1], t)
		if i > 0 {
			total += c.Measure(dot, TS{F: FBody, Size: 15})
		}
	}
	x := cx - total/2
	for i, it := range items {
		if i > 0 {
			x += c.Text(dot, x, y, TS{F: FBody, Size: 15, C: cInk}, 0)
		}
		w := c.Text(it[1], x, y, t, 0)
		c.FillRect(x, y+3.5, w, 1, Solid(cAccent))
		u.addModalHit(x, y-15, w, 22, it[0], fns[i])
		x += w
	}
}

// newGamePressed: back to the title screen (asking first if that would leave an online game).
func (u *UI) newGamePressed() {
	a := u.a
	if a.online() && a.G.Result == nil {
		u.ask("Leave this game?", "You're playing "+a.playerName(-a.yourSide())+" online. Leaving ends the game for both of you.", "Leave", "Keep Playing", func(yes bool) {
			if yes {
				a.leaveOnline(true)
				a.G = nil
				a.restoreTitleBoard()
				u.showTitle()
			}
		})
		return
	}
	if a.online() {
		a.leaveOnline(true)
		a.G = nil
		a.restoreTitleBoard()
	}
	u.showTitle()
}

// showTitle opens the title screen. The regulars get fresh names each time,
// except the one you have an unfinished game with.
func (u *UI) showTitle() {
	a := u.a
	Folks = rollNames()
	if g := a.sd.Game; g != nil && g.Result == nil && g.Mode == "ai" && g.OppFull != "" {
		for i := range Folks {
			if Folks[i].ID == g.Opp {
				Folks[i] = folkNamed(g.Opp, g.OppFull)
			}
		}
	}
	u.focus = ""
	u.hide("over")
	u.show("title")
}

func (u *UI) paintTitle(c *Canvas) {
	a := u.a
	setup := &a.sd.Setup
	w, h := 1080.0, 790.0
	x, y := (LW-w)/2, (LH-h)/2
	u.panel(c, x, y, w, h)
	cx := float64(LW) / 2
	// the store sign
	sw, sh := 380.0, 86.0
	sx, sy := cx-sw/2, y+22
	c.Shadow(sx, sy, sw, sh, 10, 10, 6, RGBA(0, 0, 0, .35))
	c.FillRR(sx, sy, sw, sh, 10, Lin(0, sy, 0, sy+sh, Hex("#2f4a33"), Hex("#1d3021")))
	c.StrokeRR(sx+1.5, sy+1.5, sw-3, sh-3, 9, 3, Solid(Hex("#d8b866")))
	c.StrokeRR(sx+6.5, sy+6.5, sw-13, sh-13, 5, 1, Solid(RGBA(216, 184, 102, .6)))
	signT := Hex("#f3e3b6")
	c.Alpha = .85
	c.Text("A GAME AT THE", cx, sy+24, TS{F: FBody, Size: 12, LS: 4, C: signT}, 1)
	c.Text("GENERAL STORE · EST. 1931", cx, sy+74, TS{F: FBody, Size: 12, LS: 4, C: signT}, 1)
	c.Alpha = 1
	c.Text("Hollow Creek", cx, sy+56, TS{F: FSign, Size: 34, LS: 5, C: signT}, 1)
	// title
	c.TextShadow("Checkers", cx, y+202, TS{F: FDisplay, Size: 86, LS: 4, C: cAccent}, 1, RGBA(255, 255, 255, .4), 2)
	// mode
	modes := []chipOpt{{"ai", "Play the Computer"}, {"pvp", "Same Computer"}, {"online", "Online"}}
	mw := u.chipsWidth(c, modes, 17, 22)
	u.chips(c, cx-mw/2, y+226, "mode", modes, setup.Mode, 17, 22, 42, func(v string) { setup.Mode = v; u.focus = ""; a.sd.write(false); u.dirty() })
	gy := y + 282
	switch setup.Mode {
	case "pvp":
		u.titleHotseat(c, x, gy, w)
	case "online":
		u.titleOnline(c, x, gy, w)
	default:
		u.titleComputer(c, x, gy, w)
	}
	// links and the Ironite mark
	by := y + h - 92
	u.links(c, cx, by+22, [][2]string{{"lHow", "How to Play"}, {"lOpt", "Options"}, {"lCred", "Credits & Changelog"}},
		[]func(){func() { a.sfx.Click(); u.show("rules") }, func() { a.sfx.Click(); u.show("settings") }, func() { u.show("credits") }})
	logo := Img("ironite-logo.png")
	lh := 26.0
	lw := lh * float64(logo.Bounds().Dx()) / float64(logo.Bounds().Dy())
	byT := TS{F: FBody, Size: 12, LS: 3, C: cInk}
	aw, gw := c.Measure("AN", byT), c.Measure("GAME", byT)
	total := aw + 8 + lw + 22 + 8 + gw
	lx := cx - total/2
	ly := by + 42
	c.Alpha = .9
	c.Text("AN", lx, ly+18, byT, 0)
	c.FillRR(lx+aw+8, ly, lw+22, lh+12, 6, Solid(Hex("#003b2f")))
	c.DrawImage(logo, lx+aw+8+11, ly+6, lw, lh)
	c.Text("GAME", lx+aw+8+lw+22+8, ly+18, byT, 0)
	c.Alpha = 1
}

// optionRow draws a centred "Label  [chips]" row.
func (u *UI) optionRow(c *Canvas, cx, yy float64, label, id string, opts []chipOpt, cur string, set func(string)) {
	cw := u.chipsWidth(c, opts, 15, 14)
	total := 140 + 12 + cw
	lx := cx - total/2
	c.Alpha = .8
	c.Text(label, lx+140, yy+25, TS{F: FBody, Size: 16, C: cInk}, 2)
	c.Alpha = 1
	u.chips(c, lx+152, yy, id, opts, cur, 15, 14, 36, set)
}

func (u *UI) titleComputer(c *Canvas, x, gy, w float64) {
	a := u.a
	setup := &a.sd.Setup
	cx := float64(LW) / 2
	cellW := (w - 80 - 50) / 6
	for i, f := range Folks {
		fx := x + 40 + float64(i)*(cellW+10)
		if setup.Opp == f.ID {
			c.FillRR(fx, gy, cellW, 210, 12, Solid(RGBA(142, 42, 30, .08)))
			c.StrokeRR(fx+1, gy+1, cellW-2, 208, 12, 2, Solid(cAccent))
		} else if u.paintHover == "folk"+f.ID {
			c.FillRR(fx, gy, cellW, 210, 12, Solid(RGBA(110, 60, 20, .08)))
		}
		ax := fx + cellW/2
		u.avatar(c, f.ID, ax, gy+12+56, 112)
		nameT := TS{F: FDisplay, Size: 17, C: cInk}
		ly := gy + 148
		for _, l := range c.Wrap(f.Full, nameT, cellW-8) {
			c.Text(l, ax, ly, nameT, 1)
			ly += 19
		}
		c.Text(f.Rating, ax, ly+1, TS{F: FBold, Size: 13, LS: 1, C: cAccent}, 1)
		c.Text(strings.Repeat("●", f.Level), ax, ly+17, TS{F: FBody, Size: 11, LS: 1, C: Hex("#b3802a")}, 1)
		r := a.sd.Stats.Per[f.ID]
		rec := "Not played yet"
		if r != nil && r.W+r.D+r.L > 0 {
			rec = fmt.Sprintf("You: %dW %dD %dL", r.W, r.D, r.L)
		}
		c.Alpha = .7
		c.Text(rec, ax, ly+34, TS{F: FBody, Size: 12, C: cInk}, 1)
		c.Alpha = 1
		id := f.ID
		u.addModalHit(fx, gy, cellW, 210, "folk"+f.ID, func() { setup.Opp = id; a.sd.write(false); a.sfx.Click(); u.dirty() })
	}
	c.Alpha = .85
	c.Text("“"+folkByID(setup.Opp).Tag+"”", cx, gy+238, TS{F: FItalic, Size: 16, C: cInk}, 1)
	c.Alpha = 1
	ps := setByID(a.S.Pieces)
	oy := gy + 262
	u.optionRow(c, cx, oy, "Play as", "color", []chipOpt{{"black", ps.Black + " (moves first)"}, {"random", "Random"}, {"red", ps.Red}}, setup.Color, func(v string) { setup.Color = v; a.sd.write(false); u.dirty() })
	u.optionRow(c, cx, oy+48, "Jumping", "forced", []chipOpt{{"on", "Must jump (standard)"}, {"off", "Optional (house rules)"}}, setup.Forced, func(v string) { setup.Forced = v; a.sd.write(false); u.dirty() })
	u.startButtons(c, cx, oy+104)
}

func (u *UI) startButtons(c *Canvas, cx, by float64) {
	a := u.a
	can := a.sd.Game != nil && a.sd.Game.Result == nil && len(a.sd.Game.Moves) > 0
	if can {
		u.bigButton(c, cx-7-226, by, 226, "Continue Game", "bContinue", true, true, func() { a.sfx.Click(); a.resumeGame() })
		u.bigButton(c, cx+7, by, 200, "Start Game", "bStart", false, true, func() { a.newGame(a.sd.Setup) })
	} else {
		u.bigButton(c, cx-100, by, 200, "Start Game", "bStart", false, true, func() { a.newGame(a.sd.Setup) })
	}
}

func (u *UI) titleHotseat(c *Canvas, x, gy, w float64) {
	a := u.a
	S := a.S
	cx := float64(LW) / 2
	ps := setByID(S.Pieces)
	c.Alpha = .85
	c.Text("Pass and play on one computer. The board turns to face whoever is moving.", cx, gy+14, TS{F: FItalic, Size: 16, C: cInk}, 1)
	c.Alpha = 1
	for i := 0; i < 2; i++ {
		i := i
		bx := cx - 470 + float64(i)*480
		by := gy + 36
		c.FillRR(bx, by, 460, 180, 14, Solid(RGBA(255, 255, 255, .35)))
		c.StrokeRR(bx+.5, by+.5, 459, 179, 14, 1, Solid(RGBA(110, 70, 30, .3)))
		side := []string{ps.Black + " · moves first", ps.Red}[i]
		c.Text(strings.ToUpper(side), bx+230, by+28, TS{F: FBold, Size: 13, LS: 2, C: cAccent}, 1)
		u.facePicker(c, bx+92, by+108, 96, fmt.Sprint("hf", i), &S.HotFaces[i])
		c.Alpha = .75
		c.Text("Name", bx+176, by+82, TS{F: FBody, Size: 14, C: cInk}, 0)
		c.Alpha = 1
		u.textField(c, bx+176, by+92, 250, fmt.Sprint("hot", i), &S.HotNames[i], 14, "Player "+fmt.Sprint(i+1))
	}
	oy := gy + 262
	u.optionRow(c, cx, oy+48, "Jumping", "forced", []chipOpt{{"on", "Must jump (standard)"}, {"off", "Optional (house rules)"}}, a.sd.Setup.Forced, func(v string) { a.sd.Setup.Forced = v; a.sd.write(false); u.dirty() })
	u.startButtons(c, cx, oy+104)
}

func (u *UI) titleOnline(c *Canvas, x, gy, w float64) {
	a := u.a
	S := a.S
	setup := &a.sd.Setup
	cx := float64(LW) / 2
	ps := setByID(S.Pieces)
	// you
	bx, by := cx-470, gy+4
	c.FillRR(bx, by, 300, 250, 14, Solid(RGBA(255, 255, 255, .35)))
	c.StrokeRR(bx+.5, by+.5, 299, 249, 14, 1, Solid(RGBA(110, 70, 30, .3)))
	c.Text("YOU", bx+150, by+28, TS{F: FBold, Size: 13, LS: 2, C: cAccent}, 1)
	u.facePicker(c, bx+150, by+102, 96, "of", &S.Face)
	u.textField(c, bx+25, by+172, 250, "name", &S.Name, 14, "Your name")
	c.Alpha = .65
	c.Text("Your friend sees this name and face", bx+150, by+232, TS{F: FBody, Size: 12, C: cInk}, 1)
	c.Alpha = 1
	// host
	hx := cx - 150
	c.FillRR(hx, by, 300, 250, 14, Solid(RGBA(255, 255, 255, .35)))
	c.StrokeRR(hx+.5, by+.5, 299, 249, 14, 1, Solid(RGBA(110, 70, 30, .3)))
	c.Text("HOST A GAME", hx+150, by+28, TS{F: FBold, Size: 13, LS: 2, C: cAccent}, 1)
	c.Alpha = .8
	c.Text("You play", hx+150, by+60, TS{F: FBody, Size: 14, C: cInk}, 1)
	c.Alpha = 1
	col := []chipOpt{{"black", ps.Black}, {"random", "Random"}, {"red", ps.Red}}
	cw := u.chipsWidth(c, col, 14, 12)
	u.chips(c, hx+150-cw/2, by+70, "color", col, setup.Color, 14, 12, 34, func(v string) { setup.Color = v; a.sd.write(false); u.dirty() })
	c.Alpha = .8
	c.Text("Jumping", hx+150, by+128, TS{F: FBody, Size: 14, C: cInk}, 1)
	c.Alpha = 1
	jo := []chipOpt{{"on", "Must jump"}, {"off", "Optional"}}
	jw := u.chipsWidth(c, jo, 14, 12)
	u.chips(c, hx+150-jw/2, by+138, "forced", jo, setup.Forced, 14, 12, 34, func(v string) { setup.Forced = v; a.sd.write(false); u.dirty() })
	u.bigButtonSmall(c, hx+40, by+190, 220, "Host a Game", "bHost", func() { u.focus = ""; a.hostGame() })
	// join
	jx := cx + 170
	c.FillRR(jx, by, 300, 250, 14, Solid(RGBA(255, 255, 255, .35)))
	c.StrokeRR(jx+.5, by+.5, 299, 249, 14, 1, Solid(RGBA(110, 70, 30, .3)))
	c.Text("JOIN A FRIEND", jx+150, by+28, TS{F: FBold, Size: 13, LS: 2, C: cAccent}, 1)
	c.Alpha = .8
	for i, l := range c.Wrap("Type the address your friend's game shows when they host.", TS{F: FBody, Size: 14}, 250) {
		c.Text(l, jx+150, by+62+float64(i)*20, TS{F: FBody, Size: 14, C: cInk}, 1)
	}
	c.Alpha = 1
	u.textField(c, jx+25, by+118, 250, "addr", &S.Host, 40, "e.g. 192.168.1.20")
	u.bigButtonSmall(c, jx+40, by+190, 220, "Join Game", "bJoin", func() { u.focus = ""; a.joinGame(S.Host) })
	c.Alpha = .7
	c.Text(fmt.Sprintf("Both computers need to be on the same network, or the host's port %d forwarded to their computer.", netPort), cx, gy+290, TS{F: FBody, Size: 13, C: cInk}, 1)
	c.Alpha = 1
}

// avatar draws a portrait in a gold ring, centred at cx, cy.
func (u *UI) avatar(c *Canvas, face string, cx, cy, d float64) {
	key := fmt.Sprint("ring", int(d))
	pad := 12.0
	c.Cached(key, cx-d/2, cy-d/2, d, d, pad, func(cv *Canvas) {
		cv.Shadow(0, 0, d, d, d/2, 6, 4, RGBA(0, 0, 0, .35))
		cv.FillEllipse(d/2, d/2, d/2+3, d/2+3, Solid(Hex("#c9a24c")))
	})
	c.DrawImage(Img("avatar-"+face+".png"), cx-d/2, cy-d/2, d, d)
}

// facePicker: a portrait with arrows to step through the faces.
func (u *UI) facePicker(c *Canvas, cx, cy, d float64, id string, face *string) {
	u.avatar(c, *face, cx, cy, d)
	step := func(dir int) func() {
		return func() {
			i := 0
			for k, f := range FaceChoices {
				if f == *face {
					i = k
				}
			}
			*face = FaceChoices[(i+dir+len(FaceChoices))%len(FaceChoices)]
			u.a.sfx.Click()
			u.a.sd.write(false)
			u.dirty()
		}
	}
	for k, dir := range []int{-1, 1} {
		ax := cx + float64(dir)*(d/2+22)
		hid := fmt.Sprint(id, k)
		bg := RGBA(255, 255, 255, .6)
		if u.paintHover == hid {
			bg = RGBA(255, 255, 255, .95)
		}
		c.FillEllipse(ax, cy, 15, 15, Solid(bg))
		c.StrokeEllipse(ax, cy, 15, 15, 1, Solid(cBtnEdge))
		p := &Path{}
		p.MoveTo(ax-float64(dir)*4, cy-7)
		p.LineTo(ax+float64(dir)*6, cy)
		p.LineTo(ax-float64(dir)*4, cy+7)
		p.Close()
		c.Fill(p, Solid(cInk))
		u.addModalHit(ax-17, cy-17, 34, 34, hid, step(dir))
	}
}

// textField draws an editable one-line field; clicking it gives it the keyboard.
func (u *UI) textField(c *Canvas, x, y, w float64, id string, val *string, max int, placeholder string) {
	h := 40.0
	on := u.focus == id
	c.FillRR(x, y, w, h, 8, Solid(color.NRGBA{255, 255, 255, 235}))
	if on {
		c.StrokeRR(x+1, y+1, w-2, h-2, 8, 2, Solid(cAccent))
	} else {
		c.StrokeRR(x+.5, y+.5, w-1, h-1, 8, 1, Solid(RGBA(110, 70, 30, .5)))
	}
	t := TS{F: FBody, Size: 18, C: cInk}
	if *val == "" && !on {
		c.Alpha = .45
		c.Text(placeholder, x+12, y+27, t, 0)
		c.Alpha = 1
	} else {
		tw := c.Text(*val, x+12, y+27, t, 0)
		if on {
			c.FillRect(x+13+tw, y+9, 2, 22, Solid(cAccent))
		}
	}
	u.addModalHit(x, y, w, h, "field-"+id, func() { u.focus = id; u.fieldMax, u.fieldVal = max, val; u.dirty() })
	if on {
		u.fieldMax, u.fieldVal = max, val
	}
}

func (u *UI) paintLobby(c *Canvas) {
	a := u.a
	n := a.net
	w, h := 640.0, 430.0
	x, y := (LW-w)/2, (LH-h)/2
	u.panel(c, x, y, w, h)
	cx := float64(LW) / 2
	c.Text("Hosting a Game", cx, y+66, TS{F: FDisplay, Size: 34, C: cAccent}, 1)
	c.Alpha = .8
	c.Text("Your friend opens Checkers, chooses Online, and joins with:", cx, y+110, TS{F: FBody, Size: 17, C: cInk}, 1)
	c.Alpha = 1
	ly := y + 160
	if n == nil || len(n.addrs) == 0 {
		c.Text("This computer's IP address", cx, ly, TS{F: FDisplay, Size: 30, C: cInk}, 1)
		ly += 44
	} else {
		for _, ad := range n.addrs {
			c.Text(ad, cx, ly, TS{F: FDisplay, Size: 30, C: cInk}, 1)
			ly += 44
		}
	}
	c.Alpha = .65
	c.Text(fmt.Sprintf("Port %d. Over the internet, forward that port to this computer.", netPort), cx, y+h-138, TS{F: FBody, Size: 14, C: cInk}, 1)
	c.Alpha = 1
	dots := strings.Repeat(".", int(time.Now().UnixMilli()/500%4))
	c.Text("Waiting for a friend to join"+dots, cx, y+h-104, TS{F: FItalic, Size: 18, C: cAccent}, 1)
	u.bigButtonSmall(c, cx-70, y+h-74, 140, "Cancel", "lobbyCancel", func() { a.leaveOnline(false); u.hide("lobby") })
}

func (u *UI) paintJoining(c *Canvas) {
	a := u.a
	w, h := 560.0, 260.0
	x, y := (LW-w)/2, (LH-h)/2
	u.panel(c, x, y, w, h)
	cx := float64(LW) / 2
	addr := ""
	if a.net != nil {
		addr = a.net.addr
	}
	c.Text("Joining "+addr, cx, y+70, TS{F: FDisplay, Size: 30, C: cAccent}, 1)
	dots := strings.Repeat(".", int(time.Now().UnixMilli()/500%4))
	msg := "Connecting" + dots
	if a.net != nil && a.net.peer != nil {
		msg = "Connected. Waiting for the host to start" + dots
	}
	c.Text(msg, cx, y+120, TS{F: FItalic, Size: 18, C: cInk}, 1)
	u.bigButtonSmall(c, cx-70, y+h-80, 140, "Cancel", "joinCancel", func() { a.leaveOnline(true); u.hide("joining") })
}

func (u *UI) paintMsg(c *Canvas) {
	w := 600.0
	t := TS{F: FBody, Size: 17, C: cInk}
	var lines []string
	for _, para := range strings.Split(u.msgText, "\n") {
		if para == "" {
			lines = append(lines, "")
			continue
		}
		lines = append(lines, c.Wrap(para, t, w-92)...)
	}
	h := 190 + float64(len(lines))*24
	x, y := (LW-w)/2, (LH-h)/2
	u.panel(c, x, y, w, h)
	cx := float64(LW) / 2
	c.Text(u.msgTitle, cx, y+64, TS{F: FDisplay, Size: 30, C: cAccent}, 1)
	ly := y + 104
	for _, l := range lines {
		c.Text(l, cx, ly, t, 1)
		ly += 24
	}
	u.bigButtonSmall(c, cx-60, y+h-72, 120, "OK", "msgOK", func() { u.hide("msg") })
}

func (u *UI) paintOver(c *Canvas) {
	a := u.a
	o := u.over
	w := 640.0
	t := TS{F: FBody, Size: 18, C: cInk}
	lines := c.Wrap(o.text, t, w-92)
	h := 470 + float64(len(lines)-1)*24
	x, y := (LW-w)/2, (LH-h)/2
	u.panel(c, x, y, w, h)
	cx := float64(LW) / 2
	c.Cached("ring120", cx-60, y+30, 120, 120, 12, func(cv *Canvas) {
		cv.Shadow(0, 0, 120, 120, 60, 8, 6, RGBA(0, 0, 0, .4))
		cv.FillEllipse(60, 60, 64, 64, Solid(Hex("#c9a24c")))
	})
	c.DrawImage(Img("avatar-"+o.face+".png"), cx-60, y+30, 120, 120)
	c.Alpha = .7
	c.Text(strings.ToUpper(o.kick), cx, y+178, TS{F: FBody, Size: 14, LS: 4, C: cInk}, 1)
	c.Alpha = 1
	c.Text(o.title, cx, y+232, TS{F: FDisplay, Size: 46, C: cAccent}, 1)
	ly := y + 270
	for _, l := range lines {
		c.Text(l, cx, ly, t, 1)
		ly += 24
	}
	if o.quote != "" {
		c.Alpha = .85
		c.Text(o.quote, cx, ly+14, TS{F: FItalic, Size: 18, C: cInk}, 1)
		c.Alpha = 1
	}
	by := y + h - 126
	canRematch := a.G.Mode != "online" || a.online()
	u.bigButton(c, cx-7-170, by, 170, "Rematch", "bRematch", false, canRematch, a.rematch)
	u.bigButton(c, cx+7, by, 190, "New Game", "bOverNew", true, true, func() { a.sfx.Click(); u.newGamePressed() })
	u.links(c, cx, by+88, [][2]string{{"lReview", "Review the game"}, {"lCopy", "Copy moves"}},
		[]func(){func() { u.hide("over"); a.setView(0) }, u.copyMoves})
}

func (u *UI) copyMoves() {
	if setClipboard(u.a.movesText()) {
		u.banner("Moves copied", "Squares numbered 1 to 32, Black first")
	} else {
		u.banner("Could not copy", "")
	}
}

func (u *UI) paintConfirm(c *Canvas) {
	f := u.conf
	if f == nil {
		return
	}
	w := 560.0
	t := TS{F: FBody, Size: 18, C: cInk}
	lines := c.Wrap(f.text, t, w-92)
	h := 196 + float64(len(lines))*25
	x, y := (LW-w)/2, (LH-h)/2
	u.panel(c, x, y, w, h)
	cx := float64(LW) / 2
	c.Text(f.title, cx, y+66, TS{F: FDisplay, Size: 30, C: cAccent}, 1)
	ly := y + 106
	for _, l := range lines {
		c.Text(l, cx, ly, t, 1)
		ly += 25
	}
	by := y + h - 86
	yw := math.Max(150, c.Measure(f.yes, TS{F: FDisplay, Size: 22})+68)
	nw := math.Max(150, c.Measure(f.no, TS{F: FDisplay, Size: 22})+68)
	bx := cx - (yw+14+nw)/2
	done := func(yes bool) func() {
		return func() { u.hide("confirm"); u.a.sfx.Click(); f.cb(yes) }
	}
	u.bigButton(c, bx, by, yw, f.yes, "cYes", false, true, done(true))
	u.bigButton(c, bx+yw+14, by, nw, f.no, "cNo", true, true, done(false))
}

func (u *UI) paintSettings(c *Canvas) {
	a := u.a
	S := a.S
	w, h := 880.0, 540.0
	x, y := (LW-w)/2, (LH-h)/2
	u.panel(c, x, y, w, h)
	c.Text("Options", x+46, y+64, TS{F: FDisplay, Size: 30, C: cAccent}, 0)
	set := func(p *string, rebuild bool) func(string) {
		return func(v string) {
			*p = v
			a.sd.write(false)
			if rebuild {
				u.boardDirty()
			}
			u.dirty()
		}
	}
	type row struct {
		label, id string
		opts      []chipOpt
		cur       *string
		rebuild   bool
	}
	rows := []row{
		{"Board", "board", []chipOpt{{"barrel", "Barrelhead"}, {"parlor", "Parlor Red"}, {"tourney", "Tournament Green"}}, &S.Board, true},
		{"Pieces", "pieces", []chipOpt{{"classic", "Red & Black"}, {"wood", "Walnut & Maple"}, {"caps", "Bottle Caps"}}, &S.Pieces, true},
		{"Show legal moves", "dots", []chipOpt{{"on", "On"}, {"off", "Off"}}, &S.Dots, false},
		{"Square numbers", "nums", []chipOpt{{"on", "Show"}, {"off", "Hide"}}, &S.Nums, true},
		{"Move speed", "anim", []chipOpt{{"slow", "Relaxed"}, {"normal", "Normal"}, {"fast", "Brisk"}}, &S.Anim, false},
		{"Two players", "turn", []chipOpt{{"on", "Turn the board each move"}, {"off", "Keep it still"}}, &S.Turn, false},
	}
	ry := y + 96
	for _, r := range rows {
		c.Alpha = .8
		c.Text(r.label, x+46+170, ry+24, TS{F: FBody, Size: 16, C: cInk}, 2)
		c.Alpha = 1
		u.chips(c, x+46+182, ry, r.id, r.opts, *r.cur, 15, 14, 36, set(r.cur, r.rebuild))
		ry += 50
	}
	c.Alpha = .7
	for i, l := range c.Wrap("The square numbers (1 to 32) are the ones used on the score sheet, the way checkers games have been written down for two hundred years.", TS{F: FBody, Size: 14}, w-92) {
		c.Text(l, x+46, ry+22+float64(i)*20, TS{F: FBody, Size: 14, C: cInk}, 0)
	}
	c.Alpha = 1
	u.bigButtonSmall(c, LW/2-60, y+h-70, 120, "Done", "sDone", func() { u.hide("settings") })
}

func (u *UI) bigButtonSmall(c *Canvas, x, y, w float64, label, id string, fn func()) {
	h := 44.0
	c1, c2 := cBtn1, cBtn2
	if u.paintHover == id {
		c1 = mix(c1, color.NRGBA{255, 255, 255, 255}, .15)
	}
	c.Shadow(x, y, w, h, 10, 6, 3, RGBA(0, 0, 0, .4))
	c.FillRR(x, y, w, h, 10, Lin(0, y, 0, y+h, c1, c2))
	c.StrokeRR(x+.5, y+.5, w-1, h-1, 10, 1, Solid(cBtnEdge))
	c.Text(label, x+w/2, y+29, TS{F: FDisplay, Size: 18, C: cBtnInk}, 1)
	u.addModalHit(x, y, w, h, id, fn)
}

// textPanel draws a scrollable text dialog; body draws content from y and returns the y after.
func (u *UI) textPanel(c *Canvas, title, closeLabel, id string, body func(cc *Canvas, x, y, w float64, draw bool) float64) {
	w := 860.0
	inner := w - 92
	// measure
	mh := body(c, 0, 0, inner, false)
	maxBody := 800.0 - 64 - 36 - 90
	bodyH := math.Min(mh, maxBody)
	h := 64 + 26 + bodyH + 90
	x, y := (LW-w)/2, (LH-h)/2
	u.panel(c, x, y, w, h)
	c.Text(title, x+46, y+60, TS{F: FDisplay, Size: 30, C: cAccent}, 0)
	top := y + 82
	u.panelMax = math.Max(0, mh-bodyH)
	u.panelScroll = math.Max(0, math.Min(u.panelMax, u.panelScroll))
	clip(c, x, top, w, bodyH, func(cc *Canvas) { body(cc, x+46, top-u.panelScroll, inner, true) })
	if u.panelMax > 0 { // scroll bar
		bh := bodyH * bodyH / mh
		by := top + (bodyH-bh)*u.panelScroll/u.panelMax
		c.FillRR(x+w-16, by, 6, bh, 3, Solid(RGBA(110, 70, 30, .4)))
		if u.panelScroll < u.panelMax-1 {
			c.FillRect(x+20, top+bodyH-1, w-40, 1, Solid(RGBA(110, 70, 30, .2)))
		}
	}
	bw := math.Max(130, c.Measure(closeLabel, TS{F: FDisplay, Size: 18})+52)
	u.bigButtonSmall(c, LW/2-bw/2, y+h-68, bw, closeLabel, id, func() { u.hide(u.topModal()) })
}

func (u *UI) paintRules(c *Canvas) {
	u.textPanel(c, "How to Play", "Back to the Board", "rDone", func(cc *Canvas, x, y, w float64, draw bool) float64 {
		const sz, lh = 17.0, 25.5
		col := cInk
		p := func(runs []Run, yy float64) float64 { return cc.RichParagraph(runs, x, yy, w, sz, lh, col, draw) + 10 }
		y += 4
		y = p([]Run{{"Each side starts with twelve pieces on the dark squares.", FBody}, {" Black moves first,", FBold}, {" then the players take turns. You win by capturing all of your opponent's pieces, or by leaving them with no move to make.", FBody}}, y)
		items := [][]Run{
			{{"Moving.", FBold}, {" Pieces move one square diagonally forward onto an empty dark square. Click a piece and then a square, or drag it there. Dots show where it can go.", FBody}},
			{{"Jumping.", FBold}, {" Capture an enemy piece by jumping diagonally over it onto the empty square just beyond. If you can keep jumping from where you land, you must, even changing direction. Click each landing square in turn, or just the last one.", FBody}},
			{{"You must jump.", FBold}, {" When a capture is available you have to take one (pick whichever you like). Pieces that can jump glow gold. On the title screen you can make jumping optional, as some families do.", FBody}},
			{{"Kings.", FBold}, {" A piece that reaches the far row is crowned and becomes a king, and its move ends there. Kings move and jump diagonally both forward and backward.", FBody}},
			{{"Draws.", FBold}, {" The same position three times, forty moves each with no capture and no plain piece moved, or a draw offer that's accepted.", FBody}},
		}
		for i, it := range items {
			if draw {
				cc.Text(fmt.Sprintf("%d.", i+1), x+18, y, TS{F: FBody, Size: sz, C: col}, 2)
			}
			y = cc.RichParagraph(it, x+26, y, w-26, sz, lh, col, draw) + 4
		}
		y += 8
		y = p([]Run{{"Six regulars at the Hollow Creek General Store will give you a game, from Little Tucker (just learning) to Old Gus, county champion. Your record against each of them is kept, and an unfinished game is waiting when you come back.", FBody}}, y)
		if draw {
			cc.Alpha = .7
		}
		y = cc.RichParagraph([]Run{{"Keys: Ctrl+Z undo · H hint · F flip board · ← → step through the moves · N new game · M music · Esc cancel a jump · F11 full screen", FBody}}, x, y, w, 14, 20, col, draw)
		cc.Alpha = 1
		return y
	})
}

func (u *UI) paintCredits(c *Canvas) {
	u.textPanel(c, "Credits", "Close", "cDone", func(cc *Canvas, x, y, w float64, draw bool) float64 {
		rows := [][2]string{{"Developed & Published by", "Ironite"}, {"Game Design", "Ironite"}, {"Art & Illustration", "Ironite"}, {"Checkers Engine", "Ironite"}, {"Music", "Ironite"}, {"", "“Potbelly Stove Waltz”, “Kings Row Breakdown”, “Penny Candy Rag”, “Sweet Tea on the Porch”"}, {"Sound Effects", "Ironite"}, {"Programming", "Ironite"}, {"Typefaces", "National Park, Young Serif, Work Sans (SIL Open Font License)"}, {"Built with", "Go and Ebitengine (Apache License 2.0)"}}
		t := TS{F: FBody, Size: 17, C: cInk}
		b := TS{F: FBold, Size: 17, C: cInk}
		for _, r := range rows {
			if r[0] == "" {
				if draw {
					cc.Alpha = .85
					cc.Text(r[1], x+18, y+20, TS{F: FItalic, Size: 15, C: cInk}, 0)
					cc.Alpha = 1
				}
				y += 28
				continue
			}
			if draw {
				cc.Alpha = .75
				cc.Text(r[0], x, y+24, t, 0)
				cc.Alpha = 1
				cc.Text(r[1], x+w, y+24, b, 2)
				for dx := 0.0; dx < w; dx += 7 {
					cc.FillRect(x+dx, y+35, 3.5, 1, Solid(RGBA(0, 0, 0, .18)))
				}
			}
			y += 37
		}
		if draw {
			cc.Alpha = .7
			cc.Text("Copyright © 2026 Ironite. All rights reserved.", x, y+26, TS{F: FBody, Size: 14, C: cInk}, 0)
			cc.Alpha = 1
			cc.Text("Changelog", x, y+76, TS{F: FDisplay, Size: 30, C: cAccent}, 0)
		}
		y += 92
		return u.changelog(cc, x, y, w, draw)
	})
}

func (u *UI) changelog(cc *Canvas, x, y, w float64, draw bool) float64 {
	started := false
	for _, l := range strings.Split(strings.ReplaceAll(changelogText(), "\r", ""), "\n") {
		switch {
		case strings.HasPrefix(l, "## "):
			started = true
			v, d := l[3:], ""
			if i := strings.Index(l, "]"); strings.HasPrefix(l, "## [") && i > 0 {
				v = l[4:i]
				d = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(l[i+1:]), "-"))
			}
			y += 30
			if draw {
				tw := cc.Text("Version "+v, x, y, TS{F: FDisplay, Size: 21, C: cAccent}, 0)
				cc.Alpha = .7
				cc.Text(d, x+tw+8, y, TS{F: FBody, Size: 13, C: cInk}, 0)
				cc.Alpha = 1
			}
			y += 6
		case strings.HasPrefix(l, "### ") && started:
			y += 24
			if draw {
				cc.Alpha = .7
				cc.Text(strings.ToUpper(l[4:]), x, y, TS{F: FBody, Size: 14, LS: 1, C: cInk}, 0)
				cc.Alpha = 1
			}
			y += 2
		case strings.HasPrefix(l, "- ") && started:
			y += 23
			if draw {
				cc.FillEllipse(x+10, y-5, 2.5, 2.5, Solid(cInk))
			}
			y = cc.RichParagraph([]Run{{strings.ReplaceAll(l[2:], "**", ""), FBody}}, x+24, y, w-24, 16, 23, cInk, draw) - 23
		}
	}
	return y + 16
}

/* ---------------------------------------------------------------- transient overlays */

// sprite renders a small logical-size image with the 2D library at the current scale.
func (u *UI) sprite(w, h float64, fn func(c *Canvas)) gpuImg {
	pw, ph := int(math.Ceil(w*u.s))+2, int(math.Ceil(h*u.s))+2
	rgba := image.NewRGBA(image.Rect(0, 0, pw, ph))
	fn(NewCanvas(rgba, u.s, 1, 1))
	return uploadRGBA(rgba, nil)
}

func (u *UI) banner(text, sub string) {
	if u.s == 0 {
		return
	}
	w, h := 900.0, 120.0
	img := u.sprite(w, h, func(c *Canvas) {
		t := TS{F: FDisplay, Size: 54, LS: 2, C: color.NRGBA{255, 255, 255, 255}}
		c.Text(text, w/2+2, 66+3, TS{F: FDisplay, Size: 54, LS: 2, C: RGBA(0, 0, 0, .45)}, 1)
		c.Text(text, w/2, 66, t, 1)
		if sub != "" {
			c.Text(sub, w/2+1, 102+2, TS{F: FBody, Size: 20, LS: 1, C: RGBA(0, 0, 0, .5)}, 1)
			c.Text(sub, w/2, 102, TS{F: FBody, Size: 20, LS: 1, C: RGBA(255, 255, 255, .92)}, 1)
		}
	})
	if u.banner_ != nil {
		freeImg(u.banner_.img)
	}
	u.banner_ = &bannerT{img: img, start: time.Now(), w: w, h: h}
}

func (u *UI) bubble(text string) {
	a := u.a
	if u.s == 0 || a.G == nil {
		return
	}
	topIsAI := u.topSide() == a.aiSide()
	t := TS{F: FItalic, Size: 18, C: Hex("#2b1d10")}
	tmp := NewCanvas(image.NewRGBA(image.Rect(0, 0, 1, 1)), u.s, 0, 0)
	lines := tmp.Wrap(text, t, 410)
	tw := 0.0
	for _, l := range lines {
		tw = math.Max(tw, tmp.Measure(l, t))
	}
	w, h := tw+30, float64(len(lines))*24+18
	down := topIsAI
	img := u.sprite(w, h+12, func(c *Canvas) {
		by := 0.0
		if down {
			by = 9
		}
		c.Shadow(0, by, w, h, 14, 9, 6, RGBA(0, 0, 0, .35))
		c.FillRR(0, by, w, h, 14, Solid(Hex("#fffaf0")))
		p := &Path{}
		if down {
			p.MoveTo(w/2-9, by+1)
			p.LineTo(w/2, 0)
			p.LineTo(w/2+9, by+1)
		} else {
			p.MoveTo(w/2-9, h-1)
			p.LineTo(w/2, h+9)
			p.LineTo(w/2+9, h-1)
		}
		p.Close()
		c.Fill(p, Solid(Hex("#fffaf0")))
		for i, l := range lines {
			c.Text(l, w/2, by+9+18+float64(i)*24, t, 1)
		}
	})
	px := sideX + 64 + 200
	var y float64
	if topIsAI {
		y = sideY - 56 + 140 + 4
	} else {
		y = botY - 56 - 12 - h + 40
	}
	u.bubbles = append(u.bubbles, &bubbleT{img: img, x: px - w/2, y: y, w: w, h: h + 12, start: time.Now(), dur: 3200 * time.Millisecond})
}

func (u *UI) nowPlaying(name string) { u.tune, u.tuneAt = name, time.Now(); u.tuneImg = nil }
