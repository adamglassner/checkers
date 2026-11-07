//go:build windows

package main

import (
	"syscall"
	"unicode/utf16"
	"unsafe"
)

var (
	user32           = syscall.NewLazyDLL("user32.dll")
	kernel32         = syscall.NewLazyDLL("kernel32.dll")
	openClipboard    = user32.NewProc("OpenClipboard")
	closeClipboard   = user32.NewProc("CloseClipboard")
	emptyClipboard   = user32.NewProc("EmptyClipboard")
	setClipboardData = user32.NewProc("SetClipboardData")
	globalAlloc      = kernel32.NewProc("GlobalAlloc")
	globalLock       = kernel32.NewProc("GlobalLock")
	globalUnlock     = kernel32.NewProc("GlobalUnlock")
	moveMemory       = kernel32.NewProc("RtlMoveMemory")
)

// setClipboard puts text on the Windows clipboard (for "Copy moves").
func setClipboard(s string) bool {
	if r, _, _ := openClipboard.Call(0); r == 0 {
		return false
	}
	defer closeClipboard.Call()
	emptyClipboard.Call()
	u := utf16.Encode([]rune(s + "\x00"))
	const gmemMoveable, cfUnicodeText = 0x0002, 13
	h, _, _ := globalAlloc.Call(gmemMoveable, uintptr(len(u)*2))
	if h == 0 {
		return false
	}
	p, _, _ := globalLock.Call(h)
	if p == 0 {
		return false
	}
	moveMemory.Call(p, uintptr(unsafe.Pointer(&u[0])), uintptr(len(u)*2))
	globalUnlock.Call(h)
	r, _, _ := setClipboardData.Call(cfUnicodeText, h)
	return r != 0
}
