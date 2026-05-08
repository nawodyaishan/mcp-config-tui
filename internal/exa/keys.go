package exa

import (
	"fmt"
	"net/url"
	"os"
	"regexp"
	"strings"
)

var uuidPattern = regexp.MustCompile(`(?i)\b[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}\b`)
var exaURLPattern = regexp.MustCompile(`https://mcp\.exa\.ai/mcp\?[^\s"']+`)

func ParseKeys(input string) ([]string, error) {
	matches := uuidPattern.FindAllString(input, -1)
	seen := make(map[string]struct{}, len(matches))
	keys := make([]string, 0, len(matches))
	for _, match := range matches {
		key := strings.ToLower(match)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		keys = append(keys, key)
	}
	if len(keys) == 0 {
		return nil, fmt.Errorf("no UUID-style Exa API keys found")
	}
	return keys, nil
}

func ParseKeysCSV(input string) ([]string, error) {
	parts := strings.Split(input, ",")
	builder := strings.Builder{}
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" {
			continue
		}
		builder.WriteString(trimmed)
		builder.WriteByte('\n')
	}
	return ParseKeys(builder.String())
}

func ParseKeysFile(path string) ([]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read keys file: %w", err)
	}
	return ParseKeys(string(data))
}

func RedactKey(key string) string {
	if len(key) <= 8 {
		return key
	}
	return key[:4] + "..." + key[len(key)-4:]
}

func RedactText(text string) string {
	redactedURLs := exaURLPattern.ReplaceAllStringFunc(text, func(raw string) string {
		parsed, err := url.Parse(raw)
		if err != nil {
			return "[redacted exa mcp url]"
		}

		query := parsed.Query()
		if query.Get("exaApiKey") == "" {
			return "[redacted exa mcp url]"
		}

		return "[redacted exa mcp url]"
	})

	return uuidPattern.ReplaceAllStringFunc(redactedURLs, RedactKey)
}
