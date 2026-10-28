//go:build windows

package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"unsafe"
)

var (
	user32          = syscall.NewLazyDLL("user32.dll")
	kernel32        = syscall.NewLazyDLL("kernel32.dll")
	pMessageBox     = user32.NewProc("MessageBoxW")
	pFindWindow     = user32.NewProc("FindWindowW")
	pShowWindow     = user32.NewProc("ShowWindow")
	pSetForeground  = user32.NewProc("SetForegroundWindow")
	pCreateMutex    = kernel32.NewProc("CreateMutexW")
	pSetDpiAwareCtx = user32.NewProc("SetProcessDpiAwarenessContext")
)

func u16(s string) *uint16 { p, _ := syscall.UTF16PtrFromString(s); return p }

func msgBox(text string, flags uintptr) {
	pMessageBox.Call(0, uintptr(unsafe.Pointer(u16(text))), uintptr(unsafe.Pointer(u16(title))), flags)
}
func fatal(text string) { msgBox(text, 0x10); os.Exit(1) }

// singleInstance returns false (after bringing the open window forward) if
// the game is already running.
func singleInstance() bool {
	if pSetDpiAwareCtx.Find() == nil {
		pSetDpiAwareCtx.Call(^uintptr(3)) // PER_MONITOR_AWARE_V2
	}
	_, _, err := pCreateMutex.Call(0, 0, uintptr(unsafe.Pointer(u16(`Local\Ironite`+gameID))))
	if err == syscall.Errno(183) { // ERROR_ALREADY_EXISTS
		if h, _, _ := pFindWindow.Call(0, uintptr(unsafe.Pointer(u16(title)))); h != 0 {
			pShowWindow.Call(h, 9)
			pSetForeground.Call(h)
		} else {
			msgBox(title+" is already open.", 0x40)
		}
		return false
	}
	return true
}

func findBrowser() string {
	var roots []string
	for _, k := range []string{"ProgramFiles(x86)", "ProgramFiles", "LOCALAPPDATA"} {
		if v := os.Getenv(k); v != "" {
			roots = append(roots, v)
		}
	}
	for _, rel := range []string{`Microsoft\Edge\Application\msedge.exe`, `Google\Chrome\Application\chrome.exe`, `BraveSoftware\Brave-Browser\Application\brave.exe`} {
		for _, r := range roots {
			p := filepath.Join(r, rel)
			if _, err := os.Stat(p); err == nil {
				return p
			}
		}
	}
	return ""
}

// openWindow starts an app-mode window with its own browser profile, so it
// runs as its own process: the game (and the launcher's play timer) ends
// when the player closes the window.
func openWindow(url string) (func(), bool) {
	b := findBrowser()
	if b == "" {
		return nil, false
	}
	profile := filepath.Join(cacheDir(), "WebView")
	_ = os.MkdirAll(profile, 0o755)
	cmd := exec.Command(b,
		"--app="+url,
		"--user-data-dir="+profile,
		"--window-size=1440,900",
		"--autoplay-policy=no-user-gesture-required",
		"--no-first-run", "--no-default-browser-check", "--disable-extensions",
		"--disable-background-timer-throttling", "--disable-renderer-backgrounding",
		"--disable-features=Translate,msEdgeSidebarV2,msUndersideButton,msHubApps",
		"--disable-pinch", "--overscroll-history-navigation=0",
	)
	if err := cmd.Start(); err != nil {
		return nil, false
	}
	return func() { _ = cmd.Wait() }, true
}

func openDefaultBrowser(url string) {
	_ = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
}
