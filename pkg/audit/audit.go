package audit

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const privateFilePerm = 0o600
const privateDirPerm = 0o700

type Entry struct {
	Timestamp    time.Time `json:"ts"`
	Command      string    `json:"cmd"`
	PlanID       string    `json:"plan_id,omitempty"`
	Targets      []string  `json:"targets,omitempty"`
	FilesTouched []string  `json:"files,omitempty"`
	ExitCode     int       `json:"exit_code"`
	Error        string    `json:"error,omitempty"`
}

type Writer struct {
	HomeDir string
	Path    string
}

func DefaultPath(homeDir string) (string, error) {
	if homeDir == "" {
		return "", fmt.Errorf("missing home directory")
	}
	return filepath.Join(homeDir, ".usync", "audit.log"), nil
}

func NewWriter(homeDir string) (Writer, error) {
	path, err := DefaultPath(homeDir)
	if err != nil {
		return Writer{}, err
	}
	return Writer{
		HomeDir: homeDir,
		Path:    path,
	}, nil
}

func (w Writer) Append(entry Entry) error {
	if w.Path == "" {
		return fmt.Errorf("missing audit log path")
	}
	if err := os.MkdirAll(filepath.Dir(w.Path), privateDirPerm); err != nil {
		return fmt.Errorf("create audit directory for %s: %w", w.Path, err)
	}

	file, err := os.OpenFile(w.Path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, privateFilePerm)
	if err != nil {
		return fmt.Errorf("open audit log %s: %w", w.Path, err)
	}
	defer file.Close()

	if err := file.Chmod(privateFilePerm); err != nil {
		return fmt.Errorf("chmod audit log %s: %w", w.Path, err)
	}

	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("marshal audit entry: %w", err)
	}
	if _, err := file.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("write audit log %s: %w", w.Path, err)
	}
	return nil
}
