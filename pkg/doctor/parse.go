package doctor

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/nawodyaishan/universal-mcp-sync/pkg/manifest"
)

func parseCandidateConfig(data []byte, format manifest.ConfigFormat, rootKey string) ([]string, bool, string, error) {
	switch format {
	case manifest.FormatJSON:
		return parseJSONObject(data, rootKey)
	case manifest.FormatJSONC:
		return parseJSONObject(stripJSONCComments(data), rootKey)
	case manifest.FormatTOML:
		return parseCodexTOML(data)
	default:
		return nil, false, "", fmt.Errorf("unsupported config format %q", format)
	}
}

func parseJSONObject(data []byte, rootKey string) ([]string, bool, string, error) {
	var root map[string]any
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 {
		return nil, false, "missing", nil
	}
	if err := json.Unmarshal(trimmed, &root); err != nil {
		return nil, false, "", fmt.Errorf("parse failed")
	}

	var container any = root
	rootOK := true
	rootType := ""
	if rootKey != "" {
		value, ok := root[rootKey]
		if !ok {
			return nil, false, "missing", nil
		}
		container = value
	}

	object, ok := container.(map[string]any)
	if !ok {
		rootOK = false
		rootType = jsonType(container)
		return nil, rootOK, rootType, nil
	}
	if rootKey == "" {
		rootOK = true
	}
	return filterKnownProviders(object), rootOK, rootType, nil
}

func parseCodexTOML(data []byte) ([]string, bool, string, error) {
	scanner := bufio.NewScanner(bytes.NewReader(data))
	sectionPattern := regexp.MustCompile(`^\[(.+)\]$`)
	providers := make(map[string]bool)
	rootOK := false

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.Contains(line, "[") && !strings.Contains(line, "]") {
			return nil, false, "", fmt.Errorf("parse failed")
		}
		match := sectionPattern.FindStringSubmatch(line)
		if len(match) != 2 {
			continue
		}
		section := strings.TrimSpace(match[1])
		if !strings.HasPrefix(section, "mcp_servers.") {
			continue
		}
		rootOK = true
		name := strings.TrimPrefix(section, "mcp_servers.")
		name = strings.Trim(name, `"`)
		if isKnownProvider(name) {
			providers[name] = true
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, false, "", fmt.Errorf("parse failed")
	}
	return sortedProviderIDs(providers), rootOK, "", nil
}

func stripJSONCComments(data []byte) []byte {
	var out bytes.Buffer
	inString := false
	escaped := false

	for i := 0; i < len(data); i++ {
		ch := data[i]
		if inString {
			out.WriteByte(ch)
			if escaped {
				escaped = false
				continue
			}
			if ch == '\\' {
				escaped = true
				continue
			}
			if ch == '"' {
				inString = false
			}
			continue
		}

		if ch == '"' {
			inString = true
			out.WriteByte(ch)
			continue
		}
		if ch == '/' && i+1 < len(data) && data[i+1] == '/' {
			for i < len(data) && data[i] != '\n' {
				i++
			}
			if i < len(data) {
				out.WriteByte('\n')
			}
			continue
		}
		out.WriteByte(ch)
	}

	return out.Bytes()
}

func filterKnownProviders(object map[string]any) []string {
	providers := make([]string, 0, len(object))
	for key := range object {
		if isKnownProvider(key) {
			providers = append(providers, key)
		}
	}
	slices.Sort(providers)
	return providers
}

func isKnownProvider(key string) bool {
	_, ok := manifest.ProviderByID(manifest.ProviderID(key))
	return ok
}

func jsonType(value any) string {
	switch value.(type) {
	case nil:
		return "null"
	case string:
		return "string"
	case bool:
		return "bool"
	case float64:
		return "number"
	case []any:
		return "array"
	case map[string]any:
		return "object"
	default:
		return "unknown"
	}
}
