package main

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"

	"github.com/1mr0-tech/logcloak/pkg/masker"
	"github.com/1mr0-tech/logcloak/pkg/patterns"
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
	case "scan":
		cmdScan(os.Args[2:])
	case "audit":
		cmdAudit(os.Args[2:])
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
  logcloak scan [file]               Audit logs for PII using all built-in patterns
  logcloak audit <policy.yaml> [file] Check a MaskingPolicy for coverage gaps against real logs
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
		defer in.Close() //nolint:errcheck
	} else {
		in = os.Stdin
	}

	scanner := bufio.NewScanner(in)
	for scanner.Scan() {
		line := scanner.Text()
		masked, matched := m.MaskLine(line)
		if len(matched) > 0 {
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

// cmdScan audits a log file (or stdin) against all built-in PII patterns and
// prints a summary of what logcloak would mask. No Kubernetes cluster needed.
func cmdScan(args []string) {
	var in *os.File
	var err error
	if len(args) >= 1 {
		in, err = os.Open(args[0])
		if err != nil {
			fmt.Fprintf(os.Stderr, "open: %v\n", err)
			os.Exit(1)
		}
		defer in.Close() //nolint:errcheck
	} else {
		in = os.Stdin
	}

	var rules []masker.Rule
	for name, b := range patterns.Library {
		rules = append(rules, masker.Rule{
			Name:    name,
			Pattern: b.Pattern,
			Replace: "[REDACTED:" + name + "]",
		})
	}
	m := masker.New(rules)

	type hit struct {
		lineNum int
		masked  string
		matched []string
	}
	var hits []hit
	patternCounts := make(map[string]int)
	totalLines := 0

	scanner := bufio.NewScanner(in)
	for scanner.Scan() {
		line := scanner.Text()
		totalLines++
		masked, matched := m.MaskLine(line)
		if len(matched) > 0 {
			hits = append(hits, hit{totalLines, masked, matched})
			for _, name := range matched {
				patternCounts[name]++
			}
		}
	}
	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "scan: %v\n", err)
		os.Exit(1)
	}

	for _, h := range hits {
		fmt.Printf("line %-6d \033[33m%s\033[0m  \033[90m[%s]\033[0m\n",
			h.lineNum, h.masked, strings.Join(h.matched, ", "))
	}

	sep := strings.Repeat("─", 56)
	fmt.Printf("\n%s\n", sep)
	fmt.Printf("logcloak scan summary\n")
	fmt.Printf("%s\n", sep)
	fmt.Printf("Total lines:     %d\n", totalLines)
	pct := 0.0
	if totalLines > 0 {
		pct = float64(len(hits)) / float64(totalLines) * 100
	}
	fmt.Printf("Lines with PII:  %d (%.1f%%)\n", len(hits), pct)
	if len(patternCounts) > 0 {
		fmt.Printf("Pattern hits:\n")
		type kv struct {
			name  string
			count int
		}
		sorted := make([]kv, 0, len(patternCounts))
		for k, v := range patternCounts {
			sorted = append(sorted, kv{k, v})
		}
		sort.Slice(sorted, func(i, j int) bool {
			return sorted[i].count > sorted[j].count
		})
		for _, item := range sorted {
			fmt.Printf("  %-20s %d\n", item.name+":", item.count)
		}
	}
	fmt.Printf("%s\n", sep)
	if len(hits) > 0 {
		fmt.Printf("\nProtect these logs → kubectl label namespace <ns> logcloak.io/inject=true\n")
	}
}
