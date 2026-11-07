// Game flow: turns, the computer opponent, moves and jumps, undo, hints,
// draw offers, reviewing and the player's record. (c) 2026 Ironite
package main

import (
	"fmt"
	"math/rand"
	"strings"
	"sync/atomic"
	"time"
)

const Version = "2.0.0"

// Board geometry, in the game's 1600x900 logical coordinates.
const (
	LW, LH = 1600, 900
	BarH   = 56
	BX, BY = 30, BarH + 22 // board frame top-left
	FR     = 800           // frame size
	BM     = 40            // frame margin
	SQ     = 90            // square size
)

type histItem struct {
	m Move
	n string
}

type anim struct {
	piece          int8
	x0, y0, x1, y1 float64
	jump, fade     bool
	t0             time.Time
	dur            time.Duration
	started        bool
	onStart, done  func()
}

type dragState struct {
	from          int
	x, y, x0, y0  float64
	moved, wasSel bool
}

type flashT struct {
	squares []int
	until   time.Time
}

type timerT struct {
	at       time.Time
	fn       func()
	idleOnly bool // wait for animations to finish first
}

type aiReq struct {
	stop *atomic.Bool
	gen  int
	cb   func(AIResult)
}

type aiDone struct {
	id  int
	res AIResult
}

type App struct {
	sd    *SaveData
	S     *Settings
	synth *Synth
	sfx   *SFX

	G       *GameRec
	pos     *Position
	hist    []histItem
	legal   []Move
	view    int // -1 live, else number of plies shown
	viewPos *Position
	flipped bool

	sel       int
	hops      []int
	drag      *dragState
	anims     []*anim
	hidden    map[int]bool
	ghosts    map[int]bool
	lingering map[int]int8
	capCache  []int8
	hintPath  []int
	busy      bool
	thinking  bool
	gen       int
	drawPly   int
	flash     *flashT

	timers  []*timerT
	reqID   int
	pending map[int]*aiReq
	results chan aiDone

	ui  *UI
	net *Net
}

func NewApp(sd *SaveData, synth *Synth) *App {
	a := &App{sd: sd, S: &sd.Settings, synth: synth, view: -1, sel: -1, drawPly: -99,
		hidden: map[int]bool{}, ghosts: map[int]bool{}, lingering: map[int]int8{},
		pending: map[int]*aiReq{}, results: make(chan aiDone, 8)}
	a.sfx = &SFX{s: synth, caps: func() bool { return a.S.Pieces == "caps" }}
	return a
}

/* ---------------------------------------------------------------- helpers */

func (a *App) forced() bool { return a.G.forced() }
func (a *App) isAI() bool   { return a.G != nil && a.G.Mode == "ai" }
func (a *App) aiSide() int {
	if a.G.YouBlack {
		return Red
	}
	return Black
}
func (a *App) yourSide() int { return -a.aiSide() }
func (a *App) yourTurn() bool {
	return a.G != nil && a.G.Result == nil && (a.G.Mode == "pvp" || a.pos.Side == a.yourSide())
}
func (a *App) live() bool { return a.view < 0 }
func (a *App) curPos() *Position {
	if a.live() {
		return a.pos
	}
	return a.viewPos
}
func (a *App) opp() Folk { return folkNamed(a.G.Opp, a.G.OppFull) }

// playerName: what to call whoever plays `side` in messages.
func (a *App) playerName(s int) string {
	g := a.G
	switch {
	case g.Mode == "ai":
		if s == a.yourSide() {
			return "You"
		}
		return a.opp().Name
	case len(g.Names) == 2:
		if g.Mode == "online" && s == a.yourSide() {
			return "You"
		}
		if s == Black {
			return g.Names[0]
		}
		return g.Names[1]
	}
	return a.sideName(s)
}

// faceOf: the portrait for whoever plays `side`.
func (a *App) faceOf(s int) string {
	g := a.G
	if g.Mode == "ai" {
		if s == a.yourSide() {
			return "you"
		}
		return g.Opp
	}
	if len(g.Faces) == 2 {
		if s == Black {
			return g.Faces[0]
		}
		return g.Faces[1]
	}
	if s == Red {
		return "p2"
	}
	return "you"
}
func (a *App) animDur() time.Duration {
	switch a.S.Anim {
	case "slow":
		return 330 * time.Millisecond
	case "fast":
		return 120 * time.Millisecond
	}
	return 220 * time.Millisecond
}
func (a *App) sideName(s int) string {
	ps := setByID(a.S.Pieces)
	if s == Black {
		return ps.Black
	}
	return ps.Red
}
func (a *App) later(d time.Duration, fn func()) {
	a.timers = append(a.timers, &timerT{at: time.Now().Add(d), fn: fn})
}
func (a *App) whenIdle(fn func()) {
	a.timers = append(a.timers, &timerT{at: time.Now(), fn: fn, idleOnly: true})
}
func pick(list []string) string {
	if len(list) == 0 {
		return ""
	}
	return list[rand.Intn(len(list))]
}

