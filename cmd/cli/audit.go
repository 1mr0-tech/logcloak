package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"sigs.k8s.io/yaml"

	"github.com/1mr0-tech/logcloak/pkg/masker"
	"github.com/1mr0-tech/logcloak/pkg/patterns"
	"github.com/1mr0-tech/logcloak/pkg/rules"
)

// auditGapHit is one log line where the ground-truth built-in scan found PII
// that the given policy's configured builtin patterns would not have masked.
type auditGapHit struct {
	LineNum int
	Display string
	Gaps    []string
}

// auditResult summarizes an audit run for both reporting and testing.
type auditResult struct {
	TotalLines   int
	LinesCovered int
	GapHits      []auditGapHit
	GapCounts    map[string]int
}

// parseMaskingPolicy loads a MaskingPolicy from a YAML file on disk.
func parseMaskingPolicy(path string) (rules.MaskingPolicy, error) {
	var policy rules.MaskingPolicy
	b, err := os.ReadFile(path)
	if err != nil {
		return policy, fmt.Errorf("read policy: %w", err)
	}
	if err := yaml.Unmarshal(b, &policy); err != nil {
		return policy, fmt.Errorf("parse policy: %w", err)
	}
	if len(policy.Spec.Patterns) == 0 {
		return policy, fmt.Errorf("policy has no patterns in spec.patterns")
	}
	return policy, nil
}

// runAudit compares a compiled MaskingPolicy against the full built-in
// pattern library over every line of in, reporting lines where the ground
// truth scan finds PII that the policy's configured builtin patterns miss.
func runAudit(policy rules.MaskingPolicy, in io.Reader) (auditResult, error) {
	policyRules, err := rules.Merge([]rules.MaskingPolicy{policy}, nil)
	if err != nil {
		return auditResult{}, fmt.Errorf("compile policy: %w", err)
	}
	policyMasker := masker.New(policyRules)

	covered := make(map[string]bool)
	for _, p := range policy.Spec.Patterns {
		if p.Builtin != "" {
			covered[p.Builtin] = true
		}
	}

	var groundTruthRules []masker.Rule
	for name, b := range patterns.Library {
		groundTruthRules = append(groundTruthRules, masker.Rule{Name: name, Pattern: b.Pattern, Replace: "[REDACTED:" + name + "]"})
	}
	groundTruthMasker := masker.New(groundTruthRules)

	result := auditResult{GapCounts: make(map[string]int)}

	scanner := bufio.NewScanner(in)
	for scanner.Scan() {
		line := scanner.Text()
		result.TotalLines++

		policyMasked, policyMatched := policyMasker.MaskLine(line)
		_, groundTruthMatched := groundTruthMasker.MaskLine(line)

		if len(policyMatched) > 0 {
			result.LinesCovered++
		}

		var gaps []string
		for _, name := range groundTruthMatched {
			if !covered[name] {
				gaps = append(gaps, name)
			}
		}
		if len(gaps) == 0 {
			continue
		}
		for _, g := range gaps {
			result.GapCounts[g]++
		}

		display := policyMasked
		for _, g := range gaps {
			b, ok := patterns.Get(g)
			if !ok {
				continue
			}
			display = b.Pattern.ReplaceAllStringFunc(display, func(s string) string {
				return "\033[31m" + s + "\033[0m"
			})
		}
		result.GapHits = append(result.GapHits, auditGapHit{result.TotalLines, display, gaps})
	}
	if err := scanner.Err(); err != nil {
		return result, fmt.Errorf("audit: %w", err)
	}
	return result, nil
}

func printAuditReport(w io.Writer, policyPath string, r auditResult) {
	for _, h := range r.GapHits {
		_, _ = fmt.Fprintf(w, "line %-6d %s  \033[90m[MISSING: %s]\033[0m\n", h.LineNum, h.Display, strings.Join(h.Gaps, ", "))
	}

	sep := strings.Repeat("─", 56)
	_, _ = fmt.Fprintf(w, "\n%s\n", sep)
	_, _ = fmt.Fprintf(w, "logcloak audit summary — policy: %s\n", policyPath)
	_, _ = fmt.Fprintf(w, "%s\n", sep)
	_, _ = fmt.Fprintf(w, "Total lines:              %d\n", r.TotalLines)
	_, _ = fmt.Fprintf(w, "Lines masked by policy:   %d\n", r.LinesCovered)
	_, _ = fmt.Fprintf(w, "Lines with exposed PII:   %d", len(r.GapHits))
	if r.TotalLines > 0 {
		_, _ = fmt.Fprintf(w, " (%.1f%%)", float64(len(r.GapHits))/float64(r.TotalLines)*100)
	}
	_, _ = fmt.Fprintln(w)
	if len(r.GapCounts) > 0 {
		_, _ = fmt.Fprintf(w, "Missing built-in patterns:\n")
		type kv struct {
			name  string
			count int
		}
		sorted := make([]kv, 0, len(r.GapCounts))
		for k, v := range r.GapCounts {
			sorted = append(sorted, kv{k, v})
		}
		sort.Slice(sorted, func(i, j int) bool { return sorted[i].count > sorted[j].count })
		for _, item := range sorted {
			_, _ = fmt.Fprintf(w, "  %-20s %d line(s)\n", item.name+":", item.count)
		}
	}
	_, _ = fmt.Fprintf(w, "%s\n", sep)
	if len(r.GapCounts) > 0 {
		names := make([]string, 0, len(r.GapCounts))
		for k := range r.GapCounts {
			names = append(names, k)
		}
		sort.Strings(names)
		_, _ = fmt.Fprintf(w, "\nAdd these to %s → spec.patterns:\n", policyPath)
		for _, n := range names {
			_, _ = fmt.Fprintf(w, "  - name: %s\n    builtin: %s\n", n, n)
		}
	} else if r.TotalLines > 0 {
		_, _ = fmt.Fprintf(w, "\nNo gaps found against the built-in pattern library for this sample.\n")
	}
}

// cmdAudit checks a MaskingPolicy against real log samples and reports gaps:
// PII that the full built-in pattern library would catch but the given
// policy's configured builtin patterns would not. No cluster needed.
func cmdAudit(args []string) {
	if len(args) == 0 {
		_, _ = fmt.Fprintln(os.Stderr, "usage: logcloak audit <maskingpolicy.yaml> [file]")
		os.Exit(1)
	}

	policy, err := parseMaskingPolicy(args[0])
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	var in *os.File
	if len(args) >= 2 {
		in, err = os.Open(args[1])
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "open: %v\n", err)
			os.Exit(1)
		}
		defer in.Close() //nolint:errcheck
	} else {
		in = os.Stdin
	}

	result, err := runAudit(policy, in)
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	printAuditReport(os.Stdout, args[0], result)
}
