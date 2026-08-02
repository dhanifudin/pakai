package pi

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/dhanifudin/pakai/internal/config"
	"github.com/dhanifudin/pakai/internal/schema"
)

type Provider struct {
	sessionsPath string
}

type entry struct {
	Timestamp time.Time `json:"timestamp"`
	Message   struct {
		Role     string `json:"role"`
		Provider string `json:"provider"`
		Usage    struct {
			TotalTokens float64 `json:"totalTokens"`
		} `json:"usage"`
	} `json:"message"`
}

func New() *Provider {
	home, _ := os.UserHomeDir()
	return NewWithPath(filepath.Join(home, ".pi", "agent", "sessions"))
}

func NewWithPath(path string) *Provider {
	return &Provider{sessionsPath: path}
}

func (p *Provider) ID() string {
	return "pi"
}

func (p *Provider) Fetch(ctx context.Context) (*schema.Usage, error) {
	usages, err := p.FetchAll(ctx)
	if err != nil {
		return &schema.Usage{Provider: p.ID(), Label: p.ID(), Status: schema.StatusError, Error: err.Error(), RefreshedAt: time.Now()}, nil
	}
	if len(usages) == 0 {
		return &schema.Usage{Provider: p.ID(), Label: p.ID(), Unit: "tokens", Status: schema.StatusOK, RefreshedAt: time.Now()}, nil
	}
	return usages[0], nil
}

func (p *Provider) FetchAll(ctx context.Context) ([]*schema.Usage, error) {
	now := time.Now()
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	totals := map[string]float64{}

	err := filepath.WalkDir(p.sessionsPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if d.IsDir() || filepath.Ext(path) != ".jsonl" {
			return nil
		}
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		scanner.Buffer(make([]byte, 64*1024), 16*1024*1024)
		for scanner.Scan() {
			var item entry
			if json.Unmarshal(scanner.Bytes(), &item) != nil || item.Message.Role != "assistant" || item.Message.Provider == "" {
				continue
			}
			if item.Timestamp.Before(monthStart) || !item.Timestamp.Before(monthStart.AddDate(0, 1, 0)) {
				continue
			}
			totals[item.Message.Provider] += item.Message.Usage.TotalTokens
		}
		return scanner.Err()
	})
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("Pi sessions not found — run Pi at least once")
		}
		return nil, fmt.Errorf("read Pi sessions: %w", err)
	}

	ids := make([]string, 0, len(totals))
	for id := range totals {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	usages := make([]*schema.Usage, 0, len(ids))
	for _, id := range ids {
		providerID := "pi/" + id
		usages = append(usages, &schema.Usage{
			Provider:    providerID,
			Label:       config.GetProviderLabel(providerID),
			PeriodStart: monthStart,
			PeriodEnd:   monthStart.AddDate(0, 1, 0).Add(-time.Second),
			Used:        totals[id],
			Limit:       config.GetProviderLimit(providerID),
			Unit:        "tokens",
			Status:      schema.StatusOK,
			Warning:     "local Pi session usage",
			RefreshedAt: now,
		})
	}
	return usages, nil
}
