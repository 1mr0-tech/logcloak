package rules_test

import (
	"testing"

	"github.com/1mr0-tech/logcloak/pkg/rules"
)

func policy(name string, patterns []rules.PatternSpec) rules.MaskingPolicy {
	return rules.MaskingPolicy{
		Spec: rules.MaskingPolicySpec{
			Patterns:   patterns,
			RedactWith: "[REDACTED]",
		},
	}
}

func TestMerge_EmptyInputs(t *testing.T) {
	result, err := rules.Merge(nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 0 {
		t.Errorf("expected 0 rules, got %d", len(result))
	}
}

func TestMerge_BuiltinPattern(t *testing.T) {
	p := policy("test", []rules.PatternSpec{{Name: "email", Builtin: "email"}})
	result, err := rules.Merge([]rules.MaskingPolicy{p}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(result))
	}
	if result[0].Name != "email" {
		t.Errorf("expected name 'email', got %q", result[0].Name)
	}
}

func TestMerge_Deduplication(t *testing.T) {
	p1 := policy("p1", []rules.PatternSpec{{Name: "email", Builtin: "email"}})
	p2 := policy("p2", []rules.PatternSpec{{Name: "email", Builtin: "email"}})
	result, err := rules.Merge([]rules.MaskingPolicy{p1, p2}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 1 {
		t.Errorf("expected 1 deduplicated rule, got %d", len(result))
	}
}

func TestMerge_CRDRuleCannotBeRemovedByAnnotation(t *testing.T) {
	crdRule := rules.PatternSpec{Name: "email", Builtin: "email"}
	p := policy("baseline", []rules.PatternSpec{crdRule})
	annotationRules := []rules.PatternSpec{{Name: "otp", Builtin: "otp-6digit"}}
	result, err := rules.Merge([]rules.MaskingPolicy{p}, annotationRules)
	if err != nil {
		t.Fatal(err)
	}
	names := make(map[string]bool)
	for _, r := range result {
		names[r.Name] = true
	}
	if !names["email"] {
		t.Error("CRD rule 'email' should always be present even when annotation rules added")
	}
	if !names["otp"] {
		t.Error("annotation rule 'otp' should be present")
	}
}

func TestParseAnnotations_Builtins(t *testing.T) {
	annotations := map[string]string{
		"logcloak.io/patterns": "email,otp-6digit",
	}
	specs := rules.ParseAnnotations(annotations)
	if len(specs) != 2 {
		t.Fatalf("expected 2 specs, got %d", len(specs))
	}
}

func TestParseAnnotations_CustomRegex(t *testing.T) {
	annotations := map[string]string{
		"logcloak.io/regex-order-id": `ORD-[0-9]{8}`,
	}
	specs := rules.ParseAnnotations(annotations)
	if len(specs) != 1 {
		t.Fatalf("expected 1 spec, got %d", len(specs))
	}
	if specs[0].Regex != `ORD-[0-9]{8}` {
		t.Errorf("expected regex preserved, got %q", specs[0].Regex)
	}
}

func TestParseAnnotations_Empty(t *testing.T) {
	specs := rules.ParseAnnotations(nil)
	if len(specs) != 0 {
		t.Errorf("expected 0 specs from nil annotations")
	}
}
