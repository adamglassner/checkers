// Synthesizer and mixer: every instrument and sound effect in the game is
// generated here, live. Instruments are ported from the original Web Audio
// versions (same oscillators, envelopes and filters). (c) 2026 Ironite
package main

import (
	"math"
	"math/rand"
	"sync"
)

const SR = 44100

func mtof(m float64) float64 { return 440 * math.Pow(2, (m-69)/12) }

// ---------------------------------------------------------------- envelopes

// Param is an automation curve with Web Audio semantics (local time, seconds).
type Param struct {
	v0 float64
	ev []pev
}
type pev struct {
	kind      int // 0 set, 1 linear ramp, 2 exponential ramp, 3 target
	t, v, tau float64
}

func NewParam(v float64) *Param                  { return &Param{v0: v} }
func (p *Param) Set(v, t float64) *Param         { p.ev = append(p.ev, pev{0, t, v, 0}); return p }
func (p *Param) Lin(v, t float64) *Param         { p.ev = append(p.ev, pev{1, t, v, 0}); return p }
func (p *Param) Exp(v, t float64) *Param         { p.ev = append(p.ev, pev{2, t, v, 0}); return p }
func (p *Param) Target(v, t, tau float64) *Param { p.ev = append(p.ev, pev{3, t, v, tau}); return p }

// Render returns n per-sample values.
func (p *Param) Render(n int) []float64 {
	out := make([]float64, n)
	cur := p.v0
	prevT, prevV := 0.0, p.v0
	i := 0
	targeting := false
	var tS, vS, tgt, tau float64
	for s := 0; s < n; s++ {
		t := float64(s) / SR
		for i < len(p.ev) && p.ev[i].t <= t {
			e := p.ev[i]
			switch e.kind {
			case 0, 1, 2:
				cur = e.v
				targeting = false
			case 3:
				targeting, tS, vS, tgt, tau = true, e.t, cur, e.v, e.tau
			}
			prevT, prevV = e.t, cur
			i++
		}
		if i < len(p.ev) && (p.ev[i].kind == 1 || p.ev[i].kind == 2) {
			e := p.ev[i]
			span := e.t - prevT
			f := 1.0
			if span > 0 {
				f = (t - prevT) / span
			}
			if e.kind == 1 {
				cur = prevV + (e.v-prevV)*f
			} else {
				a, b := math.Max(prevV, 1e-6), math.Max(e.v, 1e-6)
				cur = a * math.Pow(b/a, f)
			}
		} else if targeting {
			if tau > 0 {
				cur = tgt + (vS-tgt)*math.Exp(-(t-tS)/tau)
			} else {
				cur = tgt
			}
		}
		out[s] = cur
	}
	return out
}

// env mirrors the shared attack/decay/sustain/release helper.
func env(a, peak, d, sus, rel, end float64) *Param {
	e := math.Max(a+.01, end)
	return NewParam(0).Set(0, 0).Lin(peak, a).Target(peak*sus, a, d).Set(peak*sus, e).Target(0, e, rel)
}

// ---------------------------------------------------------------- oscillators

const (
	wSine = iota
	wTri
	wSaw
	wSquare
)

func polyBlep(t, dt float64) float64 {
	if t < dt {
		t /= dt
		return t + t - t*t - 1
	}
	if t > 1-dt {
		t = (t - 1) / dt
		return t*t + t + t + 1
	}
	return 0
}

// osc renders a band-limited oscillator; freq is per-sample (Hz) or constant f when nil.
func osc(wave int, f float64, freq []float64, n int) []float64 {
	out := make([]float64, n)
	ph := 0.0
	for i := 0; i < n; i++ {
		fr := f
		if freq != nil {
			fr = freq[i]
		}
		dt := fr / SR
		var v float64
		switch wave {
		case wSine:
			v = math.Sin(2 * math.Pi * ph)
		case wTri:
			v = 1 - 4*math.Abs(ph-.5)
			v = -v
		case wSaw:
			v = 2*ph - 1 - polyBlep(ph, dt)
		case wSquare:
			if ph < .5 {
				v = 1
			} else {
				v = -1
			}
			v += polyBlep(ph, dt)
			v -= polyBlep(math.Mod(ph+.5, 1), dt)
		}
		out[i] = v
		ph += dt
		ph -= math.Floor(ph)
	}
	return out
}

