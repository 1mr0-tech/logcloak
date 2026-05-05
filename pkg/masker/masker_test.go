package masker_test

import (
	"regexp"
	"testing"

	"github.com/1mr0-tech/logcloak/pkg/masker"
)

func email() masker.Rule {
	return masker.Rule{
		Name:    "email",
		Pattern: regexp.MustCompile(`[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}`),
		Replace: "[REDACTED]",
	}
}

func otp() masker.Rule {
	return masker.Rule{
		Name:    "otp",
		Pattern: regexp.MustCompile(`\b[0-9]{6}\b`),
		Replace: "[REDACTED]",
	}
}

func TestMaskLine_NoMatch(t *testing.T) {
	m := masker.New([]masker.Rule{email()})
	masked, matched := m.MaskLine("hello world")
	if len(matched) > 0 {
		t.Errorf("expected no match, got %v, masked=%q", matched, masked)
	}
	if masked != "hello world" {
		t.Errorf("expected original line, got %q", masked)
	}
}

func TestMaskLine_EmailMatch(t *testing.T) {
	m := masker.New([]masker.Rule{email()})
	masked, matched := m.MaskLine("user@example.com logged in")
	if len(matched) == 0 {
		t.Error("expected at least one matched rule")
	}
	if matched[0] != "email" {
		t.Errorf("expected matched rule name 'email', got %q", matched[0])
	}
	if masked != "[REDACTED] logged in" {
		t.Errorf("got %q", masked)
	}
}

func TestMaskLine_MultiplePatterns(t *testing.T) {
	m := masker.New([]masker.Rule{email(), otp()})
	masked, matched := m.MaskLine("user@example.com OTP=123456")
	if len(matched) != 2 {
		t.Errorf("expected 2 matched rules, got %d: %v", len(matched), matched)
	}
	if masked != "[REDACTED] OTP=[REDACTED]" {
		t.Errorf("got %q", masked)
	}
}

func TestMaskLine_EmptyLine(t *testing.T) {
	m := masker.New([]masker.Rule{email()})
	masked, matched := m.MaskLine("")
	if len(matched) > 0 {
		t.Error("expected no match on empty line")
	}
	if masked != "" {
		t.Errorf("got %q", masked)
	}
}

func TestMaskLine_NoRules(t *testing.T) {
	m := masker.New(nil)
	masked, matched := m.MaskLine("sensitive@data.com")
	if len(matched) > 0 {
		t.Error("no rules should cause no match")
	}
	if masked != "sensitive@data.com" {
		t.Errorf("got %q", masked)
	}
}

func TestMaskLine_ReturnsMatchedRuleNames(t *testing.T) {
	m := masker.New([]masker.Rule{email(), otp()})
	_, matched := m.MaskLine("contact@example.com your otp is 837261")
	names := make(map[string]bool)
	for _, n := range matched {
		names[n] = true
	}
	if !names["email"] {
		t.Error("expected 'email' in matched rules")
	}
	if !names["otp"] {
		t.Error("expected 'otp' in matched rules")
	}
}

func TestMaskLine_OnlyMatchedRulesReturned(t *testing.T) {
	m := masker.New([]masker.Rule{email(), otp()})
	_, matched := m.MaskLine("no sensitive data here")
	if len(matched) != 0 {
		t.Errorf("expected empty matched, got %v", matched)
	}
}

func TestMaskLine_DollarSignInReplacement(t *testing.T) {
	// redactWith containing "$1" must be treated as a literal string,
	// not a capture-group back-reference — otherwise PII leaks through.
	m := masker.New([]masker.Rule{{
		Name:    "email",
		Pattern: regexp.MustCompile(`[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}`),
		Replace: "$1",
	}})
	masked, matched := m.MaskLine("user@example.com logged in")
	if len(matched) == 0 {
		t.Fatal("expected rule to match")
	}
	if masked != "$1 logged in" {
		t.Errorf("expected literal $1 replacement, got %q", masked)
	}
}

func TestMaskLine_LargeLine(t *testing.T) {
	// 500KB line should not panic or block
	large := make([]byte, 512*1024)
	for i := range large {
		large[i] = 'x'
	}
	copy(large[100:], "user@example.com")
	m := masker.New([]masker.Rule{email()})
	masked, matched := m.MaskLine(string(large))
	if len(matched) == 0 {
		t.Error("expected email match in large line")
	}
	if len(masked) == 0 {
		t.Error("masked line should not be empty")
	}
}
