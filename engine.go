// Checkers rules engine. (c) 2026 Ironite.
//
// American checkers (English draughts) on the 32 dark squares, numbered 1-32
// in the standard way: Black starts on 1-12 at the top of the diagram and
// moves first, Red starts on 21-32. Internally squares are 0..31.
// Pieces: 1 black man, 2 black king, -1 red man, -2 red king.
//
// Men move and capture diagonally forward, kings both ways, one square at a
// time. Captures are compulsory (the player may choose which) and a jump must
// be continued while it can be. A man that reaches the far row is crowned and
// his move ends there. You lose with no piece or no legal move. Draws:
// threefold repetition, or forty moves each with no capture and no man moved.
package main

import (
	"strconv"
	"strings"
)

const (
	Black = 1
	Red   = -1
	Man   = 1
	King  = 2
)

const StartFEN = "B:W21-32:B1-12"

var (
	dirRow  = [4]int{-1, -1, 1, 1} // up-left, up-right, down-left, down-right (diagram rows grow downward)
	dirCol  = [4]int{-1, 1, -1, 1}
	nb      [32][4]int // neighbour in each direction, -1 off the board
	jp      [32][4]int // landing square of a jump in each direction
	zob     [5][32]uint64
	zobSide uint64
)

func rowOf(s int) int { return s >> 2 }
func colOf(s int) int {
	r := s >> 2
	if r&1 == 1 {
		return 2 * (s & 3)
	}
	return 2*(s&3) + 1
}
func sqAt(r, c int) int {
	if r < 0 || r > 7 || c < 0 || c > 7 || (r+c)&1 == 0 {
		return -1
	}
	if r&1 == 1 {
		return r*4 + c/2
	}
	return r*4 + (c-1)/2
}

func init() {
	for s := 0; s < 32; s++ {
		for d := 0; d < 4; d++ {
			nb[s][d] = sqAt(rowOf(s)+dirRow[d], colOf(s)+dirCol[d])
			jp[s][d] = sqAt(rowOf(s)+2*dirRow[d], colOf(s)+2*dirCol[d])
		}
	}
	seed := uint64(0x2545f4914f6cdd1d)
	rnd := func() uint64 { seed ^= seed << 13; seed ^= seed >> 7; seed ^= seed << 17; return seed }
	for p := 0; p < 5; p++ {
		for s := 0; s < 32; s++ {
			zob[p][s] = rnd()
		}
	}
	zobSide = rnd()
}

func sign(v int8) int {
	if v > 0 {
		return 1
	}
	if v < 0 {
		return -1
	}
	return 0
}
func abs8(v int8) int8 {
	if v < 0 {
		return -v
	}
	return v
}

var allDirs = []int{0, 1, 2, 3}
var downDirs = []int{2, 3}
var upDirs = []int{0, 1}

func dirsFor(p int8) []int {
	if abs8(p) == King {
		return allDirs
	}
	if p > 0 {
		return downDirs
	}
	return upDirs
}
func crownRow(side int) int {
	if side == Black {
		return 7
	}
	return 0
}

// Move is one complete move: the landing squares of each step and the squares captured.
type Move struct {
	From  int
	Path  []int
	Caps  []int
	Piece int8
	Crown bool
}

func (m Move) To() int { return m.Path[len(m.Path)-1] }

func sameMove(a, b Move) bool {
	if a.From != b.From || len(a.Path) != len(b.Path) {
		return false
	}
	for i := range a.Path {
		if a.Path[i] != b.Path[i] {
			return false
		}
	}
	return true
}

type undoRec struct {
	m       Move
	quiet   int
	hash    uint64
	capList []int8
}

type Position struct {
	Board [32]int8
	Side  int
	Quiet int // plies since a capture or a man move
	Hash  uint64
	hist  []undoRec
	Keys  []uint64 // position keys after each ply, for repetition
}

func NewPosition(fen string) *Position {
	p := &Position{}
	p.Load(fen)
	return p
}

