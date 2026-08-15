package opencode

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dhanifudin/pakai/internal/config"
	"github.com/dhanifudin/pakai/internal/schema"
	_ "modernc.org/sqlite"
)

// knownDBNames lists candidate filenames in preference order.
// Older opencode releases used "opencode.db"; newer ones use "opencode-stable.db".
var knownDBNames = []string{"opencode-stable.db", "opencode.db"}

// resolveDBPath returns the first existing db file under dir, falling back to
// the first candidate if none exist yet.
func resolveDBPath(dir string) string {
	for _, name := range knownDBNames {
		p := filepath.Join(dir, name)
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return filepath.Join(dir, knownDBNames[0])
}

// Per-window usage limits from https://opencode.ai/docs/go/
const (
	go5HLimit    = 12.0
	goWeekLimit  = 30.0
	goMonthLimit = 60.0
)

// providerRow holds a single row from the OpenCode query.
type providerRow struct {
	ProviderID   string
	MonthTokens  float64
	WeekTokens   float64
	FiveHTokens  float64
	MonthCostUSD float64
	WeekCostUSD  float64
	FiveHCostUSD float64
}

func (r providerRow) TotalTokens() float64 {
	return r.MonthTokens
}

// Provider implements the providers.Provider interface for OpenCode.
type Provider struct {
	db     *sql.DB
	dbPath string
}

// New creates a new OpenCode provider.
func New() *Provider {
	home, _ := os.UserHomeDir()
	return &Provider{
		dbPath: resolveDBPath(filepath.Join(home, ".local", "share", "opencode")),
	}
}

// NewWithPath creates a new OpenCode provider with a custom db path (for testing).
func NewWithPath(path string) *Provider {
	return &Provider{dbPath: path}
}

// NewFromDB creates a new OpenCode provider with an injected *sql.DB (for testing).
func NewFromDB(db *sql.DB) *Provider {
	return &Provider{db: db}
}

// ID returns "opencode" — we return multiple usages via FetchAll.
func (p *Provider) ID() string {
	return "opencode"
}

// DBPath returns the path to the SQLite database.
func (p *Provider) DBPath() string {
	return p.dbPath
}

// Fetch returns aggregate usage across all OpenCode providers combined.
func (p *Provider) Fetch(ctx context.Context) (*schema.Usage, error) {
	usages, err := p.FetchAll(ctx)
	if err != nil {
		return &schema.Usage{
			Provider:    p.ID(),
			Label:       config.GetProviderLabel(p.ID()),
			Unit:        "usd",
			Status:      schema.StatusError,
			Error:       err.Error(),
			RefreshedAt: time.Now(),
		}, nil
	}
	if len(usages) == 0 {
		return &schema.Usage{
			Provider:    p.ID(),
			Label:       config.GetProviderLabel(p.ID()),
			Unit:        "usd",
			Used:        0,
			Limit:       config.GetProviderLimit(p.ID()),
			Status:      schema.StatusOK,
			RefreshedAt: time.Now(),
		}, nil
	}

	return usages[0], nil
}

type authEntry struct {
	Type      string `json:"type"`
	Key       string `json:"key"`
	Access    string `json:"access"`
	AccountID string `json:"accountId"`
}

func readOpenCodeAuth() (map[string]authEntry, error) {
	home, _ := os.UserHomeDir()
	data, err := os.ReadFile(filepath.Join(home, ".local", "share", "opencode", "auth.json"))
	if err != nil {
		return nil, err
	}
	var auth map[string]authEntry
	if err := json.Unmarshal(data, &auth); err != nil {
		return nil, err
	}
	return auth, nil
}

func readZenGoKey() string {
	auth, err := readOpenCodeAuth()
	if err != nil {
		return ""
	}
	return auth["opencode-go"].Key
}

func readOpenAICreds() (access, accountID string, ok bool) {
	auth, err := readOpenCodeAuth()
	if err != nil {
		return "", "", false
	}
	e := auth["openai"]
	if e.Type != "oauth" || e.Access == "" {
		return "", "", false
	}
	return e.Access, e.AccountID, true
}

var openaiUsageURL = "https://chatgpt.com/backend-api/wham/usage"

func fetchOpenAIUsageWindows(ctx context.Context, access, accountID string) ([]schema.UsageWindow, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, openaiUsageURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+access)
	if accountID != "" {
		req.Header.Set("chatgpt-account-id", accountID)
	}

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return nil, fmt.Errorf("opencode openai token expired — run opencode to refresh")
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("opencode openai usage request failed: status %d", resp.StatusCode)
	}

	var payload struct {
		RateLimit *struct {
			PrimaryWindow *struct {
				UsedPercent float64 `json:"used_percent"`
			} `json:"primary_window"`
			SecondaryWindow *struct {
				UsedPercent float64 `json:"used_percent"`
				ResetAt     float64 `json:"reset_at"`
			} `json:"secondary_window"`
		} `json:"rate_limit"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("opencode openai usage parse error: %w", err)
	}

	var windows []schema.UsageWindow
	if rl := payload.RateLimit; rl != nil {
		if pw := rl.PrimaryWindow; pw != nil {
			windows = append(windows, schema.UsageWindow{
				Key:   "5h",
				Label: "5h",
				Used:  pw.UsedPercent,
				Limit: 100,
				Unit:  "percent",
			})
		}
		if sw := rl.SecondaryWindow; sw != nil {
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
	return windows, nil
}

type zenUsageWindow struct {
	Status   string  `json:"status"`
	Percent  float64 `json:"percent"`
	ResetsAt string  `json:"resetsAt"`
}

type zenUsagePayload struct {
	Usage struct {
		Rolling zenUsageWindow `json:"rolling"`
		Weekly  zenUsageWindow `json:"weekly"`
		Monthly zenUsageWindow `json:"monthly"`
	} `json:"usage"`
	Type  string `json:"type"`
	Error struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error"`
}

