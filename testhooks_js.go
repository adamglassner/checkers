//go:build js

// Test hooks for the browser build only (used by tools/webtest). Not in the Windows game.
package main

import (
	"encoding/json"
	"syscall/js"
)

var testApp *App

func init() {
	js.Global().Set("__ck", js.ValueOf(map[string]any{
		"state": js.FuncOf(func(this js.Value, args []js.Value) any {
			a := testApp
			if a == nil || a.G == nil {
				return "{}"
			}
			var legal []string
			for _, m := range a.legal {
				legal = append(legal, a.pos.Notation(m))
			}
			var res any
			if a.G.Result != nil {
				res = a.G.Result
			}
			moves := make([]string, len(a.hist))
			for i, h := range a.hist {
				moves[i] = h.n
			}
			b, _ := json.Marshal(map[string]any{"side": a.pos.Side, "busy": a.busy, "anims": len(a.anims), "legal": legal, "moves": moves,
				"result": res, "modal": a.ui.topModal(), "sel": a.sel, "hops": a.hops, "flipped": a.flipped, "you": a.yourSide(), "mode": a.G.Mode,
				"status": func() string { s, _ := a.ui.statusText(); return s }(), "stats": a.sd.Stats, "copied": js.Global().Get("__copied").String()})
			return string(b)
		}),
		// square centre in page pixels (1-based square number)
		"sq": js.FuncOf(func(this js.Value, args []js.Value) any {
			a := testApp
			x, y := a.sqXY(args[0].Int() - 1)
			dx, dy := a.ui.dev(BX+x+SQ/2, BY+y+SQ/2)
			r := js.Global().Get("devicePixelRatio").Float()
			return []any{dx / r, dy / r}
		}),
		// logical point to page pixels
		"pt": js.FuncOf(func(this js.Value, args []js.Value) any {
			dx, dy := testApp.ui.dev(args[0].Float(), args[1].Float())
			r := js.Global().Get("devicePixelRatio").Float()
			return []any{dx / r, dy / r}
		}),
		"setFen": js.FuncOf(func(this js.Value, args []js.Value) any {
			a := testApp
			a.G.FEN, a.G.Moves, a.G.Result = args[0].String(), []string{}, nil
			a.rebuild()
			a.ui.dirty()
			return nil
		}),
		"play": js.FuncOf(func(this js.Value, args []js.Value) any {
			a := testApp
			if m, ok := a.findLegal(args[0].String()); ok {
				a.playMove(m, 0)
				return true
			}
			return false
		}),
		"hits": js.FuncOf(func(this js.Value, args []js.Value) any {
			u := testApp.ui
			hs := u.hits
			if u.topModal() != "" {
				hs = u.modalHits
			}
			out := map[string]any{}
			r := js.Global().Get("devicePixelRatio").Float()
			for _, h := range hs {
				if h.fn == nil {
					continue
				}
				dx, dy := u.dev(h.x+h.w/2, h.y+h.h/2)
				out[h.id] = []any{dx / r, dy / r}
			}
			b, _ := json.Marshal(out)
			return string(b)
		}),
	}))
}
