package opencodego

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/dhanifudin/pakai/internal/config"
	"github.com/dhanifudin/pakai/internal/schema"
)

const providerID = "opencode-go"

// Env vars for web dashboard auth.
const (
	envCookie      = "OPENCODE_COOKIE"
	envWorkspaceID = "OPENCODE_WORKSPACE_ID"
)

// Provider fetches opencode-go usage from the opencode.ai web dashboard.
type Provider struct {
	httpClient *http.Client
}

// New creates a new opencode-go provider.
func New() *Provider {
	return &Provider{
		httpClient: &http.Client{Timeout: 15 * time.Second},
	}
}

// ID returns "opencode-go".
func (p *Provider) ID() string {
	return providerID
}

// Fetch returns usage windows from the opencode.ai billing dashboard.
// Requires OPENCODE_COOKIE and OPENCODE_WORKSPACE_ID env vars.
func (p *Provider) Fetch(ctx context.Context) (*schema.Usage, error) {
	cookie := getEnv(envCookie)
	wid := getEnv(envWorkspaceID)

	if cookie == "" || wid == "" {
		return &schema.Usage{
			Provider:    providerID,
			Label:       config.GetProviderLabel(providerID),
			Status:      schema.StatusError,
			Error:       "OPENCODE_COOKIE and OPENCODE_WORKSPACE_ID env vars not set",
			RefreshedAt: time.Now(),
		}, nil
	}

	windows, err := fetchDashboard(ctx, p.httpClient, cookie, wid)
	if err != nil {
		return &schema.Usage{
			Provider:    providerID,
			Label:       config.GetProviderLabel(providerID),
			Status:      schema.StatusError,
			Error:       err.Error(),
			RefreshedAt: time.Now(),
		}, nil
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

// getEnv reads from process env first, then falls back to .env files.
func getEnv(key string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	// Check ~/.config/pakai/.env
	home, _ := os.UserHomeDir()
	if home != "" {
		loadDotenv(filepath.Join(home, ".config", "pakai", ".env"))
	}
	// Check cwd/.env
	if wd, err := os.Getwd(); err == nil {
		loadDotenv(filepath.Join(wd, ".env"))
	}
	return os.Getenv(key)
}

func loadDotenv(path string) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		k = strings.TrimSpace(k)
		v = strings.TrimSpace(v)
		// Only set if not already in process env
		if os.Getenv(k) == "" {
			os.Setenv(k, v)
		}
	}
}
func fetchDashboard(ctx context.Context, client *http.Client, cookie, workspaceID string) ([]schema.UsageWindow, error) {
	url := fmt.Sprintf("https://opencode.ai/workspace/%s/go", workspaceID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("opencode-go: request error: %w", err)
	}
	req.Header.Set("Cookie", "auth="+cookie)
	req.Header.Set("User-Agent", "pakai/1.0")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("opencode-go: fetch error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return nil, fmt.Errorf("opencode-go: cookie expired or invalid (status %d)", resp.StatusCode)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("opencode-go: billing page returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("opencode-go: read error: %w", err)
	}

	return parseDashboard(string(body)), nil
}

var (
	rollingPctRe = regexp.MustCompile(`rollingUsage[^}]*?usagePercent\s*:\s*([0-9]+(?:\.[0-9]+)?)`)
	rollingRstRe = regexp.MustCompile(`rollingUsage[^}]*?resetInSec\s*:\s*([0-9]+)`)
	weeklyPctRe  = regexp.MustCompile(`weeklyUsage[^}]*?usagePercent\s*:\s*([0-9]+(?:\.[0-9]+)?)`)
	weeklyRstRe  = regexp.MustCompile(`weeklyUsage[^}]*?resetInSec\s*:\s*([0-9]+)`)
	monthlyPctRe = regexp.MustCompile(`monthlyUsage[^}]*?usagePercent\s*:\s*([0-9]+(?:\.[0-9]+)?)`)
	monthlyRstRe = regexp.MustCompile(`monthlyUsage[^}]*?resetInSec\s*:\s*([0-9]+)`)
)

func parseDashboard(text string) []schema.UsageWindow {
	now := time.Now()

	rp, rok := extractFloat(rollingPctRe, text)
	rr, rrok := extractInt(rollingRstRe, text)
	if !rok || !rrok {
		return nil
	}

	windows := []schema.UsageWindow{{
		Key:     "5h",
		Label:   "5h",
		Used:    rp,
		Limit:   100,
		Unit:    "percent",
		ResetAt: now.Add(time.Duration(rr) * time.Second),
	}}

	if wp, wok := extractFloat(weeklyPctRe, text); wok {
		if wr, wrok := extractInt(weeklyRstRe, text); wrok {
			windows = append(windows, schema.UsageWindow{
				Key:     "weekly",
				Label:   "weekly",
				Used:    wp,
				Limit:   100,
				Unit:    "percent",
				ResetAt: now.Add(time.Duration(wr) * time.Second),
			})
		}
	}

	mp, mok := extractFloat(monthlyPctRe, text)
	mr, mrok := extractInt(monthlyRstRe, text)
	if mok || mrok {
		windows = append(windows, schema.UsageWindow{
			Key:     "monthly",
			Label:   "monthly",
			Used:    mp,
			Limit:   100,
			Unit:    "percent",
			ResetAt: now.Add(time.Duration(mr) * time.Second),
		})
	}

	return windows
}

func extractFloat(re *regexp.Regexp, text string) (float64, bool) {
	m := re.FindStringSubmatch(text)
	if len(m) < 2 {
		return 0, false
	}
	v, err := strconv.ParseFloat(m[1], 64)
	if err != nil {
		return 0, false
	}
	return v, true
}

func extractInt(re *regexp.Regexp, text string) (int, bool) {
	m := re.FindStringSubmatch(text)
	if len(m) < 2 {
		return 0, false
	}
	v, err := strconv.Atoi(m[1])
	if err != nil {
		return 0, false
	}
	return v, true
}