func (a *App) rebuild() {
	a.pos = NewPosition(a.G.FEN)
	a.hist = a.hist[:0]
	for _, n := range a.G.Moves {
		m, ok := a.pos.FindMove(n, false)
		if !ok {
			break
		}
		a.hist = append(a.hist, histItem{m, n})
		a.pos.Make(m)
	}
	moves := make([]string, len(a.hist))
	for i, h := range a.hist {
		moves[i] = h.n
	}
	a.G.Moves = moves
	a.legal = a.pos.Legal(a.forced())
}

// persist saves the game in progress. Online games aren't saved (they can't
// be resumed alone), so they never replace an unfinished single-player game.
func (a *App) persist() {
	if a.G != nil && a.G.Mode == "online" {
		return
	}
	a.sd.Game = a.G
	a.sd.write(false)
}

/* ---------------------------------------------------------------- geometry */

// sqXY: top-left of a square, relative to the board frame origin.
func (a *App) sqXY(s int) (float64, float64) {
	r, c := rowOf(s), colOf(s)
	if a.flipped {
		return float64(BM + (7-c)*SQ), float64(BM + (7-r)*SQ)
	}
	return float64(BM + c*SQ), float64(BM + r*SQ)
}

// xySq: the dark square under a point in frame coordinates, or -1.
func (a *App) xySq(x, y float64) int {
	if x < BM || y < BM {
		return -1
	}
	c, r := int((x-BM)/SQ), int((y-BM)/SQ)
	if c < 0 || c > 7 || r < 0 || r > 7 {
		return -1
	}
	if a.flipped {
		c, r = 7-c, 7-r
	}
	return sqAt(r, c)
}

/* ---------------------------------------------------------------- AI */

func (a *App) think(o AIOpts, cb func(AIResult)) {
	a.reqID++
	id := a.reqID
	stop := &atomic.Bool{}
	a.pending[id] = &aiReq{stop: stop, gen: a.gen, cb: cb}
	p := a.pos.Clone()
	o.Forced = a.forced()
	go func() { a.results <- aiDone{id, Think(p, o, stop)} }()
}

// cancelThinking abandons any search in progress (new game, undo...).
func (a *App) cancelThinking() {
	for id, r := range a.pending {
		r.stop.Store(true)
		delete(a.pending, id)
	}
}

func (a *App) pollAI() {
	for {
		select {
		case d := <-a.results:
			if r, ok := a.pending[d.id]; ok {
				delete(a.pending, d.id)
				r.cb(d.res)
			}
		default:
			return
		}
	}
}

/* ---------------------------------------------------------------- moves */

func (a *App) playMove(m Move, skip int) {
	mover := a.pos.Side
	n := a.pos.Notation(m)
	a.capCache = make([]int8, len(m.Caps))
	for i, s := range m.Caps {
		a.capCache[i] = a.pos.Board[s]
	}
	a.hist = append(a.hist, histItem{m, n})
	a.G.Moves = append(a.G.Moves, n)
	a.sel, a.hops, a.hintPath = -1, nil, nil
	a.pos.Make(m)
	a.legal = a.pos.Legal(a.forced())
	if st := a.pos.Status(a.forced()); st.Over {
		a.G.Result = &Result{Reason: st.Reason, Winner: st.Winner}
	}
	a.persist()
	if a.G.Mode == "online" && mover == a.yourSide() {
		a.sendMove(len(a.hist)-1, n)
	}
	a.ui.statusOverride, a.ui.statusAlert = "", false
	if len(m.Caps) == 0 {
		a.sfx.Move()
	} else if skip == 0 {
		a.sfx.Hop(0)
	}
	a.ui.dirty()
	a.animateMove(m, skip, func() {
		a.ghosts, a.hidden, a.lingering = map[int]bool{}, map[int]bool{}, map[int]int8{}
		if len(m.Caps) > 0 {
			a.sfx.Capture(len(m.Caps))
		}
		if m.Crown {
			a.later(120*time.Millisecond, a.sfx.Crown)
			a.ui.crownBurst(m.To())
		}
		a.ui.dirty()
		if a.G.Result != nil {
			a.gameOver()
			return
		}
		if a.isAI() && mover == a.aiSide() {
			if m.Crown && rand.Float64() < .7 {
				a.quip("crown")
			} else if len(m.Caps) >= 2 && rand.Float64() < .8 {
				a.quip("jump")
			}
		}
		if a.G.Mode == "pvp" && a.S.Turn == "on" {
			a.later(250*time.Millisecond, func() { a.setFlip(a.pos.Side == Black) })
		}
		if a.isAI() && a.pos.Side == a.aiSide() {
			a.aiTurn()
		}
	})
}

