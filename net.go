// Online play, the same way as the other Ironite games: one player hosts,
// a friend joins by IP address. The host's game is the referee: it checks
// every move the guest sends and keeps both boards in step. Messages are
// JSON, one per line, over TCP port 47020.
// (c) 2026 Ironite
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"math/rand"
	"net"
	"reflect"
	"strings"
	"time"
)

const (
	netPort  = 47020
	protoVer = 1
)

// Msg is everything that goes over the wire.
//
//	hello   guest -> host  name, face, ver
//	reject  host  -> guest why
//	state   host  -> guest the whole game: moves, colours, rules, host's name/face (start, rematch, resync)
//	move    both           ply (index of this move), n (its notation)
//	offer   both           a draw offer; answer {yes}
//	resign  both
//	rematch guest -> host  asks for a new game (colours swap)
//	bye     both           leaving
type Msg struct {
	T        string   `json:"t"`
	Ver      int      `json:"ver,omitempty"`
	Name     string   `json:"name,omitempty"`
	Face     string   `json:"face,omitempty"`
	Why      string   `json:"why,omitempty"`
	Moves    []string `json:"moves,omitempty"`
	YouBlack bool     `json:"youBlack,omitempty"`
	Forced   bool     `json:"forced,omitempty"`
	Game     int      `json:"game,omitempty"`
	Ply      int      `json:"ply,omitempty"`
	N        string   `json:"n,omitempty"`
	Yes      bool     `json:"yes,omitempty"`
}

// netConn is one connection with a buffered writer goroutine.
type netConn struct {
	c      net.Conn
	out    chan []byte
	closed bool
}

func newNetConn(c net.Conn) *netConn {
	k := &netConn{c: c, out: make(chan []byte, 256)}
	go func() {
		for b := range k.out {
			_ = c.SetWriteDeadline(time.Now().Add(15 * time.Second))
			if _, err := c.Write(b); err != nil {
				_ = c.Close()
				for range k.out {
				}
				return
			}
		}
		_ = c.Close()
	}()
	return k
}

func (k *netConn) send(m Msg) {
	if k == nil || k.closed {
		return
	}
	b, err := json.Marshal(m)
	if err != nil {
		return
	}
	select {
	case k.out <- append(b, '\n'):
	default: // hopelessly behind: give up on this connection
		k.close()
	}
}

func (k *netConn) close() {
	if k != nil && !k.closed {
		k.closed = true
		close(k.out)
	}
}

type netEvent struct {
	kind string // hello, msg, left, connected, failed
	msg  Msg
	conn *netConn
	err  error
	dec  *json.Decoder
}

// Net is the online session (nil when not playing online).
type Net struct {
	role               string // "host" or "guest"
	ln                 net.Listener
	peer               *netConn
	peerName, peerFace string
	addrs              []string
	addr               string // the address a guest is joining
	events             chan netEvent
	hostBlack          bool // host's colour this game
	game               int  // game number, so stale moves from an old game are ignored
	offerOut           bool // we offered a draw and are waiting
	gone               bool
}

func (a *App) setupName() string { return cleanName(a.S.Name, "Stranger") }

func cleanName(s, def string) string {
	s = strings.TrimSpace(s)
	if r := []rune(s); len(r) > 14 {
		s = string(r[:14])
	}
	if s == "" {
		return def
	}
	return s
}

func localAddrs() []string {
	var out []string
	ifs, _ := net.InterfaceAddrs()
	for _, ad := range ifs {
		if ip, ok := ad.(*net.IPNet); ok && ip.IP.To4() != nil && !ip.IP.IsLoopback() && !ip.IP.IsLinkLocalUnicast() {
			out = append(out, ip.IP.String())
		}
	}
	if len(out) > 3 {
		out = out[:3]
	}
	return out
}

/* ---------------------------------------------------------------- hosting */

func (a *App) hostGame() {
	a.leaveOnline(false)
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", netPort))
	if err != nil {
		a.ui.message("Couldn't host a game", fmt.Sprintf("Checkers couldn't open port %d on this computer. Is another copy of Checkers already hosting?\n\n%v", netPort, err))
		return
	}
	n := &Net{role: "host", ln: ln, addrs: localAddrs(), events: make(chan netEvent, 64)}
	a.net = n
	a.sd.write(false)
	a.ui.show("lobby")
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			go func() {
				_ = c.SetReadDeadline(time.Now().Add(10 * time.Second))
				dec := json.NewDecoder(bufio.NewReader(c))
				var m Msg
				k := newNetConn(c)
				if err := dec.Decode(&m); err != nil || m.T != "hello" {
					k.close()
					return
				}
				_ = c.SetReadDeadline(time.Time{})
				n.events <- netEvent{kind: "hello", msg: m, conn: k, dec: dec}
			}()
		}
	}()
}

