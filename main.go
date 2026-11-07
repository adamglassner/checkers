//go:build !headless

// Checkers - Hollow Creek General Store. An Ironite game.
// Window, input and the frame loop. (c) 2026 Ironite
package main

import (
	"image"
	"math"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

type Game struct {
	a        *App
	u        *UI
	last     time.Time
	lastSize image.Point
	lastTick int64
}

// typing edits the focused text field on the title screen.
func (g *Game) typing() {
	u := g.u
	if u.fieldVal == nil {
		return
	}
	v := []rune(*u.fieldVal)
	changed := false
	for _, r := range ebiten.AppendInputChars(nil) {
		if len(v) < u.fieldMax && r >= ' ' {
			v = append(v, r)
			changed = true
		}
	}
	if k := inpututil.KeyPressDuration(ebiten.KeyBackspace); len(v) > 0 && (k == 1 || (k > 24 && k%3 == 0)) {
		v = v[:len(v)-1]
		changed = true
	}
	if changed {
		*u.fieldVal = string(v)
		g.a.sd.write(false)
		u.dirty()
	}
	jp := inpututil.IsKeyJustPressed
	switch {
	case jp(ebiten.KeyEscape):
		u.focus = ""
		u.dirty()
	case jp(ebiten.KeyEnter) || jp(ebiten.KeyNumpadEnter):
		f := u.focus
		u.focus = ""
		u.dirty()
		if f == "addr" {
			g.a.joinGame(g.a.S.Host)
		}
	case jp(ebiten.KeyTab):
		order := map[string]string{"hot0": "hot1", "hot1": "hot0", "name": "addr", "addr": "name"}
		if nx, ok := order[u.focus]; ok {
			u.focus = nx
			switch nx {
			case "hot0", "hot1":
				u.fieldVal = &g.a.S.HotNames[nx[3]-'0']
				u.fieldMax = 14
			case "name":
				u.fieldVal, u.fieldMax = &g.a.S.Name, 14
			case "addr":
				u.fieldVal, u.fieldMax = &g.a.S.Host, 40
			}
			u.dirty()
		}
	}
}

func (g *Game) logical() (float64, float64) {
	cx, cy := ebiten.CursorPosition()
	u := g.u
	if u.s == 0 {
		return -1, -1
	}
	return (float64(cx) - u.ox) / u.s, (float64(cy) - u.oy) / u.s
}

func inRect(x, y float64, h hit) bool { return x >= h.x && y >= h.y && x < h.x+h.w && y < h.y+h.h }

func (g *Game) Update() error {
	a, u := g.a, g.u
	now := time.Now()
	dt := math.Min(.05, now.Sub(g.last).Seconds())
	g.last = now
	a.update()
	u.stepParticles(dt)

	x, y := g.logical()
	modal := u.topModal()
	hits := u.hits
	if modal != "" {
		hits = u.modalHits
	}
	// hover
	hov := ""
	for i := len(hits) - 1; i >= 0; i-- {
		if inRect(x, y, hits[i]) {
			if hits[i].fn != nil {
				hov = hits[i].id
			}
			break
		}
	}
	if hov != u.hover {
		u.hover = hov
		u.hoverHit = nil
		for i := len(hits) - 1; i >= 0; i-- {
			if hits[i].id == hov && hov != "" {
				h := hits[i]
				u.hoverHit = &h
				break
			}
		}
	}
	bx, by := x-BX, y-BY
	onBoard := modal == "" && bx >= 0 && by >= 0 && bx < FR && by < FR
	hand := hov != "" || (onBoard && a.hoverable(a.xySq(bx, by)))
	if hand != u.cursorHand {
		u.cursorHand = hand
		if hand {
			ebiten.SetCursorShape(ebiten.CursorShapePointer)
		} else {
			ebiten.SetCursorShape(ebiten.CursorShapeDefault)
		}
	}

	// wheel: scroll the score sheet or a long dialog
	if _, wy := ebiten.Wheel(); wy != 0 {
		if modal == "rules" || modal == "credits" {
			u.panelScroll -= wy * 60
			u.dirty()
		} else if modal == "" && x >= sideX && x < sideX+sideW && y >= sheetY && y < sheetY+sheetH {
			u.sheetScroll -= wy * 60
			u.dirty()
		}
	}

	// mouse
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		if u.focus != "" { // clicking anywhere else leaves the text field
			u.focus = ""
			u.dirty()
		}
		handled := false
		for i := len(hits) - 1; i >= 0; i-- {
			if inRect(x, y, hits[i]) && hits[i].fn != nil {
				hits[i].fn()
				handled = true
				u.dirty()
				break
			}
		}
		if !handled && onBoard {
			a.boardDown(bx, by, false)
		}
	}
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonRight) && onBoard {
		a.boardDown(bx, by, true)
	}
	if u.volDrag {
		if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
			v := math.Round(math.Max(0, math.Min(100, (x-u.volX)/90*100)))
			if v != a.S.Vol {
				a.S.Vol = v
				a.synth.SetLevels(a.S.Music, a.S.Sfx, a.S.Vol)
				u.dirty()
			}
		} else {
			u.volDrag = false
			a.sd.write(false)
		}
	}
	if a.drag != nil {
		a.boardMove(bx, by)
		if !ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
			a.boardUp(bx, by)
		}
	}

	if u.shown["lobby"] || u.shown["joining"] { // the waiting dots
		if now.UnixMilli()/500 != g.lastTick {
			g.lastTick = now.UnixMilli() / 500
			u.dirty()
		}
	}
	if u.focus != "" && modal == "title" {
		g.typing()
		return nil
	}
	g.keys()
	if inpututil.IsKeyJustPressed(ebiten.KeyF11) || (inpututil.IsKeyJustPressed(ebiten.KeyEnter) && ebiten.IsKeyPressed(ebiten.KeyAlt)) {
		ebiten.SetFullscreen(!ebiten.IsFullscreen())
	}
	return nil
}

