package config

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/nawodyaishan/universal-mcp-sync/pkg/provider"
)

func buildConfigMap(cfg provider.MCPConfig, urlFieldName string, extra map[string]any) map[string]any {
	result := make(map[string]any)
	if cfg.Type == provider.TransportStdio {
		result["command"] = cfg.Command
		if len(cfg.Args) > 0 {
			result["args"] = cfg.Args
		}
		if len(cfg.Env) > 0 {
			result["env"] = cfg.Env
		}
	} else {
		// HTTP or SSE
		if urlFieldName != "" {
			result[urlFieldName] = cfg.URL
		} else {
			result["url"] = cfg.URL
		}
	}

	for k, v := range extra {
		result[k] = v
	}
	return result
}

func UpdateMCPServersJSON(data []byte, providerID, rootKey, urlFieldName string, cfg provider.MCPConfig, extra map[string]any) ([]byte, error) {
	root, err := decodeJSONObject(data)
	if err != nil {
		return nil, err
	}

	if rootKey == "" {
		rootKey = "mcpServers"
	}

	servers := ensureObject(root, rootKey)
	servers[providerID] = buildConfigMap(cfg, urlFieldName, extra)

	return marshalJSON(root)
}

func UpdateBareMCPServersJSON(data []byte, providerID, urlFieldName string, cfg provider.MCPConfig, extra map[string]any) ([]byte, error) {
	root, err := decodeJSONObject(data)
	if err != nil {
		return nil, err
	}

	root[providerID] = buildConfigMap(cfg, urlFieldName, extra)

	return marshalJSON(root)
}

func UpdateNamedServerJSON(data []byte, providerID, rootKey, urlFieldName string, cfg provider.MCPConfig, extra map[string]any) ([]byte, error) {
	root, err := decodeJSONObject(data)
	if err != nil {
		return nil, err
	}

	var parent map[string]any
	if rootKey != "" {
		parent = ensureObject(root, rootKey)
	} else {
		parent = root
	}

	server := ensureObject(parent, providerID)
	// For named server updates (like Antigravity), we typically only update the URL field
	// and preserve the rest of the object.
	if cfg.Type != provider.TransportStdio {
		if urlFieldName == "" {
			urlFieldName = "url"
		}
		server[urlFieldName] = cfg.URL
	} else {
		// If it switches to stdio, we overwrite with the full stdio map
		server = buildConfigMap(cfg, urlFieldName, extra)
		parent[providerID] = server
	}

	for k, v := range extra {
		server[k] = v
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
