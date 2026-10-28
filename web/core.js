/* Ironite card-game core.
   Shared by Go Fish, Blackjack and Texas Hold'em: stage scaling, card art,
   animated card layer, synthesized music and sound, saving, and UI helpers.
   Everything here is original and drawn/synthesized in code. (c) 2026 Ironite */
'use strict';
const IR = (() => {
const IR = {};

/* ------------------------------------------------------------------ utils */
IR.rng = seed => { let s = (seed >>> 0) || 1; return () => { s ^= s << 13; s >>>= 0; s ^= s >> 17; s ^= s << 5; s >>>= 0; return s / 4294967296; }; };
IR.shuffle = (a, rnd = Math.random) => { for (let i = a.length - 1; i > 0; i--) { const j = Math.floor(rnd() * (i + 1)); [a[i], a[j]] = [a[j], a[i]]; } return a; };
IR.sleep = ms => new Promise(r => setTimeout(r, ms / (IR.speed || 1)));
IR.$ = s => document.querySelector(s);
IR.el = (tag, cls, parent, html) => { const e = document.createElement(tag); if (cls) e.className = cls; if (html != null) e.innerHTML = html; if (parent) parent.appendChild(e); return e; };
IR.fmt = n => Math.round(n).toLocaleString('en-US');
IR.clamp = (v, a, b) => Math.max(a, Math.min(b, v));

/* card ids: 0..51, suit = floor(id/13) (0 spades,1 hearts,2 diamonds,3 clubs), rank 1..13 (A=1) */
IR.suit = id => Math.floor(id / 13) % 4;
IR.rank = id => id % 13 + 1;
IR.isRed = id => { const s = IR.suit(id); return s === 1 || s === 2; };
IR.RANKS = ['A', '2', '3', '4', '5', '6', '7', '8', '9', '10', 'J', 'Q', 'K'];
IR.RANK_NAMES = ['Ace', 'Two', 'Three', 'Four', 'Five', 'Six', 'Seven', 'Eight', 'Nine', 'Ten', 'Jack', 'Queen', 'King'];
IR.RANK_PLURAL = ['Aces', 'Twos', 'Threes', 'Fours', 'Fives', 'Sixes', 'Sevens', 'Eights', 'Nines', 'Tens', 'Jacks', 'Queens', 'Kings'];
IR.SUIT_NAMES = ['Spades', 'Hearts', 'Diamonds', 'Clubs'];
IR.SUIT_CH = ['♠', '♥', '♦', '♣'];
IR.cardName = id => IR.RANK_NAMES[IR.rank(id) - 1] + ' of ' + IR.SUIT_NAMES[IR.suit(id)];
IR.short = id => IR.RANKS[IR.rank(id) - 1] + IR.SUIT_CH[IR.suit(id)];

/* ---------------------------------------------------------------- drawing */
const D = IR.draw = {
  rr(x, X, Y, w, h, r) { x.beginPath(); x.moveTo(X + r, Y); x.arcTo(X + w, Y, X + w, Y + h, r); x.arcTo(X + w, Y + h, X, Y + h, r); x.arcTo(X, Y + h, X, Y, r); x.arcTo(X, Y, X + w, Y, r); x.closePath(); },
  ell(x, cx, cy, rx, ry, fill, rot = 0) { x.beginPath(); x.ellipse(cx, cy, Math.abs(rx), Math.abs(ry), rot, 0, Math.PI * 2); if (fill) { x.fillStyle = fill; x.fill(); } },
  poly(x, pts, fill) { x.beginPath(); x.moveTo(pts[0][0], pts[0][1]); for (let i = 1; i < pts.length; i++) x.lineTo(pts[i][0], pts[i][1]); x.closePath(); if (fill) { x.fillStyle = fill; x.fill(); } },
  line(x, a, b, c, d, col, w) { x.beginPath(); x.moveTo(a, b); x.lineTo(c, d); x.strokeStyle = col; x.lineWidth = w; x.lineCap = 'round'; x.stroke(); },
  curve(x, pts, col, w) { x.beginPath(); x.moveTo(pts[0][0], pts[0][1]); for (let i = 1; i + 1 < pts.length; i += 2) x.quadraticCurveTo(pts[i][0], pts[i][1], pts[i + 1][0], pts[i + 1][1]); x.strokeStyle = col; x.lineWidth = w; x.lineCap = 'round'; x.stroke(); },
  lin(x, x0, y0, x1, y1, stops) { const g = x.createLinearGradient(x0, y0, x1, y1); stops.forEach((c, i) => g.addColorStop(i / (stops.length - 1), c)); return g; },
  rad(x, cx, cy, r0, r1, stops) { const g = x.createRadialGradient(cx, cy, r0, cx, cy, r1); stops.forEach((c, i) => g.addColorStop(i / (stops.length - 1), c)); return g; },
  canvas(w, h) { const c = document.createElement('canvas'); c.width = Math.ceil(w); c.height = Math.ceil(h); return [c, c.getContext('2d')]; },
};

IR.suitPath = function (x, s) {
  x.beginPath();
  if (s === 1) {
    x.moveTo(0, .42); x.bezierCurveTo(-.15, .28, -.5, .05, -.5, -.18); x.bezierCurveTo(-.5, -.38, -.32, -.48, -.2, -.48);
    x.bezierCurveTo(-.08, -.48, 0, -.38, 0, -.28); x.bezierCurveTo(0, -.38, .08, -.48, .2, -.48); x.bezierCurveTo(.32, -.48, .5, -.38, .5, -.18);
    x.bezierCurveTo(.5, .05, .15, .28, 0, .42);
  } else if (s === 2) {
    x.moveTo(0, -.5); x.quadraticCurveTo(.16, -.2, .38, 0); x.quadraticCurveTo(.16, .2, 0, .5); x.quadraticCurveTo(-.16, .2, -.38, 0); x.quadraticCurveTo(-.16, -.2, 0, -.5);
  } else if (s === 0) {
    x.moveTo(0, -.5); x.bezierCurveTo(-.15, -.33, -.5, -.12, -.5, .1); x.bezierCurveTo(-.5, .28, -.36, .36, -.24, .36);
    x.bezierCurveTo(-.12, .36, -.05, .3, -.03, .22); x.bezierCurveTo(-.05, .34, -.12, .44, -.22, .5); x.lineTo(.22, .5);
    x.bezierCurveTo(.12, .44, .05, .34, .03, .22); x.bezierCurveTo(.05, .3, .12, .36, .24, .36); x.bezierCurveTo(.36, .36, .5, .28, .5, .1);
    x.bezierCurveTo(.5, -.12, .15, -.33, 0, -.5);
  } else {
    x.arc(0, -.24, .21, 0, Math.PI * 2); x.moveTo(-.03, .08); x.arc(-.24, .08, .21, 0, Math.PI * 2); x.moveTo(.45, .08); x.arc(.24, .08, .21, 0, Math.PI * 2);
    x.moveTo(-.1, -.1); x.lineTo(.1, -.1); x.lineTo(.1, .1); x.lineTo(-.1, .1); x.closePath();
    x.moveTo(-.04, .1); x.bezierCurveTo(-.05, .3, -.12, .44, -.22, .5); x.lineTo(.22, .5); x.bezierCurveTo(.12, .44, .05, .3, .04, .1); x.closePath();
  }
};
IR.drawSuit = function (x, s, cx, cy, size, flip, colors) {
  x.save(); x.translate(cx, cy); if (flip) x.rotate(Math.PI); x.scale(size, size);
  IR.suitPath(x, s);
  const red = s === 1 || s === 2, c = colors || {};
  x.fillStyle = D.lin(x, 0, -.5, 0, .5, red ? (c.red || ['#d23a2f', '#8a1913']) : (c.black || ['#3d3832', '#110f0d']));
  x.fill('nonzero');
  if (c.edge) { x.lineWidth = .025; x.strokeStyle = c.edge; x.stroke(); }
  x.restore();
};

/* --------------------------------------------------------------- portraits
   Unit coordinates: width 1, height RH. A parametric figure so each game
   can dress its own court cards. */
const RH = IR.RH = 1.62;
IR.portrait = function (x, o) {
  const skin = o.skin || '#e6b892', shade = o.skinShade || '#c99470';
  const hair = o.hair || '#6b4426', acc = o.accent || '#8e1f1b', coat = o.coat || ['#3a4f6a', '#1f2b3c'];
  // background
  if (o.bg) o.bg(x); else { x.fillStyle = D.lin(x, 0, 0, 0, RH, ['#f4e8cf', '#d9c6a0']); x.fillRect(0, 0, 1, RH); }
  // long hair behind
  if (o.longHair) { D.ell(x, .5, .9, .25, .3, hair); x.fillStyle = hair; x.fillRect(.25, .9, .5, .38); }
  // body / coat
  x.beginPath(); x.moveTo(-.05, RH); x.lineTo(0, 1.3); x.quadraticCurveTo(.08, 1.12, .3, 1.09); x.lineTo(.7, 1.09); x.quadraticCurveTo(.92, 1.12, 1, 1.3); x.lineTo(1.05, RH); x.closePath();
  x.fillStyle = D.lin(x, 0, 1.1, 1, RH, coat); x.fill();
  if (o.body) o.body(x);
  // neck
  x.fillStyle = shade; x.fillRect(.42, .95, .16, .17);
  if (o.collar) o.collar(x);
  // ears + head
  D.ell(x, .35, .86, .033, .05, shade); D.ell(x, .65, .86, .033, .05, shade);
  D.ell(x, .5, .85, .15, .185, skin);
  D.ell(x, .5, .85, .15, .185, D.rad(x, .45, .8, .02, .22, ['rgba(255,255,255,.16)', 'rgba(120,60,20,.16)']));
  // cheeks
  if (o.blush) { D.ell(x, .42, .9, .03, .02, 'rgba(220,90,90,.3)'); D.ell(x, .58, .9, .03, .02, 'rgba(220,90,90,.3)'); }
  // beard
  if (o.beard) {
    x.beginPath(); x.moveTo(.35, .85); x.quadraticCurveTo(.32, 1.18 + (o.beardLen || 0), .5, 1.25 + (o.beardLen || 0)); x.quadraticCurveTo(.68, 1.18 + (o.beardLen || 0), .65, .85);
    x.quadraticCurveTo(.6, .97, .5, .965); x.quadraticCurveTo(.4, .97, .35, .85); x.fillStyle = o.beard; x.fill();
    for (let i = 0; i < 10; i++) { const bx = .38 + i * .026; D.line(x, bx, .96, bx + (.5 - bx) * .2, 1.12 + Math.sin(i * 1.7) * .04 + (o.beardLen || 0) * .8, 'rgba(0,0,0,.12)', .006); }
  }
  // mustache
  if (o.mustache) { x.beginPath(); x.moveTo(.5, .925); x.quadraticCurveTo(.42, .885, .36, .97); x.quadraticCurveTo(.44, .945, .5, .965); x.quadraticCurveTo(.56, .945, .64, .97); x.quadraticCurveTo(.58, .885, .5, .925); x.fillStyle = o.mustache; x.fill(); }
  // nose
  D.line(x, .5, .85, .492, .9, 'rgba(150,85,55,.6)', .009); D.ell(x, .5, .9, .022, .014, 'rgba(160,90,60,.25)');
  // eyes
  const ey = .82, ec = o.eyes || '#3b2a1a';
  if (o.closedEyes) { x.beginPath(); x.arc(.445, ey - .005, .02, .2, Math.PI - .2); x.strokeStyle = '#3a2412'; x.lineWidth = .008; x.stroke(); x.beginPath(); x.arc(.555, ey - .005, .02, .2, Math.PI - .2); x.stroke(); }
  else {
    D.ell(x, .445, ey, .02, .014, '#fff'); D.ell(x, .555, ey, .02, .014, '#fff');
    D.ell(x, .447, ey + .002, .011, .012, ec); D.ell(x, .557, ey + .002, .011, .012, ec);
    D.ell(x, .443, ey - .003, .004, .004, '#fff'); D.ell(x, .553, ey - .003, .004, .004, '#fff');
  }
  if (o.lashes) { D.line(x, .42, ey - .012, .465, ey - .016, '#2a1a10', .008); D.line(x, .535, ey - .016, .58, ey - .012, '#2a1a10', .008); }
  const bc = o.brows || hair;
  D.line(x, .415, ey - .04, .475, ey - .045, bc, .014); D.line(x, .585, ey - .04, .525, ey - .045, bc, .014);
  // mouth
  if (!o.mustache || o.lips) {
    if (o.lips) { D.ell(x, .5, .955, .036, .015, o.lips); D.line(x, .468, .955, .532, .955, 'rgba(90,20,20,.5)', .004); }
    else { x.beginPath(); x.arc(.5, .925, .045, .35, Math.PI - .35); x.strokeStyle = 'rgba(120,50,40,.75)'; x.lineWidth = .01; x.stroke(); }
  }
  if (o.glasses) { x.strokeStyle = o.glasses; x.lineWidth = .009; x.beginPath(); x.arc(.445, ey, .032, 0, Math.PI * 2); x.stroke(); x.beginPath(); x.arc(.555, ey, .032, 0, Math.PI * 2); x.stroke(); D.line(x, .477, ey, .523, ey, o.glasses, .008); }
  if (o.earring) { D.ell(x, .352, .92, .013, .013, o.earring); D.ell(x, .648, .92, .013, .013, o.earring); }
  // hair on top (when no hat or under brimmed hats)
  if (o.hairTop !== false) {
    x.beginPath(); x.moveTo(.35, .84); x.quadraticCurveTo(.36, .64, .5, .645); x.quadraticCurveTo(.64, .64, .65, .84); x.quadraticCurveTo(.6, .72, .5, .73); x.quadraticCurveTo(.41, .72, .35, .84);
    x.fillStyle = hair; x.fill();
  }
  if (o.curls) [[.31, .97], [.69, .97], [.3, 1.07], [.7, 1.07]].forEach(([a, b]) => { D.ell(x, a, b, .052, .052, hair); D.ell(x, a - .012, b - .012, .018, .018, 'rgba(255,220,180,.18)'); });
  if (o.bun) { D.ell(x, .5, .62, .085, .06, hair); }
  // hat
  if (o.hat) IR.hats[o.hat.type](x, o.hat);
  if (o.front) o.front(x);
};
IR.hats = {
  bucket(x, h) { // fishing bucket hat
    x.beginPath(); x.moveTo(.27, .74); x.quadraticCurveTo(.5, .79, .73, .74); x.lineTo(.68, .67); x.quadraticCurveTo(.5, .7, .32, .67); x.closePath(); x.fillStyle = h.brim || h.color; x.fill();
    x.beginPath(); x.moveTo(.34, .69); x.quadraticCurveTo(.33, .53, .5, .52); x.quadraticCurveTo(.67, .53, .66, .69); x.closePath(); x.fillStyle = D.lin(x, .34, 0, .66, 0, [h.color, h.light || h.color, h.color]); x.fill();
    x.fillStyle = h.band || 'rgba(0,0,0,.25)'; x.fillRect(.338, .64, .324, .03);
    if (h.lure) { D.line(x, .6, .58, .64, .63, '#c0c4c8', .008); D.ell(x, .645, .645, .016, .03, h.lure, .4); D.ell(x, .62, .6, .01, .01, '#e8c66a'); }
    for (let i = 0; i < 4; i++) D.line(x, .3 + i * .13, .745, .3 + i * .13 + .02, .76, 'rgba(0,0,0,.15)', .004);
  },
  cap(x, h) {
    x.beginPath(); x.moveTo(.35, .72); x.quadraticCurveTo(.34, .55, .5, .545); x.quadraticCurveTo(.66, .55, .65, .72); x.closePath(); x.fillStyle = h.color; x.fill();
    x.beginPath(); x.moveTo(.36, .71); x.quadraticCurveTo(.25, .73, .22, .77); x.quadraticCurveTo(.35, .77, .5, .73); x.closePath(); x.fillStyle = h.brim || h.color; x.fill();
    D.ell(x, .5, .55, .018, .012, h.button || 'rgba(0,0,0,.3)');
    if (h.patch) { D.ell(x, .53, .64, .045, .035, h.patch); }
  },
  sunhat(x, h) {
    D.ell(x, .5, .71, .34, .075, h.color);
    D.ell(x, .5, .71, .34, .075, D.rad(x, .5, .71, .05, .34, ['rgba(255,255,255,.12)', 'rgba(0,0,0,.12)']));
    x.beginPath(); x.moveTo(.35, .71); x.quadraticCurveTo(.35, .55, .5, .545); x.quadraticCurveTo(.65, .55, .65, .71); x.closePath(); x.fillStyle = h.color; x.fill();
    x.fillStyle = h.band; x.fillRect(.352, .655, .296, .04);
    if (h.flower) { for (let i = 0; i < 5; i++) { const a = i / 5 * Math.PI * 2; D.ell(x, .62 + Math.cos(a) * .022, .675 + Math.sin(a) * .022, .016, .016, h.flower); } D.ell(x, .62, .675, .012, .012, '#f3d36b'); }
    for (let i = 0; i < 12; i++) { const a = i / 12 * Math.PI * 2; D.line(x, .5 + Math.cos(a) * .2, .71 + Math.sin(a) * .045, .5 + Math.cos(a) * .32, .71 + Math.sin(a) * .07, 'rgba(120,80,30,.18)', .004); }
  },
  cowboy(x, h) {
    x.beginPath(); x.moveTo(.18, .7); x.quadraticCurveTo(.22, .76, .5, .76); x.quadraticCurveTo(.78, .76, .82, .7); x.quadraticCurveTo(.76, .735, .5, .725); x.quadraticCurveTo(.24, .735, .18, .7); x.closePath();
    x.fillStyle = h.color; x.fill();
    D.ell(x, .5, .72, .27, .04, h.color);
    x.beginPath(); x.moveTo(.34, .72); x.quadraticCurveTo(.33, .55, .4, .5); x.quadraticCurveTo(.5, .56, .6, .5); x.quadraticCurveTo(.67, .55, .66, .72); x.closePath();
    x.fillStyle = D.lin(x, .34, 0, .66, 0, [h.color, h.light || h.color, h.color]); x.fill();
    D.line(x, .5, .55, .5, .66, 'rgba(0,0,0,.22)', .01);
    x.fillStyle = h.band || '#3a2412'; x.fillRect(.338, .665, .324, .035);
    if (h.concho) { D.ell(x, .5, .682, .018, .016, h.concho); D.ell(x, .497, .678, .007, .006, '#fff'); }
    x.beginPath(); x.moveTo(.18, .7); x.quadraticCurveTo(.22, .76, .5, .76); x.quadraticCurveTo(.78, .76, .82, .7); x.strokeStyle = 'rgba(0,0,0,.25)'; x.lineWidth = .006; x.stroke();
  },
  crown(x, h) {
    const g = h.color || '#d9b04a';
    D.poly(x, [[.33, .69], [.33, .55], [.39, .61], [.44, .52], [.5, .6], [.56, .52], [.61, .61], [.67, .55], [.67, .69]], D.lin(x, .33, 0, .67, 0, ['#9c7524', g, '#fff0b0', g, '#9c7524']));
    x.fillStyle = h.band || '#7a1a1a'; x.fillRect(.33, .655, .34, .035);
    [.39, .5, .61].forEach((cx, i) => D.ell(x, cx, .672, .014, .012, i === 1 ? (h.jewel || '#2f6bd1') : '#e5e5e5'));
    [[.33, .55], [.44, .52], [.56, .52], [.67, .55]].forEach(([a, b]) => D.ell(x, a, b, .012, .012, '#fff3c0'));
  },
  tiara(x, h) {
    x.beginPath(); x.moveTo(.37, .7); x.quadraticCurveTo(.5, .6, .63, .7); x.lineTo(.6, .7); x.quadraticCurveTo(.5, .64, .4, .7); x.closePath(); x.fillStyle = h.color || '#e3c45e'; x.fill();
    D.poly(x, [[.47, .64], [.5, .58], [.53, .64]], h.color || '#e3c45e'); D.ell(x, .5, .63, .012, .012, h.jewel || '#c22');
  },
  fedora(x, h) {
    D.ell(x, .5, .715, .27, .05, h.color);
    x.beginPath(); x.moveTo(.35, .715); x.quadraticCurveTo(.34, .56, .42, .545); x.quadraticCurveTo(.5, .58, .58, .545); x.quadraticCurveTo(.66, .56, .65, .715); x.closePath();
    x.fillStyle = D.lin(x, .35, 0, .65, 0, [h.color, h.light || h.color, h.color]); x.fill();
    x.fillStyle = h.band || '#111'; x.fillRect(.348, .665, .304, .035);
  },
  bandana(x, h) {
    x.beginPath(); x.moveTo(.34, .76); x.quadraticCurveTo(.34, .6, .5, .6); x.quadraticCurveTo(.66, .6, .66, .76); x.quadraticCurveTo(.5, .7, .34, .76); x.fillStyle = h.color; x.fill();
    for (let i = 0; i < 7; i++) D.ell(x, .38 + i * .04, .69 - Math.sin(i / 6 * Math.PI) * .05, .007, .007, 'rgba(255,255,255,.8)');
  },
};

/* --------------------------------------------------------------- card art
   theme = {paper, border, pip:{red,black}, index:{red,black,font}, court(x,s,r,w,h), ace(x,s,w,h), back(x,w,h), radius} */
IR.CardArt = class {
  constructor(theme, cw, ch, res) { this.t = theme; this.cw = cw; this.ch = ch; this.res = res; this.cache = {}; this.backURL = null; }
  setRes(res) { if (Math.abs(res - this.res) > .01) { this.res = res; this.cache = {}; this.backURL = null; } }
  body(x, face) {
    const { cw, ch } = this, r = cw * (this.t.radius || .07), t = this.t;
    D.rr(x, 0, 0, cw, ch, r); x.save(); x.clip();
    if (face) {
      x.fillStyle = D.lin(x, 0, 0, cw * .6, ch, t.paper || ['#fdf8ec', '#efe3c8']); x.fillRect(0, 0, cw, ch);
      const R = IR.rng(cw * 977 + ch);
      for (let i = 0; i < 160; i++) { x.fillStyle = `rgba(110,80,40,${R() * .05})`; const s = R() * cw * .012 + .3; x.fillRect(R() * cw, R() * ch, s, s); }
      x.fillStyle = D.rad(x, cw / 2, ch / 2, cw * .3, ch * .75, ['rgba(0,0,0,0)', t.vignette || 'rgba(110,80,30,.16)']); x.fillRect(0, 0, cw, ch);
    }
    x.restore();
  }
  index(x, id) {
    const { cw, ch } = this, t = this.t, s = IR.suit(id), r = IR.rank(id), red = IR.isRed(id);
    const pass = () => {
      x.fillStyle = red ? (t.index?.red || '#a5221b') : (t.index?.black || '#1d1915');
      x.font = `${t.index?.weight || 'bold'} ${cw * .2}px ${t.index?.font || 'Georgia, serif'}`; x.textAlign = 'center'; x.textBaseline = 'alphabetic';
      x.save(); x.translate(cw * .118, cw * .235); if (r === 10) x.scale(.76, 1); x.fillText(IR.RANKS[r - 1], 0, 0); x.restore();
      IR.drawSuit(x, s, cw * .118, cw * .34, cw * .13, false, t.pip);
    };
    pass(); x.save(); x.translate(cw, ch); x.rotate(Math.PI); pass(); x.restore();
  }
  render(id) {
    const { cw, ch, res } = this, t = this.t;
    const [c, x] = D.canvas(cw * res, ch * res); x.scale(res, res);
    this.body(x, true);
    const s = IR.suit(id), r = IR.rank(id);
    const PIPS = IR.PIPS;
    if (r >= 11) {
      const fw = cw * .64, fh = fw * RH, fx = (cw - fw) / 2, fy = (ch - fh) / 2;
      x.save(); x.beginPath(); x.rect(fx, fy, fw, fh); x.clip(); x.translate(fx, fy); x.scale(fw, fw);
      t.court(x, s, r); x.restore();
      x.strokeStyle = t.frame?.[0] || '#5a3a1c'; x.lineWidth = cw * .014; x.strokeRect(fx, fy, fw, fh);
      x.strokeStyle = D.lin(x, fx, fy, fx + fw, fy + fh, t.frame?.slice(1) || ['#f2d47a', '#b8862d', '#f0cf6e']); x.lineWidth = cw * .009; x.strokeRect(fx + cw * .012, fy + cw * .012, fw - cw * .024, fh - cw * .024);
      const badge = (bx, by) => { D.ell(x, bx, by, fw * .1, fw * .1, 'rgba(253,248,236,.93)'); x.strokeStyle = t.frame?.[2] || '#b8862d'; x.lineWidth = cw * .006; x.stroke(); IR.drawSuit(x, s, bx, by + fw * .005, fw * .12, false, t.pip); };
      badge(fx + fw * .13, fy + fw * .13);
      x.save(); x.translate(cw, ch); x.rotate(Math.PI); badge(fx + fw * .13, fy + fw * .13); x.restore();
    } else if (r === 1) {
      if (t.ace) t.ace(x, s, cw, ch); else IR.drawSuit(x, s, cw / 2, ch / 2, cw * .38, false, t.pip);
    } else {
      const sz = r >= 9 ? cw * .17 : cw * .19;
      for (const [px, py] of PIPS[r]) {
        if (t.pipDecor) t.pipDecor(x, s, px * cw, py * ch, sz, py > .5);
        IR.drawSuit(x, s, px * cw, py * ch, sz, py > .5, t.pip);
      }
    }
    this.index(x, id);
    D.rr(x, cw * .006, cw * .006, cw - cw * .012, ch - cw * .012, cw * (t.radius || .07)); x.strokeStyle = t.border || '#6a4524'; x.lineWidth = cw * .012; x.stroke();
    return c;
  }
  face(id) { id %= 52; return this.cache[id] || (this.cache[id] = this.render(id).toDataURL('image/png')); }
  back() {
    if (this.backURL) return this.backURL;
    const { cw, ch, res } = this, t = this.t;
    const [c, x] = D.canvas(cw * res, ch * res); x.scale(res, res);
    D.rr(x, 0, 0, cw, ch, cw * (t.radius || .07)); x.save(); x.clip(); t.back(x, cw, ch); x.restore();
    D.rr(x, cw * .006, cw * .006, cw - cw * .012, ch - cw * .012, cw * (t.radius || .07)); x.strokeStyle = t.backBorder || '#1a0a06'; x.lineWidth = cw * .012; x.stroke();
    return (this.backURL = c.toDataURL('image/png'));
  }
  preload(ids) { ids.forEach(i => this.face(i)); this.back(); }
};
IR.PIPS = {
  2: [[.5, .2], [.5, .8]], 3: [[.5, .2], [.5, .5], [.5, .8]], 4: [[.3, .2], [.7, .2], [.3, .8], [.7, .8]],
  5: [[.3, .2], [.7, .2], [.5, .5], [.3, .8], [.7, .8]], 6: [[.3, .2], [.7, .2], [.3, .5], [.7, .5], [.3, .8], [.7, .8]],
  7: [[.3, .2], [.7, .2], [.5, .35], [.3, .5], [.7, .5], [.3, .8], [.7, .8]], 8: [[.3, .2], [.7, .2], [.5, .35], [.3, .5], [.7, .5], [.5, .65], [.3, .8], [.7, .8]],
  9: [[.3, .2], [.7, .2], [.3, .4], [.7, .4], [.5, .5], [.3, .6], [.7, .6], [.3, .8], [.7, .8]],
  10: [[.3, .2], [.7, .2], [.5, .3], [.3, .4], [.7, .4], [.3, .6], [.7, .6], [.5, .7], [.3, .8], [.7, .8]],
};

/* ---------------------------------------------------------------- stage
   Fixed 1600x900 logical stage, scaled to fit the window. */
IR.Stage = {
  W: 1600, H: 900, scale: 1,
  init(el, onResize) {
    this.el = el;
    const fit = () => {
      const s = Math.min(innerWidth / this.W, innerHeight / this.H);
      this.scale = s;
      el.style.transform = `translate(${(innerWidth - this.W * s) / 2}px,${(innerHeight - this.H * s) / 2}px) scale(${s})`;
      if (onResize) onResize(s);
    };
    addEventListener('resize', fit); fit();
  },
};

/* ------------------------------------------------------------- card layer
   Each card is a DOM element; moves and flips are CSS transitions. */
IR.Cards = class {
  constructor(layer, art) { this.layer = layer; this.art = art; this.z = 10; }
  make(id, opts = {}) {
    const c = IR.el('div', 'card', this.layer);
    c.innerHTML = `<div class="face front"></div><div class="face back"></div>`;
    c._id = id; c._x = opts.x ?? 0; c._y = opts.y ?? 0; c._r = opts.r || 0; c._up = !!opts.up; c._s = opts.s || 1;
    c.querySelector('.back').style.backgroundImage = `url(${this.art.back()})`;
    if (id != null) this.setFace(c, id);
    c.style.transition = 'none'; this.apply(c); void c.offsetWidth; c.style.transition = '';
    c.style.zIndex = this.z++;
    return c;
  }
  setFace(c, id) { c._id = id; if (id != null) c.querySelector('.front').style.backgroundImage = `url(${this.art.face(id)})`; }
  apply(c) { c.style.transform = `translate(${c._x}px,${c._y}px) rotate(${c._r}deg) scale(${c._s}) rotateY(${c._up ? 0 : 180}deg)`; }
  move(c, x, y, o = {}) {
    c._x = x; c._y = y; if (o.r != null) c._r = o.r; if (o.up != null) c._up = o.up; if (o.s != null) c._s = o.s;
    c.style.transitionDuration = ((o.dur ?? 420) / (IR.speed || 1)) + 'ms';
    if (o.top !== false) c.style.zIndex = o.z ?? this.z++;
    this.apply(c);
  }
  flip(c, up, dur = 360) { this.move(c, c._x, c._y, { up, dur, top: false }); }
  remove(c) { c.remove(); }
  clear() { this.layer.innerHTML = ''; this.z = 10; }
};

/* ------------------------------------------------------------------ audio */
const NOTE = { C: 0, 'C#': 1, Db: 1, D: 2, 'D#': 3, Eb: 3, E: 4, F: 5, 'F#': 6, Gb: 6, G: 7, 'G#': 8, Ab: 8, A: 9, 'A#': 10, Bb: 10, B: 11 };
IR.NOTE = NOTE;
IR.n2m = n => { const m = n.match(/^([A-G][#b]?)(-?\d)$/); return 12 * (+m[2] + 1) + NOTE[m[1]]; };
IR.mtof = m => 440 * Math.pow(2, (m - 69) / 12);
// chord symbol -> pitch classes, e.g. "Dm7", "G7", "Cmaj7", "F#m7b5", "Bb6", "Gsus4"
IR.chord = sym => {
  const m = sym.match(/^([A-G][#b]?)(.*)$/); const root = NOTE[m[1]]; const q = m[2];
  let iv;
  if (q === '' || q === 'maj') iv = [0, 4, 7];
  else if (q === 'm') iv = [0, 3, 7];
  else if (q === '7') iv = [0, 4, 7, 10];
  else if (q === 'm7') iv = [0, 3, 7, 10];
  else if (q === 'maj7') iv = [0, 4, 7, 11];
  else if (q === '6') iv = [0, 4, 7, 9];
  else if (q === 'm6') iv = [0, 3, 7, 9];
  else if (q === 'm7b5') iv = [0, 3, 6, 10];
  else if (q === '9') iv = [0, 4, 7, 10, 14];
  else if (q === 'maj9') iv = [0, 4, 7, 11, 14];
  else if (q === 'm9') iv = [0, 3, 7, 10, 14];
  else if (q === 'add9') iv = [0, 4, 7, 14];
  else if (q === 'sus4') iv = [0, 5, 7];
  else if (q === 'sus2') iv = [0, 2, 7];
  else if (q === 'dim') iv = [0, 3, 6];
  else if (q === '7b9') iv = [0, 4, 7, 10, 13];
  else iv = [0, 4, 7];
  return { root, iv, pcs: iv.map(i => (root + i) % 12) };
};
// voice: nearest midi >= lo for a pitch class
IR.pcAbove = (pc, lo) => { let m = lo + ((pc - lo) % 12 + 12) % 12; return m; };

IR.Audio = class {
  constructor(settings) {
    this.set = settings; this.ac = null; this.running = false; this.ks = {}; this.tunes = []; this.idx = 0;
  }
  ensure() {
    if (this.ac) return true;
    try { this.ac = new (window.AudioContext || window.webkitAudioContext)(); } catch (e) { return false; }
    const ac = this.ac;
    const comp = ac.createDynamicsCompressor(); comp.threshold.value = -16; comp.ratio.value = 3.5; comp.connect(ac.destination);
    this.master = ac.createGain(); this.master.gain.value = .9; this.master.connect(comp);
    this.music = ac.createGain(); this.music.gain.value = 0; this.music.connect(this.master);
    this.sfxBus = ac.createGain(); this.sfxBus.gain.value = .85; this.sfxBus.connect(this.master);
    this.bus = ac.createGain(); this.bus.connect(this.music);
    const rv = ac.createConvolver(); rv.buffer = this.impulse(2.8); this.wet = ac.createGain(); this.wet.gain.value = .3; this.bus.connect(rv); rv.connect(this.wet); this.wet.connect(this.music);
    this.sfxRev = ac.createGain(); this.sfxRev.gain.value = .18; this.sfxBus.connect(this.sfxRev); this.sfxRev.connect(rv);
    this.noise = ac.createBuffer(1, ac.sampleRate * 2, ac.sampleRate); const d = this.noise.getChannelData(0); for (let i = 0; i < d.length; i++) d[i] = Math.random() * 2 - 1;
    return true;
  }
  impulse(sec) { const ac = this.ac, len = Math.floor(ac.sampleRate * sec), b = ac.createBuffer(2, len, ac.sampleRate); for (let c = 0; c < 2; c++) { const d = b.getChannelData(c); for (let i = 0; i < len; i++) d[i] = (Math.random() * 2 - 1) * Math.pow(1 - i / len, 3); } return b; }
  get now() { return this.ac.currentTime; }
  unlock() { if (!this.ensure()) return; if (this.ac.state === 'suspended') this.ac.resume(); if (this.set.music && !this.running) this.start(); }
  vol() { return this.set.music ? Math.pow(this.set.vol / 100, 1.6) * .95 : 0; }
  setVol() { if (this.ac) { this.music.gain.cancelScheduledValues(this.now); this.music.gain.setTargetAtTime(this.running ? this.vol() : 0, this.now, .15); this.sfxBus.gain.value = this.set.sfx ? .85 * Math.max(.35, Math.pow(this.set.vol / 100, .6)) : 0; } }

  /* ---- instruments (out defaults to music bus) ---- */
  env(g, t, a, peak, d, sus, rel, end) {
    g.gain.setValueAtTime(0, t); g.gain.linearRampToValueAtTime(peak, t + a);
    g.gain.setTargetAtTime(peak * sus, t + a, d); g.gain.setValueAtTime(peak * sus, Math.max(t + a + .01, end)); g.gain.setTargetAtTime(0, Math.max(t + a + .01, end), rel);
  }
  ksBuf(m, decay, bright, dur) {
    const key = m + '|' + decay + '|' + bright + '|' + dur; if (this.ks[key]) return this.ks[key];
    const ac = this.ac, sr = ac.sampleRate, N = Math.max(2, Math.round(sr / IR.mtof(m))), len = Math.floor(sr * dur);
    const b = ac.createBuffer(1, len, sr), d = b.getChannelData(0); let p = 0, mx = 0;
    for (let i = 0; i < N; i++) { p += bright * ((Math.random() * 2 - 1) - p); d[i] = p; mx = Math.max(mx, Math.abs(p)); }
    for (let i = 0; i < N; i++) d[i] = d[i] / mx * .8;
    for (let i = N; i < len; i++) d[i] = decay * .5 * (d[i - N] + d[i - N - 1 >= 0 ? i - N - 1 : 0]);
    return (this.ks[key] = b);
  }
  pluck(t, m, v, o = {}) {
    const ac = this.ac, s = ac.createBufferSource(); s.buffer = this.ksBuf(m, o.decay ?? .996, o.bright ?? .5, o.dur ?? 1.6);
    const g = ac.createGain(); g.gain.value = v; let n = s;
    if (o.lp) { const f = ac.createBiquadFilter(); f.type = 'lowpass'; f.frequency.value = o.lp; n.connect(f); n = f; }
    if (o.body) { const f = ac.createBiquadFilter(); f.type = 'peaking'; f.frequency.value = o.body; f.Q.value = 1.2; f.gain.value = 5; n.connect(f); n = f; }
    n.connect(g); g.connect(o.out || this.bus); s.start(t);
    if (o.stop) s.stop(t + o.stop);
  }
  epiano(t, m, v, dur, out) { // FM electric piano
    const ac = this.ac, f = IR.mtof(m);
    const car = ac.createOscillator(), mod = ac.createOscillator(), mg = ac.createGain(), g = ac.createGain();
    car.frequency.value = f; mod.frequency.value = f; mg.gain.setValueAtTime(f * 1.6 * v + 40, t); mg.gain.exponentialRampToValueAtTime(f * .15 + 1, t + 1.2);
    mod.connect(mg); mg.connect(car.frequency);
    const bell = ac.createOscillator(), bg = ac.createGain(); bell.frequency.value = f * 4.01; bg.gain.setValueAtTime(v * .05, t); bg.gain.exponentialRampToValueAtTime(.0001, t + .35); bell.connect(bg); bg.connect(g);
    const trem = ac.createOscillator(), tg = ac.createGain(); trem.frequency.value = 4.6; tg.gain.value = .12; trem.connect(tg); tg.connect(g.gain);
    g.gain.setValueAtTime(0, t); g.gain.linearRampToValueAtTime(v * .26, t + .006); g.gain.setTargetAtTime(v * .1, t + .01, .45); g.gain.setTargetAtTime(0, t + dur, .18);
    car.connect(g); g.connect(out || this.bus);
    const e = t + dur + 1; [car, mod, bell, trem].forEach(o => { o.start(t); o.stop(e); });
  }
  vibes(t, m, v, dur, out) { // vibraphone
    const ac = this.ac, f = IR.mtof(m), g = ac.createGain();
    [[1, 1], [4, .18], [10, .04]].forEach(([r, a]) => { const o = ac.createOscillator(), og = ac.createGain(); o.frequency.value = f * r; og.gain.setValueAtTime(a, t); og.gain.exponentialRampToValueAtTime(.001, t + (r === 1 ? 2.6 : .5)); o.connect(og); og.connect(g); o.start(t); o.stop(t + 3); });
    const trem = ac.createOscillator(), tg = ac.createGain(); trem.frequency.value = 5.5; tg.gain.value = v * .06; trem.connect(tg); tg.connect(g.gain); trem.start(t); trem.stop(t + 3);
    g.gain.setValueAtTime(v * .2, t); g.gain.setTargetAtTime(0, t + Math.max(.2, dur), .5); g.connect(out || this.bus);
  }
  bell(t, m, v, out, len = 1.6) {
    const ac = this.ac, f = IR.mtof(m), g = ac.createGain(); g.connect(out || this.bus);
    [[1, 1], [2.76, .35], [5.4, .18], [8.93, .08]].forEach(([r, a]) => { const o = ac.createOscillator(), og = ac.createGain(); o.frequency.value = f * r; og.gain.setValueAtTime(0, t); og.gain.linearRampToValueAtTime(a * v * .3, t + .004); og.gain.exponentialRampToValueAtTime(.0001, t + len / Math.sqrt(r)); o.connect(og); og.connect(g); o.start(t); o.stop(t + len + .1); });
  }
  bass(t, m, v, dur, o = {}) { // upright bass
    const ac = this.ac, f = IR.mtof(m), g = ac.createGain();
    const o1 = ac.createOscillator(), o2 = ac.createOscillator(); o1.type = 'triangle'; o2.type = 'sine'; o1.frequency.value = f; o2.frequency.value = f * 2;
    const g2 = ac.createGain(); g2.gain.value = .25; o2.connect(g2); g2.connect(g);
    const lp = ac.createBiquadFilter(); lp.type = 'lowpass'; lp.frequency.setValueAtTime(o.bright || 900, t); lp.frequency.exponentialRampToValueAtTime(260, t + .4);
    o1.connect(g); g.connect(lp); lp.connect(o.out || this.bus);
    g.gain.setValueAtTime(0, t); g.gain.linearRampToValueAtTime(v * .55, t + .012); g.gain.setTargetAtTime(v * .25, t + .02, .25); g.gain.setTargetAtTime(0, t + dur, .07);
    [o1, o2].forEach(x => { x.start(t); x.stop(t + dur + .5); });
    this.pluck(t, m + 12, v * .06, { decay: .99, bright: .3, dur: .4, lp: 1200, out: o.out });
  }
  noiseHit(t, v, o = {}) {
    const ac = this.ac, s = ac.createBufferSource(); s.buffer = this.noise;
    const f = ac.createBiquadFilter(); f.type = o.type || 'bandpass'; f.frequency.value = o.freq || 3000; f.Q.value = o.q || .8;
    const g = ac.createGain(); const a = o.attack || .002, d = o.decay || .08;
    g.gain.setValueAtTime(0, t); g.gain.linearRampToValueAtTime(v, t + a); g.gain.exponentialRampToValueAtTime(.0005, t + a + d);
    s.connect(f); f.connect(g); g.connect(o.out || this.bus); s.start(t, Math.random() * 1.5, a + d + .05);
    if (o.sweep) { f.frequency.setValueAtTime(o.freq, t); f.frequency.exponentialRampToValueAtTime(o.sweep, t + a + d); }
  }
  brush(t, v, long) { this.noiseHit(t, v * .22, { freq: 4500, q: .5, attack: long ? .06 : .01, decay: long ? .22 : .08, type: 'bandpass' }); }
  ride(t, v) { this.noiseHit(t, v * .13, { freq: 8000, q: 1.5, decay: .35, type: 'highpass' }); const ac = this.ac; [1, 1.47, 2.09].forEach(r => { const o = ac.createOscillator(), g = ac.createGain(); o.type = 'square'; o.frequency.value = 520 * r; g.gain.setValueAtTime(v * .006, t); g.gain.exponentialRampToValueAtTime(.0001, t + .4); const hp = ac.createBiquadFilter(); hp.type = 'highpass'; hp.frequency.value = 5000; o.connect(hp); hp.connect(g); g.connect(this.bus); o.start(t); o.stop(t + .45); }); }
  kick(t, v) { const ac = this.ac, o = ac.createOscillator(), g = ac.createGain(); o.frequency.setValueAtTime(110, t); o.frequency.exponentialRampToValueAtTime(45, t + .12); g.gain.setValueAtTime(v * .5, t); g.gain.exponentialRampToValueAtTime(.001, t + .25); o.connect(g); g.connect(this.bus); o.start(t); o.stop(t + .3); }
  flute(t, m, v, dur, out) {
    const ac = this.ac, f = IR.mtof(m), g = ac.createGain();
    const o = ac.createOscillator(); o.frequency.value = f; const o2 = ac.createOscillator(); o2.frequency.value = f * 2; const g2 = ac.createGain(); g2.gain.value = .12; o2.connect(g2); g2.connect(g);
    const lfo = ac.createOscillator(), lg = ac.createGain(); lfo.frequency.value = 4.8; lg.gain.setValueAtTime(0, t); lg.gain.linearRampToValueAtTime(f * .006, t + Math.min(.5, dur * .7)); lfo.connect(lg); lg.connect(o.frequency); lg.connect(o2.frequency);
    o.connect(g);
    const br = ac.createBufferSource(); br.buffer = this.noise; const bf = ac.createBiquadFilter(); bf.type = 'bandpass'; bf.frequency.value = f * 2; bf.Q.value = 2; const bg = ac.createGain(); bg.gain.value = .05; br.connect(bf); bf.connect(bg); bg.connect(g);
    this.env(g, t, .07, v * .14, .3, .82, .12, t + dur);
    g.connect(out || this.bus); const e = t + dur + .7; [o, o2, lfo].forEach(x => { x.start(t); x.stop(e); }); br.start(t, Math.random(), dur + .6);
  }
  fiddle(t, m, v, dur, out) {
    const ac = this.ac, f = IR.mtof(m);
    const o1 = ac.createOscillator(), o2 = ac.createOscillator(); o1.type = o2.type = 'sawtooth'; o1.frequency.value = f; o2.frequency.value = f * 1.003;
    const lfo = ac.createOscillator(); lfo.frequency.value = 5.4; const lg = ac.createGain(); lg.gain.setValueAtTime(0, t); lg.gain.linearRampToValueAtTime(f * .007, t + Math.min(.35, dur * .6)); lfo.connect(lg); lg.connect(o1.frequency); lg.connect(o2.frequency);
    const lp = ac.createBiquadFilter(); lp.type = 'lowpass'; lp.frequency.value = Math.min(3800, f * 4); const pk = ac.createBiquadFilter(); pk.type = 'peaking'; pk.frequency.value = 2600; pk.Q.value = 1.3; pk.gain.value = 4;
    const hp = ac.createBiquadFilter(); hp.type = 'highpass'; hp.frequency.value = 200;
    const g = ac.createGain(); this.env(g, t, .08, v * .07, .2, .8, .1, t + dur);
    o1.connect(lp); o2.connect(lp); lp.connect(pk); pk.connect(hp); hp.connect(g); g.connect(out || this.bus);
    [o1, o2, lfo].forEach(x => { x.start(t); x.stop(t + dur + .6); });
  }
  harmonica(t, m, v, dur, out) {
    const ac = this.ac, f = IR.mtof(m);
    const o1 = ac.createOscillator(), o2 = ac.createOscillator(); o1.type = 'square'; o2.type = 'sawtooth'; o1.frequency.value = f; o2.frequency.value = f * 1.005;
    const bp = ac.createBiquadFilter(); bp.type = 'bandpass'; bp.frequency.value = Math.min(2400, f * 2.2); bp.Q.value = .6;
    const lp = ac.createBiquadFilter(); lp.type = 'lowpass'; lp.frequency.value = 3000;
    const g = ac.createGain(), tr = ac.createGain(), lfo = ac.createOscillator(), lg = ac.createGain(); lfo.frequency.value = 5.6; lg.gain.value = .15; lfo.connect(lg); lg.connect(tr.gain); tr.gain.value = .85;
    this.env(g, t, .05, v * .065, .2, .85, .08, t + dur);
    o1.connect(bp); o2.connect(bp); bp.connect(lp); lp.connect(tr); tr.connect(g); g.connect(out || this.bus);
    [o1, o2, lfo].forEach(x => { x.start(t); x.stop(t + dur + .5); });
  }
  steel(t, m, v, dur, out) { // pedal steel: slide into the note, long sustain, vibrato
    const ac = this.ac, f = IR.mtof(m);
    const o1 = ac.createOscillator(), o2 = ac.createOscillator(); o1.type = 'triangle'; o2.type = 'sine'; o2.frequency.value = f * 2;
    o1.frequency.setValueAtTime(f * .944, t); o1.frequency.exponentialRampToValueAtTime(f, t + .16); o2.frequency.setValueAtTime(f * 1.888, t); o2.frequency.exponentialRampToValueAtTime(f * 2, t + .16);
    const lfo = ac.createOscillator(), lg = ac.createGain(); lfo.frequency.value = 5; lg.gain.setValueAtTime(0, t); lg.gain.linearRampToValueAtTime(f * .005, t + .6); lfo.connect(lg); lg.connect(o1.frequency);
    const g = ac.createGain(), g2 = ac.createGain(); g2.gain.value = .3; o2.connect(g2); g2.connect(g); o1.connect(g);
    this.env(g, t, .12, v * .12, .8, .7, .25, t + dur);
    const lp = ac.createBiquadFilter(); lp.type = 'lowpass'; lp.frequency.value = 2200; g.connect(lp); lp.connect(out || this.bus);
    [o1, o2, lfo].forEach(x => { x.start(t); x.stop(t + dur + 1.2); });
  }
  pad(t, ms, v, dur, out) { // soft string/organ pad
    const ac = this.ac, g = ac.createGain(), lp = ac.createBiquadFilter(); lp.type = 'lowpass'; lp.frequency.value = 1100; lp.Q.value = .3;
    ms.forEach(m => { [-4, 4].forEach(dt => { const o = ac.createOscillator(); o.type = 'sawtooth'; o.frequency.value = IR.mtof(m); o.detune.value = dt; o.connect(lp); o.start(t); o.stop(t + dur + 2); }); });
    g.gain.setValueAtTime(0, t); g.gain.linearRampToValueAtTime(v * .02 / Math.sqrt(ms.length), t + Math.min(1.2, dur * .4)); g.gain.setValueAtTime(v * .02 / Math.sqrt(ms.length), t + dur); g.gain.linearRampToValueAtTime(0, t + dur + 1.5);
    lp.connect(g); g.connect(out || this.bus);
  }
  piano(t, m, v, dur, out) {
    const ac = this.ac, f = IR.mtof(m), g = ac.createGain();
    [[1, 1, 'triangle'], [2, .35, 'sine'], [3, .12, 'sine'], [4.02, .05, 'sine']].forEach(([r, a, ty]) => { const o = ac.createOscillator(), og = ac.createGain(); o.type = ty; o.frequency.value = f * r; og.gain.setValueAtTime(a, t); og.gain.exponentialRampToValueAtTime(.001, t + 2.2 / r + .3); o.connect(og); og.connect(g); o.start(t); o.stop(t + dur + 2.5); });
    g.gain.setValueAtTime(0, t); g.gain.linearRampToValueAtTime(v * .16, t + .005); g.gain.setTargetAtTime(v * .07, t + .01, .4); g.gain.setTargetAtTime(0, t + dur, .25);
    const lp = ac.createBiquadFilter(); lp.type = 'lowpass'; lp.frequency.value = 2400 + v * 2000; g.connect(lp); lp.connect(out || this.bus);
  }
  play(e, t, spb) {
    const d = (e.d || 1) * spb, v = e.v ?? 1, inst = this.instruments[e.i];
    if (inst) inst.call(this, t, e, d, v);
  }
  /* ---- sequencer: tunes = [{name, bpm, swing, events:[{t (beats), i, m|ms, d, v}], len (beats)}] */
  setTunes(tunes, instruments) { this.tunes = tunes; this.instruments = instruments; this.idx = Math.floor(Math.random() * tunes.length); }
  start() {
    if (!this.ensure() || this.running || !this.tunes.length) return;
    this.running = true; this.cur = this.tunes[this.idx]; this.ev = 0; this.t0 = this.now + .3;
    this.setVol(); clearInterval(this.timer); this.timer = setInterval(() => this.tick(), 70); this.tick();
  }
  stop() { if (!this.ac) return; this.running = false; clearInterval(this.timer); this.music.gain.cancelScheduledValues(this.now); this.music.gain.setTargetAtTime(0, this.now, .12); this.idx = (this.idx + 1) % this.tunes.length; }
  tick() {
    if (!this.running) return; const now = this.now, ahead = now + .35;
    for (let guard = 0; guard < 500; guard++) {
      const tune = this.cur, spb = 60 / tune.bpm;
      if (this.ev >= tune.events.length) { this.t0 += tune.len * spb + (tune.gap ?? 2); this.idx = (this.idx + 1) % this.tunes.length; this.cur = this.tunes[this.idx]; this.ev = 0; IR.emit('tune', this.cur.name); continue; }
      const e = tune.events[this.ev];
      let bt = e.t; if (tune.swing) { const fr = bt % 1; if (Math.abs(fr - .5) < .01) bt += tune.swing * .5; }
      const tt = this.t0 + bt * spb; if (tt > ahead) break;
      if (tt >= now - .05) this.play(e, tt + (Math.random() - .5) * .008, spb);
      this.ev++;
    }
  }
  sfx(fn) { if (!this.ac || !this.set.sfx || this.ac.state !== 'running') return; fn.call(this, this.now + .01, this.sfxBus); }
  /* ---- shared sound effects ---- */
  sCard() { this.sfx((t, o) => { this.noiseHit(t, .5, { freq: 2600, q: .9, decay: .05, out: o }); this.noiseHit(t, .3, { freq: 300, type: 'lowpass', decay: .04, out: o }); }); }
  sDeal() { this.sfx((t, o) => { this.noiseHit(t, .35, { freq: 5200, q: .7, attack: .015, decay: .06, out: o, sweep: 2400 }); this.noiseHit(t + .05, .4, { freq: 1600, q: .9, decay: .04, out: o }); }); }
  sFlip() { this.sfx((t, o) => this.noiseHit(t, .3, { freq: 4200, type: 'highpass', decay: .04, out: o })); }
  sShuffle() { this.sfx((t, o) => { for (let i = 0; i < 26; i++) this.noiseHit(t + i * .028 + Math.random() * .01, .22, { freq: 1800 + Math.random() * 2600, q: 1, decay: .025, out: o }); for (let i = 0; i < 18; i++) this.noiseHit(t + .9 + i * .03, .2, { freq: 2200 + Math.random() * 2000, q: 1, decay: .025, out: o }); this.noiseHit(t + 1.5, .4, { freq: 400, type: 'lowpass', decay: .06, out: o }); }); }
  sChip(n = 1) { this.sfx((t, o) => { for (let k = 0; k < n; k++) { const tt = t + k * .06; [1, 1.5].forEach((r, j) => { const ac = this.ac, os = ac.createOscillator(), g = ac.createGain(); os.type = 'triangle'; os.frequency.value = 2900 * r + Math.random() * 200; g.gain.setValueAtTime(.12 / (j + 1), tt + j * .018); g.gain.exponentialRampToValueAtTime(.0005, tt + j * .018 + .07); os.connect(g); g.connect(o); os.start(tt + j * .018); os.stop(tt + .2); }); this.noiseHit(tt, .25, { freq: 5000, q: 2, decay: .03, out: o }); } }); }
  sChime(notes = [72, 76, 79, 84], gap = .09, v = 1) { this.sfx((t, o) => notes.forEach((m, i) => this.bell(t + i * gap, m, v, o, 1.8))); }
  sLose() { this.sfx((t, o) => [67, 63, 60].forEach((m, i) => this.vibes(t + i * .16, m, .8, .3, o))); }
};

/* ---------------------------------------------------------------- events */
const listeners = {};
IR.on = (n, f) => (listeners[n] = listeners[n] || []).push(f);
IR.emit = (n, ...a) => (listeners[n] || []).forEach(f => f(...a));

/* ------------------------------------------------------------------ saves
   Saved by the game shell to %APPDATA%\Ironite\<Game>\save.json
   (falls back to browser storage when run outside the shell). */
IR.Save = class {
  constructor(key, defaults) { this.key = key; this.data = JSON.parse(JSON.stringify(defaults)); this.defaults = defaults; this.timer = 0; }
  async load() {
    let d = null;
    try { const r = await fetch('api/save', { cache: 'no-store' }); if (r.ok) d = await r.json(); } catch (e) { }
    if (!d) try { d = JSON.parse(localStorage.getItem(this.key) || 'null'); } catch (e) { }
    if (d) for (const k of Object.keys(this.defaults)) { if (d[k] != null) this.data[k] = (typeof this.defaults[k] === 'object' && !Array.isArray(this.defaults[k])) ? Object.assign({}, this.defaults[k], d[k]) : d[k]; }
    setInterval(() => fetch('api/ping').catch(() => { }), 30000);
    return this.data;
  }
  write(now) { clearTimeout(this.timer); const go = () => { const s = JSON.stringify(this.data); try { localStorage.setItem(this.key, s); } catch (e) { } fetch('api/save', { method: 'POST', headers: { 'Content-Type': 'application/json', 'X-Ironite': '1' }, body: s }).catch(() => { }); }; if (now) go(); else this.timer = setTimeout(go, 400); }
};

/* ------------------------------------------------------------------- UI */
IR.mdChangelog = md => {
  const out = []; let inList = false;
  md.split(/\r?\n/).forEach(l => {
    if (/^## /.test(l)) { if (inList) { out.push('</ul>'); inList = false; } const m = l.match(/^## \[(.+?)\]\s*-\s*(.+)$/); out.push(m ? `<h4>Version ${m[1]} <small>${m[2]}</small></h4>` : `<h4>${l.slice(3)}</h4>`); }
    else if (/^### /.test(l)) { if (inList) { out.push('</ul>'); inList = false; } out.push(`<h5>${l.slice(4)}</h5>`); }
    else if (/^- /.test(l)) { if (!inList) { out.push('<ul>'); inList = true; } out.push(`<li>${l.slice(2).replace(/\*\*(.+?)\*\*/g, '<b>$1</b>')}</li>`); }
  });
  if (inList) out.push('</ul>'); return out.join('');
};
IR.panel = {
  show(id) { IR.$('#' + id).classList.remove('hidden'); },
  hide(id) { IR.$('#' + id).classList.add('hidden'); },
  wire() { document.querySelectorAll('[data-close]').forEach(b => b.addEventListener('click', () => b.closest('.veil').classList.add('hidden'))); addEventListener('keydown', e => { if (e.key === 'Escape') document.querySelectorAll('.veil.closable:not(.hidden)').forEach(v => v.classList.add('hidden')); }); },
};
IR.wireAudioControls = (audio, save) => {
  const s = save.data.settings, bM = IR.$('#bMusic'), bS = IR.$('#bSfx'), vol = IR.$('#vol');
  const sync = () => { bM.classList.toggle('off', !s.music); bS.classList.toggle('off', !s.sfx); vol.value = s.vol; };
  bM.onclick = () => { s.music = !s.music; save.write(); sync(); audio.unlock(); s.music ? audio.start() : audio.stop(); };
  bS.onclick = () => { s.sfx = !s.sfx; save.write(); sync(); audio.setVol(); };
  vol.oninput = () => { s.vol = +vol.value; save.write(); audio.setVol(); };
  addEventListener('pointerdown', () => audio.unlock(), { capture: true });
  addEventListener('keydown', e => { audio.unlock(); if ((e.key === 'm' || e.key === 'M') && !/INPUT|TEXTAREA/.test(document.activeElement.tagName)) bM.click(); });
  sync();
};
IR.loadChangelog = async () => { try { const r = await fetch('CHANGELOG.md', { cache: 'no-store' }); return await r.text(); } catch (e) { return ''; } };

/* speech bubble / toast helpers */
IR.bubble = (parent, text, x, y, cls = '', ms = 2200) => {
  const b = IR.el('div', 'bubble ' + cls, parent, text); b.style.left = x + 'px'; b.style.top = y + 'px';
  requestAnimationFrame(() => b.classList.add('show'));
  if (ms) setTimeout(() => { b.classList.remove('show'); setTimeout(() => b.remove(), 400); }, ms / (IR.speed || 1));
  return b;
};

/* confetti / sparkle particle canvas */
IR.Particles = class {
  constructor(canvas) { this.c = canvas; this.x = canvas.getContext('2d'); this.p = []; this.running = false; }
  resize(w, h, s) { this.c.width = w * s; this.c.height = h * s; this.s = s; }
  burst(x, y, n, colors, o = {}) {
    for (let i = 0; i < n; i++) { const a = (o.angle ?? -Math.PI / 2) + (Math.random() - .5) * (o.spread ?? Math.PI * 1.6), v = (o.speed || 520) * (.4 + Math.random() * .8);
      this.p.push({ x, y, vx: Math.cos(a) * v, vy: Math.sin(a) * v, g: o.gravity ?? 900, life: o.life || 2.2, t: 0, c: colors[i % colors.length], s: (o.size || 7) * (.6 + Math.random() * .8), r: Math.random() * 6, vr: (Math.random() - .5) * 12, shape: o.shape || 'rect' }); }
    if (!this.running) { this.running = true; this.last = performance.now(); requestAnimationFrame(t => this.step(t)); }
  }
  step(now) {
    const dt = Math.min(.05, (now - this.last) / 1000); this.last = now; const x = this.x, s = this.s || 1;
    x.setTransform(1, 0, 0, 1, 0, 0); x.clearRect(0, 0, this.c.width, this.c.height); x.setTransform(s, 0, 0, s, 0, 0);
    this.p = this.p.filter(p => { p.t += dt; p.vy += p.g * dt; p.vx *= .992; p.x += p.vx * dt; p.y += p.vy * dt; p.r += p.vr * dt;
      const a = Math.max(0, 1 - p.t / p.life); x.globalAlpha = a; x.save(); x.translate(p.x, p.y); x.rotate(p.r); x.fillStyle = p.c;
      if (p.shape === 'circle') { x.beginPath(); x.arc(0, 0, p.s / 2, 0, 7); x.fill(); } else if (p.shape === 'ring') { x.beginPath(); x.arc(0, 0, p.s, 0, 7); x.strokeStyle = p.c; x.lineWidth = 2; x.stroke(); } else x.fillRect(-p.s / 2, -p.s / 4, p.s, p.s / 2);
      x.restore(); return p.t < p.life; });
    x.globalAlpha = 1;
    if (this.p.length) requestAnimationFrame(t => this.step(t)); else { this.running = false; x.setTransform(1, 0, 0, 1, 0, 0); x.clearRect(0, 0, this.c.width, this.c.height); }
  }
};

return IR;
})();
