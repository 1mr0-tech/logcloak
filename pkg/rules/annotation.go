package rules

import "strings"

// ParseAnnotations extracts PatternSpecs from pod annotations.
// logcloak.io/patterns: "email,otp-6digit"   → builtin patterns by name
// logcloak.io/regex-<name>: "<pattern>"       → custom regex patterns
func ParseAnnotations(annotations map[string]string) []PatternSpec {
	var specs []PatternSpec
	if annotations == nil {
		return specs
	}
	if builtins, ok := annotations["logcloak.io/patterns"]; ok {
		for _, name := range strings.Split(builtins, ",") {
			name = strings.TrimSpace(name)
			if name != "" {
				specs = append(specs, PatternSpec{Name: name, Builtin: name})
			}
		}
	}
	for k, v := range annotations {
		if strings.HasPrefix(k, "logcloak.io/regex-") {
			name := strings.TrimPrefix(k, "logcloak.io/regex-")
			if name != "" && v != "" {
				specs = append(specs, PatternSpec{Name: name, Regex: v})
			}
		}
	}
	return specs
}
