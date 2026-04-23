package webhook

import "strings"

// Op is a single RFC 6902 JSON Patch operation.
type Op struct {
	Op    string      `json:"op"`
	Path  string      `json:"path"`
	Value interface{} `json:"value,omitempty"`
}

// escapePath escapes a JSON Pointer token per RFC 6901.
func escapePath(s string) string {
	s = strings.ReplaceAll(s, "~", "~0")
	s = strings.ReplaceAll(s, "/", "~1")
	return s
}

func addOp(path string, value interface{}) Op {
	return Op{Op: "add", Path: path, Value: value}
}

func replaceOp(path string, value interface{}) Op {
	return Op{Op: "replace", Path: path, Value: value}
}
