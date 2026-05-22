package validate

import (
	"bufio"
	"bytes"
	"fmt"
	"strings"
)

type KeyValue struct {
	Key   string
	Value string
}

type ParsedKeyFile struct {
	Entries []KeyValue
}

func ParseKeyFile(data []byte) (ParsedKeyFile, error) {
	scanner := bufio.NewScanner(bytes.NewReader(data))
	parsed := ParsedKeyFile{}
	lineNumber := 0

	for scanner.Scan() {
		lineNumber++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		index := strings.IndexByte(line, '=')
		if index <= 0 {
			return ParsedKeyFile{}, fmt.Errorf("invalid credential line %d: expected KEY=value", lineNumber)
		}

		key := strings.TrimSpace(line[:index])
		if key == "" {
			return ParsedKeyFile{}, fmt.Errorf("invalid credential line %d: missing key name", lineNumber)
		}

		value := trimMatchingQuotes(strings.TrimSpace(line[index+1:]))
		parsed.Entries = append(parsed.Entries, KeyValue{
			Key:   key,
			Value: value,
		})
	}

	if err := scanner.Err(); err != nil {
		return ParsedKeyFile{}, fmt.Errorf("read credential file: %w", err)
	}

	return parsed, nil
}

func (p ParsedKeyFile) Values() map[string]string {
	values := make(map[string]string, len(p.Entries))
	for _, entry := range p.Entries {
		values[entry.Key] = entry.Value
	}
	return values
}

func (p ParsedKeyFile) ValuesForKey(key string) []string {
	values := make([]string, 0)
	for _, entry := range p.Entries {
		if entry.Key == key {
			values = append(values, entry.Value)
		}
	}
	return values
}

func trimMatchingQuotes(value string) string {
	if len(value) < 2 {
		return value
	}
	if (value[0] == '"' && value[len(value)-1] == '"') || (value[0] == '\'' && value[len(value)-1] == '\'') {
		return value[1 : len(value)-1]
	}
	return value
}
