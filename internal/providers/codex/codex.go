package codex

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/dhanifudin/pakai/internal/schema"
)

const providerID = "openai"

const (
	credentialsFileName = "auth.json"
	oauthUsageURL       = "https://chatgpt.com/backend-api/wham/usage"
	oauthTokenURL       = "https://auth.openai.com/oauth/token"
	oauthClientID       = "app_EMoamEEZ73f0CkXaXp7hrann"
	apiCacheTTL         = 60 * time.Second
	refreshBuffer       = 5 * time.Minute
)

type Provider struct {
	credsPath  string
	httpClient *http.Client
	now        func() time.Time

	mu             sync.Mutex
	lastAPIAttempt time.Time
	cachedWindows  []schema.UsageWindow
}

type oauthCredentials struct {
	Tokens struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		AccountID    string `json:"account_id"`
	} `json:"tokens"`
}

type usageAPIResponse struct {
	RateLimit           *rateLimitResponse `json:"rate_limit"`
	CodeReviewRateLimit *rateLimitResponse `json:"code_review_rate_limit"`
}

type rateLimitResponse struct {
	PrimaryWindow   *windowResponse `json:"primary_window"`
	SecondaryWindow *windowResponse `json:"secondary_window"`
}

type windowResponse struct {
	UsedPercent float64 `json:"used_percent"`
	ResetAt     float64 `json:"reset_at"`
}

type cachedUsageWindows struct {
	FetchedAt time.Time            `json:"fetched_at"`
	Windows   []schema.UsageWindow `json:"windows"`
}

func New() *Provider {
	home, _ := os.UserHomeDir()
	return &Provider{
		credsPath:  filepath.Join(home, ".codex", credentialsFileName),
		httpClient: &http.Client{Timeout: 15 * time.Second},
		now:        time.Now,
	}
}

func (p *Provider) ID() string {
	return providerID
}

func (p *Provider) Fetch(ctx context.Context) (*schema.Usage, error) {
	windows, err := p.fetchUsageWindows(ctx)
	if err != nil {
		return &schema.Usage{
			Provider:    providerID,
			Label:       providerID,
			Status:      schema.StatusError,
			Error:       err.Error(),
			RefreshedAt: p.now(),
		}, nil
	}
	if len(windows) == 0 {
		return &schema.Usage{
			Provider:    providerID,
			Label:       providerID,
			Status:      schema.StatusOK,
			RefreshedAt: p.now(),
		}, nil
	}

	return &schema.Usage{
		Provider:    providerID,
		Label:       providerID,
		Windows:     windows,
		Status:      schema.StatusOK,
		RefreshedAt: p.now(),
	}, nil
}

func (p *Provider) fetchUsageWindows(ctx context.Context) ([]schema.UsageWindow, error) {
	now := p.now()

	p.mu.Lock()
	if !p.lastAPIAttempt.IsZero() && now.Sub(p.lastAPIAttempt) < apiCacheTTL {
		cached := cloneWindows(p.cachedWindows)
		p.mu.Unlock()
		return cached, nil
	}
	p.lastAPIAttempt = now
	p.mu.Unlock()

	creds, err := p.readCredentials()
	if err != nil {
		return nil, err
	}

	accessToken, err := p.ensureAccessToken(ctx, creds)
	if err != nil || accessToken == "" {
		return nil, err
	}

	resp, statusCode, err := p.requestUsage(ctx, accessToken, creds.Tokens.AccountID)
	if err != nil {
		return nil, err
	}
	if statusCode != http.StatusOK {
		return nil, fmt.Errorf("codex usage request failed: status %d", statusCode)
	}

	windows := parseWindows(p.now(), resp)

	p.mu.Lock()
	p.cachedWindows = cloneWindows(windows)
	p.mu.Unlock()

	return windows, nil
}

func (p *Provider) readCredentials() (*oauthCredentials, error) {
	data, err := os.ReadFile(p.credsPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("Codex credentials not found — ensure Codex is installed and you are logged in")
		}
		return nil, fmt.Errorf("failed to read Codex credentials: %w", err)
	}
	var creds oauthCredentials
	if err := json.Unmarshal(data, &creds); err != nil {
		return nil, fmt.Errorf("failed to parse Codex credentials: %w", err)
	}
	return &creds, nil
}

