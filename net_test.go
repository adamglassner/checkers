//go:build headless

// Online play over real TCP: a host and a guest in one process, each with
// its own game, playing whole games. Checks both boards stay identical, and
// covers rematches, draw offers, resigning, a bad move from the guest, and
// the guest leaving (the computer takes over).
//
//	go test -tags headless -run Online -v .
package main

import (
	"math/rand"
	"reflect"
	"testing"
	"time"
)

func newTestApp(name, face string) *App {
	sd := defaultSave()
	sd.Settings.Music, sd.Settings.Sfx = false, false
	sd.Settings.Anim = "fast"
	sd.Settings.Name, sd.Settings.Face = name, face
	a := NewApp(sd, NewSynth())
	NewUI(a)
	return a
}

// step runs both games for a while, letting whoever's turn it is make a random move.
func step(t *testing.T, apps []*App, until func() bool, autoplay bool, limit time.Duration) {
	t.Helper()
	end := time.Now().Add(limit)
	for time.Now().Before(end) {
		for _, a := range apps {
			a.update()
			if autoplay && a.online() && a.G.Result == nil && a.yourTurn() && !a.busy && len(a.anims) == 0 && len(a.legal) > 0 {
				a.playMove(a.legal[rand.Intn(len(a.legal))], 0)
			}
		}
		if until() {
			return
		}
		time.Sleep(2 * time.Millisecond)
	}
	for i, a := range apps {
		t.Logf("app %d: mode=%s res=%+v hist=%d busy=%v yourTurn=%v net=%v status=%q", i, a.G.Mode, a.G.Result, len(a.hist), a.busy, a.yourTurn(), a.net != nil, a.ui.statusOverride)
	}
	t.Fatalf("timed out")
}

func movesOf(a *App) []string { return append([]string{}, a.G.Moves...) }

