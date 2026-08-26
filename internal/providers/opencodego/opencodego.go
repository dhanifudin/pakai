package opencodego

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/dhanifudin/pakai/internal/config"
	"github.com/dhanifudin/pakai/internal/schema"
)

const (
	providerID = "opencode-go"
	usageURL   = "https://opencode.ai/zen/go/v1/usage"
)

type Provider struct {
	authPath   string
	source     string
	usageURL   string
	httpClient *http.Client
}

type apiKeyCredentials struct {
	Type string `json:"type"`
	Key  string `json:"key"`
}

type usageResponse struct {
	Usage struct {
		Rolling *usageWindow `json:"rolling"`
		Weekly  *usageWindow `json:"weekly"`
		Monthly *usageWindow `json:"monthly"`
	} `json:"usage"`
}

type usageWindow struct {
	Percent  float64   `json:"percent"`
	ResetsAt time.Time `json:"resetsAt"`
}

func New() *Provider {
	home, _ := os.UserHomeDir()
	source := config.GetProviderSource(providerID)
	if source == "" {
		source = "pi"
	}
	path := filepath.Join(home, ".pi", "agent", "auth.json")
	if source == "opencode" {
		path = filepath.Join(home, ".local", "share", "opencode", "auth.json")
	}
	p := newWithPath(path)
	p.source = source
	return p
}

func newWithPath(authPath string) *Provider {
	return &Provider{
		authPath:   authPath,
		source:     "pi",
		usageURL:   usageURL,
		httpClient: &http.Client{Timeout: 15 * time.Second},
	}
}

func (p *Provider) ID() string {
	return providerID
}

func (p *Provider) CredentialsPath() string {
	return p.authPath
}

func (p *Provider) Fetch(ctx context.Context) (*schema.Usage, error) {
	key, err := p.readAPIKey()
	if err != nil {
		return p.errorUsage(err), nil
	}

	windows, err := p.fetchUsage(ctx, key)
	if err != nil {
		return p.errorUsage(err), nil
	}

	return &schema.Usage{
		Provider:    providerID,
		Label:       config.GetProviderLabel(providerID),
		Unit:        "percent",
		Windows:     windows,
		Status:      schema.StatusOK,
		RefreshedAt: time.Now(),
	}, nil
}

func (p *Provider) errorUsage(err error) *schema.Usage {
	return &schema.Usage{
		Provider:    providerID,
		Label:       config.GetProviderLabel(providerID),
		Status:      schema.StatusError,
		Error:       err.Error(),
		RefreshedAt: time.Now(),
	}
}

func (p *Provider) readAPIKey() (string, error) {
	data, err := os.ReadFile(p.authPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("%s OpenCode Go API key not found", p.source)
		}
		return "", fmt.Errorf("read %s OpenCode credentials: %w", p.source, err)
	}
	var auth map[string]apiKeyCredentials
	if err := json.Unmarshal(data, &auth); err != nil {
		return "", fmt.Errorf("parse %s OpenCode credentials: %w", p.source, err)
	}
	keyID, keyType := providerID, "api_key"
	if p.source == "opencode" {
		keyID, keyType = "opencode", "api"
	}
	creds, ok := auth[keyID]
	if !ok || creds.Type != keyType || creds.Key == "" {
		return "", fmt.Errorf("%s OpenCode Go API key not found", p.source)
	}
	return creds.Key, nil
}

func (p *Provider) fetchUsage(ctx context.Context, key string) ([]schema.UsageWindow, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.usageURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create OpenCode usage request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+key)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch OpenCode usage: %w", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
	case http.StatusUnauthorized:
		return nil, fmt.Errorf("OpenCode API key is invalid")
	case http.StatusForbidden:
		return nil, fmt.Errorf("OpenCode API key has no Go subscription")
	default:
		return nil, fmt.Errorf("OpenCode usage request failed: status %d", resp.StatusCode)
	}

	var usage usageResponse
	if err := json.NewDecoder(resp.Body).Decode(&usage); err != nil {
		return nil, fmt.Errorf("parse OpenCode usage: %w", err)
	}
	return usage.windows(), nil
}

func (u usageResponse) windows() []schema.UsageWindow {
	windows := make([]schema.UsageWindow, 0, 3)
	for _, item := range []struct {
		key, label string
		window     *usageWindow
	}{
		{"5h", "5h", u.Usage.Rolling},
		{"weekly", "weekly", u.Usage.Weekly},
		{"monthly", "monthly", u.Usage.Monthly},
	} {
		if item.window == nil {
			continue
		}
		windows = append(windows, schema.UsageWindow{
			Key:     item.key,
			Label:   item.label,
			Used:    item.window.Percent,
			Limit:   100,
			Unit:    "percent",
			ResetAt: item.window.ResetsAt,
		})
	}
	return windows
}
