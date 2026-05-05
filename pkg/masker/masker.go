package masker

import "regexp"

type Rule struct {
	Name    string
	Pattern *regexp.Regexp
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
func (m *Masker) MaskLine(line string) (masked string, matched []string) {
	masked = line
	for _, r := range m.rules {
		result := r.Pattern.ReplaceAllLiteralString(masked, r.Replace)
		if result != masked {
			masked = result
			matched = append(matched, r.Name)
		}
	}
	return masked, matched
}
