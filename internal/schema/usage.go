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

// PctLeft returns the remaining percentage (100 - Pct()).
// Returns -1 if no limit is configured.
func (w UsageWindow) PctLeft() float64 {
	p := w.Pct()
	if p < 0 {
		return -1
	}
	return 100 - p
}

// Reserve estimates the remaining quota at window end as a percentage of limit,
// based on elapsed time and burn rate. Positive = won't exhaust before reset.
// Returns (0, 0, false) when there isn't enough data (percent unit, no times, zero limit).
func (w UsageWindow) Reserve() (reservePct float64, depletion time.Duration, ok bool) {
	if w.Limit <= 0 || w.ResetAt.IsZero() || w.PeriodStart.IsZero() {
		return 0, 0, false
	}
	if w.Unit == "percent" {
		return 0, 0, false
	}

	now := time.Now()
	elapsed := now.Sub(w.PeriodStart)
	if elapsed <= 0 {
		return 0, 0, false
	}

	remaining := w.ResetAt.Sub(now)
	if remaining <= 0 {
		return 0, 0, false
	}

	rate := w.Used / elapsed.Seconds()
	projectedEnd := w.Used + rate*remaining.Seconds()
	reserve := w.Limit - projectedEnd
	reservePct = (reserve / w.Limit) * 100
	depletion = time.Duration((w.Limit - w.Used) / rate * float64(time.Second))
	return reservePct, depletion, true
}

// ReserveStr returns a short human-readable string for the reserve status.
// Returns "" when the window doesn't support reserve calculation.
func (w UsageWindow) ReserveStr() string {
	reservePct, depletion, ok := w.Reserve()
	if !ok {
		return ""
	}
	if reservePct >= 0 {
		if reservePct > 10 {
			return "On pace"
		}
		return fmt.Sprintf("%.0f%% in reserve", reservePct)
	}
	if depletion <= 0 {
		return ""
	}
	d := depletion.Round(time.Minute)
	if d >= 24*time.Hour {
		days := int(d.Hours()) / 24
		hours := int(d.Hours()) % 24
		if hours == 0 {
			return fmt.Sprintf("Runs out in %dd", days)
		}
		return fmt.Sprintf("Runs out in %dd %dh", days, hours)
	}
	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60
	if hours == 0 {
		return fmt.Sprintf("Runs out in %dm", minutes)
	}
	return fmt.Sprintf("Runs out in %dh %02dm", hours, minutes)
}
