package main

import (
	"bufio"
	"fmt"
	"os"
	"regexp"

	"github.com/1mr0-tech/logcloak/pkg/masker"
	"github.com/1mr0-tech/logcloak/pkg/regex"
)

var version = "dev"

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "validate":
		cmdValidate(os.Args[2:])
	case "preview":
		cmdPreview(os.Args[2:])
	case "version":
		fmt.Printf("logcloak-cli %s\n", version)
	default:
		usage()
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, `logcloak — kubectl plugin for log masking

Usage:
  logcloak validate <regex>          Validate that a regex is RE2-safe
  logcloak preview <regex> [file]    Preview masking output (reads stdin if no file)
  logcloak version                   Print version`)
}

func cmdValidate(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: logcloak validate <regex>")
		os.Exit(1)
	}
	if err := regex.Validate(args[0]); err != nil {
		fmt.Fprintf(os.Stderr, "INVALID: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("OK: %q is a valid RE2 pattern\n", args[0])
}

func cmdPreview(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: logcloak preview <regex> [file]")
		os.Exit(1)
	}
	pattern := args[0]
	if err := regex.Validate(pattern); err != nil {
		fmt.Fprintf(os.Stderr, "invalid regex: %v\n", err)
		os.Exit(1)
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		fmt.Fprintf(os.Stderr, "compile error: %v\n", err)
		os.Exit(1)
	}

	m := masker.New([]masker.Rule{{Name: "preview", Pattern: re, Replace: "[REDACTED]"}})

	var in *os.File
	if len(args) >= 2 {
		in, err = os.Open(args[1])
		if err != nil {
			fmt.Fprintf(os.Stderr, "open file: %v\n", err)
			os.Exit(1)
		}
		defer in.Close()
	} else {
		in = os.Stdin
	}

	scanner := bufio.NewScanner(in)
	for scanner.Scan() {
		line := scanner.Text()
		masked, changed := m.MaskLine(line)
		if changed {
			fmt.Printf("\033[33m%s\033[0m\n", masked)
		} else {
			fmt.Println(line)
		}
	}
	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "read error: %v\n", err)
		os.Exit(1)
	}
}