// animateMove slides the piece along its path from hop `skip`, fading each
// captured piece as the jump passes over it.
func (a *App) animateMove(m Move, skip int, done func()) {
	dur := a.animDur()
	if len(m.Caps) > 0 {
		dur = dur * 5 / 4
	}
	to := m.To()
	stops := append([]int{m.From}, m.Path...)
	a.hidden[to] = true
	for i := skip; i < len(m.Caps); i++ {
		a.lingering[m.Caps[i]] = a.capturedPiece(m, i)
	}
	t0 := time.Now()
	for i := skip; i < len(m.Path); i++ {
		i := i
		x0, y0 := a.sqXY(stops[i])
		x1, y1 := a.sqXY(stops[i+1])
		k := i - skip
		last := i == len(m.Path)-1
		an := &anim{piece: m.Piece, x0: x0, y0: y0, x1: x1, y1: y1, jump: len(m.Caps) > 0, t0: t0.Add(time.Duration(k) * dur), dur: dur}
		if k > 0 {
			an.onStart = func() { a.sfx.Hop(k) }
		}
		an.done = func() {
			if i < len(m.Caps) {
				cx, cy := a.sqXY(m.Caps[i])
				delete(a.lingering, m.Caps[i])
				a.anims = append(a.anims, &anim{piece: a.capturedPiece(m, i), x0: cx, y0: cy, fade: true, t0: time.Now(), dur: dur * 9 / 10})
				delete(a.ghosts, m.Caps[i])
			}
			if last {
				delete(a.hidden, to)
				done()
			}
		}
		a.anims = append(a.anims, an)
	}
}

func (a *App) capturedPiece(m Move, i int) int8 {
	if i < len(a.capCache) && a.capCache[i] != 0 {
		return a.capCache[i]
	}
	if m.Piece > 0 {
		return -Man
	}
	return Man
}

func (a *App) aiTurn() {
	a.gen++
	my := a.gen
	a.busy, a.thinking = true, true
	a.ui.dirty()
	L, ok := Levels[a.G.Opp]
	if !ok {
		L = Levels["tucker"]
	}
	t0 := time.Now()
	a.think(L.AI, func(r AIResult) {
		if my != a.gen || a.G == nil || a.G.Result != nil {
			return
		}
		n := r.Move
		if !r.OK || n == "" {
			if len(a.legal) == 0 {
				return
			}
			n = a.pos.Notation(a.legal[0])
		}
		minWait := time.Duration(L.Delay[0]+rand.Intn(L.Delay[1]-L.Delay[0]+1)) * time.Millisecond
		wait := minWait - time.Since(t0)
		if wait < 0 {
			wait = 0
		}
		a.later(wait, func() {
			a.whenIdle(func() {
				if my != a.gen || a.G == nil || a.G.Result != nil {
					return
				}
				a.busy, a.thinking = false, false
				if m, ok := a.findLegal(n); ok {
					a.playMove(m, 0)
				}
				a.ui.dirty()
			})
		})
	})
}

func (a *App) findLegal(n string) (Move, bool) {
	for _, m := range a.legal {
		if a.pos.Notation(m) == n {
			return m, true
		}
	}
	return Move{}, false
}

/* ---------------------------------------------------------------- player input */

func (a *App) clearSel() { a.sel, a.hops = -1, nil; a.ghosts = map[int]bool{} }

func (a *App) selMoves() []Move {
	if a.sel < 0 {
		return nil
	}
	var out []Move
outer:
	for _, m := range a.legal {
		if m.From != a.sel || len(m.Path) < len(a.hops) {
			continue
		}
		for i, h := range a.hops {
			if m.Path[i] != h {
				continue outer
			}
		}
		out = append(out, m)
	}
	return out
}

func (a *App) nextSquares() []int {
	seen := map[int]bool{}
	var out []int
	for _, m := range a.selMoves() {
		if len(m.Path) > len(a.hops) {
			s := m.Path[len(a.hops)]
			if !seen[s] {
				seen[s] = true
				out = append(out, s)
			}
		}
	}
	return out
}