// readLoop forwards a connection's messages to the game until it closes.
func (n *Net) readLoop(k *netConn, dec *json.Decoder) {
	for {
		var m Msg
		if err := dec.Decode(&m); err != nil {
			break
		}
		n.events <- netEvent{kind: "msg", msg: m, conn: k}
		if m.T == "bye" {
			break
		}
	}
	n.events <- netEvent{kind: "left", conn: k}
}

/* ---------------------------------------------------------------- joining */

func (a *App) joinGame(addr string) {
	a.leaveOnline(false)
	addr = strings.TrimSpace(addr)
	if addr == "" {
		a.ui.message("Join a game", "Type the address your friend's Checkers shows when they host, like 192.168.1.20.")
		return
	}
	a.S.Host = addr
	a.sd.write(false)
	dial := addr
	if _, _, err := net.SplitHostPort(addr); err != nil {
		dial = net.JoinHostPort(addr, fmt.Sprint(netPort))
	}
	n := &Net{role: "guest", addr: addr, events: make(chan netEvent, 64)}
	a.net = n
	a.ui.show("joining")
	hello := Msg{T: "hello", Ver: protoVer, Name: a.setupName(), Face: a.S.Face}
	go func() {
		c, err := net.DialTimeout("tcp", dial, 8*time.Second)
		if err != nil {
			n.events <- netEvent{kind: "failed", err: err}
			return
		}
		k := newNetConn(c)
		k.send(hello)
		n.events <- netEvent{kind: "connected", conn: k}
		n.readLoop(k, json.NewDecoder(bufio.NewReader(c)))
	}()
}

/* ---------------------------------------------------------------- the game */

// startOnlineGame (host) starts a new game with the guest and tells them about it.
func (a *App) startOnlineGame(hostBlack bool) {
	n := a.net
	n.hostBlack = hostBlack
	n.game++
	n.offerOut = false
	forced := a.sd.Setup.Forced != "off"
	a.beginOnline(hostBlack, forced, n.peerName, n.peerFace, nil)
	n.peer.send(Msg{T: "state", YouBlack: !hostBlack, Forced: forced, Name: a.setupName(), Face: a.S.Face, Game: n.game})
}

// beginOnline sets up the board for an online game from this player's point of view.
func (a *App) beginOnline(youBlack, forced bool, oppName, oppFace string, moves []string) {
	a.cancelThinking()
	a.gen++
	a.busy, a.thinking = false, false
	f := forced
	names := []string{a.setupName(), oppName}
	faces := []string{a.S.Face, oppFace}
	if !youBlack {
		names[0], names[1] = names[1], names[0]
		faces[0], faces[1] = faces[1], faces[0]
	}
	if moves == nil {
		moves = []string{}
	}
	a.G = &GameRec{FEN: StartFEN, Moves: moves, Mode: "online", YouBlack: youBlack, Forced: &f, Started: time.Now().UnixMilli(), Names: names, Faces: faces, Online: a.net.role}
	a.drawPly, a.view, a.viewPos = -99, -1, nil
	a.clearSel()
	a.hintPath, a.anims = nil, nil
	a.hidden, a.ghosts, a.lingering = map[int]bool{}, map[int]bool{}, map[int]int8{}
	a.rebuild()
	a.flipped = youBlack
	a.G.Flipped = a.flipped
	a.ui.boardDirty()
	for _, id := range []string{"title", "over", "lobby", "joining", "confirm"} {
		a.ui.hide(id)
	}
	if len(moves) == 0 {
		a.sfx.Start()
		a.ui.banner("You play "+a.sideName(a.yourSide()), map[bool]string{true: "You move first", false: oppName + " moves first"}[youBlack])
	}
}

func (a *App) online() bool { return a.net != nil && a.G != nil && a.G.Mode == "online" }

// sendMove tells the other player about a move we just made.
func (a *App) sendMove(ply int, n string) {
	if a.net != nil && a.net.peer != nil {
		a.net.peer.send(Msg{T: "move", Ply: ply, N: n, Game: a.net.game})
	}
}

