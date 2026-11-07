// The regulars at the Hollow Creek General Store, their difficulty, their
// sayings, and the colours and names of each piece set. (c) 2026 Ironite
package main

import (
	"image/color"
	"math/rand"
	"time"
)

type Folk struct {
	ID, Name, Full, Rating, Tag string
	Level                       int
}

// The six regulars are the six difficulty levels. Each keeps their face,
// but their name is drawn at random (always one that fits the face) each
// time you sit down to a new game.
type folkBase struct {
	ID, Rating, Tag string
	Level           int
	Female          bool
	Title           string   // "Miss", "Deacon"... prefixed to the name
	Names           []string // first names (or "Doc" surnames)
}

var folkBases = []folkBase{
	{"tucker", "Beginner", "The grocer's grandson. Sweeps the porch and crowns anything he can.", 1, false, "Little", []string{"Tucker", "Billy", "Jody", "Sammy", "Wyatt", "Cody", "Otis"}},
	{"opal", "Novice", "Runs the post office counter. Chatty, friendly and a little reckless.", 2, true, "Miss", []string{"Opal", "Pearl", "Ruby", "Dottie", "Mabel", "Edna", "Violet"}},
	{"earl", "Club player", "Sunday best every day of the week. Never rushes a move.", 3, false, "Deacon", []string{"Earl", "Amos", "Clement", "Booker", "Moses", "Luther", "Silas"}},
	{"hattie", "Strong", "Owns the store. Plays between customers and rarely loses one.", 4, true, "", []string{"Hattie Mae", "Lula Belle", "Ida Jean", "Etta Mae", "Nora Faye", "Callie Jo", "Ruby Lee"}},
	{"doc", "Expert", "The county vet. Calm hands, sharp eyes and sharper traps.", 5, false, "Doc", []string{"Whitfield", "Abernathy", "Pruitt", "Holloway", "McCall", "Langley", "Bishop"}},
	{"gus", "Champion", "County champion since anyone can remember. Whittles while he waits.", 6, false, "Old", []string{"Gus", "Ike", "Jeb", "Zeke", "Lem", "Cal", "Ned"}},
}

// Folks are the six regulars with this session's names.
var Folks = rollNames()

func makeFolk(b folkBase, name string) Folk {
	full := name
	if b.Title != "" {
		full = b.Title + " " + name
	}
	short := full
	if b.ID == "tucker" {
		short = name // "Little Tucker" is just Tucker in conversation
	}
	return Folk{ID: b.ID, Name: short, Full: full, Rating: b.Rating, Tag: b.Tag, Level: b.Level}
}

// rollNames gives every regular a fresh random name.
func rollNames() []Folk {
	out := make([]Folk, len(folkBases))
	for i, b := range folkBases {
		out[i] = makeFolk(b, b.Names[rand.Intn(len(b.Names))])
	}
	return out
}

func folkByID(id string) Folk {
	for _, f := range Folks {
		if f.ID == id {
			return f
		}
	}
	return Folks[0]
}

// folkNamed is the regular `id` with a particular full name (from a saved game).
func folkNamed(id, full string) Folk {
	f := folkByID(id)
	if full == "" {
		return f
	}
	for _, b := range folkBases {
		if b.ID != id {
			continue
		}
		for _, n := range b.Names {
			if g := makeFolk(b, n); g.Full == full {
				return g
			}
		}
	}
	return f
}

// Faces a person can pick for themselves in two-player and online games.
var FaceChoices = []string{"you", "p2", "tucker", "opal", "earl", "hattie", "doc", "gus"}

func validFace(f string) bool {
	for _, c := range FaceChoices {
		if c == f {
			return true
		}
	}
	return false
}

// Level: search settings and how long the opponent "thinks" at least (ms).
type Level struct {
	AI    AIOpts
	Delay [2]int
}

var Levels = map[string]Level{
	"tucker": {AIOpts{MaxDepth: 2, Noise: 70, Blunder: .25}, [2]int{700, 1500}},
	"opal":   {AIOpts{MaxDepth: 3, Noise: 32, Blunder: .1}, [2]int{700, 1400}},
	"earl":   {AIOpts{MaxDepth: 6, Noise: 12, Blunder: .03}, [2]int{600, 1200}},
	"hattie": {AIOpts{Time: 450 * time.Millisecond, MaxDepth: 11}, [2]int{500, 900}},
	"doc":    {AIOpts{Time: 1300 * time.Millisecond}, [2]int{300, 700}},
	"gus":    {AIOpts{Time: 2800 * time.Millisecond}, [2]int{200, 500}},
}

