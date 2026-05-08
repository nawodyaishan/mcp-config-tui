package config

import (
	"bytes"
	"encoding/json"
	"fmt"
)

func UpdateMCPServersJSON(data []byte, exaURL string) ([]byte, error) {
	root, err := decodeJSONObject(data)
	if err != nil {
		return nil, err
	}

	servers := ensureObject(root, "mcpServers")
	servers["exa"] = map[string]any{
		"type": "sse",
		"url":  exaURL,
	}

	return marshalJSON(root)
}

func UpdateBareMCPServersJSON(data []byte, exaURL string) ([]byte, error) {
	root, err := decodeJSONObject(data)
	if err != nil {
		return nil, err
	}

	root["exa"] = map[string]any{
		"type": "sse",
		"url":  exaURL,
	}

	return marshalJSON(root)
}

func UpdateNamedServerJSON(data []byte, serverName, fieldName, exaURL string) ([]byte, error) {
	root, err := decodeJSONObject(data)
	if err != nil {
		return nil, err
	}

	root[serverName] = map[string]any{
		fieldName: exaURL,
	}

	return marshalJSON(root)
}

func decodeJSONObject(data []byte) (map[string]any, error) {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 {
		return map[string]any{}, nil
	}

	root := make(map[string]any)
	if err := json.Unmarshal(trimmed, &root); err != nil {
		return nil, fmt.Errorf("parse JSON config: %w", err)
	}
	return root, nil
}

func ensureObject(root map[string]any, key string) map[string]any {
	existing, ok := root[key]
	if !ok {
		child := make(map[string]any)
		root[key] = child
		return child
	}

	child, ok := existing.(map[string]any)
	if ok {
		return child
	}

	child = make(map[string]any)
	root[key] = child
	return child
}

func marshalJSON(root map[string]any) ([]byte, error) {
	data, err := json.MarshalIndent(root, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal JSON config: %w", err)
	}
	data = append(data, '\n')
	return data, nil
}