// sendState (host) re-sends the whole game, to fix a guest that got out of step.
func (a *App) sendState() {
	n := a.net
	if n == nil || n.role != "host" || a.G == nil {
		return
	}
	n.peer.send(Msg{T: "state", YouBlack: !n.hostBlack, Forced: a.forced(), Name: a.setupName(), Face: a.S.Face, Moves: a.G.Moves, Game: n.game})
}

func (a *App) pollNet() {
	n := a.net
	if n == nil {
		return
	}
	for {
		select {
		case ev := <-n.events:
			if a.net != n { // an old session
				continue
			}
			a.netEvent(ev)
		default:
			return
		}
	}
}

func (a *App) netEvent(ev netEvent) {
	n := a.net
	switch ev.kind {
	case "hello": // host: a friend arrived
		m := ev.msg
		switch {
		case n.peer != nil:
			ev.conn.send(Msg{T: "reject", Why: "That game already has two players."})
			ev.conn.close()
			return
		case m.Ver != protoVer:
			ev.conn.send(Msg{T: "reject", Why: "That game is running a different version of Checkers. Update both copies to the same version and try again."})
			ev.conn.close()
			return
		}
		n.peer = ev.conn
		n.peerName = cleanName(m.Name, "Friend")
		n.peerFace = m.Face
		if !validFace(n.peerFace) {
			n.peerFace = "p2"
		}
		go n.readLoop(ev.conn, ev.dec)
		hostBlack := a.sd.Setup.Color != "red"
		if a.sd.Setup.Color == "random" {
			hostBlack = rand.Intn(2) == 0
		}
		a.startOnlineGame(hostBlack)
		a.sfx.Start()
	case "connected": // guest: waiting for the host's "state"
		n.peer = ev.conn
	case "failed":
		a.net = nil
		a.ui.hide("joining")
		a.ui.message("Couldn't join", fmt.Sprintf("Checkers couldn't reach a game at %s.\n\nCheck the address, and that your friend has clicked Host a Game. Both computers need to be on the same network (or the host's port %d forwarded to their computer).", n.addr, netPort))
	case "left":
		if ev.conn != n.peer {
			return
		}
		a.peerLeft()
	case "msg":
		if ev.conn != n.peer {
			return
		}
		a.netMsg(ev.msg)
	}
}

func (a *App) netMsg(m Msg) {
	n := a.net
	switch m.T {
	case "reject":
		a.leaveOnline(false)
		a.ui.hide("joining")
		a.ui.message("Couldn't join", m.Why)
	case "state": // guest: a new game, or a full resync
		if n.role != "guest" {
			return
		}
		n.peerName, n.peerFace = cleanName(m.Name, "Host"), m.Face
		if !validFace(n.peerFace) {
			n.peerFace = "p2"
		}
		fresh := m.Game != n.game || a.G == nil || a.G.Mode != "online"
		n.game = m.Game
		if fresh {
			n.offerOut = false
			a.beginOnline(m.YouBlack, m.Forced, n.peerName, n.peerFace, m.Moves)
		} else if !reflect.DeepEqual(m.Moves, a.G.Moves) {
			// the same game, out of step: take the host's moves, keep everything else
			busy, status, offer := a.busy, a.ui.statusOverride, n.offerOut
			a.beginOnline(m.YouBlack, m.Forced, n.peerName, n.peerFace, m.Moves)
			a.busy, a.ui.statusOverride, n.offerOut = busy, status, offer
		}
		if a.G.Result == nil {
			if st := a.pos.Status(a.forced()); st.Over {
				a.G.Result = &Result{Reason: st.Reason, Winner: st.Winner}
				a.gameOver()
			}
		}
	case "move":
		if m.Game != n.game || a.G == nil || a.G.Mode != "online" {
			return
		}
		a.whenIdle(func() { a.remoteMove(m.Ply, m.N) })
	case "offer":
		if a.G == nil || a.G.Result != nil {
			return
		}
		a.ui.ask("Draw offered", n.peerName+" offers a draw. Do you accept?", "Accept Draw", "Play On", func(yes bool) {
			if a.net != n || a.G == nil || a.G.Result != nil {
				return
			}
			n.peer.send(Msg{T: "answer", Yes: yes, Game: n.game})
			if yes {
				a.G.Result = &Result{Reason: "agreement"}
				a.gameOver()
			}
		})
	case "answer":
		if !n.offerOut || a.G == nil || a.G.Result != nil {
			return
		}
		n.offerOut = false
		a.busy = false
		a.ui.statusOverride = ""
		if m.Yes {
			a.G.Result = &Result{Reason: "agreement"}
			a.gameOver()
		} else {
			a.ui.banner(n.peerName+" plays on", "Your draw offer was declined")
			a.ui.dirty()
		}
	case "resign":
		if a.G == nil || a.G.Result != nil || m.Game != n.game {
			return
		}
		a.cancelThinking()
		a.busy = false
		a.G.Result = &Result{Reason: "resignation", Winner: a.yourSide()}
		a.gameOver()
	case "rematch":
		if n.role == "host" && a.G != nil && a.G.Result != nil {
			a.ui.banner(n.peerName+" wants a rematch", "Colours swap")
			a.startOnlineGame(!n.hostBlack)
		}
	case "bye":
		a.peerLeft()
	}
}