func (a *App) canMoveFrom(sq int) bool {
	if a.G == nil || a.G.Result != nil || !a.live() || a.busy || !a.yourTurn() {
		return false
	}
	for _, m := range a.legal {
		if m.From == sq {
			return true
		}
	}
	return false
}

func (a *App) ownPiece(sq int) bool {
	return a.G != nil && a.pos.Board[sq] != 0 && sign(a.pos.Board[sq]) == a.pos.Side
}

func (a *App) mustJumpNow() bool {
	return a.forced() && len(a.legal) > 0 && len(a.legal[0].Caps) > 0
}

func (a *App) selectSq(sq int) {
	a.clearSel()
	a.sel = sq
	if !strings.HasPrefix(a.ui.statusOverride, "Waiting") {
		a.ui.statusOverride, a.ui.statusAlert = "", false
	}
	a.sfx.Pick()
	a.ui.dirty()
}

// stepTo handles a click or drop on `to` while a piece is selected.
func (a *App) stepTo(to int) bool {
	cands := a.selMoves()
	var next []Move
	for _, m := range cands {
		if len(m.Path) > len(a.hops) && m.Path[len(a.hops)] == to {
			next = append(next, m)
		}
	}
	if len(next) > 0 {
		a.hops = append(a.hops, to)
		for _, m := range next {
			if len(m.Path) == len(a.hops) && len(next) == 1 {
				a.finishMove(m, len(a.hops)-1)
				return true
			}
		}
		i := len(a.hops) - 1
		if i < len(next[0].Caps) {
			a.ghosts[next[0].Caps[i]] = true
		}
		a.sfx.Hop(i)
		a.ui.dirty()
		return true
	}
	// straight to the final landing square, if that identifies one move
	var fin []Move
	for _, m := range cands {
		if m.To() == to {
			fin = append(fin, m)
		}
	}
	if len(fin) >= 1 {
		same := true
		for _, m := range fin[1:] {
			if !sameMove(m, fin[0]) {
				same = false
			}
		}
		if same {
			a.finishMove(fin[0], len(a.hops))
			return true
		}
	}
	return false
}

func (a *App) finishMove(m Move, skip int) {
	for i := 0; i < skip && i < len(m.Caps); i++ {
		a.ghosts[m.Caps[i]] = true
	}
	a.hops, a.sel = nil, -1
	a.playMove(m, skip)
}

func (a *App) nudgeJump(sq int) {
	seen := map[int]bool{}
	var js []int
	for _, m := range a.legal {
		if !seen[m.From] {
			seen[m.From] = true
			js = append(js, m.From)
		}
	}
	a.flash = &flashT{js, time.Now().Add(900 * time.Millisecond)}
	a.sfx.MustJump()
	sub := ""
	if a.ownPiece(sq) {
		sub = "A glowing piece has a capture to make"
	}
	a.ui.banner("You must jump!", sub)
}

// Board pointer events, in frame coordinates (0..800).
func (a *App) boardDown(x, y float64, right bool) {
	a.synthUnlock()
	if right {
		if len(a.hops) > 0 {
			a.clearSel()
			a.ui.dirty()
		}
		return
	}
	sq := a.xySq(x, y)
	if a.G == nil {
		return
	}
	if !a.live() {
		a.setView(len(a.hist))
		return
	}
	if sq < 0 {
		if len(a.hops) == 0 {
			a.clearSel()
		}
		return
	}
	if a.sel >= 0 && sq != a.sel && (len(a.hops) == 0 || sq != a.hops[len(a.hops)-1]) && a.stepTo(sq) {
		return
	}
	if len(a.hops) > 0 { // mid-jump: only the piece's next landings are allowed
		if sq == a.sel || sq == a.hops[len(a.hops)-1] {
			return
		}
		a.sfx.Illegal()
		a.flash = &flashT{a.nextSquares(), time.Now().Add(900 * time.Millisecond)}
		return
	}
	if a.canMoveFrom(sq) {
		wasSel := a.sel == sq
		if !wasSel {
			a.selectSq(sq)
		}
		a.drag = &dragState{from: sq, x: x, y: y, x0: x, y0: y, wasSel: wasSel}
	} else {
		if a.yourTurn() && !a.busy && a.ownPiece(sq) && a.mustJumpNow() {
			a.nudgeJump(sq)
		} else if a.sel >= 0 && a.pos.Board[sq] != 0 {
			a.sfx.Illegal()
		}
		a.clearSel()
	}
	a.ui.dirty()
}

