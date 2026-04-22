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
	masked, changed := m.MaskLine("hello world")
	if changed {
		t.Errorf("expected no change, got %q", masked)
	}
	if masked != "hello world" {
		t.Errorf("expected original line, got %q", masked)
	}
}

func TestMaskLine_EmailMatch(t *testing.T) {
	m := masker.New([]masker.Rule{email()})
	masked, changed := m.MaskLine("user@example.com logged in")
	if !changed {
		t.Error("expected changed=true")
	}
	if masked != "[REDACTED] logged in" {
		t.Errorf("got %q", masked)
	}
}

func TestMaskLine_MultiplePatterns(t *testing.T) {
	m := masker.New([]masker.Rule{email(), otp()})
	masked, changed := m.MaskLine("user@example.com OTP=123456")
	if !changed {
		t.Error("expected changed=true")
	}
	if masked != "[REDACTED] OTP=[REDACTED]" {
		t.Errorf("got %q", masked)
	}
}

func TestMaskLine_EmptyLine(t *testing.T) {
	m := masker.New([]masker.Rule{email()})
	masked, changed := m.MaskLine("")
	if changed {
		t.Error("expected no change on empty line")
	}
	if masked != "" {
		t.Errorf("got %q", masked)
	}
}

func TestMaskLine_NoRules(t *testing.T) {
	m := masker.New(nil)
	masked, changed := m.MaskLine("sensitive@data.com")
	if changed {
		t.Error("no rules should cause no change")
	}
	if masked != "sensitive@data.com" {
		t.Errorf("got %q", masked)
	}
}
