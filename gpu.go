//go:build !headless

package main

import (
	"image"

	"github.com/hajimehoshi/ebiten/v2"
)

type gpuImg = *ebiten.Image

// uploadRGBA copies pixels to a GPU image (reusing old when it's the right size).
func uploadRGBA(rgba *image.RGBA, old gpuImg) gpuImg {
	w, h := rgba.Rect.Dx(), rgba.Rect.Dy()
	if old == nil || old.Bounds().Dx() != w || old.Bounds().Dy() != h {
		freeImg(old)
		old = ebiten.NewImage(w, h)
	}
	old.WritePixels(rgba.Pix)
	return old
}

func freeImg(g gpuImg) {
	if g != nil {
		g.Deallocate()
	}
}
