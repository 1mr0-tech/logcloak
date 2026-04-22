package main

import (
	"fmt"
	"os"
)

var version = "dev"

func main() {
	fmt.Fprintf(os.Stderr, "logcloak-controller %s\n", version)
	// TODO: start controller-runtime manager, watch MaskingPolicy CRDs and pod annotations
	os.Exit(0)
}