func (a *App) boardMove(x, y float64) {
	if a.drag == nil {
		return
	}
	a.drag.x, a.drag.y = x, y
	if !a.drag.moved && (x-a.drag.x0)*(x-a.drag.x0)+(y-a.drag.y0)*(y-a.drag.y0) > 36 {
		a.drag.moved = true
	}
}

func (a *App) boardUp(x, y float64) {
	d := a.drag
	if d == nil {
		return
	}
	a.drag = nil
	sq := a.xySq(x, y)
	if d.moved {
		if sq >= 0 && sq != d.from && a.stepTo(sq) {
			return
		}
		if sq >= 0 && sq != d.from {
			a.sfx.Illegal()
		}
	} else if d.wasSel {
		a.clearSel()
	}
	a.ui.dirty()
}

// hoverable reports whether the pointer over this square should show a hand.
func (a *App) hoverable(sq int) bool {
	if sq < 0 {
		return false
	}
	if a.canMoveFrom(sq) {
		return true
	}
	if a.sel >= 0 {
		for _, s := range a.nextSquares() {
			if s == sq {
				return true
			}
		}
		for _, m := range a.selMoves() {
			if m.To() == sq {
				return true
			}
		}
	}
	return false
}

func (a *App) setFlip(f bool) {
	if f == a.flipped {
		return
	}
	a.flipped = f
	if a.G != nil {
		a.G.Flipped = f
	}
	a.ui.boardDirty()
}

/* ---------------------------------------------------------------- reviewing */

func (a *App) setView(v int) {
	if a.G == nil {
		return
	}
	a.clearSel()
	if v >= len(a.hist) {
		a.view, a.viewPos = -1, nil
	} else {
		if v < 0 {
			v = 0
		}
		a.view = v
		a.viewPos = NewPosition(a.G.FEN)
		for i := 0; i < v; i++ {
			a.viewPos.Make(a.hist[i].m)
		}
	}
	a.hintPath = nil
	a.sfx.Click()
	a.ui.dirty()
	a.ui.scrollToCurrent = true
}

/* ---------------------------------------------------------------- game over */

func (a *App) quip(kind string) {
	if !a.isAI() {
		return
	}
	q := Quips[a.G.Opp]
	if q == nil {
		q = Quips["tucker"]
	}
	if s := pick(q[kind]); s != "" {
		a.ui.bubble(s)
	}
}

func (a *App) resultLine() string {
	r := a.G.Result
	if r == nil {
		return ""
	}
	why := map[string]string{"captured": "took every piece", "blocked": "left no moves", "resignation": "by resignation", "threefold repetition": "Threefold repetition", "forty-move rule": "Forty-move rule", "agreement": "Draw agreed"}[r.Reason]
	if why == "" {
		why = r.Reason
	}
	if r.Winner == 0 {
		return "Draw · " + why
	}
	who, verb := a.sideName(r.Winner), " wins"
	if a.G.Mode != "ai" && len(a.G.Names) == 2 {
		who = a.playerName(r.Winner)
		if who == "You" {
			verb = " win"
		}
	}
	if r.Reason == "resignation" {
		return who + verb + " by resignation"
	}
	return who + verb + " · " + why
}