func TestOnline(t *testing.T) {
	saveDisabled = true
	host := newTestApp("Adam", "you")
	guest := newTestApp("Beth", "hattie")
	host.sd.Setup.Color = "black"
	host.hostGame()
	if host.net == nil {
		t.Fatal("host didn't start")
	}
	guest.joinGame("127.0.0.1")
	apps := []*App{host, guest}
	step(t, apps, func() bool { return guest.online() && host.online() }, false, 5*time.Second)
	if !host.G.YouBlack || guest.G.YouBlack {
		t.Fatalf("colours: host black=%v guest black=%v", host.G.YouBlack, guest.G.YouBlack)
	}
	if guest.G.Names[0] != "Adam" || guest.G.Names[1] != "Beth" || guest.G.Faces[1] != "hattie" {
		t.Fatalf("names/faces wrong: %v %v", guest.G.Names, guest.G.Faces)
	}
	if guest.playerName(Black) != "Adam" || host.playerName(Red) != "Beth" {
		t.Fatalf("player names: %q %q", guest.playerName(Black), host.playerName(Red))
	}

	// Game 1: random moves to the end (or a resignation after 160 plies).
	step(t, apps, func() bool {
		if host.G.Result == nil && len(host.hist) >= 160 && host.yourTurn() && !host.busy {
			host.cancelThinking()
			host.G.Result = &Result{Reason: "resignation", Winner: Red}
			host.net.peer.send(Msg{T: "resign", Game: host.net.game})
			host.gameOver()
		}
		return host.G.Result != nil && guest.G.Result != nil && len(host.anims) == 0 && len(guest.anims) == 0
	}, true, 60*time.Second)
	if !reflect.DeepEqual(movesOf(host), movesOf(guest)) {
		t.Fatalf("boards differ:\nhost  %v\nguest %v", movesOf(host), movesOf(guest))
	}
	if !reflect.DeepEqual(*host.G.Result, *guest.G.Result) {
		t.Fatalf("results differ: %+v vs %+v", *host.G.Result, *guest.G.Result)
	}
	t.Logf("game 1: %d plies, %s (host sees %q)", len(host.hist), host.G.Result.Reason, host.resultLine())
	hr, gr := host.sd.Stats.Net, guest.sd.Stats.Net
	if hr.W+hr.D+hr.L != 1 || gr.W+gr.D+gr.L != 1 || hr.W != gr.L || hr.L != gr.W {
		t.Fatalf("online records don't match: host %+v guest %+v", hr, gr)
	}
	if host.sd.Game != nil || guest.sd.Game != nil {
		t.Fatalf("an online game was saved as the single-player game")
	}

	// Game 2: the guest asks for a rematch; colours swap.
	guest.rematch()
	step(t, apps, func() bool { return guest.G.Result == nil && host.G.Result == nil && len(guest.hist) == 0 }, false, 5*time.Second)
	if !guest.G.YouBlack || host.G.YouBlack {
		t.Fatalf("rematch didn't swap colours")
	}
	step(t, apps, func() bool { return len(host.hist) >= 10 && len(guest.hist) == len(host.hist) }, true, 20*time.Second)
	step(t, apps, func() bool { return len(host.anims) == 0 && len(guest.anims) == 0 }, false, 5*time.Second)
	// a bad move from the guest: the host puts them right
	guest.net.peer.send(Msg{T: "move", Ply: 999, N: "1-5", Game: guest.net.game})
	wait := time.Now().Add(400 * time.Millisecond)
	step(t, apps, func() bool { return time.Now().After(wait) }, false, 2*time.Second)
	if !reflect.DeepEqual(movesOf(host), movesOf(guest)) {
		t.Fatalf("not resynced after a bad move")
	}
	// draw offer, accepted
	off := host
	if !off.yourTurn() {
		off = guest
	}
	other := guest
	if off == guest {
		other = host
	}
	off.offerDraw()
	step(t, apps, func() bool { return other.ui.shown["confirm"] }, false, 5*time.Second)
	other.ui.hide("confirm")
	other.ui.conf.cb(true)
	step(t, apps, func() bool { return host.G.Result != nil && guest.G.Result != nil }, false, 5*time.Second)
	if host.G.Result.Reason != "agreement" || guest.G.Result.Reason != "agreement" {
		t.Fatalf("draw not agreed: %+v %+v", host.G.Result, guest.G.Result)
	}
	t.Logf("game 2: draw agreed after %d plies", len(host.hist))

	// Game 3: the host starts a rematch, then the guest leaves mid-game.
	host.rematch()
	step(t, apps, func() bool { return guest.G.Result == nil && len(guest.hist) == 0 && host.G.YouBlack }, false, 5*time.Second)
	step(t, apps, func() bool { return len(host.hist) >= 6 && len(guest.hist) == len(host.hist) }, true, 20*time.Second)
	guest.leaveOnline(true)
	step(t, []*App{host}, func() bool { return host.ui.shown["confirm"] }, false, 5*time.Second)
	host.ui.hide("confirm")
	host.ui.conf.cb(true) // finish against the computer
	if host.G.Mode != "ai" || host.net != nil {
		t.Fatalf("host didn't switch to the computer: mode %s", host.G.Mode)
	}
	plies := len(host.hist)
	// play the host's moves at random; the computer must answer
	end := time.Now().Add(30 * time.Second)
	for host.G.Result == nil && len(host.hist) < plies+12 && time.Now().Before(end) {
		host.update()
		if host.yourTurn() && !host.busy && len(host.anims) == 0 && len(host.legal) > 0 {
			host.playMove(host.legal[rand.Intn(len(host.legal))], 0)
		}
		time.Sleep(2 * time.Millisecond)
	}
	if len(host.hist) < plies+4 && host.G.Result == nil {
		t.Fatalf("the computer didn't take over (%d -> %d plies)", plies, len(host.hist))
	}
	t.Logf("game 3: guest left after %d plies; the computer played on to ply %d", plies, len(host.hist))

	// A second guest is refused while a game is being hosted with someone in it.
	host2 := newTestApp("Cal", "doc")
	host2.hostGame()
	g1, g2 := newTestApp("Dee", "opal"), newTestApp("Eli", "earl")
	g1.joinGame("127.0.0.1")
	step(t, []*App{host2, g1}, func() bool { return g1.online() }, false, 5*time.Second)
	g2.joinGame("127.0.0.1")
	step(t, []*App{host2, g1, g2}, func() bool { return g2.ui.shown["msg"] }, false, 5*time.Second)
	t.Logf("third player refused: %q", g2.ui.msgText)
	host2.leaveOnline(true)
	step(t, []*App{g1}, func() bool { return g1.ui.shown["confirm"] }, false, 5*time.Second)
}

func TestRandomNames(t *testing.T) {
	seen := map[string]map[string]bool{}
	for i := 0; i < 200; i++ {
		for _, f := range rollNames() {
			if seen[f.ID] == nil {
				seen[f.ID] = map[string]bool{}
			}
			seen[f.ID][f.Full] = true
		}
	}
	for _, b := range folkBases {
		if len(seen[b.ID]) < 4 {
			t.Errorf("%s only got names %v", b.ID, seen[b.ID])
		}
		for full := range seen[b.ID] {
			if g := folkNamed(b.ID, full); g.Full != full {
				t.Errorf("saved name %q doesn't come back (%q)", full, g.Full)
			}
		}
	}
	for _, id := range []string{"opal", "hattie"} { // the two women
		for full := range seen[id] {
			for _, man := range []string{"Earl", "Gus", "Tucker", "Amos"} {
				if full == man {
					t.Errorf("%s got a man's name %q", id, full)
				}
			}
		}
	}
	t.Logf("e.g. %v", rollNames())
}
