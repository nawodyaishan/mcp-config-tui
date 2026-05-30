package uxexplore

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
)

type Exclusion struct {
	Screen            string `json:"screen"`
	PreconditionClass string `json:"precondition_class"`
	Reason            string `json:"reason"`
}

func LoadExclusions(path string) ([]Exclusion, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	return ParseExclusions(f)
}

func ParseExclusions(r io.Reader) ([]Exclusion, error) {
	var exclusions []Exclusion
	var current *Exclusion
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") || line == "exclusions:" {
			continue
		}
		if strings.HasPrefix(line, "- ") {
			if current != nil {
				exclusions = append(exclusions, *current)
			}
			current = &Exclusion{}
			line = strings.TrimSpace(strings.TrimPrefix(line, "- "))
			if line == "" {
				continue
			}
		}
		if current == nil {
			return nil, fmt.Errorf("exclusion field before list item: %q", line)
		}
		key, value, ok := strings.Cut(line, ":")
		if !ok {
			return nil, fmt.Errorf("invalid exclusion line %q", line)
		}
		value = strings.Trim(strings.TrimSpace(value), `"`)
		switch strings.TrimSpace(key) {
		case "screen":
			current.Screen = value
		case "precondition_class":
			current.PreconditionClass = value
		case "reason":
			current.Reason = value
		default:
			return nil, fmt.Errorf("unknown exclusion field %q", key)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if current != nil {
		exclusions = append(exclusions, *current)
	}
	for _, exclusion := range exclusions {
		if exclusion.Screen == "" || exclusion.PreconditionClass == "" || exclusion.Reason == "" {
			return nil, fmt.Errorf("incomplete exclusion: %#v", exclusion)
		}
	}
	return exclusions, nil
}
