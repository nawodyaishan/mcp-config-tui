package tui

import (
	"context"
	"errors"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/nawodyaishan/universal-mcp-sync/pkg/doctor"
)

// FakeScanner implements DashboardScanner for testing.
type FakeScanner struct {
	Report doctor.Report
	Err    error
}

func (s *FakeScanner) Scan(ctx context.Context) (doctor.Report, error) {
	return s.Report, s.Err
}

func TestDashboardScanner_FakeScanner(t *testing.T) {
	ctx := context.Background()
	
	t.Run("returns injected report", func(t *testing.T) {
		expectedReport := doctor.Report{
			Platform: "test-platform",
			Warnings: []string{"test warning"},
		}
		scanner := &FakeScanner{Report: expectedReport}
		report, err := scanner.Scan(ctx)
		
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if report.Platform != expectedReport.Platform {
			t.Errorf("expected platform %q, got %q", expectedReport.Platform, report.Platform)
		}
	})

	t.Run("returns injected error", func(t *testing.T) {
		expectedErr := errors.New("test error")
		scanner := &FakeScanner{Err: expectedErr}
		_, err := scanner.Scan(ctx)
		
		if !errors.Is(err, expectedErr) && err.Error() != expectedErr.Error() {
			t.Fatalf("expected error %v, got %v", expectedErr, err)
		}
	})
}

func TestDashboardModel(t *testing.T) {
	scanner := &FakeScanner{
		Report: doctor.Report{Platform: "test"},
		Err:    nil,
	}

	model := NewDashboardModel(scanner)

	t.Run("initial state", func(t *testing.T) {
		if !model.scanning {
			t.Error("expected scanning to be true")
		}
		view := model.View()
		if !strings.Contains(view, "Scanning for AI") {
			t.Errorf("expected loading view, got %q", view)
		}
	})

	t.Run("init returns command", func(t *testing.T) {
		cmd := model.Init()
		if cmd == nil {
			t.Fatal("expected non-nil tea.Cmd")
		}
		msg := cmd()
		resultMsg, ok := msg.(scanResultMsg)
		if !ok {
			t.Fatalf("expected scanResultMsg, got %T", msg)
		}
		if resultMsg.report.Platform != "test" {
			t.Errorf("expected platform test, got %q", resultMsg.report.Platform)
		}
	})

	t.Run("update handles scan success", func(t *testing.T) {
		cmd := model.Init()
		msg := cmd()
		
		nextModel, cmd := model.Update(msg)
		updatedModel, ok := nextModel.(DashboardModel)
		if !ok {
			t.Fatal("expected DashboardModel")
		}
		
		if updatedModel.scanning {
			t.Error("expected scanning to be false")
		}
		if updatedModel.report.Platform != "test" {
			t.Errorf("expected platform test, got %q", updatedModel.report.Platform)
		}
		if cmd != nil {
			t.Error("expected nil command")
		}
	})

	t.Run("update handles scan error", func(t *testing.T) {
		errScanner := &FakeScanner{Err: errors.New("scan failed")}
		errModel := NewDashboardModel(errScanner)
		
		cmd := errModel.Init()
		msg := cmd()
		
		nextModel, _ := errModel.Update(msg)
		updatedModel := nextModel.(DashboardModel)
		
		if updatedModel.scanning {
			t.Error("expected scanning to be false")
		}
		if updatedModel.err == nil || updatedModel.err.Error() != "scan failed" {
			t.Errorf("expected scan failed error, got %v", updatedModel.err)
		}
		view := updatedModel.View()
		if !strings.Contains(view, "scan failed") {
			t.Errorf("expected error view, got %q", view)
		}
	})

	t.Run("quit keys", func(t *testing.T) {
		keys := []string{"q", "ctrl+c"}
		for _, key := range keys {
			keyMsg := makeKeyMsg(key)
			_, cmd := model.Update(keyMsg)
			if cmd == nil {
				t.Fatalf("expected tea.Quit for %q, got nil", key)
			}
		}
	})
}

func makeKeyMsg(s string) tea.KeyMsg {
	if s == "ctrl+c" {
		return tea.KeyMsg{Type: tea.KeyCtrlC}
	}
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
}

func TestDashboardRedaction(t *testing.T) {
	scanner := &FakeScanner{
		Report: doctor.Report{
			Platform: "test",
			Warnings: []string{"test_sk_secretkey123"},
			Clients: []doctor.ClientFinding{
				{
					Name:          "TestClient",
					Installed:     true,
					Confidence:    doctor.ConfidenceHigh,
					EffectivePath: "/users/test/.config/test_sk_secretkey456.json",
					Issues:        []string{"found token test_sk_secretkey789"},
				},
			},
		},
	}
	model := NewDashboardModel(scanner)
	cmd := model.Init()
	msg := cmd()
	nextModel, _ := model.Update(msg)
	view := nextModel.View()

	if strings.Contains(view, "test_sk_secretkey") {
		t.Errorf("expected sensitive data to be redacted in view, got %q", view)
	}
}