// remoteMove plays the other player's move, or (host) corrects them if it doesn't fit.
func (a *App) remoteMove(ply int, notation string) {
	if a.G == nil || a.G.Mode != "online" || a.G.Result != nil {
		return
	}
	if ply != len(a.hist) || a.pos.Side == a.yourSide() {
		if a.net.role == "host" {
			a.sendState()
		}
		return
	}
	m, ok := a.findLegal(notation)
	if !ok {
		if a.net.role == "host" {
			a.sendState()
		}
		return
	}
	a.ui.statusOverride = ""
	a.playMove(m, 0)
}

func (a *App) peerLeft() {
	n := a.net
	if n == nil || n.gone {
		return
	}
	n.gone = true
	name := n.peerName
	if name == "" {
		name = "Your friend"
	}
	a.leaveOnline(false)
	if a.G == nil || a.G.Mode != "online" {
		if a.ui.shown["joining"] {
			a.ui.hide("joining")
			a.ui.message("Couldn't join", "The host closed the connection.")
		}
		return
	}
	if a.G.Result != nil {
		a.ui.banner(name+" left", "")
		a.ui.dirty()
		return
	}
	a.busy = false
	a.ui.statusOverride = ""
	a.ui.ask(name+" left the game", "You can finish this game against the computer (Hattie Mae's level), or go back to the title screen.", "Play the Computer", "Back to Title", func(yes bool) {
		if !yes {
			a.G = nil
			a.restoreTitleBoard()
			a.ui.showTitle()
			return
		}
		// carry on: the computer takes the empty seat
		g := a.G
		g.Mode, g.Opp, g.Online = "ai", "hattie", ""
		g.OppFull = folkByID("hattie").Full
		g.Names, g.Faces = nil, nil
		a.persist()
		a.ui.dirty()
		if a.pos.Side == a.aiSide() {
			a.aiTurn()
		}
	})
}

// leaveOnline closes the connection (telling the other player when bye is set).
func (a *App) leaveOnline(bye bool) {
	n := a.net
	if n == nil {
		return
	}
	if bye && n.peer != nil {
		n.peer.send(Msg{T: "bye"})
	}
	if n.peer != nil {
		pk := n.peer
		time.AfterFunc(300*time.Millisecond, pk.close) // let the bye go out first
	}
	if n.ln != nil {
		_ = n.ln.Close()
	}
	n.gone = true
	a.net = nil
}

// restoreTitleBoard puts the saved single-player game (or a fresh board) behind the title screen.
func (a *App) restoreTitleBoard() {
	a.cancelThinking()
	a.gen++
	a.busy, a.thinking = false, false
	a.anims = nil
	a.hidden, a.ghosts, a.lingering = map[int]bool{}, map[int]bool{}, map[int]int8{}
	a.view, a.viewPos = -1, nil
	a.clearSel()
	if g := a.sd.Game; g != nil && g.Moves != nil {
		a.G = g
	} else {
		f := true
		a.G = &GameRec{FEN: StartFEN, Moves: []string{}, Mode: "ai", Opp: a.sd.Setup.Opp, YouBlack: true, Forced: &f, Flipped: true, Started: time.Now().UnixMilli()}
	}
	a.rebuild()
	a.flipped = a.G.Flipped
	a.ui.boardDirty()
}