// lfoAdd returns base + depth(t)*sin(2*pi*rate*t) per sample.
func vibrato(base float64, rate float64, depth []float64) []float64 {
	out := make([]float64, len(depth))
	for i := range out {
		out[i] = base + depth[i]*math.Sin(2*math.Pi*rate*float64(i)/SR)
	}
	return out
}

// ---------------------------------------------------------------- filters

const (
	fLow = iota
	fHigh
	fBand
	fPeak
)

type biquad struct {
	b0, b1, b2, a1, a2 float64
	x1, x2, y1, y2     float64
}

// set uses the Web Audio biquad formulas (lowpass/highpass Q in dB).
func (b *biquad) set(kind int, f, q, gainDB float64) {
	f = math.Max(10, math.Min(f, SR/2-100))
	w0 := 2 * math.Pi * f / SR
	cs, sn := math.Cos(w0), math.Sin(w0)
	var b0, b1, b2, a0, a1, a2 float64
	switch kind {
	case fLow, fHigh:
		alpha := sn / (2 * math.Pow(10, q/20))
		if kind == fLow {
			b0, b1, b2 = (1-cs)/2, 1-cs, (1-cs)/2
		} else {
			b0, b1, b2 = (1+cs)/2, -(1 + cs), (1+cs)/2
		}
		a0, a1, a2 = 1+alpha, -2*cs, 1-alpha
	case fBand:
		alpha := sn / (2 * math.Max(q, .0001))
		b0, b1, b2 = alpha, 0, -alpha
		a0, a1, a2 = 1+alpha, -2*cs, 1-alpha
	case fPeak:
		A := math.Pow(10, gainDB/40)
		alpha := sn / (2 * math.Max(q, .0001))
		b0, b1, b2 = 1+alpha*A, -2*cs, 1-alpha*A
		a0, a1, a2 = 1+alpha/A, -2*cs, 1-alpha/A
	}
	b.b0, b.b1, b.b2, b.a1, b.a2 = b0/a0, b1/a0, b2/a0, a1/a0, a2/a0
}
func (b *biquad) proc(x float64) float64 {
	y := b.b0*x + b.b1*b.x1 + b.b2*b.x2 - b.a1*b.y1 - b.a2*b.y2
	b.x2, b.x1, b.y2, b.y1 = b.x1, x, b.y1, y
	return y
}

// filt runs a biquad over buf in place; freq may vary per sample (nil = constant f).
func filt(buf []float64, kind int, f float64, freq []float64, q, gain float64) {
	var b biquad
	b.set(kind, f, q, gain)
	for i := range buf {
		if freq != nil && i%16 == 0 {
			b.set(kind, freq[i], q, gain)
		}
		buf[i] = b.proc(buf[i])
	}
}

func mulParam(buf []float64, g []float64) {
	for i := range buf {
		buf[i] *= g[i]
	}
}
func scale(buf []float64, g float64) {
	for i := range buf {
		buf[i] *= g
	}
}
func addInto(dst, src []float64, g float64) {
	for i := 0; i < len(dst) && i < len(src); i++ {
		dst[i] += src[i] * g
	}
}
func samples(sec float64) int { return int(math.Max(1, sec*SR)) }

// ---------------------------------------------------------------- mixer

const (
	busMusic = 0
	busSFX   = 1
)

type voice struct {
	buf   []float32
	start int64 // absolute sample index
	bus   int
}

type Synth struct {
	mu       sync.Mutex
	voices   []*voice
	rendered int64 // samples mixed so far

	musicOn, sfxOn bool
	vol            float64
	musicGain      float64 // smoothed
	musicTarget    float64
	musicTau       float64

	ksMu sync.Mutex
	ks   map[[4]float64][]float64

	rev     *reverb
	compEnv float64

	// music sequencer
	seqMu   sync.Mutex
	running bool
	tunes   []*Tune
	idx     int
	cur     *Tune
	ev      int
	t0      float64
	OnTune  func(name string)
	seqStop chan struct{}
}

func NewSynth() *Synth {
	s := &Synth{ks: map[[4]float64][]float64{}, rev: newReverb(), vol: 60, musicTau: .15}
	return s
}

