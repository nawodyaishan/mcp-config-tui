package validate

import (
	"net/http"
	"time"

	"github.com/nawodyaishan/universal-mcp-sync/pkg/provider"
)

type Status string

const (
	StatusOK      Status = "ok"
	StatusWarning Status = "warning"
	StatusFailed  Status = "failed"
	StatusSkipped Status = "skipped"
)

type Mode string

const (
	ModeOffline Mode = "offline"
	ModeLive    Mode = "live"
)

type Result struct {
	ProviderID string `json:"provider_id"`
	Key        string `json:"key"`
	Label      string `json:"label"`
	Status     Status `json:"status"`
	Mode       Mode   `json:"mode"`
	Message    string `json:"message"`
	Cached     bool   `json:"cached,omitempty"`
	QuotaCost  bool   `json:"quota_cost,omitempty"`
	HelpURL    string `json:"help_url,omitempty"`
}

type Report struct {
	ProviderID string   `json:"provider_id"`
	Live       bool     `json:"live"`
	Results    []Result `json:"results"`
	Warnings   []string `json:"warnings,omitempty"`
}

type Request struct {
	Provider provider.MCPProvider
	Values   map[string]string
	Live     bool
	Now      time.Time
}

type HTTPDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

func (r Report) HasFailures() bool {
	return HasFailures(r.Results)
}

func HasFailures(results []Result) bool {
	for _, result := range results {
		if result.Status == StatusFailed {
			return true
		}
	}
	return false
}