func (g *Game) keys() {
	a, u := g.a, g.u
	jp := inpututil.IsKeyJustPressed
	ctrl := ebiten.IsKeyPressed(ebiten.KeyControl) || ebiten.IsKeyPressed(ebiten.KeyMeta)
	if jp(ebiten.KeyM) {
		u.toggleMusic()
	}
	modal := u.topModal()
	if jp(ebiten.KeyEscape) {
		switch modal {
		case "settings", "rules", "credits":
			u.hide(modal)
			return
		}
		if a.sel >= 0 {
			a.clearSel()
			u.dirty()
		}
	}
	if modal == "title" {
		if jp(ebiten.KeyEnter) && !ebiten.IsKeyPressed(ebiten.KeyAlt) && a.sd.Setup.Mode != "online" {
			a.newGame(a.sd.Setup)
		}
		return
	}
	if modal != "" {
		return
	}
	switch {
	case (ctrl && jp(ebiten.KeyZ)) || jp(ebiten.KeyBackspace):
		a.undo()
	case jp(ebiten.KeyH):
		a.hint()
	case jp(ebiten.KeyF):
		a.sfx.Click()
		a.setFlip(!a.flipped)
		a.persist()
	case jp(ebiten.KeyN):
		u.newGamePressed()
	}
	if a.G != nil {
		n, v := len(a.hist), len(a.hist)
		if !a.live() {
			v = a.view
		}
		switch {
		case jp(ebiten.KeyArrowLeft):
			a.setView(v - 1)
		case jp(ebiten.KeyArrowRight):
			a.setView(v + 1)
		case jp(ebiten.KeyHome):
			a.setView(0)
		case jp(ebiten.KeyEnd):
			a.setView(n)
		}
	}
}

func (g *Game) Draw(screen *ebiten.Image) {
	u := g.u
	b := screen.Bounds()
	u.layout(b.Dx(), b.Dy())
	u.repaint()
	screen.Fill(cPage)
	screen.DrawImage(u.bg.img, nil)
	u.drawBoard(screen)
	drawSprites(screen, u.mainSprites)
	screen.DrawImage(u.main.img, nil)
	u.drawParticles(screen)
	u.drawOverlays(screen)
	if u.topModal() != "" && u.modalL.img != nil {
		u.drawVeil(screen)
		drawSprites(screen, u.modalSprites)
		screen.DrawImage(u.modalL.img, nil)
	}
	u.drawHover(screen)
}

// Layout works in real device pixels so the game is drawn at full resolution.
func (g *Game) LayoutF(w, h float64) (float64, float64) {
	s := ebiten.Monitor().DeviceScaleFactor()
	return math.Ceil(w * s), math.Ceil(h * s)
}
func (g *Game) Layout(w, h int) (int, int) { panic("LayoutF is used") }

func main() {
	loadFonts()
	sd := loadSave()
	synth := NewSynth()
	app := NewApp(sd, synth)
	setTestApp(app)
	ui := NewUI(app)
	synth.OnTune = func(name string) { ui.nowPlaying(name) }
	synth.InitMusic()
	synth.SetLevels(sd.Settings.Music, sd.Settings.Sfx, sd.Settings.Vol)
	startAudio(synth)
	if sd.Settings.Music {
		synth.StartMusic()
	}
	go preloadAssets()

	// the board behind the title screen: the unfinished game, or a fresh one
	if sd.Game != nil && sd.Game.Moves != nil {
		app.G = sd.Game
		app.rebuild()
		app.flipped = app.G.Flipped
	} else {
		f := true
		app.G = &GameRec{FEN: StartFEN, Moves: []string{}, Mode: "ai", Opp: sd.Setup.Opp, YouBlack: true, Forced: &f, Flipped: true, Started: time.Now().UnixMilli()}
		app.rebuild()
		app.flipped = true
	}
	ui.show("title")
	ui.boardDirty()

	ebiten.SetWindowTitle("Checkers")
	ebiten.SetWindowIcon(windowIcons())
	mw, mh := ebiten.Monitor().Size()
	w := int(math.Min(float64(mw)*.85, float64(mh)*.85*16/9))
	if w < 960 {
		w = 960
	}
	ebiten.SetWindowSize(w, w*9/16)
	ebiten.SetWindowSizeLimits(800, 450, -1, -1)
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	ebiten.SetTPS(60)
	g := &Game{a: app, u: ui, last: time.Now()}
	runGame(g)
	app.leaveOnline(true)
	sd.write(true)
	time.Sleep(150 * time.Millisecond) // let the goodbye reach the other player
}

func drawSprites(dst *ebiten.Image, ds []gpuDraw) {
	for _, d := range ds {
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(d.x, d.y)
		dst.DrawImage(d.img, op)
	}
}
