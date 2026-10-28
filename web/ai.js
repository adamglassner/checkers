/* Checkers - computer opponent. Runs in a Web Worker so the window never stalls.
   Alpha-beta (PVS) with iterative deepening, a transposition table, forced-
   capture extensions, killer/history ordering and an evaluation that knows
   about back-rank guards, the centre, runaway men and trading down when ahead.
   Weaker players search shallower, add noise and sometimes miss things.
   (c) 2026 Ironite */
'use strict';
if (typeof importScripts === 'function') importScripts('checkers.js');
const K = self.Checkers;
const { BLACK, RED, MAN, KING, NB, rowOf, colOf } = K;

/* ----------------------------------------------------------- evaluation */
const CENTRE = new Set([9, 10, 13, 14, 17, 18, 21, 22]); // squares 10,11,14,15,18,19,22,23
const KING_SQ = new Int16Array(32);
for (let s = 0; s < 32; s++) { const r = rowOf(s), c = colOf(s); KING_SQ[s] = Math.round(14 - (Math.abs(3.5 - r) + Math.abs(3.5 - c)) * 3); }
// Back-rank guard squares: Black guards from 1,3 (index 0,2) and Red from 30,32 (29,31); the whole row counts a little.
function evaluate(pos) {
  const b = pos.board;
  let men = [0, 0], kings = [0, 0], score = 0;
  for (let s = 0; s < 32; s++) {
    const p = b[s]; if (!p) continue;
    const i = p > 0 ? 0 : 1, sgn = p > 0 ? 1 : -1, r = rowOf(s);
    if (Math.abs(p) === MAN) {
      men[i]++;
      const adv = p > 0 ? r : 7 - r;        // rows advanced, 0..6
      score += sgn * (adv * adv * .9);
      if (CENTRE.has(s)) score += sgn * 5;
      // runaway: nothing of the enemy ahead on the path to the crown row
      if (adv >= 4) {
        let free = true;
        for (let rr = r + sgn; rr !== (p > 0 ? 8 : -1) && free; rr += sgn) for (let t = 0; t < 32; t++) if (rowOf(t) === rr && b[t] * p < 0 && Math.abs(colOf(t) - colOf(s)) <= Math.abs(rr - r)) { free = false; break; }
        if (free) score += sgn * (12 + adv * 4);
      }
    } else {
      kings[i]++;
      score += sgn * KING_SQ[s];
    }
  }
  const total = men[0] + men[1] + kings[0] + kings[1];
  const mat = [men[0] * 100 + kings[0] * 150, men[1] * 100 + kings[1] * 150];
  score += mat[0] - mat[1];
  // back rank, while there's still a midgame to defend
  if (total > 10) {
    if (b[0] === MAN) score += 9; if (b[2] === MAN) score += 11; if (b[1] === MAN) score += 4; if (b[3] === MAN) score += 3;
    if (b[31] === -MAN) score -= 9; if (b[29] === -MAN) score -= 11; if (b[30] === -MAN) score -= 4; if (b[28] === -MAN) score -= 3;
  }
  // trade down when ahead
  const diff = mat[0] - mat[1];
  if (diff) score += Math.sign(diff) * Math.min(80, Math.abs(diff)) * (24 - total) / 24;
  // endgame: the stronger side's kings close in, the weaker side hides in a double corner
  if (total <= 10 && diff) {
    const strong = diff > 0 ? 1 : -1;
    let dist = 0, n = 0;
    for (let s = 0; s < 32; s++) if (b[s] * strong === KING) for (let t = 0; t < 32; t++) if (b[t] * strong < 0) { dist += Math.max(Math.abs(rowOf(s) - rowOf(t)), Math.abs(colOf(s) - colOf(t))); n++; }
    if (n) score -= strong * (dist / n) * 4;
    // squeeze the weaker side: fewer free squares for its pieces is better for us
    let mob = 0;
    for (let s = 0; s < 32; s++) { const p = b[s]; if (p * strong >= 0) continue; for (const d of K.dirsFor(p)) { const t = NB[s][d]; if (t >= 0 && !b[t]) mob++; } }
    score -= strong * mob * 4;
    for (const dc of [0, 4, 27, 31]) if (b[dc] * strong < 0) score -= strong * 6; // double corners: squares 1,5 and 28,32
  }
  return pos.side === BLACK ? score : -score;
}

/* --------------------------------------------------------------- search */
const TT_SIZE = 1 << 20;
const ttKey1 = new Uint32Array(TT_SIZE), ttKey2 = new Uint32Array(TT_SIZE);
const ttScore = new Int32Array(TT_SIZE), ttDepth = new Int8Array(TT_SIZE), ttFlag = new Uint8Array(TT_SIZE), ttMove = new Int16Array(TT_SIZE);
const WIN = 100000;
let nodes = 0, deadline = 0, stopped = false, killers = [], hist = new Int32Array(32 * 32), forced = true;
const mcode = m => m.from * 32 + m.path[m.path.length - 1];

function order(moves, ttm, ply) {
  const k = killers[ply] || [];
  return moves.map(m => {
    const c = mcode(m);
    let s = hist[c];
    if (c === ttm) s += 1e9; else if (c === k[0]) s += 1e7; else if (c === k[1]) s += 9e6;
    s += m.caps.length * 1e8 + (m.crown ? 5e7 : 0);
    return [s, m];
  }).sort((a, b) => b[0] - a[0]).map(x => x[1]);
}