func (a *App) gameOver() {
	a.busy, a.thinking = false, false
	a.clearSel()
	r := a.G.Result
	moves := (len(a.hist) + 1) / 2
	movesWord := fmt.Sprintf("%d moves", moves)
	if moves == 1 {
		movesWord = "1 move"
	}
	o := a.ui.over
	o.face = "you"
	if a.isAI() {
		f := a.opp()
		o.face = f.ID
		you := a.yourSide()
		rec := a.sd.Stats.Per[a.G.Opp]
		if rec == nil {
			rec = &Rec{}
			a.sd.Stats.Per[a.G.Opp] = rec
		}
		if !a.G.Counted {
			switch {
			case r.Winner == 0:
				a.sd.Stats.Drawn++
				rec.D++
			case r.Winner == you:
				a.sd.Stats.Won++
				rec.W++
			default:
				a.sd.Stats.Lost++
				rec.L++
			}
			a.G.Counted = true
		}
		kind := "win"
		switch {
		case r.Winner == 0:
			o.title, o.kick, kind = "A Draw", "Even at the barrel", "draw"
			a.sfx.Draw()
		case r.Winner == you:
			o.title, kind = "You Win!", "lose"
			o.kick = map[string]string{"blocked": "Nowhere left to go", "captured": "Board cleared"}[r.Reason]
			if o.kick == "" {
				o.kick = "Victory"
			}
			a.sfx.Win()
			a.ui.confetti()
		default:
			o.title = f.Name + " Wins"
			o.kick = map[string]string{"blocked": "Blocked in", "captured": "Cleaned out"}[r.Reason]
			if o.kick == "" {
				o.kick = "Defeat"
			}
			a.sfx.Lose()
		}
		o.quote = "“" + pick(Quips[f.ID][kind]) + "” — " + f.Full
		o.text = fmt.Sprintf("%s after %s. Your record against %s: %d won, %d drawn, %d lost.", a.resultLine(), movesWord, f.Name, rec.W, rec.D, rec.L)
	} else if a.G.Mode == "online" {
		rec := &a.sd.Stats.Net
		you := a.yourSide()
		opp := a.faceOf(-you)
		if !a.G.Counted {
			switch {
			case r.Winner == 0:
				rec.D++
			case r.Winner == you:
				rec.W++
			default:
				rec.L++
			}
			a.G.Counted = true
		}
		name := a.G.Names[0]
		if you == Black {
			name = a.G.Names[1]
		}
		o.face, o.quote = opp, ""
		switch {
		case r.Winner == 0:
			o.title, o.kick = "A Draw", "Even across the wire"
			a.sfx.Draw()
		case r.Winner == you:
			o.title, o.kick, o.face = "You Win!", "Online game", a.S.Face
			a.sfx.Win()
			a.ui.confetti()
		default:
			o.title, o.kick = name+" Wins", "Online game"
			a.sfx.Lose()
		}
		o.text = fmt.Sprintf("%s after %s. Your online record: %d won, %d drawn, %d lost.", a.resultLine(), movesWord, rec.W, rec.D, rec.L)
	} else {
		if !a.G.Counted {
			a.sd.Stats.Pvp++
			a.G.Counted = true
		}
		o.kick, o.quote = "Game over", ""
		if r.Winner != 0 {
			o.title = a.playerName(r.Winner) + " Wins"
			o.face = a.faceOf(r.Winner)
			a.sfx.Win()
			a.ui.confetti()
		} else {
			o.title = "A Draw"
			a.sfx.Draw()
		}
		o.text = fmt.Sprintf("%s after %s.", a.resultLine(), movesWord)
	}
	if a.G.Mode != "online" {
		a.sd.Game = a.G
	}
	a.sd.write(true)
	a.ui.dirty()
	a.later(900*time.Millisecond, func() { a.ui.show("over") })
}

/* ---------------------------------------------------------------- commands */

func (a *App) newGame(setup Setup) {
	a.cancelThinking()
	a.gen++
	a.busy, a.thinking = false, false
	youB := setup.Color != "red"
	if setup.Color == "random" {
		youB = rand.Intn(2) == 0
	}
	forced := setup.Forced != "off"
	a.leaveOnline(true)
	a.G = &GameRec{FEN: StartFEN, Moves: []string{}, Mode: setup.Mode, Opp: setup.Opp, YouBlack: setup.Mode != "ai" || youB, Forced: &forced, Started: time.Now().UnixMilli()}
	if setup.Mode == "ai" {
		a.G.OppFull = folkByID(setup.Opp).Full
	} else {
		st := a.S
		a.G.Names = []string{cleanName(st.HotNames[0], "Player 1"), cleanName(st.HotNames[1], "Player 2")}
		a.G.Faces = []string{st.HotFaces[0], st.HotFaces[1]}
	}
	a.drawPly, a.view, a.viewPos = -99, -1, nil
	a.clearSel()
	a.hintPath, a.anims = nil, nil
	a.hidden, a.ghosts, a.lingering = map[int]bool{}, map[int]bool{}, map[int]int8{}
	a.rebuild()
	if a.G.Mode == "ai" {
		a.flipped = a.G.YouBlack
	} else {
		a.flipped = true
	}
	a.G.Flipped = a.flipped
	a.persist()
	a.ui.boardDirty()
	a.ui.hide("title")
	a.ui.hide("over")
	a.sfx.Start()
	if a.isAI() {
		a.later(500*time.Millisecond, func() { a.quip("greet") })
		if a.pos.Side == a.aiSide() {
			a.aiTurn()
		}
	}
}

func (a *App) resumeGame() {
	a.G = a.sd.Game
	a.rebuild()
	a.flipped = a.G.Flipped
	a.view, a.viewPos, a.busy = -1, nil, false
	a.clearSel()
	a.ui.boardDirty()
	a.ui.hide("title")
	if a.isAI() && a.G.Result == nil && a.pos.Side == a.aiSide() {
		a.aiTurn()
	}
}

