package openai_auth

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestResolveFallsBackToAPIKey(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "api-key-123")
	t.Setenv("HOME", t.TempDir())

	auth, err := Resolve()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if auth.Token != "api-key-123" {
		t.Fatalf("expected api key token, got %q", auth.Token)
	}
	if auth.IsCodex {
		t.Fatalf("expected non-codex auth")
	}
}

func TestResolveUsesPersistedOAuth(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "api-key-123")
	home := t.TempDir()
	t.Setenv("HOME", home)

	path := filepath.Join(home, openAIOAuthFilePath)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}
	file := oauthFile{
		Type:      "oauth",
		Access:    "oauth-access",
		Refresh:   "oauth-refresh",
		Expires:   time.Now().Add(10 * time.Minute).UnixMilli(),
		AccountID: "acc-1",
	}
	encoded, _ := json.Marshal(file)
	if err := os.WriteFile(path, encoded, 0o600); err != nil {
		t.Fatalf("write failed: %v", err)
	}

	auth, err := Resolve()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if auth.Token != "oauth-access" {
		t.Fatalf("expected oauth token, got %q", auth.Token)
	}
	if auth.AccountID != "acc-1" {
		t.Fatalf("expected account id, got %q", auth.AccountID)
	}
	if !auth.IsCodex {
		t.Fatalf("expected codex auth mode")
	}
}

func TestResolveRefreshesExpiredToken(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "")
	home := t.TempDir()
	t.Setenv("HOME", home)

	path := filepath.Join(home, openAIOAuthFilePath)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}
	file := oauthFile{
		Type:      "oauth",
		Access:    "old-access",
		Refresh:   "oauth-refresh",
		Expires:   time.Now().Add(-time.Minute).UnixMilli(),
		AccountID: "acc-1",
	}
	encoded, _ := json.Marshal(file)
	if err := os.WriteFile(path, encoded, 0o600); err != nil {
		t.Fatalf("write failed: %v", err)
	}

	original := refreshToken
	t.Cleanup(func() { refreshToken = original })
	refreshToken = func(token string) (refreshResponse, error) {
		return refreshResponse{AccessToken: "new-access", RefreshToken: "new-refresh", ExpiresIn: 3600}, nil
	}

	auth, err := Resolve()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if auth.Token != "new-access" {
		t.Fatalf("expected refreshed token, got %q", auth.Token)
	}

	updatedRaw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read failed: %v", err)
	}
	var updated oauthFile
	if err := json.Unmarshal(updatedRaw, &updated); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if updated.accessToken() != "new-access" {
		t.Fatalf("expected persisted refreshed token, got %q", updated.accessToken())
	}
}

func TestCurrentStatus(t *testing.T) {
	t.Run("oauth", func(t *testing.T) {
		home := t.TempDir()
		t.Setenv("HOME", home)
		t.Setenv("OPENAI_API_KEY", "")

		path := filepath.Join(home, openAIOAuthFilePath)
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatalf("mkdir failed: %v", err)
		}
		content := []byte(`{"type":"oauth","access":"token","refresh":"refresh","expires":9999999999999}`)
		if err := os.WriteFile(path, content, 0o600); err != nil {
			t.Fatalf("write failed: %v", err)
		}

		if got := CurrentStatus(); got != StatusOAuth {
			t.Fatalf("expected oauth status, got %q", got)
		}
	})

	t.Run("api_key", func(t *testing.T) {
		t.Setenv("HOME", t.TempDir())
		t.Setenv("OPENAI_API_KEY", "api-key-123")
		if got := CurrentStatus(); got != StatusAPIKey {
			t.Fatalf("expected api key status, got %q", got)
		}
	})

	t.Run("none", func(t *testing.T) {
		t.Setenv("HOME", t.TempDir())
		t.Setenv("OPENAI_API_KEY", "")
		if got := CurrentStatus(); got != StatusNone {
			t.Fatalf("expected none status, got %q", got)
		}
	})
}

func TestLogoutRemovesAuthFile(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	path := filepath.Join(home, openAIOAuthFilePath)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}
	if err := os.WriteFile(path, []byte(`{"type":"oauth"}`), 0o600); err != nil {
		t.Fatalf("write failed: %v", err)
	}

	if err := Logout(); err != nil {
		t.Fatalf("logout failed: %v", err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("expected auth file removed, got err=%v", err)
	}
}