var zenGoUsageURL = "https://opencode.ai/zen/go/v1/usage"

// fetchZenGoUsage returns the server-side usage windows for the Go
// subscription. The API is authoritative: it tracks usage across all of the
// user's machines, not just local DB spend.
func fetchZenGoUsage(ctx context.Context, key string) (*zenUsagePayload, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, zenGoUsageURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+key)

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("opencode zen usage request failed: status %d", resp.StatusCode)
	}

	var payload zenUsagePayload
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("opencode zen usage parse error: %w", err)
	}
	if payload.Type == "error" || payload.Error.Type != "" {
		return nil, fmt.Errorf("opencode zen usage error: %s", payload.Error.Message)
	}
	return &payload, nil
}

// zenUsageWindows converts the API payload into percent-based usage windows.
// A "rate-limited" window counts as fully used.
func zenUsageWindows(payload *zenUsagePayload) []schema.UsageWindow {
	mk := func(key string, w zenUsageWindow) schema.UsageWindow {
		used := w.Percent
		if w.Status == "rate-limited" {
			used = 100
		}
		out := schema.UsageWindow{
			Key:   key,
			Label: key,
			Used:  used,
			Limit: 100,
			Unit:  "percent",
		}
		if t, err := time.Parse(time.RFC3339, w.ResetsAt); err == nil {
			out.ResetAt = t
		}
		return out
	}
	return []schema.UsageWindow{
		mk("5h", payload.Usage.Rolling),
		mk("weekly", payload.Usage.Weekly),
		mk("monthly", payload.Usage.Monthly),
	}
}

