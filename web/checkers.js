/* Checkers rules engine - (c) 2026 Ironite.
 * American checkers (English draughts) on the 32 dark squares, numbered 1-32
 * in the standard way: Black starts on 1-12 at the top of the diagram and
 * moves first, Red starts on 21-32. Internally squares are 0..31.
 * Pieces: 1 black man, 2 black king, -1 red man, -2 red king.
 * Rules: men move and capture diagonally forward, kings both ways, one square
 * at a time. Captures are compulsory (the player may choose which), and a
 * jump must be continued while it can be. A man that reaches the far row is
 * crowned and his move ends there. You lose when you have no piece or no
 * legal move. Draws: threefold repetition, or forty moves each with no
 * capture and no man moved. Works in the page and in the AI worker. */
(function (root) {
  'use strict';
  const BLACK = 1, RED = -1, MAN = 1, KING = 2;
  const DR = [-1, -1, 1, 1], DC = [-1, 1, -1, 1]; // up-left, up-right, down-left, down-right (diagram rows grow downward)
  const rowOf = s => s >> 2;
  const colOf = s => { const r = s >> 2; return (r & 1) ? 2 * (s & 3) : 2 * (s & 3) + 1; };
  const sqAt = (r, c) => { if (r < 0 || r > 7 || c < 0 || c > 7 || ((r + c) & 1) === 0) return -1; return r * 4 + ((r & 1) ? c / 2 : (c - 1) / 2); };
  const NB = [], JP = [];
  for (let s = 0; s < 32; s++) {
    NB.push(DR.map((dr, d) => sqAt(rowOf(s) + dr, colOf(s) + DC[d])));
    JP.push(DR.map((dr, d) => sqAt(rowOf(s) + 2 * dr, colOf(s) + 2 * DC[d])));
  }
  const dirsFor = p => (Math.abs(p) === KING ? [0, 1, 2, 3] : p > 0 ? [2, 3] : [0, 1]);
  const crownRow = side => (side === BLACK ? 7 : 0);

  // Zobrist keys, two 32-bit halves.
  let seed = 0x2545f491;
  const rnd = () => { seed ^= seed << 13; seed ^= seed >>> 17; seed ^= seed << 5; return seed >>> 0; };
  const Z1 = new Uint32Array(5 * 32), Z2 = new Uint32Array(5 * 32);
  for (let i = 0; i < Z1.length; i++) { Z1[i] = rnd(); Z2[i] = rnd(); }
  const ZS1 = rnd(), ZS2 = rnd();
  const zi = (p, s) => (p + 2) * 32 + s;

  const START = 'B:W21-32:B1-12';

  class Position {
    constructor(fen) {
      this.board = new Int8Array(32);
      this.side = BLACK;
      this.quiet = 0;          // plies since a capture or a man move
      this.hist = [];          // undo records
      this.keys = [];          // position keys, for repetition
      this.load(fen || START);
    }
    load(fen) {
      // FEN-like: "B:W21-32,K5:B1-12" (side to move, then each side's squares; K = king)
      this.board.fill(0);
      const parts = fen.split(':');
      this.side = parts[0] === 'W' || parts[0] === 'R' ? RED : BLACK;
      for (const part of parts.slice(1)) {
        const col = part[0] === 'B' ? BLACK : RED;
        for (const tok of part.slice(1).split(',').filter(Boolean)) {
          const k = tok[0] === 'K', t = k ? tok.slice(1) : tok;
          const [a, b] = t.split('-').map(Number);
          for (let n = a; n <= (b || a); n++) this.board[n - 1] = col * (k ? KING : MAN);
        }
      }
      this.quiet = 0; this.hist = []; this.computeHash(); this.keys = [this.key()];
    }
    fen() {
      const list = side => { const a = []; for (let s = 0; s < 32; s++) if (this.board[s] * side > 0) a.push((this.board[s] * side === KING ? 'K' : '') + (s + 1)); return a.join(','); };
      return `${this.side === BLACK ? 'B' : 'W'}:W${list(RED)}:B${list(BLACK)}`;
    }
    clone() { const p = new Position(); p.board.set(this.board); p.side = this.side; p.quiet = this.quiet; p.h1 = this.h1; p.h2 = this.h2; p.keys = this.keys.slice(); return p; }
    computeHash() {
      let a = 0, b = 0;
      for (let s = 0; s < 32; s++) if (this.board[s]) { a ^= Z1[zi(this.board[s], s)]; b ^= Z2[zi(this.board[s], s)]; }
      if (this.side === RED) { a ^= ZS1; b ^= ZS2; }
      this.h1 = a >>> 0; this.h2 = b >>> 0;
    }
    key() { return this.h1 * 4294967296 + this.h2; }
    count(side) { let n = 0; for (let s = 0; s < 32; s++) if (this.board[s] * side > 0) n++; return n; }

    /* ---- move generation. A move: {from, path:[landing squares], caps:[squares], piece, crown} */
    captures() {
      const out = [], b = this.board, side = this.side;
      for (let s = 0; s < 32; s++) {
        const p = b[s]; if (p * side <= 0) continue;
        b[s] = 0;
        this._jumps(s, s, p, [], [], out);
        b[s] = p;
      }
      return out;
    }
    _jumps(from, at, p, path, caps, out) {
      const b = this.board; let found = false;
      for (const d of dirsFor(p)) {
        const over = NB[at][d], to = JP[at][d];
        if (to < 0 || b[to] !== 0 || b[over] * p >= 0 || caps.includes(over)) continue;
        found = true;
        const np = path.concat(to), nc = caps.concat(over);
        if (Math.abs(p) === MAN && rowOf(to) === crownRow(Math.sign(p))) out.push({ from, path: np, caps: nc, piece: p, crown: true });
        else this._jumps(from, to, p, np, nc, out);
      }
      if (!found && path.length) out.push({ from, path, caps, piece: p, crown: false });
    }
    simple() {
      const out = [], b = this.board, side = this.side;
      for (let s = 0; s < 32; s++) {
        const p = b[s]; if (p * side <= 0) continue;
        for (const d of dirsFor(p)) {
          const to = NB[s][d];
          if (to >= 0 && b[to] === 0) out.push({ from: s, path: [to], caps: [], piece: p, crown: Math.abs(p) === MAN && rowOf(to) === crownRow(side) });
        }
      }
      return out;
    }
    // forced: captures are compulsory (standard). With forced=false, all moves are allowed.
    legal(forced = true) {
      const c = this.captures();
      if (c.length && forced) return c;
      return c.concat(this.simple());
    }
    make(m) {
      const b = this.board, to = m.path[m.path.length - 1];
      const rec = { m, quiet: this.quiet, h1: this.h1, h2: this.h2, capPieces: m.caps.map(s => b[s]) };
      b[m.from] = 0;
      for (const s of m.caps) b[s] = 0;
      b[to] = m.crown ? Math.sign(m.piece) * KING : m.piece;
      this.side = -this.side;
      this.quiet = (m.caps.length || Math.abs(m.piece) === MAN) ? 0 : this.quiet + 1;
      // incremental hash
      let a = this.h1 ^ Z1[zi(m.piece, m.from)] ^ Z1[zi(b[to], to)] ^ ZS1, c = this.h2 ^ Z2[zi(m.piece, m.from)] ^ Z2[zi(b[to], to)] ^ ZS2;
      m.caps.forEach((s, i) => { a ^= Z1[zi(rec.capPieces[i], s)]; c ^= Z2[zi(rec.capPieces[i], s)]; });
      this.h1 = a >>> 0; this.h2 = c >>> 0;
      this.hist.push(rec); this.keys.push(this.key());
    }
    unmake() {
      const rec = this.hist.pop(); if (!rec) return;
      const m = rec.m, b = this.board, to = m.path[m.path.length - 1];
      b[to] = 0; b[m.from] = m.piece;
      m.caps.forEach((s, i) => { b[s] = rec.capPieces[i]; });
      this.side = -this.side; this.quiet = rec.quiet; this.h1 = rec.h1; this.h2 = rec.h2; this.keys.pop();
    }
    repetitions() { const k = this.key(); let n = 0; for (let i = this.keys.length - 1; i >= 0; i -= 2) if (this.keys[i] === k) n++; return n; }
    status(forced = true) {
      const legal = this.legal(forced);
      if (!legal.length) return { over: true, winner: -this.side, reason: this.count(this.side) ? 'blocked' : 'captured' };
      if (this.repetitions() >= 3) return { over: true, winner: 0, reason: 'threefold repetition' };
      if (this.quiet >= 80) return { over: true, winner: 0, reason: 'forty-move rule' };
      return { over: false };
    }
    notation(m) { return m.caps.length ? [m.from].concat(m.path).map(s => s + 1).join('x') : `${m.from + 1}-${m.path[0] + 1}`; }
    perft(d, forced = true) {
      const ms = this.legal(forced); if (d === 1) return ms.length;
      let n = 0; for (const m of ms) { this.make(m); n += this.perft(d - 1, forced); this.unmake(); }
      return n;
    }
  }
  const sameMove = (a, b) => a && b && a.from === b.from && a.path.length === b.path.length && a.path.every((s, i) => s === b.path[i]);

  root.Checkers = { Position, BLACK, RED, MAN, KING, START, NB, JP, rowOf, colOf, sqAt, dirsFor, crownRow, sameMove };
})(typeof self !== 'undefined' ? self : this);
