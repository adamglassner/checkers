//go:build !headless

package main

import (
	"image"

	"github.com/hajimehoshi/ebiten/v2"
)

// windowIcons: the game icon at the sizes Windows asks for (taskbar, title bar, Alt+Tab).
func windowIcons() []image.Image {
	src := Img("icon-256.png")
	var out []image.Image
	for _, s := range []int{16, 24, 32, 48, 64, 128, 256} {
		out = append(out, Scaled(src, s, s, false))
	}
	return out
}

func runGame(g *Game) {
	if err := ebiten.RunGameWithOptions(g, &ebiten.RunGameOptions{}); err != nil {
		panic(err)
	}
}
