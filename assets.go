// Embedded assets: the baked graphics, fonts and changelog. (c) 2026 Ironite
package main

import (
	"bytes"
	"embed"
	"image"
	"image/jpeg"
	"image/png"
	"strings"
	"sync"
)

//go:embed assets/game assets/fonts CHANGELOG.md
var assetFS embed.FS

var (
	imgMu  sync.Mutex
	images = map[string]image.Image{}
)

// Img returns a decoded embedded image from assets/game (decoded once).
func Img(name string) image.Image {
	imgMu.Lock()
	defer imgMu.Unlock()
	if im, ok := images[name]; ok {
		return im
	}
	b, err := assetFS.ReadFile("assets/game/" + name)
	if err != nil {
		panic(err)
	}
	var im image.Image
	if strings.HasSuffix(name, ".jpg") {
		im, err = jpeg.Decode(bytes.NewReader(b))
	} else {
		im, err = png.Decode(bytes.NewReader(b))
	}
	if err != nil {
		panic(name + ": " + err.Error())
	}
	im = toRGBA(im)
	images[name] = im
	return im
}

// PieceImg returns the image for a piece code (1 black man, 2 black king, -1 red man, -2 red king).
func PieceImg(set string, code int8) image.Image {
	side := "b"
	if code < 0 {
		side = "r"
	}
	n := "1"
	if abs8(code) == King {
		n = "2"
	}
	return Img("piece-" + set + "-" + side + n + ".png")
}

func changelogText() string {
	b, _ := assetFS.ReadFile("CHANGELOG.md")
	return string(b)
}

// Preload decodes everything up front (in the background) so the first frames are quick.
func preloadAssets() {
	ents, _ := assetFS.ReadDir("assets/game")
	for _, e := range ents {
		Img(e.Name())
	}
}
