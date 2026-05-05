package tools

import "testing"

func TestIsForbiddenEnvFile_BlockedCases(t *testing.T) {
	t.Parallel()

	blocked := []string{
		".env",
		"./.env",
		"config/.env",
		".env.local",
		"a/b/.env.production",
		`a\\b\\.env.stage`,
	}

	for _, path := range blocked {
		path := path
		t.Run(path, func(t *testing.T) {
			t.Parallel()
			if !isForbiddenEnvFile(path) {
				t.Fatalf("expected path to be blocked: %s", path)
			}
		})
	}
}

func TestIsForbiddenEnvFile_AllowedCases(t *testing.T) {
	t.Parallel()

	allowed := []string{
		".env.example",
		"config/.env.example",
		"env.md",
		"docs/environment.md",
		".environment",
	}

	for _, path := range allowed {
		path := path
		t.Run(path, func(t *testing.T) {
			t.Parallel()
			if isForbiddenEnvFile(path) {
				t.Fatalf("expected path to be allowed: %s", path)
			}
		})
	}
}