func (a *App) undo() {
	if a.G == nil || a.G.Result != nil || len(a.hist) == 0 || a.G.Mode == "online" {
		return
	}
	if !a.live() {
		a.setView(len(a.hist))
	}
	a.cancelThinking()
	a.gen++
	a.busy, a.thinking = false, false
	a.clearSel()
	a.anims = nil
	a.hidden, a.lingering = map[int]bool{}, map[int]int8{}
	n := 1
	if a.isAI() {
		n = 2
		if a.pos.Side == a.aiSide() {
			n = 1
		}
		if len(a.hist)-n < 0 {
			n = len(a.hist)
		}
		if !a.G.YouBlack && len(a.hist)-n == 0 {
			n = len(a.hist) - 1 // keep the computer's opening move
		}
		if n <= 0 {
			return
		}
	}
	last := a.hist[len(a.hist)-n].m
	a.G.Moves = a.G.Moves[:len(a.G.Moves)-n]
	a.rebuild()
	// slide the piece back to where it started
	x0, y0 := a.sqXY(last.To())
	x1, y1 := a.sqXY(last.From)
	a.hidden[last.From] = true
	from := last.From
	a.anims = append(a.anims, &anim{piece: last.Piece, x0: x0, y0: y0, x1: x1, y1: y1, t0: time.Now(), dur: a.animDur(), done: func() { delete(a.hidden, from) }})
	a.sfx.Move()
	a.hintPath = nil
	if a.G.Mode == "pvp" && a.S.Turn == "on" {
		a.setFlip(a.pos.Side == Black)
	}
	a.persist()
	a.ui.dirty()
	if a.isAI() && a.pos.Side == a.aiSide() {
		a.aiTurn()
	}
}

func (a *App) hint() {
	if a.G == nil || !a.yourTurn() || !a.live() || a.busy || a.G.Mode == "online" {
		return
	}
	a.busy = true
	a.clearSel()
	a.ui.statusOverride = "Studying the board…"
	a.ui.dirty()
	my := a.gen
	a.think(AIOpts{Time: 1200 * time.Millisecond}, func(r AIResult) {
		a.busy = false
		a.ui.statusOverride = ""
		if my != a.gen || !r.OK {
			a.ui.dirty()
			return
		}
		m, ok := a.findLegal(r.Move)
		if !ok {
			a.ui.dirty()
			return
		}
		a.hintPath = append([]int{m.From}, m.Path...)
		a.sfx.Hint()
		a.ui.statusOverride = "Hint: " + r.Move
		a.ui.statusAlert = true
		a.ui.dirty()
	})
}

func (a *App) resign() {
	if a.G == nil || a.G.Result != nil || len(a.hist) == 0 {
		return
	}
	text := a.opp().Name + " will be credited with the win."
	switch a.G.Mode {
	case "pvp":
		text = fmt.Sprintf("%s resigns and %s wins the game.", a.playerName(a.pos.Side), a.playerName(-a.pos.Side))
	case "online":
		text = a.playerName(-a.yourSide()) + " will be credited with the win."
	}
	a.ui.ask("Resign?", text, "Resign", "Keep Playing", func(yes bool) {
		if !yes {
			return
		}
		a.cancelThinking()
		a.gen++
		a.busy = false
		loser := a.yourSide()
		if a.G.Mode == "pvp" {
			loser = a.pos.Side
		}
		if a.online() {
			a.net.peer.send(Msg{T: "resign", Game: a.net.game})
		}
		a.G.Result = &Result{Reason: "resignation", Winner: -loser}
		a.persist()
		a.gameOver()
	})
}

