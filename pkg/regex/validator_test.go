package regex_test

import (
	"testing"

	"github.com/1mr0-tech/logcloak/pkg/regex"
)

func TestValidate_ValidPattern(t *testing.T) {
	if err := regex.Validate(`[a-z]+@[a-z]+\.[a-z]{2,}`); err != nil {
		t.Errorf("valid pattern rejected: %v", err)
	}
}

func TestValidate_InvalidSyntax(t *testing.T) {
	if err := regex.Validate(`[unclosed`); err == nil {
		t.Error("invalid syntax should be rejected")
	}
}

func TestValidate_RejectsLookahead(t *testing.T) {
	if err := regex.Validate(`foo(?=bar)`); err == nil {
		t.Error("lookahead should be rejected")
	}
}

func TestValidate_RejectsLookbehind(t *testing.T) {
	if err := regex.Validate(`(?<=foo)bar`); err == nil {
		t.Error("lookbehind should be rejected")
	}
}

func TestValidate_RejectsNegativeLookahead(t *testing.T) {
	if err := regex.Validate(`foo(?!bar)`); err == nil {
		t.Error("negative lookahead should be rejected")
	}
}

func TestValidate_RejectsBackreference(t *testing.T) {
	if err := regex.Validate(`(foo)\1`); err == nil {
		t.Error("backreference should be rejected")
	}
}

func TestValidate_RejectsNamedCapturePCRE(t *testing.T) {
	if err := regex.Validate(`(?P<name>[a-z]+)`); err == nil {
		t.Error("PCRE-style named capture should be rejected")
	}
}

func TestValidate_EmptyPatternAllowed(t *testing.T) {
	if err := regex.Validate(``); err != nil {
		t.Errorf("empty pattern is valid RE2, should be allowed: %v", err)
	}
}

func TestValidate_ComplexSafePattern(t *testing.T) {
	// Groups, alternation, anchors — all RE2-safe
	safe := `^(foo|bar)\d{3}[a-zA-Z]+$`
	if err := regex.Validate(safe); err != nil {
		t.Errorf("safe complex pattern rejected: %v", err)
	}
}

func TestValidate_RejectsBackreferenceTwo(t *testing.T) {
	if err := regex.Validate(`(a)(b)\2`); err == nil {
		t.Error(`backreference \2 should be rejected`)
	}
}
