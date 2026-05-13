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

// providerRow holds a single row from the OpenCode query.
type providerRow struct {
	ProviderID   string
	InputTokens  float64
	OutputTokens float64
	TotalCostUSD float64
}

// Provider implements the providers.Provider interface for OpenCode.
type Provider struct {
	db   *sql.DB
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

	query := `
SELECT
  json_extract(data,'$.providerID') as provider,
  SUM(json_extract(data,'$.tokens.input'))  as input_tokens,
  SUM(json_extract(data,'$.tokens.output')) as output_tokens,
  SUM(json_extract(data,'$.cost'))          as total_cost_usd
FROM message
WHERE json_extract(data,'$.role') = 'assistant'
  AND json_extract(data,'$.time.created') >= ?
  AND json_extract(data,'$.time.created') < ?
GROUP BY provider;`

	rows, err := db.QueryContext(ctx, query, startMillis, endMillis)
	if err != nil {
		if isSQLiteBusy(err) {
			return nil, fmt.Errorf("database is busy — another process may be writing: %w", err)
		}
		return nil, fmt.Errorf("failed to query opencode database: %w", err)
	}
	defer rows.Close()

	var usages []*schema.Usage
	for rows.Next() {
		var row providerRow
		if err := rows.Scan(&row.ProviderID, &row.InputTokens, &row.OutputTokens, &row.TotalCostUSD); err != nil {
			continue
		}

		provID := row.ProviderID
		if provID == "" {
			provID = "opencode-unknown"
		}

		label := config.GetProviderLabel(provID)
		limit := config.GetProviderLimit(provID)

		u := &schema.Usage{
			Provider:    provID,
			Label:       label,
			PeriodStart: periodStart,
			PeriodEnd:   end.Add(-time.Second),
			Used:        row.TotalCostUSD,
			Limit:       limit,
			Unit:        "usd",
			Status:      schema.StatusOK,
			RefreshedAt: now,
		}
		usages = append(usages, u)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error reading opencode rows: %w", err)
	}

	if len(usages) == 0 {
		return nil, nil
	}

	return usages, nil
}

func isSQLiteBusy(err error) bool {
	if err == nil {
		return false
	}
	return err.Error() == "database is locked" ||
		strings.Contains(err.Error(), "SQLITE_BUSY") ||
		strings.Contains(err.Error(), "database is locked")
}
