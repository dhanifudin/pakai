package renderer

import (
	"fmt"
	"strings"
	"time"

	"github.com/dhanifudin/pakai/internal/schema"
)

// RenderStatus renders human-readable multi-provider status output.
func RenderStatus(usages []*schema.Usage, refreshedAt time.Time) string {
	if len(usages) == 0 {
		return "No providers configured.\n"
	}

	var sb strings.Builder

	for _, u := range usages {
		sb.WriteString(renderUsageLine(u))
	}

	sb.WriteString("\n")

	// Footer
	plural := "provider"
	if len(usages) != 1 {
		plural = "providers"
	}

	if !refreshedAt.IsZero() {
		ago := time.Since(refreshedAt).Round(time.Second)
		sb.WriteString(fmt.Sprintf("  %d %s · refreshed %s ago\n", len(usages), plural, ago))
	} else {
		sb.WriteString(fmt.Sprintf("  %d %s\n", len(usages), plural))
	}

	return sb.String()
}

func renderUsageLine(u *schema.Usage) string {
	label := u.Label
	if label == "" {
		label = u.Provider
	}
	windows := u.WindowsOrDefault()

	if len(windows) > 1 {
		parts := make([]string, 0, len(windows))
		for _, w := range windows {
			parts = append(parts, renderWindowDetail(w))
		}
		return fmt.Sprintf("  %-14s %s\n", label, strings.Join(parts, " | "))
	}

	pct := u.Pct()

	if u.Status == schema.StatusError {
		return fmt.Sprintf("  %-20s error: %s\n", label, u.Error)
	}

	if pct < 0 {
		// No limit set
		return fmt.Sprintf("  %-20s %-12s (no limit set)\n", label, u.FormatUsed())
	}

	// Limit set — show progress bar and percentage
	bar := progressBar(pct, 10)
	limitStr := formatLimit(u)
	return fmt.Sprintf("  %-14s %s %3.0f%%   %s / %s\n", label, bar, pct, u.FormatUsed(), limitStr)
}

func progressBar(pct float64, width int) string {
	filled := int(pct / 100.0 * float64(width))
	if filled > width {
		filled = width
	}
	if filled < 0 {
		filled = 0
	}
	empty := width - filled
	return strings.Repeat("█", filled) + strings.Repeat("░", empty)
}

func formatLimit(u *schema.Usage) string {
	switch u.Unit {
	case "messages":
		return fmt.Sprintf("%.0f msg", u.Limit)
	case "usd":
		return fmt.Sprintf("$%.2f", u.Limit)
	case "tokens":
		if u.Limit >= 1_000_000 {
			return fmt.Sprintf("%.1fM tok", u.Limit/1_000_000)
		}
		if u.Limit >= 1_000 {
			return fmt.Sprintf("%.0fK tok", u.Limit/1_000)
		}
		return fmt.Sprintf("%.0f tok", u.Limit)
	default:
		return fmt.Sprintf("%.2f %s", u.Limit, u.Unit)
	}
}
