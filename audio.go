//go:build !headless

// Feeds the synthesizer's live mix to the sound card. (c) 2026 Ironite
package main

import (
	"encoding/binary"
	"math"
	"time"

	"github.com/hajimehoshi/ebiten/v2/audio"
)

// synthStream is an endless stereo float32 stream rendered by the synth.
type synthStream struct {
	s   *Synth
	buf []float32
}

func (r *synthStream) Read(p []byte) (int, error) {
	frames := len(p) / 8
	if frames == 0 {
		return 0, nil
	}
	if cap(r.buf) < frames*2 {
		r.buf = make([]float32, frames*2)
	}
	b := r.buf[:frames*2]
	r.s.Mix(b)
	for i, v := range b {
		binary.LittleEndian.PutUint32(p[i*4:], math.Float32bits(v))
	}
	return frames * 8, nil
}

var audioPlayer *audio.Player

func startAudio(s *Synth) {
	ctx := audio.NewContext(SR)
	pl, err := ctx.NewPlayerF32(&synthStream{s: s})
	if err != nil {
		return
	}
	pl.SetBufferSize(60 * time.Millisecond) // short, so effects land right on the move
	pl.Play()
	audioPlayer = pl
}
