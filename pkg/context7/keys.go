package context7

import (
    "fmt"
    "strings"
)

const keyPrefix = "ctx7sk_"
const minKeyLen = len(keyPrefix) + 8

// ParseKey validates a Context7 API key.
// Valid keys start with "ctx7sk_" and have sufficient length.
func ParseKey(key string) (string, error) {
    key = strings.TrimSpace(key)
    if !strings.HasPrefix(key, keyPrefix) {
        return "", fmt.Errorf("Context7 API key must start with %q", keyPrefix)
    }
    if len(key) < minKeyLen {
        return "", fmt.Errorf("Context7 API key is too short")
    }
    return key, nil
}

// RedactKey masks a Context7 API key for display.
// e.g. "ctx7sk_abcdef1234567890wxyz" → "ctx7sk_abcd...wxyz"
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