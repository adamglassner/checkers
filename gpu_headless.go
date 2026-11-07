//go:build headless

// Test build with no window or sound (see net_test.go).
package main

import "image"

type gpuImg = *struct{}

func uploadRGBA(rgba *image.RGBA, old gpuImg) gpuImg { return nil }
func freeImg(g gpuImg)                               {}
