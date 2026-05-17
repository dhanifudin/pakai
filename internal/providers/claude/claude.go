package claude

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/dhanifudin/pakai/internal/config"
	"github.com/dhanifudin/pakai/internal/schema"
)

const providerID = "claude"

const (
	credentialsFileName = ".credentials.json"
	oauthUsageURL       = "https://api.anthropic.com/api/oauth/usage"
	oauthTokenURL       = "https://platform.claude.com/v1/oauth/token"
	oauthClientID       = "9d1c250a-e61b-44d9-88ed-5944d1962f5e"
	oauthBetaHeader     = "oauth-2025-04-20"
	apiCacheTTL         = 60 * time.Second
	refreshBuffer       = 5 * time.Minute
)

// statsCache represents the structure of ~/.claude/stats-cache.json
type statsCache struct {
	Version          int                   `json:"version"`
	LastComputedDate string                `json:"lastComputedDate"`
	DailyActivity    []dailyActivity       `json:"dailyActivity"`
	ModelUsage       map[string]modelUsage `json:"modelUsage"`
}

type dailyActivity struct {
	Date          string `json:"date"`
	MessageCount  int    `json:"messageCount"`
	SessionCount  int    `json:"sessionCount"`
	ToolCallCount int    `json:"toolCallCount"`
}

type modelUsage struct {
	InputTokens  int     `json:"inputTokens"`
	OutputTokens int     `json:"outputTokens"`
	CostUSD      float64 `json:"costUSD"`
}

// Provider implements the providers.Provider interface for Claude.
type Provider struct {
	cachePath     string
	credsPath     string
	apiCachePath  string
	readerFactory func() (io.ReadCloser, error)
	httpClient    *http.Client
	now           func() time.Time

	mu             sync.Mutex
	lastAPIAttempt time.Time
	cachedWindows  []schema.UsageWindow
}

type oauthCredentials struct {
	ClaudeAiOauth struct {
		AccessToken  string  `json:"accessToken"`
		RefreshToken string  `json:"refreshToken"`
		ExpiresAt    float64 `json:"expiresAt"`
	} `json:"claudeAiOauth"`
}

type cachedUsageWindows struct {
	FetchedAt time.Time            `json:"fetched_at"`
	Windows   []schema.UsageWindow `json:"windows"`
}

type usageAPIResponse struct {
	FiveHour *usageWindowResponse `json:"five_hour"`
	SevenDay *usageWindowResponse `json:"seven_day"`
}

type usageWindowResponse struct {
	Utilization float64 `json:"utilization"`
	ResetsAt    string  `json:"resets_at"`
}

// New creates a new Claude provider.
func New() *Provider {
	home, _ := os.UserHomeDir()
	return &Provider{
		cachePath:    filepath.Join(home, ".claude", "stats-cache.json"),
		credsPath:    filepath.Join(home, ".claude", credentialsFileName),
		apiCachePath: filepath.Join(home, ".cache", "pakai", "claude-usage.json"),
		httpClient:   &http.Client{Timeout: 15 * time.Second},
		now:          time.Now,
	}
}

// NewWithPath creates a new Claude provider with a custom cache path (for testing).
func NewWithPath(path string) *Provider {
	home, _ := os.UserHomeDir()
	return &Provider{
		cachePath:    path,
		credsPath:    filepath.Join(home, ".claude", credentialsFileName),
		apiCachePath: filepath.Join(home, ".cache", "pakai", "claude-usage.json"),
		httpClient:   &http.Client{Timeout: 15 * time.Second},
		now:          time.Now,
	}
}

// ID returns the provider ID.
func (p *Provider) ID() string {
	return providerID
}

// CachePath returns the path to the stats cache file.
func (p *Provider) CachePath() string {
	return p.cachePath
}

// Fetch reads the Claude stats cache and returns the current month's usage.
func (p *Provider) Fetch(ctx context.Context) (*schema.Usage, error) {
	monthlyUsage, statsErr := p.fetchMonthlyUsage(ctx)
	var apiWindows []schema.UsageWindow
	var apiErr error
	if p.readerFactory == nil {
		apiWindows, apiErr = p.fetchAPIWindows(ctx)
	}

	if monthlyUsage != nil {
		if len(apiWindows) > 0 {
			monthlyWindow := schema.UsageWindow{
				Key:         "monthly",
				Label:       "monthly",
				PeriodStart: monthlyUsage.PeriodStart,
				PeriodEnd:   monthlyUsage.PeriodEnd,
				Used:        monthlyUsage.Used,
				Limit:       monthlyUsage.Limit,
				Unit:        monthlyUsage.Unit,
			}
			monthlyUsage.Windows = append(apiWindows, monthlyWindow)
		} else if apiErr != nil {
			monthlyUsage.Warning = "live percentage unavailable"
		}
		return monthlyUsage, nil
	}

	if len(apiWindows) > 0 {
		now := p.now()
		return &schema.Usage{
			Provider:    providerID,
			Label:       providerID,
			Windows:     apiWindows,
			Status:      schema.StatusOK,
			RefreshedAt: now,
		}, nil
	}

	if statsErr != nil {
		return statsErr, nil
	}

	return p.errorUsage(ctx, "no Claude usage data available"), nil
}

