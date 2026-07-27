package main

import (
	"os"
	"strings"
	"testing"

	"github.com/1mr0-tech/logcloak/pkg/rules"
)

func policyWithBuiltins(names ...string) rules.MaskingPolicy {
	var p rules.MaskingPolicy
	p.Spec.RedactWith = "[REDACTED]"
	for _, n := range names {
		p.Spec.Patterns = append(p.Spec.Patterns, rules.PatternSpec{Name: n, Builtin: n})
	}
	return p
}

func TestRunAudit_FindsGaps(t *testing.T) {
	policy := policyWithBuiltins("email", "otp-6digit")
	in := strings.NewReader("user jane@example.com card=4111111111111111 otp=847291\n")

	result, err := runAudit(policy, in)
	if err != nil {
		t.Fatalf("runAudit: %v", err)
	}
	if result.TotalLines != 1 {
		t.Fatalf("TotalLines = %d, want 1", result.TotalLines)
	}
	if result.LinesCovered != 1 {
		t.Fatalf("LinesCovered = %d, want 1 (email+otp should still mask something)", result.LinesCovered)
	}
	if len(result.GapHits) != 1 {
		t.Fatalf("GapHits = %d, want 1", len(result.GapHits))
	}
	if result.GapCounts["credit-card"] != 1 {
		t.Fatalf("expected credit-card gap, got %+v", result.GapCounts)
	}
	if _, ok := result.GapCounts["email"]; ok {
		t.Fatalf("email should not be reported as a gap since the policy covers it: %+v", result.GapCounts)
	}
}

func TestRunAudit_NoGapsWhenFullyCovered(t *testing.T) {
	policy := policyWithBuiltins("email", "otp-6digit", "credit-card", "ssn", "jwt")
	in := strings.NewReader("user jane@example.com card=4111111111111111 otp=847291\nssn 123-45-6789\n")

	result, err := runAudit(policy, in)
	if err != nil {
		t.Fatalf("runAudit: %v", err)
	}
	if len(result.GapHits) != 0 {
		t.Fatalf("expected no gaps, got %+v", result.GapHits)
	}
	if len(result.GapCounts) != 0 {
		t.Fatalf("expected empty gap counts, got %+v", result.GapCounts)
	}
}

func TestRunAudit_FieldPatternsIgnoredInGapCheck(t *testing.T) {
	var policy rules.MaskingPolicy
	policy.Spec.Patterns = []rules.PatternSpec{{Name: "password-field", Field: "password"}}
	in := strings.NewReader(`{"password":"s3cr3t","email":"jane@example.com"}` + "\n")

	result, err := runAudit(policy, in)
	if err != nil {
		t.Fatalf("runAudit: %v", err)
	}
	if result.GapCounts["email"] != 1 {
		t.Fatalf("expected email gap since field-only policy doesn't cover it, got %+v", result.GapCounts)
	}
}

func TestRunAudit_CleanLineNoGaps(t *testing.T) {
	policy := policyWithBuiltins("email")
	in := strings.NewReader("nothing sensitive in this line\n")

	result, err := runAudit(policy, in)
	if err != nil {
		t.Fatalf("runAudit: %v", err)
	}
	if result.TotalLines != 1 || len(result.GapHits) != 0 || result.LinesCovered != 0 {
		t.Fatalf("unexpected result for clean line: %+v", result)
	}
}

func TestParseMaskingPolicy_RejectsEmptyPatterns(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/empty.yaml"
	content := "apiVersion: logcloak.io/v1alpha1\nkind: MaskingPolicy\nmetadata:\n  name: x\nspec:\n  patterns: []\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := parseMaskingPolicy(path); err == nil {
		t.Fatal("expected error for policy with no patterns")
	}
}
