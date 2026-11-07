//go:build js

package main

import "syscall/js"

func setClipboard(s string) bool {
	js.Global().Set("__copied", s)
	nav := js.Global().Get("navigator")
	if cb := nav.Get("clipboard"); !cb.IsUndefined() {
		cb.Call("writeText", s)
	}
	return true
}
