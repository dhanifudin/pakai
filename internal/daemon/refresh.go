package daemon

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/dhanifudin/pakai/internal/cache"
	"github.com/dhanifudin/pakai/internal/config"
	"github.com/dhanifudin/pakai/internal/providers"
	"github.com/dhanifudin/pakai/internal/schema"
	"golang.org/x/sync/singleflight"
)

const defaultPollInterval = 30 * time.Second

// RefreshLoop manages the periodic refresh of provider data.
type RefreshLoop struct {
	mu           sync.RWMutex
	providers    []providers.Provider
	cache        *cache.Cache
	current      []*schema.Usage
	onRefresh    func([]*schema.Usage)
	stopCh       chan struct{}
	pollInterval time.Duration
	sf           singleflight.Group
}

// NewRefreshLoop creates a new refresh loop.
func NewRefreshLoop(provs []providers.Provider, c *cache.Cache, onRefresh func([]*schema.Usage)) *RefreshLoop {
	return &RefreshLoop{
		providers:    provs,
		cache:        c,
		onRefresh:    onRefresh,
		stopCh:       make(chan struct{}),
		pollInterval: defaultPollInterval,
	}
}

// Start begins the refresh loop.
func (r *RefreshLoop) Start() {
	go r.run()
}

// Stop signals the refresh loop to stop.
func (r *RefreshLoop) Stop() {
	close(r.stopCh)
}

// Current returns the current usage data.
func (r *RefreshLoop) Current() []*schema.Usage {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.current
}

// UpdateProviders replaces the provider list and triggers an immediate refresh.
func (r *RefreshLoop) UpdateProviders(provs []providers.Provider) {
	r.mu.Lock()
	r.providers = provs
	r.current = nil
	r.mu.Unlock()
	go r.refresh()
}

// RefreshNow triggers an immediate refresh of all providers.
func (r *RefreshLoop) RefreshNow() {
	go r.refresh()
}

func (r *RefreshLoop) run() {
	r.refresh()

	for {
		interval := r.adaptiveInterval()
		select {
		case <-r.stopCh:
			return
		case <-time.After(interval):
			r.refresh()
		}
	}
}

func (r *RefreshLoop) refresh() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var wg sync.WaitGroup
	resultsCh := make(chan *schema.Usage, len(r.providers)*5)

	for _, p := range r.providers {
		wg.Add(1)
		go func(prov providers.Provider) {
			defer wg.Done()
			defer r.recoverProvider(prov)

			// Check if provider is enabled in live config
			cfg := config.Config()
			if pc, ok := cfg.Provider[prov.ID()]; ok && !pc.Enabled {
				return
			}

			r.fetchProvider(ctx, prov, resultsCh)
		}(p)
	}

	wg.Wait()
	close(resultsCh)

	results := make([]*schema.Usage, 0, len(r.providers)*2)
	for u := range resultsCh {
		results = append(results, u)
	}

	r.mu.Lock()
	r.current = results
	r.mu.Unlock()

	if err := r.cache.Write(results); err != nil {
		slog.Error("failed to write cache", "error", err)
	}

	if r.onRefresh != nil {
		r.onRefresh(results)
	}
}

func (r *RefreshLoop) fetchProvider(ctx context.Context, prov providers.Provider, resultsCh chan<- *schema.Usage) {
	key := fmt.Sprintf("refresh:%s", prov.ID())

	v, err, _ := r.sf.Do(key, func() (interface{}, error) {
		type multiProvider interface {
			FetchAll(ctx context.Context) ([]*schema.Usage, error)
		}

		if mp, ok := prov.(multiProvider); ok {
			usages, err := mp.FetchAll(ctx)
			if err != nil {
				return []*schema.Usage{{
					Provider:    prov.ID(),
					Label:       prov.ID(),
					Status:      schema.StatusError,
					Error:       err.Error(),
					RefreshedAt: time.Now(),
				}}, nil
			}
			return usages, nil
		}

		u, err := prov.Fetch(ctx)
		if err != nil {
			return []*schema.Usage{{
				Provider:    prov.ID(),
				Label:       prov.ID(),
				Status:      schema.StatusError,
				Error:       err.Error(),
				RefreshedAt: time.Now(),
			}}, nil
		}
		return []*schema.Usage{u}, nil
	})

	if err != nil {
		slog.Error("singleflight fetch failed", "provider", prov.ID(), "error", err)
		return
	}

	if usages, ok := v.([]*schema.Usage); ok {
		for _, u := range usages {
			resultsCh <- u
		}
	}
}

func (r *RefreshLoop) recoverProvider(prov providers.Provider) {
	if rec := recover(); rec != nil {
		slog.Error("provider panic recovered",
			"provider", prov.ID(),
			"panic", rec,
		)
	}
}

// adaptiveInterval returns the refresh interval based on max percentage across providers.
func (r *RefreshLoop) adaptiveInterval() time.Duration {
	r.mu.RLock()
	current := r.current
	r.mu.RUnlock()

	cfg := config.Config()
	warning := cfg.Thresholds.Warning
	critical := cfg.Thresholds.Critical

	maxPct := -1.0
	for _, u := range current {
		if u.Status != schema.StatusOK {
			continue
		}
		pct := u.WorstPct()
		if pct < 0 {
			continue
		}
		if pct > maxPct {
			maxPct = pct
		}
	}

	switch {
	case maxPct < 0:
		return defaultPollInterval
	case maxPct >= 95:
		return 10 * time.Second
	case maxPct >= float64(critical):
		return 30 * time.Second
	case maxPct >= float64(warning):
		return 120 * time.Second
	default:
		return defaultPollInterval
	}
}
