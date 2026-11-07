// Saved settings, the unfinished game and the player's record. Stored as
// JSON in %APPDATA%\Ironite\Checkers\save.json (the same file and format the
// first version of the game used, so existing saves carry over).
// (c) 2026 Ironite
package main

import (
	"encoding/json"
	"sync"
	"time"
)

type Settings struct {
	Music  bool    `json:"music"`
	Sfx    bool    `json:"sfx"`
	Vol    float64 `json:"vol"`
	Board  string  `json:"board"`
	Pieces string  `json:"pieces"`
	Dots   string  `json:"dots"`
	Nums   string  `json:"nums"`
	Anim   string  `json:"anim"`
	Turn   string  `json:"turn"`
	// two players on one computer, and online play
	HotNames [2]string `json:"hotNames"`
	HotFaces [2]string `json:"hotFaces"`
	Name     string    `json:"name"`
	Face     string    `json:"face"`
	Host     string    `json:"host"`
}

type Setup struct {
	Mode   string `json:"mode"`
	Opp    string `json:"opp"`
	Color  string `json:"color"`
	Forced string `json:"forced"`
}

type Result struct {
	Reason string `json:"reason"`
	Winner int    `json:"winner"`
}

type GameRec struct {
	FEN      string   `json:"fen"`
	Moves    []string `json:"moves"`
	Mode     string   `json:"mode"`
	Opp      string   `json:"opp"`
	YouBlack bool     `json:"youBlack"`
	Forced   *bool    `json:"forced,omitempty"`
	Result   *Result  `json:"result"`
	Counted  bool     `json:"counted"`
	Flipped  bool     `json:"flipped"`
	Started  int64    `json:"started"`
	OppFull  string   `json:"oppFull,omitempty"` // the opponent's name this game (computer games)
	Names    []string `json:"names,omitempty"`   // Black's and Red's names (two-player games)
	Faces    []string `json:"faces,omitempty"`
	Online   string   `json:"-"` // "host" or "guest" while playing online (never saved)
}

func (g *GameRec) forced() bool { return g == nil || g.Forced == nil || *g.Forced }

type Rec struct {
	W int `json:"w"`
	D int `json:"d"`
	L int `json:"l"`
}

type Stats struct {
	Won   int             `json:"won"`
	Drawn int             `json:"drawn"`
	Lost  int             `json:"lost"`
	Pvp   int             `json:"pvp"`
	Net   Rec             `json:"online"`
	Per   map[string]*Rec `json:"per"`
}

type SaveData struct {
	Settings Settings `json:"settings"`
	Setup    Setup    `json:"setup"`
	Game     *GameRec `json:"game"`
	Stats    Stats    `json:"stats"`
}

func defaultSave() *SaveData {
	return &SaveData{
		Settings: Settings{Music: true, Sfx: true, Vol: 60, Board: "barrel", Pieces: "classic", Dots: "on", Nums: "off", Anim: "normal", Turn: "on",
			HotFaces: [2]string{"you", "p2"}, Face: "you"},
		Setup: Setup{Mode: "ai", Opp: "tucker", Color: "black", Forced: "on"},
		Stats: Stats{Per: map[string]*Rec{}},
	}
}

var (
	saveMu       sync.Mutex
	saveTimer    *time.Timer
	saveDisabled bool // tests
)

func loadSave() *SaveData {
	sd := defaultSave()
	if b, err := readSaveFile(); err == nil && len(b) > 0 {
		_ = json.Unmarshal(b, sd) // fields missing from an older save keep their defaults
	}
	if sd.Stats.Per == nil {
		sd.Stats.Per = map[string]*Rec{}
	}
	st := &sd.Settings
	for i := range st.HotFaces {
		if !validFace(st.HotFaces[i]) {
			st.HotFaces[i] = [2]string{"you", "p2"}[i]
		}
	}
	if !validFace(st.Face) {
		st.Face = "you"
	}
	if sd.Setup.Mode != "ai" && sd.Setup.Mode != "pvp" && sd.Setup.Mode != "online" {
		sd.Setup.Mode = "ai"
	}
	if sd.Game != nil && sd.Game.FEN == "" {
		sd.Game.FEN = StartFEN
	}
	return sd
}

// write saves soon (debounced), or right away with now=true.
func (sd *SaveData) write(now bool) {
	if saveDisabled {
		return
	}
	b, err := json.Marshal(sd)
	if err != nil {
		return
	}
	saveMu.Lock()
	defer saveMu.Unlock()
	if saveTimer != nil {
		saveTimer.Stop()
	}
	if now {
		_ = writeSaveFile(b)
		return
	}
	saveTimer = time.AfterFunc(400*time.Millisecond, func() { _ = writeSaveFile(b) })
}
