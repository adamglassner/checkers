global.self = global; require('../web/checkers.js'); const { Position } = self.Checkers;
const p = new Position(); const want = [7, 49, 302, 1469, 7361, 36768, 179740, 845931];
let ok = true; want.forEach((n, i) => { const g = p.perft(i + 1); if (g !== n) { ok = false; } console.log(i + 1, g, n, g === n ? 'ok' : 'FAIL'); });
const k = p.key(); p.computeHash(); console.log('hash', k === p.key(), p.fen());
// multi-jump + crowning test
const q = new Position('B:W6,7,15,23:B1'); console.log(q.legal().map(m => q.notation(m)));
const r = new Position('B:WK18,K27,26,19:BK10'); console.log(r.legal().map(m => r.notation(m)));
console.log(ok ? 'PERFT OK' : 'PERFT FAIL');