// Now is the time (seconds) of the next sample to be mixed.
func (s *Synth) Now() float64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return float64(s.rendered) / SR
}

func (s *Synth) add(buf []float64, t float64, bus int) {
	if len(buf) == 0 {
		return
	}
	f := make([]float32, len(buf))
	for i, v := range buf {
		f[i] = float32(v)
	}
	s.mu.Lock()
	s.voices = append(s.voices, &voice{buf: f, start: int64(t * SR), bus: bus})
	s.mu.Unlock()
}

// SetLevels applies the music/sfx switches and volume (0-100).
func (s *Synth) SetLevels(music, sfx bool, vol float64) {
	s.mu.Lock()
	s.musicOn, s.sfxOn, s.vol = music, sfx, vol
	if s.running && music {
		s.musicTarget = math.Pow(vol/100, 1.6) * .95
	} else {
		s.musicTarget = 0
	}
	s.mu.Unlock()
}
func (s *Synth) sfxGain() float64 {
	if !s.sfxOn {
		return 0
	}
	return .85 * math.Max(.35, math.Pow(s.vol/100, .6))
}

// Mix renders n stereo frames into out (interleaved float32, -1..1).
func (s *Synth) Mix(out []float32) {
	n := len(out) / 2
	music := make([]float64, n)
	sfx := make([]float64, n)
	s.mu.Lock()
	start := s.rendered
	keep := s.voices[:0]
	for _, v := range s.voices {
		off := int(v.start - start) // where the voice starts within this block
		from := 0
		if off < 0 {
			from, off = -off, 0
		}
		dst := music
		if v.bus == busSFX {
			dst = sfx
		}
		for i, j := off, from; i < n && j < len(v.buf); i, j = i+1, j+1 {
			dst[i] += float64(v.buf[j])
		}
		if int64(len(v.buf))+v.start > start+int64(n) {
			keep = append(keep, v)
		}
	}
	for i := len(keep); i < len(s.voices); i++ {
		s.voices[i] = nil
	}
	s.voices = keep
	s.rendered += int64(n)
	target, tau, sg := s.musicTarget, s.musicTau, s.sfxGain()
	s.mu.Unlock()

	k := 1 - math.Exp(-1/(tau*SR))
	thr := math.Pow(10, -16.0/20)
	for i := 0; i < n; i++ {
		s.musicGain += (target - s.musicGain) * k
		m := music[i] * s.musicGain
		x := sfx[i] * sg
		// reverb: the music bus sends .3, effects .18 (both before the bus gains in the original graph)
		wl, wr := s.rev.proc(music[i]*.3*s.musicGain + sfx[i]*.18*sg)
		l := (m + x + wl) * .9
		r := (m + x + wr) * .9
		// gentle compressor (threshold -16 dB, ratio 3.5) then a soft limit
		pk := math.Max(math.Abs(l), math.Abs(r))
		if pk > s.compEnv {
			s.compEnv += (pk - s.compEnv) * .3
		} else {
			s.compEnv += (pk - s.compEnv) * (1 - math.Exp(-1/(.25*SR)))
		}
		g := 1.0
		if s.compEnv > thr {
			over := 20 * math.Log10(s.compEnv/thr)
			g = math.Pow(10, -(over-over/3.5)/20)
		}
		out[2*i] = float32(softClip(l * g * 1.4))
		out[2*i+1] = float32(softClip(r * g * 1.4))
	}
}

func softClip(x float64) float64 {
	if x > 1 {
		return 1
	}
	if x < -1 {
		return -1
	}
	return x * (1.5 - .5*x*x)
}

// ---------------------------------------------------------------- reverb (Freeverb-style)

type comb struct {
	buf   []float64
	i     int
	fb    float64
	damp  float64
	store float64
}

func (c *comb) proc(x float64) float64 {
	y := c.buf[c.i]
	c.store = y*(1-c.damp) + c.store*c.damp
	c.buf[c.i] = x + c.store*c.fb
	c.i = (c.i + 1) % len(c.buf)
	return y
}

type allpass struct {
	buf []float64
	i   int
}

func (a *allpass) proc(x float64) float64 {
	b := a.buf[a.i]
	y := -x + b
	a.buf[a.i] = x + b*.5
	a.i = (a.i + 1) % len(a.buf)
	return y
}

