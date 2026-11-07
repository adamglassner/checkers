//go:build headless

package main

import "fmt"

// The headless build only exists for tests: go test -tags headless .
func main() { fmt.Println("Checkers test build. Run: go test -tags headless .") }
