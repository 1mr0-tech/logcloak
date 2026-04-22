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
