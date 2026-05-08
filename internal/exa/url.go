package exa

import (
	"fmt"
	"net/url"
	"strings"
)

func BuildURL(key string, tools []string) (string, error) {
	if key == "" {
		return "", fmt.Errorf("missing Exa API key")
	}
	if len(tools) == 0 {
		return "", fmt.Errorf("missing Exa tools")
	}

	base, err := url.Parse("https://mcp.exa.ai/mcp")
	if err != nil {
		return "", fmt.Errorf("parse Exa base URL: %w", err)
	}

	query := base.Query()
	query.Set("exaApiKey", key)
	query.Set("tools", strings.Join(tools, ","))
	base.RawQuery = query.Encode()
	return base.String(), nil
}