function search(pos, depth, alpha, beta, ply, rootKeys) {
  if ((++nodes & 2047) === 0 && Date.now() > deadline) stopped = true;
  if (stopped) return 0;
  if (ply && (pos.quiet >= 80 || isRep(pos, rootKeys))) return 0;
  const moves = pos.legal(forced);
  if (!moves.length) return -WIN + ply;
  // forced moves and captures don't use up depth
  if (moves.length === 1 || (moves[0].caps.length && depth <= 0)) depth += moves.length === 1 ? 1 : 0;
  if (depth <= 0 && !moves[0].caps.length) return evaluate(pos);
  if (depth <= 0) depth = 1; // finish the capture sequence

  const idx = pos.h1 & (TT_SIZE - 1);
  let ttm = -1;
  if (ttKey1[idx] === pos.h1 && ttKey2[idx] === pos.h2) {
    ttm = ttMove[idx];
    if (ply && ttDepth[idx] >= depth) {
      const sc = ttScore[idx], f = ttFlag[idx];
      if (f === 0) return sc; if (f === 1 && sc >= beta) return sc; if (f === 2 && sc <= alpha) return sc;
    }
  }
  const a0 = alpha;
  let best = -Infinity, bestM = null, first = true;
  for (const m of order(moves, ttm, ply)) {
    pos.make(m);
    let sc;
    if (first) sc = -search(pos, depth - 1, -beta, -alpha, ply + 1, rootKeys);
    else {
      sc = -search(pos, depth - 1, -alpha - 1, -alpha, ply + 1, rootKeys);
      if (sc > alpha && sc < beta) sc = -search(pos, depth - 1, -beta, -alpha, ply + 1, rootKeys);
    }
    pos.unmake(); first = false;
    if (stopped) return 0;
    if (sc > best) { best = sc; bestM = m; }
    if (sc > alpha) alpha = sc;
    if (alpha >= beta) {
      if (!m.caps.length) { const c = mcode(m); hist[c] += depth * depth; const k = killers[ply] || (killers[ply] = [-1, -1]); if (k[0] !== c) { k[1] = k[0]; k[0] = c; } }
      break;
    }
  }
  ttKey1[idx] = pos.h1; ttKey2[idx] = pos.h2; ttScore[idx] = best; ttDepth[idx] = Math.min(127, depth); ttMove[idx] = bestM ? mcode(bestM) : -1;
  ttFlag[idx] = best <= a0 ? 2 : best >= beta ? 1 : 0;
  return best;
}
function isRep(pos, rootKeys) {
  const k = pos.key(), n = pos.keys.length;
  // within the search line, or matching an earlier position of the real game
  for (let i = n - 3; i >= 0; i -= 2) if (pos.keys[i] === k) return true;
  return rootKeys.has(k);
}

/* Root: returns {move, score, depth}. Options: time (ms), maxDepth, noise (cp), blunder (0..1). */
function think(pos, o) {
  forced = o.forced !== false;
  const legal = pos.legal(forced);
  if (!legal.length) return { move: null, score: -WIN };
  if (legal.length === 1) return { move: pos.notation(legal[0]), score: 0, depth: 0 };
  nodes = 0; stopped = false; killers = []; hist.fill(0);
  deadline = Date.now() + (o.time || 60000);
  const maxDepth = o.maxDepth || 60;
  // earlier game positions count as repetitions (keys before the root, same side to move)
  const rootKeys = new Set(); for (let i = pos.keys.length - 3; i >= 0; i -= 2) rootKeys.add(pos.keys[i]);
  let scored = legal.map(m => ({ m, s: 0 })), bestDepth = 0;
  for (let d = 1; d <= maxDepth; d++) {
    const res = [];
    let alpha = -Infinity;
    for (const it of scored) {
      pos.make(it.m);
      // full window for every root move so the weaker levels can choose among near-equal moves
      const sc = -search(pos, d - 1, -WIN - 1, o.noise ? WIN + 1 : -alpha, 1, rootKeys);
      pos.unmake();
      if (stopped) break;
      res.push({ m: it.m, s: sc }); if (sc > alpha) alpha = sc;
    }
    if (stopped && res.length < scored.length) { if (res.length && res[0].s >= scored[0].s) { /* keep the partial if its best improved */ const done = new Set(res.map(r => r.m)); scored = res.concat(scored.filter(x => !done.has(x.m))).sort((a, b) => b.s - a.s); } break; }
    scored = res.sort((a, b) => b.s - a.s); bestDepth = d;
    if (Math.abs(scored[0].s) > WIN - 200) break; // found a forced win or loss
  }
  let pick = scored[0];
  if (o.noise) {
    const noisy = scored.map(x => ({ m: x.m, s: x.s + (Math.random() * 2 - 1) * o.noise }));
    noisy.sort((a, b) => b.s - a.s); pick = scored.find(x => x.m === noisy[0].m);
  }
  if (o.blunder && Math.random() < o.blunder && scored.length > 1) pick = scored[1 + Math.floor(Math.random() * Math.min(3, scored.length - 1))];
  // a little variety in the opening between equal moves
  if (pos.keys.length < 8 && !o.noise) { const near = scored.filter(x => x.s >= scored[0].s - 6); pick = near[Math.floor(Math.random() * near.length)]; }
  return { move: pos.notation(pick.m), score: scored[0].s, depth: bestDepth, nodes };
}

function replay(fen, moves) {
  const pos = new K.Position(fen);
  for (const n of moves) { const m = pos.legal(false).find(x => pos.notation(x) === n); if (!m) break; pos.make(m); }
  return pos;
}

if (typeof onmessage !== 'undefined' || typeof importScripts === 'function') {
  onmessage = e => {
    const d = e.data;
    try { const pos = replay(d.fen || K.START, d.moves || []); const r = think(pos, d); postMessage(Object.assign({ id: d.id }, r)); }
    catch (err) { postMessage({ id: d.id, error: String(err && err.stack || err) }); }
  };
}
self.CheckersAI = { think, replay, evaluate };
