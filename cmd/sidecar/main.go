package main

import (
	"fmt"
	"os"
)

var version = "dev"

func main() {
	fmt.Fprintf(os.Stderr, "logcloak-sidecar %s\n", version)
	// TODO: open FIFO, read lines, apply masking rules, write to stdout
	os.Exit(0)
}
