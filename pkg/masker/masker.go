package masker

import "regexp"

type Rule struct {
	Name    string
	Pattern *regexp.Regexp // nil for field rules
	Field   string         // if non-empty, mask JSON fields with this name
	Replace string
}

type Masker struct {
	rules []Rule
}

func New(rules []Rule) *Masker {
	return &Masker{rules: rules}
}

// MaskLine applies all rules to line and returns the masked result plus the
// names of every rule that matched at least once.
// Field rules run first (JSON field masking), then regex rules on the result.
func (m *Masker) MaskLine(line string) (masked string, matched []string) {
	masked = line

	var fieldRules []Rule
	for _, r := range m.rules {
		if r.Field != "" {
			fieldRules = append(fieldRules, r)
		}
	}
	if len(fieldRules) > 0 {
		if result, fm, ok := maskJSONFields(masked, fieldRules); ok {
			masked = result
			matched = append(matched, fm...)
		}
	}

	for _, r := range m.rules {
		if r.Field != "" {
			continue
		}
		result := r.Pattern.ReplaceAllLiteralString(masked, r.Replace)
		if result != masked {
			masked = result
			matched = append(matched, r.Name)
		}
	}

	return masked, matched
}
