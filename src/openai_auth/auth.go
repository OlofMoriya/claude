package openai_auth

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

const (
	openAIOAuthClientID   = "app_EMoamEEZ73f0CkXaXp7hrann"
	openAIOAuthTokenURL   = "https://auth.openai.com/oauth/token"
	openAIOAuthEarlySkew  = 60 * 1000
	openAIOAuthFilePath   = ".owl/auth/openai.json"
	openAIOAuthFilePerm   = 0o600
	openAIOAuthFolderPerm = 0o700
)

type ResolvedAuth struct {
	Token     string
	AccountID string
	IsCodex   bool
}

type Status string

const (
	StatusNone   Status = "none"
	StatusAPIKey Status = "api_key"
	StatusOAuth  Status = "oauth"
)

type oauthFile struct {
	Type         string `json:"type"`
	Access       string `json:"access"`
	Refresh      string `json:"refresh"`
	Expires      int64  `json:"expires"`
	AccountID    string `json:"accountId"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresAt    int64  `json:"expires_at"`
	AccountIDAlt string `json:"account_id"`
}

type refreshResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
}

var refreshToken = refreshTokenFromAPI

func Resolve() (ResolvedAuth, error) {
	oauth, path, err := loadOAuthFile()
	if err == nil {
		now := time.Now().UnixMilli()
		expiresAt := oauth.expiresUnixMs()
		if oauth.refreshToken() != "" && oauth.accessToken() != "" && now+openAIOAuthEarlySkew >= expiresAt {
			refreshed, refreshErr := refreshToken(oauth.refreshToken())
			if refreshErr != nil {
				return ResolvedAuth{}, fmt.Errorf("openai oauth refresh failed: %w", refreshErr)
			}
			oauth.updateFromRefresh(refreshed)
			if writeErr := writeOAuthFile(path, oauth); writeErr != nil {
				return ResolvedAuth{}, fmt.Errorf("openai oauth persist failed: %w", writeErr)
			}
		}

		if oauth.accessToken() != "" {
			return ResolvedAuth{Token: oauth.accessToken(), AccountID: oauth.accountID(), IsCodex: true}, nil
		}
	}

	apiKey, ok := os.LookupEnv("OPENAI_API_KEY")
	if !ok || apiKey == "" {
		return ResolvedAuth{}, fmt.Errorf("could not fetch OPENAI_API_KEY and no codex oauth token found")
	}

	return ResolvedAuth{Token: apiKey, IsCodex: false}, nil
}

func HasCodexOAuthCredential() bool {
	oauth, _, err := loadOAuthFile()
	if err != nil {
		return false
	}
	return oauth.accessToken() != "" && oauth.refreshToken() != ""
}

func CurrentStatus() Status {
	if HasCodexOAuthCredential() {
		return StatusOAuth
	}
	apiKey, ok := os.LookupEnv("OPENAI_API_KEY")
	if ok && apiKey != "" {
		return StatusAPIKey
	}
	return StatusNone
}

func loadOAuthFile() (oauthFile, string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return oauthFile{}, "", err
	}
	path := filepath.Join(home, openAIOAuthFilePath)
	content, err := os.ReadFile(path)
	if err != nil {
		return oauthFile{}, path, err
	}

	var auth oauthFile
	if err := json.Unmarshal(content, &auth); err != nil {
		return oauthFile{}, path, err
	}

	return auth, path, nil
}

func writeOAuthFile(path string, auth oauthFile) error {
	if err := os.MkdirAll(filepath.Dir(path), openAIOAuthFolderPerm); err != nil {
		return err
	}

	data, err := json.MarshalIndent(auth, "", "  ")
	if err != nil {
		return err
	}

	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, openAIOAuthFilePerm); err != nil {
		return err
	}

	return os.Rename(tmp, path)
}

func refreshTokenFromAPI(refreshTok string) (refreshResponse, error) {
	body := map[string]string{
		"grant_type":    "refresh_token",
		"refresh_token": refreshTok,
		"client_id":     openAIOAuthClientID,
	}

	encoded, err := json.Marshal(body)
	if err != nil {
		return refreshResponse{}, err
	}

	req, err := http.NewRequest(http.MethodPost, openAIOAuthTokenURL, bytes.NewBuffer(encoded))
	if err != nil {
		return refreshResponse{}, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return refreshResponse{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return refreshResponse{}, fmt.Errorf("oauth refresh failed with status %d", resp.StatusCode)
	}

	var out refreshResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return refreshResponse{}, err
	}
	if out.AccessToken == "" {
		return refreshResponse{}, fmt.Errorf("oauth refresh response missing access_token")
	}

	return out, nil
}

func (a oauthFile) accessToken() string {
	if a.Access != "" {
		return a.Access
	}
	return a.AccessToken
}

func (a oauthFile) refreshToken() string {
	if a.Refresh != "" {
		return a.Refresh
	}
	return a.RefreshToken
}

func (a oauthFile) expiresUnixMs() int64 {
	if a.Expires > 0 {
		return a.Expires
	}
	return a.ExpiresAt
}

func (a oauthFile) accountID() string {
	if a.AccountID != "" {
		return a.AccountID
	}
	return a.AccountIDAlt
}

func (a *oauthFile) updateFromRefresh(refresh refreshResponse) {
	a.Access = refresh.AccessToken
	a.AccessToken = refresh.AccessToken
	if refresh.RefreshToken != "" {
		a.Refresh = refresh.RefreshToken
		a.RefreshToken = refresh.RefreshToken
	}
	expires := time.Now().UnixMilli() + refresh.ExpiresIn*1000
	if refresh.ExpiresIn <= 0 {
		expires = time.Now().UnixMilli() + 3600*1000
	}
	a.Expires = expires
	a.ExpiresAt = expires
}
