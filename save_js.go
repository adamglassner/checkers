//go:build js

// Browser build (used for automated testing): saves go to localStorage.
package main

import (
	"errors"
	"syscall/js"
)

func readSaveFile() ([]byte, error) {
	v := js.Global().Get("localStorage").Call("getItem", "checkers.save")
	if v.IsNull() || v.IsUndefined() {
		return nil, errors.New("no save")
	}
	return []byte(v.String()), nil
}

func writeSaveFile(b []byte) error {
	js.Global().Get("localStorage").Call("setItem", "checkers.save", string(b))
	return nil
}