type reverb struct {
	cl, cr []*comb
	al, ar []*allpass
}

func newReverb() *reverb {
	r := &reverb{}
	for _, d := range []int{1116, 1188, 1277, 1356, 1422, 1491, 1557, 1617} {
		r.cl = append(r.cl, &comb{buf: make([]float64, d), fb: .86, damp: .3})
		r.cr = append(r.cr, &comb{buf: make([]float64, d+23), fb: .86, damp: .3})
	}
	for _, d := range []int{556, 441, 341, 225} {
		r.al = append(r.al, &allpass{buf: make([]float64, d)})
		r.ar = append(r.ar, &allpass{buf: make([]float64, d+23)})
	}
	return r
}
func (r *reverb) proc(x float64) (float64, float64) {
	x *= .12
	var l, rr float64
	for i := range r.cl {
		l += r.cl[i].proc(x)
		rr += r.cr[i].proc(x)
	}
	for i := range r.al {
		l = r.al[i].proc(l)
		rr = r.ar[i].proc(rr)
	}
	return l, rr
}

// ---------------------------------------------------------------- instruments
// Each takes a start time t (seconds, on the mixer clock) and a bus.

func (s *Synth) ksBuf(m, decay, bright, dur float64) []float64 {
	key := [4]float64{m, decay, bright, dur}
	s.ksMu.Lock()
	defer s.ksMu.Unlock()
	if b, ok := s.ks[key]; ok {
		return b
	}
	N := int(math.Max(2, math.Round(SR/mtof(m))))
	n := samples(dur)
	d := make([]float64, n)
	p, mx := 0.0, 0.0
	for i := 0; i < N && i < n; i++ {
		p += bright * ((rand.Float64()*2 - 1) - p)
		d[i] = p
		mx = math.Max(mx, math.Abs(p))
	}
	for i := 0; i < N && i < n; i++ {
		d[i] = d[i] / mx * .8
	}
	for i := N; i < n; i++ {
		j := i - N - 1
		if j < 0 {
			j = 0
		}
		d[i] = decay * .5 * (d[i-N] + d[j])
	}
	s.ks[key] = d
	return d
}

type pluckOpt struct {
	decay, bright, dur, lp, body float64
	bus                          int
}

func (s *Synth) pluck(t, m, v float64, o pluckOpt) {
	if o.decay == 0 {
		o.decay = .996
	}
	if o.bright == 0 {
		o.bright = .5
	}
	if o.dur == 0 {
		o.dur = 1.6
	}
	src := s.ksBuf(m, o.decay, o.bright, o.dur)
	buf := make([]float64, len(src))
	for i, x := range src {
		buf[i] = x * v
	}
	if o.lp > 0 {
		filt(buf, fLow, o.lp, nil, 1, 0)
	}
	if o.body > 0 {
		filt(buf, fPeak, o.body, nil, 1.2, 5)
	}
	s.add(buf, t, o.bus)
}

func (s *Synth) bass(t, m, v, dur float64, bus int) {
	f := mtof(m)
	n := samples(dur + .5)
	o1 := osc(wTri, f, nil, n)
	addInto(o1, osc(wSine, f*2, nil, n), .25)
	g := NewParam(0).Set(0, 0).Lin(v*.55, .012).Target(v*.25, .02, .25).Target(0, dur, .07).Render(n)
	mulParam(o1, g)
	lpf := NewParam(900).Set(900, 0).Exp(260, .4).Render(n)
	filt(o1, fLow, 900, lpf, 1, 0)
	s.add(o1, t, bus)
	s.pluck(t, m+12, v*.06, pluckOpt{decay: .99, bright: .3, dur: .4, lp: 1200, bus: bus})
}

type noiseOpt struct {
	kind          int
	freq, q       float64
	attack, decay float64
	sweep         float64
	bus           int
}

