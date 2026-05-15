package opencode

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dhanifudin/pakai/internal/config"
	"github.com/dhanifudin/pakai/internal/schema"
	_ "modernc.org/sqlite"
)

const dbFileName = "opencode.db"

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
		dbPath: filepath.Join(home, ".local", "share", "opencode", dbFileName),
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
			return nil, fmt.Errorf("opencode database not found: %s", p.dbPath)
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
	totalRealCost := 0.0
	totalTokens := 0.0
	codexInstalled := codexDetector()
	for _, row := range providerRows {
		if codexInstalled && strings.ToLower(row.ProviderID) == "openai" {
			continue
		}
		totalRealCost += row.MonthCostUSD
		totalTokens += row.TotalTokens()
	}

	var usages []*schema.Usage
	for _, row := range providerRows {
		if codexInstalled && strings.ToLower(row.ProviderID) == "openai" {
			continue
		}
		provID := row.ProviderID
		if provID == "" {
			provID = "opencode-unknown"
		}

		label := config.GetProviderLabel(provID)
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

func isCodexInstalled() bool {
	home, err := os.UserHomeDir()
	if err != nil {
		return false
	}
	_, err = os.Stat(filepath.Join(home, ".codex", "auth.json"))
	return err == nil
}

var codexDetector = isCodexInstalled