func (p *Provider) fetchMonthlyUsage(ctx context.Context) (*schema.Usage, *schema.Usage) {
	if p.readerFactory != nil {
		return p.fetchFromFactory(ctx)
	}
	return p.fetchFromPath(ctx)
}

func (p *Provider) fetchFromFactory(ctx context.Context) (*schema.Usage, *schema.Usage) {
	rc, err := p.readerFactory()
	if err != nil {
		errUsage := p.errorUsage(ctx, fmt.Sprintf("failed to open stats cache: %v", err))
		return nil, errUsage
	}
	defer rc.Close()

	data, err := io.ReadAll(rc)
	if err != nil {
		errUsage := p.errorUsage(ctx, fmt.Sprintf("failed to read stats cache: %v", err))
		return nil, errUsage
	}

	return p.parseStats(ctx, data)
}

func (p *Provider) fetchFromPath(ctx context.Context) (*schema.Usage, *schema.Usage) {
	data, err := os.ReadFile(p.cachePath)
	if err != nil {
		if os.IsNotExist(err) {
			errUsage := p.errorUsage(ctx, "Claude stats cache not found — ensure Claude Code is running")
			return nil, errUsage
		}
		errUsage := p.errorUsage(ctx, fmt.Sprintf("failed to read stats cache: %v", err))
		return nil, errUsage
	}

	return p.parseStats(ctx, data)
}

func (p *Provider) errorUsage(ctx context.Context, msg string) *schema.Usage {
	now := time.Now()
	return &schema.Usage{
		Provider:    providerID,
		Label:       providerID,
		Status:      schema.StatusError,
		Error:       msg,
		RefreshedAt: now,
	}
}

func (p *Provider) parseStats(ctx context.Context, data []byte) (*schema.Usage, *schema.Usage) {
	now := p.now()

	var cache statsCache
	if err := json.Unmarshal(data, &cache); err != nil {
		errUsage := p.errorUsage(ctx, fmt.Sprintf("failed to parse stats cache: %v", err))
		return nil, errUsage
	}

	periodStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	periodEnd := periodStart.AddDate(0, 1, 0).Add(-time.Second)

	// Aggregate message count for the current calendar month
	currentMonth := now.Format("2006-01")
	var totalMessages int
	for _, activity := range cache.DailyActivity {
		if len(activity.Date) >= 7 && activity.Date[:7] == currentMonth {
			totalMessages += activity.MessageCount
		}
	}

	u := &schema.Usage{
		Provider:    providerID,
		Label:       providerID,
		PeriodStart: periodStart,
		PeriodEnd:   periodEnd,
		Unit:        "messages",
		Used:        float64(totalMessages),
		Limit:       config.GetProviderLimit(providerID),
		Status:      schema.StatusOK,
		RefreshedAt: now,
	}

	return u, nil
}

func (p *Provider) fetchAPIWindows(ctx context.Context) ([]schema.UsageWindow, error) {
	now := p.now()

	p.mu.Lock()
	if !p.lastAPIAttempt.IsZero() && now.Sub(p.lastAPIAttempt) < apiCacheTTL {
		cached := cloneWindows(p.cachedWindows)
		p.mu.Unlock()
		return cached, nil
	}
	p.lastAPIAttempt = now
	p.mu.Unlock()

	if cached, ok := p.readAPICache(false); ok {
		p.mu.Lock()
		p.cachedWindows = cloneWindows(cached)
		p.mu.Unlock()
		return cached, nil
	}

	creds, err := p.readCredentials()
	if err != nil || creds.ClaudeAiOauth.AccessToken == "" {
		if cached, ok := p.readAPICache(true); ok {
			return cached, nil
		}
		return nil, err
	}

	accessToken, err := p.ensureAccessToken(ctx, creds, false)
	if err != nil || accessToken == "" {
		if cached, ok := p.readAPICache(true); ok {
			return cached, nil
		}
		return nil, err
	}

	windows, statusCode, err := p.requestUsageWindows(ctx, accessToken)
	if statusCode == http.StatusUnauthorized && creds.ClaudeAiOauth.RefreshToken != "" {
		accessToken, err = p.ensureAccessToken(ctx, creds, true)
		if err == nil && accessToken != "" {
			windows, statusCode, err = p.requestUsageWindows(ctx, accessToken)
		}
	}
	if err != nil {
		if cached, ok := p.readAPICache(true); ok {
			return cached, nil
		}
		return nil, err
	}
	if statusCode != http.StatusOK {
		if cached, ok := p.readAPICache(true); ok {
			return cached, nil
		}
		return nil, fmt.Errorf("oauth usage request failed: status %d", statusCode)
	}

	p.mu.Lock()
	p.cachedWindows = cloneWindows(windows)
	p.mu.Unlock()
	p.writeAPICache(windows)

	return windows, nil
}

