package masker_test

import (
	"regexp"
	"testing"

	"github.com/1mr0-tech/logcloak/pkg/masker"
)

func fieldRule(field, replace string) masker.Rule {
	return masker.Rule{Name: "field:" + field, Field: field, Replace: replace}
}

func TestMaskLine_JSONFieldBasic(t *testing.T) {
	m := masker.New([]masker.Rule{fieldRule("password", "[REDACTED]")})
	masked, matched := m.MaskLine(`{"user":"alice","password":"s3cr3t"}`)
	if len(matched) != 1 || matched[0] != "field:password" {
		t.Errorf("expected matched=[field:password], got %v", matched)
	}
	if masked == `{"user":"alice","password":"s3cr3t"}` {
		t.Errorf("password field was not masked: %s", masked)
	}
}

func TestMaskLine_JSONFieldNested(t *testing.T) {
	m := masker.New([]masker.Rule{fieldRule("token", "[REDACTED]")})
	_, matched := m.MaskLine(`{"auth":{"token":"abc123","type":"bearer"}}`)
	if len(matched) != 1 || matched[0] != "field:token" {
		t.Errorf("expected matched=[field:token], got %v", matched)
	}
}

func TestMaskLine_JSONFieldNotFound(t *testing.T) {
	m := masker.New([]masker.Rule{fieldRule("password", "[REDACTED]")})
	masked, matched := m.MaskLine(`{"user":"alice","email":"a@b.com"}`)
	if len(matched) != 0 {
		t.Errorf("expected no match, got %v", matched)
	}
	if masked != `{"user":"alice","email":"a@b.com"}` {
		t.Errorf("line should be unchanged, got %s", masked)
	}
}

func TestMaskLine_JSONFieldNonJSON(t *testing.T) {
	m := masker.New([]masker.Rule{fieldRule("password", "[REDACTED]")})
	line := "2026-01-01 INFO password=s3cr3t user=alice"
	masked, matched := m.MaskLine(line)
	if masked != line {
		t.Errorf("non-JSON line should be unchanged, got %q", masked)
	}
	if len(matched) != 0 {
		t.Errorf("expected no match on non-JSON line, got %v", matched)
	}
}

func TestMaskLine_JSONFieldAndRegex(t *testing.T) {
	emailRegex := regexp.MustCompile(`[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}`)
	m := masker.New([]masker.Rule{
		fieldRule("password", "[REDACTED]"),
		{Name: "email", Pattern: emailRegex, Replace: "[REDACTED:email]"},
	})
	masked, matched := m.MaskLine(`{"password":"s3cr3t","contact":"user@example.com"}`)
	if len(matched) < 2 {
		t.Errorf("expected both rules to match, got %v; masked=%s", matched, masked)
	}
}

func TestMaskLine_JSONFieldMultiple(t *testing.T) {
	m := masker.New([]masker.Rule{
		fieldRule("password", "[REDACTED]"),
		fieldRule("token", "[REDACTED]"),
	})
	_, matched := m.MaskLine(`{"password":"s3cr3t","token":"abc","user":"alice"}`)
	if len(matched) != 2 {
		t.Errorf("expected 2 matched rules, got %d: %v", len(matched), matched)
	}
}

func TestMaskLine_JSONFieldInArray(t *testing.T) {
	m := masker.New([]masker.Rule{fieldRule("secret", "[REDACTED]")})
	_, matched := m.MaskLine(`{"items":[{"secret":"x"},{"secret":"y"}]}`)
	if len(matched) == 0 {
		t.Error("expected match in array elements, got none")
	}
}

func TestMaskLine_JSONFieldEmptyObject(t *testing.T) {
	m := masker.New([]masker.Rule{fieldRule("password", "[REDACTED]")})
	masked, matched := m.MaskLine(`{}`)
	if len(matched) != 0 {
		t.Errorf("expected no match on empty object, got %v", matched)
	}
	if masked != `{}` {
		t.Errorf("empty object should roundtrip unchanged, got %s", masked)
	}
}
