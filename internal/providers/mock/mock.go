package mock

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/dhanifudin/pakai/internal/schema"
)

// mockData represents the persisted mock state.
type mockData struct {
	Provider string  `json:"provider"`
	Used     float64 `json:"used"`
	Limit    float64 `json:"limit"`
	Unit     string  `json:"unit"`
}

// Provider implements a mock provider for testing.
type Provider struct {
	id    string
	used  float64
	limit float64
	unit  string
}

// New creates a new mock provider with the given parameters.
func New(id string, used, limit float64, unit string) *Provider {
	return &Provider{
		id:    id,
		used:  used,
		limit: limit,
		unit:  unit,
	}
}

// ID returns the provider ID.
func (p *Provider) ID() string {
	return p.id
}

// Fetch returns mock usage data.
func (p *Provider) Fetch(ctx context.Context) (*schema.Usage, error) {
	now := time.Now()
	periodStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	periodEnd := periodStart.AddDate(0, 1, 0).Add(-time.Second)

	return &schema.Usage{
		Provider:    p.id,
		Label:       p.id,
		PeriodStart: periodStart,
		PeriodEnd:   periodEnd,
		Used:        p.used,
		Limit:       p.limit,
		Unit:        p.unit,
		Status:      schema.StatusMock,
		RefreshedAt: now,
	}, nil
}

// MockFilePath returns the path where mock data is stored for a given provider ID.
func MockFilePath(id string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".cache", "pakai", fmt.Sprintf("mock-%s.json", id)), nil
}

// Save persists the mock provider configuration.
func (p *Provider) Save() error {
	path, err := MockFilePath(p.id)
	if err != nil {
		return err
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data := mockData{
		Provider: p.id,
		Used:     p.used,
		Limit:    p.limit,
		Unit:     p.unit,
	}

	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, b, 0644)
}

// Load loads a mock provider configuration from disk.
func Load(id string) (*Provider, error) {
	path, err := MockFilePath(id)
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("mock provider %q not found: %w", id, err)
	}

	var md mockData
	if err := json.Unmarshal(data, &md); err != nil {
		return nil, fmt.Errorf("failed to parse mock data: %w", err)
	}

	return &Provider{
		id:    md.Provider,
		used:  md.Used,
		limit: md.Limit,
		unit:  md.Unit,
	}, nil
}

// Remove deletes the mock provider configuration.
func Remove(id string) error {
	path, err := MockFilePath(id)
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// ListMocked returns all mocked provider IDs.
func ListMocked() ([]string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	cacheDir := filepath.Join(home, ".cache", "pakai")
	entries, err := os.ReadDir(cacheDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var ids []string
	for _, e := range entries {
		name := e.Name()
		if len(name) > 5 && name[:5] == "mock-" && len(name) > 10 && name[len(name)-5:] == ".json" {
			id := name[5 : len(name)-5]
			ids = append(ids, id)
		}
	}
	return ids, nil
}
