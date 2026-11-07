// Checkers computer opponent. (c) 2026 Ironite.
//
// Alpha-beta (PVS) with iterative deepening, a transposition table, forced-
// capture extensions, killer/history ordering, and an evaluation that knows
// about back-rank guards, the centre, runaway men and trading down when ahead.
// Weaker players search shallower, add noise and sometimes miss things.
package main

import (
	"math"
	"math/rand"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

var centre = map[int]bool{9: true, 10: true, 13: true, 14: true, 17: true, 18: true, 21: true, 22: true}
var kingSq [32]float64

func init() {
	for s := 0; s < 32; s++ {
		r, c := float64(rowOf(s)), float64(colOf(s))
		kingSq[s] = math.Round(14 - (math.Abs(3.5-r)+math.Abs(3.5-c))*3)
	}
}

func evaluate(p *Position) float64 {
	b := &p.Board
	var men, kings [2]int
	score := 0.0
	for s := 0; s < 32; s++ {
		pc := b[s]
		if pc == 0 {
			continue
		}
		i, sg := 0, 1.0
		if pc < 0 {
			i, sg = 1, -1
		}
		r := rowOf(s)
		if abs8(pc) == Man {
			men[i]++
			adv := r
			if pc < 0 {
				adv = 7 - r
			}
			score += sg * float64(adv*adv) * .9
			if centre[s] {
				score += sg * 5
			}
			if adv >= 4 { // runaway: nothing of the enemy ahead on the path to the crown row
				free := true
				step := 1
				end := 8
				if pc < 0 {
					step, end = -1, -1
				}
				for rr := r + step; rr != end && free; rr += step {
					for t := 0; t < 32; t++ {
						if rowOf(t) == rr && int(b[t])*int(pc) < 0 && iabs(colOf(t)-colOf(s)) <= iabs(rr-r) {
							free = false
							break
						}
					}
				}
				if free {
					score += sg * float64(12+adv*4)
				}
			}
		} else {
			kings[i]++
			score += sg * kingSq[s]
		}
	}
	total := men[0] + men[1] + kings[0] + kings[1]
	mat := [2]float64{float64(men[0]*100 + kings[0]*150), float64(men[1]*100 + kings[1]*150)}
	score += mat[0] - mat[1]
	if total > 10 { // back rank, while there's still a midgame to defend
		if b[0] == Man {
			score += 9
		}
		if b[2] == Man {
			score += 11
		}
		if b[1] == Man {
			score += 4
		}
		if b[3] == Man {
			score += 3
		}
		if b[31] == -Man {
			score -= 9
		}
		if b[29] == -Man {
			score -= 11
		}
		if b[30] == -Man {
			score -= 4
		}
		if b[28] == -Man {
			score -= 3
		}
	}
	diff := mat[0] - mat[1]
	if diff != 0 { // trade down when ahead
		score += math.Copysign(math.Min(80, math.Abs(diff)), diff) * float64(24-total) / 24
	}
	if total <= 10 && diff != 0 { // endgame: close in, squeeze the weaker side
		strong := 1
		if diff < 0 {
			strong = -1
		}
		dist, n := 0.0, 0
		for s := 0; s < 32; s++ {
			if int(b[s])*strong == King {
				for t := 0; t < 32; t++ {
					if int(b[t])*strong < 0 {
						dist += float64(imax(iabs(rowOf(s)-rowOf(t)), iabs(colOf(s)-colOf(t))))
						n++
					}
				}
			}
		}
		if n > 0 {
			score -= float64(strong) * (dist / float64(n)) * 4
		}
		mob := 0
		for s := 0; s < 32; s++ {
			pc := b[s]
			if int(pc)*strong >= 0 {
				continue
			}
			for _, d := range dirsFor(pc) {
				if t := nb[s][d]; t >= 0 && b[t] == 0 {
					mob++
				}
			}
		}
		score -= float64(strong*mob) * 4
		for _, dc := range []int{0, 4, 27, 31} { // double corners: squares 1,5 and 28,32
			if int(b[dc])*strong < 0 {
				score -= float64(strong) * 6
			}
		}
	}
	if p.Side == Black {
		return score
	}
	return -score
}

func iabs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
func imax(a, b int) int {
	if a > b {
		return a
	}
	return b
}

const (
	ttSize = 1 << 20
	winVal = 100000.0
)

type ttEntry struct {
	key   uint64
	score float64
	depth int8
	flag  uint8 // 0 exact, 1 lower bound, 2 upper bound
	move  int16
}

// AIOpts mirrors the per-opponent settings.
type AIOpts struct {
	Time     time.Duration // 0 = no time limit (use MaxDepth)
	MaxDepth int
	Noise    float64
	Blunder  float64
	Forced   bool
}

type AIResult struct {
	Move  string
	Score float64
	Depth int
	OK    bool
}

type searcher struct {
	tt       []ttEntry
	nodes    int
	deadline time.Time
	stop     *atomic.Bool
	stopped  bool
	killers  [128][2]int
	hist     [32 * 32]int
	forced   bool
	rootKeys map[uint64]bool
}

var (
	aiMu sync.Mutex
	aiTT []ttEntry
)

func mcode(m Move) int { return m.From*32 + m.To() }

func (s *searcher) order(moves []Move, ttm int, ply int) {
	type sm struct {
		s float64
		m Move
	}
	k := s.killers[imin(ply, 127)]
	arr := make([]sm, len(moves))
	for i, m := range moves {
		c := mcode(m)
		v := float64(s.hist[c])
		if c == ttm {
			v += 1e9
		} else if c == k[0] {
			v += 1e7
		} else if c == k[1] {
			v += 9e6
		}
		v += float64(len(m.Caps)) * 1e8
		if m.Crown {
			v += 5e7
		}
		arr[i] = sm{v, m}
	}
	sort.SliceStable(arr, func(a, b int) bool { return arr[a].s > arr[b].s })
	for i := range arr {
		moves[i] = arr[i].m
	}
}

// Win/loss scores are stored relative to the node, so a cached "win in 5"
// stays a win in 5 wherever it's found (otherwise the engine stops making progress).
func toTT(v float64, ply int) float64 {
	if v > winVal-1000 {
		return v + float64(ply)
	}
	if v < -winVal+1000 {
		return v - float64(ply)
	}
	return v
}
func fromTT(v float64, ply int) float64 {
	if v > winVal-1000 {
		return v - float64(ply)
	}
	if v < -winVal+1000 {
		return v + float64(ply)
	}
	return v
}

func imin(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (s *searcher) isRep(p *Position) bool {
	k, n := p.Hash, len(p.Keys)
	for i := n - 3; i >= 0; i -= 2 {
		if p.Keys[i] == k {
			return true
		}
	}
	return s.rootKeys[k]
}

func (s *searcher) search(p *Position, depth int, alpha, beta float64, ply int) float64 {
	s.nodes++
	if s.nodes&2047 == 0 && ((!s.deadline.IsZero() && time.Now().After(s.deadline)) || s.stop.Load()) {
		s.stopped = true
	}
	if s.stopped {
		return 0
	}
	if ply > 0 && (p.Quiet >= 80 || s.isRep(p)) {
		return 0
	}
	moves := p.Legal(s.forced)
	if len(moves) == 0 {
		return -winVal + float64(ply)
	}
	if len(moves) == 1 { // forced moves don't use up depth
		depth++
	}
	if depth <= 0 && len(moves[0].Caps) == 0 {
		return evaluate(p)
	}
	if depth <= 0 {
		depth = 1 // finish the capture sequence
	}
	if ply > 120 {
		return evaluate(p)
	}
	e := &s.tt[p.Hash&(ttSize-1)]
	ttm := -1
	if e.key == p.Hash {
		ttm = int(e.move)
		if ply > 0 && int(e.depth) >= depth {
			v := fromTT(e.score, ply)
			switch {
			case e.flag == 0:
				return v
			case e.flag == 1 && v >= beta:
				return v
			case e.flag == 2 && v <= alpha:
				return v
			}
		}
	}
	a0 := alpha
	best := math.Inf(-1)
	bestM := -1
	s.order(moves, ttm, ply)
	for i, m := range moves {
		p.Make(m)
		var sc float64
		if i == 0 {
			sc = -s.search(p, depth-1, -beta, -alpha, ply+1)
		} else {
			sc = -s.search(p, depth-1, -alpha-1, -alpha, ply+1)
			if sc > alpha && sc < beta {
				sc = -s.search(p, depth-1, -beta, -alpha, ply+1)
			}
		}
		p.Unmake()
		if s.stopped {
			return 0
		}
		if sc > best {
			best, bestM = sc, mcode(m)
		}
		if sc > alpha {
			alpha = sc
		}
		if alpha >= beta {
			if len(m.Caps) == 0 {
				c := mcode(m)
				s.hist[c] += depth * depth
				k := &s.killers[imin(ply, 127)]
				if k[0] != c {
					k[1], k[0] = k[0], c
				}
			}
			break
		}
	}
	flag := uint8(0)
	if best <= a0 {
		flag = 2
	} else if best >= beta {
		flag = 1
	}
	*e = ttEntry{key: p.Hash, score: toTT(best, ply), depth: int8(imin(127, depth)), flag: flag, move: int16(bestM)}
	return best
}

// Think picks a move for the side to move. stop may be set from another goroutine to abandon the search.
func Think(pos *Position, o AIOpts, stop *atomic.Bool) AIResult {
	aiMu.Lock()
	defer aiMu.Unlock()
	if aiTT == nil {
		aiTT = make([]ttEntry, ttSize)
	}
	p := pos.Clone()
	legal := p.Legal(o.Forced)
	if len(legal) == 0 {
		return AIResult{}
	}
	if len(legal) == 1 {
		return AIResult{Move: p.Notation(legal[0]), OK: true}
	}
	if stop == nil {
		stop = &atomic.Bool{}
	}
	s := &searcher{tt: aiTT, stop: stop, forced: o.Forced, rootKeys: map[uint64]bool{}}
	if o.Time > 0 {
		s.deadline = time.Now().Add(o.Time)
	} else {
		s.deadline = time.Now().Add(60 * time.Second)
	}
	maxDepth := o.MaxDepth
	if maxDepth == 0 {
		maxDepth = 60
	}
	for i := len(p.Keys) - 3; i >= 0; i -= 2 { // earlier game positions count as repetitions
		s.rootKeys[p.Keys[i]] = true
	}
	type scored struct {
		m Move
		s float64
	}
	sc := make([]scored, len(legal))
	for i, m := range legal {
		sc[i] = scored{m, 0}
	}
	bestDepth := 0
	for d := 1; d <= maxDepth; d++ {
		var res []scored
		alpha := math.Inf(-1)
		for _, it := range sc {
			p.Make(it.m)
			beta := -alpha
			if o.Noise > 0 { // full window so weaker players can choose among near-equal moves
				beta = winVal + 1
			}
			v := -s.search(p, d-1, -winVal-1, beta, 1)
			p.Unmake()
			if s.stopped {
				break
			}
			res = append(res, scored{it.m, v})
			if v > alpha {
				alpha = v
			}
		}
		if s.stopped {
			if len(res) > 0 && res[0].s >= sc[0].s {
				done := map[int]bool{}
				for _, r := range res {
					done[mcode(r.m)*64+len(r.m.Path)] = true
				}
				for _, x := range sc {
					if !done[mcode(x.m)*64+len(x.m.Path)] {
						res = append(res, x)
					}
				}
				sort.SliceStable(res, func(a, b int) bool { return res[a].s > res[b].s })
				sc = res
			}
			break
		}
		sort.SliceStable(res, func(a, b int) bool { return res[a].s > res[b].s })
		sc, bestDepth = res, d
		if math.Abs(sc[0].s) > winVal-200 {
			break // found a forced win or loss
		}
		if stop.Load() {
			break
		}
	}
	pick := sc[0]
	if o.Noise > 0 {
		bestI, bestV := 0, math.Inf(-1)
		for i, x := range sc {
			v := x.s + (rand.Float64()*2-1)*o.Noise
			if v > bestV {
				bestI, bestV = i, v
			}
		}
		pick = sc[bestI]
	}
	if o.Blunder > 0 && rand.Float64() < o.Blunder && len(sc) > 1 {
		pick = sc[1+rand.Intn(imin(3, len(sc)-1))]
	}
	if len(p.Keys) < 8 && o.Noise == 0 { // a little variety in the opening between equal moves
		var near []scored
		for _, x := range sc {
			if x.s >= sc[0].s-6 {
				near = append(near, x)
			}
		}
		pick = near[rand.Intn(len(near))]
	}
	return AIResult{Move: p.Notation(pick.m), Score: sc[0].s, Depth: bestDepth, OK: !stop.Load()}
}
