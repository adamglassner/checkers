/* Checkers - art. The boards, every piece set, the general store, the
   regulars' portraits and the game's icon are drawn here in code at whatever
   resolution the screen needs, so everything stays sharp from 720p to 4K.
   (c) 2026 Ironite */
'use strict';
const ART = (() => {
const D = IR.draw;
const rng = (seed) => { let s = seed >>> 0 || 1; return () => { s ^= s << 13; s >>>= 0; s ^= s >> 17; s ^= s << 5; s >>>= 0; return s / 4294967296; }; };
const canvas = (w, h) => { const c = document.createElement('canvas'); c.width = Math.max(1, Math.ceil(w)); c.height = Math.max(1, Math.ceil(h)); return [c, c.getContext('2d')]; };

/* ------------------------------------------------------------ piece sets
   Each side: top face colours [light, mid, dark], rim colours, groove ink. */
const SETS = {
  classic: { name: 'Red & Black', names: { 1: 'Black', '-1': 'Red' },
    1: { face: ['#4a4a50', '#1d1d22', '#08080a'], rim: ['#2a2a30', '#0e0e12', '#000'], groove: 'rgba(0,0,0,.55)', hi: 'rgba(255,255,255,.55)', crown: '#e3bd5a' },
    '-1': { face: ['#e8574a', '#b51f1c', '#6e0c0c'], rim: ['#9c1a18', '#6a0d0c', '#3a0505'], groove: 'rgba(70,0,0,.5)', hi: 'rgba(255,235,225,.6)', crown: '#f5d277' } },
  wood: { name: 'Walnut & Maple', names: { 1: 'Walnut', '-1': 'Maple' },
    1: { face: ['#8a5a36', '#5a361c', '#2f1a0b'], rim: ['#5e3a1e', '#3e2412', '#1e1006'], groove: 'rgba(25,10,2,.55)', hi: 'rgba(255,220,180,.35)', crown: '#e9c46a', grain: 'rgba(30,12,2,.35)', inlay: '#e8cfa0' },
    '-1': { face: ['#f6e2b8', '#e2c28a', '#b48b52'], rim: ['#d3ae74', '#a98050', '#6e4e2a'], groove: 'rgba(110,70,25,.45)', hi: 'rgba(255,255,240,.6)', crown: '#7a3e16', grain: 'rgba(150,100,45,.35)', inlay: '#5a3418' } },
  caps: { name: 'Bottle Caps', names: { 1: 'Blue', '-1': 'Red' }, caps: true,
    1: { face: ['#5f86c8', '#23488e', '#0d214a'], rim: ['#c9ccd2', '#7d828c', '#3c4048'], groove: 'rgba(0,10,40,.45)', hi: 'rgba(255,255,255,.6)', crown: '#f4d36b', star: '#f4efe2' },
    '-1': { face: ['#f2705e', '#c2241d', '#6c0c0a'], rim: ['#d6d8dc', '#8a8e96', '#43464e'], groove: 'rgba(60,0,0,.45)', hi: 'rgba(255,255,255,.6)', crown: '#fff1b8', star: '#fbf3df' } },
};

// A crown emblem centred at 0,0 roughly 1 unit wide.
function crown(x, col, dark) {
  x.beginPath();
  x.moveTo(-.5, .28); x.lineTo(-.5, -.12); x.lineTo(-.27, .06); x.lineTo(0, -.34); x.lineTo(.27, .06); x.lineTo(.5, -.12); x.lineTo(.5, .28); x.closePath();
  x.fillStyle = col; x.fill(); x.lineWidth = .07; x.strokeStyle = dark; x.lineJoin = 'round'; x.stroke();
  x.fillStyle = dark; x.fillRect(-.5, .18, 1, .1);
  [[-.5, -.12], [0, -.34], [.5, -.12]].forEach(([a, b]) => { x.beginPath(); x.arc(a, b, .085, 0, Math.PI * 2); x.fillStyle = col; x.fill(); x.stroke(); });
}

function star(x, r, col) {
  x.beginPath();
  for (let i = 0; i < 10; i++) { const a = -Math.PI / 2 + i * Math.PI / 5, rr = i % 2 ? r * .45 : r; x.lineTo(Math.cos(a) * rr, Math.sin(a) * rr); }
  x.closePath(); x.fillStyle = col; x.fill();
}

/* One disc. cx, cy = centre of the top face; rx, ry = top ellipse; th = thickness. */
function disc(x, set, side, cx, cy, rx, ry, th, seed) {
  const P = SETS[set][side], caps = SETS[set].caps, r = rng(seed);
  // side band
  x.beginPath(); x.ellipse(cx, cy + th, rx, ry, 0, 0, Math.PI); x.lineTo(cx - rx, cy); x.ellipse(cx, cy, rx, ry, 0, Math.PI, 0, true); x.closePath();
  x.fillStyle = D.lin(x, cx - rx, 0, cx + rx, 0, [P.rim[2], P.rim[0], P.rim[1], P.rim[2]]); x.fill();
  if (caps) { // crimped skirt
    for (let i = 0; i <= 24; i++) { const a = Math.PI * i / 24, px = cx + Math.cos(a) * rx, py = cy + Math.sin(a) * ry; D.line(x, px, py + th * .1, px, py + th * .95, i % 2 ? 'rgba(255,255,255,.35)' : 'rgba(0,0,0,.35)', rx * .05); }
  } else {
    for (const f of [.35, .7]) { x.beginPath(); x.ellipse(cx, cy + th * f, rx, ry, 0, .1, Math.PI - .1); x.strokeStyle = 'rgba(0,0,0,.35)'; x.lineWidth = rx * .02; x.stroke(); }
  }
  // top face
  x.save();
  x.beginPath(); x.ellipse(cx, cy, rx, ry, 0, 0, Math.PI * 2); x.clip();
  x.fillStyle = D.rad(x, cx - rx * .3, cy - ry * .45, rx * .05, rx * 1.25, P.face); x.fillRect(cx - rx, cy - ry, rx * 2, ry * 2);
  if (P.grain) { // wood grain
    for (let i = 0; i < 26; i++) {
      const y0 = cy - ry + r() * ry * 2, a = ry * (.03 + r() * .05), f = 3 + r() * 5, ph = r() * 6;
      x.beginPath(); for (let k = 0; k <= 20; k++) { const px = cx - rx + k * rx / 10, py = y0 + Math.sin(k / 20 * f + ph) * a; k ? x.lineTo(px, py) : x.moveTo(px, py); }
      x.strokeStyle = P.grain; x.lineWidth = rx * (.008 + r() * .02); x.stroke();
    }
  }
  x.restore();
  if (caps) {
    // crimped edge ring + printed star
    for (let i = 0; i < 21; i++) { const a = i / 21 * Math.PI * 2; D.ell(x, cx + Math.cos(a) * rx * .93, cy + Math.sin(a) * ry * .93, rx * .1, ry * .1, i % 2 ? 'rgba(255,255,255,.18)' : 'rgba(0,0,0,.22)'); }
    x.beginPath(); x.ellipse(cx, cy, rx * .82, ry * .82, 0, 0, Math.PI * 2); x.strokeStyle = 'rgba(0,0,0,.35)'; x.lineWidth = rx * .03; x.stroke();
    x.beginPath(); x.ellipse(cx, cy, rx * .7, ry * .7, 0, 0, Math.PI * 2); x.strokeStyle = P.star; x.lineWidth = rx * .045; x.stroke();
    x.save(); x.translate(cx, cy); x.scale(1, ry / rx); star(x, rx * .42, P.star); x.restore();
  } else {
    // turned grooves
    for (const f of [.9, .74, .4]) {
      x.beginPath(); x.ellipse(cx, cy + ry * .02, rx * f, ry * f, 0, 0, Math.PI * 2); x.strokeStyle = P.groove; x.lineWidth = rx * .035; x.stroke();
      x.beginPath(); x.ellipse(cx, cy + ry * .05, rx * f, ry * f, 0, .15, Math.PI - .15); x.strokeStyle = 'rgba(255,255,255,.14)'; x.lineWidth = rx * .02; x.stroke();
    }
    if (P.inlay) { x.beginPath(); x.ellipse(cx, cy, rx * .57, ry * .57, 0, 0, Math.PI * 2); x.strokeStyle = P.inlay; x.lineWidth = rx * .05; x.stroke(); }
  }
  // rim highlight and gloss
  x.beginPath(); x.ellipse(cx, cy, rx * .985, ry * .985, 0, Math.PI * 1.05, Math.PI * 1.75); x.strokeStyle = P.hi; x.lineWidth = rx * .045; x.lineCap = 'round'; x.stroke();
  x.save(); x.beginPath(); x.ellipse(cx, cy, rx, ry, 0, 0, Math.PI * 2); x.clip();
  x.fillStyle = D.rad(x, cx - rx * .35, cy - ry * .55, 0, rx * .55, ['rgba(255,255,255,.32)', 'rgba(255,255,255,0)']); x.fillRect(cx - rx, cy - ry, rx * 2, ry * 2);
  x.restore();
  x.beginPath(); x.ellipse(cx, cy, rx, ry, 0, 0, Math.PI * 2); x.strokeStyle = 'rgba(0,0,0,.45)'; x.lineWidth = rx * .02; x.stroke();
}

const pieceCache = new Map();
// code: 1 black man, 2 black king, -1 red man, -2 red king. Square canvas `size` px; the piece sits in the lower part.
function piece(code, size, set = 'classic') {
  size = Math.max(8, Math.round(size));
  const key = set + code + ':' + size;
  let c = pieceCache.get(key); if (c) return c;
  let x; [c, x] = canvas(size, size);
  const s = size, side = code > 0 ? 1 : -1, king = Math.abs(code) === 2;
  const rx = s * .39, ry = s * .33, th = s * .085;
  const levels = king ? 2 : 1, top = s * .5 + (king ? -th * .55 : th * .05) - th * (levels - 1) * .5;
  const lift = th * 1.02;
  // contact shadow
  x.save(); x.filter = `blur(${Math.max(1, s * .03)}px)`;
  D.ell(x, s * .5 + s * .03, top + (levels - 1) * lift + th + s * .03, rx * 1.02, ry * 1.0, 'rgba(0,0,0,.5)'); x.restore();
  for (let k = levels - 1; k >= 0; k--) disc(x, set, side, s * .5, top + k * lift - (k ? 0 : 0), rx, ry, th, 7 + k * 13 + (side > 0 ? 0 : 99));
  if (king) {
    const P = SETS[set][side];
    x.save(); x.translate(s * .5, top + ry * .05); x.scale(rx * .78, rx * .78 * (ry / rx) * 1.05);
    x.shadowColor = 'rgba(0,0,0,.45)'; x.shadowBlur = s * .02; x.shadowOffsetY = s * .01;
    crown(x, P.crown, 'rgba(60,35,5,.75)'); x.restore();
  }
  pieceCache.set(key, c);
  if (pieceCache.size > 400) pieceCache.delete(pieceCache.keys().next().value);
  return c;
}

/* ---------------------------------------------------------------- boards */
const BOARDS = {
  barrel: { name: 'Barrelhead', light: ['#d9b47c', '#c49a5e'], dark: ['#2a1e16', '#1a120c'], frame: 'iron', coord: '#f1dcae', num: 'rgba(240,220,180,.42)' },
  parlor: { name: 'Parlor Red', light: ['#b5332a', '#8e221c'], dark: ['#1c1715', '#0f0c0b'], frame: 'mahogany', coord: '#f2d78e', num: 'rgba(240,210,150,.4)' },
  tourney: { name: 'Tournament Green', light: ['#efe2c0', '#d9c79c'], dark: ['#2f6a43', '#1f4d30'], frame: 'walnut', coord: '#f3e4be', num: 'rgba(235,245,225,.45)' },
};
const boardCache = {};
function boardSurface(theme, px) {
  const k = theme + px; if (boardCache[k]) return boardCache[k];
  const T = BOARDS[theme], [c, x] = canvas(px, px), s = px / 8, r = rng(px * 7 + theme.length);
  if (theme === 'barrel') {
    // oak staves run top to bottom under the whole board, then the dark squares are painted on
    const staves = 9, sw = px / staves;
    for (let i = 0; i < staves; i++) {
      const tone = .85 + r() * .2;
      x.fillStyle = D.lin(x, i * sw, 0, (i + 1) * sw, 0, [`rgba(${Math.round(196 * tone)},${Math.round(150 * tone)},${Math.round(92 * tone)},1)`, `rgba(${Math.round(220 * tone)},${Math.round(178 * tone)},${Math.round(118 * tone)},1)`, `rgba(${Math.round(180 * tone)},${Math.round(136 * tone)},${Math.round(82 * tone)},1)`]);
      x.fillRect(i * sw, 0, sw, px);
      for (let g = 0; g < 26; g++) {
        const x0 = i * sw + r() * sw, a = 1 + r() * 4, f = .004 + r() * .01, ph = r() * 7;
        x.beginPath(); for (let y = 0; y <= px; y += px / 40) { const xx = x0 + Math.sin(y * f + ph) * a; y ? x.lineTo(xx, y) : x.moveTo(xx, y); }
        x.strokeStyle = `rgba(90,55,20,${.08 + r() * .14})`; x.lineWidth = .5 + r() * 1.6; x.stroke();
      }
      x.fillStyle = 'rgba(40,20,5,.55)'; x.fillRect(i * sw - 1, 0, 2, px);
      x.fillStyle = 'rgba(255,230,190,.18)'; x.fillRect(i * sw + 1, 0, 1, px);
    }
    for (let i = 0; i < 64; i++) {
      const f = i & 7, rk = i >> 3; if (((f + rk) & 1) === 0) continue;
      // painted squares, slightly worn at the edges
      x.fillStyle = D.lin(x, f * s, rk * s, f * s + s, rk * s + s, T.dark); x.fillRect(f * s + 1, rk * s + 1, s - 2, s - 2);
      for (let k = 0; k < 40; k++) { x.fillStyle = `rgba(200,160,110,${.05 + r() * .12})`; const ex = r() < .5; x.fillRect(f * s + (ex ? r() * s : (r() < .5 ? 0 : s - 3)), rk * s + (ex ? (r() < .5 ? 0 : s - 3) : r() * s), 2 + r() * 4, 2 + r() * 3); }
      for (let k = 0; k < 8; k++) { x.strokeStyle = `rgba(255,240,210,${.03 + r() * .05})`; x.lineWidth = 1; const yy = rk * s + r() * s; x.beginPath(); x.moveTo(f * s + 2, yy); x.lineTo(f * s + s - 2, yy + (r() - .5) * 4); x.stroke(); }
    }
  } else {
    for (let i = 0; i < 64; i++) {
      const f = i & 7, rk = i >> 3, dark = (f + rk) & 1, col = dark ? T.dark : T.light;
      x.fillStyle = D.lin(x, f * s, rk * s, f * s + s, rk * s + s, col); x.fillRect(f * s, rk * s, s, s);
      if (theme === 'parlor') { // lacquer with a soft sheen and fine crackle
        x.fillStyle = D.lin(x, f * s, rk * s, f * s + s * .6, rk * s + s * .6, ['rgba(255,255,255,.10)', 'rgba(255,255,255,0)']); x.fillRect(f * s, rk * s, s, s);
        for (let k = 0; k < 5; k++) { x.strokeStyle = `rgba(0,0,0,${.05 + r() * .06})`; x.lineWidth = .7; x.beginPath(); let px0 = f * s + r() * s, py0 = rk * s + r() * s; x.moveTo(px0, py0); for (let j = 0; j < 4; j++) { px0 += (r() - .5) * s * .3; py0 += (r() - .5) * s * .3; x.lineTo(px0, py0); } x.stroke(); }
      } else { // painted card stock: fine speckle
        for (let k = 0; k < 50; k++) { x.fillStyle = r() < .5 ? 'rgba(255,255,255,.05)' : 'rgba(0,0,0,.05)'; x.fillRect(f * s + r() * s, rk * s + r() * s, 1.5, 1.5); }
      }
    }
    if (theme === 'parlor') { x.strokeStyle = 'rgba(230,190,100,.55)'; x.lineWidth = Math.max(1, px * .0016); for (let i = 1; i < 8; i++) { x.beginPath(); x.moveTo(i * s, 0); x.lineTo(i * s, px); x.moveTo(0, i * s); x.lineTo(px, i * s); x.stroke(); } }
  }
  // soft light from the upper left
  x.fillStyle = D.rad(x, px * .3, px * .25, 10, px * 1.05, ['rgba(255,235,200,.10)', 'rgba(0,0,0,0)', 'rgba(0,0,0,.22)']); x.fillRect(0, 0, px, px);
  boardCache[k] = c; return c;
}

// The frame (with coordinates around the edge, matching the side shown at the bottom).
function boardFrame(theme, px, inner, flipped) {
  const T = BOARDS[theme], [c, x] = canvas(px, px), m = (px - inner) / 2, r = rng(px + theme.length * 3);
  x.save(); D.rr(x, 0, 0, px, px, m * .3); x.clip();
  if (T.frame === 'iron') {
    // oak rim with an iron hoop and rivets
    x.fillStyle = D.lin(x, 0, 0, px, px, ['#7a5230', '#5a3a1e', '#3c2410']); x.fillRect(0, 0, px, px);
    for (let i = 0; i < 160; i++) { x.strokeStyle = `rgba(30,15,5,${.08 + r() * .12})`; x.lineWidth = .6 + r(); x.beginPath(); const yy = r() * px; x.moveTo(0, yy); x.lineTo(px, yy + (r() - .5) * 6); x.stroke(); }
    const band = m * .5, off = m * .22;
    x.lineWidth = band; x.strokeStyle = D.lin(x, 0, 0, px, px, ['#5d5f63', '#2c2e31', '#46484c', '#1c1d1f']);
    x.strokeRect(off + band / 2, off + band / 2, px - 2 * off - band, px - 2 * off - band);
    x.lineWidth = Math.max(1, m * .04); x.strokeStyle = 'rgba(255,255,255,.18)'; x.strokeRect(off + 1, off + 1, px - 2 * off - 2, px - 2 * off - 2);
    x.strokeStyle = 'rgba(0,0,0,.6)'; x.strokeRect(off + band, off + band, px - 2 * (off + band), px - 2 * (off + band));
    // rust bloom
    for (let i = 0; i < 90; i++) { x.fillStyle = `rgba(${130 + r() * 40},${60 + r() * 20},20,${.08 + r() * .14})`; const a = r() * 4, t = r() * (px - 2 * off); const p = [[off + t, off + r() * band], [px - off - r() * band, off + t], [off + t, px - off - r() * band], [off + r() * band, off + t]][Math.floor(a)]; D.ell(x, p[0], p[1], 1 + r() * m * .08, 1 + r() * m * .05, x.fillStyle); }
    const nr = 9;
    for (let i = 0; i < nr; i++) {
      const t = off + band / 2 + (px - 2 * off - band) * (i + .5) / nr;
      for (const [rx, ry] of [[t, off + band / 2], [t, px - off - band / 2], [off + band / 2, t], [px - off - band / 2, t]]) {
        D.ell(x, rx, ry, m * .085, m * .085, D.rad(x, rx - m * .02, ry - m * .02, 0, m * .1, ['#9a9ca0', '#4a4c50', '#1a1b1d']));
      }
    }
  } else {
    const cols = T.frame === 'mahogany' ? ['#6a2416', '#4a160c', '#2a0a04'] : ['#5e3d22', '#3f2712', '#24150a'];
    x.fillStyle = D.lin(x, 0, 0, px, px, cols); x.fillRect(0, 0, px, px);
    for (let i = 0; i < 220; i++) { x.strokeStyle = `rgba(${r() < .5 ? '0,0,0' : '255,200,150'},${.04 + r() * .08})`; x.lineWidth = .5 + r() * 1.5; x.beginPath(); const yy = r() * px; x.moveTo(0, yy); x.bezierCurveTo(px * .3, yy + (r() - .5) * 14, px * .7, yy + (r() - .5) * 14, px, yy + (r() - .5) * 8); x.stroke(); }
    // mitred corners
    x.strokeStyle = 'rgba(0,0,0,.4)'; x.lineWidth = 2; [[0, 0, m, m], [px, 0, px - m, m], [0, px, m, px - m], [px, px, px - m, px - m]].forEach(([a, b, cc, d]) => { x.beginPath(); x.moveTo(a, b); x.lineTo(cc, d); x.stroke(); });
    if (T.frame === 'mahogany') { x.strokeStyle = '#d4a94e'; x.lineWidth = Math.max(1.5, m * .05); x.strokeRect(m * .26, m * .26, px - m * .52, px - m * .52); x.lineWidth = Math.max(1, m * .025); x.strokeRect(m * .36, m * .36, px - m * .72, px - m * .72); }
    x.fillStyle = 'rgba(255,220,170,.12)'; x.fillRect(0, 0, px, m * .1); x.fillRect(0, 0, m * .1, px);
  }
  x.restore();
  // inner bevel
  x.fillStyle = 'rgba(0,0,0,.55)'; x.fillRect(m - m * .12, m - m * .12, inner + m * .24, inner + m * .24);
  void flipped;
  return c;
}

/* ---------------------------------------------------- the general store */
function scene(cv, W, H, res, layout) {
  cv.width = Math.round(W * res); cv.height = Math.round(H * res);
  cv.style.width = W + 'px'; cv.style.height = H + 'px';
  const x = cv.getContext('2d'); x.scale(res, res);
  const r = rng(1931);
  // worn heart-pine planks
  x.fillStyle = D.lin(x, 0, 0, 0, H, ['#6b4425', '#58371c', '#3e2511']); x.fillRect(0, 0, W, H);
  const plank = 132;
  for (let py = -20; py < H; py += plank) {
    x.fillStyle = `rgba(${r() < .5 ? '255,200,140' : '0,0,0'},${.03 + r() * .05})`; x.fillRect(0, py, W, plank);
    for (let i = 0; i < 60; i++) {
      const y0 = py + r() * plank, a = 2 + r() * 5, f = .002 + r() * .006, ph = r() * 9;
      x.beginPath(); for (let px = 0; px <= W; px += 20) { const yy = y0 + Math.sin(px * f + ph) * a; px ? x.lineTo(px, yy) : x.moveTo(px, yy); }
      x.strokeStyle = r() < .6 ? `rgba(40,20,5,${.06 + r() * .1})` : `rgba(255,210,150,${.04 + r() * .05})`; x.lineWidth = .6 + r() * 1.8; x.stroke();
    }
    // a knot or two, and square nails at the joists
    for (let k = 0; k < 2; k++) { const kx = r() * W, ky = py + 20 + r() * (plank - 40); D.ell(x, kx, ky, 10 + r() * 10, 5 + r() * 4, 'rgba(50,25,8,.35)'); D.ell(x, kx, ky, 4, 2.5, 'rgba(30,14,4,.5)'); }
    for (let nx = 90; nx < W; nx += 420) { x.fillStyle = 'rgba(25,20,18,.7)'; x.fillRect(nx + (py % 3) * 7, py + 14, 5, 5); x.fillRect(nx + (py % 3) * 7, py + plank - 20, 5, 5); }
    x.fillStyle = 'rgba(20,10,3,.6)'; x.fillRect(0, py - 1.5, W, 3);
    x.fillStyle = 'rgba(255,215,160,.08)'; x.fillRect(0, py + 1.5, W, 1);
  }
  // red gingham cloth under the board
  if (layout) {
    const { lx, ly, lw, lh } = layout;
    x.save(); x.shadowColor = 'rgba(0,0,0,.55)'; x.shadowBlur = 22; x.shadowOffsetY = 6; x.fillStyle = '#efe6d6'; x.fillRect(lx, ly, lw, lh); x.restore();
    x.save(); x.beginPath(); x.rect(lx, ly, lw, lh); x.clip();
    x.fillStyle = '#f3ebdc'; x.fillRect(lx, ly, lw, lh);
    const g = 26;
    for (let i = 0; i * g < lw + g; i++) { x.fillStyle = 'rgba(176,32,36,.42)'; x.fillRect(lx + i * g * 2, ly, g, lh); }
    for (let j = 0; j * g < lh + g; j++) { x.fillStyle = 'rgba(176,32,36,.42)'; x.fillRect(lx, ly + j * g * 2, lw, g); }
    for (let i = 0; i < 2600; i++) { x.fillStyle = r() < .5 ? 'rgba(255,255,255,.06)' : 'rgba(80,0,0,.05)'; x.fillRect(lx + r() * lw, ly + r() * lh, 2, 1); }
    // soft folds
    for (const fy of [.33, .7]) { x.fillStyle = D.lin(x, 0, ly + lh * fy - 14, 0, ly + lh * fy + 14, ['rgba(0,0,0,0)', 'rgba(0,0,0,.12)', 'rgba(255,255,255,.12)', 'rgba(0,0,0,0)']); x.fillRect(lx, ly + lh * fy - 14, lw, 28); }
    x.restore();
  }
  // warm stove glow from the lower left, lamp from above, vignette
  x.fillStyle = D.rad(x, -60, H * .95, 20, W * .55, ['rgba(255,130,40,.28)', 'rgba(255,110,30,.08)', 'rgba(0,0,0,0)']); x.fillRect(0, 0, W, H);
  x.fillStyle = D.rad(x, W * .3, H * .05, 40, W * .9, ['rgba(255,215,150,.16)', 'rgba(255,190,110,.04)', 'rgba(0,0,0,.45)']); x.fillRect(0, 0, W, H);
}

/* ------------------------------------------------------------ the regulars */
const storeBg = (a, b, seed) => x => {
  x.fillStyle = D.lin(x, 0, 0, 0, IR.RH, [a, b]); x.fillRect(0, 0, 1, IR.RH);
  const r = rng(seed);
  // shelves of jars and tins
  for (let row = 0; row < 3; row++) {
    const y0 = .1 + row * .42;
    x.fillStyle = 'rgba(60,32,12,.6)'; x.fillRect(0, y0 + .3, 1, .03);
    let bx = .01;
    while (bx < 1) {
      const kind = r(), w = .07 + r() * .05;
      if (kind < .45) { // glass jar of candy
        const h = .18 + r() * .06, hue = [[220, 80, 80], [240, 200, 90], [120, 190, 120], [230, 140, 190], [130, 170, 230]][Math.floor(r() * 5)];
        x.fillStyle = `rgba(${hue.join(',')},.35)`; D.rr(x, bx, y0 + .3 - h, w - .01, h, .015); x.fill();
        x.fillStyle = 'rgba(255,255,255,.18)'; x.fillRect(bx + w * .15, y0 + .3 - h * .9, w * .1, h * .75);
        x.fillStyle = 'rgba(70,40,20,.6)'; x.fillRect(bx - .003, y0 + .3 - h - .02, w - .004, .025);
      } else { // painted tin
        const h = .12 + r() * .1, hue = [[150, 40, 30], [40, 80, 60], [190, 140, 50], [40, 60, 110]][Math.floor(r() * 4)];
        x.fillStyle = `rgba(${hue.join(',')},.6)`; x.fillRect(bx, y0 + .3 - h, w - .012, h);
        x.fillStyle = 'rgba(240,220,170,.35)'; x.fillRect(bx, y0 + .3 - h * .65, w - .012, h * .22);
      }
      bx += w;
    }
  }
  x.fillStyle = D.rad(x, .5, .85, .05, .9, ['rgba(255,200,120,.35)', 'rgba(0,0,0,.3)']); x.fillRect(0, 0, 1, IR.RH);
};
const overalls = (col, shirt) => x => {
  x.fillStyle = shirt; x.beginPath(); x.moveTo(-.05, IR.RH); x.lineTo(0, 1.3); x.quadraticCurveTo(.08, 1.12, .3, 1.09); x.lineTo(.7, 1.09); x.quadraticCurveTo(.92, 1.12, 1, 1.3); x.lineTo(1.05, IR.RH); x.closePath(); x.fill();
  for (let i = -2; i < 8; i++) { x.fillStyle = 'rgba(0,0,0,.12)'; x.fillRect(i * .14, 1.09, .05, .6); }
  x.fillStyle = col; x.fillRect(.3, 1.3, .4, .4); x.fillRect(.3, 1.12, .07, .2); x.fillRect(.63, 1.12, .07, .2);
  D.ell(x, .335, 1.3, .02, .02, '#d9c27a'); D.ell(x, .665, 1.3, .02, .02, '#d9c27a');
};
const FOLKS = [
  { id: 'tucker', level: 1, name: 'Tucker', full: 'Little Tucker', rating: 'Beginner', tag: "The grocer's grandson. Sweeps the porch and crowns anything he can.",
    look: { coat: ['#4a6fa8', '#2c4a7a'], hair: '#c47a33', blush: true, skin: '#f3cba6', body: overalls('#3e5f96', '#c7473c'), hat: { type: 'cap', color: '#c7473c', brim: '#a3362e', button: '#f2e3b0' } } },
  { id: 'opal', level: 2, name: 'Miss Opal', full: 'Miss Opal', rating: 'Novice', tag: 'Runs the post office counter. Chatty, friendly and a little reckless.',
    look: { coat: ['#8a6aa8', '#5a4278'], hair: '#d8d2cc', bun: true, glasses: '#b08a3a', lips: '#b0566a', lashes: true, skin: '#f0cdb0', earring: '#f4f0e6', collar: x => D.poly(x, [[.4, 1.09], [.5, 1.19], [.6, 1.09]], '#fbf5ea') } },
  { id: 'earl', level: 3, name: 'Deacon Earl', full: 'Deacon Earl', rating: 'Club player', tag: 'Sunday best every day of the week. Never rushes a move.',
    look: { coat: ['#2e2a2a', '#151212'], hair: '#3a2c22', hairTop: false, mustache: '#cfc7bd', brows: '#bdb4a8', skin: '#8a5a3a', skinShade: '#6e4428', eyes: '#2a1a0e',
      collar: x => { D.poly(x, [[.42, 1.09], [.5, 1.16], [.58, 1.09]], '#f6f2ea'); D.poly(x, [[.44, 1.1], [.5, 1.13], [.44, 1.16]], '#7a1f22'); D.poly(x, [[.56, 1.1], [.5, 1.13], [.56, 1.16]], '#7a1f22'); D.ell(x, .5, 1.13, .012, .012, '#5a1416'); } } },
  { id: 'hattie', level: 4, name: 'Hattie Mae', full: 'Hattie Mae', rating: 'Strong', tag: 'Owns the store. Plays between customers and rarely loses one.',
    look: { coat: ['#4f7a5a', '#2f4e38'], hair: '#5a3a26', curls: true, lips: '#a8484a', lashes: true, skin: '#e2b08c', earring: '#e8c66a',
      body: x => { x.fillStyle = '#f2ead8'; x.beginPath(); x.moveTo(.32, IR.RH); x.lineTo(.36, 1.2); x.lineTo(.64, 1.2); x.lineTo(.68, IR.RH); x.closePath(); x.fill(); D.line(x, .36, 1.2, .3, 1.1, '#f2ead8', .03); D.line(x, .64, 1.2, .7, 1.1, '#f2ead8', .03); },
      hat: { type: 'sunhat', color: '#e6cf98', band: '#b0322c', flower: '#f4f0e6' } } },
  { id: 'doc', level: 5, name: 'Doc Whitfield', full: 'Doc Whitfield', rating: 'Expert', tag: 'The county vet. Calm hands, sharp eyes and sharper traps.',
    look: { coat: ['#6a5a40', '#43382a'], hair: '#8a7a6a', glasses: '#2a2a2a', mustache: '#8a7a6a', skin: '#ebbf9a',
      collar: x => { D.poly(x, [[.42, 1.09], [.5, 1.17], [.58, 1.09]], '#e9e4d8'); D.poly(x, [[.48, 1.11], [.5, 1.3], [.52, 1.11]], '#3a4a6a'); },
      hat: { type: 'fedora', color: '#7a6a52', light: '#9a8a70', band: '#3a2a1a' } } },
  { id: 'gus', level: 6, name: 'Old Gus', full: 'Old Gus', rating: 'Champion', tag: "County champion since anyone can remember. Whittles while he waits.",
    look: { coat: ['#7a3a2a', '#4a2018'], hair: '#eeeae2', brows: '#e2ddd4', beard: '#f0ece4', beardLen: .06, mustache: '#f5f2ea', skin: '#e0a98a', body: overalls('#4a5a6e', '#7a3a2a'),
      hat: { type: 'cap', color: '#5a6a3a', brim: '#46552c', button: '#d9c27a', patch: '#e8dcc0' } } },
];
const BGS = [['#6a4a2a', '#2e1e10'], ['#5a4030', '#2a1a10'], ['#4a3a28', '#22180c'], ['#5a3a24', '#28160a'], ['#3e3a2c', '#1c180e'], ['#4a2e22', '#20120a']];
function avatar(f, size) {
  const [c, x] = canvas(size, size), s = size;
  x.save(); x.beginPath(); x.arc(s / 2, s / 2, s / 2, 0, Math.PI * 2); x.clip();
  x.translate(s * -.12, s * -.44); x.scale(s * 1.24, s * 1.24);
  const i = Math.max(0, FOLKS.indexOf(f));
  IR.portrait(x, Object.assign({ bg: storeBg(BGS[i][0], BGS[i][1], 11 + i * 17) }, f.look || {}));
  x.restore();
  return c;
}

/* --------------------------------------------------------------- icon */
function icon(size) {
  const [c, x] = canvas(size, size), s = size;
  x.scale(s / 1024, s / 1024);
  D.rr(x, 40, 40, 944, 944, 190);
  x.save(); x.clip();
  // a corner of the barrelhead board, tilted
  x.translate(512, 512); x.rotate(-.18); x.translate(-512, -512);
  x.drawImage(boardSurface('barrel', 1400), -190, -190, 1400, 1400);
  x.setTransform(s / 1024, 0, 0, s / 1024, 0, 0);
  x.fillStyle = D.rad(x, 380, 300, 60, 900, ['rgba(255,220,160,.18)', 'rgba(0,0,0,.08)', 'rgba(0,0,0,.62)']); x.fillRect(0, 0, 1024, 1024);
  x.restore();
  D.rr(x, 40, 40, 944, 944, 190); x.strokeStyle = '#4a4c50'; x.lineWidth = 34; x.stroke();
  D.rr(x, 40, 40, 944, 944, 190); x.strokeStyle = 'rgba(255,255,255,.18)'; x.lineWidth = 4; x.stroke();
  D.rr(x, 70, 70, 884, 884, 166); x.strokeStyle = 'rgba(20,10,4,.7)'; x.lineWidth = 6; x.stroke();
  // black man behind, red king in front
  x.drawImage(piece(1, 560), 470, 150, 560, 560);
  x.drawImage(piece(-2, 760), 70, 250, 760, 760);
  return c;
}

return { piece, boardSurface, boardFrame, BOARDS, SETS, scene, FOLKS, avatar, icon };
})();
