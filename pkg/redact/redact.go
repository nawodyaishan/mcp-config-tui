package redact

import (
	"regexp"

	"github.com/nawodyaishan/universal-mcp-sync/pkg/context7"
)

var uuidRE = regexp.MustCompile(
    `(?i)\b[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}\b`,
)

var ctx7RE = regexp.MustCompile(`ctx7sk_[A-Za-z0-9_\-]{8,}`)

// Text replaces every UUID-shaped substring in s with a truncated token.
func Text(s string) string {
    s = uuidRE.ReplaceAllStringFunc(s, Key)
    s = ctx7RE.ReplaceAllStringFunc(s, func(key string) string {
        return context7.RedactKey(key)
    })
    return s
}

// Key returns the first 4 and last 4 characters of key separated by "...".
// Keys shorter than 9 characters are returned unchanged.
func Key(key string) string {
    if len(key) <= 8 {
        return key
    }
    return key[:4] + "..." + key[len(key)-4:]
}

// Attrs redacts string values in a slog-style key-value variadic slice.
// Non-string values are passed through unchanged.
func Attrs(attrs []any) []any {
    out := make([]any, 0, len(attrs))
    for _, a := range attrs {
        if s, ok := a.(string); ok {
            out = append(out, Text(s))
            continue
        }
        out = append(out, a)
    }
    return out
}
