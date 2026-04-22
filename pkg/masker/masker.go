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

func (m *Masker) MaskLine(line string) (masked string, changed bool) {
	masked = line
	for _, r := range m.rules {
		result := r.Pattern.ReplaceAllString(masked, r.Replace)
		if result != masked {
			masked = result
			changed = true
		}
	}
	return masked, changed
}