// Load reads "B:W21-32,K5:B1-12" (side to move, then each side's squares; K = king).
func (p *Position) Load(fen string) {
	p.Board = [32]int8{}
	parts := strings.Split(fen, ":")
	p.Side = Black
	if parts[0] == "W" || parts[0] == "R" {
		p.Side = Red
	}
	for _, part := range parts[1:] {
		if part == "" {
			continue
		}
		col := int8(Red)
		if part[0] == 'B' {
			col = Black
		}
		for _, tok := range strings.Split(part[1:], ",") {
			if tok == "" {
				continue
			}
			k := tok[0] == 'K'
			if k {
				tok = tok[1:]
			}
			ab := strings.SplitN(tok, "-", 2)
			a, _ := strconv.Atoi(ab[0])
			b := a
			if len(ab) == 2 {
				b, _ = strconv.Atoi(ab[1])
			}
			for n := a; n <= b; n++ {
				if n >= 1 && n <= 32 {
					if k {
						p.Board[n-1] = col * King
					} else {
						p.Board[n-1] = col * Man
					}
				}
			}
		}
	}
	p.Quiet = 0
	p.hist = nil
	p.computeHash()
	p.Keys = []uint64{p.Hash}
}

func (p *Position) FEN() string {
	list := func(side int8) string {
		var a []string
		for s := 0; s < 32; s++ {
			if p.Board[s]*side > 0 {
				if p.Board[s]*side == King {
					a = append(a, "K"+strconv.Itoa(s+1))
				} else {
					a = append(a, strconv.Itoa(s+1))
				}
			}
		}
		return strings.Join(a, ",")
	}
	side := "B"
	if p.Side == Red {
		side = "W"
	}
	return side + ":W" + list(Red) + ":B" + list(Black)
}

func (p *Position) Clone() *Position {
	q := &Position{Board: p.Board, Side: p.Side, Quiet: p.Quiet, Hash: p.Hash}
	q.Keys = append([]uint64(nil), p.Keys...)
	return q
}

func (p *Position) computeHash() {
	var h uint64
	for s := 0; s < 32; s++ {
		if p.Board[s] != 0 {
			h ^= zob[p.Board[s]+2][s]
		}
	}
	if p.Side == Red {
		h ^= zobSide
	}
	p.Hash = h
}

func (p *Position) Count(side int) int {
	n := 0
	for s := 0; s < 32; s++ {
		if int(p.Board[s])*side > 0 {
			n++
		}
	}
	return n
}

func (p *Position) Captures() []Move {
	var out []Move
	for s := 0; s < 32; s++ {
		pc := p.Board[s]
		if int(pc)*p.Side <= 0 {
			continue
		}
		p.Board[s] = 0
		p.jumps(s, s, pc, nil, nil, &out)
		p.Board[s] = pc
	}
	return out
}

func contains(a []int, v int) bool {
	for _, x := range a {
		if x == v {
			return true
		}
	}
	return false
}

func (p *Position) jumps(from, at int, pc int8, path, caps []int, out *[]Move) {
	found := false
	for _, d := range dirsFor(pc) {
		over, to := nb[at][d], jp[at][d]
		if to < 0 || p.Board[to] != 0 || int(p.Board[over])*int(pc) >= 0 || contains(caps, over) {
			continue
		}
		found = true
		np := append(append([]int(nil), path...), to)
		nc := append(append([]int(nil), caps...), over)
		if abs8(pc) == Man && rowOf(to) == crownRow(sign(pc)) {
			*out = append(*out, Move{From: from, Path: np, Caps: nc, Piece: pc, Crown: true})
		} else {
			p.jumps(from, to, pc, np, nc, out)
		}
	}
	if !found && len(path) > 0 {
		*out = append(*out, Move{From: from, Path: path, Caps: caps, Piece: pc})
	}
}

