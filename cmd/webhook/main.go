package main

import (
	"fmt"
	"os"
)

var version = "dev"

func main() {
	fmt.Fprintf(os.Stderr, "logcloak-webhook %s\n", version)
	// TODO: start mutating admission webhook server
	os.Exit(0)
}
