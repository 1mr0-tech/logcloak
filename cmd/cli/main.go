package main

import (
	"fmt"
	"os"
)

var version = "dev"

func main() {
	fmt.Fprintf(os.Stderr, "logcloak-cli %s\n", version)
	// TODO: kubectl plugin for validating regex patterns and previewing masking output
	os.Exit(0)
}
