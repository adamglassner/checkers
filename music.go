// Checkers music and sound effects. Four original old-time tunes played live:
// fiddle, banjo, mandolin, harmonica, guitar, upright bass, washboard and a
// ragtime parlor piano, plus the sounds of checkers on a board.
// (c) 2026 Ironite
package main

import (
	"math"
	"math/rand"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

var noteNum = map[string]int{"C": 0, "C#": 1, "Db": 1, "D": 2, "D#": 3, "Eb": 3, "E": 4, "F": 5, "F#": 6, "Gb": 6, "G": 7, "G#": 8, "Ab": 8, "A": 9, "A#": 10, "Bb": 10, "B": 11}
var noteRe = regexp.MustCompile(`^([A-G][#b]?)(-?\d)$`)
var chordRe = regexp.MustCompile(`^([A-G][#b]?)(.*)$`)

func n2m(n string) int {
	m := noteRe.FindStringSubmatch(n)
	o, _ := strconv.Atoi(m[2])
	return 12*(o+1) + noteNum[m[1]]
}

type chordT struct {
	root int
	pcs  []int
}

func chord(sym string) chordT {
	m := chordRe.FindStringSubmatch(sym)
	root, q := noteNum[m[1]], m[2]
	var iv []int
	switch q {
	case "", "maj":
		iv = []int{0, 4, 7}
	case "m":
		iv = []int{0, 3, 7}
	case "7":
		iv = []int{0, 4, 7, 10}
	case "m7":
		iv = []int{0, 3, 7, 10}
	case "maj7":
		iv = []int{0, 4, 7, 11}
	case "6":
		iv = []int{0, 4, 7, 9}
	default:
		iv = []int{0, 4, 7}
	}
	c := chordT{root: root}
	for _, i := range iv {
		c.pcs = append(c.pcs, (root+i)%12)
	}
	return c
}

// pcAbove: the nearest midi note >= lo with pitch class pc.
func pcAbove(pc, lo int) int { return lo + ((pc-lo)%12+12)%12 }

type note struct {
	t   float64 // beats
	ins string
	m   int
	ms  []int
	d   float64 // beats
	v   float64
	up  bool
}

type section struct {
	ch  []string
	mel string
}

type formPart struct {
	sec   string
	ins   string
	shift int
}

type tuneDef struct {
	name  string
	bpm   float64
	beats float64
	feel  string
	swing float64
	intro []string
	secs  map[string]section
	form  []formPart
	tag   []string
}

type Tune struct {
	Name   string
	BPM    float64
	Swing  float64
	Events []note
	Len    float64
	Gap    float64
}

type melNote struct {
	n    string
	t, d float64
}

func parseMel(s string, at float64) []melNote {
	var ev []melNote
	t := at
	for _, tok := range strings.Fields(s) {
		if tok == "|" {
			continue
		}
		p := strings.SplitN(tok, ":", 2)
		d, _ := strconv.ParseFloat(p[1], 64)
		if p[0] != "R" {
			ev = append(ev, melNote{p[0], t, d})
		}
		t += d
	}
	return ev
}

func arrange(tn tuneDef) *Tune {
	var ev []note
	B := tn.beats
	rhythm := func(bars []string, at float64, soft bool) float64 {
		for bi, sym := range bars {
			c := chord(sym)
			t0 := at + float64(bi)*B
			v := 1.0
			if soft {
				v = .7
			}
			root, fifth := pcAbove(c.root, 40), pcAbove((c.root+7)%12, 40)
			var strum []int
			for _, pc := range c.pcs {
				strum = append(strum, pcAbove(pc, 52))
			}
			strum = append(strum, pcAbove(c.root, 52)+12)
			sort.Ints(strum)
			above := func(lo int) []int {
				var h []int
				for _, pc := range c.pcs {
					h = append(h, pcAbove(pc, lo))
				}
				sort.Ints(h)
				return h
			}
			switch tn.feel {
			case "waltz":
				bm := root
				if bi%2 == 1 {
					bm = fifth
				}
				ev = append(ev, note{t: t0, ins: "bass", m: bm, d: 1, v: v})
				ev = append(ev, note{t: t0 + 1, ins: "strum", ms: strum, v: .7 * v}, note{t: t0 + 2, ins: "strum", ms: strum, v: .6 * v})
				if !soft {
					pc := c.root
					if len(c.pcs) > 2 {
						pc = c.pcs[2]
					}
					ev = append(ev, note{t: t0, ins: "banjo", m: pcAbove(pc, 62), v: .35})
				}
			case "breakdown":
				ev = append(ev, note{t: t0, ins: "bass", m: root, d: .9, v: v}, note{t: t0 + 2, ins: "bass", m: fifth, d: .9, v: .85 * v})
				ev = append(ev, note{t: t0 + 1, ins: "chop", ms: strum, v: .8 * v}, note{t: t0 + 3, ins: "chop", ms: strum, v: .75 * v})
				hi := above(62)
				h2 := hi[0] + 12
				if len(hi) > 2 {
					h2 = hi[2]
				}
				roll := []int{hi[0], hi[1], h2, hi[0] + 12, hi[1], h2, hi[0] + 12, hi[1]}
				if !soft {
					for k, m := range roll {
						vv := .32
						if k%3 == 0 {
							vv = .5
						}
						ev = append(ev, note{t: t0 + float64(k)*.5, ins: "banjo", m: m, v: vv})
					}
				}
				for _, o := range []float64{1, 3} {
					ev = append(ev, note{t: t0 + o, ins: "wash", v: .9 * v})
				}
				for _, o := range []float64{.5, 1.5, 2.5, 3.5} {
					ev = append(ev, note{t: t0 + o, ins: "wash", v: .4 * v})
				}
			case "rag":
				lo := pcAbove(c.root, 31)
				ev = append(ev, note{t: t0, ins: "pLow", m: lo, d: .9, v: .9 * v}, note{t: t0, ins: "pLow", m: lo + 12, d: .9, v: .5 * v})
				mid := above(55)
				ev = append(ev, note{t: t0 + 1, ins: "pChord", ms: mid, d: .7, v: .5 * v})
				ev = append(ev, note{t: t0 + 2, ins: "pLow", m: pcAbove((c.root+7)%12, 31), d: .9, v: .75 * v})
				ev = append(ev, note{t: t0 + 3, ins: "pChord", ms: mid, d: .7, v: .45 * v})
				if !soft {
					ev = append(ev, note{t: t0 + 1, ins: "wash", v: .35}, note{t: t0 + 3, ins: "wash", v: .35})
				}
			default: // slow porch fingerpicking
				hi := above(59)
				h2 := hi[1]
				if len(hi) > 2 {
					h2 = hi[2]
				}
				for _, p := range [][3]float64{{0, float64(root), .9}, {1, float64(hi[0]), .5}, {1.5, float64(hi[1]), .45}, {2, float64(fifth), .75}, {2.5, float64(h2), .45}, {3, float64(hi[1]), .5}, {3.5, float64(hi[0]), .4}} {
					ev = append(ev, note{t: t0 + p[0], ins: "pick", m: int(p[1]), v: p[2] * v})
				}
				ev = append(ev, note{t: t0, ins: "bassSoft", m: pcAbove(c.root, 33), d: B * .9, v: .5 * v})
			}
		}
		return at + float64(len(bars))*B
	}
	t := rhythm(tn.intro, 0, true)
	for _, f := range tn.form {
		sec := tn.secs[f.sec]
		for _, e := range parseMel(sec.mel, t) {
			ev = append(ev, note{t: e.t, ins: f.ins, m: n2m(e.n) + f.shift, d: e.d, v: 1})
		}
		t = rhythm(sec.ch, t, false)
	}
	t = rhythm(tn.tag, t, true)
	ev = append(ev, note{t: t - B, ins: "banjo", m: pcAbove(chord(tn.tag[len(tn.tag)-1]).root, 67), v: .5})
	sort.SliceStable(ev, func(a, b int) bool { return ev[a].t < ev[b].t })
	return &Tune{Name: tn.name, BPM: tn.bpm, Swing: tn.swing, Events: ev, Len: t, Gap: 3}
}

var tuneDefs = []tuneDef{
	{name: "Potbelly Stove Waltz", bpm: 108, beats: 3, feel: "waltz",
		intro: []string{"G", "D", "G", "G"},
		secs: map[string]section{
			"A": {ch: strings.Fields("G G C G D D G G G G7 C Am D D7 G G"),
				mel: "D4:1 G4:1 B4:1 | D5:2 B4:1 | E5:2 C5:1 | B4:3 | A4:1 B4:1 C5:1 | D5:2 F#4:1 | G4:3 | G4:2 D4:1 | G4:1 B4:1 D5:1 | F5:2 D5:1 | E5:1.5 D5:.5 C5:1 | A4:3 | D5:1 C5:1 A4:1 | F#4:2 A4:1 | G4:3 | R:3"},
			"B": {ch: strings.Fields("C C G G A7 A7 D D7"),
				mel: "E5:2 G5:1 | E5:2 C5:1 | D5:1 B4:1 G4:1 | B4:3 | C#5:1 E5:1 A5:1 | G5:2 E5:1 | F#5:2 E5:.5 D5:.5 | C5:2 A4:1"},
		},
		form: []formPart{{"A", "fiddle", 0}, {"B", "mandolin", 0}, {"A", "harmonica", 0}}, tag: []string{"C", "D", "G"}},
	{name: "Kings Row Breakdown", bpm: 124, beats: 4, feel: "breakdown",
		intro: []string{"D", "A", "D", "D"},
		secs: map[string]section{
			"A": {ch: strings.Fields("D D G D D D A D"),
				mel: "F#5:.5 E5:.5 D5:.5 E5:.5 F#5:.5 A5:.5 F#5:.5 E5:.5 | D5:.5 E5:.5 F#5:.5 G5:.5 A5:1 A5:1 | B5:.5 A5:.5 G5:.5 A5:.5 B5:.5 D6:.5 B5:.5 A5:.5 | A5:.5 F#5:.5 E5:.5 D5:.5 F#5:1 A5:1 | F#5:.5 E5:.5 D5:.5 E5:.5 F#5:.5 A5:.5 B5:.5 A5:.5 | F#5:.5 A5:.5 D6:1 A5:1 F#5:1 | E5:.5 F#5:.5 G5:.5 F#5:.5 E5:.5 D5:.5 C#5:.5 E5:.5 | D5:2 D5:1 R:1"},
			"B": {ch: strings.Fields("G G D D A A D D"),
				mel: "B5:1 B5:.5 A5:.5 G5:.5 A5:.5 B5:1 | G5:.5 B5:.5 D6:1 B5:1 G5:1 | A5:1 A5:.5 G5:.5 F#5:.5 G5:.5 A5:1 | F#5:.5 A5:.5 D6:1 A5:1 F#5:1 | E5:.5 F#5:.5 G5:.5 A5:.5 B5:.5 A5:.5 G5:.5 E5:.5 | C#5:.5 E5:.5 A5:1 G5:1 E5:1 | F#5:.5 E5:.5 D5:.5 E5:.5 F#5:.5 G5:.5 A5:.5 F#5:.5 | D5:2 D5:1 R:1"},
		},
		form: []formPart{{"A", "fiddle", 0}, {"A", "banjoLead", -12}, {"B", "fiddle", 0}, {"B", "mandolin", 0}, {"A", "fiddle", 0}}, tag: []string{"A", "D"}},
	{name: "Penny Candy Rag", bpm: 92, beats: 4, feel: "rag", swing: .18,
		intro: []string{"C", "G7", "C", "G7"},
		secs: map[string]section{
			"A": {ch: strings.Fields("C C G7 G7 C A7 D7 G7"),
				mel: "E5:.5 G5:.5 C6:1 A5:.5 G5:.5 E5:1 | D#5:.5 E5:.5 G5:1 E5:1 C5:1 | D5:.5 F5:.5 B5:1 A5:.5 G5:.5 F5:1 | D5:.5 E5:.5 F5:.5 G5:.5 B4:2 | E5:.5 G5:.5 C6:1 E6:1 C6:1 | C#6:.5 A5:.5 G5:.5 E5:.5 C#5:2 | D5:.5 F#5:.5 A5:1 C6:1 A5:1 | G5:1 F5:.5 D5:.5 B4:2"},
			"B": {ch: strings.Fields("F F C C D7 G7 C C"),
				mel: "A5:.5 C6:.5 F6:1 C6:1 A5:1 | G#5:.5 A5:.5 C6:.5 A5:.5 F5:2 | G5:.5 E5:.5 C5:1 E5:.5 G5:.5 C6:1 | C6:2 R:2 | F#5:.5 A5:.5 D6:1 C6:1 A5:1 | B5:.5 G5:.5 F5:.5 D5:.5 B4:2 | C5:.5 E5:.5 G5:.5 C6:.5 E6:1 C6:1 | C6:3 R:1"},
		},
		form: []formPart{{"A", "piano", 0}, {"B", "piano", 0}, {"A", "mandolin", -12}, {"B", "piano", 0}}, tag: []string{"F", "G7", "C"}},
	{name: "Sweet Tea on the Porch", bpm: 72, beats: 4, feel: "pick",
		intro: []string{"F", "Bb", "C", "F"},
		secs: map[string]section{
			"A": {ch: strings.Fields("F Dm Bb C F Dm C F"),
				mel: "A4:2 C5:1 F5:1 | F5:1.5 E5:.5 D5:2 | D5:1 F5:1 Bb5:1.5 A5:.5 | G5:4 | A5:1.5 G5:.5 F5:1 C5:1 | D5:2 F5:1 A5:1 | G5:1 E5:1 C5:1.5 E5:.5 | F5:4"},
			"B": {ch: strings.Fields("Bb Bb F F Gm C F F"),
				mel: "D5:1 F5:1 Bb5:2 | A5:1 G5:1 F5:2 | C5:1 F5:1 A5:1.5 G5:.5 | F5:4 | Bb4:1 D5:1 G5:1.5 F5:.5 | E5:1 G5:1 C6:2 | A5:1.5 G5:.5 F5:1 C5:1 | F5:4"},
		},
		form: []formPart{{"A", "harmonica", 0}, {"B", "fiddle", -12}, {"A", "harmonica", 0}}, tag: []string{"Bb", "C", "F"}},
}

func melLen(s string) float64 {
	t := 0.0
	for _, tok := range strings.Fields(s) {
		if tok == "|" {
			continue
		}
		d, _ := strconv.ParseFloat(strings.SplitN(tok, ":", 2)[1], 64)
		t += d
	}
	return t
}

// playNote sends one arranged note to its instrument. d is in seconds.
func (s *Synth) playNote(e note, t, d float64) {
	v, m := e.v, float64(e.m)
	b := busMusic
	switch e.ins {
	case "bass":
		s.bass(t, m, .7*v, math.Min(d, .8), b)
	case "bassSoft":
		s.bass(t, m, .5*v, d, b)
	case "strum":
		ms := e.ms
		for i, n := range ms {
			s.pluck(t+float64(i)*.013, float64(n), .12*v, pluckOpt{decay: .9965, bright: .6, dur: 1.3, lp: 3200, body: 240, bus: b})
		}
	case "chop":
		for i, n := range e.ms {
			s.pluck(t+float64(i)*.008, float64(n), .1*v, pluckOpt{decay: .985, bright: .7, dur: .25, lp: 3600, body: 260, bus: b})
		}
	case "pick":
		s.pluck(t, m, .22*v, pluckOpt{decay: .998, bright: .45, dur: 2.2, lp: 3000, body: 230, bus: b})
	case "banjo":
		s.pluck(t, m, .2*v, pluckOpt{decay: .9935, bright: .95, dur: .7, lp: 7000, body: 700, bus: b})
	case "banjoLead":
		s.pluck(t, m+12, .3*v, pluckOpt{decay: .994, bright: .95, dur: .8, lp: 7000, body: 700, bus: b})
	case "mandolin": // tremolo on long notes
		n := 1
		if d > .45 {
			n = int(math.Max(2, math.Floor(d/.075)))
		}
		for k := 0; k < n; k++ {
			g := .1
			if k == 0 {
				g = .17
			}
			s.pluck(t+float64(k)*.075, m, g*v, pluckOpt{decay: .996, bright: .85, dur: .5, lp: 5500, body: 500, bus: b})
		}
	case "fiddle":
		s.fiddle(t, m, .9*v, d*.95, b)
	case "harmonica":
		s.harmonica(t, m, .9*v, d*.92, b)
	case "piano":
		s.piano(t, m, 1.5*v, d, b)
	case "pLow":
		s.piano(t, m, v, d, b)
	case "pChord":
		for i, n := range e.ms {
			s.piano(t+float64(i)*.005, float64(n), .55*v, d, b)
		}
	case "wash":
		s.noiseHit(t, .1*v, noiseOpt{kind: fBand, freq: 6500, q: 1.2, decay: .045, bus: b})
		s.noiseHit(t+.012, .05*v, noiseOpt{kind: fHigh, freq: 9000, decay: .03, bus: b})
	}
}

// ---------------------------------------------------------------- sequencer

func (s *Synth) InitMusic() {
	for _, d := range tuneDefs {
		s.tunes = append(s.tunes, arrange(d))
	}
	s.idx = rand.Intn(len(s.tunes))
	go s.seqLoop()
}

func (s *Synth) StartMusic() {
	s.seqMu.Lock()
	if s.running {
		s.seqMu.Unlock()
		return
	}
	s.running = true
	s.cur = s.tunes[s.idx]
	s.ev = 0
	s.t0 = s.Now() + .3
	s.seqMu.Unlock()
	s.mu.Lock()
	s.musicTau = .15
	if s.musicOn {
		s.musicTarget = math.Pow(s.vol/100, 1.6) * .95
	}
	s.mu.Unlock()
}

func (s *Synth) StopMusic() {
	s.seqMu.Lock()
	if s.running {
		s.running = false
		s.idx = (s.idx + 1) % len(s.tunes)
	}
	s.seqMu.Unlock()
	s.mu.Lock()
	s.musicTarget, s.musicTau = 0, .12
	// drop notes that haven't started yet
	keep := s.voices[:0]
	for _, v := range s.voices {
		if v.bus != busMusic || v.start <= s.rendered {
			keep = append(keep, v)
		}
	}
	s.voices = keep
	s.mu.Unlock()
}

func (s *Synth) seqLoop() {
	for {
		time.Sleep(70 * time.Millisecond)
		s.tick()
	}
}

func (s *Synth) tick() {
	s.seqMu.Lock()
	if !s.running {
		s.seqMu.Unlock()
		return
	}
	now := s.Now()
	ahead := now + .5
	type pending struct {
		e    note
		t, d float64
	}
	var todo []pending
	var announce string
	for guard := 0; guard < 500; guard++ {
		tn := s.cur
		spb := 60 / tn.BPM
		if s.ev >= len(tn.Events) {
			s.t0 += tn.Len*spb + tn.Gap
			s.idx = (s.idx + 1) % len(s.tunes)
			s.cur = s.tunes[s.idx]
			s.ev = 0
			announce = s.cur.Name
			continue
		}
		e := tn.Events[s.ev]
		bt := e.t
		if tn.Swing > 0 {
			if fr := bt - math.Floor(bt); math.Abs(fr-.5) < .01 {
				bt += tn.Swing * .5
			}
		}
		tt := s.t0 + bt*spb
		if tt > ahead {
			break
		}
		if tt >= now-.05 {
			d := e.d
			if d == 0 {
				d = 1
			}
			todo = append(todo, pending{e, tt + (rand.Float64()-.5)*.008, d * spb})
		}
		s.ev++
	}
	cb := s.OnTune
	s.seqMu.Unlock()
	for _, p := range todo {
		s.playNote(p.e, p.t, p.d)
	}
	if announce != "" && cb != nil {
		cb(announce)
	}
}

// ---------------------------------------------------------------- sound effects

type SFX struct {
	s    *Synth
	caps func() bool // bottle caps clink instead of thock
}

func (x *SFX) at() float64 { return x.s.Now() + .01 }

func (x *SFX) thock(t, v, pitch float64) {
	s := x.s
	b := busSFX
	if x.caps() {
		v *= 2
		s.noiseHit(t, .4*v, noiseOpt{kind: fBand, freq: 3800 * pitch, q: 6, decay: .03, bus: b})
		for i, f := range []float64{2310, 3570, 5120} {
			ff := f * pitch * (1 + rand.Float64()*.02)
			s.tone(t, wSine, ff, ff, .01, .05*v/float64(i+1), .18-float64(i)*.03, b)
		}
		s.noiseHit(t, .3*v, noiseOpt{kind: fLow, freq: 600, decay: .03, bus: b})
		return
	}
	v *= 3
	s.noiseHit(t, .6*v, noiseOpt{kind: fLow, freq: 420 * pitch, decay: .06, bus: b})
	s.noiseHit(t, .3*v, noiseOpt{kind: fBand, freq: 1100 * pitch, q: 3, decay: .05, bus: b})
	s.tone(t, wSine, 330*pitch, 260*pitch, .08, .16*v, .12, b)
}
func (x *SFX) slide(t, v float64) {
	x.s.noiseHit(t, .16*v, noiseOpt{kind: fBand, freq: 1400, q: .7, attack: .03, decay: .16, sweep: 700, bus: busSFX})
}
func (x *SFX) on() bool { x.s.mu.Lock(); defer x.s.mu.Unlock(); return x.s.sfxOn }

func (x *SFX) Pick() {
	if x.on() {
		x.thock(x.at(), .25, 1.6)
	}
}
func (x *SFX) Move() {
	if x.on() {
		t := x.at()
		x.slide(t, .6)
		x.thock(t+.05, 1, .95+rand.Float64()*.1)
	}
}
func (x *SFX) Hop(i int) {
	if x.on() {
		x.thock(x.at(), .95, 1+float64(i)*.06)
	}
}
func (x *SFX) Capture(n int) {
	if !x.on() {
		return
	}
	t := x.at()
	for k := 0; k < n; k++ {
		tt := t + .12 + float64(k)*.09
		x.slide(tt, .9)
		x.thock(tt+.06, .45, .75)
	}
}
func (x *SFX) banjo(t, m, v, dur float64) {
	x.s.pluck(t, m, v, pluckOpt{decay: .995, bright: .95, dur: dur, lp: 7000, body: 700, bus: busSFX})
}
func (x *SFX) Crown() {
	if !x.on() {
		return
	}
	t := x.at()
	x.thock(t, 1, 1.15)
	x.thock(t+.08, .8, 1.25)
	for i, m := range []float64{67, 71, 74, 79} {
		x.banjo(t+.15+float64(i)*.07, m, .22, 1)
	}
	x.s.bell(t+.45, 86, .55, busSFX, 1.6)
}
func (x *SFX) MustJump() {
	if x.on() {
		t := x.at()
		x.s.bell(t, 76, .45, busSFX, 1)
		x.s.bell(t+.1, 76, .35, busSFX, 1)
	}
}
func (x *SFX) Illegal() {
	if x.on() {
		x.s.noiseHit(x.at(), .7, noiseOpt{kind: fLow, freq: 220, decay: .07, bus: busSFX})
	}
}
func (x *SFX) Click() {
	if x.on() {
		x.s.noiseHit(x.at(), .45, noiseOpt{kind: fBand, freq: 3800, q: 2.5, decay: .018, bus: busSFX})
	}
}
func (x *SFX) Hint() {
	if x.on() {
		t := x.at()
		x.s.bell(t, 84, .45, busSFX, 1.2)
		x.s.bell(t+.09, 88, .35, busSFX, 1.2)
	}
}
func (x *SFX) Start() {
	if x.on() {
		t := x.at()
		for i, m := range []float64{62, 66, 69, 74} {
			x.banjo(t+float64(i)*.06, m, .22, 1)
		}
	}
}
func (x *SFX) Win() {
	if !x.on() {
		return
	}
	t := x.at()
	for i, m := range []float64{67, 71, 74, 79, 83, 86} {
		x.banjo(t+float64(i)*.08, m, .24, 1.2)
	}
	x.s.fiddle(t+.5, 79, 1, 1.2, busSFX)
	for i, m := range []float64{79, 83, 86, 91} {
		x.s.bell(t+.5+float64(i)*.1, m, .6, busSFX, 2)
	}
}
func (x *SFX) Lose() {
	if !x.on() {
		return
	}
	t := x.at()
	for i, m := range []float64{64, 60, 57, 52} {
		x.s.pluck(t+float64(i)*.22, m, .22, pluckOpt{decay: .997, bright: .6, dur: 2, lp: 3000, body: 400, bus: busSFX})
	}
}
func (x *SFX) Draw() {
	if x.on() {
		t := x.at()
		x.s.bell(t, 67, .6, busSFX, 1.8)
		x.s.bell(t+.18, 72, .6, busSFX, 1.8)
	}
}
