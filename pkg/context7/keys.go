package context7

import (
	"fmt"
	"strings"
)

const keyPrefix = "ctx7sk"
const minKeyLen = len(keyPrefix) + 1 + 8

// ParseKey validates a Context7 API key.
// Valid keys start with "ctx7sk-" or "ctx7sk_" and have sufficient length.
func ParseKey(key string) (string, error) {
	key = strings.TrimSpace(key)
	if !strings.HasPrefix(key, keyPrefix) || len(key) <= len(keyPrefix) || !isKeySeparator(key[len(keyPrefix)]) {
		return "", fmt.Errorf("Context7 API key must start with %q or %q", keyPrefix+"-", keyPrefix+"_")
	}
	if len(key) < minKeyLen {
		return "", fmt.Errorf("Context7 API key is too short")
	}
	return key, nil
}

// RedactKey masks a Context7 API key for display.
// e.g. "ctx7sk-abcdef1234567890wxyz" becomes "ctx7sk-abcd...wxyz".
func RedactKey(key string) string {
	prefixLen := len(keyPrefix) + 1
	if len(key) <= prefixLen+8 {
		return key
	}
	if !strings.HasPrefix(key, keyPrefix) || !isKeySeparator(key[len(keyPrefix)]) {
		return key
	}
	prefix := key[:prefixLen]
	suffix := key[prefixLen:]
	if len(suffix) <= 8 {
		return key
	}
	return prefix + suffix[:4] + "..." + suffix[len(suffix)-4:]
}

func isKeySeparator(ch byte) bool {
	return ch == '-' || ch == '_'
}
