package models

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGetModelForQuery_DefaultsToClaudeWithoutCodexAuth(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	_, modelName := GetModelForQuery("", nil, nil, nil, false, false, false, false)
	if modelName != "claude" {
		t.Fatalf("expected default model claude, got %q", modelName)
	}
}

func TestGetModelForQuery_DefaultsToGptWithCodexAuth(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	path := filepath.Join(home, ".owl", "auth", "openai.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}
	content := []byte(`{"type":"oauth","access":"token","refresh":"refresh","expires":9999999999999}`)
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatalf("write failed: %v", err)
	}

	_, modelName := GetModelForQuery("", nil, nil, nil, false, false, false, false)
	if modelName != "gpt" {
		t.Fatalf("expected default model gpt, got %q", modelName)
	}
}
