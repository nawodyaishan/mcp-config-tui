package app

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type PlanStore struct {
	HomeDir string
	PlanDir string
	Now     func() time.Time
}

type PlanFile struct {
	Path       string
	PlanID     string
	ProviderID string
	CreatedAt  time.Time
	ExpiresAt  time.Time
	Expired    bool
}

type CleanOptions struct {
	ExpiredOnly bool
	RemoveAll   bool
}

func DefaultPlanDir(home string) (string, error) {
	if home == "" {
		return "", fmt.Errorf("missing home directory")
	}
	if custom := strings.TrimSpace(os.Getenv("USYNC_PLAN_DIR")); custom != "" {
		return filepath.Clean(custom), nil
	}
	if xdg := strings.TrimSpace(os.Getenv("XDG_CACHE_HOME")); xdg != "" {
		return filepath.Join(xdg, "usync", "plans"), nil
	}
	return filepath.Join(home, ".cache", "usync", "plans"), nil
}

func NewPlanStore(homeDir string) (PlanStore, error) {
	if homeDir == "" {
		var err error
		homeDir, err = os.UserHomeDir()
		if err != nil {
			return PlanStore{}, fmt.Errorf("resolve home directory: %w", err)
		}
	}
	planDir, err := DefaultPlanDir(homeDir)
	if err != nil {
		return PlanStore{}, err
	}
	return PlanStore{
		HomeDir: homeDir,
		PlanDir: planDir,
		Now:     time.Now,
	}, nil
}

func (s PlanStore) Save(plan SavedPlan, outPath string) (string, error) {
	if s.Now == nil {
		s.Now = time.Now
	}

	if plan.SchemaVersion == 0 {
		plan.SchemaVersion = SavedPlanSchemaVersion
	}
	if plan.CreatedAt.IsZero() {
		plan.CreatedAt = s.Now().UTC()
	}
	if plan.ExpiresAt.IsZero() {
		plan.ExpiresAt = plan.CreatedAt.Add(defaultPlanTTL)
	}

	targetPath, err := s.resolveOutputPath(plan, outPath)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(targetPath), 0o700); err != nil {
		return "", fmt.Errorf("create plan directory for %s: %w", targetPath, err)
	}

	data, err := MarshalSavedPlanJSON(plan)
	if err != nil {
		return "", err
	}
	if err := writePlanAtomic(targetPath, data); err != nil {
		return "", err
	}
	return targetPath, nil
}

func (s PlanStore) Load(path string) (SavedPlan, error) {
	resolved, err := s.validatePlanPath(path)
	if err != nil {
		return SavedPlan{}, err
	}
	info, err := os.Stat(resolved)
	if err != nil {
		return SavedPlan{}, fmt.Errorf("stat plan %s: %w", resolved, err)
	}
	if info.Mode().Perm() != 0o600 {
		return SavedPlan{}, fmt.Errorf("plan %s must have 0600 permissions", resolved)
	}
	data, err := os.ReadFile(resolved)
	if err != nil {
		return SavedPlan{}, fmt.Errorf("read plan %s: %w", resolved, err)
	}
	var plan SavedPlan
	if err := json.Unmarshal(data, &plan); err != nil {
		return SavedPlan{}, fmt.Errorf("parse plan %s: %w", resolved, err)
	}
	if plan.SchemaVersion != SavedPlanSchemaVersion {
		return SavedPlan{}, fmt.Errorf("plan %s uses schema version %d, expected %d", resolved, plan.SchemaVersion, SavedPlanSchemaVersion)
	}
	return plan, nil
}

func (s PlanStore) List() ([]PlanFile, error) {
	entries, err := os.ReadDir(s.PlanDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read plan directory %s: %w", s.PlanDir, err)
	}

	now := time.Now()
	if s.Now != nil {
		now = s.Now()
	}

	plans := make([]PlanFile, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		fullPath := filepath.Join(s.PlanDir, entry.Name())
		plan, err := s.Load(fullPath)
		if err != nil {
			return nil, err
		}
		plans = append(plans, PlanFile{
			Path:       fullPath,
			PlanID:     plan.PlanID,
			ProviderID: plan.ProviderID,
			CreatedAt:  plan.CreatedAt,
			ExpiresAt:  plan.ExpiresAt,
			Expired:    !plan.ExpiresAt.IsZero() && plan.ExpiresAt.Before(now),
		})
	}

	sort.Slice(plans, func(i, j int) bool {
		if plans[i].CreatedAt.Equal(plans[j].CreatedAt) {
			return plans[i].Path < plans[j].Path
		}
		return plans[i].CreatedAt.Before(plans[j].CreatedAt)
	})
	return plans, nil
}

