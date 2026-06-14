package masker

import "encoding/json"

// maskJSONFields tries to parse line as a JSON object, masks string values
// whose key matches a field rule, and re-serialises. Returns (maskedLine,
// matchedRuleNames, ok). ok is false if line is not JSON — callers fall
// through to regex-only masking.
func maskJSONFields(line string, fieldRules []Rule) (string, []string, bool) {
	if len(line) == 0 || line[0] != '{' {
		return line, nil, false
	}
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(line), &data); err != nil {
		return line, nil, false
	}

	fieldReplace := make(map[string]string, len(fieldRules))
	fieldRuleName := make(map[string]string, len(fieldRules))
	for _, r := range fieldRules {
		repl := r.Replace
		if repl == "" {
			repl = "[REDACTED]"
		}
		fieldReplace[r.Field] = repl
		fieldRuleName[r.Field] = r.Name
	}

	matchedSet := make(map[string]bool)
	maskJSONObject(data, fieldReplace, fieldRuleName, matchedSet)

	b, err := json.Marshal(data)
	if err != nil {
		return line, nil, false
	}

	if len(matchedSet) == 0 {
		return line, nil, false
	}

	var matched []string
	for name := range matchedSet {
		matched = append(matched, name)
	}
	return string(b), matched, true
}

func maskJSONObject(obj map[string]interface{}, fieldReplace, fieldRuleName map[string]string, matched map[string]bool) {
	for k, v := range obj {
		if repl, ok := fieldReplace[k]; ok {
			obj[k] = repl
			matched[fieldRuleName[k]] = true
			continue
		}
		switch child := v.(type) {
		case map[string]interface{}:
			maskJSONObject(child, fieldReplace, fieldRuleName, matched)
		case []interface{}:
			maskJSONArray(child, fieldReplace, fieldRuleName, matched)
		}
	}
}

func maskJSONArray(arr []interface{}, fieldReplace, fieldRuleName map[string]string, matched map[string]bool) {
	for _, item := range arr {
		switch child := item.(type) {
		case map[string]interface{}:
			maskJSONObject(child, fieldReplace, fieldRuleName, matched)
		case []interface{}:
			maskJSONArray(child, fieldReplace, fieldRuleName, matched)
		}
	}
}
