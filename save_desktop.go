//go:build !js

package main

import (
	"os"
	"path/filepath"
)

func saveDir() string { // %APPDATA%\Ironite\Checkers
	d, err := os.UserConfigDir()
	if err != nil {
		d = os.TempDir()
	}
	return filepath.Join(d, "Ironite", "Checkers")
}

func readSaveFile() ([]byte, error) { return os.ReadFile(filepath.Join(saveDir(), "save.json")) }

// writeSaveFile writes atomically, so a crash mid-write never corrupts the save.
func writeSaveFile(b []byte) error {
	if err := os.MkdirAll(saveDir(), 0o755); err != nil {
		return err
	}
	tmp := filepath.Join(saveDir(), "save.json.tmp")
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, filepath.Join(saveDir(), "save.json"))
}