var Quips = map[string]map[string][]string{
	"tucker": {"greet": {"Grandpa says I can play till supper!", "I'm gonna get a king this time!"}, "crown": {"KING ME!", "Crown him! Crown him!"}, "jump": {"Boing boing boing!", "Double jump! Did you see that?"}, "win": {"I won! Wait till I tell Grandpa!"}, "lose": {"Good game! One more? Please?"}, "draw": {"Nobody won? That's weird."}, "accept": {"Okay! Want a candy stick?"}, "decline": {"No way, I'm winning!"}},
	"opal":   {"greet": {"Oh good, the mail's sorted. Your move, hon.", "Just one game, then I've got stamps to sell."}, "crown": {"King me, hon!", "Ooh, a crown! Special delivery."}, "jump": {"Return to sender!", "Well, would you look at that."}, "win": {"Signed, sealed, delivered."}, "lose": {"Well played! I'll put it in the newsletter."}, "draw": {"Even Steven."}, "accept": {"A draw it is. Peppermint?"}, "decline": {"Not just yet, dear."}},
	"earl":   {"greet": {"Bless this board. Begin when you are ready.", "Patience, friend. Patience."}, "crown": {"A crown, by grace.", "King me, if you would."}, "jump": {"The meek shall be jumped.", "Mm. Two at once."}, "win": {"Slow and steady, as always."}, "lose": {"A fine game. I shall reflect on it Sunday."}, "draw": {"Peace be with us both."}, "accept": {"Peace. I accept."}, "decline": {"I believe there is more to play."}},
	"hattie": {"greet": {"Let me ring this up and I'm all yours.", "Board's set. Mind the barrel, it wobbles."}, "crown": {"King me, sugar.", "That's a crown, honey."}, "jump": {"Cleaned out, like the candy jar on a Saturday.", "That'll cost you."}, "win": {"House always wins, sugar."}, "lose": {"Well, I'll be. Coffee's on me."}, "draw": {"Call it even and buy a soda."}, "accept": {"Fair enough. Draw."}, "decline": {"Not a chance, sugar."}},
	"doc":    {"greet": {"Steady hands. Let's begin.", "I've seen worse positions on a sick mule. Your move."}, "crown": {"King, please.", "Crowned."}, "jump": {"A clean procedure.", "Textbook."}, "win": {"Prognosis: you lose. Good game."}, "lose": {"Remarkable. Truly."}, "draw": {"A stable condition. Draw."}, "accept": {"Agreed. The position is level."}, "decline": {"No. There's life in it yet."}},
	"gus":    {"greet": {"Mm. Sit down.", "Fifty years on this barrel. Go ahead."}, "crown": {"King.", "Crown it."}, "jump": {"Mm-hm.", "Saw that coming in nineteen fifty-two."}, "win": {"Again, whenever you like."}, "lose": {"...Well. Ain't that something. Good game."}, "draw": {"A draw. You played it proper."}, "accept": {"Fair."}, "decline": {"Nope."}},
}

// Piece sets: the names of the two sides and the colour used for each in the score sheet.
type PieceSet struct {
	ID, Name   string
	Black, Red string
	BCol, RCol color.NRGBA
	Caps       bool
}

var Sets = []PieceSet{
	{"classic", "Red & Black", "Black", "Red", Hex("#1d1d22"), Hex("#b51f1c"), false},
	{"wood", "Walnut & Maple", "Walnut", "Maple", Hex("#5a361c"), Hex("#e2c28a"), false},
	{"caps", "Bottle Caps", "Blue", "Red", Hex("#23488e"), Hex("#c2241d"), true},
}

func setByID(id string) PieceSet {
	for _, s := range Sets {
		if s.ID == id {
			return s
		}
	}
	return Sets[0]
}

type BoardTheme struct {
	ID, Name string
	Num      color.NRGBA // square-number ink
}

var Boards = []BoardTheme{
	{"barrel", "Barrelhead", RGBA(240, 220, 180, .42)},
	{"parlor", "Parlor Red", RGBA(240, 210, 150, .4)},
	{"tourney", "Tournament Green", RGBA(235, 245, 225, .45)},
}

func boardByID(id string) BoardTheme {
	for _, b := range Boards {
		if b.ID == id {
			return b
		}
	}
	return Boards[0]
}

var TuneNames = []string{"Potbelly Stove Waltz", "Kings Row Breakdown", "Penny Candy Rag", "Sweet Tea on the Porch"}

// Theme colours (from the original stylesheet).
var (
	cInk     = Hex("#2b1a0e")
	cPaper   = Hex("#f7ecd4")
	cPaper2  = Hex("#e8d3a8")
	cAccent  = Hex("#8e2a1e")
	cGold    = Hex("#eac46a")
	cBar1    = Hex("#5a3a1e")
	cBar2    = Hex("#2a180a")
	cBarText = Hex("#f3e0bb")
	cBtn1    = Hex("#f4e2b8")
	cBtn2    = Hex("#cfa461")
	cBtnInk  = Hex("#2b170a")
	cBtnEdge = Hex("#6e4520")
	cPage    = Hex("#1a0f06")
	cCream   = Hex("#fbe9c4")
)
