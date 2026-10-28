/* Checkers music and sound effects. Four original old-time tunes played live
   by the game: fiddle, banjo, mandolin, harmonica, guitar, upright bass,
   washboard and a ragtime parlor piano. Plus the sounds of checkers on a
   board (wooden or bottle caps). (c) 2026 Ironite */
'use strict';
const MUSIC = (() => {
const parseMel = (s, at) => { const ev = []; let t = at; s.split(/\s+/).filter(x => x && x !== '|').forEach(tok => { const [n, d] = tok.split(':'); if (n !== 'R') ev.push({ n, t, d: +d }); t += +d; }); return ev; };

function arrange(tune) {
  const ev = [], B = tune.beats;
  const rhythm = (bars, at, soft = false, feel = tune.feel) => {
    bars.forEach((sym, bi) => {
      const c = IR.chord(sym), t0 = at + bi * B, v = soft ? .7 : 1;
      const root = IR.pcAbove(c.root, 40), fifth = IR.pcAbove((c.root + 7) % 12, 40);
      const strum = c.pcs.map(pc => IR.pcAbove(pc, 52)).concat([IR.pcAbove(c.root, 52) + 12]).sort((a, b) => a - b);
      if (feel === 'waltz') {
        ev.push({ t: t0, i: 'bass', m: bi % 2 ? fifth : root, d: 1, v });
        ev.push({ t: t0 + 1, i: 'strum', ms: strum, v: .7 * v }); ev.push({ t: t0 + 2, i: 'strum', ms: strum, v: .6 * v });
        if (!soft) ev.push({ t: t0, i: 'banjo', m: IR.pcAbove(c.pcs[2] ?? c.root, 62), v: .35 });
      } else if (feel === 'breakdown') {
        ev.push({ t: t0, i: 'bass', m: root, d: .9, v }); ev.push({ t: t0 + 2, i: 'bass', m: fifth, d: .9, v: .85 * v });
        ev.push({ t: t0 + 1, i: 'chop', ms: strum, v: .8 * v }); ev.push({ t: t0 + 3, i: 'chop', ms: strum, v: .75 * v });
        // forward banjo roll through the chord
        const hi = c.pcs.map(pc => IR.pcAbove(pc, 62)).sort((a, b) => a - b), roll = [hi[0], hi[1], hi[2] || hi[0] + 12, hi[0] + 12, hi[1], hi[2] || hi[0] + 12, hi[0] + 12, hi[1]];
        if (!soft) roll.forEach((m, k) => ev.push({ t: t0 + k * .5, i: 'banjo', m, v: (k % 3 === 0 ? .5 : .32) }));
        [1, 3].forEach(o => ev.push({ t: t0 + o, i: 'wash', v: .9 * v })); [.5, 1.5, 2.5, 3.5].forEach(o => ev.push({ t: t0 + o, i: 'wash', v: .4 * v }));
      } else if (feel === 'rag') {
        ev.push({ t: t0, i: 'pLow', m: IR.pcAbove(c.root, 31), d: .9, v: .9 * v }); ev.push({ t: t0, i: 'pLow', m: IR.pcAbove(c.root, 31) + 12, d: .9, v: .5 * v });
        const mid = c.pcs.map(pc => IR.pcAbove(pc, 55)).sort((a, b) => a - b);
        ev.push({ t: t0 + 1, i: 'pChord', ms: mid, d: .7, v: .5 * v });
        ev.push({ t: t0 + 2, i: 'pLow', m: IR.pcAbove((c.root + 7) % 12, 31), d: .9, v: .75 * v });
        ev.push({ t: t0 + 3, i: 'pChord', ms: mid, d: .7, v: .45 * v });
        if (!soft) ev.push({ t: t0 + 1, i: 'wash', v: .35 }), ev.push({ t: t0 + 3, i: 'wash', v: .35 });
      } else { // slow porch fingerpicking
        const hi = c.pcs.map(pc => IR.pcAbove(pc, 59)).sort((a, b) => a - b);
        [[0, root, .9], [1, hi[0], .5], [1.5, hi[1], .45], [2, fifth, .75], [2.5, hi[2] || hi[1], .45], [3, hi[1], .5], [3.5, hi[0], .4]].forEach(([o, m, vv]) => ev.push({ t: t0 + o, i: 'pick', m, v: vv * v }));
        ev.push({ t: t0, i: 'bassSoft', m: IR.pcAbove(c.root, 33), d: B * .9, v: .5 * v });
      }
    });
    return at + bars.length * B;
  };
  const lead = (mel, at, inst, shift = 0) => parseMel(mel, at).forEach(e => ev.push({ t: e.t, i: inst, m: IR.n2m(e.n) + shift, d: e.d, v: 1 }));
  let t = rhythm(tune.intro, 0, true);
  for (const [sec, inst, shift] of tune.form) { lead(tune[sec].mel, t, inst, shift || 0); t = rhythm(tune[sec].ch, t); }
  t = rhythm(tune.tag, t, true);
  ev.push({ t: t - B, i: 'banjo', m: IR.pcAbove(IR.chord(tune.tag[tune.tag.length - 1]).root, 67), v: .5 });
  ev.sort((a, b) => a.t - b.t);
  return { name: tune.name, bpm: tune.bpm, swing: tune.swing || 0, events: ev, len: t, gap: 3 };
}

const TUNES = [
  { name: 'Potbelly Stove Waltz', bpm: 108, beats: 3, feel: 'waltz',
    intro: ['G', 'D', 'G', 'G'],
    A: { ch: ['G', 'G', 'C', 'G', 'D', 'D', 'G', 'G', 'G', 'G7', 'C', 'Am', 'D', 'D7', 'G', 'G'],
      mel: 'D4:1 G4:1 B4:1 | D5:2 B4:1 | E5:2 C5:1 | B4:3 | A4:1 B4:1 C5:1 | D5:2 F#4:1 | G4:3 | G4:2 D4:1 | G4:1 B4:1 D5:1 | F5:2 D5:1 | E5:1.5 D5:.5 C5:1 | A4:3 | D5:1 C5:1 A4:1 | F#4:2 A4:1 | G4:3 | R:3' },
    B: { ch: ['C', 'C', 'G', 'G', 'A7', 'A7', 'D', 'D7'],
      mel: 'E5:2 G5:1 | E5:2 C5:1 | D5:1 B4:1 G4:1 | B4:3 | C#5:1 E5:1 A5:1 | G5:2 E5:1 | F#5:2 E5:.5 D5:.5 | C5:2 A4:1' },
    form: [['A', 'fiddle'], ['B', 'mandolin'], ['A', 'harmonica']], tag: ['C', 'D', 'G'] },
  { name: "Kings Row Breakdown", bpm: 124, beats: 4, feel: 'breakdown',
    intro: ['D', 'A', 'D', 'D'],
    A: { ch: ['D', 'D', 'G', 'D', 'D', 'D', 'A', 'D'],
      mel: 'F#5:.5 E5:.5 D5:.5 E5:.5 F#5:.5 A5:.5 F#5:.5 E5:.5 | D5:.5 E5:.5 F#5:.5 G5:.5 A5:1 A5:1 | B5:.5 A5:.5 G5:.5 A5:.5 B5:.5 D6:.5 B5:.5 A5:.5 | A5:.5 F#5:.5 E5:.5 D5:.5 F#5:1 A5:1 | F#5:.5 E5:.5 D5:.5 E5:.5 F#5:.5 A5:.5 B5:.5 A5:.5 | F#5:.5 A5:.5 D6:1 A5:1 F#5:1 | E5:.5 F#5:.5 G5:.5 F#5:.5 E5:.5 D5:.5 C#5:.5 E5:.5 | D5:2 D5:1 R:1' },
    B: { ch: ['G', 'G', 'D', 'D', 'A', 'A', 'D', 'D'],
      mel: 'B5:1 B5:.5 A5:.5 G5:.5 A5:.5 B5:1 | G5:.5 B5:.5 D6:1 B5:1 G5:1 | A5:1 A5:.5 G5:.5 F#5:.5 G5:.5 A5:1 | F#5:.5 A5:.5 D6:1 A5:1 F#5:1 | E5:.5 F#5:.5 G5:.5 A5:.5 B5:.5 A5:.5 G5:.5 E5:.5 | C#5:.5 E5:.5 A5:1 G5:1 E5:1 | F#5:.5 E5:.5 D5:.5 E5:.5 F#5:.5 G5:.5 A5:.5 F#5:.5 | D5:2 D5:1 R:1' },
    form: [['A', 'fiddle'], ['A', 'banjoLead', -12], ['B', 'fiddle'], ['B', 'mandolin'], ['A', 'fiddle']], tag: ['A', 'D'] },
  { name: 'Penny Candy Rag', bpm: 92, beats: 4, feel: 'rag', swing: .18,
    intro: ['C', 'G7', 'C', 'G7'],
    A: { ch: ['C', 'C', 'G7', 'G7', 'C', 'A7', 'D7', 'G7'],
      mel: 'E5:.5 G5:.5 C6:1 A5:.5 G5:.5 E5:1 | D#5:.5 E5:.5 G5:1 E5:1 C5:1 | D5:.5 F5:.5 B5:1 A5:.5 G5:.5 F5:1 | D5:.5 E5:.5 F5:.5 G5:.5 B4:2 | E5:.5 G5:.5 C6:1 E6:1 C6:1 | C#6:.5 A5:.5 G5:.5 E5:.5 C#5:2 | D5:.5 F#5:.5 A5:1 C6:1 A5:1 | G5:1 F5:.5 D5:.5 B4:2' },
    B: { ch: ['F', 'F', 'C', 'C', 'D7', 'G7', 'C', 'C'],
      mel: 'A5:.5 C6:.5 F6:1 C6:1 A5:1 | G#5:.5 A5:.5 C6:.5 A5:.5 F5:2 | G5:.5 E5:.5 C5:1 E5:.5 G5:.5 C6:1 | C6:2 R:2 | F#5:.5 A5:.5 D6:1 C6:1 A5:1 | B5:.5 G5:.5 F5:.5 D5:.5 B4:2 | C5:.5 E5:.5 G5:.5 C6:.5 E6:1 C6:1 | C6:3 R:1' },
    form: [['A', 'piano'], ['B', 'piano'], ['A', 'mandolin', -12], ['B', 'piano']], tag: ['F', 'G7', 'C'] },
  { name: 'Sweet Tea on the Porch', bpm: 72, beats: 4, feel: 'pick',
    intro: ['F', 'Bb', 'C', 'F'],
    A: { ch: ['F', 'Dm', 'Bb', 'C', 'F', 'Dm', 'C', 'F'],
      mel: 'A4:2 C5:1 F5:1 | F5:1.5 E5:.5 D5:2 | D5:1 F5:1 Bb5:1.5 A5:.5 | G5:4 | A5:1.5 G5:.5 F5:1 C5:1 | D5:2 F5:1 A5:1 | G5:1 E5:1 C5:1.5 E5:.5 | F5:4' },
    B: { ch: ['Bb', 'Bb', 'F', 'F', 'Gm', 'C', 'F', 'F'],
      mel: 'D5:1 F5:1 Bb5:2 | A5:1 G5:1 F5:2 | C5:1 F5:1 A5:1.5 G5:.5 | F5:4 | Bb4:1 D5:1 G5:1.5 F5:.5 | E5:1 G5:1 C6:2 | A5:1.5 G5:.5 F5:1 C5:1 | F5:4' },
    form: [['A', 'harmonica'], ['B', 'fiddle', -12], ['A', 'harmonica']], tag: ['Bb', 'C', 'F'] },
];
// sanity: every melody fits its chords
TUNES.forEach(t => ['A', 'B'].forEach(k => { const s = t[k]; const len = s.mel.split(/\s+/).filter(x => x && x !== '|').reduce((a, tok) => a + +tok.split(':')[1], 0); if (Math.abs(len - s.ch.length * t.beats) > .01) console.warn('tune length', t.name, k, len, s.ch.length * t.beats); }));

const instruments = {
  bass(t, e, d, v) { this.bass(t, e.m, .7 * v, Math.min(d, .8)); },
  bassSoft(t, e, d, v) { this.bass(t, e.m, .5 * v, d); },
  strum(t, e, d, v) { e.ms.forEach((m, i) => this.pluck(t + i * .013, m, .12 * v, { decay: .9965, bright: .6, dur: 1.3, lp: 3200, body: 240 })); },
  chop(t, e, d, v) { e.ms.forEach((m, i) => this.pluck(t + i * .008, m, .1 * v, { decay: .985, bright: .7, dur: .25, lp: 3600, body: 260 })); },
  pick(t, e, d, v) { this.pluck(t, e.m, .22 * v, { decay: .998, bright: .45, dur: 2.2, lp: 3000, body: 230 }); },
  banjo(t, e, d, v) { this.pluck(t, e.m, .2 * v, { decay: .9935, bright: .95, dur: .7, lp: 7000, body: 700 }); },
  banjoLead(t, e, d, v) { this.pluck(t, e.m + 12, .3 * v, { decay: .994, bright: .95, dur: .8, lp: 7000, body: 700 }); },
  mandolin(t, e, d, v) { // tremolo on long notes
    const n = d > .45 ? Math.max(2, Math.floor(d / .075)) : 1;
    for (let k = 0; k < n; k++) this.pluck(t + k * .075, e.m, (k ? .1 : .17) * v, { decay: .996, bright: .85, dur: .5, lp: 5500, body: 500 });
  },
  fiddle(t, e, d, v) { this.fiddle(t, e.m, .9 * v, d * .95); },
  harmonica(t, e, d, v) { this.harmonica(t, e.m, .9 * v, d * .92); },
  piano(t, e, d, v) { this.piano(t, e.m, 1.5 * v, d); },
  pLow(t, e, d, v) { this.piano(t, e.m, 1 * v, d); },
  pChord(t, e, d, v) { e.ms.forEach((m, i) => this.piano(t + i * .005, m, .55 * v, d)); },
  wash(t, e, d, v) { this.noiseHit(t, .1 * v, { freq: 6500, q: 1.2, decay: .045, type: 'bandpass' }); this.noiseHit(t + .012, .05 * v, { freq: 9000, type: 'highpass', decay: .03 }); },
};

/* ---------------------------------------------------------------- sounds */
function sfx(audio, getSet) {
  // A turned wooden checker set down on the board: a round, hollow "thock".
  // Bottle caps give a brighter, tinny clink instead.
  const thock = function (t, o, v = 1, pitch = 1) {
    if (getSet && getSet() === 'caps') {
      v *= 2;
      this.noiseHit(t, .4 * v, { freq: 3800 * pitch, q: 6, decay: .03, out: o });
      [2310, 3570, 5120].forEach((f, i) => { const ac = this.ac, os = ac.createOscillator(), g = ac.createGain(); os.type = 'sine'; os.frequency.value = f * pitch * (1 + Math.random() * .02); g.gain.setValueAtTime(.05 * v / (i + 1), t); g.gain.exponentialRampToValueAtTime(.0004, t + .18 - i * .03); os.connect(g); g.connect(o); os.start(t); os.stop(t + .2); });
      this.noiseHit(t, .3 * v, { type: 'lowpass', freq: 600, decay: .03, out: o });
      return;
    }
    v *= 3;
    this.noiseHit(t, .6 * v, { type: 'lowpass', freq: 420 * pitch, decay: .06, out: o });
    this.noiseHit(t, .3 * v, { freq: 1100 * pitch, q: 3, decay: .05, out: o });
    const ac = this.ac, os = ac.createOscillator(), g = ac.createGain(); os.type = 'sine'; os.frequency.setValueAtTime(330 * pitch, t); os.frequency.exponentialRampToValueAtTime(260 * pitch, t + .08);
    g.gain.setValueAtTime(.16 * v, t); g.gain.exponentialRampToValueAtTime(.0005, t + .12); os.connect(g); g.connect(o); os.start(t); os.stop(t + .14);
  };
  const slide = function (t, o, v = 1) { this.noiseHit(t, .16 * v, { freq: 1400, q: .7, attack: .03, decay: .16, sweep: 700, out: o }); };
  return {
    pick() { audio.sfx(function (t, o) { thock.call(this, t, o, .25, 1.6); }); },
    move() { audio.sfx(function (t, o) { slide.call(this, t, o, .6); thock.call(this, t + .05, o, 1, .95 + Math.random() * .1); }); },
    hop(i = 0) { audio.sfx(function (t, o) { thock.call(this, t, o, .95, 1 + i * .06); }); },
    capture(n = 1) { audio.sfx(function (t, o) { for (let k = 0; k < n; k++) { const tt = t + .12 + k * .09; slide.call(this, tt, o, .9); thock.call(this, tt + .06, o, .45, .75); } }); },
    crown() { audio.sfx(function (t, o) { thock.call(this, t, o, 1, 1.15); thock.call(this, t + .08, o, .8, 1.25); [67, 71, 74, 79].forEach((m, i) => this.pluck(t + .15 + i * .07, m, .22, { decay: .995, bright: .95, dur: 1, lp: 7000, body: 700, out: o })); this.bell(t + .45, 86, .55, o, 1.6); }); },
    mustJump() { audio.sfx(function (t, o) { this.bell(t, 76, .45, o, 1); this.bell(t + .1, 76, .35, o, 1); }); },
    illegal() { audio.sfx(function (t, o) { this.noiseHit(t, .7, { type: 'lowpass', freq: 220, decay: .07, out: o }); }); },
    click() { audio.sfx(function (t, o) { this.noiseHit(t, .45, { freq: 3800, q: 2.5, decay: .018, out: o }); }); },
    hint() { audio.sfx(function (t, o) { this.bell(t, 84, .45, o, 1.2); this.bell(t + .09, 88, .35, o, 1.2); }); },
    start() { audio.sfx(function (t, o) { [62, 66, 69, 74].forEach((m, i) => this.pluck(t + i * .06, m, .22, { decay: .995, bright: .95, dur: 1, lp: 7000, body: 700, out: o })); }); },
    win() { audio.sfx(function (t, o) { [67, 71, 74, 79, 83, 86].forEach((m, i) => this.pluck(t + i * .08, m, .24, { decay: .995, bright: .95, dur: 1.2, lp: 7000, body: 700, out: o })); this.fiddle(t + .5, 79, 1, 1.2, o); [79, 83, 86, 91].forEach((m, i) => this.bell(t + .5 + i * .1, m, .6, o, 2)); }); },
    lose() { audio.sfx(function (t, o) { [64, 60, 57, 52].forEach((m, i) => this.pluck(t + i * .22, m, .22, { decay: .997, bright: .6, dur: 2, lp: 3000, body: 400, out: o })); }); },
    draw() { audio.sfx(function (t, o) { [67, 72].forEach((m, i) => this.bell(t + i * .18, m, .6, o, 1.8)); }); },
  };
}

return { tunes: () => TUNES.map(arrange), instruments, sfx, TUNES };
})();
