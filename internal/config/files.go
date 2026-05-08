package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const privatePerm = 0o600
const privateDirPerm = 0o700

type WriteOutcome struct {
	Path       string
	BackupPath string
	Existed    bool
}

func ReadFileOrEmpty(path string) ([]byte, bool, error) {
	data, err := os.ReadFile(path)
	if err == nil {
		return data, true, nil
	}
	if os.IsNotExist(err) {
		return []byte{}, false, nil
	}
	return nil, false, fmt.Errorf("read %s: %w", path, err)
}

func EnsureParentDir(path string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, privateDirPerm); err != nil {
		return fmt.Errorf("create parent directory for %s: %w", path, err)
	}
	return nil
}

func BuildBackupPath(path string, now time.Time) string {
	return fmt.Sprintf("%s.bak-exa-%s", path, now.Format("20060102-150405"))
}

func WriteWithBackup(path string, data []byte, now time.Time) (WriteOutcome, error) {
	if err := EnsureParentDir(path); err != nil {
		return WriteOutcome{}, err
	}

	existing, existed, err := ReadFileOrEmpty(path)
	if err != nil {
		return WriteOutcome{}, err
	}

	outcome := WriteOutcome{
		Path:    path,
		Existed: existed,
	}

	if existed {
		outcome.BackupPath = BuildBackupPath(path, now)
		if err := writeAtomic(outcome.BackupPath, existing); err != nil {
			return WriteOutcome{}, fmt.Errorf("write backup %s: %w", outcome.BackupPath, err)
		}
	}

	if err := writeAtomic(path, data); err != nil {
		return WriteOutcome{}, fmt.Errorf("write %s: %w", path, err)
	}

	return outcome, nil
}

func RollbackWrite(outcome WriteOutcome) error {
	if !outcome.Existed {
		if err := os.Remove(outcome.Path); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("remove created file %s: %w", outcome.Path, err)
		}
		return nil
	}

	backupData, err := os.ReadFile(outcome.BackupPath)
	if err != nil {
		return fmt.Errorf("read backup %s: %w", outcome.BackupPath, err)
	}

	if err := writeAtomic(outcome.Path, backupData); err != nil {
		return fmt.Errorf("restore %s: %w", outcome.Path, err)
	}

	return nil
}

func FilePerm(path string) (os.FileMode, error) {
	info, err := os.Stat(path)
	if err != nil {
		return 0, err
	}
	return info.Mode().Perm(), nil
}

func writeAtomic(path string, data []byte) error {
	if err := EnsureParentDir(path); err != nil {
		return err
	}

	dir := filepath.Dir(path)
	temp, err := os.CreateTemp(dir, ".exa-mcp-*")
	if err != nil {
		return fmt.Errorf("create temp file for %s: %w", path, err)
	}

	tempPath := temp.Name()
	cleanup := true
	defer func() {
		if cleanup {
			_ = os.Remove(tempPath)
		}
	}()

	if err := temp.Chmod(privatePerm); err != nil {
		_ = temp.Close()
		return fmt.Errorf("chmod temp file for %s: %w", path, err)
	}
	if _, err := temp.Write(data); err != nil {
		_ = temp.Close()
		return fmt.Errorf("write temp file for %s: %w", path, err)
	}
	if err := temp.Close(); err != nil {
		return fmt.Errorf("close temp file for %s: %w", path, err)
	}
	if err := os.Rename(tempPath, path); err != nil {
		return fmt.Errorf("rename temp file for %s: %w", path, err)
	}

	cleanup = false
	return nil
}
