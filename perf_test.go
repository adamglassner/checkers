//go:build headless

package main

import (
	"os"
	"runtime/pprof"
	"testing"
	"time"
)

// How long a full repaint of the panels takes at common window sizes.
func TestRepaintSpeed(t *testing.T) {
	saveDisabled = true
	loadFonts()
	a := newTestApp("Adam", "you")
	f := true
	a.G = &GameRec{FEN: StartFEN, Moves: []string{"11-15", "23-19", "8-11", "22-17"}, Mode: "ai", Opp: "doc", YouBlack: true, Forced: &f}
	a.rebuild()
	u := a.ui
	for _, sz := range [][2]int{{1920, 1080}, {2560, 1440}, {3840, 2160}} {
		u.layout(sz[0], sz[1])
		u.bgDirty, u.uiDirty = true, true
		u.repaint() // warm the caches
		for _, title := range []bool{false, true} {
			if title {
				u.show("title")
			} else {
				u.hide("title")
			}
			u.uiDirty = true
			u.repaint() // warm this screen's caches
			n := 5
			t0 := time.Now()
			for i := 0; i < n; i++ {
				u.uiDirty = true
				u.repaint()
			}
			t.Logf("%dx%d title=%v: %.1f ms per repaint", sz[0], sz[1], title, float64(time.Since(t0).Microseconds())/1000/float64(n))
		}
	}
}

func TestRepaint4K(t *testing.T) {
	saveDisabled = true
	loadFonts()
	a := newTestApp("Adam", "you")
	f := true
	a.G = &GameRec{FEN: StartFEN, Moves: []string{"11-15", "23-19"}, Mode: "ai", Opp: "doc", YouBlack: true, Forced: &f}
	a.rebuild()
	u := a.ui
	u.layout(3840, 2160)
	u.show("title")
	u.repaint()
	u.uiDirty = true
	u.repaint()
	f2, _ := os.Create(os.Getenv("PROF"))
	if f2 != nil {
		pprof.StartCPUProfile(f2)
	}
	t0 := time.Now()
	for i := 0; i < 10; i++ {
		u.uiDirty = true
		u.repaint()
	}
	if f2 != nil {
		pprof.StopCPUProfile()
		f2.Close()
	}
	t.Logf("4K title: %.1f ms", float64(time.Since(t0).Microseconds())/10000)
}