func (p *Position) Simple() []Move {
	var out []Move
	for s := 0; s < 32; s++ {
		pc := p.Board[s]
		if int(pc)*p.Side <= 0 {
			continue
		}
		for _, d := range dirsFor(pc) {
			to := nb[s][d]
			if to >= 0 && p.Board[to] == 0 {
				out = append(out, Move{From: s, Path: []int{to}, Piece: pc, Crown: abs8(pc) == Man && rowOf(to) == crownRow(p.Side)})
			}
		}
	}
	return out
}

// Legal returns the legal moves. With forced (the standard rule) captures are compulsory.
func (p *Position) Legal(forced bool) []Move {
	c := p.Captures()
	if len(c) > 0 && forced {
		return c
	}
	return append(c, p.Simple()...)
}

func (p *Position) Make(m Move) {
	to := m.To()
	rec := undoRec{m: m, quiet: p.Quiet, hash: p.Hash, capList: make([]int8, len(m.Caps))}
	for i, s := range m.Caps {
		rec.capList[i] = p.Board[s]
	}
	p.Board[m.From] = 0
	for _, s := range m.Caps {
		p.Board[s] = 0
	}
	if m.Crown {
		p.Board[to] = int8(sign(m.Piece)) * King
	} else {
		p.Board[to] = m.Piece
	}
	p.Side = -p.Side
	if len(m.Caps) > 0 || abs8(m.Piece) == Man {
		p.Quiet = 0
	} else {
		p.Quiet++
	}
	h := p.Hash ^ zob[m.Piece+2][m.From] ^ zob[p.Board[to]+2][to] ^ zobSide
	for i, s := range m.Caps {
		h ^= zob[rec.capList[i]+2][s]
	}
	p.Hash = h
	p.hist = append(p.hist, rec)
	p.Keys = append(p.Keys, h)
}

func (p *Position) Unmake() {
	n := len(p.hist)
	if n == 0 {
		return
	}
	rec := p.hist[n-1]
	p.hist = p.hist[:n-1]
	m := rec.m
	p.Board[m.To()] = 0
	p.Board[m.From] = m.Piece
	for i, s := range m.Caps {
		p.Board[s] = rec.capList[i]
	}
	p.Side = -p.Side
	p.Quiet = rec.quiet
	p.Hash = rec.hash
	p.Keys = p.Keys[:len(p.Keys)-1]
}

func (p *Position) Repetitions() int {
	n := 0
	for i := len(p.Keys) - 1; i >= 0; i -= 2 {
		if p.Keys[i] == p.Hash {
			n++
		}
	}
	return n
}

type Status struct {
	Over   bool
	Winner int // 0 = draw
	Reason string
}

func (p *Position) Status(forced bool) Status {
	if len(p.Legal(forced)) == 0 {
		r := "blocked"
		if p.Count(p.Side) == 0 {
			r = "captured"
		}
		return Status{Over: true, Winner: -p.Side, Reason: r}
	}
	if p.Repetitions() >= 3 {
		return Status{Over: true, Reason: "threefold repetition"}
	}
	if p.Quiet >= 80 {
		return Status{Over: true, Reason: "forty-move rule"}
	}
	return Status{}
}

// Notation: "11-15" for a move, "15x24x31" for a jump (every landing square).
func (p *Position) Notation(m Move) string {
	if len(m.Caps) == 0 {
		return strconv.Itoa(m.From+1) + "-" + strconv.Itoa(m.Path[0]+1)
	}
	parts := []string{strconv.Itoa(m.From + 1)}
	for _, s := range m.Path {
		parts = append(parts, strconv.Itoa(s+1))
	}
	return strings.Join(parts, "x")
}

func (p *Position) FindMove(n string, forced bool) (Move, bool) {
	for _, m := range p.Legal(forced) {
		if p.Notation(m) == n {
			return m, true
		}
	}
	return Move{}, false
}

func (p *Position) Perft(d int) int {
	ms := p.Legal(true)
	if d == 1 {
		return len(ms)
	}
	n := 0
	for _, m := range ms {
		p.Make(m)
		n += p.Perft(d - 1)
		p.Unmake()
	}
	return n
}
