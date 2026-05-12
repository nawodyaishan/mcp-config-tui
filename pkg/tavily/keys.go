package tavily

import (
	"fmt"
	"strings"
)

const keyPrefix = "tvly-"
const minKeyLen = len(keyPrefix) + 8

// ParseKey validates a Tavily API key.
// Valid keys start with "tvly-" and have sufficient length.
func ParseKey(key string) (string, error) {
	key = strings.TrimSpace(key)
	if !strings.HasPrefix(key, keyPrefix) {
		return "", fmt.Errorf("tavily API key must start with %q", keyPrefix)
	}
	if len(key) < minKeyLen {
		return "", fmt.Errorf("tavily API key is too short")
	}
	return key, nil
}

// RedactKey masks a Tavily API key for display.
func RedactKey(key string) string {
	if len(key) <= len(keyPrefix)+8 {
		return key
	}
	suffix := key[len(keyPrefix):]
	if len(suffix) <= 8 {
		return key
	}
	return keyPrefix + suffix[:4] + "..." + suffix[len(suffix)-4:]
}
