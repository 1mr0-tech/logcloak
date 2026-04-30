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

func TestMerge_UnknownBuiltin(t *testing.T) {
	p := policy("bad", []rules.PatternSpec{{Name: "x", Builtin: "nonexistent-pattern"}})
	_, err := rules.Merge([]rules.MaskingPolicy{p}, nil)
	if err == nil {
		t.Error("unknown builtin should return error")
	}
}

func TestMerge_CustomRegex(t *testing.T) {
	p := policy("custom", []rules.PatternSpec{{Name: "order-id", Regex: `ORD-[0-9]{8}`}})
	result, err := rules.Merge([]rules.MaskingPolicy{p}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(result))
	}
	if result[0].Name != "order-id" {
		t.Errorf("expected name 'order-id', got %q", result[0].Name)
	}
}

func TestMerge_InvalidCustomRegex(t *testing.T) {
	p := policy("bad", []rules.PatternSpec{{Name: "bad", Regex: `[unclosed`}})
	_, err := rules.Merge([]rules.MaskingPolicy{p}, nil)
	if err == nil {
		t.Error("invalid regex should return error from Merge")
	}
}

func TestMerge_PatternWithNeitherBuiltinNorRegex(t *testing.T) {
	p := policy("empty", []rules.PatternSpec{{Name: "empty"}})
	_, err := rules.Merge([]rules.MaskingPolicy{p}, nil)
	if err == nil {
		t.Error("pattern with no builtin and no regex should return error")
	}
}

func TestSerializeDeserialize_RoundTrip(t *testing.T) {
	p := policy("test", []rules.PatternSpec{
		{Name: "email", Builtin: "email"},
		{Name: "order-id", Regex: `ORD-[0-9]{8}`},
	})
	compiled, err := rules.Merge([]rules.MaskingPolicy{p}, nil)
	if err != nil {
		t.Fatal(err)
	}

	jsonStr, err := rules.Serialize(compiled)
	if err != nil {
		t.Fatalf("serialize: %v", err)
	}
	if jsonStr == "" {
		t.Fatal("serialized string must not be empty")
	}

	restored, err := rules.Deserialize(jsonStr)
	if err != nil {
		t.Fatalf("deserialize: %v", err)
	}
	if len(restored) != len(compiled) {
		t.Fatalf("expected %d rules after round-trip, got %d", len(compiled), len(restored))
	}
	for i, r := range restored {
		if r.Name != compiled[i].Name {
			t.Errorf("rule %d: name mismatch: want %q got %q", i, compiled[i].Name, r.Name)
		}
		if r.Pattern.String() != compiled[i].Pattern.String() {
			t.Errorf("rule %d: pattern mismatch: want %q got %q", i, compiled[i].Pattern.String(), r.Pattern.String())
		}
		if r.Replace != compiled[i].Replace {
			t.Errorf("rule %d: replace mismatch: want %q got %q", i, compiled[i].Replace, r.Replace)
		}
	}
}

func TestDeserialize_EmptyString(t *testing.T) {
	result, err := rules.Deserialize("")
	if err != nil {
		t.Fatalf("empty string should return nil, nil; got err: %v", err)
	}
	if result != nil {
		t.Errorf("expected nil result for empty string, got %v", result)
	}
}

func TestDeserialize_InvalidJSON(t *testing.T) {
	_, err := rules.Deserialize("{not json}")
	if err == nil {
		t.Error("invalid JSON should return error")
	}
}

func TestParseAnnotations_IgnoresEmptyValues(t *testing.T) {
	annotations := map[string]string{
		"logcloak.io/regex-": "should-be-ignored",  // empty name
		"logcloak.io/regex-valid": `[0-9]+`,
	}
	specs := rules.ParseAnnotations(annotations)
	for _, s := range specs {
		if s.Name == "" {
			t.Error("empty-named annotation spec should be ignored")
		}
	}
}
