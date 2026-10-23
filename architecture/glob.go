package architecture

import (
	"errors"
	"fmt"
	"path"
	"strings"
)

// match reports whether the slash-separated name matches pattern. A "**"
// segment matches zero or more path segments. Every other segment follows
// path.Match syntax. pattern must have passed validatePatterns.
func match(pattern, name string) bool {
	return matchSegments(strings.Split(pattern, "/"), strings.Split(name, "/"))
}

func matchSegments(pattern, name []string) bool {
	for len(pattern) > 0 {
		if pattern[0] == "**" {
			for i := range len(name) + 1 {
				if matchSegments(pattern[1:], name[i:]) {
					return true
				}
			}
			return false
		}
		if len(name) == 0 {
			return false
		}
		// path.Match fails only on malformed patterns, which validatePatterns rejects.
		if ok, _ := path.Match(pattern[0], name[0]); !ok {
			return false
		}
		pattern, name = pattern[1:], name[1:]
	}
	return len(name) == 0
}

func validatePatterns(patterns []string) error {
	for _, pattern := range patterns {
		if pattern == "" {
			return errors.New("empty pattern")
		}
		for _, segment := range strings.Split(pattern, "/") {
			if _, err := path.Match(segment, ""); err != nil {
				return fmt.Errorf("pattern %q: %w", pattern, err)
			}
		}
	}
	return nil
}
