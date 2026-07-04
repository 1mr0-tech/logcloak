package main

import (
	"bytes"
	"regexp"
	"strings"
	"testing"

	"github.com/1mr0-tech/logcloak/pkg/masker"
	"github.com/1mr0-tech/logcloak/pkg/metrics"
	"github.com/1mr0-tech/logcloak/pkg/sentinel"
)

func TestMain(m *testing.M) {
	metrics.MustRegister()
	m.Run()
}

// emailRule returns a simple email masking rule for test use.
func emailRule() masker.Rule {
	return masker.Rule{
		Name:    "email",
		Pattern: regexp.MustCompile(`[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}`),
		Replace: "[REDACTED]",
	}
}

func TestProcessPipe_CleanLinePassesThrough(t *testing.T) {
	m := masker.New([]masker.Rule{emailRule()})
	var out bytes.Buffer
	if err := processPipe(strings.NewReader("no pii here\n"), &out, m, "pod", "ns"); err != nil {
		t.Fatal(err)
	}
	if got := strings.TrimRight(out.String(), "\n"); got != "no pii here" {
		t.Errorf("expected clean line unchanged, got %q", got)
	}
}

func TestProcessPipe_MasksEmail(t *testing.T) {
	m := masker.New([]masker.Rule{emailRule()})
	var out bytes.Buffer
	if err := processPipe(strings.NewReader("login: alice@example.com\n"), &out, m, "pod", "ns"); err != nil {
		t.Fatal(err)
	}
	got := out.String()
	if strings.Contains(got, "alice@example.com") {
		t.Errorf("email must be redacted, got %q", got)
	}
	if !strings.Contains(got, "[REDACTED]") {
		t.Errorf("expected [REDACTED] in output, got %q", got)
	}
}

func TestProcessPipe_MultipleLines(t *testing.T) {
	m := masker.New([]masker.Rule{emailRule()})
	input := "clean line\nlogin: user@example.com\nanother clean line\n"
	var out bytes.Buffer
	if err := processPipe(strings.NewReader(input), &out, m, "pod", "ns"); err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimRight(out.String(), "\n"), "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 output lines, got %d: %q", len(lines), out.String())
	}
	if lines[0] != "clean line" {
		t.Errorf("line 1: expected passthrough, got %q", lines[0])
	}
	if strings.Contains(lines[1], "user@example.com") {
		t.Errorf("line 2: email must be redacted, got %q", lines[1])
	}
	if lines[2] != "another clean line" {
		t.Errorf("line 3: expected passthrough, got %q", lines[2])
	}
}

func TestProcessPipe_MultipleRulesOnSameLine(t *testing.T) {
	creditCard := masker.Rule{
		Name:    "cc",
		Pattern: regexp.MustCompile(`\b(?:\d[ -]?){13,15}\d\b`),
		Replace: "[CC-REDACTED]",
	}
	m := masker.New([]masker.Rule{emailRule(), creditCard})
	input := "user@example.com charged 4111 1111 1111 1111\n"
	var out bytes.Buffer
	if err := processPipe(strings.NewReader(input), &out, m, "pod", "ns"); err != nil {
		t.Fatal(err)
	}
	got := out.String()
	if strings.Contains(got, "user@example.com") {
		t.Errorf("email should be redacted, got %q", got)
	}
	if strings.Contains(got, "4111") {
		t.Errorf("credit card should be redacted, got %q", got)
	}
}

func TestProcessPipe_EmptyInputProducesNoOutput(t *testing.T) {
	m := masker.New([]masker.Rule{emailRule()})
	var out bytes.Buffer
	if err := processPipe(strings.NewReader(""), &out, m, "pod", "ns"); err != nil {
		t.Fatal(err)
	}
	if out.Len() != 0 {
		t.Errorf("empty input should produce empty output, got %q", out.String())
	}
}

func TestProcessPipe_JSONFieldMasking(t *testing.T) {
	m := masker.New([]masker.Rule{
		{Name: "token", Field: "token", Replace: "[REDACTED]"},
	})
	input := `{"user":"alice","token":"secret-value"}` + "\n"
	var out bytes.Buffer
	if err := processPipe(strings.NewReader(input), &out, m, "pod", "ns"); err != nil {
		t.Fatal(err)
	}
	got := out.String()
	if strings.Contains(got, "secret-value") {
		t.Errorf("JSON field token must be masked, got %q", got)
	}
	if !strings.Contains(got, "alice") {
		t.Errorf("unmasked field user must be preserved, got %q", got)
	}
}

func TestDropPipe_EmitsSentinelForEveryLine(t *testing.T) {
	input := "line one\nline two\nline three\n"
	var out bytes.Buffer
	dropPipe(strings.NewReader(input), &out, "mypod", "ns", "rules_parse_error")
	lines := strings.Split(strings.TrimRight(out.String(), "\n"), "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 sentinel lines, got %d: %q", len(lines), out.String())
	}
	for i, line := range lines {
		if !strings.HasPrefix(line, sentinel.Prefix) {
			t.Errorf("line %d: expected %s prefix, got %q", i+1, sentinel.Prefix, line)
		}
	}
}

func TestDropPipe_EmptyInputProducesNoOutput(t *testing.T) {
	var out bytes.Buffer
	dropPipe(strings.NewReader(""), &out, "pod", "ns", "reason")
	if out.Len() != 0 {
		t.Errorf("empty input should produce no sentinel lines, got %q", out.String())
	}
}
