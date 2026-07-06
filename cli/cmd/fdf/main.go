package main

import (
	"fmt"
	"os"
)

// version is set by goreleaser via -ldflags "-X main.version=...".
var version = "0.2.0-dev"

func main() {
	if len(os.Args) > 1 && os.Args[1] == "version" {
		fmt.Println("fdf", version)
		return
	}
	fmt.Fprintln(os.Stderr, "fdf: unknown command; usage comes in later tasks")
	os.Exit(2)
}