// FetchAll returns one Usage entry per providerID found in the database.
func (p *Provider) FetchAll(ctx context.Context) ([]*schema.Usage, error) {
	now := time.Now()
	periodStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	end := periodStart.AddDate(0, 1, 0)
	weekStart := now.Add(-7 * 24 * time.Hour)
	fiveHStart := now.Add(-5 * time.Hour)

	db := p.db
	if db == nil {
		if _, err := os.Stat(p.dbPath); os.IsNotExist(err) {
			return nil, fmt.Errorf("opencode database not found — ensure opencode is installed and has been used at least once")
		}
		dsn := fmt.Sprintf("file:%s?_mode=ro&_query_only=true&_busy_timeout=5000", p.dbPath)
		var err error
		db, err = sql.Open("sqlite", dsn)
		if err != nil {
			return nil, fmt.Errorf("failed to open opencode database: %w", err)
		}
		defer db.Close()
	}

	startMillis := periodStart.UnixMilli()
	endMillis := end.UnixMilli()
	weekStartMillis := weekStart.UnixMilli()
	fiveHStartMillis := fiveHStart.UnixMilli()

	query := `
SELECT
  json_extract(data,'$.providerID') as provider,
	  SUM(COALESCE(json_extract(data,'$.tokens.input'),0) + COALESCE(json_extract(data,'$.tokens.output'),0)) as month_tokens,
	  SUM(CASE WHEN json_extract(data,'$.time.created') >= ? THEN COALESCE(json_extract(data,'$.tokens.input'),0) + COALESCE(json_extract(data,'$.tokens.output'),0) ELSE 0 END) as week_tokens,
	  SUM(CASE WHEN json_extract(data,'$.time.created') >= ? THEN COALESCE(json_extract(data,'$.tokens.input'),0) + COALESCE(json_extract(data,'$.tokens.output'),0) ELSE 0 END) as five_hour_tokens,
	  SUM(COALESCE(json_extract(data,'$.cost'),0)) as month_cost_usd,
	  SUM(CASE WHEN json_extract(data,'$.time.created') >= ? THEN COALESCE(json_extract(data,'$.cost'),0) ELSE 0 END) as week_cost_usd,
	  SUM(CASE WHEN json_extract(data,'$.time.created') >= ? THEN COALESCE(json_extract(data,'$.cost'),0) ELSE 0 END) as five_hour_cost_usd
FROM message
WHERE json_extract(data,'$.role') = 'assistant'
  AND json_extract(data,'$.time.created') >= ?
  AND json_extract(data,'$.time.created') < ?
GROUP BY provider;`

	rows, err := db.QueryContext(ctx, query, weekStartMillis, fiveHStartMillis, weekStartMillis, fiveHStartMillis, startMillis, endMillis)
	if err != nil {
		if isSQLiteBusy(err) {
			return nil, fmt.Errorf("database is busy — another process may be writing: %w", err)
		}
		return nil, fmt.Errorf("failed to query opencode database: %w", err)
	}
	defer rows.Close()

	var providerRows []providerRow
	for rows.Next() {
		var row providerRow
		if err := rows.Scan(&row.ProviderID, &row.MonthTokens, &row.WeekTokens, &row.FiveHTokens, &row.MonthCostUSD, &row.WeekCostUSD, &row.FiveHCostUSD); err != nil {
			continue
		}
		providerRows = append(providerRows, row)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error reading opencode rows: %w", err)
	}

	if len(providerRows) == 0 {
		return nil, nil
	}

	sharedLimit := config.GetProviderLimit("opencode-go")
	// sharedMonthly is the effective shared budget cap; always > 0 so real-cost
	// sub-providers without a per-sub configured limit can still compute a %.
	sharedMonthly := sharedLimit
	if sharedMonthly <= 0 {
		sharedMonthly = goMonthLimit
	}
	totalRealCost := 0.0
	totalTokens := 0.0
	for _, row := range providerRows {
		totalRealCost += row.MonthCostUSD
		totalTokens += row.TotalTokens()
	}

	var usages []*schema.Usage
	for _, row := range providerRows {
		rawID := row.ProviderID
		if rawID == "" {
			rawID = "unknown"
		}
		// Namespace the provider ID so it never collides with top-level providers
		// (e.g. opencode's "openai" rows vs. the Codex "openai" provider).
		provID := "opencode/" + rawID

		label := config.GetProviderLabel(provID)

		// opencode/openai has its own OAuth token and a potentially different
		// ChatGPT account from the standalone codex CLI — fetch real usage directly.
		if rawID == "openai" {
			if access, accountID, ok := readOpenAICreds(); ok {
				windows, err := fetchOpenAIUsageWindows(ctx, access, accountID)
				if err != nil {
					usages = append(usages, &schema.Usage{
						Provider:    provID,
						Label:       label,
						Unit:        "percent",
						Status:      schema.StatusError,
						Error:       err.Error(),
						RefreshedAt: now,
					})
				} else {
					usages = append(usages, &schema.Usage{
						Provider:    provID,
						Label:       label,
						Unit:        "percent",
						Windows:     windows,
						Status:      schema.StatusOK,
						RefreshedAt: now,
					})
				}
				continue
			}
		}

		limit := config.GetProviderLimit(provID)
		used := row.MonthCostUSD
		warning := ""
		isZeroCost := row.MonthCostUSD == 0 && row.TotalTokens() > 0

		if isZeroCost && totalRealCost > 0 && totalTokens > 0 {
			if limit <= 0 {
				limit = sharedLimit
			}
			if limit > 0 {
				used = estimateSharedCost(totalRealCost, row.MonthTokens, totalTokens)
				warning = "estimated from shared opencode subscription"
			}
		}

		// Real-cost sub-providers with no per-sub limit configured fall back to
		// the shared monthly budget so per-window percentages can be computed.
		if !isZeroCost && limit <= 0 {
			limit = sharedMonthly
		}

		windows := buildUsageWindows(now, periodStart, end.Add(-time.Second), limit, row, totalRealCost, totalTokens, isZeroCost && sharedLimit > 0)

		u := &schema.Usage{
			Provider:    provID,
			Label:       label,
			PeriodStart: periodStart,
			PeriodEnd:   end.Add(-time.Second),
			Used:        used,
			Limit:       limit,
			Unit:        "usd",
			Windows:     windows,
			Status:      schema.StatusOK,
			Warning:     warning,
			RefreshedAt: now,
		}

		// For the opencode-go real-cost sub, prefer the server-side usage API:
		// it reports authoritative percent windows (rolling/weekly/monthly) and
		// rate-limit status across all machines. Fall back to the local DB
		// windows when the API is unreachable or no key is present.
		if rawID == "opencode-go" {
			if key := readZenGoKey(); key != "" {
				if payload, err := fetchZenGoUsage(ctx, key); err == nil {
					u.Windows = zenUsageWindows(payload)
				}
			}
		}

		usages = append(usages, u)
	}

	return usages, nil
}

