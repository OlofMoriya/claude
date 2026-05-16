package tools

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"owl/interaction"
)

func TestDisplayFileInTUIToolRequiresTUI(t *testing.T) {
	orig := interaction.FileDisplayPromptChan
	interaction.FileDisplayPromptChan = nil
	defer func() { interaction.FileDisplayPromptChan = orig }()

	tool := &DisplayFileInTUITool{}
	_, err := tool.Run(map[string]string{"path": "README.md"})
	if err == nil || !strings.Contains(err.Error(), "requires TUI interactive mode") {
		t.Fatalf("expected TUI mode error, got %v", err)
	}
}

func TestDisplayFileInTUIToolValidatesRange(t *testing.T) {
	_, _, err := parseLineRange(map[string]string{"start_line": "10", "end_line": "2"})
	if err == nil || !strings.Contains(err.Error(), "start_line cannot be greater") {
		t.Fatalf("expected range error, got %v", err)
	}
}

func TestDisplayFileInTUIToolDisplaysAndReturnsAck(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sample.txt")
	if err := os.WriteFile(path, []byte("line1\nline2\nline3\n"), 0o644); err != nil {
		t.Fatalf("write failed: %v", err)
	}

	orig := interaction.FileDisplayPromptChan
	interaction.FileDisplayPromptChan = make(chan interaction.FileDisplayPrompt, 1)
	defer func() { interaction.FileDisplayPromptChan = orig }()

	go func() {
		prompt := <-interaction.FileDisplayPromptChan
		if !strings.Contains(prompt.Content, "line2") {
			t.Errorf("expected content in prompt")
		}
		prompt.ResponseChan <- interaction.FileDisplayResult{}
	}()

	tool := &DisplayFileInTUITool{}
	out, err := tool.Run(map[string]string{"path": path, "start_line": "2", "end_line": "3"})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if !strings.Contains(out, "Displayed file in TUI") {
		t.Fatalf("unexpected result: %s", out)
	}
}
