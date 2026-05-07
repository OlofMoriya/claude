package openai_auth

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	openAIOAuthClientID   = "app_EMoamEEZ73f0CkXaXp7hrann"
	openAIOAuthIssuer     = "https://auth.openai.com"
	openAIOAuthTokenURL   = "https://auth.openai.com/oauth/token"
	openAIOAuthEarlySkew  = 60 * 1000
	openAIOAuthPollMargin = 3 * time.Second
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

type deviceCodeResponse struct {
	DeviceAuthID string `json:"device_auth_id"`
	UserCode     string `json:"user_code"`
	Interval     string `json:"interval"`
}

type deviceTokenResponse struct {
	AuthorizationCode string `json:"authorization_code"`
	CodeVerifier      string `json:"code_verifier"`
}

type loginResult struct {
	VerificationURL string
	UserCode        string
}

var refreshToken = refreshTokenFromAPI

func Login() (string, error) {
	result, tokens, err := loginDeviceFlow()
	if err != nil {
		return "", err
	}

	auth := oauthFile{
		Type:      "oauth",
		Access:    tokens.AccessToken,
		Refresh:   tokens.RefreshToken,
		AccountID: extractAccountID(tokens.AccessToken),
	}
	auth.updateFromRefresh(tokens)

	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	path := filepath.Join(home, openAIOAuthFilePath)
	if err := writeOAuthFile(path, auth); err != nil {
		return "", err
	}

	return fmt.Sprintf("OpenAI login successful. Visit %s and enter code %s", result.VerificationURL, result.UserCode), nil
}

func Logout() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	path := filepath.Join(home, openAIOAuthFilePath)
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

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

func loginDeviceFlow() (loginResult, refreshResponse, error) {
	deviceURL := openAIOAuthIssuer + "/api/accounts/deviceauth/usercode"

	body := map[string]string{"client_id": openAIOAuthClientID}
	encoded, err := json.Marshal(body)
	if err != nil {
		return loginResult{}, refreshResponse{}, err
	}

	req, err := http.NewRequest(http.MethodPost, deviceURL, bytes.NewBuffer(encoded))
	if err != nil {
		return loginResult{}, refreshResponse{}, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return loginResult{}, refreshResponse{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return loginResult{}, refreshResponse{}, fmt.Errorf("device auth start failed with status %d", resp.StatusCode)
	}

	var device deviceCodeResponse
	if err := json.NewDecoder(resp.Body).Decode(&device); err != nil {
		return loginResult{}, refreshResponse{}, err
	}
	if device.DeviceAuthID == "" || device.UserCode == "" {
		return loginResult{}, refreshResponse{}, fmt.Errorf("device auth response missing fields")
	}

	interval := 5 * time.Second
	if parsed, err := time.ParseDuration(device.Interval + "s"); err == nil && parsed > 0 {
		interval = parsed
	}

	statusURL := openAIOAuthIssuer + "/api/accounts/deviceauth/token"
	for attempt := 0; attempt < 120; attempt++ {
		tokens, done, err := pollDeviceToken(statusURL, device)
		if err != nil {
			return loginResult{}, refreshResponse{}, err
		}
		if done {
			return loginResult{VerificationURL: openAIOAuthIssuer + "/codex/device", UserCode: device.UserCode}, tokens, nil
		}
		time.Sleep(interval + openAIOAuthPollMargin)
	}

	return loginResult{}, refreshResponse{}, fmt.Errorf("device auth timed out")
}

func pollDeviceToken(statusURL string, device deviceCodeResponse) (refreshResponse, bool, error) {
	body := map[string]string{
		"device_auth_id": device.DeviceAuthID,
		"user_code":      device.UserCode,
	}
	encoded, err := json.Marshal(body)
	if err != nil {
		return refreshResponse{}, false, err
	}

	req, err := http.NewRequest(http.MethodPost, statusURL, bytes.NewBuffer(encoded))
	if err != nil {
		return refreshResponse{}, false, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return refreshResponse{}, false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusNotFound {
		return refreshResponse{}, false, nil
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return refreshResponse{}, false, fmt.Errorf("device auth poll failed with status %d", resp.StatusCode)
	}

	var tokenResp deviceTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return refreshResponse{}, false, err
	}
	if tokenResp.AuthorizationCode == "" || tokenResp.CodeVerifier == "" {
		return refreshResponse{}, false, fmt.Errorf("device auth token response missing fields")
	}

	tokens, err := exchangeAuthorizationCode(tokenResp.AuthorizationCode, tokenResp.CodeVerifier)
	if err != nil {
		return refreshResponse{}, false, err
	}
	return tokens, true, nil
}

func exchangeAuthorizationCode(code string, verifier string) (refreshResponse, error) {
	body := map[string]string{
		"grant_type":    "authorization_code",
		"code":          code,
		"redirect_uri":  openAIOAuthIssuer + "/deviceauth/callback",
		"client_id":     openAIOAuthClientID,
		"code_verifier": verifier,
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
		return refreshResponse{}, fmt.Errorf("token exchange failed with status %d", resp.StatusCode)
	}

	var out refreshResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return refreshResponse{}, err
	}
	if out.AccessToken == "" || out.RefreshToken == "" {
		return refreshResponse{}, fmt.Errorf("token exchange response missing token fields")
	}
	return out, nil
}

func extractAccountID(accessToken string) string {
	parts := bytes.Split([]byte(accessToken), []byte("."))
	if len(parts) != 3 {
		return ""
	}
	payload, err := decodeBase64URL(parts[1])
	if err != nil {
		return ""
	}
	var claims map[string]any
	if err := json.Unmarshal(payload, &claims); err != nil {
		return ""
	}
	if value, ok := claims["chatgpt_account_id"].(string); ok {
		return value
	}
	if nested, ok := claims["https://api.openai.com/auth"].(map[string]any); ok {
		if value, ok := nested["chatgpt_account_id"].(string); ok {
			return value
		}
	}
	return ""
}

func decodeBase64URL(raw []byte) ([]byte, error) {
	for len(raw)%4 != 0 {
		raw = append(raw, '=')
	}
	normalized := strings.NewReplacer("-", "+", "_", "/").Replace(string(raw))
	out := make([]byte, len(normalized))
	n, err := base64.StdEncoding.Decode(out, []byte(normalized))
	if err != nil {
		return nil, err
	}
	return out[:n], nil
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
