//go:build !windows

package main

import (
	"fmt"
	"os"
)

// Non-Windows stubs, used for development and testing only.
func fatal(text string)                { fmt.Fprintln(os.Stderr, text); os.Exit(1) }
func singleInstance() bool             { return true }
func openWindow(string) (func(), bool) { return nil, false }
func openDefaultBrowser(url string)    { fmt.Println("Open:", url) }
