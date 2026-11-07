package main

import (
	"testing"
	"time"
)

// Move counts from the start position, as published for English draughts.
func TestPerft(t *testing.T) {
	want := []int{7, 49, 302, 1469, 7361, 36768, 179740, 845931}
	p := NewPosition(StartFEN)
	for d, n := range want {
		if got := p.Perft(d + 1); got != n {
			t.Fatalf("perft(%d) = %d, want %d", d+1, got, n)
		}
	}
	h := p.Hash
	p.computeHash()
	if h != p.Hash {
		t.Fatal("hash changed after make/unmake")
	}
}

func TestMultiJumpAndCrown(t *testing.T) {
	p := NewPosition("B:W6,7,15,23:B1")
	var got []string
	for _, m := range p.Legal(true) {
		got = append(got, p.Notation(m))
	}
	if len(got) != 1 || got[0] != "1x10x19x26" {
		t.Fatalf("got %v", got)
	}
	q := NewPosition("W:W10:B")
	ms := q.Legal(true)
	_ = ms
	r := NewPosition("W:W6:B1,3") // red man on 6 can't jump back; moves to 1/2? (1 occupied)
	for _, m := range r.Legal(true) {
		if r.Notation(m) == "6-2" && !m.Crown {
			t.Fatal("reaching row 0 must crown")
		}
	}
}

func TestFEN(t *testing.T) {
	p := NewPosition(StartFEN)
	if p.FEN() != "B:W21,22,23,24,25,26,27,28,29,30,31,32:B1,2,3,4,5,6,7,8,9,10,11,12" {
		t.Fatal(p.FEN())
	}
}

// The strongest player converts 2 kings vs 1 before the forty-move rule.
func TestEndgameConversion(t *testing.T) {
	for _, fen := range []string{"B:WK29:BK10,K14", "B:WK1:BK18,K19", "B:WK32:BK10,K11"} {
		p := NewPosition(fen)
		i := 0
		for ; i < 120; i++ {
			if p.Status(true).Over {
				break
			}
			d := 250 * time.Millisecond
			if p.Side == Red {
				d = 120 * time.Millisecond
			}
			r := Think(p, AIOpts{Time: d, Forced: true}, nil)
			m, ok := p.FindMove(r.Move, true)
			if !ok {
				t.Fatalf("illegal %s", r.Move)
			}
			p.Make(m)
		}
		st := p.Status(true)
		if !st.Over || st.Winner != Black {
			t.Fatalf("%s: %+v after %d plies", fen, st, i)
		}
	}
}

// Levels: the champion should beat the beginner.
func TestLevels(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	p := NewPosition(StartFEN)
	for i := 0; i < 300; i++ {
		if st := p.Status(true); st.Over {
			if st.Winner != Red {
				t.Fatalf("champion (Red) didn't win: %+v", st)
			}
			return
		}
		o := Levels["tucker"].AI
		if p.Side == Red {
			o = AIOpts{Time: 200 * time.Millisecond}
		}
		o.Forced = true
		r := Think(p, o, nil)
		m, ok := p.FindMove(r.Move, true)
		if !ok {
			t.Fatalf("illegal %s", r.Move)
		}
		p.Make(m)
	}
	t.Fatal("game did not finish")
}
