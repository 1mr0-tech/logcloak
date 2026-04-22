package rules

import (
	"encoding/json"
	"fmt"
	"regexp"

	"github.com/1mr0-tech/logcloak/pkg/masker"
	"github.com/1mr0-tech/logcloak/pkg/patterns"
)

// SerializedRule is the wire format for LOGCLOAK_RULES env var.
type SerializedRule struct {
	Name    string `json:"name"`
	Pattern string `json:"pattern"`
	Replace string `json:"replace"`
}

// Merge combines MaskingPolicy rules and pod annotation rules into compiled masker.Rules.
// CRD rules are always included; annotation rules extend but cannot remove them.
// Deduplication is by Name — first occurrence wins.
func Merge(policies []MaskingPolicy, annotationSpecs []PatternSpec) ([]masker.Rule, error) {
	seen := make(map[string]bool)
	var result []masker.Rule

	add := func(spec PatternSpec, redactWith string) error {
		if seen[spec.Name] {
			return nil
		}
		if redactWith == "" {
			redactWith = "[REDACTED]"
		}
		rule, err := compileSpec(spec, redactWith)
		if err != nil {
			return err
		}
		seen[spec.Name] = true
		result = append(result, rule)
		return nil
	}

	for _, p := range policies {
		for _, spec := range p.Spec.Patterns {
			if err := add(spec, p.Spec.RedactWith); err != nil {
				return nil, err
			}
		}
	}

	for _, spec := range annotationSpecs {
		if err := add(spec, "[REDACTED]"); err != nil {
			return nil, err
		}
	}

	return result, nil
}

func compileSpec(spec PatternSpec, replace string) (masker.Rule, error) {
	if spec.Builtin != "" {
		b, ok := patterns.Get(spec.Builtin)
		if !ok {
			return masker.Rule{}, fmt.Errorf("unknown built-in pattern %q", spec.Builtin)
		}
		return masker.Rule{Name: spec.Name, Pattern: b.Pattern, Replace: replace}, nil
	}
	if spec.Regex != "" {
		re, err := regexp.Compile(spec.Regex)
		if err != nil {
			return masker.Rule{}, fmt.Errorf("invalid regex for %q: %w", spec.Name, err)
		}
		return masker.Rule{Name: spec.Name, Pattern: re, Replace: replace}, nil
	}
	return masker.Rule{}, fmt.Errorf("pattern %q has neither builtin nor regex", spec.Name)
}

// Serialize converts compiled rules into the JSON string injected as LOGCLOAK_RULES.
func Serialize(rules []masker.Rule) (string, error) {
	var sr []SerializedRule
	for _, r := range rules {
		sr = append(sr, SerializedRule{
			Name:    r.Name,
			Pattern: r.Pattern.String(),
			Replace: r.Replace,
		})
	}
	b, err := json.Marshal(sr)
	return string(b), err
}

// Deserialize parses LOGCLOAK_RULES JSON back into masker.Rules.
func Deserialize(jsonStr string) ([]masker.Rule, error) {
	if jsonStr == "" {
		return nil, nil
	}
	var sr []SerializedRule
	if err := json.Unmarshal([]byte(jsonStr), &sr); err != nil {
		return nil, fmt.Errorf("parse LOGCLOAK_RULES: %w", err)
	}
	var result []masker.Rule
	for _, s := range sr {
		re, err := regexp.Compile(s.Pattern)
		if err != nil {
			return nil, fmt.Errorf("recompile pattern %q: %w", s.Name, err)
		}
		result = append(result, masker.Rule{Name: s.Name, Pattern: re, Replace: s.Replace})
	}
	return result, nil
}
