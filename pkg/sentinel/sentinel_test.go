package sentinel_test

import (
	"strings"
	"testing"

	"github.com/1mr0-tech/logcloak/pkg/sentinel"
)

func TestLine_ContainsPrefix(t *testing.T) {
	s := sentinel.Line("regex_timeout", "my-pod")
	if !strings.HasPrefix(s, "[LOGCLOAK-DROP]") {
		t.Errorf("sentinel must start with [LOGCLOAK-DROP], got: %q", s)
	}
}

func TestLine_ContainsReason(t *testing.T) {
	s := sentinel.Line("regex_timeout", "my-pod")
	if !strings.Contains(s, "reason=regex_timeout") {
		t.Errorf("sentinel must contain reason=, got: %q", s)
	}
}

func TestLine_ContainsPod(t *testing.T) {
	s := sentinel.Line("panic", "checkout-7d9f")
	if !strings.Contains(s, "pod=checkout-7d9f") {
		t.Errorf("sentinel must contain pod=, got: %q", s)
	}
}

func TestLine_ContainsSuppressedText(t *testing.T) {
	s := sentinel.Line("buffer_full", "test-pod")
	if !strings.Contains(s, "line suppressed") {
		t.Errorf("sentinel must contain 'line suppressed', got: %q", s)
	}
}
