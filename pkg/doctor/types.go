package doctor

import (
	"context"
	"time"

	"github.com/nawodyaishan/universal-mcp-sync/pkg/manifest"
)

type Confidence string

const (
	ConfidenceHigh     Confidence = "high"
	ConfidenceMedium   Confidence = "medium"
	ConfidenceLow      Confidence = "low"
	ConfidenceConflict Confidence = "conflict"
)

type Options struct {
	HomeDir        string
	WorkspaceDir   string
	GOOS           string
	CheckRuntimes  bool
	CommandTimeout time.Duration
	Now            func() time.Time
}

type Report struct {
	Platform string           `json:"platform"`
	Clients  []ClientFinding  `json:"clients"`
	Runtimes []RuntimeFinding `json:"runtimes,omitempty"`
	Warnings []string         `json:"warnings,omitempty"`
}

type ClientFinding struct {
	ID                  manifest.ClientID  `json:"id"`
	Name                string             `json:"name"`
	Installed           bool               `json:"installed"`
	CLIAvailable        bool               `json:"cli_available,omitempty"`
	EffectivePath       string             `json:"effective_path,omitempty"`
	Confidence          Confidence         `json:"confidence"`
	ConfiguredProviders []string           `json:"configured_providers,omitempty"`
	Candidates          []CandidateFinding `json:"candidates"`
	Issues              []string           `json:"issues,omitempty"`
	Warnings            []string           `json:"warnings,omitempty"`
	MigrationHints      []MigrationHint    `json:"migration_hints,omitempty"`
}

type CandidateFinding struct {
	Label      string             `json:"label"`
	Path       string             `json:"path,omitempty"`
	Scope      manifest.ScopeKind `json:"scope"`
	Deprecated bool               `json:"deprecated,omitempty"`
	Exists     bool               `json:"exists"`
	IsSymlink  bool               `json:"is_symlink,omitempty"`
	Resolved   string             `json:"resolved_path,omitempty"`
	ParseOK    bool               `json:"parse_ok"`
	ParseError string             `json:"parse_error,omitempty"`
	Writable   bool               `json:"writable"`
	RootKey    string             `json:"root_key,omitempty"`
	RootKeyOK  bool               `json:"root_key_ok"`
	RootType   string             `json:"root_key_type,omitempty"`
	Providers  []string           `json:"providers,omitempty"`
}

type RuntimeFinding struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Available   bool     `json:"available"`
	Path        string   `json:"path,omitempty"`
	Version     string   `json:"version,omitempty"`
	RequiredFor []string `json:"required_for,omitempty"`
	Error       string   `json:"error,omitempty"`
}

type MigrationHint struct {
	FromID   manifest.ClientID `json:"from_id"`
	ToID     manifest.ClientID `json:"to_id"`
	Reason   string            `json:"reason"`
	Deadline string            `json:"deadline,omitempty"`
}

func (r Report) HasFindings() bool {
	if len(r.Warnings) > 0 {
		return true
	}
	for _, client := range r.Clients {
		if len(client.Issues) > 0 || len(client.Warnings) > 0 || client.Confidence == ConfidenceConflict {
			return true
		}
	}
	for _, runtime := range r.Runtimes {
		if !runtime.Available {
			return true
		}
	}
	return false
}

type Doctor struct {
	options Options

	lookPath func(string) (string, error)
	runCmd   func(context.Context, string, ...string) (string, error)
}
