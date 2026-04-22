package regex

import (
	"fmt"
	"regexp"
	"strings"
)

var unsafeConstructs = []string{
	"(?=", "(?!", "(?<=", "(?<!", // lookaheads and lookbehinds
	"(?P<",                        // named capture (PCRE-style risk indicator)
	"\\1", "\\2", "\\3",           // backreferences
}

func Validate(pattern string) error {
	for _, construct := range unsafeConstructs {
		if strings.Contains(pattern, construct) {
			return fmt.Errorf("unsafe regex construct %q detected — only RE2-compatible patterns allowed", construct)
		}
	}
	if _, err := regexp.Compile(pattern); err != nil {
		return fmt.Errorf("invalid regex: %w", err)
	}
	return nil
}