func (p *Provider) ensureAccessToken(ctx context.Context, creds *oauthCredentials) (string, error) {
	if creds.Tokens.AccessToken == "" {
		return "", nil
	}

	// Check if token expiry is within the refresh buffer
	expiry := jwtExp(creds.Tokens.AccessToken)
	if !expiry.IsZero() && expiry.After(p.now().Add(refreshBuffer)) {
		return creds.Tokens.AccessToken, nil
	}

	// Refresh token
	if creds.Tokens.RefreshToken == "" {
		return creds.Tokens.AccessToken, nil
	}

	body, _ := json.Marshal(map[string]string{
		"client_id":     oauthClientID,
		"grant_type":    "refresh_token",
		"refresh_token": creds.Tokens.RefreshToken,
		"scope":         "openid profile email",
	})

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, oauthTokenURL, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("codex token refresh failed: %s", resp.Status)
	}

	var payload struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		IDToken      string `json:"id_token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return "", err
	}
	if payload.AccessToken == "" {
		return "", fmt.Errorf("codex token refresh returned empty access token")
	}

	// Update credentials file
	updated := *creds
	if payload.AccessToken != "" {
		updated.Tokens.AccessToken = payload.AccessToken
	}
	if payload.RefreshToken != "" {
		updated.Tokens.RefreshToken = payload.RefreshToken
	}
	_ = p.writeCredentials(&updated)

	return payload.AccessToken, nil
}

func (p *Provider) requestUsage(ctx context.Context, accessToken, accountID string) (*usageAPIResponse, int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, oauthUsageURL, nil)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	if accountID != "" {
		req.Header.Set("chatgpt-account-id", accountID)
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, resp.StatusCode, nil
	}

	var payload usageAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, resp.StatusCode, err
	}
	return &payload, resp.StatusCode, nil
}

func parseWindows(now time.Time, resp *usageAPIResponse) []schema.UsageWindow {
	var windows []schema.UsageWindow

	if resp.RateLimit != nil {
		if pw := resp.RateLimit.PrimaryWindow; pw != nil {
			windows = append(windows, schema.UsageWindow{
				Key:   "5h",
				Label: "5h",
				Used:  pw.UsedPercent,
				Limit: 100,
				Unit:  "percent",
			})
		}
		if sw := resp.RateLimit.SecondaryWindow; sw != nil {
			windows = append(windows, schema.UsageWindow{
				Key:     "weekly",
				Label:   "weekly",
				Used:    sw.UsedPercent,
				Limit:   100,
				Unit:    "percent",
				ResetAt: time.Unix(int64(sw.ResetAt), 0),
			})
		}
	}

	return windows
}

func (p *Provider) writeCredentials(creds *oauthCredentials) error {
	data, err := os.ReadFile(p.credsPath)
	if err != nil {
		return err
	}
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	tokens, ok := raw["tokens"].(map[string]any)
	if !ok {
		tokens = make(map[string]any)
		raw["tokens"] = tokens
	}
	tokens["access_token"] = creds.Tokens.AccessToken
	tokens["refresh_token"] = creds.Tokens.RefreshToken
	if creds.Tokens.AccountID != "" {
		tokens["account_id"] = creds.Tokens.AccountID
	}
	raw["last_refresh"] = p.now().UTC().Format("2006-01-02T15:04:05.000000000Z")

	updated, err := json.MarshalIndent(raw, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p.credsPath, updated, 0600)
}

func jwtExp(token string) time.Time {
	parts := strings.Split(token, ".")
	if len(parts) < 2 {
		return time.Time{}
	}
	payload := parts[1]
	// base64url decode with padding
	if l := len(payload) % 4; l != 0 {
		payload += strings.Repeat("=", 4-l)
	}
	payload = strings.ReplaceAll(payload, "-", "+")
	payload = strings.ReplaceAll(payload, "_", "/")
	decoded, err := base64.StdEncoding.DecodeString(payload)
	if err != nil {
		return time.Time{}
	}
	var claims struct {
		Exp int64 `json:"exp"`
	}
	if err := json.Unmarshal(decoded, &claims); err != nil {
		return time.Time{}
	}
	if claims.Exp <= 0 {
		return time.Time{}
	}
	return time.Unix(claims.Exp, 0)
}

func cloneWindows(in []schema.UsageWindow) []schema.UsageWindow {
	if len(in) == 0 {
		return nil
	}
	out := make([]schema.UsageWindow, len(in))
	copy(out, in)
	return out
}
