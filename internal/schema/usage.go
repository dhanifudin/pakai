package schema

import (
	"fmt"
	"time"
)

// Status represents the status of a usage entry.
type Status string

const (
	StatusOK    Status = "ok"
	StatusError Status = "error"
	StatusStale Status = "stale"
	StatusMock  Status = "mock"
)

// UsageWindow represents usage for a specific billing or quota window.
type UsageWindow struct {
	Key         string    `json:"key,omitempty"`
	Label       string    `json:"label,omitempty"`
	PeriodStart time.Time `json:"period_start,omitempty"`
	PeriodEnd   time.Time `json:"period_end,omitempty"`
	ResetAt     time.Time `json:"reset_at,omitempty"`
	Used        float64   `json:"used"`
	Limit       float64   `json:"limit"`
	Unit        string    `json:"unit"`
}

// Usage represents normalized usage data from a provider.
type Usage struct {
	Provider    string        `json:"provider"`
	Label       string        `json:"label"`
	Plan        string        `json:"plan,omitempty"`
	PeriodStart time.Time     `json:"period_start"`
	PeriodEnd   time.Time     `json:"period_end"`
	Used        float64       `json:"used"`
	Limit       float64       `json:"limit"` // 0 = no limit configured
	Unit        string        `json:"unit"`  // "messages", "tokens", "usd"
	Windows     []UsageWindow `json:"windows,omitempty"`
	Status      Status        `json:"status"`
	Error       string        `json:"error,omitempty"`
	Warning     string        `json:"warning,omitempty"`
	RefreshedAt time.Time     `json:"refreshed_at"`
}

func (w UsageWindow) Pct() float64 {
	if w.Limit <= 0 {
		return -1
	}
	return (w.Used / w.Limit) * 100
}

// Pct returns the percentage of usage relative to the limit.
// Returns -1 if no limit is configured.
func (u *Usage) Pct() float64 {
	if u.Limit <= 0 {
		return -1
	}
	return (u.Used / u.Limit) * 100
}

func (u *Usage) WindowsOrDefault() []UsageWindow {
	if len(u.Windows) > 0 {
		return u.Windows
	}

	return []UsageWindow{{
		Key:         "monthly",
		Label:       "monthly",
		PeriodStart: u.PeriodStart,
		PeriodEnd:   u.PeriodEnd,
		Used:        u.Used,
		Limit:       u.Limit,
		Unit:        u.Unit,
	}}
}

func (u *Usage) WorstPct() float64 {
	maxPct := -1.0
	for _, w := range u.WindowsOrDefault() {
		pct := w.Pct()
		if pct > maxPct {
			maxPct = pct
		}
	}
	return maxPct
}

// FormatUsed returns the usage formatted based on the unit.
func (u *Usage) FormatUsed() string {
	return formatUsed(u.Used, u.Unit)
}

func (w UsageWindow) FormatUsed() string {
	return formatUsed(w.Used, w.Unit)
}

func formatUsed(used float64, unit string) string {
	switch unit {
	case "messages":
		return fmt.Sprintf("%.0f msg", used)
	case "usd":
		return fmt.Sprintf("$%.2f", used)
	case "tokens":
		if used >= 1_000_000 {
			return fmt.Sprintf("%.1fM tok", used/1_000_000)
		}
		if used >= 1_000 {
			return fmt.Sprintf("%.0fK tok", used/1_000)
		}
		return fmt.Sprintf("%.0f tok", used)
	case "percent":
		return fmt.Sprintf("%.0f%%", used)
	default:
		return fmt.Sprintf("%.2f %s", used, unit)
	}
}