func estimateSharedCost(totalCost, providerTokens, totalTokens float64) float64 {
	if totalCost <= 0 || providerTokens <= 0 || totalTokens <= 0 {
		return 0
	}
	return totalCost * (providerTokens / totalTokens)
}

func buildUsageWindows(now, monthStart, monthEnd time.Time, limit float64, row providerRow, totalRealCost, totalTokens float64, isZeroCostEstimated bool) []schema.UsageWindow {
	if isZeroCostEstimated {
		return []schema.UsageWindow{{
			Key:         "monthly",
			Label:       "monthly",
			PeriodStart: monthStart,
			PeriodEnd:   monthEnd,
			Used:        estimateSharedCost(totalRealCost, row.MonthTokens, totalTokens),
			Limit:       limit,
			Unit:        "usd",
		}}
	}

	weekWindow := schema.UsageWindow{
		Key:         "weekly",
		Label:       "weekly",
		PeriodStart: now.Add(-7 * 24 * time.Hour),
		PeriodEnd:   now,
		ResetAt:     now.Add(7 * 24 * time.Hour),
		Used:        row.WeekCostUSD,
		Limit:       goWeekLimit,
		Unit:        "usd",
	}
	fiveHWindow := schema.UsageWindow{
		Key:         "5h",
		Label:       "5h",
		PeriodStart: now.Add(-5 * time.Hour),
		PeriodEnd:   now,
		ResetAt:     now.Add(5 * time.Hour),
		Used:        row.FiveHCostUSD,
		Limit:       go5HLimit,
		Unit:        "usd",
	}
	monthWindow := schema.UsageWindow{
		Key:         "monthly",
		Label:       "monthly",
		PeriodStart: monthStart,
		PeriodEnd:   monthEnd,
		ResetAt:     monthEnd.Add(time.Second),
		Used:        row.MonthCostUSD,
		Limit:       limit,
		Unit:        "usd",
	}
	if limit <= 0 {
		return []schema.UsageWindow{monthWindow}
	}
	return []schema.UsageWindow{fiveHWindow, weekWindow, monthWindow}
}

func isSQLiteBusy(err error) bool {
	if err == nil {
		return false
	}
	return err.Error() == "database is locked" ||
		strings.Contains(err.Error(), "SQLITE_BUSY") ||
		strings.Contains(err.Error(), "database is locked")
}