func (s PlanStore) Clean(opts CleanOptions) ([]string, error) {
	if !opts.ExpiredOnly && !opts.RemoveAll {
		return nil, fmt.Errorf("clean requires --expired or --all")
	}
	plans, err := s.List()
	if err != nil {
		return nil, err
	}

	removed := make([]string, 0, len(plans))
	for _, plan := range plans {
		if opts.ExpiredOnly && !plan.Expired {
			continue
		}
		if err := os.Remove(plan.Path); err != nil {
			return removed, fmt.Errorf("remove plan %s: %w", plan.Path, err)
		}
		removed = append(removed, plan.Path)
	}
	return removed, nil
}

func NewPlanID() (string, error) {
	var buf [16]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return "", fmt.Errorf("generate plan id: %w", err)
	}
	return hex.EncodeToString(buf[:]), nil
}

func PlanFileName(planID string, createdAt time.Time) string {
	prefix := planID
	if len(prefix) > 8 {
		prefix = prefix[:8]
	}
	return fmt.Sprintf("usync-plan-%s-%s.json", createdAt.UTC().Format("20060102"), prefix)
}

func MarshalSavedPlanJSON(plan SavedPlan) ([]byte, error) {
	data, err := json.MarshalIndent(plan, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal plan json: %w", err)
	}
	return append(data, '\n'), nil
}

func (s PlanStore) resolveOutputPath(plan SavedPlan, outPath string) (string, error) {
	if strings.TrimSpace(outPath) == "" {
		return filepath.Join(s.PlanDir, PlanFileName(plan.PlanID, plan.CreatedAt)), nil
	}

	resolved := filepath.Clean(outPath)
	if info, err := os.Stat(resolved); err == nil && info.IsDir() {
		resolved = filepath.Join(resolved, PlanFileName(plan.PlanID, plan.CreatedAt))
	}
	absResolved, err := filepath.Abs(resolved)
	if err != nil {
		return "", fmt.Errorf("resolve plan path %s: %w", outPath, err)
	}
	if err := s.ensureAllowedPlanPath(absResolved); err != nil {
		return "", err
	}
	return absResolved, nil
}

func (s PlanStore) validatePlanPath(path string) (string, error) {
	if strings.TrimSpace(path) == "" {
		return "", fmt.Errorf("missing plan path")
	}
	resolved, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("resolve plan path %s: %w", path, err)
	}
	if err := s.ensureAllowedPlanPath(resolved); err != nil {
		return "", err
	}
	return resolved, nil
}

func (s PlanStore) ensureAllowedPlanPath(path string) error {
	withinHome, err := pathWithinRoot(s.HomeDir, path)
	if err != nil {
		return err
	}
	withinPlanDir, err := pathWithinRoot(s.PlanDir, path)
	if err != nil {
		return err
	}
	if withinHome || withinPlanDir {
		return nil
	}
	return fmt.Errorf("plan path %s must be inside the configured home or plan directory", path)
}

func pathWithinRoot(root, target string) (bool, error) {
	if root == "" {
		return false, nil
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return false, fmt.Errorf("resolve root %s: %w", root, err)
	}
	absTarget, err := filepath.Abs(target)
	if err != nil {
		return false, fmt.Errorf("resolve target %s: %w", target, err)
	}
	rel, err := filepath.Rel(absRoot, absTarget)
	if err != nil {
		return false, fmt.Errorf("resolve relative path from %s to %s: %w", absRoot, absTarget, err)
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return false, nil
	}
	return true, nil
}

func writePlanAtomic(path string, data []byte) error {
	dir := filepath.Dir(path)
	temp, err := os.CreateTemp(dir, ".usync-plan-*")
	if err != nil {
		return fmt.Errorf("create temp plan file for %s: %w", path, err)
	}

	tempPath := temp.Name()
	cleanup := true
	defer func() {
		if cleanup {
			_ = os.Remove(tempPath)
		}
	}()

	if err := temp.Chmod(0o600); err != nil {
		_ = temp.Close()
		return fmt.Errorf("chmod temp plan file for %s: %w", path, err)
	}
	if _, err := temp.Write(data); err != nil {
		_ = temp.Close()
		return fmt.Errorf("write temp plan file for %s: %w", path, err)
	}
	if err := temp.Close(); err != nil {
		return fmt.Errorf("close temp plan file for %s: %w", path, err)
	}
	if err := os.Rename(tempPath, path); err != nil {
		return fmt.Errorf("rename temp plan file for %s: %w", path, err)
	}
	if err := os.Chmod(path, 0o600); err != nil {
		return fmt.Errorf("chmod plan file %s: %w", path, err)
	}
	cleanup = false
	return nil
}
