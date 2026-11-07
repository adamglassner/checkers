//go:build !windows && !js

package main

func setClipboard(s string) bool { return false }
