package context7

import "testing"

func TestEndpoint(t *testing.T) {
	if Endpoint != "https://mcp.context7.com/mcp" {
		t.Errorf("unexpected endpoint %s", Endpoint)
	}
}