func (s *Synth) noiseHit(t, v float64, o noiseOpt) {
	if o.freq == 0 {
		o.freq = 3000
	}
	if o.q == 0 {
		o.q = .8
	}
	if o.attack == 0 {
		o.attack = .002
	}
	if o.decay == 0 {
		o.decay = .08
	}
	n := samples(o.attack + o.decay + .05)
	buf := make([]float64, n)
	for i := range buf {
		buf[i] = rand.Float64()*2 - 1
	}
	var fr []float64
	if o.sweep > 0 {
		fr = NewParam(o.freq).Set(o.freq, 0).Exp(o.sweep, o.attack+o.decay).Render(n)
	}
	filt(buf, o.kind, o.freq, fr, o.q, 0)
	g := NewParam(0).Set(0, 0).Lin(v, o.attack).Exp(.0005, o.attack+o.decay).Render(n)
	mulParam(buf, g)
	s.add(buf, t, o.bus)
}

func (s *Synth) bell(t, m, v float64, bus int, length float64) {
	if length == 0 {
		length = 1.6
	}
	f := mtof(m)
	n := samples(length + .1)
	out := make([]float64, n)
	for _, ra := range [][2]float64{{1, 1}, {2.76, .35}, {5.4, .18}, {8.93, .08}} {
		o := osc(wSine, f*ra[0], nil, n)
		g := NewParam(0).Set(0, 0).Lin(ra[1]*v*.3, .004).Exp(.0001, length/math.Sqrt(ra[0])).Render(n)
		mulParam(o, g)
		addInto(out, o, 1)
	}
	s.add(out, t, bus)
}

func (s *Synth) fiddle(t, m, v, dur float64, bus int) {
	f := mtof(m)
	n := samples(dur + .6)
	depth := NewParam(0).Set(0, 0).Lin(f*.007, math.Min(.35, dur*.6)).Render(n)
	fr1 := vibrato(f, 5.4, depth)
	fr2 := vibrato(f*1.003, 5.4, depth)
	buf := osc(wSaw, 0, fr1, n)
	addInto(buf, osc(wSaw, 0, fr2, n), 1)
	filt(buf, fLow, math.Min(3800, f*4), nil, 1, 0)
	filt(buf, fPeak, 2600, nil, 1.3, 4)
	filt(buf, fHigh, 200, nil, 1, 0)
	mulParam(buf, env(.08, v*.07, .2, .8, .1, dur).Render(n))
	s.add(buf, t, bus)
}

func (s *Synth) harmonica(t, m, v, dur float64, bus int) {
	f := mtof(m)
	n := samples(dur + .5)
	buf := osc(wSquare, f, nil, n)
	addInto(buf, osc(wSaw, f*1.005, nil, n), 1)
	filt(buf, fBand, math.Min(2400, f*2.2), nil, .6, 0)
	filt(buf, fLow, 3000, nil, 1, 0)
	for i := range buf {
		buf[i] *= .85 + .15*math.Sin(2*math.Pi*5.6*float64(i)/SR)
	}
	mulParam(buf, env(.05, v*.065, .2, .85, .08, dur).Render(n))
	s.add(buf, t, bus)
}

func (s *Synth) piano(t, m, v, dur float64, bus int) {
	f := mtof(m)
	n := samples(dur + 2.5)
	out := make([]float64, n)
	for _, p := range []struct {
		r, a float64
		w    int
	}{{1, 1, wTri}, {2, .35, wSine}, {3, .12, wSine}, {4.02, .05, wSine}} {
		o := osc(p.w, f*p.r, nil, n)
		g := NewParam(p.a).Set(p.a, 0).Exp(.001, 2.2/p.r+.3).Render(n)
		mulParam(o, g)
		addInto(out, o, 1)
	}
	g := NewParam(0).Set(0, 0).Lin(v*.16, .005).Target(v*.07, .01, .4).Target(0, dur, .25).Render(n)
	mulParam(out, g)
	filt(out, fLow, 2400+v*2000, nil, 1, 0)
	s.add(out, t, bus)
}

// tone: a single oscillator with a pitch glide and a gain from g0 decaying exponentially to ~0 at gEnd.
func (s *Synth) tone(t float64, wave int, f0, f1, glide, g0, gEnd float64, bus int) {
	n := samples(gEnd + .02)
	fr := NewParam(f0).Set(f0, 0).Exp(f1, glide).Render(n)
	o := osc(wave, 0, fr, n)
	g := NewParam(g0).Set(g0, 0).Exp(.0005, gEnd).Render(n)
	mulParam(o, g)
	s.add(o, t, bus)
}
