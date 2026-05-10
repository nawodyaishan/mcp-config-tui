package redact

import "regexp"

var uuidRE = regexp.MustCompile(
    `(?i)\b[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}\b`,
)

// Text replaces every UUID-shaped substring in s with a truncated token.
func Text(s string) string {
    return uuidRE.ReplaceAllStringFunc(s, Key)
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