func (a *App) offerDraw() {
	if a.G == nil || a.G.Result != nil || !a.yourTurn() {
		return
	}
	if a.online() {
		a.drawPly = len(a.hist)
		a.net.offerOut = true
		a.net.peer.send(Msg{T: "offer", Game: a.net.game})
		a.busy = true
		a.clearSel()
		a.ui.statusOverride = "Waiting for " + a.playerName(-a.yourSide()) + " to answer…"
		a.ui.dirty()
		return
	}
	if a.G.Mode == "pvp" {
		a.ui.ask("Draw offered", fmt.Sprintf("%s offers a draw. %s, do you accept?", a.playerName(a.pos.Side), a.playerName(-a.pos.Side)), "Accept Draw", "Play On", func(yes bool) {
			if yes {
				a.G.Result = &Result{Reason: "agreement"}
				a.persist()
				a.gameOver()
			}
		})
		return
	}
	a.drawPly = len(a.hist)
	a.busy = true
	a.ui.statusOverride = a.opp().Name + " considers your offer…"
	a.ui.dirty()
	my := a.gen
	a.think(AIOpts{Time: 700 * time.Millisecond}, func(r AIResult) {
		a.busy = false
		a.ui.statusOverride = ""
		if my != a.gen {
			return
		}
		aiScore := -r.Score // r.Score is from your side (you're to move)
		accept := aiScore <= -70 || (aiScore <= 25 && len(a.hist) >= 60) || (a.G.Opp == "tucker" && aiScore <= 0)
		if accept {
			a.quip("accept")
			a.G.Result = &Result{Reason: "agreement"}
			a.persist()
			a.gameOver()
		} else {
			a.quip("decline")
			a.ui.dirty()
		}
	})
}

func (a *App) movesText() string {
	ds := time.UnixMilli(a.G.Started).Format("January 2, 2006")
	bName, rName := "Player one", "Player two"
	if len(a.G.Names) == 2 {
		bName, rName = a.G.Names[0], a.G.Names[1]
	}
	if a.G.Mode == "ai" {
		if a.G.YouBlack {
			bName, rName = "You", a.opp().Full
		} else {
			bName, rName = a.opp().Full, "You"
		}
	}
	var s strings.Builder
	fmt.Fprintf(&s, "Checkers at the Hollow Creek General Store, %s\r\nBlack: %s\r\nRed: %s\r\n", ds, bName, rName)
	if !a.forced() {
		s.WriteString("Jumping optional (house rules)\r\n")
	}
	s.WriteString("\r\n")
	line := ""
	for i, h := range a.hist {
		part := h.n
		if i%2 == 0 {
			part = fmt.Sprintf("%d. %s", i/2+1, h.n)
		}
		if len(line)+len(part)+1 > 72 {
			s.WriteString(strings.TrimSpace(line) + "\r\n")
			line = ""
		}
		line += " " + part
	}
	s.WriteString(strings.TrimSpace(line) + "\r\n")
	if a.G.Result != nil {
		rl := strings.Replace(strings.Replace(a.resultLine(), a.sideName(Black), "Black", 1), a.sideName(Red), "Red", 1)
		s.WriteString("\r\n" + rl + "\r\n")
	}
	return s.String()
}

func (a *App) synthUnlock() {}

/* ---------------------------------------------------------------- per-frame */

// stepAnims starts and finishes move animations on time (drawing just shows them).
func (a *App) stepAnims() {
	now := time.Now()
	var keep []*anim
	var done []func()
	for _, an := range a.anims {
		if now.Before(an.t0) {
			keep = append(keep, an)
			continue
		}
		if !an.started {
			an.started = true
			if an.onStart != nil {
				an.onStart()
			}
		}
		if now.Sub(an.t0) < an.dur {
			keep = append(keep, an)
		} else if an.done != nil {
			done = append(done, an.done)
		}
	}
	a.anims = keep
	for _, f := range done { // after the list is updated: they may add animations
		f()
	}
}

func (a *App) update() {
	a.pollAI()
	a.pollNet()
	a.stepAnims()
	now := time.Now()
	idle := len(a.anims) == 0
	var keep []*timerT
	var due []*timerT
	for _, t := range a.timers {
		if now.After(t.at) && (!t.idleOnly || idle) {
			due = append(due, t)
		} else {
			keep = append(keep, t)
		}
	}
	a.timers = keep
	for _, t := range due {
		t.fn()
	}
}

// rematch: a new game against the same opponent with the colours swapped.
func (a *App) rematch() {
	if a.online() {
		if a.net.role == "host" {
			a.startOnlineGame(!a.net.hostBlack)
		} else {
			a.net.peer.send(Msg{T: "rematch"})
			a.ui.hide("over")
			a.ui.banner("Rematch?", "Asked "+a.net.peerName+" for another game")
		}
		return
	}
	if a.G == nil {
		return
	}
	s := a.sd.Setup
	s.Mode, s.Opp = a.G.Mode, a.G.Opp
	if a.G.Mode == "ai" {
		if a.G.YouBlack {
			s.Color = "red"
		} else {
			s.Color = "black"
		}
	}
	s.Forced = "on"
	if !a.forced() {
		s.Forced = "off"
	}
	full := a.G.OppFull
	a.newGame(s)
	if a.G.Mode == "ai" && full != "" { // same opponent, same name
		a.G.OppFull = full
		a.persist()
	}
}
