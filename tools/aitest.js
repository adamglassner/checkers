// Self-play between levels in Node; checks the AI never picks an illegal move and stronger beats weaker.
global.self = global; require('../web/checkers.js'); require('../web/ai.js');
const K = self.Checkers, AI = self.CheckersAI;
const L = { tucker: { maxDepth: 2, noise: 60, blunder: .25 }, opal: { maxDepth: 3, noise: 30, blunder: .1 }, earl: { maxDepth: 5, noise: 12, blunder: .03 }, hattie: { time: 150, maxDepth: 9 }, doc: { time: 300 }, gus: { time: 600 } };
function game(a, b) {
  const pos = new K.Position(), moves = [];
  for (let ply = 0; ply < 300; ply++) {
    const st = pos.status(); if (st.over) return { winner: st.winner, reason: st.reason, plies: ply };
    const lvl = pos.side === K.BLACK ? a : b;
    const r = AI.think(pos.clone(), Object.assign({}, L[lvl]));
    const m = pos.legal().find(x => pos.notation(x) === r.move);
    if (!m) throw new Error('illegal ' + r.move);
    pos.make(m); moves.push(r.move);
  }
  return { winner: 0, reason: 'cap', plies: 300 };
}
const pairs = (process.argv[2] || 'tucker:gus,gus:tucker,opal:earl,earl:opal,earl:hattie,hattie:earl,hattie:gus,gus:hattie').split(',');
for (const p of pairs) { const [a, b] = p.split(':'); const t = Date.now(); const r = game(a, b); console.log(`${a}(B) vs ${b}(R):`, r.winner === 1 ? a : r.winner === -1 ? b : 'draw', r.reason, r.plies, ((Date.now() - t) / 1000).toFixed(1) + 's'); }
