/* Checkers - Hollow Creek General Store. Game flow, board, input and saving.
   (c) 2026 Ironite */
'use strict';
(async () => {
const VERSION = '1.0.0';
const $ = IR.$, K = Checkers;
const { BLACK, RED, MAN, KING } = K;

/* ---------- the regulars ---------- */
const LEVELS = {
  tucker: { ai: { maxDepth: 2, noise: 70, blunder: .25 }, delay: [700, 1500] },
  opal: { ai: { maxDepth: 3, noise: 32, blunder: .1 }, delay: [700, 1400] },
  earl: { ai: { maxDepth: 6, noise: 12, blunder: .03 }, delay: [600, 1200] },
  hattie: { ai: { time: 450, maxDepth: 11 }, delay: [500, 900] },
  doc: { ai: { time: 1300 }, delay: [300, 700] },
  gus: { ai: { time: 2800 }, delay: [200, 500] },
};
const QUIPS = {
  tucker: { greet: ["Grandpa says I can play till supper!", "I'm gonna get a king this time!"], crown: ['KING ME!', 'Crown him! Crown him!'], jump: ['Boing boing boing!', 'Double jump! Did you see that?'], win: ['I won! Wait till I tell Grandpa!'], lose: ['Good game! One more? Please?'], draw: ["Nobody won? That's weird."], accept: ['Okay! Want a candy stick?'], decline: ["No way, I'm winning!"] },
  opal: { greet: ["Oh good, the mail's sorted. Your move, hon.", "Just one game, then I've got stamps to sell."], crown: ['King me, hon!', 'Ooh, a crown! Special delivery.'], jump: ['Return to sender!', "Well, would you look at that."], win: ["Signed, sealed, delivered."], lose: ["Well played! I'll put it in the newsletter."], draw: ['Even Steven.'], accept: ["A draw it is. Peppermint?"], decline: ["Not just yet, dear."] },
  earl: { greet: ['Bless this board. Begin when you are ready.', 'Patience, friend. Patience.'], crown: ['A crown, by grace.', 'King me, if you would.'], jump: ['The meek shall be jumped.', 'Mm. Two at once.'], win: ['Slow and steady, as always.'], lose: ['A fine game. I shall reflect on it Sunday.'], draw: ['Peace be with us both.'], accept: ['Peace. I accept.'], decline: ['I believe there is more to play.'] },
  hattie: { greet: ["Let me ring this up and I'm all yours.", "Board's set. Mind the barrel, it wobbles."], crown: ['King me, sugar.', "That's a crown, honey."], jump: ['Cleaned out, like the candy jar on a Saturday.', "That'll cost you."], win: ["House always wins, sugar."], lose: ["Well, I'll be. Coffee's on me."], draw: ["Call it even and buy a soda."], accept: ['Fair enough. Draw.'], decline: ["Not a chance, sugar."] },
  doc: { greet: ["Steady hands. Let's begin.", "I've seen worse positions on a sick mule. Your move."], crown: ['King, please.', 'Crowned.'], jump: ['A clean procedure.', 'Textbook.'], win: ['Prognosis: checkmate. Well, the checkers kind.'], lose: ['Remarkable. Truly.'], draw: ['A stable condition. Draw.'], accept: ['Agreed. The position is level.'], decline: ["No. There's life in it yet."] },
  gus: { greet: ['Mm. Sit down.', "Fifty years on this barrel. Go ahead."], crown: ['King.', 'Crown it.'], jump: ['Mm-hm.', 'Saw that coming in nineteen fifty-two.'], win: ['Again, whenever you like.'], lose: ["...Well. Ain't that something. Good game."], draw: ['A draw. You played it proper.'], accept: ['Fair.'], decline: ['Nope.'] },
};
const YOU = { id: 'you', full: 'You', look: { coat: ['#6a3a2a', '#3e1e14'], hair: '#4a2e1c', skin: '#e8b892', collar: x => { IR.draw.poly(x, [[.42, 1.09], [.5, 1.18], [.58, 1.09]], '#efe4cc'); } } };
const P2 = { id: 'p2', full: 'Player two', look: { coat: ['#2f4a6a', '#1a2c42'], hair: '#1e1a17', skin: '#c99068', skinShade: '#a8704c', glasses: '#2a2a2a' } };

/* ---------- save ---------- */
const save = new IR.Save('checkers.save', {
  settings: { music: true, sfx: true, vol: 60, board: 'barrel', pieces: 'classic', dots: 'on', nums: 'off', anim: 'normal', turn: 'on' },
  setup: { mode: 'ai', opp: 'tucker', color: 'black', forced: 'on' },
  game: null,
  stats: { won: 0, drawn: 0, lost: 0, pvp: 0, per: {} },
});
await save.load();
const S = save.data.settings, SD = save.data, ST = SD.stats;

/* ---------- stage ---------- */
const W = 1600, H = 844, BX = 30, BY = 22, FR = 800, M = 40, SQ = 90;
const boardCv = $('#board'), bx = boardCv.getContext('2d');
const fx = new IR.Particles($('#fx'));
let res = 0, staticCv = null;
const dpr = () => window.devicePixelRatio || 1;

/* ---------- audio ---------- */
const audio = new IR.Audio(S);
audio.setTunes(MUSIC.tunes(), MUSIC.instruments);
const sfx = MUSIC.sfx(audio, () => S.pieces);
IR.wireAudioControls(audio, save);
IR.on('tune', n => { const e = $('#nowPlaying'); e.textContent = '♫ ' + n; e.classList.add('show'); setTimeout(() => e.classList.remove('show'), 5000); });
$('#ver').textContent = 'v' + VERSION + ' · © 2026 Ironite';

/* ---------- AI worker ---------- */
let worker = null, reqId = 0;
const pending = new Map();
function startWorker() {
  worker = new Worker('ai.js');
  worker.onmessage = e => { const p = pending.get(e.data.id); if (p) { pending.delete(e.data.id); p(e.data); } };
  worker.onerror = () => { for (const p of pending.values()) p({ error: 'worker' }); pending.clear(); worker.terminate(); startWorker(); };
}
startWorker();
const think = opts => new Promise(res => { const id = ++reqId; pending.set(id, res); worker.postMessage(Object.assign({ id, forced: !G || G.forced !== false }, opts)); });
function cancelThinking() { if (!pending.size) return; worker.terminate(); for (const p of pending.values()) p({ cancelled: true }); pending.clear(); startWorker(); }

/* ---------- game state ---------- */
let G = null;          // the saved game record
let pos = null;        // live position
let hist = [];         // per ply: {m, n (notation)}
let view = -1, viewPos = null;
let flipped = false;
let sel = -1, hops = [], legalNow = [];
let drag = null;       // {from, x, y, moved}
let anims = [];
let hidden = new Set(), ghosts = new Set(); // squares not drawn / drawn faded
let lingering = new Map(); // captured pieces still shown until the jump passes over them
let hintPath = null;
let busy = false, gen = 0, drawOfferPly = -99, flash = null;

const forced = () => !G || G.forced !== false;
const isAI = () => G && G.mode === 'ai';
const aiSide = () => (G.youBlack ? RED : BLACK);
const yourSide = () => (G.youBlack ? BLACK : RED);
const yourTurn = () => G && !G.result && (G.mode === 'pvp' || pos.side === yourSide());
const live = () => view < 0;
const curPos = () => (live() ? pos : viewPos);
const oppInfo = () => ART.FOLKS.find(f => f.id === G.opp) || ART.FOLKS[0];
const animMs = () => ({ slow: 330, normal: 220, fast: 120 }[S.anim] || 220);
const sideName = s => ART.SETS[S.pieces].names[s];
const endOf = m => m.path[m.path.length - 1];

function rebuild() {
  pos = new K.Position(G.fen || K.START); hist = [];
  for (const n of G.moves) {
    const m = pos.legal(false).find(x => pos.notation(x) === n);
    if (!m) break;
    hist.push({ m, n }); pos.make(m);
  }
  G.moves = hist.map(h => h.n);
  legalNow = pos.legal(forced());
}
function persist() { SD.game = G; save.write(); }

/* ---------- geometry ---------- */
function sqXY(s) { const r = K.rowOf(s), c = K.colOf(s); return flipped ? [M + (7 - c) * SQ, M + (7 - r) * SQ] : [M + c * SQ, M + r * SQ]; }
function xySq(x, y) {
  let c = Math.floor((x - M) / SQ), r = Math.floor((y - M) / SQ);
  if (c < 0 || c > 7 || r < 0 || r > 7) return -1;
  if (flipped) { c = 7 - c; r = 7 - r; }
  return K.sqAt(r, c);
}
function evXY(e) { const b = boardCv.getBoundingClientRect(); return [(e.clientX - b.left) / b.width * FR, (e.clientY - b.top) / b.height * FR]; }

/* ---------- drawing ---------- */
function buildStatic() {
  if (!res) return;
  const px = Math.round(FR * res), off = Math.round(M * res), inner = px - off * 2;
  const c = document.createElement('canvas'); c.width = c.height = px;
  const cx = c.getContext('2d');
  cx.drawImage(ART.boardFrame(S.board, px, inner, flipped), 0, 0);
  const surf = ART.boardSurface(S.board, inner);
  if (flipped) { cx.save(); cx.translate(off + inner, off + inner); cx.rotate(Math.PI); cx.drawImage(surf, 0, 0); cx.restore(); }
  else cx.drawImage(surf, off, off);
  if (S.nums === 'on') {
    cx.save(); cx.scale(res, res);
    cx.font = "600 15px 'Work Sans', sans-serif"; cx.fillStyle = ART.BOARDS[S.board].num; cx.textBaseline = 'top';
    for (let s = 0; s < 32; s++) { const [a, b] = sqXY(s); cx.fillText(String(s + 1), a + 6, b + 5); }
    cx.restore();
  }
  staticCv = c;
}

let raf = 0;
function dirty() { if (!raf) raf = requestAnimationFrame(draw); }
function pieceImg(code, size) { return ART.piece(code, Math.round(size * res), S.pieces); }
function draw(now) {
  raf = 0; now = now || performance.now();
  if (!staticCv) return;
  const x = bx, p = curPos();
  x.setTransform(1, 0, 0, 1, 0, 0);
  x.drawImage(staticCv, 0, 0);
  x.setTransform(res, 0, 0, res, 0, 0);
  // last move: from, landings
  const last = live() ? hist[hist.length - 1] : hist[view - 1];
  if (last && !(hops.length && live())) for (const s of [last.m.from].concat(last.m.path)) { const [a, b] = sqXY(s); x.fillStyle = 'rgba(255,196,64,.26)'; x.fillRect(a, b, SQ, SQ); }
  // pieces that must jump
  const mustJump = live() && G && !G.result && yourTurn() && !busy && legalNow.length && legalNow[0].caps.length && forced() && sel < 0;
  const jumpers = mustJump ? new Set(legalNow.map(m => m.from)) : null, pulse = .6 + .4 * Math.sin(now / 240);
  if (mustJump) for (const s of jumpers) { const [a, b] = sqXY(s); x.fillStyle = `rgba(255,205,70,${.22 + .18 * pulse})`; x.fillRect(a, b, SQ, SQ); }
  if (flash && now < flash.until) { for (const s of flash.squares) { const [a, b] = sqXY(s); x.strokeStyle = `rgba(255,210,80,${(flash.until - now) / 900})`; x.lineWidth = 6; x.strokeRect(a + 4, b + 4, SQ - 8, SQ - 8); } }
  // selection + path so far
  if (sel >= 0 && live()) {
    const trail = [sel].concat(hops);
    trail.forEach((s, i) => { const [a, b] = sqXY(s); x.fillStyle = i === trail.length - 1 ? 'rgba(255,214,90,.5)' : 'rgba(255,214,90,.28)'; x.fillRect(a, b, SQ, SQ); });
  }
  // pieces
  const ps = SQ;
  for (let s = 0; s < 32; s++) {
    const pc = p.board[s];
    if (!pc || hidden.has(s) || (drag && drag.moved && drag.from === s)) continue;
    if (live() && sel >= 0 && hops.length && s === sel) continue; // drawn at the last hop below
    const [a, b] = sqXY(s);
    if (ghosts.has(s)) x.globalAlpha = .32;
    x.drawImage(pieceImg(pc, ps), a, b, SQ, SQ);
    x.globalAlpha = 1;
  }
  if (mustJump) for (const s of jumpers) {
    const [a, b] = sqXY(s);
    x.save(); x.shadowColor = 'rgba(255,200,60,.95)'; x.shadowBlur = 10 + 10 * pulse;
    x.beginPath(); x.ellipse(a + SQ / 2, b + SQ * .5, SQ * .43, SQ * .4, 0, 0, Math.PI * 2); x.strokeStyle = `rgba(255,214,90,${.55 + .45 * pulse})`; x.lineWidth = 4; x.stroke(); x.restore();
  }
  for (const [s, pc] of lingering) { const [a, b] = sqXY(s); x.globalAlpha = ghosts.has(s) ? .32 : 1; x.drawImage(pieceImg(pc, ps), a, b, SQ, SQ); x.globalAlpha = 1; }
  if (live() && sel >= 0 && hops.length && !(drag && drag.moved)) { const [a, b] = sqXY(hops[hops.length - 1]); x.drawImage(pieceImg(pos.board[sel], ps), a, b, SQ, SQ); }
  // legal destinations
  if (sel >= 0 && live() && S.dots === 'on' && !busy) {
    for (const s of nextSquares()) {
      const [a, b] = sqXY(s);
      const jump = selMoves().some(m => m.caps.length);
      x.beginPath(); x.arc(a + SQ / 2, b + SQ / 2, SQ * (jump ? .2 : .15), 0, Math.PI * 2); x.fillStyle = jump ? 'rgba(255,205,70,.75)' : 'rgba(255,240,210,.45)'; x.fill();
      x.lineWidth = 2; x.strokeStyle = 'rgba(40,20,5,.5)'; x.stroke();
    }
  }
  // animations
  let animating = false;
  anims = anims.filter(an => {
    if (now < an.t0) { animating = true; return true; }
    const t = Math.min(1, (now - an.t0) / an.dur), e = t < .5 ? 2 * t * t : 1 - Math.pow(-2 * t + 2, 2) / 2;
    if (an.fade) { x.globalAlpha = 1 - t; x.drawImage(pieceImg(an.piece, ps), an.x0, an.y0 - t * 10, SQ, SQ); x.globalAlpha = 1; }
    else {
      const lift = Math.sin(t * Math.PI) * SQ * (an.jump ? .32 : .08);
      x.save(); x.shadowColor = 'rgba(0,0,0,.35)'; x.shadowBlur = lift * .5; x.shadowOffsetY = lift * .4;
      x.drawImage(pieceImg(an.piece, ps), an.x0 + (an.x1 - an.x0) * e - lift * .1, an.y0 + (an.y1 - an.y0) * e - lift, SQ + lift * .2, SQ + lift * .2);
      x.restore();
    }
    if (!an.started) { an.started = true; an.onStart && an.onStart(); }
    if (t < 1) { animating = true; return true; }
    if (an.done) an.done();
    return false;
  });
  if (hintPath) pathArrow(x, hintPath, 'rgba(226,170,40,.9)');
  // dragged piece on top
  if (drag && drag.moved) {
    const pc = pos.board[drag.from], sz = SQ * 1.14;
    x.save(); x.shadowColor = 'rgba(0,0,0,.45)'; x.shadowBlur = 16; x.shadowOffsetY = 10;
    x.drawImage(pieceImg(pc, sz), drag.x - sz / 2, drag.y - sz * .56, sz, sz); x.restore();
  }
  if (animating || mustJump || (flash && now < flash.until)) dirty();
}
function pathArrow(x, squares, col) {
  const pts = squares.map(s => { const [a, b] = sqXY(s); return [a + SQ / 2, b + SQ / 2]; });
  x.save(); x.strokeStyle = col; x.fillStyle = col; x.lineWidth = SQ * .16; x.lineCap = 'round'; x.lineJoin = 'round';
  const n = pts.length, [px, py] = pts[n - 2], [qx, qy] = pts[n - 1], ang = Math.atan2(qy - py, qx - px), back = SQ * .3;
  x.beginPath(); x.moveTo(pts[0][0], pts[0][1]); for (let i = 1; i < n - 1; i++) x.lineTo(pts[i][0], pts[i][1]);
  x.lineTo(qx - Math.cos(ang) * back, qy - Math.sin(ang) * back); x.stroke();
  x.beginPath(); x.moveTo(qx, qy); x.lineTo(qx - Math.cos(ang - .5) * SQ * .38, qy - Math.sin(ang - .5) * SQ * .38); x.lineTo(qx - Math.cos(ang + .5) * SQ * .38, qy - Math.sin(ang + .5) * SQ * .38); x.closePath(); x.fill();
  x.restore();
}
/* Animate a move along its path, starting from hop `skip`. Returns when it's done. */
function animateMove(m, skip = 0) {
  return new Promise(resolve => {
    const dur = animMs() * (m.caps.length ? 1.25 : 1), to = endOf(m), t0 = performance.now();
    const stops = [m.from].concat(m.path);
    hidden.add(to);
    for (let i = skip; i < m.caps.length; i++) lingering.set(m.caps[i], capturedPiece(m, i));
    for (let i = skip; i < m.path.length; i++) {
      const [x0, y0] = sqXY(stops[i]), [x1, y1] = sqXY(stops[i + 1]), k = i - skip, last = i === m.path.length - 1;
      anims.push({ piece: m.piece, x0, y0, x1, y1, jump: !!m.caps.length, t0: t0 + k * dur, dur,
        onStart: k > 0 ? () => sfx.hop(k) : null,
        done: () => {
          if (m.caps[i] != null) { const [cx, cy] = sqXY(m.caps[i]); lingering.delete(m.caps[i]); anims.push({ piece: capturedPiece(m, i), x0: cx, y0: cy, fade: true, t0: performance.now(), dur: dur * .9 }); ghosts.delete(m.caps[i]); }
          if (last) { hidden.delete(to); resolve(); }
          dirty();
        } });
    }
    dirty();
  });
}
let capturedCache = null;
function capturedPiece(m, i) { return capturedCache && capturedCache[i] != null ? capturedCache[i] : -Math.sign(m.piece); }
const waitAnims = () => new Promise(r => { const chk = () => (anims.length ? setTimeout(chk, 30) : r()); chk(); });

/* ---------- side panel ---------- */
const av = {};
function avatarFor(f) { if (!av[f.id]) av[f.id] = ART.avatar(f, 200); return av[f.id]; }
function plaqueFor(side) {
  if (!G) return { f: YOU, nm: 'You', rt: '' };
  if (G.mode === 'pvp') return side === BLACK ? { f: YOU, nm: sideName(BLACK), rt: 'Player one · moves first' } : { f: P2, nm: sideName(RED), rt: 'Player two' };
  if (side === yourSide()) { const r = ST.per[G.opp] || { w: 0, d: 0, l: 0 }; return { f: YOU, nm: 'You', rt: `Playing ${sideName(side)} · vs ${oppInfo().name}: ${r.w}–${r.d}–${r.l}` }; }
  const o = oppInfo(); return { f: o, nm: o.full, rt: `${o.rating} · ${sideName(side)}` };
}
function refreshPlaques() {
  if (!G) return;
  const topSide = flipped ? RED : BLACK;
  [['#pTop', topSide], ['#pBot', -topSide]].forEach(([id, side]) => {
    const el = $(id), info = plaqueFor(side);
    el.dataset.side = side;
    const cv = el.querySelector('.av'), cx = cv.getContext('2d'); cx.clearRect(0, 0, 200, 200); cx.drawImage(avatarFor(info.f), 0, 0);
    el.querySelector('.nm').textContent = info.nm;
    el.querySelector('.rt').textContent = info.rt;
    el.classList.toggle('turn', !G.result && pos.side === side && live());
  });
  refreshCaps();
}
function refreshCaps() {
  const p = curPos(), cnt = { 1: [0, 0], '-1': [0, 0] };
  for (let s = 0; s < 32; s++) { const pc = p.board[s]; if (pc) cnt[pc > 0 ? 1 : -1][Math.abs(pc) - 1]++; }
  const ps = Math.round(32 * Math.max(1, res));
  for (const id of ['#pTop', '#pBot']) {
    const el = $(id), side = +el.dataset.side, box = el.querySelector('.caps');
    box.innerHTML = '';
    const taken = 12 - cnt[-side][0] - cnt[-side][1]; // enemy pieces this side has captured
    for (let i = 0; i < taken; i++) { const c = document.createElement('canvas'); c.width = c.height = ps; c.getContext('2d').drawImage(ART.piece(-side, ps, S.pieces), 0, 0); box.appendChild(c); }
    const mine = cnt[side][0] + cnt[side][1] * 1.5, theirs = cnt[-side][0] + cnt[-side][1] * 1.5;
    if (mine > theirs && taken) IR.el('span', 'adv', box, '+' + (cnt[side][0] + cnt[side][1] - cnt[-side][0] - cnt[-side][1] || '♛'));
    const n = cnt[side][0] + cnt[side][1], k = cnt[side][1];
    el.querySelector('.left').innerHTML = `<b>${n}</b><span>${k ? `${k} KING${k > 1 ? 'S' : ''}` : 'PIECES'}</span>`;
  }
}
function setThinking(on) {
  for (const id of ['#pTop', '#pBot']) $(id).querySelector('.think').textContent = '';
  if (on && G && isAI()) { const id = +$('#pTop').dataset.side === aiSide() ? '#pTop' : '#pBot'; $(id).querySelector('.think').textContent = 'Thinking…'; }
}
function refreshMoves() {
  const box = $('#moveList'); box.innerHTML = '';
  const col = s => (s === BLACK ? ART.SETS[S.pieces][1].face[1] : ART.SETS[S.pieces][-1].face[1]);
  for (let i = 0; i < hist.length; i += 2) {
    IR.el('div', 'no', box, (i / 2 + 1) + '.');
    for (let j = 0; j < 2; j++) {
      const k = i + j;
      if (k >= hist.length) { IR.el('div', '', box, ''); continue; }
      const h = hist[k], e = IR.el('div', 'mv' + (h.m.crown ? ' k' : ''), box, `<i style="background:${col(Math.sign(h.m.piece))}"></i>${h.n}`);
      const shown = live() ? hist.length : view;
      if (k === shown - 1) e.classList.add('cur');
      e.onclick = () => setView(k + 1);
    }
  }
  if (G && G.result) IR.el('div', 'res', box, resultLine());
  const cur = box.querySelector('.cur'); if (cur) cur.scrollIntoView({ block: 'nearest' }); else box.scrollTop = box.scrollHeight;
  $('#moveNote').textContent = G ? (G.forced === false ? 'Jumping optional' : '') : '';
  const n = hist.length, v = live() ? n : view;
  const nb = document.querySelectorAll('.nav button');
  nb[0].disabled = nb[1].disabled = v === 0; nb[2].disabled = nb[3].disabled = v === n;
}
function setStatus(t, cls = '') { $('#statusText').textContent = t; $('#status').className = cls; }
function refreshStatus() {
  if (!G) return setStatus(' ');
  if (!live()) return setStatus(`Reviewing move ${Math.ceil(view / 2) || 0} · press End to return`, 'review');
  if (G.result) return setStatus(resultLine(), 'alert');
  const jump = legalNow.length && legalNow[0].caps.length && forced();
  if (G.mode === 'pvp') return setStatus(`${sideName(pos.side)} to move${jump ? ' · must jump!' : ''}`, jump ? 'alert' : '');
  if (yourTurn()) return setStatus(hops.length ? 'Keep jumping!' : jump ? 'You must jump!' : 'Your move', jump ? 'alert' : '');
  setStatus(`${oppInfo().name} is thinking…`);
}
function resultLine() {
  const r = G.result; if (!r) return '';
  const why = { captured: 'took every piece', blocked: 'left no moves', resignation: 'by resignation', 'threefold repetition': 'Threefold repetition', 'forty-move rule': 'Forty-move rule', agreement: 'Draw agreed' }[r.reason] || r.reason;
  if (!r.winner) return `Draw · ${why}`;
  return r.reason === 'resignation' ? `${sideName(r.winner)} wins by resignation` : `${sideName(r.winner)} wins · ${why}`;
}
function refreshButtons() {
  const ok = G && !G.result && live();
  $('#bUndo').disabled = !G || !!G.result || !hist.length;
  $('#bHint').disabled = !ok || !yourTurn() || busy;
  $('#bDraw').disabled = !ok || !yourTurn() || hist.length < 2 || (isAI() && hist.length - drawOfferPly < 10);
  $('#bResign').disabled = !ok || hist.length < 1;
  $('#sWon').textContent = ST.won; $('#sDrawn').textContent = ST.drawn; $('#sLost').textContent = ST.lost;
}
function refreshAll() { refreshPlaques(); refreshMoves(); refreshStatus(); refreshButtons(); dirty(); }

/* ---------- reviewing ---------- */
function setView(v) {
  if (!G) return;
  clearSel();
  if (v >= hist.length) { view = -1; viewPos = null; }
  else { view = Math.max(0, v); viewPos = new K.Position(G.fen || K.START); for (let i = 0; i < view; i++) viewPos.make(hist[i].m); }
  hintPath = null; sfx.click(); refreshAll();
}

/* ---------- playing moves ---------- */
async function playMove(m, opts = {}) {
  const mover = pos.side, n = pos.notation(m);
  capturedCache = m.caps.map(s => pos.board[s]);
  hist.push({ m, n }); G.moves.push(n);
  const skip = opts.skip || 0;
  clearSel(true); hintPath = null;
  // the hops already shown while the player was choosing stay faded until this lands
  pos.make(m); legalNow = pos.legal(forced());
  const st = pos.status(forced());
  if (st.over) G.result = { reason: st.reason, winner: st.winner };
  persist();
  if (!m.caps.length) sfx.move(); else if (!skip) sfx.hop(0);
  refreshAll();
  await animateMove(m, skip);
  ghosts.clear(); hidden.clear(); lingering.clear();
  if (m.caps.length) sfx.capture(m.caps.length);
  if (m.crown) { setTimeout(() => sfx.crown(), 120); crownBurst(endOf(m)); }
  refreshAll();
  if (G.result) return gameOver();
  if (isAI() && mover === aiSide()) { if (m.crown && Math.random() < .7) quip('crown'); else if (m.caps.length >= 2 && Math.random() < .8) quip('jump'); }
  if (G.mode === 'pvp' && S.turn === 'on') setTimeout(() => setFlip(pos.side === BLACK), 250);
  if (isAI() && pos.side === aiSide()) aiTurn();
}
function crownBurst(s) { const [a, b] = sqXY(s); fx.burst(BX + a + SQ / 2, BY + b + SQ / 2, 40, ['#f5d277', '#fff3c4', '#e3bd5a'], { speed: 320, life: 1.2 }); }

async function aiTurn() {
  const my = ++gen;
  busy = true; setThinking(true); refreshStatus(); refreshButtons();
  const L = LEVELS[G.opp] || LEVELS.tucker;
  const t0 = performance.now();
  await waitAnims();
  const r = await think(Object.assign({ fen: G.fen || K.START, moves: G.moves }, L.ai));
  if (my !== gen || r.cancelled) return;
  let n = r.move;
  if (r.error || !n) { console.error(r.error); n = legalNow[0] && pos.notation(legalNow[0]); }
  const minWait = L.delay[0] + Math.random() * (L.delay[1] - L.delay[0]);
  const waited = performance.now() - t0;
  if (waited < minWait) await new Promise(res => setTimeout(res, minWait - waited));
  if (my !== gen || !G || G.result) return;
  const m = legalNow.find(x => pos.notation(x) === n);
  busy = false; setThinking(false);
  if (m) await playMove(m);
}

/* ---------- player input ---------- */
function clearSel(keepGhosts) { sel = -1; hops = []; if (!keepGhosts) { ghosts.clear(); } dirty(); }
function selMoves() { return sel < 0 ? [] : legalNow.filter(m => m.from === sel && hops.every((h, i) => m.path[i] === h)); }
function nextSquares() { return [...new Set(selMoves().map(m => m.path[hops.length]).filter(s => s != null))]; }
function canMoveFrom(sq) {
  if (!G || G.result || !live() || busy) return false;
  if (!yourTurn()) return false;
  return legalNow.some(m => m.from === sq);
}
function ownPiece(sq) { return G && pos.board[sq] && Math.sign(pos.board[sq]) === pos.side; }
function select(sq) { clearSel(); sel = sq; sfx.pick(); refreshStatus(); dirty(); }
// Click/drop on `to` with `sel` selected. Returns true if it was a legal step.
function stepTo(to) {
  const cands = selMoves();
  const next = cands.filter(m => m.path[hops.length] === to);
  if (next.length) {
    const prev = hops.length ? hops[hops.length - 1] : sel;
    hops.push(to);
    const done = next.find(m => m.path.length === hops.length);
    if (done && next.length === 1) { const skip = hops.length - 1; finishMove(done, skip, prev); return true; }
    // partway through a multi-jump: show the hop, fade the jumped piece
    const mv = next[0], i = hops.length - 1;
    if (mv.caps[i] != null) ghosts.add(mv.caps[i]);
    sfx.hop(i); refreshStatus(); dirty();
    return true;
  }
  // straight to the final landing square, if that identifies one move
  const fin = cands.filter(m => endOf(m) === to);
  if (fin.length === 1 || (fin.length > 1 && fin.every(m => m.path.join() === fin[0].path.join()))) { finishMove(fin[0], hops.length, null); return true; }
  return false;
}
function finishMove(m, skip, prev) { void prev; for (let i = 0; i < skip; i++) if (m.caps[i] != null) ghosts.add(m.caps[i]); hops = []; sel = -1; playMove(m, { skip }); }
function nudgeJump(sq) {
  const jumpers = [...new Set(legalNow.map(m => m.from))];
  flash = { squares: jumpers, until: performance.now() + 900 }; sfx.mustJump();
  banner('You must jump!', ownPiece(sq) ? 'A glowing piece has a capture to make' : ''); dirty();
}

boardCv.addEventListener('contextmenu', e => e.preventDefault());
boardCv.addEventListener('pointerdown', e => {
  audio.unlock();
  if (e.button !== 0) { if (hops.length) { clearSel(); refreshStatus(); } return; }
  const [x, y] = evXY(e), sq = xySq(x, y);
  if (!live()) { setView(hist.length); return; }
  if (sq < 0) { if (!hops.length) clearSel(); return; }
  if (sel >= 0 && sq !== sel && (hops.length ? sq !== hops[hops.length - 1] : true) && stepTo(sq)) return;
  if (hops.length) { // mid-jump: only the piece's next landings are allowed
    if (sq === sel || sq === hops[hops.length - 1]) return;
    sfx.illegal(); flash = { squares: nextSquares(), until: performance.now() + 900 }; dirty(); return;
  }
  if (canMoveFrom(sq)) {
    const wasSel = sel === sq;
    if (!wasSel) select(sq);
    drag = { from: sq, x, y, x0: x, y0: y, moved: false, wasSel };
    boardCv.setPointerCapture(e.pointerId); boardCv.classList.add('grab');
  } else {
    if (yourTurn() && !busy && ownPiece(sq) && forced() && legalNow.length && legalNow[0].caps.length) nudgeJump(sq);
    else if (sel >= 0 && pos.board[sq]) sfx.illegal();
    clearSel();
  }
});
boardCv.addEventListener('pointermove', e => {
  const [x, y] = evXY(e);
  if (drag) { drag.x = x; drag.y = y; if (!drag.moved && Math.hypot(x - drag.x0, y - drag.y0) > 6) drag.moved = true; dirty(); return; }
  const sq = xySq(x, y);
  boardCv.classList.toggle('hand', sq >= 0 && (canMoveFrom(sq) || (sel >= 0 && (nextSquares().includes(sq) || selMoves().some(m => endOf(m) === sq)))));
});
boardCv.addEventListener('pointerup', e => {
  if (!drag) return;
  const [x, y] = evXY(e), sq = xySq(x, y);
  const d = drag; drag = null; boardCv.classList.remove('grab');
  if (d.moved) {
    if (sq >= 0 && sq !== d.from && stepTo(sq)) return;
    if (sq >= 0 && sq !== d.from) sfx.illegal();
    dirty();
  } else if (d.wasSel) clearSel();
});

/* ---------- flipping ---------- */
function setFlip(f) { if (f === flipped) return; flipped = f; if (G) G.flipped = f; buildStatic(); refreshPlaques(); dirty(); }

/* ---------- game over ---------- */
function quip(kind) {
  if (!isAI()) return;
  const list = (QUIPS[G.opp] || QUIPS.tucker)[kind]; if (!list) return;
  const el = +$('#pTop').dataset.side === aiSide() ? $('#pTop') : $('#pBot');
  const r = el.getBoundingClientRect(), sr = $('#stage').getBoundingClientRect(), sc = IR.Stage.scale;
  const x = (r.left - sr.left) / sc + 64, y = (r.top - sr.top) / sc - 56 + (el.id === 'pTop' ? 140 : -12);
  IR.bubble($('#table'), list[Math.floor(Math.random() * list.length)], x + 200, y, el.id === 'pTop' ? 'down' : '', 3200);
}
function gameOver() {
  busy = false; setThinking(false); clearSel();
  const r = G.result, moves = Math.ceil(hist.length / 2);
  let kick = '', title = '', text = resultLine(), face = YOU;
  if (isAI()) {
    face = oppInfo();
    const you = yourSide();
    const rec = ST.per[G.opp] = ST.per[G.opp] || { w: 0, d: 0, l: 0 };
    if (!G.counted) {
      if (!r.winner) { ST.drawn++; rec.d++; } else if (r.winner === you) { ST.won++; rec.w++; } else { ST.lost++; rec.l++; }
      G.counted = true;
    }
    if (!r.winner) { title = 'A Draw'; kick = 'Even at the barrel'; sfx.draw(); }
    else if (r.winner === you) { title = 'You Win!'; kick = r.reason === 'blocked' ? 'Nowhere left to go' : r.reason === 'captured' ? 'Board cleared' : 'Victory'; sfx.win(); confetti(); }
    else { title = `${oppInfo().name} Wins`; kick = r.reason === 'blocked' ? 'Blocked in' : r.reason === 'captured' ? 'Cleaned out' : 'Defeat'; sfx.lose(); }
    const q = (QUIPS[G.opp] || QUIPS.tucker)[!r.winner ? 'draw' : r.winner === you ? 'lose' : 'win'];
    $('#overQuote').textContent = `“${q[Math.floor(Math.random() * q.length)]}” — ${oppInfo().full}`;
    text = `${resultLine()} after ${moves} moves. Your record against ${oppInfo().name}: ${rec.w} won, ${rec.d} drawn, ${rec.l} lost.`;
  } else {
    if (!G.counted) { ST.pvp++; G.counted = true; }
    title = r.winner ? `${sideName(r.winner)} Wins` : 'A Draw'; kick = 'Game over';
    face = r.winner === RED ? P2 : YOU;
    $('#overQuote').textContent = '';
    if (r.winner) { sfx.win(); confetti(); } else sfx.draw();
    text = `${resultLine()} after ${moves} moves.`;
  }
  SD.game = G; save.write(true);
  $('#overKick').textContent = kick; $('#overTitle').textContent = title; $('#overText').textContent = text;
  const oc = $('#overAv').getContext('2d'); oc.clearRect(0, 0, 240, 240); oc.drawImage(avatarFor(face), 0, 0, 240, 240);
  refreshAll();
  setTimeout(() => IR.panel.show('over'), 900);
}
function confetti() { const cols = ['#eac46a', '#f7ecd4', '#8e2a1e', '#2f4a33', '#c2241d', '#1d1d22']; fx.burst(BX + FR / 2, BY + FR * .45, 150, cols, { speed: 720, life: 2.6 }); }

/* ---------- commands ---------- */
function newGame(setup) {
  cancelThinking(); gen++; busy = false;
  const youB = setup.color === 'random' ? Math.random() < .5 : setup.color !== 'red';
  G = { fen: K.START, moves: [], mode: setup.mode, opp: setup.opp, youBlack: setup.mode === 'ai' ? youB : true, forced: setup.forced !== 'off', result: null, counted: false, started: Date.now() };
  drawOfferPly = -99; view = -1; viewPos = null; clearSel(); hintPath = null; anims = []; hidden.clear(); ghosts.clear(); lingering.clear();
  rebuild();
  flipped = G.mode === 'ai' ? G.youBlack : true; G.flipped = flipped;
  buildStatic(); persist(); refreshAll();
  IR.panel.hide('title'); IR.panel.hide('over');
  sfx.start();
  if (isAI()) { setTimeout(() => quip('greet'), 500); if (pos.side === aiSide()) aiTurn(); }
}
function resumeGame() {
  G = SD.game; rebuild();
  flipped = !!G.flipped; view = -1; viewPos = null; busy = false; clearSel();
  buildStatic(); refreshAll();
  IR.panel.hide('title');
  if (isAI() && !G.result && pos.side === aiSide()) aiTurn();
}
function undo() {
  if (!G || G.result || !hist.length) return;
  if (!live()) setView(hist.length);
  cancelThinking(); gen++; busy = false; setThinking(false); clearSel(); anims = []; hidden.clear(); lingering.clear();
  let n = 1;
  if (isAI()) { n = pos.side === aiSide() ? 1 : 2; if (hist.length - n < 0) n = hist.length; if (!G.youBlack && hist.length - n === 0) n = hist.length - 1; if (n <= 0) return; }
  const undone = hist.slice(-n);
  G.moves = G.moves.slice(0, -n); rebuild();
  const last = undone[0].m;
  // slide the piece back to where it started
  const [x0, y0] = sqXY(endOf(last)), [x1, y1] = sqXY(last.from);
  hidden.add(last.from); anims.push({ piece: last.piece, x0, y0, x1, y1, t0: performance.now(), dur: animMs(), done: () => hidden.delete(last.from) });
  sfx.move(); hintPath = null;
  if (G.mode === 'pvp' && S.turn === 'on') setFlip(pos.side === BLACK);
  persist(); refreshAll();
  if (isAI() && pos.side === aiSide()) aiTurn();
}
async function hint() {
  if (!G || !yourTurn() || !live() || busy) return;
  busy = true; clearSel(); refreshButtons(); setStatus('Studying the board…');
  const my = gen;
  const r = await think({ fen: G.fen || K.START, moves: G.moves, time: 1200 });
  busy = false;
  if (my !== gen || r.cancelled || !r.move) { refreshAll(); return; }
  const m = legalNow.find(x => pos.notation(x) === r.move);
  if (!m) { refreshAll(); return; }
  hintPath = [m.from].concat(m.path); sfx.hint();
  refreshButtons(); setStatus('Hint: ' + r.move, 'alert'); dirty();
}
function ask(title, text, yes, no = 'No') {
  return new Promise(res => {
    $('#cTitle').textContent = title; $('#cText').textContent = text; $('#cYes').textContent = yes; $('#cNo').textContent = no;
    IR.panel.show('confirm');
    $('#cYes').onclick = () => { IR.panel.hide('confirm'); sfx.click(); res(true); };
    $('#cNo').onclick = () => { IR.panel.hide('confirm'); sfx.click(); res(false); };
  });
}
async function resign() {
  if (!G || G.result || !hist.length) return;
  if (!await ask('Resign?', G.mode === 'pvp' ? `${sideName(pos.side)} resigns and ${sideName(-pos.side)} wins the game.` : `${oppInfo().name} will be credited with the win.`, 'Resign', 'Keep Playing')) return;
  cancelThinking(); gen++; busy = false;
  const loser = G.mode === 'pvp' ? pos.side : yourSide();
  G.result = { reason: 'resignation', winner: -loser };
  persist(); gameOver();
}
async function offerDraw() {
  if (!G || G.result || !yourTurn()) return;
  if (G.mode === 'pvp') {
    if (!await ask('Draw offered', `${sideName(pos.side)} offers a draw. ${sideName(-pos.side)}, do you accept?`, 'Accept Draw', 'Play On')) return;
    G.result = { reason: 'agreement', winner: 0 }; persist(); return gameOver();
  }
  drawOfferPly = hist.length; refreshButtons();
  busy = true; setStatus(`${oppInfo().name} considers your offer…`);
  const my = gen;
  const r = await think({ fen: G.fen || K.START, moves: G.moves, time: 700 });
  busy = false;
  if (my !== gen || r.cancelled) return;
  const aiScore = -(r.score || 0); // r.score is from your side (you're to move)
  const accept = aiScore <= -70 || (aiScore <= 25 && hist.length >= 60) || (G.opp === 'tucker' && aiScore <= 0);
  if (accept) { quip('accept'); G.result = { reason: 'agreement', winner: 0 }; persist(); gameOver(); }
  else { quip('decline'); refreshAll(); }
}
function movesText() {
  const d = new Date(G.started || Date.now()), ds = d.toLocaleDateString('en-US', { year: 'numeric', month: 'long', day: 'numeric' });
  const bName = G.mode === 'pvp' ? 'Player one' : G.youBlack ? 'You' : oppInfo().full, rName = G.mode === 'pvp' ? 'Player two' : G.youBlack ? oppInfo().full : 'You';
  let s = `Checkers at the Hollow Creek General Store, ${ds}\nBlack: ${bName}\nRed: ${rName}\n${G.forced === false ? 'Jumping optional (house rules)\n' : ''}\n`;
  const parts = []; hist.forEach((h, i) => { if (i % 2 === 0) parts.push((i / 2 + 1) + '.'); parts.push(h.n); });
  let line = ''; for (const p of parts) { if ((line + ' ' + p).length > 72) { s += line.trim() + '\n'; line = ''; } line += ' ' + p; }
  s += line.trim() + '\n';
  if (G.result) s += '\n' + resultLine().replace(sideName(BLACK), 'Black').replace(sideName(RED), 'Red') + '\n';
  return s;
}
async function copyMoves() {
  const t = movesText(); let ok = false;
  try { await navigator.clipboard.writeText(t); ok = true; } catch (e) { const ta = IR.el('textarea', '', document.body); ta.value = t; ta.select(); try { ok = document.execCommand('copy'); } catch (e2) { } ta.remove(); }
  banner(ok ? 'Moves copied' : 'Could not copy', ok ? 'Squares numbered 1 to 32, Black first' : '');
}
function banner(t, sub) { const b = $('#banner'); b.innerHTML = t + (sub ? `<small>${sub}</small>` : ''); b.classList.add('show'); clearTimeout(banner.t); banner.t = setTimeout(() => b.classList.remove('show'), 1600); }

/* ---------- title screen ---------- */
const setup = SD.setup;
function chips(id, obj, key, after) {
  const box = $(id); const sync = () => box.querySelectorAll('button').forEach(b => b.classList.toggle('on', b.dataset.v === obj[key]));
  box.querySelectorAll('button').forEach(b => b.onclick = () => { obj[key] = b.dataset.v; save.write(); sync(); sfx.click(); after && after(); }); sync();
}
function colorLabels() {
  const n = ART.SETS[S.pieces].names;
  const b = $('#optColor').querySelectorAll('button'); b[0].textContent = `${n[1]} (moves first)`; b[2].textContent = n[-1];
}
function buildFolks() {
  const box = $('#folks'); box.innerHTML = '';
  ART.FOLKS.forEach(f => {
    const r = ST.per[f.id] || { w: 0, d: 0, l: 0 };
    const d = IR.el('div', 'folk' + (setup.opp === f.id ? ' on' : ''), box);
    const c = document.createElement('canvas'); c.width = c.height = 200; c.getContext('2d').drawImage(avatarFor(f), 0, 0); d.appendChild(c);
    IR.el('b', '', d, f.full);
    IR.el('span', 'lv', d, `${f.rating} <span class="stars">${'●'.repeat(f.level)}</span>`);
    IR.el('span', 'rec', d, r.w + r.d + r.l ? `You: ${r.w}W ${r.d}D ${r.l}L` : 'Not played yet');
    d.onclick = () => { setup.opp = f.id; save.write(); sfx.click(); buildFolks(); };
  });
  const f = ART.FOLKS.find(x => x.id === setup.opp) || ART.FOLKS[0];
  $('#folkTag').textContent = setup.mode === 'ai' ? `“${f.tag}”` : 'Pass and play on one computer. The board turns to face whoever is moving.';
  box.classList.toggle('off', setup.mode !== 'ai');
  $('#rowColor').style.visibility = setup.mode === 'ai' ? 'visible' : 'hidden';
}
chips('#optMode', setup, 'mode', buildFolks);
chips('#optColor', setup, 'color');
chips('#optForced', setup, 'forced');
chips('#optBoard', S, 'board', () => { buildStatic(); dirty(); });
chips('#optPieces', S, 'pieces', () => { colorLabels(); refreshAll(); });
chips('#optDots', S, 'dots', dirty);
chips('#optNums', S, 'nums', () => { buildStatic(); dirty(); });
chips('#optAnim', S, 'anim');
chips('#optTurn', S, 'turn');
colorLabels();
function showTitle() {
  buildFolks();
  const can = SD.game && !SD.game.result && SD.game.moves && SD.game.moves.length;
  $('#bContinue').classList.toggle('hidden', !can);
  IR.panel.hide('over');
  IR.panel.show('title');
}
$('#bStart').onclick = () => { audio.unlock(); newGame(setup); };
$('#bContinue').onclick = () => { audio.unlock(); sfx.click(); resumeGame(); };
$('#bNew').onclick = () => { sfx.click(); showTitle(); };
$('#bUndo').onclick = undo;
$('#bHint').onclick = hint;
$('#bFlip').onclick = () => { sfx.click(); setFlip(!flipped); persist(); };
$('#bDraw').onclick = offerDraw;
$('#bResign').onclick = resign;
$('#bRematch').onclick = () => { const s = Object.assign({}, setup, { mode: G.mode, opp: G.opp, color: G.mode === 'ai' ? (G.youBlack ? 'red' : 'black') : 'black', forced: G.forced === false ? 'off' : 'on' }); newGame(s); };
$('#bOverNew').onclick = () => { sfx.click(); showTitle(); };
$('#bReview').onclick = () => { IR.panel.hide('over'); setView(0); };
$('#bCopy').onclick = copyMoves;
$('#bSettings').onclick = $('#bSettings2').onclick = () => { sfx.click(); IR.panel.show('settings'); };
$('#bHow').onclick = () => { sfx.click(); IR.panel.show('rules'); };
$('#bCredits').onclick = $('#bCredits2').onclick = async () => { $('#changelog').innerHTML = IR.mdChangelog(await IR.loadChangelog()); IR.panel.show('credits'); };
document.querySelectorAll('.nav button').forEach(b => b.onclick = () => {
  const n = hist.length, v = live() ? n : view;
  setView({ start: 0, prev: v - 1, next: v + 1, end: n }[b.dataset.nav]);
});
IR.panel.wire();
addEventListener('keydown', e => {
  if (!$('#title').classList.contains('hidden') || !$('#confirm').classList.contains('hidden')) { if (e.key === 'Enter' && !$('#title').classList.contains('hidden')) $('#bStart').click(); return; }
  const k = e.key.toLowerCase();
  if ((e.ctrlKey && k === 'z') || k === 'backspace') { e.preventDefault(); undo(); }
  else if (k === 'h') hint();
  else if (k === 'f') $('#bFlip').click();
  else if (k === 'n') showTitle();
  else if (k === 'escape' && sel >= 0) { clearSel(); refreshStatus(); }
  else if (G && (k === 'arrowleft' || k === 'arrowright' || k === 'home' || k === 'end')) {
    e.preventDefault(); const n = hist.length, v = live() ? n : view;
    setView({ arrowleft: v - 1, arrowright: v + 1, home: 0, end: n }[k]);
  }
});

/* ---------- start ---------- */
IR.Stage.init($('#stage'), s => {
  const r = IR.clamp(Math.round(s * dpr() * 100) / 100, 1, 4);
  if (Math.abs(r - res) > .02) {
    res = r;
    ART.scene($('#scene'), W, H, Math.min(2.5, r), { lx: BX - 16, ly: BY - 14, lw: FR + 32, lh: FR + 28 });
    boardCv.width = Math.round(FR * res); boardCv.height = Math.round(FR * res);
    buildStatic(); dirty();
    refreshPlaques();
  }
  fx.resize(W, H, Math.min(2, s * dpr()));
});

if (SD.game && SD.game.moves) { G = SD.game; rebuild(); flipped = !!G.flipped; buildStatic(); refreshAll(); }
else { G = { fen: K.START, moves: [], mode: 'ai', opp: setup.opp, youBlack: true, forced: true, result: null }; flipped = true; rebuild(); buildStatic(); refreshAll(); G = null; refreshButtons(); }
showTitle();

// test hooks
window.__checkers = { get G() { return G; }, get pos() { return pos; }, get busy() { return busy; }, get hist() { return hist; }, get sel() { return sel; }, get hops() { return hops; }, get anims() { return anims.length; }, K, newGame, sqXY, ST, SD, undo, hint, setView, resign: () => resign(), movesText, get legal() { return legalNow.map(m => pos.notation(m)); },
  setFen: f => { G.fen = f; G.moves = []; G.result = null; rebuild(); refreshAll(); },
  play: n => { const m = legalNow.find(x => pos.notation(x) === n); if (m) playMove(m); return !!m; } };
})();