func (p *Provider) readCredentials() (*oauthCredentials, error) {
	data, err := os.ReadFile(p.credsPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("Claude credentials not found — ensure Claude Code is installed and you are logged in")
		}
		return nil, fmt.Errorf("failed to read Claude credentials: %w", err)
	}

	var creds oauthCredentials
	if err := json.Unmarshal(data, &creds); err != nil {
		return nil, fmt.Errorf("failed to parse Claude credentials: %w", err)
	}
	return &creds, nil
}

func (p *Provider) ensureAccessToken(ctx context.Context, creds *oauthCredentials, forceRefresh bool) (string, error) {
	accessToken := creds.ClaudeAiOauth.AccessToken
	refreshToken := creds.ClaudeAiOauth.RefreshToken
	expiresAt := time.UnixMilli(int64(creds.ClaudeAiOauth.ExpiresAt))

	if accessToken == "" {
		return "", nil
	}

	if !forceRefresh && (expiresAt.IsZero() || expiresAt.After(p.now().Add(refreshBuffer)) || refreshToken == "") {
		return accessToken, nil
	}
	if refreshToken == "" {
		return accessToken, nil
	}

	body, err := json.Marshal(map[string]string{
		"grant_type":    "refresh_token",
		"client_id":     oauthClientID,
		"refresh_token": refreshToken,
	})
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, oauthTokenURL, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("anthropic-beta", oauthBetaHeader)
	req.Header.Set("User-Agent", "claude-cli/1.0")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("oauth token refresh failed: %s", resp.Status)
	}

	var payload struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int64  `json:"expires_in"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return "", err
	}
	if payload.AccessToken == "" {
		return "", fmt.Errorf("oauth token refresh returned empty access token")
	}
	updated := *creds
	updated.ClaudeAiOauth.AccessToken = payload.AccessToken
	if payload.RefreshToken != "" {
		updated.ClaudeAiOauth.RefreshToken = payload.RefreshToken
	}
	if payload.ExpiresIn > 0 {
		updated.ClaudeAiOauth.ExpiresAt = float64(p.now().Add(time.Duration(payload.ExpiresIn) * time.Second).UnixMilli())
	}
	_ = p.writeCredentials(&updated)
	return payload.AccessToken, nil
}

func (p *Provider) requestUsageWindows(ctx context.Context, accessToken string) ([]schema.UsageWindow, int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, oauthUsageURL, nil)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("anthropic-beta", oauthBetaHeader)

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
		return nil, resp.StatusCode, fmt.Errorf("failed to parse oauth usage response: %w", err)
	}

	windows := make([]schema.UsageWindow, 0, 2)
	if payload.FiveHour != nil {
		windows = append(windows, usageWindowFromAPI("5h", "5h", payload.FiveHour))
	}
	if payload.SevenDay != nil {
		windows = append(windows, usageWindowFromAPI("weekly", "weekly", payload.SevenDay))
	}
	return windows, resp.StatusCode, nil
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

	oauth, ok := raw["claudeAiOauth"].(map[string]any)
	if !ok {
		oauth = make(map[string]any)
		raw["claudeAiOauth"] = oauth
	}
	oauth["accessToken"] = creds.ClaudeAiOauth.AccessToken
	oauth["refreshToken"] = creds.ClaudeAiOauth.RefreshToken
	oauth["expiresAt"] = creds.ClaudeAiOauth.ExpiresAt

	updated, err := json.MarshalIndent(raw, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p.credsPath, updated, 0600)
}

func (p *Provider) readAPICache(allowStale bool) ([]schema.UsageWindow, bool) {
	data, err := os.ReadFile(p.apiCachePath)
	if err != nil {
		return nil, false
	}
	var cached cachedUsageWindows
	if err := json.Unmarshal(data, &cached); err != nil {
		return nil, false
	}
	if len(cached.Windows) == 0 {
		return nil, false
	}
	if !allowStale && !cached.FetchedAt.IsZero() && p.now().Sub(cached.FetchedAt) > apiCacheTTL {
		return nil, false
	}
	return cloneWindows(cached.Windows), true
}

func (p *Provider) writeAPICache(windows []schema.UsageWindow) {
	if len(windows) == 0 {
		return
	}
	if err := os.MkdirAll(filepath.Dir(p.apiCachePath), 0755); err != nil {
		return
	}
	payload := cachedUsageWindows{FetchedAt: p.now(), Windows: windows}
	data, err := json.Marshal(payload)
	if err != nil {
		return
	}
	_ = os.WriteFile(p.apiCachePath, data, 0644)
}

func usageWindowFromAPI(key, label string, in *usageWindowResponse) schema.UsageWindow {
	window := schema.UsageWindow{
		Key:   key,
		Label: label,
		Used:  in.Utilization,
		Limit: 100,
		Unit:  "percent",
	}
	if in.ResetsAt != "" {
		if resetAt, err := time.Parse(time.RFC3339, in.ResetsAt); err == nil {
			window.ResetAt = resetAt
		}
	}
	return window
}

func cloneWindows(in []schema.UsageWindow) []schema.UsageWindow {
	if len(in) == 0 {
		return nil
	}
	out := make([]schema.UsageWindow, len(in))
	copy(out, in)
	return out
}
