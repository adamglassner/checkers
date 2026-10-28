// Ironite game shell.
// Serves the game's web files on 127.0.0.1, opens them in an app-mode
// browser window (Microsoft Edge, built into Windows 10/11) and keeps the
// player's save file in %APPDATA%\Ironite\<Game>\save.json, outside the
// install folder, so updates and uninstalls never touch it.
package main

import (
	"embed"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"
)

// Set at build time with -ldflags "-X main.gameID=GoFish -X main.title=... -X main.version=...".
var (
	gameID  = "Game"
	title   = "Ironite Game"
	version = "0.0.0-dev"
)

//go:embed web
var webFS embed.FS

var (
	lastPing atomic.Int64
	saveMu   sync.Mutex
)

func dataDir() string { // %APPDATA%\Ironite\<Game>
	d, err := os.UserConfigDir()
	if err != nil {
		d = os.TempDir()
	}
	return filepath.Join(d, "Ironite", gameID)
}

func cacheDir() string { // %LOCALAPPDATA%\Ironite\<Game>
	d, err := os.UserCacheDir()
	if err != nil {
		d = os.TempDir()
	}
	return filepath.Join(d, "Ironite", gameID)
}

func routes(addr string) http.Handler {
	mux := http.NewServeMux()
	sub, _ := fs.Sub(webFS, "web")
	files := http.FileServer(http.FS(sub))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		files.ServeHTTP(w, r)
	})
	mux.HandleFunc("/api/ping", func(w http.ResponseWriter, r *http.Request) {
		lastPing.Store(time.Now().Unix())
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("/api/info", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"version":%q,"saveDir":%q}`, version, dataDir())
	})
	mux.HandleFunc("/api/save", func(w http.ResponseWriter, r *http.Request) {
		path := filepath.Join(dataDir(), "save.json")
		saveMu.Lock()
		defer saveMu.Unlock()
		switch r.Method {
		case http.MethodGet:
			b, err := os.ReadFile(path)
			if err != nil {
				http.Error(w, "no save yet", http.StatusNotFound)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Cache-Control", "no-store")
			_, _ = w.Write(b)
		case http.MethodPost:
			// Only the game page itself may write (a custom header can't be
			// sent cross-origin without a preflight, which we never allow).
			if r.Header.Get("X-Ironite") != "1" {
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}
			b, err := io.ReadAll(io.LimitReader(r.Body, 2<<20))
			if err != nil || len(b) == 0 || b[0] != '{' {
				http.Error(w, "bad save", http.StatusBadRequest)
				return
			}
			_ = os.MkdirAll(dataDir(), 0o755)
			tmp := path + ".tmp"
			if err := os.WriteFile(tmp, b, 0o644); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			if err := os.Rename(tmp, path); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusNoContent)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})
	// Refuse requests addressed to any other host name (DNS rebinding).
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Host != addr {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		mux.ServeHTTP(w, r)
	})
}

func main() {
	if len(os.Args) > 1 && (os.Args[1] == "--version" || os.Args[1] == "/version") {
		fmt.Println(version)
		return
	}
	_ = os.MkdirAll(cacheDir(), 0o755)
	if lf, err := os.OpenFile(filepath.Join(cacheDir(), "log.txt"), os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644); err == nil {
		log.SetOutput(lf)
	}
	log.Printf("%s %s starting", title, version)
	if !singleInstance() {
		return
	}

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		fatal(title + " could not start: " + err.Error())
	}
	addr := ln.Addr().String()
	go func() { _ = http.Serve(ln, routes(addr)) }()
	url := "http://" + addr + "/"

	if os.Getenv("IRONITE_DEV") != "" {
		fmt.Println(url)
		select {}
	}

	started := time.Now()
	if wait, ok := openWindow(url); ok {
		wait()
		if time.Since(started) > 5*time.Second {
			return // window closed
		}
	} else {
		openDefaultBrowser(url)
	}
	// The browser handed the window to an already-running process: stay
	// alive while the page keeps checking in.
	for {
		time.Sleep(3 * time.Second)
		lp := lastPing.Load()
		if (lp == 0 && time.Since(started) > 2*time.Minute) || (lp != 0 && time.Since(time.Unix(lp, 0)) > 2*time.Minute) {
			return
		}
	}
}
